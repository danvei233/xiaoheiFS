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

	hbTicker := time.NewTicker(time.Duration(runtimeCfg.HeartbeatIntervalSec) * time.Second)
	defer hbTicker.Stop()
	ssTicker := time.NewTicker(time.Duration(runtimeCfg.SnapshotIntervalSec) * time.Second)
	defer ssTicker.Stop()
	cfgCh := make(chan client.RuntimeConfig, 1)

	errCh := make(chan error, 1)
	sendSnapshot := func() {
		snapshot := collector.Snapshot(context.Background(), s.cfg.HostnameAlias)
		_ = send(Envelope{
			Type: "snapshot",
			Payload: map[string]any{
				"at":       time.Now().Format(time.RFC3339),
				"os_type":  runtime.GOOS,
				"snapshot": snapshot,
			},
		})
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
				select {
				case cfgCh <- p.Config:
				default:
				}
			case "ping":
				_ = send(Envelope{Type: "pong", RequestID: msg.RequestID})
			case "request_log":
				go s.handleLogRequest(send, msg, runtimeCfg.LogChunkMaxBytes)
			case "request_snapshot":
				sendSnapshot()
			case "port_check_request":
				sendSnapshot()
			}
		}
	}()

	sendHeartbeat := func() {
		_ = send(Envelope{
			Type: "heartbeat",
			Payload: map[string]any{
				"at": time.Now().Format(time.RFC3339),
			},
		})
	}
	sendHeartbeat()
	sendSnapshot()

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
			sendSnapshot()
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
	if payload.Lines <= 0 {
		payload.Lines = 300
	}
	if maxChunk <= 0 {
		maxChunk = 16384
	}

	emit := func(line string) bool {
		line = strings.TrimSpace(line)
		if line == "" {
			return true
		}
		for _, chunk := range splitChunk(line, maxChunk) {
			_ = send(Envelope{
				Type:      "log_chunk",
				RequestID: msg.RequestID,
				Payload: map[string]any{
					"session_id": payload.SessionID,
					"chunk":      chunk,
				},
			})
		}
		return true
	}

	if err := logreader.Stream(payload.Source, payload.Keyword, payload.Lines, payload.Follow, emit); err != nil {
		_ = send(Envelope{
			Type:      "log_chunk",
			RequestID: msg.RequestID,
			Payload: map[string]any{
				"session_id": payload.SessionID,
				"chunk":      "[error] " + err.Error(),
			},
		})
	}
	_ = send(Envelope{
		Type:      "log_end",
		RequestID: msg.RequestID,
		Payload: map[string]any{
			"session_id": payload.SessionID,
		},
	})
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
