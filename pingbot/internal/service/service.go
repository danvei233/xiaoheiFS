package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"pingbot/internal/client"
	"pingbot/internal/collector"
	"pingbot/internal/config"
	"pingbot/internal/logreader"
)

type Envelope struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

const (
	// maxInboundFrameBytes caps inbound WS frames (8 MiB).
	maxInboundFrameBytes = 8 << 20
	// maxLogLines caps a single request_log reply so a server cannot force the
	// agent to emit an unbounded stream.
	maxLogLines = 100000
	// maxRuntimeIntervalSec caps heartbeat/snapshot intervals so a hostile
	// server config cannot cause ticker overflow or pathological load.
	maxRuntimeIntervalSec = 3600
	// maxLogChunkBytes caps the per-chunk size the agent will emit.
	maxLogChunkBytes = 1 << 20
	// maxConcurrentLogStreams bounds concurrent request_log follow streams per
	// connection so a server cannot spawn unbounded long-lived goroutines.
	maxConcurrentLogStreams = 4
)

// clampRuntimeInterval bounds an interval to [1, maxRuntimeIntervalSec],
// falling back to def when the value is non-positive.
func clampRuntimeInterval(sec, def int) int {
	if sec <= 0 {
		sec = def
	}
	if sec < 1 {
		return 1
	}
	if sec > maxRuntimeIntervalSec {
		return maxRuntimeIntervalSec
	}
	return sec
}

// applyRuntimeConfig derives the effective runtime config from a server
// set_config message, clamped to the same bounds used at startup and defaulting
// to the previous values for missing/zero fields. It is a pure function so the
// set_config path (which feeds time.NewTicker) is unit-testable.
func applyRuntimeConfig(incoming client.RuntimeConfig, prev client.RuntimeConfig) client.RuntimeConfig {
	out := prev
	out.HeartbeatIntervalSec = clampRuntimeInterval(incoming.HeartbeatIntervalSec, prev.HeartbeatIntervalSec)
	out.SnapshotIntervalSec = clampRuntimeInterval(incoming.SnapshotIntervalSec, prev.SnapshotIntervalSec)
	if incoming.LogChunkMaxBytes <= 0 {
		out.LogChunkMaxBytes = prev.LogChunkMaxBytes
	} else if incoming.LogChunkMaxBytes > maxLogChunkBytes {
		out.LogChunkMaxBytes = maxLogChunkBytes
	} else {
		out.LogChunkMaxBytes = incoming.LogChunkMaxBytes
	}
	return out
}

type Service struct {
	cfgPath string
	cfg     config.Config
	api     *client.APIClient
}

func New(cfgPath string, cfg config.Config) *Service {
	return &Service{
		cfgPath: cfgPath,
		cfg:     cfg,
		api:     client.New(cfg.ServerURL, cfg.TLSInsecureSkipVerify),
	}
}

func (s *Service) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if err := s.ensureEnrollment(); err != nil {
			log.Printf("enroll failed: %v", err)
			if !sleepCtx(ctx, backoff) {
				return nil
			}
			backoff = minDuration(backoff*2, 30*time.Second)
			continue
		}
		accessToken, runtimeCfg, err := s.api.AuthToken(s.cfg.ProbeID, s.cfg.ProbeSecret)
		if err != nil {
			log.Printf("auth token failed: %v", err)
			if !sleepCtx(ctx, backoff) {
				return nil
			}
			backoff = minDuration(backoff*2, 30*time.Second)
			continue
		}
		runtimeCfg.HeartbeatIntervalSec = clampRuntimeInterval(runtimeCfg.HeartbeatIntervalSec, 10)
		runtimeCfg.SnapshotIntervalSec = clampRuntimeInterval(runtimeCfg.SnapshotIntervalSec, 60)
		if runtimeCfg.LogChunkMaxBytes <= 0 {
			runtimeCfg.LogChunkMaxBytes = 16384
		}
		if runtimeCfg.LogChunkMaxBytes > maxLogChunkBytes {
			runtimeCfg.LogChunkMaxBytes = maxLogChunkBytes
		}
		if err := s.wsLoop(ctx, accessToken, runtimeCfg); err != nil {
			log.Printf("ws disconnected: %v", err)
		}
		if !sleepCtx(ctx, backoff) {
			return nil
		}
		backoff = minDuration(backoff*2, 30*time.Second)
	}
}

