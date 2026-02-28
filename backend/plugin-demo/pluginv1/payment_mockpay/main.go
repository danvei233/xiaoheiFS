package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

const (
	pluginID        = "mockpay"
	pluginMethod    = "mock"
	confirmTextNeed = "我已确认我在进行测试而非生产环境"
)

type config struct {
	ConfirmText string `json:"confirm_text"`
	TTLSeconds  int64  `json:"ttl_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg      config
	instance string
}

func (s *coreServer) GetManifest(context.Context, *pluginv1.Empty) (*pluginv1.Manifest, error) {
	return &pluginv1.Manifest{
		PluginId:    pluginID,
		Name:        "MockPay Disk Plugin",
		Version:     "1.0.0",
		Description: "Local test-only payment plugin. click pass => paid.",
		Payment:     &pluginv1.PaymentCapability{Methods: []string{pluginMethod}},
	}, nil
}

func (s *coreServer) GetConfigSchema(context.Context, *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title":"MockPay Disk Plugin (Test Only)",
  "type":"object",
  "properties":{
    "confirm_text":{"type":"string","title":"确认语","description":"必须填写：我已确认我在进行测试而非生产环境"},
    "ttl_sec":{"type":"integer","title":"签名有效期秒","default":86400,"minimum":60}
  },
  "required":["confirm_text"]
}`,
		UiSchema: `{}`,
	}, nil
}

