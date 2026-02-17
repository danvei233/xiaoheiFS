package paymentplugin

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"

	appshared "xiaoheiplay/internal/app/shared"
)

const (
	ProviderPluginName = "provider"
	MagicCookieKey     = "PAYMENT_PLUGIN"
	MagicCookieValue   = "xiaoheiplay"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   MagicCookieKey,
	MagicCookieValue: MagicCookieValue,
}

type Provider interface {
	Key() string
	Name() string
	SchemaJSON() string
	SetConfig(configJSON string) error
	CreatePayment(req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error)
	VerifyNotify(params map[string]string) (appshared.PaymentNotifyResult, error)
}

type ProviderPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl Provider
}

func (p *ProviderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &providerRPCServer{Impl: p.Impl}, nil
}

func (p *ProviderPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &providerRPC{client: c}, nil
}

type EmptyArgs struct{}

type ConfigArgs struct {
	ConfigJSON string
}

type CreateArgs struct {
	Request appshared.PaymentCreateRequest
}

type NotifyArgs struct {
	Params map[string]string
}

type providerRPC struct {
	client *rpc.Client
}

func (p *providerRPC) Key() string {
	var resp string
	_ = p.client.Call("Plugin.Key", EmptyArgs{}, &resp)
	return resp
}

func (p *providerRPC) Name() string {
	var resp string
	_ = p.client.Call("Plugin.Name", EmptyArgs{}, &resp)
	return resp
}

func (p *providerRPC) SchemaJSON() string {
	var resp string
	_ = p.client.Call("Plugin.SchemaJSON", EmptyArgs{}, &resp)
	return resp
}

func (p *providerRPC) SetConfig(configJSON string) error {
	var resp bool
	return p.client.Call("Plugin.SetConfig", ConfigArgs{ConfigJSON: configJSON}, &resp)
}

func (p *providerRPC) CreatePayment(req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	var resp appshared.PaymentCreateResult
	err := p.client.Call("Plugin.CreatePayment", CreateArgs{Request: req}, &resp)
	return resp, err
}

func (p *providerRPC) VerifyNotify(params map[string]string) (appshared.PaymentNotifyResult, error) {
	var resp appshared.PaymentNotifyResult
	err := p.client.Call("Plugin.VerifyNotify", NotifyArgs{Params: params}, &resp)
	return resp, err
}

type providerRPCServer struct {
	Impl Provider
}

func (p *providerRPCServer) Key(_ EmptyArgs, resp *string) error {
	*resp = p.Impl.Key()
	return nil
}

func (p *providerRPCServer) Name(_ EmptyArgs, resp *string) error {
	*resp = p.Impl.Name()
	return nil
}

func (p *providerRPCServer) SchemaJSON(_ EmptyArgs, resp *string) error {
	*resp = p.Impl.SchemaJSON()
	return nil
}

func (p *providerRPCServer) SetConfig(args ConfigArgs, resp *bool) error {
	*resp = true
	return p.Impl.SetConfig(args.ConfigJSON)
}

func (p *providerRPCServer) CreatePayment(args CreateArgs, resp *appshared.PaymentCreateResult) error {
	result, err := p.Impl.CreatePayment(args.Request)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}

func (p *providerRPCServer) VerifyNotify(args NotifyArgs, resp *appshared.PaymentNotifyResult) error {
	result, err := p.Impl.VerifyNotify(args.Params)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}
