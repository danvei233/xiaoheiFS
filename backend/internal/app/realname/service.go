package realname

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
)

const (
	// realnameMaxAttempts bounds verification attempts per user within the
	// window to prevent credential brute-forcing against the KYC provider.
	realnameMaxAttempts = 5
	// realnameMaxAttemptsPerProvider bounds attempts per user against a single
	// provider (risk dimension) within the same window.
	realnameMaxAttemptsPerProvider = 5
	realnameAttemptWindow          = 10 * time.Minute
	// realnameReasonMaxLen bounds the stored provider reason to avoid persisting
	// large/opaque blobs (tokens, URLs) into the audit trail.
	realnameReasonMaxLen = 500
	// realnameStateVersion is the provider-state payload version.
	realnameStateVersion = 1
	// realnamePollCooldown is the minimum interval between provider queries for
	// the same pending record. Without it, the 20s scheduler would bill the paid
	// KYC provider once per pending record per tick indefinitely.
	realnamePollCooldown = 5 * time.Minute
	// realnameMaxPolls caps how many times a single pending record is queried
	// against the provider before it is marked failed (a record that has not
	// resolved within the cap is unlikely to resolve, and the cap bounds the
	// provider bill).
	realnameMaxPolls = 30
	// realnamePollPageSize is the bounded cursor page size PollPending uses. The
	// SQL layer filters status='pending' and pages by id, so a large table with
	// few pending rows is not scanned in full on every scheduler tick.
	realnamePollPageSize = 50
	// realnamePollBudget caps how many pending records are handed to the KYC
	// provider per PollPending round, bounding the external-call burst on large
	// tables regardless of page size. Records beyond the budget are polled on a
	// later tick.
	realnamePollBudget = 200
	// devStateSecret is the fallback encryption secret used only when a caller
	// did not configure one (tests); production wiring always overrides it with
	// the server secret via WithEncryptionSecret.
	devStateSecret = "realname-dev-state-secret-do-not-use-in-prod"
)

type (
	RealNameProvider              = appshared.RealNameProvider
	RealNameProviderWithInput     = appshared.RealNameProviderWithInput
	RealNameProviderPendingPoller = appshared.RealNameProviderPendingPoller
	RealNameProviderRegistry      = appshared.RealNameProviderRegistry
	RealNameVerifyInput           = appshared.RealNameVerifyInput
)

// settingsRepo is the subset of SettingsRepository the realname service uses.
type settingsRepo interface {
	GetSetting(ctx context.Context, key string) (domain.Setting, error)
	UpsertSetting(ctx context.Context, setting domain.Setting) error
}

type Service struct {
	repo     appports.RealNameRepository
	registry RealNameProviderRegistry
	settings settingsRepo
	cipher   *cryptox.AESGCM
}

// Option configures a Service.
type Option func(*Service)

// WithEncryptionSecret derives the provider-state encryption key from the
// server secret. It must be stable across restarts so pending records remain
// decryptable.
func WithEncryptionSecret(secret string) Option {
	return func(s *Service) {
		if enc, err := newStateCipher(secret); err == nil {
			s.cipher = enc
		}
	}
}

func NewService(repo appports.RealNameRepository, registry appshared.RealNameProviderRegistry, settings settingsRepo, opts ...Option) *Service {
	s := &Service{repo: repo, registry: registry, settings: settings}
	for _, opt := range opts {
		opt(s)
	}
	if s.cipher == nil {
		// Test fallback only; production wiring passes WithEncryptionSecret.
		if enc, err := newStateCipher(devStateSecret); err == nil {
			s.cipher = enc
		}
	}
	return s
}

