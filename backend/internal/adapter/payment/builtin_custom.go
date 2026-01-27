package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"xiaoheiplay/internal/usecase"
)

const customSchemaJSON = `{
  "title": "Custom Payment",
  "type": "object",
  "properties": {
    "pay_url": { "type": "string", "title": "Payment URL" },
    "instructions": { "type": "string", "title": "Payment Instructions" }
  }
}`

type customConfig struct {
	PayURL       string `json:"pay_url"`
	Instructions string `json:"instructions"`
}

type customProvider struct {
	config customConfig
}

func newCustomProvider() *customProvider {
	return &customProvider{}
}

func (p *customProvider) Key() string {
	return "custom"
}

func (p *customProvider) Name() string {
	return "Custom"
}

func (p *customProvider) SchemaJSON() string {
	return customSchemaJSON
}

func (p *customProvider) SetConfig(configJSON string) error {
	if configJSON == "" {
		p.config = customConfig{}
		return nil
	}
	return json.Unmarshal([]byte(configJSON), &p.config)
}

func (p *customProvider) CreatePayment(ctx context.Context, req usecase.PaymentCreateRequest) (usecase.PaymentCreateResult, error) {
	if p.config.PayURL == "" {
		return usecase.PaymentCreateResult{}, errors.New("custom pay_url not configured")
	}
	tradeNo := fmt.Sprintf("CUSTOM-%d-%d", req.OrderID, time.Now().Unix())
	return usecase.PaymentCreateResult{
		TradeNo: tradeNo,
		PayURL:  p.config.PayURL,
		Extra: map[string]string{
			"instructions": p.config.Instructions,
		},
	}, nil
}

func (p *customProvider) VerifyNotify(ctx context.Context, params map[string]string) (usecase.PaymentNotifyResult, error) {
	return usecase.PaymentNotifyResult{}, errors.New("custom payment does not support notify")
}
