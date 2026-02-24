package realname

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type (
	RealNameProvider              = appshared.RealNameProvider
	RealNameProviderWithInput     = appshared.RealNameProviderWithInput
	RealNameProviderPendingPoller = appshared.RealNameProviderPendingPoller
	RealNameProviderRegistry      = appshared.RealNameProviderRegistry
	RealNameVerifyInput           = appshared.RealNameVerifyInput
)

type Service struct {
	repo     appports.RealNameRepository
	registry RealNameProviderRegistry
	settings appports.SettingsRepository
}

func NewService(repo appports.RealNameRepository, registry appshared.RealNameProviderRegistry, settings appports.SettingsRepository) *Service {
	return &Service{repo: repo, registry: registry, settings: settings}
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
	if ok {
		status = "verified"
		now := time.Now()
		verifiedAt = &now
	} else if isPendingReason(reason) {
		status = "pending"
		reason = strings.TrimSpace(reason)
	}
	record := domain.RealNameVerification{
		UserID:     userID,
		RealName:   strings.TrimSpace(in.RealName),
		IDNumber:   strings.TrimSpace(in.IDNumber),
		Status:     status,
		Provider:   provider.Key(),
		Reason:     reason,
		VerifiedAt: verifiedAt,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateRealNameVerification(ctx, &record); err != nil {
		return domain.RealNameVerification{}, err
	}
	return record, nil
}

func (s *Service) PollPending(ctx context.Context, limit int) (int, error) {
	if s.repo == nil || s.registry == nil {
		return 0, appshared.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 200
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
	offset := 0
	processed := 0
	for {
		items, total, err := s.repo.ListRealNameVerifications(ctx, nil, limit, offset)
		if err != nil {
			return processed, err
		}
		if len(items) == 0 {
			break
		}
		for _, item := range items {
			if strings.ToLower(strings.TrimSpace(item.Status)) != "pending" {
				continue
			}
			poller, ok := pollers[item.Provider]
			if !ok {
				continue
			}
			p, token, ok := parsePendingReason(item.Reason)
			if !ok || token == "" {
				_ = s.repo.UpdateRealNameStatus(ctx, item.ID, "failed", "invalid pending token", nil)
				processed++
				continue
			}
			st, rs, qerr := poller.QueryPending(ctx, token, p)
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
		offset += len(items)
		if offset >= total {
			break
		}
	}
	return processed, nil
}

func isPendingReason(reason string) bool {
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "pending:") {
		return true
	}
	return strings.HasPrefix(reason, "pending_face:")
}

func parsePendingReason(reason string) (provider string, token string, ok bool) {
	reason = strings.TrimSpace(reason)
	if strings.HasPrefix(reason, "pending:") {
		parts := strings.SplitN(reason, ":", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		t := strings.TrimSpace(parts[1])
		if t == "" {
			return "", "", false
		}
		return "", t, true
	}
	if !strings.HasPrefix(reason, "pending_face:") {
		return "", "", false
	}
	parts := strings.SplitN(reason, ":", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	p := strings.TrimSpace(parts[1])
	third := strings.TrimSpace(parts[2])
	if third == "" {
		return "", "", false
	}
	t := third
	// Support extended format: pending_face:<provider>:<token>:<extra>
	if idx := strings.Index(third, ":"); idx > 0 {
		t = strings.TrimSpace(third[:idx])
	}
	if t == "" {
		return "", "", false
	}
	if p != "baidu" && p != "wechat" {
		p = "baidu"
	}
	return p, t, true
}

func (s *Service) Latest(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	if s.repo == nil {
		return domain.RealNameVerification{}, appshared.ErrInvalidInput
	}
	return s.repo.GetLatestRealNameVerification(ctx, userID)
}

func (s *Service) List(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error) {
	if s.repo == nil {
		return nil, 0, appshared.ErrInvalidInput
	}
	return s.repo.ListRealNameVerifications(ctx, userID, limit, offset)
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
