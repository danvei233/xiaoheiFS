package testutil

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/app/shared"
)

func TestFakeAutomationClient(t *testing.T) {
	f := &FakeAutomationClient{}
	res, err := f.CreateHost(context.Background(), shared.AutomationCreateHostRequest{LineID: 1, OS: "linux", CPU: 1, MemoryGB: 1, DiskGB: 10, Bandwidth: 10, PortNum: 30, HostName: "vm"})
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	if res.HostID == 0 {
		t.Fatalf("expected host id")
	}
	if _, err := f.GetHostInfo(context.Background(), res.HostID); err != nil {
		t.Fatalf("get host info: %v", err)
	}
	if _, err := f.ListHostSimple(context.Background(), "vm"); err != nil {
		t.Fatalf("list host simple: %v", err)
	}
	if err := f.ElasticUpdate(context.Background(), shared.AutomationElasticUpdateRequest{HostID: res.HostID}); err != nil {
		t.Fatalf("elastic update: %v", err)
	}
	if err := f.RenewHost(context.Background(), res.HostID, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("renew host: %v", err)
	}
	if _, err := f.ListAreas(context.Background()); err != nil {
		t.Fatalf("list areas: %v", err)
	}
	if _, err := f.ListLines(context.Background()); err != nil {
		t.Fatalf("list lines: %v", err)
	}
	if _, err := f.ListProducts(context.Background(), 1); err != nil {
		t.Fatalf("list products: %v", err)
	}
	if _, err := f.GetMonitor(context.Background(), res.HostID); err != nil {
		t.Fatalf("get monitor: %v", err)
	}
	if _, err := f.GetVNCURL(context.Background(), res.HostID); err != nil {
		t.Fatalf("get vnc url: %v", err)
	}
	if err := f.LockHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("lock host: %v", err)
	}
	if err := f.UnlockHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("unlock host: %v", err)
	}
	if err := f.StartHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("start host: %v", err)
	}
	if err := f.ShutdownHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("shutdown host: %v", err)
	}
	if err := f.RebootHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("reboot host: %v", err)
	}
	if err := f.DeleteHost(context.Background(), res.HostID); err != nil {
		t.Fatalf("delete host: %v", err)
	}
	if _, err := f.GetPanelURL(context.Background(), "host", "pwd"); err != nil {
		t.Fatalf("get panel url: %v", err)
	}
	if _, err := f.ListImages(context.Background(), 1); err != nil {
		t.Fatalf("list images: %v", err)
	}
}

func TestFakePaymentProvider(t *testing.T) {
	p := &FakePaymentProvider{KeyVal: "fake", NameVal: "Fake", Schema: "{}"}
	if p.Key() != "fake" || p.Name() == "" || p.SchemaJSON() == "" {
		t.Fatalf("unexpected provider metadata")
	}
	if err := p.SetConfig(`{"k":"v"}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if _, err := p.CreatePayment(context.Background(), shared.PaymentCreateRequest{}); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if _, err := p.VerifyNotify(context.Background(), shared.RawHTTPRequest{RawQuery: "trade_no=t"}); err != nil {
		t.Fatalf("verify notify: %v", err)
	}
}

func TestFakeRegistriesAndNotifiers(t *testing.T) {
	reg := NewFakePaymentRegistry()
	provider := &FakePaymentProvider{KeyVal: "fake", NameVal: "Fake", Schema: "{}"}
	reg.RegisterProvider(provider, true, `{"k":"v"}`)
	if _, err := reg.ListProviders(context.Background(), true); err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if _, err := reg.GetProvider(context.Background(), "fake"); err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if _, enabled, err := reg.GetProviderConfig(context.Background(), "fake"); err != nil || !enabled {
		t.Fatalf("get provider config: %v", err)
	}
	if err := reg.UpdateProviderConfig(context.Background(), "fake", false, `{"k":"v2"}`); err != nil {
		t.Fatalf("update provider config: %v", err)
	}

	email := &FakeEmailSender{}
	if err := email.Send(context.Background(), "a@b.com", "subject", "body"); err != nil {
		t.Fatalf("email send: %v", err)
	}

	robot := &FakeRobotNotifier{}
	if err := robot.NotifyOrderPending(context.Background(), shared.RobotOrderPayload{OrderNo: "ORD-1", UserID: 1, Amount: 100, Currency: "CNY"}); err != nil {
		t.Fatalf("robot notify: %v", err)
	}

	realnameReg := NewFakeRealNameRegistry()
	if p, err := realnameReg.GetProvider("fake"); err != nil || p.Key() == "" || p.Name() == "" {
		t.Fatalf("realname get provider: %v", err)
	}
	if len(realnameReg.ListProviders()) == 0 {
		t.Fatalf("realname list providers empty")
	}
	providerX := &FakeRealNameProvider{KeyVal: "x", NameVal: "X", OK: true}
	realnameReg.Register(providerX)
	if ok, _, err := providerX.Verify(context.Background(), "name", "id"); err != nil || !ok {
		t.Fatalf("realname verify: %v", err)
	}
}
