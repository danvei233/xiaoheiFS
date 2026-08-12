package payment

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
)

// blockingProvider implements the legacy net-RPC Provider interface in-process
// and never returns, simulating a hung plugin.
type blockingProvider struct{}

func (b *blockingProvider) Key() string            { return "blocking" }
func (b *blockingProvider) Name() string           { return "Blocking" }
func (b *blockingProvider) SchemaJSON() string     { return "" }
func (b *blockingProvider) SetConfig(string) error { return nil }
func (b *blockingProvider) CreatePayment(appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	select {}
}
func (b *blockingProvider) VerifyNotify(map[string]string) (appshared.PaymentNotifyResult, error) {
	select {}
}
func (b *blockingProvider) VerifyNotifyV2(appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	select {}
}

// M5: pluginProvider must honor a caller-provided ctx deadline even though the
// legacy Provider interface has no ctx parameter.
func TestPluginProvider_CreatePayment_HonorsContext(t *testing.T) {
	p := &pluginProvider{provider: &blockingProvider{}}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := p.CreatePayment(ctx, appshared.PaymentCreateRequest{})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("expected a context error from a hung plugin")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("CreatePayment did not honor the context deadline")
	}
}

type notifyCaptureProvider struct {
	v1Calls int
	v2Raw   appshared.RawHTTPRequest
}

func (p *notifyCaptureProvider) Key() string            { return "capture" }
func (p *notifyCaptureProvider) Name() string           { return "Capture" }
func (p *notifyCaptureProvider) SchemaJSON() string     { return "" }
func (p *notifyCaptureProvider) SetConfig(string) error { return nil }
func (p *notifyCaptureProvider) CreatePayment(appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	return appshared.PaymentCreateResult{}, nil
}
func (p *notifyCaptureProvider) VerifyNotify(map[string]string) (appshared.PaymentNotifyResult, error) {
	p.v1Calls++
	return appshared.PaymentNotifyResult{}, nil
}
func (p *notifyCaptureProvider) VerifyNotifyV2(req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	p.v2Raw = req
	return appshared.PaymentNotifyResult{Paid: true}, nil
}

func TestPluginProvider_VerifyNotifyUsesOnlyRawV2Contract(t *testing.T) {
	impl := &notifyCaptureProvider{}
	p := &pluginProvider{provider: impl}
	raw := appshared.RawHTTPRequest{
		Method:   "POST",
		Path:     "/api/v1/payments/notify/capture",
		RawQuery: "status=query",
		Headers:  map[string][]string{"Content-Type": {"application/json"}, "X-Signature": {"one", "two"}},
		Body:     []byte(`{"status":"body"}`),
	}

	if _, err := p.VerifyNotify(context.Background(), raw); err != nil {
		t.Fatalf("verify raw callback: %v", err)
	}
	if impl.v1Calls != 0 {
		t.Fatalf("legacy host downgraded raw callback to v1 params: calls=%d", impl.v1Calls)
	}
	if !reflect.DeepEqual(impl.v2Raw, raw) {
		t.Fatalf("host changed callback before V2 dispatch\nwant: %#v\n got: %#v", raw, impl.v2Raw)
	}
}
