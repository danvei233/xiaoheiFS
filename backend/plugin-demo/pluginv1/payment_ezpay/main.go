package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	GatewayBaseURL string `json:"gateway_base_url"`
	SubmitURL      string `json:"submit_url"`
	SubmitPath     string `json:"submit_path"`
	QueryAPIURL    string `json:"query_api_url"`

	PID         string `json:"pid"`
	MerchantKey string `json:"merchant_key"`
	SiteName    string `json:"site_name"`

	SignType    string `json:"sign_type"`
	SignKeyMode string `json:"sign_key_mode"`
	TimeoutSec  int    `json:"timeout_sec"`
	// Same order+method will reuse pay link within this window.
	OrderExpireMinutes int `json:"order_expire_minutes"`

	// Backward-compatible legacy keys (old demo config).
	BaseURL   string `json:"base_url"`
	Key       string `json:"key"`
	QueryURL  string `json:"query_url"`
	NotifyURL string `json:"notify_url"`
	ReturnURL string `json:"return_url"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer

	cfg       config
	instance  string
	updatedAt time.Time
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "ezpay",
		Name:        "EZPay",
		Version:     "1.0.0",
		Description: "EZPay aggregate gateway (MD5). Fixed methods: alipay/wxpay/qqpay. Method switches are host-managed.",
		Payment:     &pluginv1.PaymentCapability{Methods: []string{"alipay", "wxpay", "qqpay"}},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "EZPay",
  "type": "object",
  "properties": {
    "gateway_base_url": { "type": "string", "title": "Gateway Base URL", "default": "https://www.ezfpy.cn", "description": "e.g. https://www.ezfpy.cn" },
    "submit_path": { "type": "string", "title": "Submit Path", "default": "mapi.php", "description": "Common values: submit.php (redirect) or mapi.php (API pay). Ignored when submit_url is set." },
    "submit_url": { "type": "string", "title": "Submit URL (optional override)", "default": "", "description": "If empty, {gateway_base_url}/{submit_path} is used." },
    "query_api_url": { "type": "string", "title": "Query API URL (optional)", "default": "", "description": "PHP-SDK-style: {gateway_base_url}/api.php?act=order ; Findorder: {gateway_base_url}/api/findorder" },
    "pid": { "type": "string", "title": "Merchant PID" },
    "merchant_key": { "type": "string", "title": "Merchant Key", "format": "password", "x-secret": true },
    "site_name": { "type": "string", "title": "Site Name (optional)", "default": "" },
    "sign_type": { "type": "string", "title": "Sign Type", "default": "MD5" },
    "sign_key_mode": { "type": "string", "title": "Sign Key Mode", "default": "plain", "enum": ["plain","amp_key"], "description": "plain => md5(query + key); amp_key => md5(query + '&key=' + key)" },
    "timeout_sec": { "type": "integer", "title": "HTTP Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 },
    "order_expire_minutes": { "type": "integer", "title": "Order Expire Minutes", "default": 5, "minimum": 1, "maximum": 120, "description": "Reuse same order+method pay link in this window." }
  },
  "required": ["pid","merchant_key"]
}`,
		UiSchema: `{
  "merchant_key": { "ui:widget": "password", "ui:help": "Leave empty means keep unchanged (handled by host)" },
  "sign_type": { "ui:widget": "hidden" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	normalizeConfig(&cfg)
	if strings.TrimSpace(cfg.PID) == "" || strings.TrimSpace(cfg.MerchantKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "pid/merchant_key required"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	if req.GetConfigJson() != "" {
		var cfg config
		if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
			return &pluginv1.InitResponse{Ok: false, Error: "invalid config"}, nil
		}
		normalizeConfig(&cfg)
		s.cfg = cfg
	}
	s.instance = req.GetInstanceId()
	s.updatedAt = time.Now()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(ctx context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: "invalid config"}, nil
	}
	normalizeConfig(&cfg)
	s.cfg = cfg
	s.updatedAt = time.Now()
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(ctx context.Context, req *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	_ = ctx
	msg := "ok"
	if req.GetInstanceId() == "" || s.instance == "" {
		msg = "not initialized"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
		Message:    msg,
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

type payServer struct {
	pluginv1.UnimplementedPaymentServiceServer
	core      *coreServer
	mu        sync.RWMutex
	linkCache map[string]cachedPayment
}

type cachedPayment struct {
	resp      *pluginv1.PaymentCreateResponse
	createdAt time.Time
}

func (p *payServer) ListMethods(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListMethodsResponse, error) {
	_ = ctx
	return &pluginv1.ListMethodsResponse{Methods: []string{"alipay", "wxpay", "qqpay"}}, nil
}

func (p *payServer) CreatePayment(ctx context.Context, req *pluginv1.CreatePaymentRpcRequest) (*pluginv1.PaymentCreateResponse, error) {
	_ = ctx
	method := strings.TrimSpace(req.GetMethod())
	switch method {
	case "alipay", "wxpay", "qqpay":
		return p.createEZPay(method, req.GetRequest())
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
}

func (p *payServer) QueryPayment(ctx context.Context, req *pluginv1.QueryPaymentRpcRequest) (*pluginv1.PaymentQueryResponse, error) {
	method := strings.TrimSpace(req.GetMethod())
	if method != "alipay" && method != "wxpay" && method != "qqpay" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}

	orderNo := strings.TrimSpace(req.GetOrderNo())
	tradeNo := strings.TrimSpace(req.GetTradeNo())
	if orderNo == "" && tradeNo == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no required")
	}

	cfg := p.core.cfg
	queryURL := strings.TrimSpace(cfg.QueryAPIURL)
	if queryURL == "" {
		base := strings.TrimSpace(cfg.GatewayBaseURL)
		if base == "" {
			return nil, status.Error(codes.FailedPrecondition, "gateway_base_url or query_api_url required for QueryPayment")
		}
		// Default to PHP SDK style: GET api.php?act=order&pid=...&key=...&out_trade_no=...
		queryURL = strings.TrimRight(base, "/") + "/api.php?act=order"
	}

	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var httpReq *http.Request
	switch detectQueryAPIKind(queryURL) {
	case "findorder":
		if orderNo == "" {
			return nil, status.Error(codes.InvalidArgument, "order_no required for findorder")
		}
		form := url.Values{}
		form.Set("type", "1")
		form.Set("order_no", buildMethodOrderNo(orderNo, method, cfg, time.Now()))
		httpReq, _ = http.NewRequestWithContext(cctx, http.MethodPost, queryURL, strings.NewReader(form.Encode()))
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case "api.php":
		u, err := url.Parse(queryURL)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid query_api_url")
		}
		q := u.Query()
		if strings.TrimSpace(q.Get("act")) == "" {
			q.Set("act", "order")
		}
		q.Set("pid", strings.TrimSpace(cfg.PID))
		q.Set("key", strings.TrimSpace(cfg.MerchantKey))
		if tradeNo != "" {
			q.Set("trade_no", tradeNo)
		} else {
			q.Set("out_trade_no", buildMethodOrderNo(orderNo, method, cfg, time.Now()))
		}
		u.RawQuery = q.Encode()
		httpReq, _ = http.NewRequestWithContext(cctx, http.MethodGet, u.String(), nil)
	default:
		return nil, status.Error(codes.InvalidArgument, "unsupported query_api_url")
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "ezpay query failed: "+err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	parsed := parseQueryResponse(b)
	if !parsed.OK {
		return nil, status.Error(codes.FailedPrecondition, parsed.Error)
	}
	return &pluginv1.PaymentQueryResponse{
		Ok:      true,
		Status:  parsed.Status,
		TradeNo: parsed.TradeNo,
		Amount:  parsed.Amount,
		RawJson: string(b),
	}, nil
}

func (p *payServer) Refund(ctx context.Context, req *pluginv1.RefundRpcRequest) (*pluginv1.RefundResponse, error) {
	_ = req
	return nil, status.Error(codes.Unimplemented, "ezpay refund not supported")
}

func (p *payServer) VerifyNotify(ctx context.Context, req *pluginv1.VerifyNotifyRequest) (*pluginv1.NotifyVerifyResult, error) {
	method := strings.TrimSpace(req.GetMethod())
	if method != "alipay" && method != "wxpay" && method != "qqpay" {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	return p.verifyEZPay(req.GetRaw(), method)
}

func (p *payServer) createEZPay(method string, in *pluginv1.PaymentCreateRequest) (*pluginv1.PaymentCreateResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	cfg := p.core.cfg
	if strings.TrimSpace(cfg.PID) == "" || strings.TrimSpace(cfg.MerchantKey) == "" {
		return nil, status.Error(codes.FailedPrecondition, "config missing pid/merchant_key")
	}
	if strings.TrimSpace(in.GetOrderNo()) == "" {
		return nil, status.Error(codes.InvalidArgument, "order_no required")
	}
	if strings.TrimSpace(in.GetNotifyUrl()) == "" || strings.TrimSpace(in.GetReturnUrl()) == "" {
		return nil, status.Error(codes.InvalidArgument, "notify_url/return_url required (host-generated)")
	}
	moneyStr, err := centsToYuanString(in.GetAmount())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	submitURL, err := resolveSubmitURL(cfg)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	hostOrderNo := strings.TrimSpace(in.GetOrderNo())
	outTradeNo := buildMethodOrderNo(hostOrderNo, method, cfg, time.Now())
	if cached := p.getCachedPayment(outTradeNo, cfg); cached != nil {
		return cached, nil
	}

	params := map[string]string{
		"pid":          strings.TrimSpace(cfg.PID),
		"type":         method,
		"out_trade_no": outTradeNo,
		"notify_url":   strings.TrimSpace(in.GetNotifyUrl()),
		"return_url":   strings.TrimSpace(in.GetReturnUrl()),
		"name":         simplifyGoodsName(strings.TrimSpace(in.GetSubject())),
		"money":        moneyStr,
		"sign_type":    firstNonEmpty(cfg.SignType, "MD5"),
	}
	if strings.TrimSpace(cfg.SiteName) != "" {
		params["sitename"] = strings.TrimSpace(cfg.SiteName)
	}
	if v := strings.TrimSpace(in.GetExtra()["param"]); v != "" {
		params["param"] = v
	}
	if params["name"] == "" {
		params["name"] = "Order " + hostOrderNo
	}
	clientIP := strings.TrimSpace(in.GetExtra()["client_ip"])
	if clientIP == "" {
		clientIP = strings.TrimSpace(in.GetExtra()["clientip"])
	}
	if clientIP == "" {
		clientIP = strings.TrimSpace(in.GetExtra()["ip"])
	}
	if clientIP == "" {
		clientIP = strings.TrimSpace(in.GetExtra()["user_ip"])
	}
	if strings.TrimSpace(clientIP) == "" {
		return nil, status.Error(codes.InvalidArgument, "client_ip required")
	}
	params["clientip"] = strings.TrimSpace(clientIP)
	device := strings.TrimSpace(in.GetExtra()["device"])
	normalizedDevice, ok := normalizeEZPayDevice(device)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "device required")
	}
	params["device"] = normalizedDevice
	signMode := strings.ToLower(strings.TrimSpace(cfg.SignKeyMode))
	if signMode == "" {
		signMode = "plain"
	}
	params["sign"] = signEZPay(params, cfg.MerchantKey, signMode)
	// mapi.php returns JSON payload (payurl/qrcode/urlscheme) and must be parsed server-side.
	if isMAPIEndpoint(submitURL) {
		resp, err := p.createEZPayByAPI(submitURL, params)
		if err != nil {
			return nil, err
		}
		p.setCachedPayment(outTradeNo, resp)
		return resp, nil
	}
	formHTML := buildAutoSubmitFormHTML(submitURL, params)

	resp := &pluginv1.PaymentCreateResponse{
		Ok:     true,
		PayUrl: "",
		Extra: map[string]string{
			"pay_kind":     "form",
			"form_html":    formHTML,
			"out_trade_no": outTradeNo,
		},
	}
	p.setCachedPayment(outTradeNo, resp)
	return resp, nil
}

func (p *payServer) createEZPayByAPI(endpoint string, params map[string]string) (*pluginv1.PaymentCreateResponse, error) {
	form := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(v) == "" {
			continue
		}
		form.Set(k, v)
	}
	timeout := p.core.cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	cctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "ezpay create failed: "+err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var out struct {
		Code      int    `json:"code"`
		Msg       string `json:"msg"`
		TradeNo   string `json:"trade_no"`
		PayURL    string `json:"payurl"`
		QRCode    string `json:"qrcode"`
		URLScheme string `json:"urlscheme"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "ezpay create returned non-json")
	}
	if out.Code != 1 {
		msg := strings.TrimSpace(out.Msg)
		if msg == "" {
			msg = "ezpay create failed"
		}
		return nil, status.Error(codes.FailedPrecondition, msg)
	}

	extra := map[string]string{}
	switch {
	case strings.TrimSpace(out.PayURL) != "":
		extra["pay_kind"] = "redirect"
		extra["pay_url"] = strings.TrimSpace(out.PayURL)
	case strings.TrimSpace(out.QRCode) != "":
		extra["pay_kind"] = "qr"
		extra["code_url"] = strings.TrimSpace(out.QRCode)
	case strings.TrimSpace(out.URLScheme) != "":
		extra["pay_kind"] = "urlscheme"
		extra["urlscheme"] = strings.TrimSpace(out.URLScheme)
	default:
		return nil, status.Error(codes.FailedPrecondition, "ezpay create response missing pay target")
	}
	extra["out_trade_no"] = strings.TrimSpace(params["out_trade_no"])
	return &pluginv1.PaymentCreateResponse{
		Ok:      true,
		TradeNo: strings.TrimSpace(out.TradeNo),
		PayUrl:  strings.TrimSpace(out.PayURL),
		Extra:   extra,
	}, nil
}

