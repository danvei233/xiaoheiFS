package realname

import (
	"errors"
	"gorm.io/gorm"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func TestParsePendingReason_ExtendedPendingFaceFormat(t *testing.T) {
	provider, token, ok := parsePendingReason("pending_face:baidu:tok123:extra_payload")
	if !ok {
		t.Fatalf("expected parse success")
	}
	if provider != "baidu" {
		t.Fatalf("unexpected provider: %q", provider)
	}
	if token != "tok123" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestMaskIDNumber(t *testing.T) {
	cases := []struct{ in, want string }{
		{"110101199001011234", "1101****1234"},
		{"12345", "****"},
		{"123456789012345678", "1234****5678"},
	}
	for _, tc := range cases {
		if got := maskIDNumber(tc.in); got != tc.want {
			t.Fatalf("maskIDNumber(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeReasonBounded(t *testing.T) {
	long := strings.Repeat("x", realnameReasonMaxLen+100)
	got := sanitizeReason(long)
	if len(got) > realnameReasonMaxLen {
		t.Fatalf("reason not bounded: len=%d", len(got))
	}
}

func TestAllowAttemptBudget(t *testing.T) {
	repo := &fakeRealNameRepo{}
	svc := NewService(repo, nil, nil)
	// The first realnameMaxAttempts calls are allowed, the next is rejected.
	for i := 0; i < realnameMaxAttempts; i++ {
		if err := svc.allowAttempt(context.Background(), 1, "fake_provider"); err != nil {
			t.Fatalf("attempt %d should be allowed, got %v", i, err)
		}
	}
	if err := svc.allowAttempt(context.Background(), 1, "fake_provider"); err == nil {
		t.Fatalf("attempt beyond budget should be rejected")
	}
	// A different user is unaffected (the per-user global budget is the binding
	// constraint, so a fresh user can attempt again).
	if err := svc.allowAttempt(context.Background(), 2, "fake_provider"); err != nil {
		t.Fatalf("different user should have its own budget")
	}
}

// TestAllowAttemptBudgetAcrossInstances verifies the DB-backed budget rejects
// once the window cap is reached even when two Service instances (sharing the
// same repo/counter) are used, simulating multiple replicas.
func TestAllowAttemptBudgetAcrossInstances(t *testing.T) {
	repo := &fakeRealNameRepo{}
	svcA := NewService(repo, nil, nil)
	svcB := NewService(repo, nil, nil)
	for i := 0; i < realnameMaxAttempts; i++ {
		// Alternate the "instance" that consumes an attempt.
		var err error
		if i%2 == 0 {
			err = svcA.allowAttempt(context.Background(), 55, "fake_provider")
		} else {
			err = svcB.allowAttempt(context.Background(), 55, "fake_provider")
		}
		if err != nil {
			t.Fatalf("attempt %d should be allowed across instances, got %v", i, err)
		}
	}
	if err := svcB.allowAttempt(context.Background(), 55, "fake_provider"); err == nil {
		t.Fatalf("budget must hold across service instances")
	}
}

// --- fakes -----------------------------------------------------------------

type fakeRealNameRepo struct {
	mu       sync.Mutex
	records  []domain.RealNameVerification
	attempts map[string]int
}

func (f *fakeRealNameRepo) CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	record.ID = int64(len(f.records) + 1)
	f.records = append(f.records, *record)
	return nil
}

func (f *fakeRealNameRepo) GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.records) - 1; i >= 0; i-- {
		if f.records[i].UserID == userID {
			return f.records[i], nil
		}
	}
	return domain.RealNameVerification{}, appshared.ErrNotFound
}

func (f *fakeRealNameRepo) ListRealNameVerifications(ctx context.Context, userID *int64, status string, limit, offset int) ([]domain.RealNameVerification, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.RealNameVerification, 0)
	for _, r := range f.records {
		if userID != nil && r.UserID != *userID {
			continue
		}
		out = append(out, r)
	}
	return out, len(out), nil
}

func (f *fakeRealNameRepo) ListPendingRealNameVerifications(ctx context.Context, afterID int64, limit int) ([]domain.RealNameVerification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.RealNameVerification, 0)
	for _, r := range f.records {
		if r.ID <= afterID || strings.ToLower(strings.TrimSpace(r.Status)) != "pending" {
			continue
		}
		out = append(out, r)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeRealNameRepo) UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.records {
		if f.records[i].ID == id {
			f.records[i].Status = status
			f.records[i].Reason = reason
			f.records[i].VerifiedAt = verifiedAt
		}
	}
	return nil
}