// allowAttempt enforces the realname attempt budget with DB atomic window
// counters, so it is consistent across service instances and restarts. A
// per-user budget is checked together with a per-user-provider (risk) budget.
func (s *Service) allowAttempt(ctx context.Context, userID int64, providerKey string) error {
	if s.repo == nil {
		return appshared.ErrInvalidInput
	}
	ok, err := s.repo.AllowRealNameAttempt(ctx, userID, "user", realnameMaxAttempts, realnameAttemptWindow)
	if err != nil {
		return err
	}
	if !ok {
		return appshared.ErrTooManyAttempts
	}
	if providerKey != "" {
		ok, err = s.repo.AllowRealNameAttempt(ctx, userID, "user:"+providerKey, realnameMaxAttemptsPerProvider, realnameAttemptWindow)
		if err != nil {
			return err
		}
		if !ok {
			return appshared.ErrTooManyAttempts
		}
	}
	return nil
}

// providerState is the versioned encrypted payload stored in the
// provider_state column. Token is the KYC continuation token; RedirectURL is
// the face-verification URL the user must visit. Both are sensitive and only
// ever exposed to the service after decryption.
type providerState struct {
	Version      int    `json:"v"`
	Token        string `json:"token,omitempty"`
	FaceProvider string `json:"face_provider,omitempty"`
	RedirectURL  string `json:"redirect_url,omitempty"`
}

func newStateCipher(secret string) (*cryptox.AESGCM, error) {
	sum := sha256.Sum256([]byte(secret))
	return cryptox.NewAESGCM(base64.RawURLEncoding.EncodeToString(sum[:]))
}

// sealState encrypts a provider state. Returns "" when there is nothing to
// store.
func (s *Service) sealState(st providerState) string {
	if s.cipher == nil || strings.TrimSpace(st.Token) == "" {
		return ""
	}
	st.Version = realnameStateVersion
	raw, err := json.Marshal(st)
	if err != nil {
		return ""
	}
	out, err := s.cipher.EncryptToString(raw)
	if err != nil {
		return ""
	}
	return out
}

// openState decrypts a provider state, failing closed on any error (bad
// ciphertext, tampering, missing key). ok=false means the state must not be
// used.
func (s *Service) openState(ciphertext string) (providerState, bool) {
	var st providerState
	if strings.TrimSpace(ciphertext) == "" {
		return st, false
	}
	if s.cipher == nil {
		return st, false
	}
	raw, err := s.cipher.DecryptString(ciphertext)
	if err != nil {
		return st, false
	}
	if err := json.Unmarshal(raw, &st); err != nil || st.Version != realnameStateVersion || strings.TrimSpace(st.Token) == "" {
		return providerState{}, false
	}
	return st, true
}

func (s *Service) GetConfig(ctx context.Context) (bool, string, []string) {
	enabled := false
	provider := "idcard_cn"
	actions := []string{"purchase_vps"}
	if s.settings == nil {
		return enabled, provider, actions
	}
	if setting, err := s.settings.GetSetting(ctx, "realname_enabled"); err == nil {
		enabled = strings.ToLower(strings.TrimSpace(setting.ValueJSON)) == "true"
	}
	if setting, err := s.settings.GetSetting(ctx, "realname_provider"); err == nil && strings.TrimSpace(setting.ValueJSON) != "" {
		provider = strings.TrimSpace(setting.ValueJSON)
	}
	if setting, err := s.settings.GetSetting(ctx, "realname_block_actions"); err == nil && strings.TrimSpace(setting.ValueJSON) != "" {
		var list []string
		if err := json.Unmarshal([]byte(setting.ValueJSON), &list); err == nil {
			actions = list
		}
	}
	return enabled, provider, actions
}

func (s *Service) UpdateConfig(ctx context.Context, enabled bool, provider string, actions []string) error {
	if s.settings == nil {
		return appshared.ErrInvalidInput
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "idcard_cn"
	}
	if s.registry != nil {
		if _, err := s.registry.GetProvider(provider); err != nil {
			return err
		}
	}
	raw, _ := json.Marshal(actions)
	enabledVal := "false"
	if enabled {
		enabledVal = "true"
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_enabled", ValueJSON: enabledVal}); err != nil {
		return err
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_provider", ValueJSON: provider}); err != nil {
		return err
	}
	return s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_block_actions", ValueJSON: string(raw)})
}

