package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type RealNameService struct {
	repo     RealNameRepository
	registry RealNameProviderRegistry
	settings SettingsRepository
}

func NewRealNameService(repo RealNameRepository, registry RealNameProviderRegistry, settings SettingsRepository) *RealNameService {
	return &RealNameService{repo: repo, registry: registry, settings: settings}
}

func (s *RealNameService) GetConfig(ctx context.Context) (bool, string, []string) {
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

func (s *RealNameService) UpdateConfig(ctx context.Context, enabled bool, provider string, actions []string) error {
	if s.settings == nil {
		return ErrInvalidInput
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "idcard_cn"
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

func (s *RealNameService) Verify(ctx context.Context, userID int64, realName, idNumber string) (domain.RealNameVerification, error) {
	if s.repo == nil || s.registry == nil {
		return domain.RealNameVerification{}, ErrInvalidInput
	}
	enabled, providerKey, _ := s.GetConfig(ctx)
	if !enabled {
		return domain.RealNameVerification{}, ErrForbidden
	}
	provider, err := s.registry.GetProvider(providerKey)
	if err != nil {
		return domain.RealNameVerification{}, err
	}
	ok, reason, err := provider.Verify(ctx, realName, idNumber)
	if err != nil {
		return domain.RealNameVerification{}, err
	}
	status := "failed"
	var verifiedAt *time.Time
	if ok {
		status = "verified"
		now := time.Now()
		verifiedAt = &now
	}
	record := domain.RealNameVerification{
		UserID:     userID,
		RealName:   strings.TrimSpace(realName),
		IDNumber:   strings.TrimSpace(idNumber),
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

func (s *RealNameService) Latest(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	if s.repo == nil {
		return domain.RealNameVerification{}, ErrInvalidInput
	}
	return s.repo.GetLatestRealNameVerification(ctx, userID)
}

func (s *RealNameService) List(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error) {
	if s.repo == nil {
		return nil, 0, ErrInvalidInput
	}
	return s.repo.ListRealNameVerifications(ctx, userID, limit, offset)
}

func (s *RealNameService) UpdateStatus(ctx context.Context, recordID int64, status string, reason string) error {
	if s.repo == nil {
		return ErrInvalidInput
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return ErrInvalidInput
	}
	var verifiedAt *time.Time
	if status == "verified" {
		now := time.Now()
		verifiedAt = &now
	}
	return s.repo.UpdateRealNameStatus(ctx, recordID, status, strings.TrimSpace(reason), verifiedAt)
}

func (s *RealNameService) Providers() []RealNameProvider {
	if s.registry == nil {
		return nil
	}
	return s.registry.ListProviders()
}

func (s *RealNameService) RequireAction(ctx context.Context, userID int64, action string) error {
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
		return ErrRealNameRequired
	}
	if latest.Status != "verified" {
		return ErrRealNameRequired
	}
	return nil
}