func (p *payServer) verifyEZPay(raw *pluginv1.RawHttpRequest, method string) (*pluginv1.NotifyVerifyResult, error) {
	if raw == nil {
		return nil, status.Error(codes.InvalidArgument, "missing raw request")
	}
	cfg := p.core.cfg
	params := rawToParams(raw)

	if strings.TrimSpace(cfg.PID) != "" && strings.TrimSpace(params["pid"]) != "" && strings.TrimSpace(params["pid"]) != strings.TrimSpace(cfg.PID) {
		return nil, status.Error(codes.InvalidArgument, "pid mismatch")
	}
	if strings.TrimSpace(cfg.MerchantKey) == "" {
		return nil, status.Error(codes.FailedPrecondition, "config missing merchant_key")
	}
	if st := strings.TrimSpace(params["sign_type"]); st != "" && strings.ToUpper(st) != "MD5" {
		return nil, status.Error(codes.InvalidArgument, "unsupported sign_type")
	}
	if t := strings.TrimSpace(params["type"]); t != "" && method != "" && t != method {
		return nil, status.Error(codes.InvalidArgument, "type mismatch")
	}

	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return nil, status.Error(codes.InvalidArgument, "missing sign")
	}
	if !verifyMD5Sign(params, cfg.MerchantKey, sign) {
		return nil, status.Error(codes.InvalidArgument, "invalid sign")
	}

	outTradeNo := strings.TrimSpace(params["out_trade_no"])
	if outTradeNo == "" {
		return nil, status.Error(codes.InvalidArgument, "missing out_trade_no")
	}
	hostOrderNo := restoreHostOrderNo(outTradeNo)
	tradeNo := strings.TrimSpace(params["trade_no"])
	if tradeNo == "" {
		tradeNo = outTradeNo
	}

	tradeStatus := strings.TrimSpace(params["trade_status"])
	ps := mapEZPayTradeStatus(tradeStatus)

	amountCents, err := parseMoneyToCentsStrict(params["money"])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid money")
	}

	rawJSON, _ := json.Marshal(params)
	return &pluginv1.NotifyVerifyResult{
		Ok:      true,
		OrderNo: hostOrderNo,
		TradeNo: tradeNo,
		Amount:  amountCents,
		Status:  ps,
		AckBody: "success",
		RawJson: string(rawJSON),
	}, nil
}

