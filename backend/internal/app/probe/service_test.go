package probe

import (
	"gorm.io/gorm"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type fakeNodesRepo struct {
	nodes        []domain.ProbeNode
	heartbeats   []time.Time
	status       map[int64]domain.ProbeStatus
	failAfter    int // fail ListProbeNodesByCursor after this many calls (0 = never)
	block        bool
	secretByID   map[int64]string // persisted secret hash per node
}

func (f *fakeNodesRepo) CreateProbeNode(ctx context.Context, node *domain.ProbeNode) error {
	return nil
}
func (f *fakeNodesRepo) GetProbeNode(ctx context.Context, id int64) (domain.ProbeNode, error) {
	for _, n := range f.nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return domain.ProbeNode{}, appshared.ErrNotFound
}
func (f *fakeNodesRepo) GetProbeNodeByAgentID(ctx context.Context, agentID string) (domain.ProbeNode, error) {
	return domain.ProbeNode{}, appshared.ErrNotFound
}
func (f *fakeNodesRepo) ListProbeNodes(ctx context.Context, filter appshared.ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error) {
	return append([]domain.ProbeNode(nil), f.nodes...), len(f.nodes), nil
}
func (f *fakeNodesRepo) ListProbeNodesByCursor(ctx context.Context, afterID int64, limit int) ([]domain.ProbeNode, error) {
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.failAfter > 0 {
		f.failAfter--
		if f.failAfter == 0 {
			return nil, appshared.ErrNotFound
		}
	}
	if limit <= 0 {
		limit = 200
	}
	out := make([]domain.ProbeNode, 0, limit)
	for _, n := range f.nodes {
		if n.ID > afterID {
			out = append(out, n)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
func (f *fakeNodesRepo) UpdateProbeNode(ctx context.Context, node domain.ProbeNode) error { return nil }
func (f *fakeNodesRepo) DeleteProbeNode(ctx context.Context, id int64) error { return nil }
func (f *fakeNodesRepo) UpdateProbeNodeStatus(ctx context.Context, id int64, status domain.ProbeStatus, reason string, at time.Time) error {
	if f.status == nil {
		f.status = map[int64]domain.ProbeStatus{}
	}
	f.status[id] = status
	return nil
}
func (f *fakeNodesRepo) UpdateProbeNodeHeartbeat(ctx context.Context, id int64, at time.Time) error {
	f.heartbeats = append(f.heartbeats, at)
	for i := range f.nodes {
		if f.nodes[i].ID == id {
			f.nodes[i].LastHeartbeatAt = &at
		}
	}
	return nil
}
func (f *fakeNodesRepo) UpdateProbeNodeSnapshot(ctx context.Context, id int64, at time.Time, snapshotJSON string, osType string) error {
	return nil
}

type fakeEventsRepo struct {
	deleteCalls int
	events      []domain.ProbeStatusEvent
}

func (f *fakeEventsRepo) CreateProbeStatusEvent(ctx context.Context, ev *domain.ProbeStatusEvent) error {
	f.events = append(f.events, *ev)
	return nil
}
func (f *fakeEventsRepo) ListProbeStatusEvents(ctx context.Context, probeID int64, from, to time.Time) ([]domain.ProbeStatusEvent, error) {
	return nil, nil
}
func (f *fakeEventsRepo) GetLatestProbeStatusEventBefore(ctx context.Context, probeID int64, before time.Time) (domain.ProbeStatusEvent, error) {
	return domain.ProbeStatusEvent{}, appshared.ErrNotFound
}
func (f *fakeEventsRepo) DeleteProbeStatusEventsBefore(ctx context.Context, before time.Time) error {
	f.deleteCalls++
	return nil
}

// fakeTokensRepo implements ProbeEnrollTokenRepository. The consume path
// (ConsumeProbeEnrollTokenAndUpdateNode) simulates the atomic claim + node
// update performed by the real repository: a mid-transaction failure leaves the
// token unconsumed so the enroll stays retryable, and a second claim of the
// same token reports claimed=false.
type fakeTokensRepo struct {
	byHash       map[string]domain.ProbeEnrollToken
	used         map[int64]bool
	nodes        *fakeNodesRepo // node rows updated by the simulated tx
	consumeErr   error          // ConsumeProbeEnrollTokenAndUpdateNode returns this before marking used
	consumeCalls int            // number of ConsumeProbeEnrollTokenAndUpdateNode invocations
}

func (f *fakeTokensRepo) CreateProbeEnrollToken(ctx context.Context, token *domain.ProbeEnrollToken) error {
	if f.byHash == nil {
		f.byHash = map[string]domain.ProbeEnrollToken{}
	}
	if f.used == nil {
		f.used = map[int64]bool{}
	}
	token.ID = int64(len(f.byHash) + 1)
	f.byHash[token.TokenHash] = *token
	return nil
}
func (f *fakeTokensRepo) GetValidProbeEnrollTokenByHash(ctx context.Context, tokenHash string, now time.Time) (domain.ProbeEnrollToken, error) {
	if f.byHash == nil {
		return domain.ProbeEnrollToken{}, appshared.ErrNotFound
	}
	tok, ok := f.byHash[tokenHash]
	if !ok {
		return domain.ProbeEnrollToken{}, appshared.ErrNotFound
	}
	if f.used != nil && f.used[tok.ID] {
		return domain.ProbeEnrollToken{}, appshared.ErrConflict
	}
	if tok.ExpiresAt.Before(now) {
		return domain.ProbeEnrollToken{}, appshared.ErrNotFound
	}
	return tok, nil
}
func (f *fakeTokensRepo) MarkProbeEnrollTokenUsed(ctx context.Context, id int64, usedAt time.Time) error {
	if f.used == nil {
		f.used = map[int64]bool{}
	}
	f.used[id] = true
	return nil
}
func (f *fakeTokensRepo) MarkProbeEnrollTokenUsedIfUnused(ctx context.Context, id int64, usedAt time.Time) (bool, error) {
	if f.used == nil {
		f.used = map[int64]bool{}
	}
	if f.used[id] {
		return false, nil
	}
	f.used[id] = true
	return true, nil
}
func (f *fakeTokensRepo) ConsumeProbeEnrollTokenAndUpdateNode(ctx context.Context, tokenID, nodeID int64, updates map[string]any, usedAt time.Time) (bool, error) {
	f.consumeCalls++
	if f.used == nil {
		f.used = map[int64]bool{}
	}
	if f.used[tokenID] {
		return false, nil // already consumed: losing race
	}
	if f.consumeErr != nil {
		return false, f.consumeErr // failure rolls back, token stays usable
	}
	f.used[tokenID] = true
	if f.nodes != nil {
		for i := range f.nodes.nodes {
			if f.nodes.nodes[i].ID == nodeID {
				if v, ok := updates["secret_hash"].(string); ok {
					f.nodes.nodes[i].SecretHash = v
					if f.nodes.secretByID == nil {
						f.nodes.secretByID = map[int64]string{}
					}
					f.nodes.secretByID[nodeID] = v
				}
				if v, ok := updates["name"].(string); ok {
					f.nodes.nodes[i].Name = v
				}
				if v, ok := updates["agent_id"].(string); ok {
					f.nodes.nodes[i].AgentID = v
				}
				if v, ok := updates["os_type"].(string); ok {
					f.nodes.nodes[i].OSType = v
				}
			}
		}
	}
	return true, nil
}
func (f *fakeTokensRepo) DeleteProbeEnrollTokensByProbe(ctx context.Context, probeID int64) error {
	return nil
}

type fakeSettingsRepo struct{}

func (f *fakeSettingsRepo) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current = domain.Setting{}
		} else {
			return err
		}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}
func (f *fakeSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	return domain.Setting{}, appshared.ErrNotFound
}
func (f *fakeSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	return nil
}
func (f *fakeSettingsRepo) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	return nil
}
func (f *fakeSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}
func (f *fakeSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}
func (f *fakeSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, appshared.ErrNotFound
}
func (f *fakeSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}
func (f *fakeSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error { return nil }

var _ appports.ProbeNodeRepository = (*fakeNodesRepo)(nil)
var _ appports.ProbeStatusEventRepository = (*fakeEventsRepo)(nil)
var _ appports.ProbeEnrollTokenRepository = (*fakeTokensRepo)(nil)
var _ appports.SettingsRepository = (*fakeSettingsRepo)(nil)

func newProbeService(nodes *fakeNodesRepo, events *fakeEventsRepo) *Service {
	return newProbeServiceFull(nodes, nil, events)
}

func newProbeServiceFull(nodes *fakeNodesRepo, tokens *fakeTokensRepo, events *fakeEventsRepo) *Service {
	return NewService(nodes, tokens, events, nil, &fakeSettingsRepo{})
}

func TestScanOfflineDoesNotDeleteStatusEvents(t *testing.T) {
	now := time.Now()
	hb := now.Add(-2 * time.Hour)
	nodes := &fakeNodesRepo{nodes: []domain.ProbeNode{
		{ID: 1, Status: domain.ProbeStatusOnline, LastHeartbeatAt: &hb},
	}}
	events := &fakeEventsRepo{}
	svc := newProbeService(nodes, events)

	svc.scanOffline(context.Background())

	if events.deleteCalls != 0 {
		t.Fatalf("scanOffline must not delete SLA/status events; got %d delete calls", events.deleteCalls)
	}
}

func TestScanOfflineScansAllNodesBeyondBatchSize(t *testing.T) {
	now := time.Now()
	hb := now.Add(-2 * time.Hour)
	nodes := &fakeNodesRepo{status: map[int64]domain.ProbeStatus{}}
	const total = probeScanBatchSize*3 + 7 // 607 nodes, > batch and > 500
	for i := 1; i <= total; i++ {
		nodes.nodes = append(nodes.nodes, domain.ProbeNode{
			ID: int64(i), Status: domain.ProbeStatusOnline, LastHeartbeatAt: &hb,
		})
	}
	svc := newProbeService(nodes, &fakeEventsRepo{})
	svc.scanOffline(context.Background())
	if len(nodes.status) != total {
		t.Fatalf("expected %d nodes marked offline, got %d", total, len(nodes.status))
	}
	// The highest-ID node must have been scanned too.
	if nodes.status[int64(total)] != domain.ProbeStatusOffline {
		t.Fatalf("highest node %d not marked offline", total)
	}
	// A fresh node at the end must NOT be marked.
	fresh := time.Now()
	nodes.nodes = append(nodes.nodes, domain.ProbeNode{ID: int64(total + 1), Status: domain.ProbeStatusOnline, LastHeartbeatAt: &fresh})
	svc.scanOffline(context.Background())
	if _, ok := nodes.status[int64(total+1)]; ok {
		t.Fatalf("fresh node was incorrectly marked offline")
	}
}

func TestScanOfflineHandlesRepoFailure(t *testing.T) {
	nodes := &fakeNodesRepo{nodes: []domain.ProbeNode{{ID: 1, Status: domain.ProbeStatusOffline}}}
	nodes.failAfter = 1 // fail the very first cursor page
	events := &fakeEventsRepo{}
	svc := newProbeService(nodes, events)

	svc.scanOffline(context.Background()) // must not panic
}

func TestScanOfflineRespondsToCancellation(t *testing.T) {
	nodes := &fakeNodesRepo{block: true}
	events := &fakeEventsRepo{}
	svc := newProbeService(nodes, events)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc.scanOffline(ctx) // must return promptly without hanging
}

func TestHandleHeartbeatClampsFutureTimestamp(t *testing.T) {
	now := time.Now()
	nodes := &fakeNodesRepo{nodes: []domain.ProbeNode{{ID: 1, Status: domain.ProbeStatusOnline}}}
	svc := newProbeService(nodes, &fakeEventsRepo{})

	future := now.Add(10 * time.Minute)
	if err := svc.HandleHeartbeat(context.Background(), 1, future); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if len(nodes.heartbeats) == 0 {
		t.Fatalf("heartbeat not recorded")
	}
	last := nodes.heartbeats[len(nodes.heartbeats)-1]
	if last.After(now.Add(time.Minute)) {
		t.Fatalf("future heartbeat timestamp was not clamped: %s", last)
	}
}

// TestEnrollPartialFailureDoesNotBurnToken verifies a mid-transaction failure
// while rotating the secret rolls back the token consumption, so the enroll
// stays retryable (P2-33).
func TestEnrollPartialFailureDoesNotBurnToken(t *testing.T) {
	nodes := &fakeNodesRepo{nodes: []domain.ProbeNode{{ID: 1, Name: "n1", Status: domain.ProbeStatusOffline}}}
	tokens := &fakeTokensRepo{nodes: nodes}
	svc := newProbeServiceFull(nodes, tokens, &fakeEventsRepo{})

	raw := "enroll_test_token"
	tok := &domain.ProbeEnrollToken{
		ProbeID:   1,
		TokenHash: hashRawToken(raw),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := tokens.CreateProbeEnrollToken(context.Background(), tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Simulate a mid-transaction failure while rotating the secret.
	tokens.consumeErr = appshared.ErrNotFound
	if _, _, err := svc.Enroll(context.Background(), raw, "agent-a", "n1", "linux"); err == nil {
		t.Fatalf("expected enroll to fail when the node update fails")
	}
	if tokens.consumeCalls != 1 {
		t.Fatalf("expected 1 enroll attempt, got %d", tokens.consumeCalls)
	}
	if nodes.secretByID[1] != "" {
		t.Fatalf("secret must not be persisted on a failed enroll")
	}
	// The token must still be valid (not burned): retry succeeds.
	tokens.consumeErr = nil
	node, secret, err := svc.Enroll(context.Background(), raw, "agent-a", "n1", "linux")
	if err != nil {
		t.Fatalf("retry enroll failed: %v", err)
	}
	if node.ID != 1 || secret == "" {
		t.Fatalf("unexpected enroll result: id=%d secret=%q", node.ID, secret)
	}
	if nodes.secretByID[1] == "" {
		t.Fatalf("secret must be persisted after a successful enroll")
	}
}

// TestEnrollReplayRejected verifies the CAS token consumption rejects a replay
// even when a concurrent enroll already succeeded.
func TestEnrollReplayRejected(t *testing.T) {
	nodes := &fakeNodesRepo{nodes: []domain.ProbeNode{{ID: 1, Name: "n1", Status: domain.ProbeStatusOffline}}}
	tokens := &fakeTokensRepo{nodes: nodes}
	svc := newProbeServiceFull(nodes, tokens, &fakeEventsRepo{})

	raw := "enroll_replay_token"
	_ = tokens.CreateProbeEnrollToken(context.Background(), &domain.ProbeEnrollToken{
		ProbeID:   1,
		TokenHash: hashRawToken(raw),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if _, _, err := svc.Enroll(context.Background(), raw, "agent-a", "n1", "linux"); err != nil {
		t.Fatalf("first enroll failed: %v", err)
	}
	if _, _, err := svc.Enroll(context.Background(), raw, "agent-b", "n1", "linux"); err == nil {
		t.Fatalf("replayed enroll must be rejected")
	}
}

func TestScanOfflineSkipsHubOnlineNode(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour)
	nodes := &fakeNodesRepo{status: map[int64]domain.ProbeStatus{}}
	nodes.nodes = []domain.ProbeNode{
		{ID: 1, Status: domain.ProbeStatusOnline, LastHeartbeatAt: &old}, // connected
		{ID: 2, Status: domain.ProbeStatusOnline, LastHeartbeatAt: &old}, // not connected
	}
	events := &fakeEventsRepo{}
	svc := newProbeService(nodes, events)
	hub := NewHub()
	hub.RegisterConn(1, func([]byte) error { return nil })
	svc.SetHub(hub)

	svc.scanOffline(context.Background())
	if nodes.status[1] != "" {
		t.Fatalf("hub-online node 1 must not be marked offline, got %q", nodes.status[1])
	}
	if nodes.status[2] != domain.ProbeStatusOffline {
		t.Fatalf("node 2 (no connection) should be marked offline, got %q", nodes.status[2])
	}
}

func TestRandomTokenUsesAlphabet(t *testing.T) {
	tok := randomToken(64)
	if len(tok) != 64 {
		t.Fatalf("unexpected token length %d", len(tok))
	}
	for _, r := range tok {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r) {
			t.Fatalf("token contains char outside alphabet: %q", r)
		}
	}
}

// TestProbeService_EnrollConsumesTokenOnce verifies a successful enroll marks
// the token used and reusing it fails.
func TestProbeService_EnrollConsumesTokenOnce(t *testing.T) {
	_, r := testutil.NewTestDB(t, false)
	svc := NewService(r, r, r, r, r)
	ctx := context.Background()
	node, rawToken, err := svc.CreateProbe(ctx, "p1", "agent1", "linux", "[]")
	if err != nil {
		t.Fatalf("create probe: %v", err)
	}

	if _, _, err := svc.Enroll(ctx, rawToken, "agent1-new", "p1", "linux"); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	// Second enroll with the same token must fail (the token is consumed).
	if _, _, err := svc.Enroll(ctx, rawToken, "agent1-new2", "p1", "linux"); err == nil {
		t.Fatalf("reused enroll token must fail")
	}
	// The node's secret must have been rotated by the first enroll.
	updated, err := svc.GetProbe(ctx, node.ID)
	if err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if updated.SecretHash == "" || updated.SecretHash == node.SecretHash {
		t.Fatalf("enroll must rotate the agent secret")
	}
}

// TestProbeService_EnrollConcurrentSingleWinner verifies two concurrent
// enrollments with the same token yield exactly one winner, and the loser's
// agent binding cannot overwrite the winner's.
func TestProbeService_EnrollConcurrentSingleWinner(t *testing.T) {
	_, r := testutil.NewTestDB(t, false)
	svc := NewService(r, r, r, r, r)
	ctx := context.Background()
	node, rawToken, err := svc.CreateProbe(ctx, "p-race", "agent-race", "linux", "[]")
	if err != nil {
		t.Fatalf("create probe: %v", err)
	}

	const goroutines = 8
	var wg sync.WaitGroup
	results := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _, err := svc.Enroll(ctx, rawToken, "agent-winner", "p-race", "linux")
			results[idx] = err
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, appshared.ErrConflict) && !errors.Is(err, appshared.ErrNotFound) {
			t.Fatalf("expected only ErrConflict/ErrNotFound for losers, got %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one winning enroll, got %d", winners)
	}

	updated, err := svc.GetProbe(ctx, node.ID)
	if err != nil {
		t.Fatalf("reload node: %v", err)
	}
	if updated.AgentID != "agent-winner" {
		t.Fatalf("loser must not overwrite the agent binding, got %q", updated.AgentID)
	}
}