func (f *fakeRealNameRepo) UpdateRealNameProviderState(ctx context.Context, id int64, providerState string, stableReason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.records {
		if f.records[i].ID == id {
			f.records[i].ProviderState = providerState
			f.records[i].Reason = stableReason
		}
	}
	return nil
}

type fakeAttemptCounter struct {
	mu      sync.Mutex
	counts  map[string]int // key: userID|dimension|windowStart
	cleaned int
}

func (f *fakeRealNameRepo) AllowRealNameAttempt(ctx context.Context, userID int64, dimension string, max int, window time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%d|%s|%d", userID, dimension, time.Now().Truncate(window).Unix())
	if f.attempts == nil {
		f.attempts = map[string]int{}
	}
	f.attempts[key]++
	return f.attempts[key] <= max, nil
}

func (f *fakeRealNameRepo) CleanupRealNameAttempts(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (f *fakeRealNameRepo) MarkRealNamePolled(ctx context.Context, id int64, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.records {
		if f.records[i].ID == id {
			f.records[i].LastPolledAt = &at
			f.records[i].PollCount++
		}
	}
	return nil
}

var _ appports.RealNameRepository = (*fakeRealNameRepo)(nil)

type fakeSettingsRepo struct {
	values map[string]string
}

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
	if v, ok := f.values[key]; ok {
		return domain.Setting{Key: key, ValueJSON: v}, nil
	}
	return domain.Setting{}, appshared.ErrNotFound
}
func (f *fakeSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

var _ settingsRepo = (*fakeSettingsRepo)(nil)

type fakePendingProvider struct {
	reason     string
	queryState map[string]string // token -> status
	queryCalls int
}

func (p *fakePendingProvider) Key() string  { return "fake_pending" }
func (p *fakePendingProvider) Name() string { return "fake pending" }
func (p *fakePendingProvider) Verify(ctx context.Context, realName, idNumber string) (bool, string, error) {
	return p.VerifyWithInput(ctx, appshared.RealNameVerifyInput{RealName: realName, IDNumber: idNumber})
}
func (p *fakePendingProvider) VerifyWithInput(ctx context.Context, in appshared.RealNameVerifyInput) (bool, string, error) {
	return false, p.reason, nil
}
func (p *fakePendingProvider) QueryPending(ctx context.Context, token, provider string) (string, string, error) {
	p.queryCalls++
	if st, ok := p.queryState[token]; ok {
		return st, "", nil
	}
	return "failed", "unknown token", nil
}

type fakeRegistry struct {
	providers map[string]appshared.RealNameProvider
}

func (r *fakeRegistry) GetProvider(key string) (appshared.RealNameProvider, error) {
	if p, ok := r.providers[key]; ok {
		return p, nil
	}
	return nil, appshared.ErrNotFound
}
func (r *fakeRegistry) ListProviders() []appshared.RealNameProvider {
	out := make([]appshared.RealNameProvider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

func newTestService(reason string, queryState map[string]string) (*Service, *fakeRealNameRepo) {
	repo := &fakeRealNameRepo{}
	reg := &fakeRegistry{providers: map[string]appshared.RealNameProvider{
		"fake_pending": &fakePendingProvider{reason: reason, queryState: queryState},
	}}
	settings := &fakeSettingsRepo{values: map[string]string{
		"realname_enabled":  "true",
		"realname_provider": "fake_pending",
	}}
	return NewService(repo, reg, settings), repo
}

// --- provider state tests ---------------------------------------------------

func TestVerifyPendingStoresEncryptedStateAndSanitizedReason(t *testing.T) {
	svc, repo := newTestService("pending:tok-abc-123", nil)
	record, err := svc.VerifyWithInput(context.Background(), 1, appshared.RealNameVerifyInput{RealName: "Alice", IDNumber: "11010519491231002X"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if record.Status != "pending" {
		t.Fatalf("status = %s, want pending", record.Status)
	}
	if strings.Contains(record.Reason, "tok-abc-123") {
		t.Fatalf("token leaked into Reason: %q", record.Reason)
	}
	if record.Reason != "pending" {
		t.Fatalf("Reason = %q, want stable 'pending'", record.Reason)
	}
	if record.ProviderState == "" {
		t.Fatalf("expected encrypted provider state")
	}
	if strings.Contains(record.ProviderState, "tok-abc-123") {
		t.Fatalf("token appears in provider_state plaintext: %q", record.ProviderState)
	}
	state, ok := svc.openState(record.ProviderState)
	if !ok || state.Token != "tok-abc-123" {
		t.Fatalf("decrypted state = %+v ok=%v, want token tok-abc-123", state, ok)
	}
	// The stored repo record also keeps the token out of Reason.
	got, err := repo.GetLatestRealNameVerification(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.Contains(got.Reason, "tok-abc-123") {
		t.Fatalf("token leaked into persisted Reason: %q", got.Reason)
	}
}

func TestVerifyPendingFaceStoresRedirectInStateNotReason(t *testing.T) {
	redirect := "https://e.example.com/face?token=abc"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(redirect))
	svc, _ := newTestService("pending_face:baidu:face-tok-9:"+encoded, nil)
	record, err := svc.VerifyWithInput(context.Background(), 2, appshared.RealNameVerifyInput{RealName: "Bob", IDNumber: "11010519491231002X"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if record.Reason != "pending_face:baidu" {
		t.Fatalf("Reason = %q, want stable 'pending_face:baidu'", record.Reason)
	}
	if strings.Contains(record.Reason, "face-tok-9") || strings.Contains(record.Reason, "example.com") {
		t.Fatalf("token/URL leaked into Reason: %q", record.Reason)
	}
	state, ok := svc.openState(record.ProviderState)
	if !ok || state.Token != "face-tok-9" || state.RedirectURL != redirect || state.FaceProvider != "baidu" {
		t.Fatalf("decrypted state = %+v ok=%v", state, ok)
	}
	if got := svc.PendingRedirectURL(context.Background(), record); got != redirect {
		t.Fatalf("PendingRedirectURL = %q, want %q", got, redirect)
	}
}

func TestOpenStateFailsClosedOnTamperedCiphertext(t *testing.T) {
	svc, _ := newTestService("pending:tamper-me", nil)
	record, err := svc.VerifyWithInput(context.Background(), 3, appshared.RealNameVerifyInput{RealName: "Carol", IDNumber: "11010519491231002X"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Flip bytes in the middle of the ciphertext; decryption must fail closed.
	tampered := record.ProviderState
	mid := len(tampered) / 2
	replacement := byte('A')
	if tampered[mid] == replacement {
		replacement = 'B'
	}
	tampered = tampered[:mid] + string(replacement) + tampered[mid+1:]
	if state, ok := svc.openState(tampered); ok {
		t.Fatalf("tampered state decrypted: %+v", state)
	}
}

func TestPollPendingMigratesLegacyReasonToken(t *testing.T) {
	svc, repo := newTestService("unused", map[string]string{"legacy-tok": "pending"})
	// Seed a legacy pending record with the token embedded in Reason.
	if err := repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
		UserID: 7, RealName: "Dave", IDNumber: "11010519491231002X",
		Status: "pending", Provider: "fake_pending", Reason: "pending:legacy-tok",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("poll: %v", err)
	}
	record, _ := repo.GetLatestRealNameVerification(context.Background(), 7)
	if record.ProviderState == "" {
		t.Fatalf("legacy record was not migrated to provider_state")
	}
	if strings.Contains(record.Reason, "legacy-tok") {
		t.Fatalf("legacy token still in Reason after migration: %q", record.Reason)
	}
	if record.Reason != "pending" {
		t.Fatalf("Reason = %q, want 'pending'", record.Reason)
	}
	state, ok := svc.openState(record.ProviderState)
	if !ok || state.Token != "legacy-tok" {
		t.Fatalf("migrated state = %+v ok=%v", state, ok)
	}
}

func TestPollPendingFailsClosedOnCorruptProviderState(t *testing.T) {
	svc, repo := newTestService("unused", nil)
	// ProviderState is garbage and Reason carries no legacy token -> fail closed.
	if err := repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
		UserID: 8, RealName: "Eve", IDNumber: "11010519491231002X",
		Status: "pending", Provider: "fake_pending", Reason: "pending", ProviderState: "not-valid-ciphertext",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("poll: %v", err)
	}
	record, _ := repo.GetLatestRealNameVerification(context.Background(), 8)
	if record.Status != "failed" {
		t.Fatalf("corrupt state status = %s, want failed", record.Status)
	}
}

func TestPollPendingUsesEncryptedStateToken(t *testing.T) {
	svc, repo := newTestService("pending:sealed-tok-1", map[string]string{"sealed-tok-1": "verified"})
	record, err := svc.VerifyWithInput(context.Background(), 9, appshared.RealNameVerifyInput{RealName: "Frank", IDNumber: "11010519491231002X"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if record.ProviderState == "" {
		t.Fatalf("expected encrypted state")
	}
	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("poll: %v", err)
	}
	got, _ := repo.GetLatestRealNameVerification(context.Background(), 9)
	if got.Status != "verified" {
		t.Fatalf("status = %s, want verified (token read from encrypted state)", got.Status)
	}
}

// TestPollPendingCooldownSkipsRecentlyPolled verifies a pending record is not
// re-queried against the provider within the poll cooldown, so the 20s scheduler
// cannot bill the paid provider per record per tick.
func TestPollPendingCooldownSkipsRecentlyPolled(t *testing.T) {
	provider := &fakePendingProvider{reason: "pending:tok-cd", queryState: map[string]string{"tok-cd": "processing"}}
	repo := &fakeRealNameRepo{}
	reg := &fakeRegistry{providers: map[string]appshared.RealNameProvider{"fake_pending": provider}}
	settings := &fakeSettingsRepo{values: map[string]string{
		"realname_enabled":  "true",
		"realname_provider": "fake_pending",
	}}
	svc := NewService(repo, reg, settings)

	if _, err := svc.VerifyWithInput(context.Background(), 1, appshared.RealNameVerifyInput{RealName: "Alice", IDNumber: "11010519491231002X"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("first poll: %v", err)
	}
	if provider.queryCalls != 1 {
		t.Fatalf("expected 1 provider query on first poll, got %d", provider.queryCalls)
	}
	// A second poll within the cooldown must NOT query the provider again.
	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if provider.queryCalls != 1 {
		t.Fatalf("cooldown failed: provider queried %d times, want 1", provider.queryCalls)
	}
}

// TestPollPendingGivesUpAfterCap verifies a pending record that never resolves
// is marked failed once it exceeds the poll cap, instead of being queried
// forever (and billed forever).
func TestPollPendingGivesUpAfterCap(t *testing.T) {
	provider := &fakePendingProvider{reason: "pending:tok-cap", queryState: map[string]string{"tok-cap": "processing"}}
	repo := &fakeRealNameRepo{}
	reg := &fakeRegistry{providers: map[string]appshared.RealNameProvider{"fake_pending": provider}}
	settings := &fakeSettingsRepo{values: map[string]string{
		"realname_enabled":  "true",
		"realname_provider": "fake_pending",
	}}
	svc := NewService(repo, reg, settings)

	rec, err := svc.VerifyWithInput(context.Background(), 2, appshared.RealNameVerifyInput{RealName: "Bob", IDNumber: "11010519491231002X"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Seed the poll count at the cap so the next cycle must give up.
	repo.mu.Lock()
	for i := range repo.records {
		if repo.records[i].ID == rec.ID {
			repo.records[i].PollCount = realnameMaxPolls
		}
	}
	repo.mu.Unlock()

	if _, err := svc.PollPending(context.Background(), 10); err != nil {
		t.Fatalf("poll: %v", err)
	}
	got, err := repo.GetLatestRealNameVerification(context.Background(), 2)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("status = %s, want failed after poll cap", got.Status)
	}
	if provider.queryCalls != 0 {
		t.Fatalf("cap must prevent further provider queries, got %d", provider.queryCalls)
	}
}

// TestPollPendingHonorsPerRoundProviderBudget seeds a large pending backlog and
// verifies a single PollPending round issues at most the per-round provider
// budget, even though the pending table is much larger. The pending-only cursor
// fetch means non-pending rows are never scanned.
func TestPollPendingHonorsPerRoundProviderBudget(t *testing.T) {
	provider := &fakePendingProvider{reason: "pending", queryState: map[string]string{}}
	repo := &fakeRealNameRepo{}
	reg := &fakeRegistry{providers: map[string]appshared.RealNameProvider{"fake_pending": provider}}
	settings := &fakeSettingsRepo{values: map[string]string{
		"realname_enabled":  "true",
		"realname_provider": "fake_pending",
	}}
	svc := NewService(repo, reg, settings)

	// Seed 3000 non-pending rows (older ids) plus 2000 pending rows. The old
	// implementation paged the whole table via offset and filtered pending in
	// application code; the fixed one pages only pending rows by id cursor.
	const nonPending = 3000
	for i := 0; i < nonPending; i++ {
		if err := repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
			UserID: int64(10_000 + i), RealName: "N", IDNumber: "11010519491231002X",
			Status: "verified", Provider: "fake_pending",
		}); err != nil {
			t.Fatalf("seed non-pending %d: %v", i, err)
		}
	}
	const pendingRecords = 2000
	for i := 0; i < pendingRecords; i++ {
		if err := repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
			UserID: int64(20_000 + i), RealName: "P", IDNumber: "11010519491231002X",
			Status: "pending", Provider: "fake_pending", Reason: fmt.Sprintf("pending:tok-%d", i),
		}); err != nil {
			t.Fatalf("seed pending %d: %v", i, err)
		}
	}

	const budget = 200
	if _, err := svc.PollPending(context.Background(), budget); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if provider.queryCalls != budget {
		t.Fatalf("provider queried %d times in one round, want hard budget %d", provider.queryCalls, budget)
	}
	// Every queried record must have been a pending one (the pending-only fetch
	// never hands verified rows to the provider).
	for _, r := range repo.records {
		if r.UserID >= 10_000 && r.UserID < 20_000 {
			if r.Status == "verified" && r.PollCount > 0 {
				t.Fatalf("a non-pending row was polled: id=%d", r.ID)
			}
		}
	}
}

// localOnlyProvider mimics the local idcard_cn provider: it validates input
// format but can never authoritatively verify, so it returns pending.
type localOnlyProvider struct{}

func (localOnlyProvider) Key() string  { return "idcard_cn" }
func (localOnlyProvider) Name() string { return "local" }
func (localOnlyProvider) Verify(ctx context.Context, realName, idNumber string) (bool, string, error) {
	if strings.TrimSpace(realName) == "" || len(strings.TrimSpace(idNumber)) != 18 {
		return false, "invalid input", nil
	}
	return false, "pending", nil
}

// TestLocalOnlyProviderCannotGrantVerifiedBypass verifies that a pending record
// (the only outcome a local-only provider can produce) does not satisfy a
// realname-gated action: the business permission check must not be bypassed by
// mere input format validity.
func TestLocalOnlyProviderCannotGrantVerifiedBypass(t *testing.T) {
	repo := &fakeRealNameRepo{}
	reg := &fakeRegistry{providers: map[string]appshared.RealNameProvider{
		"idcard_cn": localOnlyProvider{},
	}}
	settings := &fakeSettingsRepo{values: map[string]string{
		"realname_enabled":  "true",
		"realname_provider": "idcard_cn",
	}}
	svc := NewService(repo, reg, settings)

	record, err := svc.VerifyWithInput(context.Background(), 20, appshared.RealNameVerifyInput{
		RealName: "Zhang San",
		IDNumber: "11010519491231002X",
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if record.Status != "pending" {
		t.Fatalf("local-only provider must yield pending, got %q", record.Status)
	}
	// Business permission check: pending is NOT verified, so the gated action
	// must be blocked.
	if err := svc.RequireAction(context.Background(), 20, "purchase_vps"); err == nil {
		t.Fatalf("RequireAction must reject a pending-only user (permission bypass)")
	}
	// Even after the operator marks the record verified, the user passes.
	if err := svc.UpdateStatus(context.Background(), record.ID, "verified", "manual review"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if err := svc.RequireAction(context.Background(), 20, "purchase_vps"); err != nil {
		t.Fatalf("RequireAction should pass after manual verified, got %v", err)
	}
}
