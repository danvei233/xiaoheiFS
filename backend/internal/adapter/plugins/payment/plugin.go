package paymentplugin

import (
	"fmt"
	"net/rpc"
	"strings"
	"time"

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

// paymentPluginRPCTimeout bounds each net/rpc call to a legacy payment plugin.
// net/rpc has no per-call deadline of its own; without this a hung plugin would
// block the payment/create/notify request forever.
var paymentPluginRPCTimeout = 60 * time.Second

// isUnknownRPCMethod reports the net/rpc "method not found" error, which means
// the legacy plugin binary predates the versioned VerifyNotifyV2 contract.
func isUnknownRPCMethod(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "plugin.verifynotifyv2") {
		return false
	}
	return strings.Contains(msg, "can't find method") ||
		strings.Contains(msg, "can't find service") ||
		strings.Contains(msg, "unknown method")
}

type Provider interface {
	Key() string
	Name() string
	SchemaJSON() string
	SetConfig(configJSON string) error
	CreatePayment(req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error)
	// VerifyNotifyV2 is the versioned notify contract: the plugin receives the
	// original HTTP callback (method/path/raw query/headers/body) so signature
	// verification can use the raw signing materials instead of a lossy
	// params map. The host refuses to fall back to VerifyNotifyV1.
	VerifyNotifyV2(req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error)
	// Deprecated: VerifyNotify(params) is the retired v1 callback. New plugins
	// must implement VerifyNotifyV2; the host never calls this for a plugin
	// that supports V2, and refuses to downgrade a callback to params.
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

type NotifyV2Args struct {
	Request appshared.RawHTTPRequest
}

type providerRPC struct {
	client *rpc.Client
}

func (p *providerRPC) Key() string {
	var resp string
	call := p.client.Go("Plugin.Key", EmptyArgs{}, &resp, nil)
	select {
	case <-call.Done:
		return resp
	case <-time.After(paymentPluginRPCTimeout):
		return ""
	}
}

func (p *providerRPC) Name() string {
	var resp string
	call := p.client.Go("Plugin.Name", EmptyArgs{}, &resp, nil)
	select {
	case <-call.Done:
		return resp
	case <-time.After(paymentPluginRPCTimeout):
		return ""
	}
}

func (p *providerRPC) SchemaJSON() string {
	var resp string
	call := p.client.Go("Plugin.SchemaJSON", EmptyArgs{}, &resp, nil)
	select {
	case <-call.Done:
		return resp
	case <-time.After(paymentPluginRPCTimeout):
		return ""
	}
}

func (p *providerRPC) SetConfig(configJSON string) error {
	var resp bool
	call := p.client.Go("Plugin.SetConfig", ConfigArgs{ConfigJSON: configJSON}, &resp, nil)
	select {
	case <-call.Done:
		return call.Error
	case <-time.After(paymentPluginRPCTimeout):
		return fmt.Errorf("payment plugin SetConfig timed out after %s", paymentPluginRPCTimeout)
	}
}

func (p *providerRPC) CreatePayment(req appshared.PaymentCreateRequest) (appshared.PaymentCreateResult, error) {
	var resp appshared.PaymentCreateResult
	call := p.client.Go("Plugin.CreatePayment", CreateArgs{Request: req}, &resp, nil)
	select {
	case <-call.Done:
		return resp, call.Error
	case <-time.After(paymentPluginRPCTimeout):
		return appshared.PaymentCreateResult{}, fmt.Errorf("payment plugin CreatePayment timed out after %s", paymentPluginRPCTimeout)
	}
}

func (p *providerRPC) VerifyNotify(params map[string]string) (appshared.PaymentNotifyResult, error) {
	var resp appshared.PaymentNotifyResult
	call := p.client.Go("Plugin.VerifyNotify", NotifyArgs{Params: params}, &resp, nil)
	select {
	case <-call.Done:
		return resp, call.Error
	case <-time.After(paymentPluginRPCTimeout):
		return appshared.PaymentNotifyResult{}, fmt.Errorf("payment plugin VerifyNotify timed out after %s", paymentPluginRPCTimeout)
	}
}

// VerifyNotifyV2 carries the full original HTTP callback to the plugin. An
// "unknown method" response means the legacy plugin binary predates V2; the host
// treats that as fail-closed (never downgrades to the lossy params map).
func (p *providerRPC) VerifyNotifyV2(req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
	var resp appshared.PaymentNotifyResult
	call := p.client.Go("Plugin.VerifyNotifyV2", NotifyV2Args{Request: req}, &resp, nil)
	select {
	case <-call.Done:
		if call.Error != nil {
			if isUnknownRPCMethod(call.Error) {
				return appshared.PaymentNotifyResult{}, fmt.Errorf("payment plugin does not support the versioned VerifyNotifyV2 contract; refusing to downgrade to params")
			}
			return appshared.PaymentNotifyResult{}, call.Error
		}
		return resp, nil
	case <-time.After(paymentPluginRPCTimeout):
		return appshared.PaymentNotifyResult{}, fmt.Errorf("payment plugin VerifyNotifyV2 timed out after %s", paymentPluginRPCTimeout)
	}
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

func (p *providerRPCServer) VerifyNotifyV2(args NotifyV2Args, resp *appshared.PaymentNotifyResult) error {
	result, err := p.Impl.VerifyNotifyV2(args.Request)
	if err != nil {
		return err
	}
	*resp = result
	return nil
}
