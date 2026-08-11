package robot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type WebhookNotifier struct {
	settings appports.SettingsRepository
	http     *http.Client
}

func NewWebhookNotifier(settings appports.SettingsRepository) *WebhookNotifier {
	return &WebhookNotifier{
		settings: settings,
		http:     newHTTPClient(),
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
		// Guard against SSRF: only http(s) URLs resolving to public addresses.
		if err := validateWebhookURL(hook.URL); err != nil {
			log.Printf("robot webhook %q skipped: %v", hook.Name, err)
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
		if err != nil {
			log.Printf("robot webhook %q request build failed: %v", hook.Name, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Event", event)
		if secret := hook.Secret; secret != "" {
			req.Header.Set("X-Signature", signHMACSHA256Hex(body, secret))
		}
		resp, err := n.http.Do(req)
		if err != nil {
			log.Printf("robot webhook %q send failed: %v", hook.Name, err)
			continue
		}
		// Drain the body so the connection can be reused; then close.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("robot webhook %q returned status %d", hook.Name, resp.StatusCode)
		}
	}
	return nil
}

func signHMACSHA256Hex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (n *WebhookNotifier) loadWebhooks(ctx context.Context) []appshared.RobotWebhookConfig {
	setting, err := n.settings.GetSetting(ctx, "robot_webhooks")
	if err == nil && setting.ValueJSON != "" {
		if hooks := appshared.ParseRobotWebhookConfigs(setting.ValueJSON); len(hooks) > 0 {
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
	return []appshared.RobotWebhookConfig{
		{
			Name:    "default",
			URL:     urlSetting.ValueJSON,
			Secret:  secretSetting.ValueJSON,
			Enabled: true,
		},
	}
}

// newHTTPClient returns the HTTP client used for webhook delivery, with a
// redirect policy that re-validates every redirect target against the SSRF
// guard.
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			if err := validateWebhookURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect target rejected: %w", err)
			}
			return nil
		},
	}
}

// validateWebhookURL enforces an SSRF-safe webhook target: the scheme must be
// http or https, the host must resolve, and none of the resolved addresses may
// be loopback, private, link-local, unspecified or multicast.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("webhook url scheme must be http or https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("webhook url missing host")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("webhook url host lookup failed: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("webhook url %q resolves to a blocked address", host)
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	return false
}
