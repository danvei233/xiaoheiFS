package usecase

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrProbeOffline          = errors.New("probe offline")
	ErrProbeLogSessionClosed = errors.New("probe log session closed")
)

type ProbeLogMessage struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id,omitempty"`
	Data      string    `json:"data,omitempty"`
	At        time.Time `json:"at"`
}

type probeConnState struct {
	send      func([]byte) error
	updatedAt time.Time
}

type probeLogStream struct {
	probeID   int64
	expiresAt time.Time
	closed    bool
	subs      map[chan ProbeLogMessage]struct{}
	backlog   []ProbeLogMessage
}

type ProbeHub struct {
	mu       sync.RWMutex
	conns    map[int64]*probeConnState
	sessions map[string]*probeLogStream
}

const (
	probeLogChannelBuffer = 256
	probeLogBacklogLimit  = 256
)

func NewProbeHub() *ProbeHub {
	h := &ProbeHub{
		conns:    make(map[int64]*probeConnState),
		sessions: make(map[string]*probeLogStream),
	}
	go h.gcLoop()
	return h
}

func (h *ProbeHub) RegisterConn(probeID int64, send func([]byte) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[probeID] = &probeConnState{send: send, updatedAt: time.Now()}
}

func (h *ProbeHub) UnregisterConn(probeID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, probeID)
}

func (h *ProbeHub) IsOnline(probeID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.conns[probeID]
	return ok
}

func (h *ProbeHub) SendJSON(probeID int64, payload any) error {
	h.mu.RLock()
	conn := h.conns[probeID]
	h.mu.RUnlock()
	if conn == nil || conn.send == nil {
		return ErrProbeOffline
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.send(b)
}

func (h *ProbeHub) OpenLogSession(sessionID string, probeID int64, ttl time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions[sessionID] = &probeLogStream{
		probeID:   probeID,
		expiresAt: time.Now().Add(ttl),
		subs:      make(map[chan ProbeLogMessage]struct{}),
		backlog:   make([]ProbeLogMessage, 0, probeLogBacklogLimit),
	}
}

func (h *ProbeHub) HasLogSession(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.sessions[sessionID]
	return ok
}

func (h *ProbeHub) SubscribeLogSession(sessionID string) (chan ProbeLogMessage, func(), error) {
	h.mu.Lock()
	session := h.sessions[sessionID]
	if session == nil {
		h.mu.Unlock()
		return nil, nil, ErrProbeLogSessionClosed
	}
	ch := make(chan ProbeLogMessage, probeLogChannelBuffer)
	backlog := append([]ProbeLogMessage(nil), session.backlog...)
	closed := session.closed
	if !closed {
		session.subs[ch] = struct{}{}
		session.expiresAt = time.Now().Add(10 * time.Minute)
	}
	h.mu.Unlock()

	for _, msg := range backlog {
		select {
		case ch <- msg:
		default:
		}
	}
	if closed {
		close(ch)
		return ch, func() {}, nil
	}

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		s := h.sessions[sessionID]
		if s == nil {
			return
		}
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
	}
	return ch, cancel, nil
}

func (h *ProbeHub) PublishLogChunk(sessionID, requestID, chunk string) {
	h.publishLog(sessionID, ProbeLogMessage{
		Type:      "log_chunk",
		RequestID: requestID,
		Data:      chunk,
		At:        time.Now(),
	})
}

func (h *ProbeHub) PublishLogEnd(sessionID, requestID string) {
	h.publishLog(sessionID, ProbeLogMessage{
		Type:      "log_end",
		RequestID: requestID,
		At:        time.Now(),
	})
	h.CloseLogSession(sessionID)
}

func (h *ProbeHub) publishLog(sessionID string, msg ProbeLogMessage) {
	h.mu.Lock()
	session := h.sessions[sessionID]
	if session == nil || session.closed {
		h.mu.Unlock()
		return
	}
	session.backlog = append(session.backlog, msg)
	if len(session.backlog) > probeLogBacklogLimit {
		session.backlog = append([]ProbeLogMessage(nil), session.backlog[len(session.backlog)-probeLogBacklogLimit:]...)
	}
	subs := make([]chan ProbeLogMessage, 0, len(session.subs))
	for ch := range session.subs {
		subs = append(subs, ch)
	}
	h.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *ProbeHub) CloseLogSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session := h.sessions[sessionID]
	if session == nil || session.closed {
		return
	}
	session.closed = true
	for ch := range session.subs {
		close(ch)
	}
	session.subs = map[chan ProbeLogMessage]struct{}{}
}

func (h *ProbeHub) gcLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		h.mu.Lock()
		for sid, session := range h.sessions {
			if now.After(session.expiresAt) {
				for ch := range session.subs {
					close(ch)
				}
				delete(h.sessions, sid)
			}
		}
		h.mu.Unlock()
	}
}