func (s *Service) Verify(ctx context.Context, userID int64, realName, idNumber string) (domain.RealNameVerification, error) {
	return s.VerifyWithInput(ctx, userID, RealNameVerifyInput{
		RealName: realName,
		IDNumber: idNumber,
	})
}

func (s *Service) VerifyWithInput(ctx context.Context, userID int64, in RealNameVerifyInput) (domain.RealNameVerification, error) {
	if s.repo == nil || s.registry == nil {
		return domain.RealNameVerification{}, appshared.ErrInvalidInput
	}
	enabled, providerKey, _ := s.GetConfig(ctx)
	if !enabled {
		return domain.RealNameVerification{}, appshared.ErrForbidden
	}
	provider, err := s.registry.GetProvider(providerKey)
	if err != nil {
		return domain.RealNameVerification{}, err
	}
	if err := s.allowAttempt(ctx, userID, provider.Key()); err != nil {
		return domain.RealNameVerification{}, err
	}
	ok := false
	reason := ""
	if ext, yes := provider.(RealNameProviderWithInput); yes {
		ok, reason, err = ext.VerifyWithInput(ctx, in)
	} else {
		ok, reason, err = provider.Verify(ctx, in.RealName, in.IDNumber)
	}
	if err != nil {
		return domain.RealNameVerification{}, err
	}
	status := "failed"
	var verifiedAt *time.Time
	providerState := ""
	if ok {
		status = "verified"
		now := time.Now()
		verifiedAt = &now
	} else if isPendingReason(reason) {
		status = "pending"
		// Keep the continuation token (and any redirect URL) out of the Reason
		// audit trail: move it into the versioned encrypted provider_state
		// column, and store only a stable sanitized status code in Reason.
		state, ok2 := parsePendingState(reason)
		if ok2 {
			providerState = s.sealState(state)
		}
		reason = stablePendingReason(reason)
	}
	reason = sanitizeReason(reason)
	record := domain.RealNameVerification{
		UserID:        userID,
		RealName:      appshared.SanitizePlainText(strings.TrimSpace(in.RealName)),
		IDNumber:      maskIDNumber(strings.TrimSpace(in.IDNumber)),
		Status:        status,
		Provider:      provider.Key(),
		Reason:        reason,
		ProviderState: providerState,
		VerifiedAt:    verifiedAt,
		CreatedAt:     time.Now(),
	}
	if err := s.repo.CreateRealNameVerification(ctx, &record); err != nil {
		return domain.RealNameVerification{}, err
	}
	return record, nil
}

// maskIDNumber persists only a masked identity-number (first 4 + last 4), so a
// database leak does not expose the full ID number. The provider still verifies
// the full value before this mask is applied.
func maskIDNumber(idNumber string) string {
	idNumber = strings.TrimSpace(idNumber)
	if len(idNumber) <= 8 {
		return "****"
	}
	return idNumber[:4] + "****" + idNumber[len(idNumber)-4:]
}

// sanitizeReason bounds the stored provider reason and strips control chars.
// Pending reasons embed provider tokens/URLs; truncation keeps them from being
// persisted and re-echoed as PII or raw secrets.
func sanitizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) > realnameReasonMaxLen {
		reason = reason[:realnameReasonMaxLen]
	}
	return appshared.SanitizePlainText(reason)
}

