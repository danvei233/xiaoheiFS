package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type ProbeSLA struct {
	WindowFrom    time.Time
	WindowTo      time.Time
	TotalSeconds  int64
	OnlineSeconds int64
	UptimePercent float64
	Events        []domain.ProbeStatusEvent
}

type ProbeService struct {
	nodes    ProbeNodeRepository
	tokens   ProbeEnrollTokenRepository
	events   ProbeStatusEventRepository
	sessions ProbeLogSessionRepository
	settings SettingsRepository
}

func NewProbeService(
	nodes ProbeNodeRepository,
	tokens ProbeEnrollTokenRepository,
	events ProbeStatusEventRepository,
	sessions ProbeLogSessionRepository,
	settings SettingsRepository,
) *ProbeService {
	return &ProbeService{
		nodes:    nodes,
		tokens:   tokens,
		events:   events,
		sessions: sessions,
		settings: settings,
	}
}

func (s *ProbeService) CreateProbe(ctx context.Context, name, agentID, osType, tagsJSON string) (domain.ProbeNode, string, error) {
	if strings.TrimSpace(agentID) == "" {
		return domain.ProbeNode{}, "", ErrInvalidInput
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

func (s *ProbeService) ResetEnrollToken(ctx context.Context, probeID int64) (string, error) {
	if probeID <= 0 {
		return "", ErrInvalidInput
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

func (s *ProbeService) ListProbes(ctx context.Context, filter ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error) {
	return s.nodes.ListProbeNodes(ctx, filter, limit, offset)
}

func (s *ProbeService) GetProbe(ctx context.Context, id int64) (domain.ProbeNode, error) {
	return s.nodes.GetProbeNode(ctx, id)
}

func (s *ProbeService) UpdateProbe(ctx context.Context, node domain.ProbeNode) error {
	if node.ID <= 0 {
		return ErrInvalidInput
	}
	node.Name = strings.TrimSpace(node.Name)
	node.OSType = strings.TrimSpace(node.OSType)
	node.TagsJSON = normalizeJSON(node.TagsJSON, "[]")
	return s.nodes.UpdateProbeNode(ctx, node)
}

func (s *ProbeService) DeleteProbe(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}
	return s.nodes.DeleteProbeNode(ctx, id)
}

func (s *ProbeService) Enroll(ctx context.Context, enrollToken, agentID, name, osType string) (domain.ProbeNode, string, error) {
	enrollToken = strings.TrimSpace(enrollToken)
	if enrollToken == "" {
		return domain.ProbeNode{}, "", ErrInvalidInput
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
	if err := s.nodes.UpdateProbeNode(ctx, node); err != nil {
		return domain.ProbeNode{}, "", err
	}
	if err := s.tokens.MarkProbeEnrollTokenUsed(ctx, token.ID, time.Now()); err != nil {
		return domain.ProbeNode{}, "", err
	}
	return node, secret, nil
}

func (s *ProbeService) ValidateSecret(ctx context.Context, probeID int64, secret string) (domain.ProbeNode, error) {
	if probeID <= 0 || strings.TrimSpace(secret) == "" {
		return domain.ProbeNode{}, ErrInvalidInput
	}
	node, err := s.nodes.GetProbeNode(ctx, probeID)
	if err != nil {
		return domain.ProbeNode{}, err
	}
	if !strings.EqualFold(node.SecretHash, hashRawToken(secret)) {
		return domain.ProbeNode{}, ErrForbidden
	}
	return node, nil
}

func (s *ProbeService) MarkOnline(ctx context.Context, probeID int64, reason string) error {
	now := time.Now()
	if err := s.nodes.UpdateProbeNodeStatus(ctx, probeID, domain.ProbeStatusOnline, reason, now); err != nil {
		return err
	}
	ev := &domain.ProbeStatusEvent{
		ProbeID: probeID,
		Status:  domain.ProbeStatusOnline,
		At:      now,
		Reason:  reason,
	}
	return s.events.CreateProbeStatusEvent(ctx, ev)
}

func (s *ProbeService) MarkOffline(ctx context.Context, probeID int64, reason string) error {
	now := time.Now()
	if err := s.nodes.UpdateProbeNodeStatus(ctx, probeID, domain.ProbeStatusOffline, reason, now); err != nil {
		return err
	}
	ev := &domain.ProbeStatusEvent{
		ProbeID: probeID,
		Status:  domain.ProbeStatusOffline,
		At:      now,
		Reason:  reason,
	}
	return s.events.CreateProbeStatusEvent(ctx, ev)
}

func (s *ProbeService) HandleHeartbeat(ctx context.Context, probeID int64, at time.Time) error {
	if at.IsZero() {
		at = time.Now()
	}
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

func (s *ProbeService) HandleSnapshot(ctx context.Context, probeID int64, at time.Time, snapshotJSON string, osType string) error {
	if at.IsZero() {
		at = time.Now()
	}
	snapshotJSON = normalizeJSON(snapshotJSON, "{}")
	return s.nodes.UpdateProbeNodeSnapshot(ctx, probeID, at, snapshotJSON, strings.TrimSpace(osType))
}

func (s *ProbeService) ComputeSLA(ctx context.Context, probeID int64, windowDays int) (ProbeSLA, error) {
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

	// Baseline status at window start:
	// 1) latest event before window start, or
	// 2) fallback to offline when there is no historical event.
	current := domain.ProbeStatusOffline
	if prev, prevErr := s.events.GetLatestProbeStatusEventBefore(ctx, probeID, from); prevErr == nil {
		current = prev.Status
	} else if !errors.Is(prevErr, ErrNotFound) {
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

func (s *ProbeService) StartOfflineWatcher(ctx context.Context) {
	interval := 10 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scanOffline(ctx)
		}
	}
}

func (s *ProbeService) scanOffline(ctx context.Context) {
	grace := s.getIntSetting(ctx, "probe_offline_grace_sec", 90)
	if grace <= 0 {
		grace = 90
	}
	retentionDays := s.getIntSetting(ctx, "probe_sla_window_days", 7)
	if retentionDays <= 0 {
		retentionDays = 7
	}
	now := time.Now()
	nodes, _, err := s.nodes.ListProbeNodes(ctx, ProbeNodeFilter{}, 1000, 0)
	if err != nil {
		return
	}
	cutoff := now.Add(-time.Duration(grace) * time.Second)
	for _, node := range nodes {
		if node.LastHeartbeatAt == nil || node.Status == domain.ProbeStatusOffline {
			continue
		}
		if node.LastHeartbeatAt.Before(cutoff) {
			_ = s.MarkOffline(ctx, node.ID, "heartbeat_timeout")
		}
	}
	_ = s.events.DeleteProbeStatusEventsBefore(ctx, now.Add(-time.Duration(retentionDays+1)*24*time.Hour))
}

func (s *ProbeService) CreateLogSession(ctx context.Context, probeID int64, operatorID int64, source string) (domain.ProbeLogSession, error) {
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

func (s *ProbeService) FinishLogSession(ctx context.Context, sessionID int64, status string) error {
	session, err := s.sessions.GetProbeLogSession(ctx, sessionID)
	if err != nil {
		return err
	}
	now := time.Now()
	session.Status = strings.TrimSpace(status)
	session.EndedAt = &now
	return s.sessions.UpdateProbeLogSession(ctx, session)
}

func (s *ProbeService) GetLogSession(ctx context.Context, sessionID int64) (domain.ProbeLogSession, error) {
	return s.sessions.GetProbeLogSession(ctx, sessionID)
}

func (s *ProbeService) getIntSetting(ctx context.Context, key string, def int) int {
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