func restoreHostOrderNo(outTradeNo string) string {
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return ""
	}
	for _, marker := range []string{"-wx-", "-qq-", "-zfb-"} {
		if idx := strings.LastIndex(outTradeNo, marker); idx > 0 {
			return outTradeNo[:idx]
		}
	}
	for _, suffix := range []string{"-wx", "-qq", "-zfb"} {
		if strings.HasSuffix(outTradeNo, suffix) && len(outTradeNo) > len(suffix) {
			return outTradeNo[:len(outTradeNo)-len(suffix)]
		}
	}
	return outTradeNo
}

func buildMethodOrderNo(hostOrderNo, method string, cfg config, now time.Time) string {
	hostOrderNo = strings.TrimSpace(hostOrderNo)
	if hostOrderNo == "" {
		return ""
	}
	token := buildWindowToken(now, cfg.OrderExpireMinutes)
	switch strings.TrimSpace(method) {
	case "wxpay":
		return hostOrderNo + "-wx-" + token
	case "qqpay":
		return hostOrderNo + "-qq-" + token
	case "alipay":
		return hostOrderNo + "-zfb-" + token
	default:
		return hostOrderNo
	}
}

func buildWindowToken(now time.Time, expireMinutes int) string {
	if expireMinutes <= 0 {
		expireMinutes = 5
	}
	windowSec := int64(expireMinutes) * 60
	if windowSec <= 0 {
		windowSec = 300
	}
	bucket := now.Unix() / windowSec
	return strconv.FormatInt(bucket, 36)
}

