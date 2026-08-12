package paymentplugin

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"reflect"
	"testing"
	"time"

	"xiaoheiplay/internal/app/shared"
)

type fakeProvider struct {
	config   string
	req      shared.PaymentCreateRequest
	notify   map[string]string
	notifyV2 shared.RawHTTPRequest
}

func (f *fakeProvider) Key() string        { return "fake" }
func (f *fakeProvider) Name() string       { return "Fake" }
func (f *fakeProvider) SchemaJSON() string { return `{"x":1}` }
func (f *fakeProvider) SetConfig(configJSON string) error {
	f.config = configJSON
	return nil
}
func (f *fakeProvider) CreatePayment(req shared.PaymentCreateRequest) (shared.PaymentCreateResult, error) {
	f.req = req
	return shared.PaymentCreateResult{PayURL: "https://pay", TradeNo: "TN"}, nil
}
func (f *fakeProvider) VerifyNotify(params map[string]string) (shared.PaymentNotifyResult, error) {
	f.notify = params
	return shared.PaymentNotifyResult{TradeNo: "TN", Paid: true, Amount: 1000}, nil
}
func (f *fakeProvider) VerifyNotifyV2(req shared.RawHTTPRequest) (shared.PaymentNotifyResult, error) {
	f.notifyV2 = req
	return shared.PaymentNotifyResult{TradeNo: "TN", Paid: true, Amount: 1000}, nil
}