func (s *Service) PollPending(ctx context.Context, limit int) (int, error) {
	if s.repo == nil || s.registry == nil {
		return 0, appshared.ErrInvalidInput
	}
	if limit <= 0 {
		limit = realnamePollBudget
	}
	// Bound the attempt-budget table: drop windows older than one window (the
	// counters are only read within their own window).
	if s.repo != nil {
		_, _ = s.repo.CleanupRealNameAttempts(ctx, time.Now().Add(-realnameAttemptWindow))
	}
	pollers := map[string]RealNameProviderPendingPoller{}
	for _, provider := range s.registry.ListProviders() {
		if provider == nil {
			continue
		}
		poller, ok := provider.(RealNameProviderPendingPoller)
		if !ok {
			continue
		}
		pollers[provider.Key()] = poller
	}
	if len(pollers) == 0 {
		return 0, nil
	}
	// Page only status='pending' records by a stable id cursor. Each page is
	// bounded by realnamePollPageSize; the provider budget bounds how many of
	// those records are actually queried this round, so a large backlog cannot
	// burst-call the paid KYC provider in a single tick.
	var (
		afterID   int64
		processed int
		queries   int
	)
	for {
		items, err := s.repo.ListPendingRealNameVerifications(ctx, afterID, realnamePollPageSize)
		if err != nil {
			return processed, err
		}
		if len(items) == 0 {
			break
		}
		lastID := items[len(items)-1].ID
		for _, item := range items {
			poller, ok := pollers[item.Provider]
			if !ok {
				continue
			}
			// Poll cooldown: a record polled within the window is skipped this
			// tick. The persisted last_polled_at makes the throttle consistent
			// across instances and restarts, so a restart does not reset it.
			if item.LastPolledAt != nil && time.Since(*item.LastPolledAt) < realnamePollCooldown {
				continue
			}
			// Give-up cap: after realnameMaxPolls provider queries the record is
			// marked failed instead of being queried forever (and billed forever).
			if item.PollCount >= realnameMaxPolls {
				_ = s.repo.UpdateRealNameStatus(ctx, item.ID, "failed", "poll limit exceeded", nil)
				processed++
				continue
			}
			// Hard per-round budget on provider queries. Remaining records are
			// left for the next tick (the cooldown/cap checks above are cheap and
			// run regardless, but a full page of brand-new pending records cannot
			// exceed the budget in one call).
			if queries >= limit {
				return processed, nil
			}
			state, stateOK := s.openState(item.ProviderState)
			if !stateOK {
				// Controlled migration: legacy records stored the token inside
				// Reason. Parse it, re-persist it encrypted, and sanitize the
				// Reason so old data converges to the new format.
				legacy, legacyOK := parsePendingState(item.Reason)
				if !legacyOK || legacy.Token == "" {
					_ = s.repo.UpdateRealNameStatus(ctx, item.ID, "failed", "invalid pending token", nil)
					processed++
					continue
				}
				state = legacy
				encrypted := s.sealState(legacy)
				_ = s.repo.UpdateRealNameProviderState(ctx, item.ID, encrypted, stablePendingReason(item.Reason))
			}
			st, rs, qerr := poller.QueryPending(ctx, state.Token, state.FaceProvider)
			queries++
			// The cooldown/poll counters advance regardless of the provider
			// outcome (a provider error must not cause a per-tick retry).
			_ = s.repo.MarkRealNamePolled(ctx, item.ID, time.Now())
			if qerr != nil {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(st)) {
			case "verified":
				now := time.Now()
				_ = s.repo.UpdateRealNameStatus(ctx, item.ID, "verified", "", &now)
				processed++
			case "failed":
				_ = s.repo.UpdateRealNameStatus(ctx, item.ID, "failed", strings.TrimSpace(rs), nil)
				processed++
			}
		}
		afterID = lastID
	}
	return processed, nil
}

func isPendingReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "pending" {
		return true
	}
	if strings.HasPrefix(reason, "pending:") {
		return true
	}
	return strings.HasPrefix(reason, "pending_face:")
}

