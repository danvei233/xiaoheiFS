package logreader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	eventLogProbeMu         sync.Mutex
	eventLogPreferredByName = map[string]string{}
	eventLogDeniedUntil     = map[string]time.Time{}
)

func Stream(source, keyword string, lines int, follow bool, emit func(string) bool) error {
	if lines <= 0 {
		lines = 300
	}
	source = strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(source, "file:"):
		path := strings.TrimPrefix(source, "file:")
		if strings.TrimSpace(path) == "" {
			path = "logs"
		}
		return streamFile(path, keyword, lines, follow, emit)
	case strings.HasPrefix(source, "journal:"):
		unit := strings.TrimSpace(strings.TrimPrefix(source, "journal:"))
		return streamJournal(unit, keyword, lines, follow, emit)
	case strings.HasPrefix(source, "eventlog:"):
		name := strings.TrimSpace(strings.TrimPrefix(source, "eventlog:"))
		return streamEventLog(name, keyword, lines, follow, emit)
	default:
		return fmt.Errorf("unsupported source: %s", source)
	}
}

func streamFile(path, keyword string, lines int, follow bool, emit func(string) bool) error {
	target, err := resolveLogFile(path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	text := filterLines(lastLines(string(b), lines), keyword)
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !emit(line) {
			return nil
		}
	}
	if !follow {
		return nil
	}
	f, err := os.Open(target)
	if err != nil {
		return err
	}
	defer f.Close()
	offset, _ := f.Seek(0, os.SEEK_END)
	for i := 0; i < 600; i++ { // follow at most 10m
		time.Sleep(time.Second)
		info, err := f.Stat()
		if err != nil {
			continue
		}
		if info.Size() <= offset {
			continue
		}
		delta := info.Size() - offset
		buf := make([]byte, delta)
		_, _ = f.ReadAt(buf, offset)
		offset = info.Size()
		for _, line := range strings.Split(string(buf), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || (keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword))) {
				continue
			}
			if !emit(line) {
				return nil
			}
		}
	}
	return nil
}

func streamJournal(unit, keyword string, lines int, follow bool, emit func(string) bool) error {
	if runtime.GOOS == "windows" {
		return errors.New("journal not supported on windows")
	}
	args := []string{"-n", fmt.Sprintf("%d", lines), "--no-pager"}
	if unit != "" && unit != "system" {
		args = append([]string{"-u", unit}, args...)
	}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.Command("journalctl", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			continue
		}
		if !emit(line) {
			break
		}
	}
	return nil
}

func streamEventLog(name, keyword string, lines int, follow bool, emit func(string) bool) error {
	if runtime.GOOS != "windows" {
		return errors.New("eventlog only supported on windows")
	}
	logName, profile := parseEventLogSource(name)
	candidates := orderedEventLogCandidates(logName)
	initialLines := lines
	if initialLines <= 0 {
		initialLines = 100
	}
	// Hyper-V channels are expensive; keep first batch smaller.
	if isHyperVLogName(logName) && initialLines > 80 {
		initialLines = 80
	}
	var lastErr error
	unauthorizedCount := 0
	for _, candidate := range candidates {
		events, err := fetchWinEvents(candidate, initialLines, "")
		if err != nil {
			msg := strings.TrimSpace(err.Error())
			if isUnauthorizedEventLogError(msg) {
				unauthorizedCount++
				markEventLogDenied(candidate)
			}
			lastErr = fmt.Errorf("eventlog %s failed: %s", candidate, msg)
			continue
		}
		markEventLogPreferred(logName, candidate)
		seen := map[string]struct{}{}
		latest := ""
		for _, ev := range events {
			if !shouldKeepEvent(ev, profile) {
				continue
			}
			key := eventUniqueKey(ev)
			seen[key] = struct{}{}
			if ts := eventCreatedAtString(ev); ts > latest {
				latest = ts
			}
			line := formatEventLine(ev)
			if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				continue
			}
			if !emit(line) {
				return nil
			}
		}
		if follow {
			for {
				time.Sleep(2 * time.Second)
				incr, incrErr := fetchWinEvents(candidate, 64, latest)
				if incrErr != nil {
					msg := strings.TrimSpace(incrErr.Error())
					if isUnauthorizedEventLogError(msg) {
						markEventLogDenied(candidate)
						// Permission can be revoked after startup.
						return fmt.Errorf("eventlog %s failed: access denied (run pingbot as Administrator)", logName)
					}
					continue
				}
				for _, ev := range incr {
					key := eventUniqueKey(ev)
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
					if ts := eventCreatedAtString(ev); ts > latest {
						latest = ts
					}
					if !shouldKeepEvent(ev, profile) {
						continue
					}
					line := formatEventLine(ev)
					if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
						continue
					}
					if !emit(line) {
						return nil
					}
				}
			}
		}
		return nil
	}
	if unauthorizedCount == len(candidates) && len(candidates) > 0 {
		return fmt.Errorf("eventlog %s failed: access denied (run pingbot as Administrator)", logName)
	}
	if lastErr != nil {
		return lastErr
	}
	_ = follow
	return nil
}