// sleepCtx sleeps for d or returns immediately when ctx is cancelled, so a
// reconnect backoff never blocks shutdown.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func (s *Service) ensureEnrollment() error {
	if s.cfg.ProbeID > 0 && strings.TrimSpace(s.cfg.ProbeSecret) != "" {
		log.Printf("credential loaded probe_id=%d", s.cfg.ProbeID)
		return nil
	}
	if strings.TrimSpace(s.cfg.EnrollToken) == "" {
		return fmt.Errorf("missing enroll_token")
	}
	agentID := strings.TrimSpace(s.cfg.HostnameAlias)
	if agentID == "" {
		agentID = runtime.GOOS + "-" + probeMachineTag()
	}
	name := strings.TrimSpace(s.cfg.HostnameAlias)
	osType := runtime.GOOS
	probeID, probeSecret, _, runtimeCfg, err := s.api.Enroll(s.cfg.EnrollToken, agentID, name, osType)
	if err != nil {
		return err
	}
	s.cfg.ProbeID = probeID
	s.cfg.ProbeSecret = probeSecret
	s.cfg.EnrollToken = ""
	if err := config.Save(s.cfgPath, s.cfg); err != nil {
		return err
	}
	log.Printf("enrolled probe_id=%d heartbeat=%ds snapshot=%ds", probeID, runtimeCfg.HeartbeatIntervalSec, runtimeCfg.SnapshotIntervalSec)
	return nil
}

func (s *Service) wsLoop(ctx context.Context, accessToken string, runtimeCfg client.RuntimeConfig) error {
	wsURL, err := toWSURL(s.cfg.ServerURL, "/api/v1/probe/ws")
	if err != nil {
		return err
	}
	// Never log a URL that could carry userinfo or a query (both would leak
	// credentials). Only the scheme/host/path is safe to print.
	log.Printf("dial ws url=%s", SanitizeURL(wsURL))
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: s.cfg.TLSInsecureSkipVerify},
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+accessToken)
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Bound inbound frames so a compromised/misbehaving server cannot make the
	// agent buffer an unbounded frame.
	conn.SetReadLimit(maxInboundFrameBytes)
	log.Printf("ws connected probe_id=%d", s.cfg.ProbeID)

	sendMu := sync.Mutex{}
	send := func(env Envelope) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		return conn.WriteJSON(env)
	}
	// logStreamSem bounds how many concurrent request_log streams a single
	// connection may run, so a malicious/buggy server cannot spawn unbounded
	// goroutines (each follow stream can live up to 10 minutes).
	logStreamSem := make(chan struct{}, maxConcurrentLogStreams)
	// connCtx is cancelled when this connection tears down, so in-flight log
	// streams stop instead of lingering after a reconnect.
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()

	_ = send(Envelope{
		Type: "hello",
		Payload: map[string]any{
			"os_type": runtime.GOOS,
		},
	})
	log.Printf("hello sent os=%s", runtime.GOOS)

	hbTicker := time.NewTicker(time.Duration(runtimeCfg.HeartbeatIntervalSec) * time.Second)
	defer hbTicker.Stop()
	ssTicker := time.NewTicker(time.Duration(runtimeCfg.SnapshotIntervalSec) * time.Second)
	defer ssTicker.Stop()
	statusTicker := time.NewTicker(30 * time.Second)
	defer statusTicker.Stop()
	cfgCh := make(chan client.RuntimeConfig, 1)

	errCh := make(chan error, 1)
	sendSnapshot := func(trigger string) {
		snapshot, warnings := collector.Snapshot(ctx, s.cfg.HostnameAlias)
		if len(warnings) > 0 {
			log.Printf("snapshot warnings trigger=%s details=%s", trigger, strings.Join(warnings, " | "))
		}
		log.Printf(
			"snapshot collected trigger=%s host=%s cpu=%.1f%% mem=%.1f%% disks=%d ports=%d",
			trigger,
			readSnapshotHost(snapshot),
			readSnapshotPercent(snapshot, "cpu", "usage_percent"),
			readSnapshotPercent(snapshot, "memory", "usage_percent"),
			readSnapshotSliceLen(snapshot, "disks"),
			readSnapshotSliceLen(snapshot, "ports"),
		)
		if err := send(Envelope{
			Type: "snapshot",
			Payload: map[string]any{
				"at":       time.Now().Format(time.RFC3339),
				"os_type":  runtime.GOOS,
				"snapshot": snapshot,
			},
		}); err != nil {
			log.Printf("snapshot send failed: %v", err)
			return
		}
		log.Printf("snapshot sent trigger=%s", trigger)
	}

	go func() {
		for {
			var msg Envelope
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			if err := conn.ReadJSON(&msg); err != nil {
				errCh <- err
				return
			}
			switch strings.TrimSpace(msg.Type) {
			case "set_config":
				raw, _ := json.Marshal(msg.Payload)
				var p struct {
					Config client.RuntimeConfig `json:"config"`
				}
				_ = json.Unmarshal(raw, &p)
				log.Printf(
					"runtime config received heartbeat=%ds snapshot=%ds log_chunk=%d",
					p.Config.HeartbeatIntervalSec,
					p.Config.SnapshotIntervalSec,
					p.Config.LogChunkMaxBytes,
				)
				select {
				case cfgCh <- p.Config:
				default:
				}
			case "ping":
				_ = send(Envelope{Type: "pong", RequestID: msg.RequestID})
			case "request_log":
				log.Printf("request_log received request_id=%s", strings.TrimSpace(msg.RequestID))
				go func() {
					select {
					case logStreamSem <- struct{}{}:
						defer func() { <-logStreamSem }()
						s.handleLogRequest(connCtx, send, msg, runtimeCfg.LogChunkMaxBytes)
					case <-connCtx.Done():
					}
				}()
			case "request_snapshot":
				log.Printf("request_snapshot received request_id=%s", strings.TrimSpace(msg.RequestID))
				sendSnapshot("request_snapshot:" + strings.TrimSpace(msg.RequestID))
			case "port_check_request":
				log.Printf("port_check_request received request_id=%s", strings.TrimSpace(msg.RequestID))
				sendSnapshot("port_check_request:" + strings.TrimSpace(msg.RequestID))
			}
		}
	}()

	sendHeartbeat := func() {
		if err := send(Envelope{
			Type: "heartbeat",
			Payload: map[string]any{
				"at": time.Now().Format(time.RFC3339),
			},
		}); err != nil {
			log.Printf("heartbeat send failed: %v", err)
		}
	}
	sendHeartbeat()
	sendSnapshot("startup")

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case incoming := <-cfgCh:
			// A compromised/misbehaving server could send a set_config with an
			// enormous interval (overflowing time.Duration into a negative ticker
			// and panicking) or a huge chunk cap. applyRuntimeConfig applies the
			// same clamps used at startup so a hostile config can never drive a
			// pathological ticker.
			next := applyRuntimeConfig(incoming, runtimeCfg)
			if next.HeartbeatIntervalSec != runtimeCfg.HeartbeatIntervalSec {
				hbTicker.Stop()
				hbTicker = time.NewTicker(time.Duration(next.HeartbeatIntervalSec) * time.Second)
			}
			if next.SnapshotIntervalSec != runtimeCfg.SnapshotIntervalSec {
				ssTicker.Stop()
				ssTicker = time.NewTicker(time.Duration(next.SnapshotIntervalSec) * time.Second)
			}
			runtimeCfg = next
		case <-hbTicker.C:
			sendHeartbeat()
		case <-ssTicker.C:
			sendSnapshot("ticker")
		case <-statusTicker.C:
			log.Printf(
				"running probe_id=%d heartbeat=%ds snapshot=%ds",
				s.cfg.ProbeID,
				runtimeCfg.HeartbeatIntervalSec,
				runtimeCfg.SnapshotIntervalSec,
			)
		}
	}
}

