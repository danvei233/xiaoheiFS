package probe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/pkg/safego"
)

type ProbeSLA struct {
	WindowFrom    time.Time
	WindowTo      time.Time
	TotalSeconds  int64
	OnlineSeconds int64
	UptimePercent float64
	Events        []domain.ProbeStatusEvent
}

type Service struct {
	nodes    appports.ProbeNodeRepository
	tokens   appports.ProbeEnrollTokenRepository
	events   appports.ProbeStatusEventRepository
	sessions appports.ProbeLogSessionRepository
	settings appports.SettingsRepository
	hub      *Hub
	logger   *slog.Logger
}

func NewService(nodes appports.ProbeNodeRepository, tokens appports.ProbeEnrollTokenRepository, events appports.ProbeStatusEventRepository, sessions appports.ProbeLogSessionRepository, settings appports.SettingsRepository) *Service {
	return &Service{
		nodes:    nodes,
		tokens:   tokens,
		events:   events,
		sessions: sessions,
		settings: settings,
		logger:   slog.Default(),
	}
}

// SetHub attaches the WS connection hub so the offline scan can exclude probes
// that are currently connected. A probe with a configured heartbeat interval
// longer than the offline grace is still alive while its WS connection is up.
func (s *Service) SetHub(hub *Hub) {
	s.hub = hub
}

// logError emits a structured error log; failures that used to be swallowed are
// observable without exposing sensitive data.
func (s *Service) logError(msg string, err error) {
	if s.logger != nil {
		s.logger.Error(msg, "err", err)
	}
}

// logInfo emits a structured informational log with the given key/value fields.
func (s *Service) logInfo(msg string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	s.logger.Info(msg, attrs...)
}

func NewHub() *Hub {
	return newProbeHub()
}

func (s *Service) CreateProbe(ctx context.Context, name, agentID, osType, tagsJSON string) (domain.ProbeNode, string, error) {
	if strings.TrimSpace(agentID) == "" {
		return domain.ProbeNode{}, "", appshared.ErrInvalidInput
	}
	node := domain.ProbeNode{
		Name:       strings.TrimSpace(name),
		AgentID:    strings.TrimSpace(agentID),
		SecretHash: hashRawToken(randomToken(48)),
		Status:     domain.ProbeStatusOffline,
		OSType:     strings.TrimSpace(osType),
		TagsJSON:   normalizeJSON(tagsJSON, "[]"),
	}
	if node.Name == "" {
		node.Name = node.AgentID
	}
	if err := s.nodes.CreateProbeNode(ctx, &node); err != nil {
		return domain.ProbeNode{}, "", err
	}
	raw, err := s.ResetEnrollToken(ctx, node.ID)
	if err != nil {
		return domain.ProbeNode{}, "", err
	}
	return node, raw, nil
}