func TestProviderRPC(t *testing.T) {
	fake := &fakeProvider{}
	server := rpc.NewServer()
	if err := server.RegisterName("Plugin", &providerRPCServer{Impl: fake}); err != nil {
		t.Fatalf("register: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	go server.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	p := &providerRPC{client: client}

	if p.Key() != "fake" || p.Name() != "Fake" || p.SchemaJSON() == "" {
		t.Fatalf("unexpected provider info")
	}
	if err := p.SetConfig(`{"on":true}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if fake.config == "" {
		t.Fatalf("expected config set")
	}
	req := shared.PaymentCreateRequest{OrderID: 1, Amount: 990}
	if _, err := p.CreatePayment(req); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if fake.req.OrderID != 1 {
		t.Fatalf("expected request captured")
	}
	if _, err := p.VerifyNotify(map[string]string{"trade_no": "TN"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if fake.notify["trade_no"] != "TN" {
		t.Fatalf("expected notify params captured")
	}
	// Versioned V2 contract: JSON and multipart requests round-trip through
	// net/rpc unchanged. Query, headers, and body remain separate signing inputs.
	v2Requests := []shared.RawHTTPRequest{
		{
			Method:   "POST",
			Path:     "/api/v1/payments/notify/demo",
			RawQuery: "status=query&opaque=%2f",
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Signature":  {"first", "second"},
			},
			Body: []byte(`{"status":"body","cipher":"a=b"}`),
		},
		{
			Method:   "PUT",
			Path:     "/api/v1/payments/notify/demo/multipart",
			RawQuery: "status=query",
			Headers: map[string][]string{
				"Content-Type": {"multipart/form-data; boundary=raw-boundary"},
				"X-Signature":  {"multipart-signature"},
			},
			Body: []byte("--raw-boundary\r\nContent-Disposition: form-data; name=\"status\"\r\n\r\nbody\r\n--raw-boundary--\r\n"),
		},
	}
	for _, v2req := range v2Requests {
		if _, err := p.VerifyNotifyV2(v2req); err != nil {
			t.Fatalf("verify v2: %v", err)
		}
		if !reflect.DeepEqual(fake.notifyV2, v2req) {
			t.Fatalf("V2 request did not round-trip\nwant: %#v\n got: %#v", v2req, fake.notifyV2)
		}
	}
}

// TestProviderRPC_UnknownV2FailsClosed verifies a legacy plugin binary that only
// implements VerifyNotify (v1) is refused with a fail-closed error rather than
// downgrading the raw callback to a params map.
func TestProviderRPC_UnknownV2FailsClosed(t *testing.T) {
	// Register a real v1-only RPC surface with no VerifyNotifyV2 method.
	legacy := &v1OnlyRPCServer{}
	server := rpc.NewServer()
	if err := server.RegisterName("Plugin", legacy); err != nil {
		t.Fatalf("register: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	go server.ServeConn(serverConn)

	client := rpc.NewClient(clientConn)
	p := &providerRPC{client: client}
	_, err := p.VerifyNotifyV2(shared.RawHTTPRequest{Method: "POST"})
	if err == nil {
		t.Fatalf("expected fail-closed error for a v1-only plugin")
	}
	if got := err.Error(); got != "payment plugin does not support the versioned VerifyNotifyV2 contract; refusing to downgrade to params" {
		t.Fatalf("expected the error to name the missing V2 contract, got: %v", err)
	}
	if legacy.verifyNotifyCalls != 0 {
		t.Fatalf("host downgraded to VerifyNotify v1: calls=%d", legacy.verifyNotifyCalls)
	}
}

type v1OnlyRPCServer struct {
	verifyNotifyCalls int
}

func (s *v1OnlyRPCServer) VerifyNotify(_ NotifyArgs, resp *shared.PaymentNotifyResult) error {
	s.verifyNotifyCalls++
	*resp = shared.PaymentNotifyResult{Paid: true}
	return nil
}

type v2BusinessErrorRPCServer struct{}

func (s *v2BusinessErrorRPCServer) VerifyNotifyV2(_ NotifyV2Args, _ *shared.PaymentNotifyResult) error {
	return fmt.Errorf("payment order not found")
}

func TestProviderRPC_V2BusinessNotFoundErrorIsNotMisclassified(t *testing.T) {
	p, close := newTestProviderRPC(t, &v2BusinessErrorRPCServer{})
	defer close()

	_, err := p.VerifyNotifyV2(shared.RawHTTPRequest{Method: "POST"})
	if err == nil || err.Error() != "payment order not found" {
		t.Fatalf("business error was misclassified as a missing V2 contract: %v", err)
	}
}

// rpcPluginServerFail reports a plain error from CreatePayment, mirroring a
// plugin rejecting a payment request.
type rpcPluginServerFail struct{}

func (s *rpcPluginServerFail) CreatePayment(args CreateArgs, resp *shared.PaymentCreateResult) error {
	return errors.New("plugin says no")
}

// rpcPluginServerBlocking never responds to CreatePayment, simulating a hung
// legacy net-RPC plugin.
type rpcPluginServerBlocking struct{}

func (s *rpcPluginServerBlocking) CreatePayment(args CreateArgs, resp *shared.PaymentCreateResult) error {
	select {}
}

func newTestProviderRPC(t *testing.T, server any) (*providerRPC, func()) {
	t.Helper()
	srv := rpc.NewServer()
	if err := srv.RegisterName("Plugin", server); err != nil {
		t.Fatalf("register plugin server: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	go srv.ServeConn(serverConn)
	rpcClient := rpc.NewClient(clientConn)
	return &providerRPC{client: rpcClient}, func() { _ = rpcClient.Close() }
}

func TestProviderRPC_CreatePayment_PropagatesPluginError(t *testing.T) {
	p, close := newTestProviderRPC(t, &rpcPluginServerFail{})
	defer close()

	if _, err := p.CreatePayment(shared.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected plugin error to propagate")
	}
}

// M5: a hung plugin must not block forever; the call must time out.
func TestProviderRPC_CreatePayment_TimesOut(t *testing.T) {
	p, close := newTestProviderRPC(t, &rpcPluginServerBlocking{})
	defer close()

	old := paymentPluginRPCTimeout
	paymentPluginRPCTimeout = 50 * time.Millisecond
	defer func() { paymentPluginRPCTimeout = old }()

	start := time.Now()
	if _, err := p.CreatePayment(shared.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected timeout error from hung plugin")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout took too long: %v", elapsed)
	}
}
