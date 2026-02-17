package push

import (
	"context"
	"fmt"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
)

type (
	PushConfig  = appshared.PushConfig
	PushPayload = appshared.PushPayload
)

type Service struct {
	tokens   appports.PushTokenRepository
	users    appports.UserRepository
	settings appports.SettingsRepository
	sender   appports.PushSender
}

func NewService(tokens appports.PushTokenRepository, users appports.UserRepository, settings appports.SettingsRepository, sender appports.PushSender) *Service {
	return &Service{
		tokens:   tokens,
		users:    users,
		settings: settings,
		sender:   sender,
	}
}

func (s *Service) RegisterToken(ctx context.Context, userID int64, platform, token, deviceID string) error {
	if s.tokens == nil {
		return appshared.ErrInvalidInput
	}
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return appshared.ErrInvalidInput
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = "android"
	}
	now := time.Now()
	return s.tokens.UpsertPushToken(ctx, &domain.PushToken{
		UserID:    userID,
		Platform:  platform,
		Token:     token,
		DeviceID:  strings.TrimSpace(deviceID),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *Service) RemoveToken(ctx context.Context, userID int64, token string) error {
	if s.tokens == nil {
		return appshared.ErrInvalidInput
	}
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return appshared.ErrInvalidInput
	}
	return s.tokens.DeletePushToken(ctx, userID, token)
}

func (s *Service) NotifyAdminsNewOrder(ctx context.Context, order domain.Order) error {
	if s.sender == nil || s.tokens == nil || s.users == nil || s.settings == nil {
		return nil
	}
	enabled, _ := s.settings.GetSetting(ctx, "fcm_enabled")
	if enabled.ValueJSON != "" && strings.ToLower(enabled.ValueJSON) != "true" {
		return nil
	}
	cfg := PushConfig{}
	if projectSetting, err := s.settings.GetSetting(ctx, "fcm_project_id"); err == nil {
		cfg.ProjectID = strings.TrimSpace(projectSetting.ValueJSON)
	}
	if saSetting, err := s.settings.GetSetting(ctx, "fcm_service_account_json"); err == nil {
		cfg.ServiceAccountJSON = strings.TrimSpace(saSetting.ValueJSON)
	}
	if keySetting, err := s.settings.GetSetting(ctx, "fcm_server_key"); err == nil {
		cfg.LegacyServerKey = strings.TrimSpace(keySetting.ValueJSON)
	}
	if (cfg.ProjectID == "" || cfg.ServiceAccountJSON == "") && cfg.LegacyServerKey == "" {
		return nil
	}
	adminIDs := make([]int64, 0, 8)
	offset := 0
	for {
		admins, total, err := s.users.ListUsersByRoleStatus(ctx, "admin", "active", 200, offset)
		if err != nil {
			return err
		}
		if len(admins) == 0 {
			break
		}
		for _, admin := range admins {
			adminIDs = append(adminIDs, admin.ID)
		}
		offset += len(admins)
		if offset >= total {
			break
		}
	}
	if len(adminIDs) == 0 {
		return nil
	}
	tokens, err := s.tokens.ListPushTokensByUserIDs(ctx, adminIDs)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}
	uniqueTokens := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t.Token == "" {
			continue
		}
		if _, ok := seen[t.Token]; ok {
			continue
		}
		seen[t.Token] = struct{}{}
		uniqueTokens = append(uniqueTokens, t.Token)
	}
	if len(uniqueTokens) == 0 {
		return nil
	}
	orderNo := strings.TrimSpace(order.OrderNo)
	if orderNo == "" {
		orderNo = fmt.Sprintf("#%d", order.ID)
	}
	currency := strings.TrimSpace(order.Currency)
	if currency == "" {
		currency = "CNY"
	}
	amount := money.FormatCents(order.TotalAmount)
	title := "新订单待审核"
	body := fmt.Sprintf("订单 %s 金额 %s %s", orderNo, amount, currency)
	payload := PushPayload{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"order_id": fmt.Sprintf("%d", order.ID),
			"order_no": order.OrderNo,
			"status":   string(order.Status),
			"amount":   amount,
			"currency": currency,
		},
	}
	return s.sender.Send(ctx, cfg, uniqueTokens, payload)
}