func cloneCreateResponse(in *pluginv1.PaymentCreateResponse) *pluginv1.PaymentCreateResponse {
	if in == nil {
		return nil
	}
	out := &pluginv1.PaymentCreateResponse{
		Ok:      in.GetOk(),
		Error:   in.GetError(),
		TradeNo: in.GetTradeNo(),
		PayUrl:  in.GetPayUrl(),
		Extra:   map[string]string{},
	}
	for k, v := range in.GetExtra() {
		out.Extra[k] = v
	}
	return out
}

func (p *payServer) getCachedPayment(cacheKey string, cfg config) *pluginv1.PaymentCreateResponse {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.linkCache == nil {
		return nil
	}
	item, ok := p.linkCache[cacheKey]
	if !ok || item.resp == nil {
		return nil
	}
	expire := time.Duration(cfg.OrderExpireMinutes) * time.Minute
	if expire <= 0 {
		expire = 5 * time.Minute
	}
	if time.Since(item.createdAt) > expire {
		return nil
	}
	return cloneCreateResponse(item.resp)
}

func (p *payServer) setCachedPayment(cacheKey string, resp *pluginv1.PaymentCreateResponse) {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" || resp == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.linkCache == nil {
		p.linkCache = map[string]cachedPayment{}
	}
	p.linkCache[cacheKey] = cachedPayment{
		resp:      cloneCreateResponse(resp),
		createdAt: time.Now(),
	}
}