func orderedEventLogCandidates(logName string) []string {
	base := resolveEventLogCandidates(logName)
	now := time.Now()
	eventLogProbeMu.Lock()
	defer eventLogProbeMu.Unlock()

	filtered := make([]string, 0, len(base))
	for _, c := range base {
		if until, ok := eventLogDeniedUntil[c]; ok && until.After(now) {
			continue
		}
		filtered = append(filtered, c)
	}
	if len(filtered) == 0 {
		filtered = append(filtered, base...)
	}

	preferred := eventLogPreferredByName[strings.ToLower(strings.TrimSpace(logName))]
	if preferred == "" {
		return filtered
	}
	out := make([]string, 0, len(filtered))
	for _, c := range filtered {
		if c == preferred {
			out = append(out, c)
		}
	}
	for _, c := range filtered {
		if c != preferred {
			out = append(out, c)
		}
	}
	return out
}

func markEventLogPreferred(logName, candidate string) {
	key := strings.ToLower(strings.TrimSpace(logName))
	if key == "" || strings.TrimSpace(candidate) == "" {
		return
	}
	eventLogProbeMu.Lock()
	eventLogPreferredByName[key] = candidate
	delete(eventLogDeniedUntil, candidate)
	eventLogProbeMu.Unlock()
}

func markEventLogDenied(candidate string) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return
	}
	eventLogProbeMu.Lock()
	eventLogDeniedUntil[candidate] = time.Now().Add(10 * time.Minute)
	eventLogProbeMu.Unlock()
}

func fetchWinEvents(logName string, maxEvents int, startTimeISO string) ([]map[string]any, error) {
	if maxEvents <= 0 {
		maxEvents = 64
	}
	escapedLog := strings.ReplaceAll(logName, "'", "''")
	ps := ""
	if strings.TrimSpace(startTimeISO) == "" {
		ps = fmt.Sprintf(
			"$ErrorActionPreference='Stop'; Get-WinEvent -LogName '%s' -MaxEvents %d | Select-Object TimeCreated,RecordId,Id,LevelDisplayName,ProviderName,Message | ConvertTo-Json -Depth 3 -Compress",
			escapedLog,
			maxEvents,
		)
	} else {
		escapedStart := strings.ReplaceAll(startTimeISO, "'", "''")
		ps = fmt.Sprintf(
			"$ErrorActionPreference='Stop'; $st=[DateTime]::Parse('%s'); Get-WinEvent -FilterHashtable @{LogName='%s'; StartTime=$st} -MaxEvents %d | Select-Object TimeCreated,RecordId,Id,LevelDisplayName,ProviderName,Message | ConvertTo-Json -Depth 3 -Compress",
			escapedStart,
			escapedLog,
			maxEvents,
		)
	}
	cmd := exec.Command("sudo", "powershell", "-NoProfile", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, lookErr := exec.LookPath("sudo"); lookErr != nil {
			cmd = exec.Command("powershell", "-NoProfile", "-Command", ps)
			out, err = cmd.CombinedOutput()
		}
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return parseEventLogJSON(out), nil
}

func parseEventLogSource(raw string) (logName, profile string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "System", "important"
	}
	parts := strings.Split(raw, ":")
	logName = strings.TrimSpace(parts[0])
	if logName == "" {
		logName = "System"
	}
	profile = "important"
	if len(parts) > 1 {
		p := strings.ToLower(strings.TrimSpace(parts[1]))
		if p == "full" || p == "important" || p == "power" {
			profile = p
		}
	}
	return logName, profile
}

func resolveEventLogCandidates(logName string) []string {
	name := strings.TrimSpace(logName)
	lower := strings.ToLower(name)
	if name == "" {
		return []string{"System"}
	}
	switch lower {
	case "hyper-v-worker", "hyperv-worker", "hyper-v", "hyperv":
		return []string{"Microsoft-Windows-Hyper-V-Compute-Admin"}
	}
	// Common aliases
	if strings.EqualFold(name, "application") {
		return []string{"Application"}
	}
	if strings.EqualFold(name, "system") {
		return []string{"System"}
	}
	return []string{name}
}

