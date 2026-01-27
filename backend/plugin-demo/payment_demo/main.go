package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/go-plugin"

	paymentplugin "xiaoheiplay/internal/adapter/payment/plugin"
	"xiaoheiplay/internal/usecase"
)

type demoConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
}

type DemoPaymentProvider struct {
	cfg demoConfig
}

func (p *DemoPaymentProvider) Key() string {
	return "demo_pay"
}

func (p *DemoPaymentProvider) Name() string {
	return "Demo Payment"
}

func (p *DemoPaymentProvider) SchemaJSON() string {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"base_url": map[string]any{"type": "string", "title": "Base URL"},
			"api_key":  map[string]any{"type": "string", "title": "API Key"},
			"note":     map[string]any{"type": "string", "title": "Note"},
		},
	}
	raw, _ := json.Marshal(schema)
	return string(raw)
}

func (p *DemoPaymentProvider) SetConfig(configJSON string) error {
	if configJSON == "" {
		p.cfg = demoConfig{}
		return nil
	}
	return json.Unmarshal([]byte(configJSON), &p.cfg)
}

func (p *DemoPaymentProvider) CreatePayment(req usecase.PaymentCreateRequest) (usecase.PaymentCreateResult, error) {
	tradeNo := fmt.Sprintf("demo-%d-%d", req.OrderID, req.UserID)
	return usecase.PaymentCreateResult{
		TradeNo: tradeNo,
		PayURL:  fmt.Sprintf("%s/pay?trade_no=%s", p.cfg.BaseURL, tradeNo),
		Extra: map[string]string{
			"note": p.cfg.Note,
		},
	}, nil
}

func (p *DemoPaymentProvider) VerifyNotify(params map[string]string) (usecase.PaymentNotifyResult, error) {
	tradeNo := params["trade_no"]
	paid := params["status"] == "paid"
	return usecase.PaymentNotifyResult{
		TradeNo: tradeNo,
		Paid:    paid,
		Amount:  0,
		Raw:     params,
	}, nil
}

func main() {
	log.SetFlags(0)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: paymentplugin.Handshake,
		Plugins: map[string]plugin.Plugin{
			paymentplugin.ProviderPluginName: &paymentplugin.ProviderPlugin{Impl: &DemoPaymentProvider{}},
		},
	})
}