func (s *Service) ResetEnrollToken(ctx context.Context, probeID int64) (string, error) {
	if probeID <= 0 {
		return "", appshared.ErrInvalidInput
	}
	if err := s.tokens.DeleteProbeEnrollTokensByProbe(ctx, probeID); err != nil {
		return "", err
	}
	raw := "enroll_" + randomToken(48)
	tok := &domain.ProbeEnrollToken{
		ProbeID:   probeID,
		TokenHash: hashRawToken(raw),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.tokens.CreateProbeEnrollToken(ctx, tok); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Service) ListProbes(ctx context.Context, filter appshared.ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error) {
	return s.nodes.ListProbeNodes(ctx, filter, limit, offset)
}

func (s *Service) GetProbe(ctx context.Context, id int64) (domain.ProbeNode, error) {
	return s.nodes.GetProbeNode(ctx, id)
}

func (s *Service) UpdateProbe(ctx context.Context, node domain.ProbeNode) error {
	if node.ID <= 0 {
		return appshared.ErrInvalidInput
	}
	node.Name = strings.TrimSpace(node.Name)
	node.OSType = strings.TrimSpace(node.OSType)
	node.TagsJSON = normalizeJSON(node.TagsJSON, "[]")
	return s.nodes.UpdateProbeNode(ctx, node)
}

func (s *Service) DeleteProbe(ctx context.Context, id int64) error {
	if id <= 0 {
		return appshared.ErrInvalidInput
	}
	return s.nodes.DeleteProbeNode(ctx, id)
}

func (s *Service) Enroll(ctx context.Context, enrollToken, agentID, name, osType string) (domain.ProbeNode, string, error) {
	enrollToken = strings.TrimSpace(enrollToken)
	if enrollToken == "" {
		return domain.ProbeNode{}, "", appshared.ErrInvalidInput
	}
	token, err := s.tokens.GetValidProbeEnrollTokenByHash(ctx, hashRawToken(enrollToken), time.Now())
	if err != nil {
		return domain.ProbeNode{}, "", err
	}
	node, err := s.nodes.GetProbeNode(ctx, token.ProbeID)
	if err != nil {
		return domain.ProbeNode{}, "", err
	}
	secret := "psk_" + randomToken(48)
	node.SecretHash = hashRawToken(secret)
	if trimmed := strings.TrimSpace(agentID); trimmed != "" {
		node.AgentID = trimmed
	}
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		node.Name = trimmed
	}
	if trimmed := strings.TrimSpace(osType); trimmed != "" {
		node.OSType = trimmed
	}
	// Claim the enroll token and apply the node binding/secret update in one
	// atomic repository transaction, so a failed node update rolls the claim
	// back (the token is never burned) and two concurrent enrollments have a
	// single winner who cannot be overwritten by the loser (P2-33 follow-up).
	claimed, err := s.tokens.ConsumeProbeEnrollTokenAndUpdateNode(ctx, token.ID, node.ID, map[string]any{
		"name":        node.Name,
		"agent_id":    node.AgentID,
		"secret_hash": node.SecretHash,
		"os_type":     node.OSType,
		"tags_json":   node.TagsJSON,
	}, time.Now())
	if err != nil {
		return domain.ProbeNode{}, "", err
	}
	if !claimed {
		return domain.ProbeNode{}, "", appshared.ErrConflict
	}
	return node, secret, nil
}

func (s *Service) ValidateSecret(ctx context.Context, probeID int64, secret string) (domain.ProbeNode, error) {
	if probeID <= 0 || strings.TrimSpace(secret) == "" {
		return domain.ProbeNode{}, appshared.ErrInvalidInput
	}
	node, err := s.nodes.GetProbeNode(ctx, probeID)
	if err != nil {
		return domain.ProbeNode{}, err
	}
	if !strings.EqualFold(node.SecretHash, hashRawToken(secret)) {
		return domain.ProbeNode{}, appshared.ErrForbidden
	}
	return node, nil
}

func (s *Service) MarkOnline(ctx context.Context, probeID int64, reason string) error {
	now := time.Now()
	if err := s.nodes.UpdateProbeNodeStatus(ctx, probeID, domain.ProbeStatusOnline, reason, now); err != nil {
		return err
	}
	ev := &domain.ProbeStatusEvent{ProbeID: probeID, Status: domain.ProbeStatusOnline, At: now, Reason: reason}
	return s.events.CreateProbeStatusEvent(ctx, ev)
}

func (s *Service) MarkOffline(ctx context.Context, probeID int64, reason string) error {
	now := time.Now()
	if err := s.nodes.UpdateProbeNodeStatus(ctx, probeID, domain.ProbeStatusOffline, reason, now); err != nil {
		return err
	}
	ev := &domain.ProbeStatusEvent{ProbeID: probeID, Status: domain.ProbeStatusOffline, At: now, Reason: reason}
	return s.events.CreateProbeStatusEvent(ctx, ev)
}