func (s *coreServer) ValidateConfig(_ context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	cfg, err := parseConfig(req.GetConfigJson())
	if err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: err.Error()}, nil
	}
	if strings.TrimSpace(cfg.ConfirmText) != confirmTextNeed {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "confirm_text mismatch"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(_ context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	cfg, err := parseConfig(req.GetConfigJson())
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: err.Error()}, nil
	}
	s.cfg = cfg
	s.instance = req.GetInstanceId()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(_ context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	cfg, err := parseConfig(req.GetConfigJson())
	if err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: err.Error()}, nil
	}
	s.cfg = cfg
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(_ context.Context, req *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	if strings.TrimSpace(s.instance) == "" {
		s.instance = req.GetInstanceId()
	}
	statusValue := pluginv1.HealthStatus_HEALTH_STATUS_OK
	message := "ok"
	if strings.TrimSpace(s.cfg.ConfirmText) != confirmTextNeed {
		statusValue = pluginv1.HealthStatus_HEALTH_STATUS_ERROR
		message = "confirm_text mismatch"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     statusValue,
		Message:    message,
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

type payServer struct {
	pluginv1.UnimplementedPaymentServiceServer
	core     *coreServer
	mu       sync.RWMutex
	pending  map[string]pendingPayment
	webOnce  sync.Once
	warmOnce sync.Once
	webErr   error
	local    string
	public   string
}

type pendingPayment struct {
	Token     string
	OrderNo   string
	TradeNo   string
	Amount    int64
	Currency  string
	NotifyURL string
	ReturnURL string
	ExpiresAt time.Time
	Approved  bool
}

func (p *payServer) ListMethods(context.Context, *pluginv1.Empty) (*pluginv1.ListMethodsResponse, error) {
	return &pluginv1.ListMethodsResponse{Ok: true, Methods: []string{pluginMethod}}, nil
}

func (p *payServer) CreatePayment(_ context.Context, req *pluginv1.CreatePaymentRpcRequest) (*pluginv1.PaymentCreateResponse, error) {
	if strings.TrimSpace(req.GetMethod()) != pluginMethod {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	in := req.GetRequest()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	cfg := p.core.cfg
	if strings.TrimSpace(cfg.ConfirmText) != confirmTextNeed {
		return nil, status.Error(codes.FailedPrecondition, "confirm_text mismatch")
	}
	if strings.TrimSpace(in.GetOrderNo()) == "" || in.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid order")
	}
	if err := p.ensureWebServer(); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	notifyURL, err := resolveNotifyURL(in.GetNotifyUrl(), in.GetReturnUrl(), in.GetExtra())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	tradeNo := fmt.Sprintf("MOCK-%s-%d", in.GetOrderNo(), time.Now().UnixNano())
	token := randomToken()
	now := time.Now()
	expire := now.Add(time.Duration(cfg.TTLSeconds) * time.Second)
	if p.pending == nil {
		p.pending = map[string]pendingPayment{}
	}
	p.mu.Lock()
	p.pending[token] = pendingPayment{
		Token:     token,
		OrderNo:   strings.TrimSpace(in.GetOrderNo()),
		TradeNo:   tradeNo,
		Amount:    in.GetAmount(),
		Currency:  strings.TrimSpace(in.GetCurrency()),
		NotifyURL: notifyURL,
		ReturnURL: strings.TrimSpace(in.GetReturnUrl()),
		ExpiresAt: expire,
		Approved:  false,
	}
	p.mu.Unlock()
	payURL := strings.TrimRight(p.public, "/") + "/mockpay/checkout?token=" + url.QueryEscape(token)
	return &pluginv1.PaymentCreateResponse{
		Ok:      true,
		TradeNo: tradeNo,
		PayUrl:  payURL,
		Extra: map[string]string{
			"qr_code":          payURL,
			"provider":         pluginID + "." + pluginMethod,
			"debug_confirm":    confirmTextNeed,
			"debug_token":      token,
			"debug_notify_url": notifyURL,
			"debug_local_url":  p.local,
			"debug_public_url": p.public,
		},
	}, nil
}

func (p *payServer) QueryPayment(context.Context, *pluginv1.QueryPaymentRpcRequest) (*pluginv1.PaymentQueryResponse, error) {
	return &pluginv1.PaymentQueryResponse{
		Ok:     false,
		Error:  "not implemented in mock plugin",
		Status: pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED,
	}, nil
}

func (p *payServer) Refund(context.Context, *pluginv1.RefundRpcRequest) (*pluginv1.RefundResponse, error) {
	return &pluginv1.RefundResponse{
		Ok:    false,
		Error: "not implemented in mock plugin",
	}, nil
}

func (p *payServer) VerifyNotify(_ context.Context, req *pluginv1.VerifyNotifyRequest) (*pluginv1.NotifyVerifyResult, error) {
	if strings.TrimSpace(req.GetMethod()) != pluginMethod {
		return nil, status.Error(codes.InvalidArgument, "unknown method")
	}
	params := rawToParams(req.GetRaw())
	token := strings.TrimSpace(params["token"])
	orderNo := strings.TrimSpace(params["order_no"])
	tradeNo := strings.TrimSpace(params["trade_no"])
	amountText := strings.TrimSpace(params["amount"])
	paid := strings.ToLower(strings.TrimSpace(params["paid"]))
	if token == "" || orderNo == "" || tradeNo == "" || amountText == "" {
		return nil, status.Error(codes.InvalidArgument, "missing params")
	}
	if paid != "1" && paid != "true" && paid != "yes" && paid != "approved" && paid != "pass" {
		return nil, status.Error(codes.InvalidArgument, "paid flag required")
	}
	amount, err := strconv.ParseInt(amountText, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}
	p.mu.Lock()
	pd, ok := p.pending[token]
	if !ok {
		p.mu.Unlock()
		return nil, status.Error(codes.InvalidArgument, "token not found")
	}
	if time.Now().After(pd.ExpiresAt) {
		delete(p.pending, token)
		p.mu.Unlock()
		return nil, status.Error(codes.InvalidArgument, "token expired")
	}
	if pd.OrderNo != orderNo || pd.TradeNo != tradeNo || pd.Amount != amount {
		p.mu.Unlock()
		return nil, status.Error(codes.InvalidArgument, "token mismatch")
	}
	pd.Approved = true
	p.pending[token] = pd
	p.mu.Unlock()
	rawJSON, _ := json.Marshal(params)
	return &pluginv1.NotifyVerifyResult{
		Ok:      true,
		OrderNo: orderNo,
		TradeNo: tradeNo,
		Amount:  amount,
		Status:  pluginv1.PaymentStatus_PAYMENT_STATUS_PAID,
		RawJson: string(rawJSON),
		AckBody: `{"ok":true,"provider":"mockpay.mock","paid":true}`,
	}, nil
}

func parseConfig(configJSON string) (config, error) {
	cfg := config{
		TTLSeconds: 86400,
	}
	if strings.TrimSpace(configJSON) != "" {
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return config{}, fmt.Errorf("invalid config json")
		}
	}
	cfg.ConfirmText = strings.TrimSpace(cfg.ConfirmText)
	if cfg.TTLSeconds < 60 {
		cfg.TTLSeconds = 86400
	}
	return cfg, nil
}

func rawToParams(raw *pluginv1.RawHttpRequest) map[string]string {
	out := map[string]string{}
	if raw == nil {
		return out
	}
	if q, err := url.ParseQuery(strings.TrimSpace(raw.GetRawQuery())); err == nil {
		for k, v := range q {
			if len(v) > 0 && out[k] == "" {
				out[k] = v[0]
			}
		}
	}
	if len(raw.GetBody()) > 0 {
		if q, err := url.ParseQuery(string(raw.GetBody())); err == nil {
			for k, v := range q {
				if len(v) > 0 && out[k] == "" {
					out[k] = v[0]
				}
			}
		}
	}
	return out
}

func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("tok-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func resolveNotifyURL(notifyURL, returnURL string, extra map[string]string) (string, error) {
	notifyURL = strings.TrimSpace(notifyURL)
	if notifyURL != "" {
		return notifyURL, nil
	}
	returnURL = strings.TrimSpace(returnURL)
	if returnURL != "" {
		if u, err := url.Parse(returnURL); err == nil && strings.TrimSpace(u.Scheme) != "" && strings.TrimSpace(u.Host) != "" {
			return u.Scheme + "://" + u.Host + "/api/v1/payments/notify/" + pluginID + "." + pluginMethod, nil
		}
	}
	if len(extra) > 0 {
		for _, k := range []string{"base_url", "site_url", "origin", "public_base_url", "callback_base"} {
			v := strings.TrimSpace(extra[k])
			if v == "" {
				continue
			}
			if u, err := url.Parse(v); err == nil && strings.TrimSpace(u.Scheme) != "" && strings.TrimSpace(u.Host) != "" {
				return u.Scheme + "://" + u.Host + "/api/v1/payments/notify/" + pluginID + "." + pluginMethod, nil
			}
		}
	}
	return "http://127.0.0.1:8080/api/v1/payments/notify/" + pluginID + "." + pluginMethod, nil
}

func (p *payServer) ensureWebServer() error {
	p.webOnce.Do(func() {
		p.webErr = p.startWebServer()
	})
	return p.webErr
}

func (p *payServer) prewarm() {
	p.warmOnce.Do(func() {
		go func() {
			_ = p.ensureWebServer()
		}()
	})
}

func (p *payServer) startWebServer() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("mockpay listen failed: %w", err)
	}
	p.local = "http://" + ln.Addr().String()
	publicBase, err := ensurePublicBaseURL(p.local)
	if err != nil {
		_ = ln.Close()
		return err
	}
	p.public = strings.TrimRight(publicBase, "/")
	mux := http.NewServeMux()
	mux.HandleFunc("/mockpay/checkout", p.handleCheckoutPage)
	mux.HandleFunc("/mockpay/pass", p.handlePassPayment)
	go func() {
		_ = http.Serve(ln, mux)
	}()
	return nil
}