func parseEventLogJSON(raw []byte) []map[string]any {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	var one map[string]any
	if err := json.Unmarshal(raw, &one); err == nil && len(one) > 0 {
		return []map[string]any{one}
	}
	return nil
}

func shouldKeepEvent(ev map[string]any, profile string) bool {
	level := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ev["LevelDisplayName"])))
	provider := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ev["ProviderName"])))
	msg := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ev["Message"])))
	id := intFromAny(ev["Id"])

	switch profile {
	case "full":
		return true
	case "power":
		powerIDs := map[int]struct{}{
			12: {}, 13: {}, 41: {}, 1074: {}, 6005: {}, 6006: {}, 6008: {}, 6009: {},
		}
		if _, ok := powerIDs[id]; ok {
			return true
		}
		return strings.Contains(provider, "kernel-power") || strings.Contains(provider, "eventlog")
	default:
		if level != "critical" && level != "error" && level != "warning" {
			return false
		}
		if strings.Contains(provider, "distributedcom") || strings.Contains(msg, "unable to start a dcom server") {
			return false
		}
		if strings.Contains(msg, "backgroundtaskhost.exe") || strings.Contains(msg, "dllhost.exe /processid") {
			return false
		}
		return true
	}
}

func formatEventLine(ev map[string]any) string {
	timeRaw := strings.TrimSpace(fmt.Sprintf("%v", ev["TimeCreated"]))
	timeRaw = normalizeEventTime(timeRaw)
	level := strings.TrimSpace(fmt.Sprintf("%v", ev["LevelDisplayName"]))
	provider := strings.TrimSpace(fmt.Sprintf("%v", ev["ProviderName"]))
	msg := strings.TrimSpace(fmt.Sprintf("%v", ev["Message"]))
	msg = strings.ReplaceAll(msg, "\r\n", " | ")
	msg = strings.ReplaceAll(msg, "\n", " | ")
	id := intFromAny(ev["Id"])
	return fmt.Sprintf("[%s][%s][%s][id=%d] %s", timeRaw, level, provider, id, msg)
}

func eventCreatedAtString(ev map[string]any) string {
	return strings.TrimSpace(fmt.Sprintf("%v", ev["TimeCreated"]))
}

func eventUniqueKey(ev map[string]any) string {
	recordID := strings.TrimSpace(fmt.Sprintf("%v", ev["RecordId"]))
	if recordID != "" && recordID != "<nil>" {
		return "record:" + recordID
	}
	return fmt.Sprintf(
		"fallback:%v|%v|%v|%v",
		ev["TimeCreated"],
		ev["ProviderName"],
		ev["Id"],
		ev["Message"],
	)
}

func normalizeEventTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "-"
	}
	if strings.HasPrefix(raw, "/Date(") && strings.HasSuffix(raw, ")/") {
		body := strings.TrimSuffix(strings.TrimPrefix(raw, "/Date("), ")/")
		if idx := strings.IndexAny(body, "+-"); idx > 0 {
			body = body[:idx]
		}
		var ms int64
		_, _ = fmt.Sscanf(body, "%d", &ms)
		if ms > 0 {
			return time.UnixMilli(ms).Local().Format("2006-01-02 15:04:05")
		}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("2006-01-02 15:04:05")
	}
	return raw
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(t, "%d", &n)
		return n
	default:
		return 0
	}
}

func isUnauthorizedEventLogError(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	return strings.Contains(s, "unauthorized") || strings.Contains(s, "access is denied")
}

func isHyperVLogName(name string) bool {
	s := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(s, "hyper-v") || strings.Contains(s, "hyperv")
}

func resolveLogFile(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return path, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	files := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, filepath.Join(path, e.Name()))
	}
	if len(files) == 0 {
		return "", errors.New("no log file found")
	}
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		if fi == nil || fj == nil {
			return files[i] < files[j]
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return files[0], nil
}

func lastLines(s string, n int) string {
	if n <= 0 {
		return s
	}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	buf := make([]string, 0, n)
	for sc.Scan() {
		buf = append(buf, sc.Text())
		if len(buf) > n {
			buf = buf[1:]
		}
	}
	return strings.Join(buf, "\n")
}

func filterLines(s, keyword string) string {
	if strings.TrimSpace(keyword) == "" {
		return s
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	out := make([]string, 0)
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(strings.ToLower(line), kw) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
