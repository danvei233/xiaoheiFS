package probe

import (
	"encoding/json"
	"sync"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
)

var (
	ErrProbeOffline          = domain.ErrProbeOffline
	ErrProbeLogSessionClosed = domain.ErrProbeLogSessionClosed
)

type ProbeLogMessage struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id,omitempty"`
	Data      string    `json:"data,omitempty"`
	At        time.Time `json:"at"`
}

type probeConnState struct {
	id        uint64
	send      func([]byte) error
	updatedAt time.Time
}

type probeLogStream struct {
	probeID   int64
	ttl       time.Duration
	expiresAt time.Time
	closed    bool
	subs      map[chan ProbeLogMessage]struct{}
	backlog   []ProbeLogMessage
}

// defaultProbeLogSessionTTL is the fallback session lifetime when no explicit
// TTL is supplied.
const defaultProbeLogSessionTTL = 10 * time.Minute

type Hub struct {
	mu       sync.RWMutex
	conns    map[int64][]*probeConnState
	nextID   uint64
	sessions map[string]*probeLogStream
}

const (
	probeLogChannelBuffer = 256
	probeLogBacklogLimit  = 256
	// maxProbeConnsPerProbe caps the number of concurrently registered WS
	// connections per probe. A reconnect race must not accumulate stale
	// connections, and the newest connection is the one used for commands.
	maxProbeConnsPerProbe = 4
)

func newProbeHub() *Hub {
	h := &Hub{
		conns:    make(map[int64][]*probeConnState),
		sessions: make(map[string]*probeLogStream),
	}
	safego.Go("probe log session gc", h.gcLoop)
	return h
}

// RegisterConn registers a WS connection for the probe and returns an opaque
// connection id that must be passed back to UnregisterConn when the connection
// is torn down.
func (h *Hub) RegisterConn(probeID int64, send func([]byte) error) uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	conn := &probeConnState{id: h.nextID, send: send, updatedAt: time.Now()}
	list := h.conns[probeID]
	list = append(list, conn)
	if len(list) > maxProbeConnsPerProbe {
		list = append([]*probeConnState(nil), list[len(list)-maxProbeConnsPerProbe:]...)
	}
	h.conns[probeID] = list
	return conn.id
}

// UnregisterConn removes exactly the connection registered under connID and
// reports whether the probe still has any live connections. The emptiness check
// is atomic with the removal, so a teardown cannot race a concurrent reconnect:
// the caller must only mark the probe offline when false is returned.
func (h *Hub) UnregisterConn(probeID int64, connID uint64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.conns[probeID]
	kept := list[:0]
	for _, c := range list {
		if c.id == connID {
			continue
		}
		kept = append(kept, c)
	}
	if len(kept) == 0 {
		delete(h.conns, probeID)
		return false
	}
	h.conns[probeID] = kept
	return true
}

// RemoveProbe drops every connection for a probe (used when the probe is
// deleted).
func (h *Hub) RemoveProbe(probeID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, probeID)
}

func (h *Hub) IsOnline(probeID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns[probeID]) > 0
}

func (h *Hub) SendJSON(probeID int64, payload any) error {
	h.mu.RLock()
	list := h.conns[probeID]
	var conn *probeConnState
	if len(list) > 0 {
		conn = list[len(list)-1]
	}
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

func (h *Hub) OpenLogSession(sessionID string, probeID int64, ttl time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ttl <= 0 {
		ttl = defaultProbeLogSessionTTL
	}
	h.sessions[sessionID] = &probeLogStream{
		probeID:   probeID,
		ttl:       ttl,
		expiresAt: time.Now().Add(ttl),
		subs:      make(map[chan ProbeLogMessage]struct{}),
		backlog:   make([]ProbeLogMessage, 0, probeLogBacklogLimit),
	}
}

func (h *Hub) HasLogSession(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.sessions[sessionID]
	return ok
}

// LogSessionOwnedBy reports whether the given log session exists and belongs to
// the given probe. It is an in-memory check (no DB round-trip) and is used by
// the WS handler on every log_chunk/log_end frame, where the hub's session map
// is already authoritative for ownership.
func (h *Hub) LogSessionOwnedBy(sessionID string, probeID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	session := h.sessions[sessionID]
	if session == nil || session.closed {
		return false
	}
	return session.probeID == probeID
}

func (h *Hub) SubscribeLogSession(sessionID string) (chan ProbeLogMessage, func(), error) {
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
		session.expiresAt = time.Now().Add(session.ttl)
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

// PublishLogChunk appends a chunk to the given log session, but only when the
// publishing probe actually owns the session. Publishing for a foreign session
// is a no-op, which prevents cross-probe log session injection.
func (h *Hub) PublishLogChunk(probeID int64, sessionID, requestID, chunk string) {
	h.publishLog(probeID, sessionID, ProbeLogMessage{
		Type:      "log_chunk",
		RequestID: requestID,
		Data:      chunk,
		At:        time.Now(),
	})
}

func (h *Hub) PublishLogEnd(probeID int64, sessionID, requestID string) {
	h.publishLog(probeID, sessionID, ProbeLogMessage{
		Type:      "log_end",
		RequestID: requestID,
		At:        time.Now(),
	})
	h.CloseLogSession(sessionID)
}

func (h *Hub) publishLog(probeID int64, sessionID string, msg ProbeLogMessage) {
	// The sends stay inside the lock. They are non-blocking (select-default), so
	// the lock is never held while waiting on a subscriber; but because a
	// subscriber channel can only be closed while holding the lock, holding the
	// lock during the send guarantees no send is ever issued to a channel that a
	// concurrent cancel/CloseLogSession/gcLoop has just closed (sending to a
	// closed channel panics even under select-default).
	h.mu.Lock()
	defer h.mu.Unlock()
	session := h.sessions[sessionID]
	if session == nil || session.closed || session.probeID != probeID {
		return
	}
	session.backlog = append(session.backlog, msg)
	if len(session.backlog) > probeLogBacklogLimit {
		session.backlog = append([]ProbeLogMessage(nil), session.backlog[len(session.backlog)-probeLogBacklogLimit:]...)
	}
	// Active publishing keeps the session alive.
	session.expiresAt = time.Now().Add(session.ttl)
	for ch := range session.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Hub) CloseLogSession(sessionID string) {
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

func (h *Hub) gcLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		safego.Run("probe log session gc tick", func() {
			now := time.Now()
			h.mu.Lock()
			defer h.mu.Unlock()
			for sid, session := range h.sessions {
				if now.After(session.expiresAt) {
					for ch := range session.subs {
						close(ch)
					}
					delete(h.sessions, sid)
				}
			}
		})
	}
}