func rawToParams(req *pluginv1.RawHttpRequest) map[string]string {
	out := map[string]string{}
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
		ct := ""
		if v := req.GetHeaders()["Content-Type"]; v != nil && len(v.Values) > 0 {
			ct = v.Values[0]
		}
		if strings.Contains(strings.ToLower(ct), "application/x-www-form-urlencoded") || strings.Contains(string(req.GetBody()), "=") {
			if q, err := url.ParseQuery(string(req.GetBody())); err == nil {
				for k, v := range q {
					if len(v) > 0 && out[k] == "" {
						out[k] = v[0]
					}
				}
			}
		}
	}
	return out
}

func detectQueryAPIKind(queryURL string) string {
	u := strings.ToLower(strings.TrimSpace(queryURL))
	if strings.Contains(u, "/api/findorder") {
		return "findorder"
	}
	if strings.Contains(u, "/api.php") {
		return "api.php"
	}
	return ""
}

func mapEZPayTradeStatus(tradeStatus string) pluginv1.PaymentStatus {
	s := strings.ToUpper(strings.TrimSpace(tradeStatus))
	switch s {
	case "TRADE_SUCCESS":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PAID
	case "TRADE_CLOSED", "CLOSED":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_CLOSED
	case "TRADE_FAIL", "FAIL", "FAILED":
		return pluginv1.PaymentStatus_PAYMENT_STATUS_FAILED
	default:
		// Important: only TRADE_SUCCESS counts as success.
		return pluginv1.PaymentStatus_PAYMENT_STATUS_PENDING
	}
}

type queryParseResult struct {
	OK      bool
	Error   string
	Status  pluginv1.PaymentStatus
	TradeNo string
	Amount  int64
}

