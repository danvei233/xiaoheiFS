package robot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type WebhookNotifier struct {
	settings usecase.SettingsRepository
	http     *http.Client
}

func NewWebhookNotifier(settings usecase.SettingsRepository) *WebhookNotifier {
	return &WebhookNotifier{
		settings: settings,
		http:     &http.Client{Timeout: 8 * time.Second},
	}
}

func (n *WebhookNotifier) NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error {
	event := ev.Type
	webhooks := n.loadWebhooks(ctx)
	if len(webhooks) == 0 {
		return nil
	}

	envelope := map[string]any{
		"order_id":    ev.OrderID,
		"seq":         ev.Seq,
		"event":       ev.Type,
		"created_at":  ev.CreatedAt.Unix(),
		"data":        json.RawMessage(ev.DataJSON),
		"data_string": ev.DataJSON,
	}
	body, _ := json.Marshal(envelope)

	for _, hook := range webhooks {
		if !hook.Enabled || hook.URL == "" || !hook.MatchesEvent(event) {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Event", event)
		if secret := hook.Secret; secret != "" {
			req.Header.Set("X-Signature", signHMACSHA256Hex(body, secret))
		}
		resp, err := n.http.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
	}
	return nil
}

func signHMACSHA256Hex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (n *WebhookNotifier) loadWebhooks(ctx context.Context) []usecase.RobotWebhookConfig {
	setting, err := n.settings.GetSetting(ctx, "robot_webhooks")
	if err == nil && setting.ValueJSON != "" {
		if hooks := usecase.ParseRobotWebhookConfigs(setting.ValueJSON); len(hooks) > 0 {
			return hooks
		}
	}
	urlSetting, err := n.settings.GetSetting(ctx, "robot_webhook_url")
	if err != nil || urlSetting.ValueJSON == "" {
		return nil
	}
	enabledSetting, _ := n.settings.GetSetting(ctx, "robot_webhook_enabled")
	if enabledSetting.ValueJSON != "" && enabledSetting.ValueJSON != "true" {
		return nil
	}
	secretSetting, _ := n.settings.GetSetting(ctx, "robot_webhook_secret")
	return []usecase.RobotWebhookConfig{
		{
			Name:    "default",
			URL:     urlSetting.ValueJSON,
			Secret:  secretSetting.ValueJSON,
			Enabled: true,
		},
	}
}