func (s *Service) handleLogRequest(ctx context.Context, send func(Envelope) error, msg Envelope, maxChunk int) {
	raw, _ := json.Marshal(msg.Payload)
	var payload struct {
		SessionID string `json:"session_id"`
		Source    string `json:"source"`
		Keyword   string `json:"keyword"`
		Follow    bool   `json:"follow"`
		Lines     int    `json:"lines"`
	}
	_ = json.Unmarshal(raw, &payload)
	resolvedSource := s.resolveLogSource(payload.Source)
	log.Printf(
		"log request session=%s source=%s lines=%d follow=%v keyword=%q",
		strings.TrimSpace(payload.SessionID),
		strings.TrimSpace(resolvedSource),
		payload.Lines,
		payload.Follow,
		strings.TrimSpace(payload.Keyword),
	)
	if payload.Lines <= 0 {
		payload.Lines = 300
	}
	if payload.Lines > maxLogLines {
		payload.Lines = maxLogLines
	}
	if maxChunk <= 0 {
		maxChunk = 16384
	}
	if maxChunk > maxLogChunkBytes {
		maxChunk = maxLogChunkBytes
	}
	emittedLines := 0
	var sendErr error

	emit := func(line string) bool {
		line = redactSensitive(strings.TrimSpace(line))
		if line == "" {
			return true
		}
		emittedLines++
		for _, chunk := range splitChunk(line, maxChunk) {
			if err := send(Envelope{
				Type:      "log_chunk",
				RequestID: msg.RequestID,
				Payload: map[string]any{
					"session_id": payload.SessionID,
					"chunk":      chunk,
				},
			}); err != nil {
				sendErr = err
				log.Printf("log chunk send failed session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), err)
				return false
			}
		}
		return true
	}

	if err := logreader.Stream(ctx, resolvedSource, payload.Keyword, payload.Lines, payload.Follow, emit); err != nil {
		log.Printf("log stream failed session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), err)
		if serr := send(Envelope{
			Type:      "log_chunk",
			RequestID: msg.RequestID,
			Payload: map[string]any{
				"session_id": payload.SessionID,
				"chunk":      "[error] " + err.Error(),
			},
		}); serr != nil {
			log.Printf("log error chunk send failed session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), serr)
		}
	} else if sendErr != nil {
		log.Printf("log stream interrupted by send failure session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), sendErr)
	} else if emittedLines == 0 {
		hint := fmt.Sprintf(
			"[info] no log lines emitted source=%s lines=%d follow=%v keyword=%q",
			strings.TrimSpace(resolvedSource),
			payload.Lines,
			payload.Follow,
			strings.TrimSpace(payload.Keyword),
		)
		log.Printf("log stream empty session=%s %s", strings.TrimSpace(payload.SessionID), hint)
		if serr := send(Envelope{
			Type:      "log_chunk",
			RequestID: msg.RequestID,
			Payload: map[string]any{
				"session_id": payload.SessionID,
				"chunk":      hint,
			},
		}); serr != nil {
			log.Printf("log empty hint send failed session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), serr)
		}
	}
	log.Printf("log stream done session=%s source=%s emitted_lines=%d", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), emittedLines)
	if err := send(Envelope{
		Type:      "log_end",
		RequestID: msg.RequestID,
		Payload: map[string]any{
			"session_id": payload.SessionID,
		},
	}); err != nil {
		log.Printf("log end send failed session=%s source=%s err=%v", strings.TrimSpace(payload.SessionID), strings.TrimSpace(resolvedSource), err)
	}
}

// redactSensitive masks credentials that could appear in log/event content
// (Authorization: Bearer <token>, psk_ probe secrets) so they are never
// shipped back to the server or written into local logs.
func redactSensitive(line string) string {
	// Authorization header values (Bearer or raw key) are fully masked first;
	// the value class cannot match the [REDACTED] marker so this is idempotent.
	line = redactAuthHeaderRe.ReplaceAllString(line, "$1[REDACTED]")
	line = redactBearerRe.ReplaceAllString(line, "$1[REDACTED]")
	line = redactPSKRe.ReplaceAllString(line, "psk_[REDACTED]")
	return line
}

var (
	redactBearerRe = regexp.MustCompile(`(?i)(Bearer\s+)([A-Za-z0-9._~+/=-]+)`)
	redactPSKRe    = regexp.MustCompile(`psk_[A-Za-z0-9]{48,}`)
	// redactAuthHeaderRe masks the value of an Authorization header, whether it
	// is a Bearer token or a raw API key. It is idempotent because the
	// [REDACTED] marker is not matched by the value class.
	redactAuthHeaderRe = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(?:Bearer\s+)?([A-Za-z0-9._~+/=-]+)`)
)

func (s *Service) resolveLogSource(requestSource string) string {
	source := strings.TrimSpace(requestSource)
	if source == "" || strings.HasPrefix(strings.ToLower(source), "file:") {
		cfgSource := strings.TrimSpace(s.cfg.LogFileSource)
		if cfgSource == "" {
			return "file:logs"
		}
		if strings.HasPrefix(strings.ToLower(cfgSource), "file:") {
			return cfgSource
		}
		return "file:" + cfgSource
	}
	return source
}

func splitChunk(s string, max int) []string {
	if max <= 0 || len(s) <= max {
		return []string{s}
	}
	out := make([]string, 0, len(s)/max+1)
	for len(s) > max {
		out = append(out, s[:max])
		s = s[max:]
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

func toWSURL(baseURL, path string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	u.Path = path
	u.RawQuery = ""
	return u.String(), nil
}

// SanitizeURL returns a URL string safe to log: userinfo and query (either can
// carry credentials) are stripped, leaving only scheme://host/path.
func SanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid-url]"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func probeMachineTag() string {
	h, _ := os.Hostname()
	h = strings.TrimSpace(h)
	if h == "" {
		return "unknown"
	}
	return strings.ReplaceAll(strings.ToLower(h), " ", "-")
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func readSnapshotHost(snapshot map[string]any) string {
	system, ok := snapshot["system"].(map[string]any)
	if !ok {
		return "-"
	}
	host := strings.TrimSpace(fmt.Sprintf("%v", system["hostname"]))
	if host == "" {
		return "-"
	}
	return host
}

func readSnapshotPercent(snapshot map[string]any, section, key string) float64 {
	m, ok := snapshot[section].(map[string]any)
	if !ok {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func readSnapshotSliceLen(snapshot map[string]any, key string) int {
	items, ok := snapshot[key].([]map[string]any)
	if ok {
		return len(items)
	}
	raw, ok := snapshot[key].([]any)
	if ok {
		return len(raw)
	}
	return 0
}
