package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/pkg/paymentstatus"
	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	AppID         string `json:"app_id"`
	PrivateKey    string `json:"app_private_key"`
	AliPublicKey  string `json:"alipay_public_key"`
	SellerID      string `json:"seller_id"`
	IsProd        bool   `json:"is_prod"`
	DefaultNotify string `json:"default_notify_url"`
	DefaultReturn string `json:"default_return_url"`
	TimeoutSec    int    `json:"timeout_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer

	cfg        config
	instanceID string
	client     *alipay.Client
	updatedAt  time.Time
}

func normalizeKeyPEMOrBase64(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Keep PEM headers if provided; gopay will format/parse.
	if strings.Contains(s, "BEGIN") {
		return strings.ReplaceAll(s, "\r", "")
	}
	// Base64 body only.
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func parseRSAPublicKeyPEMOrBase64(raw string) (*rsa.PublicKey, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("empty public key")
	}
	if strings.Contains(s, "BEGIN") {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, fmt.Errorf("invalid public key pem")
		}
		pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		pub, ok := pubAny.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not rsa")
		}
		return pub, nil
	}
	compact := strings.ReplaceAll(s, "\r", "")
	compact = strings.ReplaceAll(compact, "\n", "")
	compact = strings.ReplaceAll(compact, "\t", "")
	compact = strings.ReplaceAll(compact, " ", "")
	der, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, fmt.Errorf("invalid public key base64")
	}
	pubAny, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not rsa")
	}
	return pub, nil
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "alipay_open",
		Name:        "Alipay Open Platform",
		Version:     "1.0.0",
		Description: "Alipay trade.* integration (RSA2).",
		Payment:     &pluginv1.PaymentCapability{Methods: []string{"alipay_wap", "alipay_page"}},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Alipay Open",
  "type": "object",
  "properties": {
    "app_id": { "type": "string", "title": "App ID" },
    "app_private_key": { "type": "string", "title": "App Private Key (PKCS1 PEM)", "format": "password" },
    "alipay_public_key": { "type": "string", "title": "Alipay Public Key (PEM)", "format": "password" },
    "seller_id": { "type": "string", "title": "Seller ID (optional)" },
    "is_prod": { "type": "boolean", "title": "Production", "default": true },
    "default_notify_url": { "type": "string", "title": "Default Notify URL (optional)" },
    "default_return_url": { "type": "string", "title": "Default Return URL (optional)" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["app_id","app_private_key","alipay_public_key"]
}`,
		UiSchema: `{
  "app_private_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" },
  "alipay_public_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.PrivateKey) == "" || strings.TrimSpace(cfg.AliPublicKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "app_id/app_private_key/alipay_public_key required"}, nil
	}
	// Parse public key to provide an early, actionable error.
	if _, err := parseRSAPublicKeyPEMOrBase64(cfg.AliPublicKey); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid alipay_public_key: " + sanitizeErr(err)}, nil
	}
	// Parse private key via gopay (supports PKCS1/PKCS8).
	if _, err := alipay.NewClient(cfg.AppID, normalizeKeyPEMOrBase64(cfg.PrivateKey), cfg.IsProd); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid app_private_key: " + sanitizeErr(err)}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	if req.GetConfigJson() == "" {
		return &pluginv1.InitResponse{Ok: false, Error: "missing config"}, nil
	}
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "invalid config json"}, nil
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 10
	}
	cfg.PrivateKey = normalizeKeyPEMOrBase64(cfg.PrivateKey)
	cfg.AliPublicKey = normalizeKeyPEMOrBase64(cfg.AliPublicKey)
	if strings.TrimSpace(cfg.AppID) == "" || strings.TrimSpace(cfg.PrivateKey) == "" || strings.TrimSpace(cfg.AliPublicKey) == "" {
		return &pluginv1.InitResponse{Ok: false, Error: "app_id/app_private_key/alipay_public_key required"}, nil
	}
	// Make sure public key is parseable before enabling.
	if _, err := parseRSAPublicKeyPEMOrBase64(cfg.AliPublicKey); err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "invalid alipay_public_key: " + sanitizeErr(err)}, nil
	}
	c, err := alipay.NewClient(cfg.AppID, cfg.PrivateKey, cfg.IsProd)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "alipay client init failed: " + err.Error()}, nil
	}
	c.SetCharset("utf-8").SetSignType(alipay.RSA2)
	s.cfg = cfg
	s.instanceID = req.GetInstanceId()
	s.client = c
	s.updatedAt = time.Now()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(ctx context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	ir, err := s.Init(ctx, &pluginv1.InitRequest{InstanceId: s.instanceID, ConfigJson: req.GetConfigJson()})
	if err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: err.Error()}, nil
	}
	if ir != nil && !ir.Ok {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: ir.Error}, nil
	}
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(ctx context.Context, _ *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	_ = ctx
	msg := "ok"
	if s.client == nil {
		msg = "client not initialized"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
		Message:    msg,
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

type payServer struct {
	pluginv1.UnimplementedPaymentServiceServer
	core *coreServer
}

func (p *payServer) ListMethods(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListMethodsResponse, error) {
	_ = ctx
	return &pluginv1.ListMethodsResponse{Methods: []string{"alipay_wap", "alipay_page"}}, nil
}

func (p *payServer) CreatePayment(ctx context.Context, req *pluginv1.CreatePaymentRpcRequest) (*pluginv1.PaymentCreateResponse, error) {
	if p.core == nil || p.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	in := req.GetRequest()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	orderNo := strings.TrimSpace(in.GetOrderNo())
	if orderNo == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no required")
	}
	total := money.FormatCents(in.GetAmount())

	notifyURL := firstNonEmpty(in.GetNotifyUrl(), p.core.cfg.DefaultNotify)
	returnURL := firstNonEmpty(in.GetReturnUrl(), p.core.cfg.DefaultReturn)

	bm := gopay.BodyMap{}
	bm.Set("subject", in.GetSubject()).
		Set("out_trade_no", orderNo).
		Set("total_amount", total)
	if notifyURL != "" {
		bm.Set("notify_url", notifyURL)
	}
	if returnURL != "" {
		bm.Set("return_url", returnURL)
	}
	if strings.TrimSpace(p.core.cfg.SellerID) != "" {
		bm.Set("seller_id", strings.TrimSpace(p.core.cfg.SellerID))
	}

	var payURL string
	var err error
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	switch method {
	case "alipay_wap":
		bm.Set("product_code", "QUICK_WAP_WAY")
		payURL, err = p.core.client.TradeWapPay(cctx, bm)
	case "alipay_page":
		bm.Set("product_code", "FAST_INSTANT_TRADE_PAY")
		payURL, err = p.core.client.TradePagePay(cctx, bm)
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "alipay create failed: "+sanitizeErr(err))
	}
	return &pluginv1.PaymentCreateResponse{
		Ok:     true,
		PayUrl: payURL,
		Extra: map[string]string{
			"amount_yuan":  total,
			"pay_kind":     "redirect",
			"instructions": "跳转到支付宝完成支付",
		},
	}, nil
}

func (p *payServer) QueryPayment(ctx context.Context, req *pluginv1.QueryPaymentRpcRequest) (*pluginv1.PaymentQueryResponse, error) {
	if p.core == nil || p.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "alipay_wap" && method != "alipay_page" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	tradeNo := strings.TrimSpace(req.GetTradeNo())
	orderNo := strings.TrimSpace(req.GetOrderNo())
	if tradeNo == "" && orderNo == "" {
		return nil, status.Error(codes.InvalidArgument, "trade_no or order_no required")
	}
	bm := gopay.BodyMap{}
	if orderNo != "" {
		bm.Set("out_trade_no", orderNo)
	} else if strings.HasPrefix(tradeNo, "ORD-") {
		bm.Set("out_trade_no", tradeNo)
	} else {
		bm.Set("trade_no", tradeNo)
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	resp, err := p.core.client.TradeQuery(cctx, bm)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "alipay query failed: "+sanitizeErr(err))
	}
	raw, _ := json.Marshal(resp)
	if resp == nil || resp.Response == nil {
		return nil, status.Error(codes.FailedPrecondition, "alipay query empty response")
	}
	ps := paymentstatus.AlipayTradeStatusToStatus(resp.Response.TradeStatus)
	amount := int64(0)
	if resp.Response.TotalAmount != "" {
		if v, err := money.ParseAmountToCents(resp.Response.TotalAmount); err == nil {
			amount = v
		}
	}
	outTradeNo := strings.TrimSpace(resp.Response.OutTradeNo)
	aliTradeNo := strings.TrimSpace(resp.Response.TradeNo)
	retTradeNo := firstNonEmpty(aliTradeNo, firstNonEmpty(tradeNo, outTradeNo))
	return &pluginv1.PaymentQueryResponse{
		Ok:      true,
		Status:  ps,
		TradeNo: strings.TrimSpace(retTradeNo),
		Amount:  amount,
		RawJson: string(raw),
	}, nil
}

func (p *payServer) Refund(ctx context.Context, req *pluginv1.RefundRpcRequest) (*pluginv1.RefundResponse, error) {
	if p.core == nil || p.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "alipay_wap" && method != "alipay_page" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	tradeNo := strings.TrimSpace(req.GetTradeNo())
	refundNo := strings.TrimSpace(req.GetRefundNo())
	if tradeNo == "" || refundNo == "" || req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "trade_no/refund_no/amount required")
	}
	bm := gopay.BodyMap{}
	if strings.HasPrefix(tradeNo, "ORD-") {
		bm.Set("out_trade_no", tradeNo)
	} else {
		bm.Set("trade_no", tradeNo)
	}
	bm.Set("out_request_no", refundNo).
		Set("refund_amount", money.FormatCents(req.GetAmount()))
	if strings.TrimSpace(req.GetReason()) != "" {
		bm.Set("refund_reason", strings.TrimSpace(req.GetReason()))
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	resp, err := p.core.client.TradeRefund(cctx, bm)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "alipay refund failed: "+sanitizeErr(err))
	}
	raw, _ := json.Marshal(resp)
	return &pluginv1.RefundResponse{
		Ok:       true,
		RefundNo: refundNo,
		Status:   pluginv1.PaymentStatus_PAYMENT_STATUS_REFUNDING,
		RawJson:  string(raw),
	}, nil
}

func (p *payServer) VerifyNotify(ctx context.Context, req *pluginv1.VerifyNotifyRequest) (*pluginv1.NotifyVerifyResult, error) {
	if p.core == nil || strings.TrimSpace(p.core.cfg.AliPublicKey) == "" {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "alipay_wap" && method != "alipay_page" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	raw := req.GetRaw()
	if raw == nil {
		return nil, status.Error(codes.InvalidArgument, "missing raw request")
	}
	params := rawToParams(raw)
	if len(params) == 0 {
		return nil, status.Error(codes.InvalidArgument, "missing notify params")
	}
	bm := gopay.BodyMap{}
	for k, v := range params {
		bm.Set(k, v)
	}
	ok, err := alipay.VerifySign(p.core.cfg.AliPublicKey, bm)
	if err != nil || !ok {
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "alipay verify failed: "+sanitizeErr(err))
		}
		return nil, status.Error(codes.InvalidArgument, "alipay invalid sign")
	}
	orderNo := strings.TrimSpace(params["out_trade_no"])
	if orderNo == "" {
		return nil, status.Error(codes.InvalidArgument, "missing out_trade_no")
	}
	tradeNo := strings.TrimSpace(params["trade_no"])
	if tradeNo == "" {
		tradeNo = orderNo
	}
	if appID := strings.TrimSpace(p.core.cfg.AppID); appID != "" {
		if got := strings.TrimSpace(params["app_id"]); got != "" && got != appID {
			return nil, status.Error(codes.InvalidArgument, "app_id mismatch")
		}
	}
	if sellerID := strings.TrimSpace(p.core.cfg.SellerID); sellerID != "" {
		if got := strings.TrimSpace(params["seller_id"]); got != "" && got != sellerID {
			return nil, status.Error(codes.InvalidArgument, "seller_id mismatch")
		}
	}
	ps := paymentstatus.AlipayTradeStatusToStatus(params["trade_status"])
	amount := int64(0)
	if v, err := money.ParseAmountToCents(params["total_amount"]); err == nil {
		amount = v
	}
	var paidAtUnix int64
	if v := strings.TrimSpace(params["gmt_payment"]); v != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local); err == nil {
			paidAtUnix = t.Unix()
		}
	}
	payer := strings.TrimSpace(params["buyer_logon_id"])
	if payer == "" {
		payer = strings.TrimSpace(params["buyer_id"])
	}
	rawJSON, _ := json.Marshal(params)

	return &pluginv1.NotifyVerifyResult{
		Ok:         true,
		OrderNo:    orderNo,
		TradeNo:    tradeNo,
		Amount:     amount,
		Status:     ps,
		PaidAtUnix: paidAtUnix,
		Payer:      payer,
		AckBody:    "success",
		RawJson:    string(rawJSON),
	}, nil
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 400 {
		s = s[:400]
	}
	return s
}

func rawToParams(req *pluginv1.RawHttpRequest) map[string]string {
	out := map[string]string{}
	if req == nil {
		return out
	}
	if req.GetRawQuery() != "" {
		if q, err := url.ParseQuery(req.GetRawQuery()); err == nil {
			for k, v := range q {
				if len(v) > 0 {
					out[k] = v[0]
				}
			}
		}
	}
	if len(req.GetBody()) > 0 {
		if q, err := url.ParseQuery(string(req.GetBody())); err == nil {
			for k, v := range q {
				if len(v) == 0 {
					continue
				}
				if _, ok := out[k]; !ok {
					out[k] = v[0]
				}
			}
		}
	}
	return out
}

func main() {
	core := &coreServer{}
	pay := &payServer{core: core}

	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:    &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyPayment: &pluginsdk.PaymentGRPCPlugin{Impl: pay},
	})
}