// clampClientTime bounds client-controlled timestamps to a small future
// tolerance and a 24h past window. Without clamping a probe could report a far
// future heartbeat and never be considered offline.
func clampClientTime(at time.Time) time.Time {
	now := time.Now()
	if at.After(now.Add(5 * time.Minute)) {
		return now
	}
	if at.Before(now.Add(-24 * time.Hour)) {
		return now
	}
	return at
}

func (s *Service) HandleHeartbeat(ctx context.Context, probeID int64, at time.Time) error {
	if at.IsZero() {
		at = time.Now()
	}
	at = clampClientTime(at)
	if err := s.nodes.UpdateProbeNodeHeartbeat(ctx, probeID, at); err != nil {
		return err
	}
	node, err := s.nodes.GetProbeNode(ctx, probeID)
	if err != nil {
		return err
	}
	if node.Status != domain.ProbeStatusOnline {
		return s.MarkOnline(ctx, probeID, "heartbeat")
	}
	return nil
}

func (s *Service) HandleSnapshot(ctx context.Context, probeID int64, at time.Time, snapshotJSON string, osType string) error {
	if at.IsZero() {
		at = time.Now()
	}
	at = clampClientTime(at)
	snapshotJSON = normalizeJSON(snapshotJSON, "{}")
	return s.nodes.UpdateProbeNodeSnapshot(ctx, probeID, at, snapshotJSON, strings.TrimSpace(osType))
}

func (s *Service) ComputeSLA(ctx context.Context, probeID int64, windowDays int) (ProbeSLA, error) {
	if windowDays <= 0 {
		windowDays = s.getIntSetting(ctx, "probe_sla_window_days", 7)
	}
	now := time.Now()
	from := now.Add(-time.Duration(windowDays) * 24 * time.Hour)
	events, err := s.events.ListProbeStatusEvents(ctx, probeID, from, now)
	if err != nil {
		return ProbeSLA{}, err
	}
	total := int64(now.Sub(from).Seconds())
	if total <= 0 {
		total = 1
	}
	current := domain.ProbeStatusOffline
	if prev, prevErr := s.events.GetLatestProbeStatusEventBefore(ctx, probeID, from); prevErr == nil {
		current = prev.Status
	} else if !errors.Is(prevErr, appshared.ErrNotFound) {
		return ProbeSLA{}, prevErr
	}
	if current == "" {
		current = domain.ProbeStatusOffline
	}
	cursor := from
	onlineSeconds := int64(0)
	for _, ev := range events {
		if ev.At.After(now) {
			break
		}
		if current == domain.ProbeStatusOnline {
			onlineSeconds += int64(ev.At.Sub(cursor).Seconds())
		}
		cursor = ev.At
		current = ev.Status
	}
	if current == domain.ProbeStatusOnline && cursor.Before(now) {
		onlineSeconds += int64(now.Sub(cursor).Seconds())
	}
	if onlineSeconds < 0 {
		onlineSeconds = 0
	}
	if onlineSeconds > total {
		onlineSeconds = total
	}
	return ProbeSLA{
		WindowFrom:    from,
		WindowTo:      now,
		TotalSeconds:  total,
		OnlineSeconds: onlineSeconds,
		UptimePercent: float64(onlineSeconds) * 100 / float64(total),
		Events:        events,
	}, nil
}

func (s *Service) StartOfflineWatcher(ctx context.Context) {
	interval := 10 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			safego.Run("probe offline watcher tick", func() { s.scanOffline(ctx) })
		}
	}
}

