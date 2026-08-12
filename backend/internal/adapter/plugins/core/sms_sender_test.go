package plugins

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// --- F4: Send must be bounded even when the plugin hangs --------------------

func TestSMSSendIsBoundedWhenPluginHangs(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", true)
	factory := &fakeClientFactory{withSMS: true}
	factory.smsSendFn = func(ctx context.Context, _ *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error) {
		// Simulate a hung plugin: block until the caller gives up or a generous
		// upper bound elapses (so the test terminates even without the fix).
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			return nil, errors.New("upstream hung")
		}
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:   "sms",
		PluginID:   "demo",
		InstanceID: "default",
		Enabled:    true,
	}}
	m := NewManager(base, repo, testCipher(t), nil)
	m.runtime.newClient = factory.new
	s := NewSMSSender(m)
	s.timeout = 100 * time.Millisecond

	start := time.Now()
	_, err := s.Send(context.Background(), "demo", "default", appshared.SMSMessage{
		Content: "hello",
		Phones:  []string{"13800000000"},
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from hung plugin")
	}
	if elapsed > time.Second {
		t.Fatalf("Send was not bounded by the operation timeout: took %v", elapsed)
	}
}

// --- F5: a rejected Send must not leak plugin-internal details ---------------

func TestSMSSendRejectionDoesNotLeakInternalDetail(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "sms", "demo", true)
	factory := &fakeClientFactory{withSMS: true}
	factory.smsSendFn = func(context.Context, *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error) {
		return &pluginv1.SendSmsResponse{
			Ok:        false,
			ErrorCode: "E_AUTH",
			Error:     "gateway refused: dial tcp 169.254.169.254:80 token=fixture_secret_token_1234567890",
		}, nil
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:   "sms",
		PluginID:   "demo",
		InstanceID: "default",
		Enabled:    true,
	}}
	m := NewManager(base, repo, testCipher(t), nil)
	m.runtime.newClient = factory.new
	s := NewSMSSender(m)

	_, err := s.Send(context.Background(), "demo", "default", appshared.SMSMessage{Content: "hi", Phones: []string{"13800000000"}})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "169.254.169.254") || strings.Contains(msg, "fixture_secret_token_1234567890") {
		t.Fatalf("plugin-internal detail leaked to caller: %q", msg)
	}
}
