package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type RealNameService struct {
	repo     RealNameRepository
	registry RealNameProviderRegistry
	settings SettingsRepository
}

type RealNameMangzhuConfig struct {
	BaseURL      string
	Key          string
	AuthMode     string
	FaceProvider string
	TimeoutSec   int
	KeySet       bool
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

func (s *RealNameService) GetMangzhuConfig(ctx context.Context) RealNameMangzhuConfig {
	cfg := RealNameMangzhuConfig{
		BaseURL:      "https://e.mangzhuyun.cn",
		AuthMode:     "three_factor",
		FaceProvider: "baidu",
		TimeoutSec:   10,
	}
	if s.settings == nil {
		return cfg
	}
	if st, err := s.settings.GetSetting(ctx, "realname_mangzhu_base_url"); err == nil && strings.TrimSpace(st.ValueJSON) != "" {
		cfg.BaseURL = strings.TrimSpace(st.ValueJSON)
	}
	if st, err := s.settings.GetSetting(ctx, "realname_mangzhu_key"); err == nil {
		cfg.Key = strings.TrimSpace(st.ValueJSON)
		cfg.KeySet = cfg.Key != ""
	}
	if st, err := s.settings.GetSetting(ctx, "realname_mangzhu_auth_mode"); err == nil && strings.TrimSpace(st.ValueJSON) != "" {
		cfg.AuthMode = strings.TrimSpace(strings.ToLower(st.ValueJSON))
	}
	if st, err := s.settings.GetSetting(ctx, "realname_mangzhu_face_provider"); err == nil && strings.TrimSpace(st.ValueJSON) != "" {
		cfg.FaceProvider = strings.TrimSpace(strings.ToLower(st.ValueJSON))
	}
	if st, err := s.settings.GetSetting(ctx, "realname_mangzhu_timeout_sec"); err == nil && strings.TrimSpace(st.ValueJSON) != "" {
		var sec int
		if _, err := fmt.Sscanf(strings.TrimSpace(st.ValueJSON), "%d", &sec); err == nil && sec > 0 && sec <= 60 {
			cfg.TimeoutSec = sec
		}
	}
	if cfg.AuthMode != "two_factor" && cfg.AuthMode != "three_factor" && cfg.AuthMode != "face" {
		cfg.AuthMode = "three_factor"
	}
	if cfg.FaceProvider != "baidu" && cfg.FaceProvider != "wechat" {
		cfg.FaceProvider = "baidu"
	}
	return cfg
}

func (s *RealNameService) UpdateMangzhuConfig(ctx context.Context, cfg RealNameMangzhuConfig) error {
	if s.settings == nil {
		return ErrInvalidInput
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://e.mangzhuyun.cn"
	}
	authMode := strings.TrimSpace(strings.ToLower(cfg.AuthMode))
	if authMode == "" {
		authMode = "three_factor"
	}
	if authMode != "two_factor" && authMode != "three_factor" && authMode != "face" {
		return ErrInvalidInput
	}
	faceProvider := strings.TrimSpace(strings.ToLower(cfg.FaceProvider))
	if faceProvider == "" {
		faceProvider = "baidu"
	}
	if faceProvider != "baidu" && faceProvider != "wechat" {
		return ErrInvalidInput
	}
	timeoutSec := cfg.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 10
	}
	if timeoutSec > 60 {
		timeoutSec = 60
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_mangzhu_base_url", ValueJSON: baseURL}); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Key) != "" {
		if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_mangzhu_key", ValueJSON: strings.TrimSpace(cfg.Key)}); err != nil {
			return err
		}
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_mangzhu_auth_mode", ValueJSON: authMode}); err != nil {
		return err
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_mangzhu_face_provider", ValueJSON: faceProvider}); err != nil {
		return err
	}
	return s.settings.UpsertSetting(ctx, domain.Setting{Key: "realname_mangzhu_timeout_sec", ValueJSON: fmt.Sprintf("%d", timeoutSec)})
}

func (s *RealNameService) Verify(ctx context.Context, userID int64, realName, idNumber string) (domain.RealNameVerification, error) {
	return s.VerifyWithInput(ctx, userID, RealNameVerifyInput{
		RealName: realName,
		IDNumber: idNumber,
	})
}

func (s *RealNameService) VerifyWithInput(ctx context.Context, userID int64, in RealNameVerifyInput) (domain.RealNameVerification, error) {
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

func (s *RealNameService) PollPending(ctx context.Context, limit int) (int, error) {
	if s.repo == nil || s.registry == nil {
		return 0, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 200
	}
	provider, err := s.registry.GetProvider("mangzhu_realname")
	if err != nil {
		return 0, nil
	}
	poller, ok := provider.(RealNameProviderPendingPoller)
	if !ok {
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
			if item.Provider != "mangzhu_realname" || strings.ToLower(strings.TrimSpace(item.Status)) != "pending" {
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
			default:
				// pending: keep unchanged
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
	return strings.HasPrefix(reason, "pending_face:")
}

func parsePendingReason(reason string) (provider string, token string, ok bool) {
	reason = strings.TrimSpace(reason)
	if !strings.HasPrefix(reason, "pending_face:") {
		return "", "", false
	}
	parts := strings.SplitN(reason, ":", 3)
	if len(parts) != 3 {
		return "", "", false
	}
	p := strings.TrimSpace(parts[1])
	t := strings.TrimSpace(parts[2])
	if t == "" {
		return "", "", false
	}
	if p != "baidu" && p != "wechat" {
		p = "baidu"
	}
	return p, t, true
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