func parseQueryResponse(b []byte) queryParseResult {
	raw := strings.TrimSpace(string(b))
	if raw == "" {
		return queryParseResult{OK: false, Error: "ezpay query empty response"}
	}

	// Try findorder response: {code:200,msg:"",data:[{...}]}
	type findorderResp struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data []json.RawMessage `json:"data"`
	}
	var fo findorderResp
	if err := json.Unmarshal(b, &fo); err == nil && fo.Code != 0 {
		if fo.Code != 200 {
			return queryParseResult{OK: false, Error: fmt.Sprintf("ezpay query failed: code=%d msg=%s", fo.Code, fo.Msg)}
		}
		if len(fo.Data) == 0 {
			return queryParseResult{OK: false, Error: "ezpay query empty data"}
		}
		var item map[string]any
		_ = json.Unmarshal(fo.Data[0], &item)
		return parseQueryItem(item)
	}

	// Try generic JSON object
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err == nil && len(obj) > 0 {
		// Some APIs might wrap the order under "data"
		if v, ok := obj["data"]; ok {
			if m, ok2 := v.(map[string]any); ok2 {
				return parseQueryItem(m)
			}
			if arr, ok2 := v.([]any); ok2 && len(arr) > 0 {
				if m, ok3 := arr[0].(map[string]any); ok3 {
					return parseQueryItem(m)
				}
			}
		}
		return parseQueryItem(obj)
	}

	return queryParseResult{OK: false, Error: "ezpay query invalid response"}
}

func parseQueryItem(item map[string]any) queryParseResult {
	if len(item) == 0 {
		return queryParseResult{OK: false, Error: "ezpay query empty item"}
	}
	tradeStatus := fmt.Sprintf("%v", item["trade_status"])
	if tradeStatus == "" || tradeStatus == "<nil>" {
		tradeStatus = fmt.Sprintf("%v", item["status"])
	}
	status := mapEZPayTradeStatus(tradeStatus)

	tradeNo := fmt.Sprintf("%v", item["trade_no"])
	if tradeNo == "" || tradeNo == "<nil>" {
		tradeNo = fmt.Sprintf("%v", item["transaction_id"])
	}
	if tradeNo == "<nil>" {
		tradeNo = ""
	}

	amountCents := int64(0)
	moneyVal, hasMoney := item["money"]
	if !hasMoney {
		moneyVal = item["amount"]
	}
	if moneyVal != nil {
		switch v := moneyVal.(type) {
		case string:
			if c, err := parseMoneyToCentsStrict(v); err == nil {
				amountCents = c
			}
		default:
			if c, err := parseMoneyToCentsStrict(fmt.Sprintf("%v", v)); err == nil {
				amountCents = c
			}
		}
	}

	return queryParseResult{
		OK:      true,
		Status:  status,
		TradeNo: tradeNo,
		Amount:  amountCents,
	}
}

func phpStripslashes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != '\\' || i+1 >= len(s) {
			b.WriteByte(ch)
			continue
		}
		next := s[i+1]
		switch next {
		case '\\', '\'', '"':
			b.WriteByte(next)
			i++
			continue
		case '0':
			b.WriteByte(0)
			i++
			continue
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func signEZPay(params map[string]string, key string, mode string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if strings.TrimSpace(params[k]) == "" {
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
		buf.WriteString(phpStripslashes(params[k]))
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "amp_key":
		buf.WriteString("&key=")
		buf.WriteString(key)
	case "plain", "":
		buf.WriteString(key)
	default:
		buf.WriteString(key)
	}
	sum := md5.Sum([]byte(buf.String()))
	return hex.EncodeToString(sum[:])
}

func verifyMD5Sign(params map[string]string, key string, sign string) bool {
	// Always accept both common modes for notify verification.
	a := signEZPay(params, key, "amp_key")
	if strings.EqualFold(a, sign) {
		return true
	}
	b := signEZPay(params, key, "plain")
	return strings.EqualFold(b, sign)
}

func parseMoneyToCentsStrict(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, fmt.Errorf("money required")
	}
	neg := false
	if strings.HasPrefix(v, "-") {
		neg = true
		v = strings.TrimPrefix(v, "-")
	}
	if v == "" {
		return 0, fmt.Errorf("money invalid")
	}
	intPart := v
	fracPart := ""
	if strings.Contains(v, ".") {
		parts := strings.SplitN(v, ".", 2)
		intPart = parts[0]
		fracPart = parts[1]
	}
	if intPart == "" {
		intPart = "0"
	}
	i, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money invalid")
	}
	if fracPart != "" {
		if len(fracPart) > 2 {
			fracPart = fracPart[:2]
		}
		for len(fracPart) < 2 {
			fracPart += "0"
		}
	} else {
		fracPart = "00"
	}
	f, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money invalid")
	}
	out := i*100 + f
	if neg {
		out = -out
	}
	return out, nil
}