func (s *Service) scanOffline(ctx context.Context) {
	grace := s.getIntSetting(ctx, "probe_offline_grace_sec", 90)
	if grace <= 0 {
		grace = 90
	}
	cutoff := time.Now().Add(-time.Duration(grace) * time.Second)
	// Page the whole table with a stable id cursor so nodes beyond the first
	// 500 are scanned too (ListProbeNodes caps at 500 and orders id DESC, which
	// would permanently miss the oldest nodes). Each batch is bounded and the
	// cursor advances monotonically, so nodes inserted/deleted mid-scan neither
	// cause an infinite loop nor skip surviving nodes.
	var afterID int64
	scanned := 0
	marked := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		nodes, err := s.nodes.ListProbeNodesByCursor(ctx, afterID, probeScanBatchSize)
		if err != nil {
			s.logError("probe offline scan failed", err)
			return
		}
		if len(nodes) == 0 {
			break
		}
		for _, node := range nodes {
			scanned++
			if node.LastHeartbeatAt == nil || node.Status == domain.ProbeStatusOffline {
				continue
			}
			// A node with a live WS connection is online even when its heartbeat
			// interval is longer than the offline grace; never mark it offline.
			if s.hub != nil && s.hub.IsOnline(node.ID) {
				continue
			}
			if node.LastHeartbeatAt.Before(cutoff) {
				if err := s.MarkOffline(ctx, node.ID, "heartbeat_timeout"); err != nil {
					// Structured, observable failure: do not silently drop the
					// offline transition.
					s.logError("mark probe offline failed", err)
					continue
				}
				marked++
			}
		}
		afterID = nodes[len(nodes)-1].ID
		if len(nodes) < probeScanBatchSize {
			break
		}
	}
	if scanned > 0 || marked > 0 {
		s.logInfo("probe offline scan completed", map[string]any{"scanned": scanned, "marked_offline": marked})
	}
	// Note: status-event retention is owned exclusively by the log cleanup
	// service. Deleting events here would destroy SLA history.
}

func (s *Service) CreateLogSession(ctx context.Context, probeID int64, operatorID int64, source string) (domain.ProbeLogSession, error) {
	session := domain.ProbeLogSession{
		ProbeID:    probeID,
		OperatorID: operatorID,
		Source:     strings.TrimSpace(source),
		Status:     "running",
		StartedAt:  time.Now(),
	}
	if err := s.sessions.CreateProbeLogSession(ctx, &session); err != nil {
		return domain.ProbeLogSession{}, err
	}
	return session, nil
}

func (s *Service) FinishLogSession(ctx context.Context, sessionID int64, status string) error {
	session, err := s.sessions.GetProbeLogSession(ctx, sessionID)
	if err != nil {
		return err
	}
	now := time.Now()
	session.Status = strings.TrimSpace(status)
	session.EndedAt = &now
	return s.sessions.UpdateProbeLogSession(ctx, session)
}

func (s *Service) GetLogSession(ctx context.Context, sessionID int64) (domain.ProbeLogSession, error) {
	return s.sessions.GetProbeLogSession(ctx, sessionID)
}

func (s *Service) getIntSetting(ctx context.Context, key string, def int) int {
	if s.settings == nil {
		return def
	}
	item, err := s.settings.GetSetting(ctx, key)
	if err != nil {
		return def
	}
	v, err := strconv.Atoi(strings.TrimSpace(item.ValueJSON))
	if err != nil {
		return def
	}
	return v
}

const (
	// probeScanBatchSize is the bounded batch size used to page the full probe
	// node table with an id cursor (see scanOffline).
	probeScanBatchSize = 200
	tokenAlphabet      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// randomToken generates a token using a uniform draw (crypto/rand.Int) so every
// alphabet byte is exactly equally likely (no modulo bias). It is used for
// secrets, so biased sampling would measurably reduce entropy.
func randomToken(n int) string {
	// crypto/rand.Int based uniform draw avoids the modulo bias of a
	// byte%len(charset) mapping (F5). A failure is unrecoverable here (the
	// caller cannot proceed with a guessable secret), so panic loudly.
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	v, err := cryptox.RandomAlphabet(alphabet, n)
	if err != nil {
		panic("probe: random token generation failed: " + err.Error())
	}
	return v
}

func hashRawToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func normalizeJSON(raw string, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	var tmp any
	if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
		return fallback
	}
	return raw
}
