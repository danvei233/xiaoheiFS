package payment

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/usecase"
)

const yipaySchemaJSON = `{
  "title": "YiPay",
  "type": "object",
  "properties": {
    "base_url": { "type": "string", "title": "Gateway URL" },
    "pid": { "type": "string", "title": "Merchant PID" },
    "key": { "type": "string", "title": "Merchant Key" },
    "pay_type": { "type": "string", "title": "Pay Type" },
    "notify_url": { "type": "string", "title": "Notify URL" },
    "return_url": { "type": "string", "title": "Return URL" },
    "sign_type": { "type": "string", "title": "Sign Type", "default": "MD5" }
  }
}`

type yipayConfig struct {
	BaseURL   string `json:"base_url"`
	PID       string `json:"pid"`
	Key       string `json:"key"`
	PayType   string `json:"pay_type"`
	NotifyURL string `json:"notify_url"`
	ReturnURL string `json:"return_url"`
	SignType  string `json:"sign_type"`
}

type yipayProvider struct {
	config yipayConfig
}

func newYipayProvider() *yipayProvider {
	return &yipayProvider{}
}

func (p *yipayProvider) Key() string {
	return "yipay"
}

func (p *yipayProvider) Name() string {
	return "YiPay"
}

func (p *yipayProvider) SchemaJSON() string {
	return yipaySchemaJSON
}

func (p *yipayProvider) SetConfig(configJSON string) error {
	if configJSON == "" {
		p.config = yipayConfig{}
		return nil
	}
	if err := json.Unmarshal([]byte(configJSON), &p.config); err != nil {
		return err
	}
	if p.config.SignType == "" {
		p.config.SignType = "MD5"
	}
	return nil
}

func (p *yipayProvider) CreatePayment(ctx context.Context, req usecase.PaymentCreateRequest) (usecase.PaymentCreateResult, error) {
	if p.config.BaseURL == "" || p.config.PID == "" || p.config.Key == "" {
		return usecase.PaymentCreateResult{}, errors.New("yipay config incomplete")
	}
	tradeNo := fmt.Sprintf("YI-%d-%d", req.OrderID, time.Now().Unix())
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = p.config.NotifyURL
	}
	returnURL := req.ReturnURL
	if returnURL == "" {
		returnURL = p.config.ReturnURL
	}
	params := map[string]string{
		"pid":          p.config.PID,
		"type":         p.config.PayType,
		"out_trade_no": tradeNo,
		"name":         req.Subject,
		"money":        money.FormatCents(req.Amount),
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"sign_type":    p.config.SignType,
	}
	sign := p.sign(params)
	params["sign"] = sign
	payURL := buildURL(p.config.BaseURL, params)
	return usecase.PaymentCreateResult{
		TradeNo: tradeNo,
		PayURL:  payURL,
	}, nil
}

func (p *yipayProvider) VerifyNotify(ctx context.Context, params map[string]string) (usecase.PaymentNotifyResult, error) {
	sign := params["sign"]
	if sign == "" {
		return usecase.PaymentNotifyResult{}, errors.New("missing sign")
	}
	expected := p.sign(params)
	if !strings.EqualFold(sign, expected) {
		return usecase.PaymentNotifyResult{}, errors.New("invalid sign")
	}
	status := strings.ToLower(params["trade_status"])
	if status == "" {
		status = strings.ToLower(params["status"])
	}
	paid := status == "trade_success" || status == "success"
	moneyStr := params["money"]
	amount, _ := money.ParseNumberStringToCents(moneyStr)
	return usecase.PaymentNotifyResult{
		TradeNo: params["out_trade_no"],
		Paid:    paid,
		Amount:  amount,
		Raw:     params,
	}, nil
}

func (p *yipayProvider) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(params[k])
	}
	buf.WriteString(p.config.Key)
	sum := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(sum[:])
}

func buildURL(base string, params map[string]string) string {
	query := url.Values{}
	for k, v := range params {
		if v == "" {
			continue
		}
		query.Set(k, v)
	}
	if strings.Contains(base, "?") {
		return base + "&" + query.Encode()
	}
	return base + "?" + query.Encode()
}
