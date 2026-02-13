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
			time.Sleep(backoff)
			backoff = minDuration(backoff*2, 30*time.Second)
			continue
		}
		accessToken, runtimeCfg, err := s.api.AuthToken(s.cfg.ProbeID, s.cfg.ProbeSecret)
		if err != nil {
			log.Printf("auth token failed: %v", err)
			time.Sleep(backoff)
			backoff = minDuration(backoff*2, 30*time.Second)
			continue
		}
		if runtimeCfg.HeartbeatIntervalSec <= 0 {
			runtimeCfg.HeartbeatIntervalSec = 10
		}
		if runtimeCfg.SnapshotIntervalSec <= 0 {
			runtimeCfg.SnapshotIntervalSec = 60
		}
		if runtimeCfg.LogChunkMaxBytes <= 0 {
			runtimeCfg.LogChunkMaxBytes = 16384
		}
		if err := s.wsLoop(ctx, accessToken, runtimeCfg); err != nil {
			log.Printf("ws disconnected: %v", err)
		}
		time.Sleep(backoff)
		backoff = minDuration(backoff*2, 30*time.Second)
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
	log.Printf("dial ws url=%s", wsURL)
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
	log.Printf("ws connected probe_id=%d", s.cfg.ProbeID)

	sendMu := sync.Mutex{}
	send := func(env Envelope) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
		return conn.WriteJSON(env)
	}

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
		snapshot, warnings := collector.Snapshot(context.Background(), s.cfg.HostnameAlias)
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
				go s.handleLogRequest(send, msg, runtimeCfg.LogChunkMaxBytes)
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
			changedHB := false
			changedSS := false
			if incoming.HeartbeatIntervalSec > 0 && incoming.HeartbeatIntervalSec != runtimeCfg.HeartbeatIntervalSec {
				runtimeCfg.HeartbeatIntervalSec = incoming.HeartbeatIntervalSec
				changedHB = true
			}
			if incoming.SnapshotIntervalSec > 0 && incoming.SnapshotIntervalSec != runtimeCfg.SnapshotIntervalSec {
				runtimeCfg.SnapshotIntervalSec = incoming.SnapshotIntervalSec
				changedSS = true
			}
			if incoming.LogChunkMaxBytes > 0 {
				runtimeCfg.LogChunkMaxBytes = incoming.LogChunkMaxBytes
			}
			if changedHB {
				hbTicker.Stop()
				hbTicker = time.NewTicker(time.Duration(runtimeCfg.HeartbeatIntervalSec) * time.Second)
			}
			if changedSS {
				ssTicker.Stop()
				ssTicker = time.NewTicker(time.Duration(runtimeCfg.SnapshotIntervalSec) * time.Second)
			}
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

func (s *Service) handleLogRequest(send func(Envelope) error, msg Envelope, maxChunk int) {
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
	if maxChunk <= 0 {
		maxChunk = 16384
	}
	emittedLines := 0
	var sendErr error

	emit := func(line string) bool {
		line = strings.TrimSpace(line)
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

	if err := logreader.Stream(resolvedSource, payload.Keyword, payload.Lines, payload.Follow, emit); err != nil {
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