func centsToYuanString(cents int64) (string, error) {
	if cents <= 0 {
		return "", fmt.Errorf("amount must be > 0")
	}
	yuan := cents / 100
	rem := cents % 100
	if rem < 0 {
		rem = -rem
	}
	return fmt.Sprintf("%d.%02d", yuan, rem), nil
}

func normalizeConfig(cfg *config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.MerchantKey) == "" && strings.TrimSpace(cfg.Key) != "" {
		cfg.MerchantKey = cfg.Key
	}
	if strings.TrimSpace(cfg.QueryAPIURL) == "" && strings.TrimSpace(cfg.QueryURL) != "" {
		cfg.QueryAPIURL = cfg.QueryURL
	}
	if strings.TrimSpace(cfg.BaseURL) != "" {
		base := strings.TrimSpace(cfg.BaseURL)
		if strings.TrimSpace(cfg.SubmitURL) == "" && strings.HasSuffix(strings.ToLower(base), "/submit.php") {
			cfg.SubmitURL = base
		}
		if strings.TrimSpace(cfg.GatewayBaseURL) == "" && !strings.HasSuffix(strings.ToLower(base), "/submit.php") {
			cfg.GatewayBaseURL = base
		}
	}
	if strings.TrimSpace(cfg.SignKeyMode) == "" {
		cfg.SignKeyMode = "plain"
	}
	if cfg.OrderExpireMinutes <= 0 {
		cfg.OrderExpireMinutes = 5
	}
}

func resolveSubmitURL(cfg config) (string, error) {
	if strings.TrimSpace(cfg.SubmitURL) != "" {
		return strings.TrimSpace(cfg.SubmitURL), nil
	}
	if strings.TrimSpace(cfg.GatewayBaseURL) == "" {
		return "", fmt.Errorf("gateway_base_url or submit_url required")
	}
	path := strings.TrimSpace(cfg.SubmitPath)
	if path == "" {
		path = "mapi.php"
	}
	path = strings.TrimPrefix(path, "/")
	return strings.TrimRight(strings.TrimSpace(cfg.GatewayBaseURL), "/") + "/" + path, nil
}

func isMAPIEndpoint(u string) bool {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return strings.Contains(strings.ToLower(u), "mapi.php")
	}
	return strings.EqualFold(path.Base(parsed.Path), "mapi.php")
}

func normalizeEZPayDevice(v string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "pc", "mobile", "qq", "wechat", "alipay", "jump":
		return s, true
	default:
		return "", false
	}
}

func simplifyGoodsName(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" || strings.HasPrefix(strings.ToLower(subject), "order ") {
		return "订单支付"
	}
	return subject
}

func buildAutoSubmitFormHTML(action string, params map[string]string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>Redirecting...</title></head><body>")
	b.WriteString("<form id=\"pay\" method=\"post\" action=\"")
	b.WriteString(htmlEscape(action))
	b.WriteString("\">")
	keys := make([]string, 0, len(params))
	for k := range params {
		if strings.TrimSpace(params[k]) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("<input type=\"hidden\" name=\"")
		b.WriteString(htmlEscape(k))
		b.WriteString("\" value=\"")
		b.WriteString(htmlEscape(params[k]))
		b.WriteString("\"/>")
	}
	b.WriteString("</form><script>document.getElementById('pay').submit();</script></body></html>")
	return b.String()
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

func firstNonEmpty(v string, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func main() {
	core := &coreServer{}
	pay := &payServer{core: core}

	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:    &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyPayment: &pluginsdk.PaymentGRPCPlugin{Impl: pay},
	})
}
