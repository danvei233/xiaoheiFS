package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/pkg/paymentstatus"
	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	MchID              string `json:"mch_id"`
	MerchantSerialNo   string `json:"merchant_serial_no"`
	MerchantPrivateKey string `json:"merchant_private_key_pem"`
	APIv3Key           string `json:"api_v3_key"`
	AppID              string `json:"app_id"`
	DefaultNotifyURL   string `json:"default_notify_url"`
	DefaultReturnURL   string `json:"default_return_url"`
	TimeoutSec         int    `json:"timeout_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer

	cfg        config
	instanceID string
	client     *core.Client
	privateKey *rsa.PrivateKey
	verifier   auth.Verifier
	updatedAt  time.Time
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "wechatpay_v3",
		Name:        "WeChat Pay v3",
		Version:     "1.0.0",
		Description: "WeChat Pay API v3 (wechatpay-go). Supports native and JSAPI.",
		Payment:     &pluginv1.PaymentCapability{Methods: []string{"wechat_native", "wechat_jsapi"}},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "WeChat Pay v3",
  "type": "object",
  "properties": {
    "mch_id": { "type": "string", "title": "Merchant ID (mchid)" },
    "merchant_serial_no": { "type": "string", "title": "Merchant Certificate Serial No" },
    "merchant_private_key_pem": { "type": "string", "title": "Merchant Private Key (PEM)", "format": "password" },
    "api_v3_key": { "type": "string", "title": "API v3 Key", "format": "password" },
    "app_id": { "type": "string", "title": "AppID" },
    "default_notify_url": { "type": "string", "title": "Default Notify URL (optional)" },
    "default_return_url": { "type": "string", "title": "Default Return URL (optional)" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["mch_id","merchant_serial_no","merchant_private_key_pem","api_v3_key","app_id"]
}`,
		UiSchema: `{
  "merchant_private_key_pem": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" },
  "api_v3_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.MchID) == "" || strings.TrimSpace(cfg.MerchantSerialNo) == "" || strings.TrimSpace(cfg.MerchantPrivateKey) == "" || strings.TrimSpace(cfg.APIv3Key) == "" || strings.TrimSpace(cfg.AppID) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "mch_id/merchant_serial_no/merchant_private_key_pem/api_v3_key/app_id required"}, nil
	}
	if len(strings.TrimSpace(cfg.APIv3Key)) != 32 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "api_v3_key must be 32 chars"}, nil
	}
	if _, err := parseRSAPrivateKeyPEM(cfg.MerchantPrivateKey); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid merchant_private_key_pem"}, nil
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
	priv, err := parseRSAPrivateKeyPEM(cfg.MerchantPrivateKey)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "invalid merchant_private_key_pem"}, nil
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()
	client, err := core.NewClient(cctx,
		option.WithWechatPayAutoAuthCipher(cfg.MchID, cfg.MerchantSerialNo, priv, cfg.APIv3Key),
	)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "wechatpay client init failed: " + err.Error()}, nil
	}

	certVisitor := downloader.MgrInstance().GetCertificateVisitor(cfg.MchID)
	verifier := verifiers.NewSHA256WithRSAVerifier(certVisitor)

	s.cfg = cfg
	s.instanceID = req.GetInstanceId()
	s.client = client
	s.privateKey = priv
	s.verifier = verifier
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

func (s *coreServer) Health(ctx context.Context, req *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	_ = ctx
	msg := "ok"
	if strings.TrimSpace(req.GetInstanceId()) == "" || strings.TrimSpace(s.instanceID) == "" {
		msg = "not initialized"
	}
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
	return &pluginv1.ListMethodsResponse{Methods: []string{"wechat_native", "wechat_jsapi"}}, nil
}

func (p *payServer) CreatePayment(ctx context.Context, req *pluginv1.CreatePaymentRpcRequest) (*pluginv1.PaymentCreateResponse, error) {
	if p.core == nil || p.core.client == nil || p.core.privateKey == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	in := req.GetRequest()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	outTradeNo := strings.TrimSpace(in.GetOrderNo())
	if outTradeNo == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no required")
	}
	notifyURL := firstNonEmpty(in.GetNotifyUrl(), p.core.cfg.DefaultNotifyURL)
	if notifyURL == "" {
		return nil, status.Error(codes.InvalidArgument, "notify_url required")
	}
	returnURL := firstNonEmpty(in.GetReturnUrl(), p.core.cfg.DefaultReturnURL)
	amountYuan := money.FormatCents(in.GetAmount())

	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()

	switch method {
	case "wechat_native":
		svc := native.NativeApiService{Client: p.core.client}
		resp, _, err := svc.Prepay(cctx, native.PrepayRequest{
			Appid:       core.String(p.core.cfg.AppID),
			Mchid:       core.String(p.core.cfg.MchID),
			Description: core.String(in.GetSubject()),
			OutTradeNo:  core.String(outTradeNo),
			Attach:      core.String(outTradeNo),
			NotifyUrl:   core.String(notifyURL),
			Amount: &native.Amount{
				Total:    core.Int64(in.GetAmount()),
				Currency: core.String(firstNonEmpty(in.GetCurrency(), "CNY")),
			},
		})
		if err != nil {
			return nil, status.Error(codes.Unavailable, "wechat native prepay failed: "+sanitizeErr(err))
		}
		codeURL := ""
		if resp != nil && resp.CodeUrl != nil {
			codeURL = *resp.CodeUrl
		}
		if codeURL == "" {
			return nil, status.Error(codes.FailedPrecondition, "wechat native prepay missing code_url")
		}
		return &pluginv1.PaymentCreateResponse{
			Ok:      true,
			TradeNo: "",
			PayUrl:  codeURL,
			Extra: map[string]string{
				"amount_yuan":  amountYuan,
				"code_url":     codeURL,
				"pay_kind":     "qr",
				"instructions": "请使用微信扫码支付",
				"return_url":   returnURL,
			},
		}, nil
	case "wechat_jsapi":
		openID := ""
		if in.GetExtra() != nil {
			openID = strings.TrimSpace(in.GetExtra()["openid"])
		}
		if openID == "" {
			return nil, status.Error(codes.InvalidArgument, "extra.openid required for wechat_jsapi")
		}
		svc := jsapi.JsapiApiService{Client: p.core.client}
		resp, _, err := svc.Prepay(cctx, jsapi.PrepayRequest{
			Appid:       core.String(p.core.cfg.AppID),
			Mchid:       core.String(p.core.cfg.MchID),
			Description: core.String(in.GetSubject()),
			OutTradeNo:  core.String(outTradeNo),
			Attach:      core.String(outTradeNo),
			NotifyUrl:   core.String(notifyURL),
			Amount: &jsapi.Amount{
				Total:    core.Int64(in.GetAmount()),
				Currency: core.String(firstNonEmpty(in.GetCurrency(), "CNY")),
			},
			Payer: &jsapi.Payer{Openid: core.String(openID)},
		})
		if err != nil {
			return nil, status.Error(codes.Unavailable, "wechat jsapi prepay failed: "+sanitizeErr(err))
		}
		prepayID := ""
		if resp != nil && resp.PrepayId != nil {
			prepayID = *resp.PrepayId
		}
		if prepayID == "" {
			return nil, status.Error(codes.FailedPrecondition, "wechat jsapi prepay missing prepay_id")
		}

		nonceStr := randomNonce(32)
		timeStamp := fmt.Sprintf("%d", time.Now().Unix())
		pkg := "prepay_id=" + prepayID
		paySign, err := signWechatJSAPI(p.core.privateKey, p.core.cfg.AppID, timeStamp, nonceStr, pkg)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, "sign jsapi failed: "+sanitizeErr(err))
		}

		jsapiParams := map[string]string{
			"appId":     p.core.cfg.AppID,
			"timeStamp": timeStamp,
			"nonceStr":  nonceStr,
			"package":   pkg,
			"signType":  "RSA",
			"paySign":   paySign,
		}
		jsapiJSON, _ := json.Marshal(jsapiParams)
		return &pluginv1.PaymentCreateResponse{
			Ok:      true,
			TradeNo: "",
			PayUrl:  "",
			Extra: map[string]string{
				"amount_yuan":       amountYuan,
				"pay_kind":          "jsapi",
				"jsapi_params_json": string(jsapiJSON),
				"instructions":      "请在微信内发起 JSAPI 支付",
				"return_url":        returnURL,
			},
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
}

func (p *payServer) QueryPayment(ctx context.Context, req *pluginv1.QueryPaymentRpcRequest) (*pluginv1.PaymentQueryResponse, error) {
	if p.core == nil || p.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "wechat_native" && method != "wechat_jsapi" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	outTradeNo := strings.TrimSpace(req.GetOrderNo())
	if outTradeNo == "" {
		outTradeNo = strings.TrimSpace(req.GetTradeNo())
	}
	if outTradeNo == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no required")
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	var resp *payments.Transaction
	var err error
	switch method {
	case "wechat_native":
		svc := native.NativeApiService{Client: p.core.client}
		resp, _, err = svc.QueryOrderByOutTradeNo(cctx, native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(outTradeNo),
			Mchid:      core.String(p.core.cfg.MchID),
		})
	case "wechat_jsapi":
		svc := jsapi.JsapiApiService{Client: p.core.client}
		resp, _, err = svc.QueryOrderByOutTradeNo(cctx, jsapi.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(outTradeNo),
			Mchid:      core.String(p.core.cfg.MchID),
		})
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "wechat query failed: "+sanitizeErr(err))
	}
	ps, amount, rawJSON := mapWechatTradeState(resp)
	return &pluginv1.PaymentQueryResponse{
		Ok:      true,
		Status:  ps,
		TradeNo: "",
		Amount:  amount,
		RawJson: rawJSON,
	}, nil
}

func (p *payServer) Refund(ctx context.Context, req *pluginv1.RefundRpcRequest) (*pluginv1.RefundResponse, error) {
	if p.core == nil || p.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "wechat_native" && method != "wechat_jsapi" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	tradeNo := strings.TrimSpace(req.GetTradeNo())
	refundNo := strings.TrimSpace(req.GetRefundNo())
	if tradeNo == "" || refundNo == "" || req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "trade_no/refund_no/amount required")
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	svc := refunddomestic.RefundsApiService{Client: p.core.client}
	resp, _, err := svc.Create(cctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(tradeNo),
		OutRefundNo: core.String(refundNo),
		Reason:      core.String(strings.TrimSpace(req.GetReason())),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(req.GetAmount()),
			Total:    core.Int64(req.GetAmount()),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		return nil, status.Error(codes.Unavailable, "wechat refund failed: "+sanitizeErr(err))
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
	if p.core == nil || p.core.verifier == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	method := strings.TrimSpace(req.GetMethod())
	if method != "wechat_native" && method != "wechat_jsapi" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	raw := req.GetRaw()
	if raw == nil {
		return nil, status.Error(codes.InvalidArgument, "missing raw request")
	}
	httpReq := rawToHTTPRequest(raw)
	nh, err := notify.NewRSANotifyHandler(p.core.cfg.APIv3Key, p.core.verifier)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "notify handler init failed: "+sanitizeErr(err))
	}

	var transaction payments.Transaction
	cctx, cancel := context.WithTimeout(ctx, time.Duration(p.core.cfg.TimeoutSec)*time.Second)
	defer cancel()
	if _, err := nh.ParseNotifyRequest(cctx, httpReq, &transaction); err != nil {
		return nil, status.Error(codes.InvalidArgument, "wechat notify verify failed: "+sanitizeErr(err))
	}
	ps := pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	if transaction.TradeState != nil {
		ps = paymentstatus.WeChatTradeStateToStatus(*transaction.TradeState)
	}
	orderNo := ""
	if transaction.OutTradeNo != nil {
		orderNo = strings.TrimSpace(*transaction.OutTradeNo)
	}
	if orderNo == "" && transaction.Attach != nil && strings.TrimSpace(*transaction.Attach) != "" {
		orderNo = strings.TrimSpace(*transaction.Attach)
	}
	tradeNo := ""
	if transaction.TransactionId != nil {
		tradeNo = strings.TrimSpace(*transaction.TransactionId)
	}
	amount := int64(0)
	if transaction.Amount != nil && transaction.Amount.Total != nil {
		amount = *transaction.Amount.Total
	}
	rawJSON, _ := json.Marshal(transaction)
	return &pluginv1.NotifyVerifyResult{
		Ok:      true,
		OrderNo: orderNo,
		TradeNo: tradeNo,
		Amount:  amount,
		Status:  ps,
		AckBody: `{"code":"SUCCESS","message":"SUCCESS"}`,
		RawJson: string(rawJSON),
	}, nil
}

func rawToHTTPRequest(raw *pluginv1.RawHttpRequest) *http.Request {
	req := &http.Request{
		Method: raw.GetMethod(),
		URL:    &url.URL{Scheme: "http", Host: "localhost", Path: "/"},
		Body:   io.NopCloser(bytes.NewReader(raw.GetBody())),
		Header: http.Header{},
	}
	for k, v := range raw.GetHeaders() {
		if v == nil {
			continue
		}
		for _, vv := range v.Values {
			req.Header.Add(k, vv)
		}
	}
	return req
}

func parseRSAPrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	pemStr = strings.TrimSpace(pemStr)
	if pemStr == "" {
		return nil, fmt.Errorf("empty private key")
	}
	tryParse := func(s string) (*rsa.PrivateKey, error) {
		block, _ := pem.Decode([]byte(s))
		if block == nil {
			return nil, fmt.Errorf("invalid pem")
		}
		if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			if rk, ok := k.(*rsa.PrivateKey); ok {
				return rk, nil
			}
		}
		if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return k, nil
		}
		return nil, fmt.Errorf("unsupported key type")
	}

	// First try direct PEM parsing (BEGIN/END provided).
	if strings.Contains(pemStr, "BEGIN") {
		if k, err := tryParse(pemStr); err == nil {
			return k, nil
		}
	}

	// Then accept base64 body only (no PEM headers).
	var compact string
	if strings.Contains(pemStr, "BEGIN") {
		lines := strings.Split(strings.ReplaceAll(pemStr, "\r", ""), "\n")
		var b strings.Builder
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "-----BEGIN") || strings.HasPrefix(line, "-----END") {
				continue
			}
			b.WriteString(line)
		}
		compact = b.String()
	} else {
		compact = strings.ReplaceAll(pemStr, "\r", "")
		compact = strings.ReplaceAll(compact, "\n", "")
		compact = strings.ReplaceAll(compact, "\t", "")
		compact = strings.ReplaceAll(compact, " ", "")
	}
	if compact == "" {
		return nil, fmt.Errorf("empty private key")
	}
	if _, err := base64.StdEncoding.DecodeString(compact); err != nil {
		return nil, fmt.Errorf("invalid private key pem/base64")
	}
	wrap := func(typ string) string {
		var b strings.Builder
		b.WriteString("-----BEGIN " + typ + "-----\n")
		for i := 0; i < len(compact); i += 64 {
			j := i + 64
			if j > len(compact) {
				j = len(compact)
			}
			b.WriteString(compact[i:j] + "\n")
		}
		b.WriteString("-----END " + typ + "-----\n")
		return b.String()
	}
	if k, err := tryParse(wrap("PRIVATE KEY")); err == nil {
		return k, nil
	}
	if k, err := tryParse(wrap("RSA PRIVATE KEY")); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf("invalid merchant_private_key_pem")
}

func signWechatJSAPI(priv *rsa.PrivateKey, appID, timeStamp, nonceStr, pkg string) (string, error) {
	msg := strings.Join([]string{appID, timeStamp, nonceStr, pkg, ""}, "\n")
	sum := sha256.Sum256([]byte(msg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func randomNonce(n int) string {
	const alpha = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = alpha[int(b[i])%len(alpha)]
	}
	return string(b)
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

func mapWechatTradeState(resp *payments.Transaction) (pluginv1.PaymentStatus, int64, string) {
	if resp == nil {
		return pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED, 0, ""
	}
	raw, _ := json.Marshal(resp)
	amount := int64(0)
	if resp.Amount != nil && resp.Amount.Total != nil {
		amount = *resp.Amount.Total
	}
	state := ""
	if resp.TradeState != nil {
		state = *resp.TradeState
	}
	return paymentstatus.WeChatTradeStateToStatus(state), amount, string(raw)
}

func main() {
	core := &coreServer{}
	pay := &payServer{core: core}

	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:    &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyPayment: &pluginsdk.PaymentGRPCPlugin{Impl: pay},
	})
}
