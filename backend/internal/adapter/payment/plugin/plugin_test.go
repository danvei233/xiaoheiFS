package paymentplugin

import (
	"net"
	"net/rpc"
	"testing"

	"xiaoheiplay/internal/usecase"
)

type fakeProvider struct {
	config string
	req    usecase.PaymentCreateRequest
	notify map[string]string
}

func (f *fakeProvider) Key() string        { return "fake" }
func (f *fakeProvider) Name() string       { return "Fake" }
func (f *fakeProvider) SchemaJSON() string { return `{"x":1}` }
func (f *fakeProvider) SetConfig(configJSON string) error {
	f.config = configJSON
	return nil
}
func (f *fakeProvider) CreatePayment(req usecase.PaymentCreateRequest) (usecase.PaymentCreateResult, error) {
	f.req = req
	return usecase.PaymentCreateResult{PayURL: "https://pay", TradeNo: "TN"}, nil
}
func (f *fakeProvider) VerifyNotify(params map[string]string) (usecase.PaymentNotifyResult, error) {
	f.notify = params
	return usecase.PaymentNotifyResult{TradeNo: "TN", Paid: true, Amount: 1000}, nil
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
	req := usecase.PaymentCreateRequest{OrderID: 1, Amount: 990}
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
}
