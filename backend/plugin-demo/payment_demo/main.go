package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/go-plugin"

	paymentplugin "xiaoheiplay/internal/adapter/plugins/payment"
	"xiaoheiplay/internal/app/shared"
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

func (p *DemoPaymentProvider) CreatePayment(req shared.PaymentCreateRequest) (shared.PaymentCreateResult, error) {
	tradeNo := fmt.Sprintf("demo-%d-%d", req.OrderID, req.UserID)
	return shared.PaymentCreateResult{
		TradeNo: tradeNo,
		PayURL:  fmt.Sprintf("%s/pay?trade_no=%s", p.cfg.BaseURL, tradeNo),
		Extra: map[string]string{
			"note": p.cfg.Note,
		},
	}, nil
}

// VerifyNotifyV2 implements the versioned net/rpc notify contract: it signs
// against the raw query/body so a callback with JSON/multipart signature
// material is verifiable.
func (p *DemoPaymentProvider) VerifyNotifyV2(req shared.RawHTTPRequest) (shared.PaymentNotifyResult, error) {
	params := rawToParams(req)
	tradeNo := params["trade_no"]
	paid := params["status"] == "paid"
	return shared.PaymentNotifyResult{
		TradeNo: tradeNo,
		Paid:    paid,
		Amount:  0,
		Raw:     params,
	}, nil
}

// VerifyNotify is the retired v1 callback. The host no longer calls it; it
// exists so a legacy host binary can still load the plugin, but new callbacks go
// through VerifyNotifyV2.
func (p *DemoPaymentProvider) VerifyNotify(params map[string]string) (shared.PaymentNotifyResult, error) {
	tradeNo := params["trade_no"]
	paid := params["status"] == "paid"
	return shared.PaymentNotifyResult{
		TradeNo: tradeNo,
		Paid:    paid,
		Amount:  0,
		Raw:     params,
	}, nil
}

// rawToParams is demo-provider compatibility logic. The host delivers the full
// RawHTTPRequest unchanged; each provider decides how its gateway defines
// parameters and signature precedence. This demo keeps headers over query over
// form body (the first non-empty value wins).
func rawToParams(req shared.RawHTTPRequest) map[string]string {
	out := map[string]string{}
	for k, vs := range req.Headers {
		if len(vs) > 0 {
			out[strings.ToLower(k)] = vs[0]
		}
	}
	if req.RawQuery != "" {
		if q, err := url.ParseQuery(req.RawQuery); err == nil {
			for k, v := range q {
				if len(v) > 0 && out[k] == "" {
					out[k] = v[0]
				}
			}
		}
	}
	if len(req.Body) == 0 {
		return out
	}
	ct := ""
	if values := req.Headers["Content-Type"]; len(values) > 0 {
		ct = strings.ToLower(strings.TrimSpace(values[0]))
	}
	// Only form-urlencoded bodies are query-syntax; multipart bodies are not
	// (url.ParseQuery would produce garbage) and JSON bodies must not be
	// heuristically treated as form just because they contain "=".
	isForm := strings.Contains(ct, "application/x-www-form-urlencoded")
	if !isForm && ct == "" {
		trimmed := strings.TrimSpace(string(req.Body))
		if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") && strings.Contains(string(req.Body), "=") {
			isForm = true
		}
	}
	if isForm {
		if q, err := url.ParseQuery(string(req.Body)); err == nil {
			for k, v := range q {
				if len(v) > 0 && out[k] == "" {
					out[k] = v[0]
				}
			}
		}
	}
	return out
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
