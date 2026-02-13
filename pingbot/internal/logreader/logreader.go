package logreader

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
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
		return fmt.Errorf("resolve file log source failed: %w", err)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("read file log failed: %w", err)
	}
	text := decodeLogBytes(b)
	text = filterLines(lastLines(text, lines), keyword)
	initialEmitted := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		initialEmitted++
		if !emit(line) {
			return nil
		}
	}
	if !follow {
		return nil
	}
	if initialEmitted == 0 {
		hint := fmt.Sprintf("[info] no historical lines in %s (keyword=%q)", target, strings.TrimSpace(keyword))
		if !emit(hint) {
			return nil
		}
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
		chunkText := decodeLogBytes(buf)
		for _, line := range strings.Split(chunkText, "\n") {
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
	log.Printf("eventlog request name=%s profile=%s lines=%d follow=%v keyword=%q", logName, profile, lines, follow, strings.TrimSpace(keyword))
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
			log.Printf("eventlog fetch failed candidate=%s err=%s", candidate, msg)
			continue
		}
		markEventLogPreferred(logName, candidate)
		seen := map[string]struct{}{}
		latest := ""
		keptCount := 0
		emittedCount := 0
		for _, ev := range events {
			if !shouldKeepEvent(ev, profile) {
				continue
			}
			keptCount++
			key := eventUniqueKey(ev)
			seen[key] = struct{}{}
			if ts := eventCreatedAtString(ev); ts > latest {
				latest = ts
			}
			line := formatEventLine(ev)
			if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				continue
			}
			emittedCount++
			if !emit(line) {
				return nil
			}
		}
		// "important" can be too strict on some Windows hosts (many useful events are Information).
		if profile == "important" && keptCount == 0 {
			fallbackKept, fallbackEmitted := emitImportantFallback(events, keyword, emit)
			keptCount += fallbackKept
			emittedCount += fallbackEmitted
		}
		// Power events may be sparse; when nothing matched, widen history once.
		if profile == "power" && keptCount == 0 {
			wideLines := initialLines * 8
			if wideLines < 2000 {
				wideLines = 2000
			}
			if wideLines > 5000 {
				wideLines = 5000
			}
			if wideLines > initialLines {
				moreEvents, moreErr := fetchWinEvents(candidate, wideLines, "")
				if moreErr == nil {
					for _, ev := range moreEvents {
						if !shouldKeepEvent(ev, profile) {
							continue
						}
						key := eventUniqueKey(ev)
						if _, ok := seen[key]; ok {
							continue
						}
						seen[key] = struct{}{}
						keptCount++
						if ts := eventCreatedAtString(ev); ts > latest {
							latest = ts
						}
						line := formatEventLine(ev)
						if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
							continue
						}
						emittedCount++
						if !emit(line) {
							return nil
						}
					}
				}
			}
		}
		log.Printf("eventlog fetched candidate=%s raw=%d kept=%d emitted=%d", candidate, len(events), keptCount, emittedCount)
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
				incKept := 0
				incEmitted := 0
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
					incKept++
					line := formatEventLine(ev)
					if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
						continue
					}
					incEmitted++
					if !emit(line) {
						return nil
					}
				}
				if incKept > 0 || incEmitted > 0 {
					log.Printf("eventlog follow candidate=%s kept=%d emitted=%d", candidate, incKept, incEmitted)
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

func emitImportantFallback(events []map[string]any, keyword string, emit func(string) bool) (kept, emitted int) {
	interestingProvider := []string{
		"kernel-power", "disk", "ntfs", "volmgr", "whea", "eventlog", "application popup",
		"hyper-v-vmswitch", "hyper-v-compute", "hyper-v",
	}
	interestingIDs := map[int]struct{}{
		26: {}, 41: {}, 51: {}, 55: {}, 67: {}, 129: {}, 153: {}, 157: {},
		291: {}, 292: {}, 10020: {}, 6008: {},
	}
	for _, ev := range events {
		level := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ev["LevelDisplayName"])))
		if level != "information" && level != "info" {
			continue
		}
		id := intFromAny(ev["Id"])
		provider := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", ev["ProviderName"])))
		_, hitID := interestingIDs[id]
		hitProvider := false
		for _, p := range interestingProvider {
			if strings.Contains(provider, p) {
				hitProvider = true
				break
			}
		}
		if !hitID && !hitProvider {
			continue
		}
		kept++
		line := formatEventLine(ev)
		if keyword != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			continue
		}
		emitted++
		if !emit(line) {
			return kept, emitted
		}
	}
	return kept, emitted
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
	escapedLog := strings.ReplaceAll(strings.TrimSpace(logName), "'", "''")
	if escapedLog == "" {
		escapedLog = "System"
	}
	psPrefix := "$ErrorActionPreference='Stop';" +
		"$ProgressPreference='SilentlyContinue';" +
		"$WarningPreference='SilentlyContinue';" +
		"$InformationPreference='SilentlyContinue';" +
		"$VerbosePreference='SilentlyContinue';" +
		"[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false);" +
		"$OutputEncoding=[System.Text.UTF8Encoding]::new($false);"
	psBody := ""
	if strings.TrimSpace(startTimeISO) == "" {
		psBody = fmt.Sprintf(
			"Get-WinEvent -LogName '%s' -MaxEvents %d -ErrorAction Stop | ForEach-Object { [PSCustomObject]@{ TimeCreated=$_.TimeCreated; RecordId=$_.RecordId; Id=$_.Id; Level=$_.Level; LevelDisplayName=$_.LevelDisplayName; ProviderName=$_.ProviderName; Message=$_.Message; Properties=@($_.Properties | ForEach-Object { $_.Value }) } } | ConvertTo-Json -Depth 6 -Compress",
			escapedLog,
			maxEvents,
		)
	} else {
		escapedStart := strings.ReplaceAll(strings.TrimSpace(startTimeISO), "'", "''")
		psBody = fmt.Sprintf(
			"$st=Get-Date -Date '%s'; Get-WinEvent -FilterHashtable @{LogName='%s'; StartTime=$st} -MaxEvents %d -ErrorAction Stop | ForEach-Object { [PSCustomObject]@{ TimeCreated=$_.TimeCreated; RecordId=$_.RecordId; Id=$_.Id; Level=$_.Level; LevelDisplayName=$_.LevelDisplayName; ProviderName=$_.ProviderName; Message=$_.Message; Properties=@($_.Properties | ForEach-Object { $_.Value }) } } | ConvertTo-Json -Depth 6 -Compress",
			escapedStart,
			escapedLog,
			maxEvents,
		)
	}
	ps := psPrefix + psBody
	encoded := encodePowerShellCommand(ps)
	cmd := exec.Command(
		"powershell",
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encoded,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(decodeWindowsOutput(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	text := strings.TrimSpace(decodeWindowsOutput(out))
	text = strings.TrimPrefix(text, "\ufeff")
	if text == "" || text == "null" {
		return nil, nil
	}
	text = extractJSONPayload(text)
	events := parseEventLogJSON([]byte(text))
	if len(events) == 0 {
		if len(text) > 220 {
			text = text[:220] + "..."
		}
		return nil, fmt.Errorf("powershell returned unparsable json: %s", strings.TrimSpace(text))
	}
	return events, nil
}

func encodePowerShellCommand(script string) string {
	u16 := utf16.Encode([]rune(script))
	buf := make([]byte, 0, len(u16)*2)
	for _, r := range u16 {
		buf = append(buf, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func extractJSONPayload(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	firstObj := strings.IndexByte(s, '{')
	firstArr := strings.IndexByte(s, '[')
	start := -1
	if firstObj >= 0 && firstArr >= 0 {
		if firstObj < firstArr {
			start = firstObj
		} else {
			start = firstArr
		}
	} else if firstObj >= 0 {
		start = firstObj
	} else if firstArr >= 0 {
		start = firstArr
	}
	if start < 0 {
		return s
	}
	lastObj := strings.LastIndexByte(s, '}')
	lastArr := strings.LastIndexByte(s, ']')
	end := -1
	if lastObj > lastArr {
		end = lastObj
	} else {
		end = lastArr
	}
	if end < start {
		return s[start:]
	}
	return strings.TrimSpace(s[start : end+1])
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

type winEventXML struct {
	System struct {
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
		EventRecordID string `xml:"EventRecordID"`
		EventID       string `xml:"EventID"`
		Level         string `xml:"Level"`
		Provider      struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
	} `xml:"System"`
	RenderingInfo struct {
		Level   string `xml:"Level"`
		Message string `xml:"Message"`
	} `xml:"RenderingInfo"`
	EventData struct {
		InnerXML string `xml:",innerxml"`
		Data     []struct {
			Name  string `xml:"Name,attr"`
			Value string `xml:",chardata"`
		} `xml:"Data"`
	} `xml:"EventData"`
	UserData struct {
		InnerXML string `xml:",innerxml"`
	} `xml:"UserData"`
}

func parseWinEventXML(raw []byte) ([]map[string]any, error) {
	text := decodeWindowsOutput(raw)
	blocks := splitWinEventBlocks(text)
	if len(blocks) == 0 {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return nil, nil
		}
		return nil, errors.New("wevtutil returned no event xml blocks")
	}
	out := make([]map[string]any, 0, len(blocks))
	parseFailures := 0
	for _, block := range blocks {
		var ev winEventXML
		if err := xml.Unmarshal([]byte(block), &ev); err != nil {
			parseFailures++
			continue
		}
		id := intFromAny(strings.TrimSpace(ev.System.EventID))
		level := strings.TrimSpace(ev.RenderingInfo.Level)
		if level == "" {
			level = eventLevelText(strings.TrimSpace(ev.System.Level))
		}
		msg := strings.TrimSpace(ev.RenderingInfo.Message)
		if msg == "" {
			msg = buildEventDataSummary(ev.EventData.Data)
		}
		rawPayload := buildEventPayloadSummary(ev.EventData.InnerXML, ev.UserData.InnerXML)
		if msg == "" {
			msg = rawPayload
		}
		out = append(out, map[string]any{
			"TimeCreated":      strings.TrimSpace(ev.System.TimeCreated.SystemTime),
			"RecordId":         strings.TrimSpace(ev.System.EventRecordID),
			"Id":               id,
			"LevelDisplayName": level,
			"ProviderName":     strings.TrimSpace(ev.System.Provider.Name),
			"Message":          msg,
			"RawPayload":       rawPayload,
		})
	}
	if len(out) == 0 && parseFailures > 0 {
		return nil, errors.New("failed to parse wevtutil xml output")
	}
	return out, nil
}

func splitWinEventBlocks(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	events := make([]string, 0)
	start := 0
	for {
		i := strings.Index(text[start:], "<Event")
		if i < 0 {
			break
		}
		i += start
		j := strings.Index(text[i:], "</Event>")
		if j < 0 {
			break
		}
		j += i + len("</Event>")
		events = append(events, text[i:j])
		start = j
	}
	return events
}

func eventLevelText(raw string) string {
	switch strings.TrimSpace(raw) {
	case "1":
		return "Critical"
	case "2":
		return "Error"
	case "3":
		return "Warning"
	case "4":
		return "Information"
	case "5":
		return "Verbose"
	default:
		return ""
	}
}

func buildEventDataSummary(data []struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:",chardata"`
}) string {
	if len(data) == 0 {
		return ""
	}
	parts := make([]string, 0, len(data))
	for _, item := range data {
		name := strings.TrimSpace(item.Name)
		value := strings.TrimSpace(item.Value)
		if name == "" && value == "" {
			continue
		}
		value = compactEventDataValue(name, value)
		if name == "" {
			parts = append(parts, value)
			continue
		}
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, " | ")
}

func compactEventDataValue(name, value string) string {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	if nameLower == "hivename" {
		return compactWindowsPath(value)
	}
	// Generic long value compaction to keep one-line readability.
	if len(value) > 140 {
		return value[:120] + "...(" + fmt.Sprintf("%d", len(value)) + " chars)"
	}
	return value
}

func compactWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	normalized := strings.ReplaceAll(path, "/", "\\")
	parts := strings.Split(normalized, "\\")
	trimmed := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "?" {
			continue
		}
		trimmed = append(trimmed, p)
	}
	if len(trimmed) <= 3 {
		return normalized
	}
	return "...\\" + strings.Join(trimmed[len(trimmed)-3:], "\\")
}

func buildEventPayloadSummary(eventDataXML, userDataXML string) string {
	joined := strings.TrimSpace(eventDataXML + " " + userDataXML)
	if joined == "" {
		return ""
	}
	plain := stripXMLTags(joined)
	plain = strings.Join(strings.Fields(plain), " ")
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	const maxLen = 260
	if len(plain) > maxLen {
		return plain[:maxLen] + "..."
	}
	return plain
}

func stripXMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
			b.WriteRune(' ')
		case '>':
			inTag = false
			b.WriteRune(' ')
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return htmlEntityDecode(b.String())
}

func htmlEntityDecode(s string) string {
	replacer := strings.NewReplacer(
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", "\"",
		"&apos;", "'",
	)
	return replacer.Replace(s)
}

func decodeWindowsOutput(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	// UTF-16 LE BOM
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		return decodeUTF16LE(raw[2:])
	}
	// Heuristic: many NUL bytes usually means UTF-16 LE.
	nullCount := 0
	for i := 1; i < len(raw); i += 2 {
		if raw[i] == 0x00 {
			nullCount++
		}
	}
	if len(raw) > 8 && nullCount >= len(raw)/6 {
		return decodeUTF16LE(raw)
	}
	return string(raw)
}

func decodeUTF16LE(raw []byte) string {
	if len(raw) < 2 {
		return string(raw)
	}
	u16 := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		u16 = append(u16, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	return string(utf16.Decode(u16))
}

func parseWinEventText(raw []byte) []map[string]any {
	text := decodeWindowsOutput(raw)
	text = strings.ReplaceAll(text, "\x00", "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	events := make([]map[string]any, 0)
	current := map[string]any{}
	inDesc := false
	descBuf := make([]string, 0, 8)

	flush := func() {
		if len(current) == 0 {
			return
		}
		if len(descBuf) > 0 {
			current["Message"] = strings.TrimSpace(strings.Join(descBuf, " | "))
		}
		if _, ok := current["LevelDisplayName"]; !ok {
			current["LevelDisplayName"] = ""
		}
		if _, ok := current["ProviderName"]; !ok {
			current["ProviderName"] = ""
		}
		if _, ok := current["TimeCreated"]; !ok {
			current["TimeCreated"] = ""
		}
		if _, ok := current["Id"]; !ok {
			current["Id"] = 0
		}
		events = append(events, current)
		current = map[string]any{}
		inDesc = false
		descBuf = descBuf[:0]
	}

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, " \t")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Event[") {
			flush()
			continue
		}
		if inDesc {
			// Description lines are not guaranteed to be indented on all hosts/locales.
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "Event[") {
				flush()
				continue
			}
			if looksLikeTextFieldLine(trimmed) {
				inDesc = false
			} else {
				descBuf = append(descBuf, trimmed)
				continue
			}
		}
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "Source:"):
			current["ProviderName"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Source:"))
		case strings.HasPrefix(trimmed, "Date:"):
			current["TimeCreated"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Date:"))
		case strings.HasPrefix(trimmed, "Event ID:"):
			current["Id"] = intFromAny(strings.TrimSpace(strings.TrimPrefix(trimmed, "Event ID:")))
		case strings.HasPrefix(trimmed, "Level:"):
			current["LevelDisplayName"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Level:"))
		case strings.HasPrefix(trimmed, "Description:"):
			inDesc = true
		}
	}
	flush()
	return events
}

func looksLikeTextFieldLine(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, "Log Name:"),
		strings.HasPrefix(trimmed, "Source:"),
		strings.HasPrefix(trimmed, "Date:"),
		strings.HasPrefix(trimmed, "Event ID:"),
		strings.HasPrefix(trimmed, "Task:"),
		strings.HasPrefix(trimmed, "Level:"),
		strings.HasPrefix(trimmed, "Opcode:"),
		strings.HasPrefix(trimmed, "Keyword:"),
		strings.HasPrefix(trimmed, "User:"),
		strings.HasPrefix(trimmed, "User Name:"),
		strings.HasPrefix(trimmed, "Computer:"),
		strings.HasPrefix(trimmed, "Description:"):
		return true
	default:
		return false
	}
}

func shouldKeepEvent(ev map[string]any, profile string) bool {
	level := normalizeEventLevel(fmt.Sprintf("%v", ev["LevelDisplayName"]))
	levelNum := intFromAny(ev["Level"])
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
		isImportantLevel := levelNum == 1 || levelNum == 2 || levelNum == 3 ||
			level == "critical" || level == "error" || level == "warning"
		if !isImportantLevel {
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

func normalizeEventLevel(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "critical", "严重", "危急", "致命":
		return "critical"
	case "error", "错误":
		return "error"
	case "warning", "warn", "警告":
		return "warning"
	case "information", "info", "信息":
		return "information"
	case "verbose", "详细", "调试":
		return "verbose"
	default:
		return s
	}
}

func formatEventLine(ev map[string]any) string {
	timeRaw := strings.TrimSpace(fmt.Sprintf("%v", ev["TimeCreated"]))
	timeRaw = normalizeEventTime(timeRaw)
	level := strings.TrimSpace(fmt.Sprintf("%v", ev["LevelDisplayName"]))
	if level == "" {
		level = eventLevelText(fmt.Sprintf("%d", intFromAny(ev["Level"])))
	}
	provider := strings.TrimSpace(fmt.Sprintf("%v", ev["ProviderName"]))
	msg := strings.TrimSpace(fmt.Sprintf("%v", ev["Message"]))
	if msg == "" {
		msg = strings.TrimSpace(fmt.Sprintf("%v", ev["RawPayload"]))
	}
	if msg == "" {
		msg = compactProperties(ev["Properties"])
	}
	if msg == "" {
		msg = "(no message payload)"
	}
	msg = strings.ReplaceAll(msg, "\r\n", " | ")
	msg = strings.ReplaceAll(msg, "\n", " | ")
	id := intFromAny(ev["Id"])
	recordID := strings.TrimSpace(fmt.Sprintf("%v", ev["RecordId"]))
	if recordID != "" && recordID != "<nil>" {
		return fmt.Sprintf("[%s][%s][%s][id=%d][rid=%s] %s", timeRaw, level, provider, id, recordID, msg)
	}
	return fmt.Sprintf("[%s][%s][%s][id=%d] %s", timeRaw, level, provider, id, msg)
}

func compactProperties(v any) string {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	parts := make([]string, 0, len(arr))
	for i, item := range arr {
		s := strings.TrimSpace(fmt.Sprintf("%v", item))
		if s == "" || s == "<nil>" {
			continue
		}
		s = compactEventDataValue("", s)
		parts = append(parts, fmt.Sprintf("p%d=%s", i+1, s))
		if len(parts) >= 8 {
			break
		}
	}
	return strings.Join(parts, " | ")
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
	return strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "access is denied") ||
		strings.Contains(s, "access denied") ||
		strings.Contains(s, "denied") ||
		strings.Contains(s, "拒绝访问")
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

func decodeLogBytes(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if runtime.GOOS == "windows" && !utf8.Valid(raw) {
		if gbkText, ok := decodeGBK(raw); ok {
			raw = []byte(gbkText)
		}
	}
	// Reuse Windows output decoder: handles UTF-16 BOM/heuristics.
	text := decodeWindowsOutput(raw)
	// Some UTF-16/ANSI logs still carry NUL bytes in edge cases.
	text = strings.ReplaceAll(text, "\x00", "")
	// Normalize line endings for downstream split/filter.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func decodeGBK(raw []byte) (string, bool) {
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil || len(decoded) == 0 || !utf8.Valid(decoded) {
		return "", false
	}
	return string(decoded), true
}