func (p *payServer) handleCheckoutPage(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	p.mu.RLock()
	pd, ok := p.pending[token]
	p.mu.RUnlock()
	if !ok {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if time.Now().After(pd.ExpiresAt) {
		http.Error(w, "token expired", http.StatusGone)
		return
	}
	body := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>MockPay</title><style>body{font-family:Arial,sans-serif;margin:24px;line-height:1.5}.box{border:1px solid #ddd;padding:16px}.ok{background:#e6ffed;padding:10px}button{padding:10px 16px}</style></head><body><h2>MockPay 测试收银台</h2><div class="box"><p>order_no=%s</p><p>trade_no=%s</p><p>amount=%s</p><p>currency=%s</p><p>expire_at=%s</p><p class="ok">点击“通过支付”将由 mock 插件服务端回调 notify 并标记已支付。</p><form method="post" action="/mockpay/pass" enctype="application/x-www-form-urlencoded"><input type="hidden" name="token" value="%s"/><button type="submit">通过支付</button></form><hr/><h4>notify_url</h4><pre>%s</pre><h4>local_url</h4><pre>%s</pre><h4>public_url</h4><pre>%s</pre></div></body></html>`,
		htmlEscape(pd.OrderNo), htmlEscape(pd.TradeNo), htmlEscape(strconv.FormatInt(pd.Amount, 10)), htmlEscape(pd.Currency), htmlEscape(pd.ExpiresAt.Format(time.RFC3339)),
		htmlEscape(pd.Token), htmlEscape(pd.NotifyURL), htmlEscape(p.local), htmlEscape(p.public))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (p *payServer) handlePassPayment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}
	p.mu.RLock()
	pd, ok := p.pending[token]
	p.mu.RUnlock()
	if !ok {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if time.Now().After(pd.ExpiresAt) {
		http.Error(w, "token expired", http.StatusGone)
		return
	}
	statusCode, respBody, err := postNotify(pd)
	if err != nil {
		http.Error(w, fmt.Sprintf("notify failed: %v", err), http.StatusBadGateway)
		return
	}
	returnURL := strings.TrimSpace(pd.ReturnURL)
	if returnURL == "" {
		returnURL = "/console/billing"
	}
	body := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>MockPay Done</title><style>body{font-family:Arial,sans-serif;margin:24px;line-height:1.5}.ok{background:#e6ffed;padding:10px}pre{white-space:pre-wrap;word-break:break-all}</style></head><body><h2>支付已提交</h2><p class="ok">notify 回调已发送。</p><p>status=%d</p><h4>notify response</h4><pre>%s</pre><p><a href="%s">返回业务页面</a></p></body></html>`,
		statusCode, htmlEscape(respBody), htmlEscape(returnURL))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func postNotify(pd pendingPayment) (int, string, error) {
	form := url.Values{}
	form.Set("token", pd.Token)
	form.Set("order_no", pd.OrderNo)
	form.Set("trade_no", pd.TradeNo)
	form.Set("amount", strconv.FormatInt(pd.Amount, 10))
	form.Set("currency", pd.Currency)
	form.Set("paid", "1")
	req, err := http.NewRequest(http.MethodPost, pd.NotifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, string(body), fmt.Errorf("notify status=%d", resp.StatusCode)
	}
	return resp.StatusCode, string(body), nil
}

func ensurePublicBaseURL(localBase string) (string, error) {
	if raw := strings.TrimSpace(os.Getenv("MOCKPAY_PUBLIC_BASE_URL")); raw != "" {
		if u, err := url.Parse(raw); err == nil && (u.Scheme == "http" || u.Scheme == "https") && strings.TrimSpace(u.Host) != "" {
			return strings.TrimRight(raw, "/"), nil
		}
		return "", fmt.Errorf("invalid MOCKPAY_PUBLIC_BASE_URL")
	}
	bin, err := ensureCloudflaredBinary()
	if err != nil {
		return "", err
	}
	publicURL, err := startCloudflaredTunnel(bin, localBase)
	if err != nil {
		return "", err
	}
	return publicURL, nil
}

func ensureCloudflaredBinary() (string, error) {
	if bin, err := exec.LookPath("cloudflared"); err == nil && strings.TrimSpace(bin) != "" {
		return bin, nil
	}
	var name, dl string
	switch runtime.GOOS + "_" + runtime.GOARCH {
	case "windows_amd64":
		name = "cloudflared.exe"
		dl = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe"
	case "linux_amd64":
		name = "cloudflared"
		dl = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64"
	default:
		return "", fmt.Errorf("cloudflared auto download not supported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	dir := filepath.Join(os.TempDir(), "mockpay-cloudflared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cloudflared dir failed: %w", err)
	}
	out := filepath.Join(dir, name)
	if st, err := os.Stat(out); err == nil && st.Size() > 1024*1024 {
		return out, nil
	}
	resp, err := http.Get(dl)
	if err != nil {
		return "", fmt.Errorf("download cloudflared failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download cloudflared failed: status=%d", resp.StatusCode)
	}
	f, err := os.Create(out)
	if err != nil {
		return "", fmt.Errorf("create cloudflared file failed: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("save cloudflared failed: %w", err)
	}
	_ = f.Close()
	_ = os.Chmod(out, 0o755)
	return out, nil
}

func startCloudflaredTunnel(bin, localBase string) (string, error) {
	cmd := exec.Command(bin, "tunnel", "--url", localBase, "--no-autoupdate", "--metrics", "127.0.0.1:0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("cloudflared stdout pipe failed: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("cloudflared stderr pipe failed: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("cloudflared start failed: %w", err)
	}
	re := regexp.MustCompile(`https://[-a-zA-Z0-9]+\.trycloudflare\.com`)
	found := make(chan string, 1)
	scan := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			line := sc.Text()
			if u := re.FindString(line); u != "" {
				select {
				case found <- strings.TrimRight(u, "/"):
				default:
				}
			}
		}
	}
	go scan(stdout)
	go scan(stderr)
	select {
	case u := <-found:
		return u, nil
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("cloudflared tunnel url not ready in time")
	}
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

func main() {
	core := &coreServer{}
	pay := &payServer{core: core, pending: map[string]pendingPayment{}}
	pay.prewarm()
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:    &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyPayment: &pluginsdk.PaymentGRPCPlugin{Impl: pay},
	})
}