// parsePendingState parses the legacy pending reasons into a provider state.
// Supported formats:
//
//	pending:<token>                          -> simple pending
//	pending_face:<provider>:<token>          -> face verification pending
//	pending_face:<provider>:<token>:<url>    -> face pending with redirect URL
func parsePendingState(reason string) (providerState, bool) {
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "pending:") {
		parts := strings.SplitN(reason, ":", 2)
		if len(parts) != 2 {
			return providerState{}, false
		}
		t := strings.TrimSpace(parts[1])
		if t == "" {
			return providerState{}, false
		}
		return providerState{Token: t}, true
	}
	if !strings.HasPrefix(reason, "pending_face:") {
		return providerState{}, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(reason, "pending_face:"))
	// rest = <provider>:<token>[:<url>]
	providerPart, tokenPart, ok := strings.Cut(rest, ":")
	if !ok {
		return providerState{}, false
	}
	p := strings.TrimSpace(providerPart)
	token := strings.TrimSpace(tokenPart)
	if token == "" {
		return providerState{}, false
	}
	if idx := strings.Index(token, ":"); idx > 0 {
		token = strings.TrimSpace(token[:idx])
	}
	if token == "" {
		return providerState{}, false
	}
	if p != "baidu" && p != "wechat" {
		p = "baidu"
	}
	st := providerState{Token: token, FaceProvider: p}
	// Optional redirect URL comes after the token, base64url-encoded:
	// <provider>:<token>:<encoded-url>.
	if idx := strings.Index(tokenPart, ":"); idx >= 0 {
		if urlPart := strings.TrimSpace(tokenPart[idx+1:]); urlPart != "" {
			if raw, err := base64.RawURLEncoding.DecodeString(urlPart); err == nil {
				st.RedirectURL = strings.TrimSpace(string(raw))
			}
		}
	}
	return st, true
}

// stablePendingReason reduces a pending reason to a stable, non-secret status
// code so the audit trail never carries provider tokens or redirect URLs.
func stablePendingReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "pending_face:") {
		parts := strings.SplitN(reason, ":", 3)
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			return "pending_face:" + strings.TrimSpace(parts[1])
		}
		return "pending_face"
	}
	return "pending"
}

// parsePendingReason is kept for backward compatibility (tests and legacy
// callers); it returns the provider and token embedded in a legacy reason.
func parsePendingReason(reason string) (provider string, token string, ok bool) {
	st, ok := parsePendingState(reason)
	if !ok {
		return "", "", false
	}
	return st.FaceProvider, st.Token, true
}

// PendingRedirectURL decrypts a record's provider state and returns the
// face-verification redirect URL (never the token). It fails closed: any
// decryption problem yields "".
func (s *Service) PendingRedirectURL(ctx context.Context, record domain.RealNameVerification) string {
	if record.Provider == "" {
		return ""
	}
	state, ok := s.openState(record.ProviderState)
	if !ok {
		return ""
	}
	return strings.TrimSpace(state.RedirectURL)
}

func (s *Service) Latest(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	if s.repo == nil {
		return domain.RealNameVerification{}, appshared.ErrInvalidInput
	}
	return s.repo.GetLatestRealNameVerification(ctx, userID)
}

func (s *Service) List(ctx context.Context, userID *int64, status string, limit, offset int) ([]domain.RealNameVerification, int, error) {
	if s.repo == nil {
		return nil, 0, appshared.ErrInvalidInput
	}
	return s.repo.ListRealNameVerifications(ctx, userID, status, limit, offset)
}

func (s *Service) UpdateStatus(ctx context.Context, recordID int64, status string, reason string) error {
	if s.repo == nil {
		return appshared.ErrInvalidInput
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return appshared.ErrInvalidInput
	}
	var verifiedAt *time.Time
	if status == "verified" {
		now := time.Now()
		verifiedAt = &now
	}
	return s.repo.UpdateRealNameStatus(ctx, recordID, status, strings.TrimSpace(reason), verifiedAt)
}

func (s *Service) Providers() []RealNameProvider {
	if s.registry == nil {
		return nil
	}
	return s.registry.ListProviders()
}

func (s *Service) RequireAction(ctx context.Context, userID int64, action string) error {
	enabled, _, actions := s.GetConfig(ctx)
	if !enabled || action == "" {
		return nil
	}
	required := false
	for _, item := range actions {
		if strings.EqualFold(strings.TrimSpace(item), action) {
			required = true
			break
		}
	}
	if !required {
		return nil
	}
	latest, err := s.Latest(ctx, userID)
	if err != nil {
		return appshared.ErrRealNameRequired
	}
	if latest.Status != "verified" {
		return appshared.ErrRealNameRequired
	}
	return nil
}
