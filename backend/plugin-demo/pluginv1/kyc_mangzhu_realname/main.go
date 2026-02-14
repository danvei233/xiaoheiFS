package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	AuthMode     string `json:"auth_mode"`
	FaceProvider string `json:"face_provider"`
	CallbackURL  string `json:"callback_url"`
	TimeoutSec   int    `json:"timeout_sec"`
}

type cachedResult struct {
	status  string
	reason  string
	rawJSON string
	expire  time.Time
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
		PluginId:    "mangzhu_realname",
		Name:        "Mangzhu Realname",
		Version:     "1.0.0",
		Description: "Mangzhu realname verification: two-factor, three-factor, and face-flow (Baidu/WeChat).",
		Kyc:         &pluginv1.KycCapability{Start: true, QueryResult: true},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Mangzhu Realname",
  "type": "object",
  "properties": {
    "base_url": { "type": "string", "title": "Base URL", "default": "https://e.mangzhuyun.cn" },
    "api_key": { "type": "string", "title": "API Key", "format": "password", "x-secret": true },
    "auth_mode": { "type": "string", "title": "Auth Mode", "default": "three_factor", "enum": ["two_factor", "three_factor", "face"] },
    "face_provider": { "type": "string", "title": "Face Provider", "default": "baidu", "enum": ["baidu", "wechat"] },
    "callback_url": { "type": "string", "title": "Face Callback URL (optional)", "default": "" },
    "timeout_sec": { "type": "integer", "title": "HTTP Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["api_key"]
}`,
		UiSchema: `{
  "api_key": { "ui:widget": "password", "ui:help": "Leave empty means keep unchanged (handled by host)." }
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
	if strings.TrimSpace(cfg.APIKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "api_key required"}, nil
	}
	if cfg.AuthMode == "face" && cfg.FaceProvider != "baidu" && cfg.FaceProvider != "wechat" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "face_provider must be baidu or wechat"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	_ = ctx
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
	_, err := s.Init(ctx, &pluginv1.InitRequest{
		InstanceId: s.instance,
		ConfigJson: req.GetConfigJson(),
	})
	if err != nil {
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: err.Error()}, nil
	}
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(ctx context.Context, _ *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	_ = ctx
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
		Message:    "ok",
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

type kycServer struct {
	pluginv1.UnimplementedKycServiceServer
	core *coreServer

	mu          sync.RWMutex
	syncResults map[string]cachedResult
	faceTokens  map[string]string
}

func (k *kycServer) Start(ctx context.Context, req *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	cfg := k.core.cfg
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, status.Error(codes.FailedPrecondition, "config missing api_key")
	}
	name := strings.TrimSpace(req.GetParams()["name"])
	idNumber := strings.TrimSpace(req.GetParams()["id_number"])
	if name == "" || idNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "params.name/params.id_number required")
	}

	switch cfg.AuthMode {
	case "two_factor":
		raw, code, msg, err := doRequest(ctx, cfg, "/index/sm_api", map[string]string{
			"key":    cfg.APIKey,
			"name":   name,
			"idcard": idNumber,
		})
		if err != nil {
			return nil, status.Error(codes.Unavailable, "mangzhu two_factor failed: "+err.Error())
		}
		ok, reason := parseTwoFactorResult(code, msg, raw)
		token := newToken()
		k.setSyncResult(token, ok, reason, raw)
		return &pluginv1.KycStartResponse{Ok: true, Token: token, NextStep: "query_result"}, nil
	case "three_factor":
		mobile := strings.TrimSpace(req.GetParams()["mobile"])
		if mobile == "" {
			mobile = strings.TrimSpace(req.GetParams()["phone"])
		}
		if mobile == "" {
			return nil, status.Error(codes.InvalidArgument, "params.mobile required for three_factor")
		}
		raw, code, msg, err := doRequest(ctx, cfg, "/index/sm3_api", map[string]string{
			"key":    cfg.APIKey,
			"name":   name,
			"idcard": idNumber,
			"mobile": mobile,
		})
		if err != nil {
			return nil, status.Error(codes.Unavailable, "mangzhu three_factor failed: "+err.Error())
		}
		ok, reason := parseThreeFactorResult(code, msg, raw)
		token := newToken()
		k.setSyncResult(token, ok, reason, raw)
		return &pluginv1.KycStartResponse{Ok: true, Token: token, NextStep: "query_result"}, nil
	case "face":
		callbackURL := strings.TrimSpace(req.GetParams()["callback_url"])
		if callbackURL == "" {
			callbackURL = strings.TrimSpace(cfg.CallbackURL)
		}
		if callbackURL == "" {
			return nil, status.Error(codes.InvalidArgument, "callback_url required for face mode")
		}
		startPath := "/index/bd_sm"
		if cfg.FaceProvider == "wechat" {
			startPath = "/index/wx_sm"
		}
		raw, code, msg, err := doRequest(ctx, cfg, startPath, map[string]string{
			"key":    cfg.APIKey,
			"name":   name,
			"idcard": idNumber,
			"url":    callbackURL,
		})
		if err != nil {
			return nil, status.Error(codes.Unavailable, "mangzhu face start failed: "+err.Error())
		}
		token, redirectURL, perr := parseFaceStart(code, msg, raw)
		if perr != "" {
			return nil, status.Error(codes.FailedPrecondition, perr)
		}
		k.mu.Lock()
		if k.faceTokens == nil {
			k.faceTokens = map[string]string{}
		}
		k.faceTokens[token] = cfg.FaceProvider
		k.mu.Unlock()
		return &pluginv1.KycStartResponse{
			Ok:       true,
			Token:    token,
			Url:      redirectURL,
			NextStep: "redirect",
		}, nil
	default:
		return nil, status.Error(codes.FailedPrecondition, "unsupported auth_mode")
	}
}

func (k *kycServer) QueryResult(ctx context.Context, req *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
	if req == nil || strings.TrimSpace(req.GetToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	token := strings.TrimSpace(req.GetToken())
	if item, ok := k.getSyncResult(token); ok {
		return &pluginv1.KycQueryResponse{
			Ok:      true,
			Status:  item.status,
			Reason:  item.reason,
			RawJson: item.rawJSON,
		}, nil
	}

	cfg := k.core.cfg
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, status.Error(codes.FailedPrecondition, "config missing api_key")
	}
	faceProvider := cfg.FaceProvider
	k.mu.RLock()
	if k.faceTokens != nil {
		if p, ok := k.faceTokens[token]; ok && strings.TrimSpace(p) != "" {
			faceProvider = p
		}
	}
	k.mu.RUnlock()
	queryPath := "/index/bd_cx"
	if faceProvider == "wechat" {
		queryPath = "/index/wx_cx"
	}
	raw, code, msg, err := doRequest(ctx, cfg, queryPath, map[string]string{
		"key":   cfg.APIKey,
		"token": token,
	})
	if err != nil {
		return nil, status.Error(codes.Unavailable, "mangzhu face query failed: "+err.Error())
	}
	st, reason := parseFaceQuery(code, msg, raw, faceProvider)
	return &pluginv1.KycQueryResponse{
		Ok:      true,
		Status:  st,
		Reason:  reason,
		RawJson: raw,
	}, nil
}

func (k *kycServer) setSyncResult(token string, ok bool, reason string, raw string) {
	item := cachedResult{
		status:  "FAILED",
		reason:  strings.TrimSpace(reason),
		rawJSON: strings.TrimSpace(raw),
		expire:  time.Now().Add(30 * time.Minute),
	}
	if ok {
		item.status = "VERIFIED"
		item.reason = ""
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.syncResults == nil {
		k.syncResults = map[string]cachedResult{}
	}
	k.syncResults[token] = item
}

func (k *kycServer) getSyncResult(token string) (cachedResult, bool) {
	k.mu.RLock()
	item, ok := k.syncResults[token]
	k.mu.RUnlock()
	if !ok {
		return cachedResult{}, false
	}
	if time.Now().After(item.expire) {
		k.mu.Lock()
		delete(k.syncResults, token)
		k.mu.Unlock()
		return cachedResult{}, false
	}
	return item, true
}

func doRequest(ctx context.Context, cfg config, endpoint string, params map[string]string) (string, int, string, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://e.mangzhuyun.cn"
	}
	timeout := cfg.TimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	form := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(v) == "" {
			continue
		}
		form.Set(k, strings.TrimSpace(v))
	}
	req, _ := http.NewRequestWithContext(cctx, http.MethodPost, base+endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	raw := strings.TrimSpace(string(body))
	code, msg := parseCodeMsg(raw)
	return raw, code, msg, nil
}

func parseCodeMsg(raw string) (int, string) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return 0, ""
	}
	code := 0
	switch v := doc["code"].(type) {
	case float64:
		code = int(v)
	case int:
		code = v
	}
	msg := strings.TrimSpace(anyToString(doc["msg"]))
	return code, msg
}

func parseTwoFactorResult(code int, msg string, raw string) (bool, string) {
	var doc struct {
		Code int `json:"code"`
		Data struct {
			Result  int    `json:"result"`
			Message string `json:"message"`
		} `json:"data"`
		Result string `json:"result"`
		Msg    string `json:"msg"`
	}
	_ = json.Unmarshal([]byte(raw), &doc)
	if code == 200 && (doc.Data.Result == 1 || strings.Contains(doc.Result, "通过")) {
		return true, ""
	}
	reason := strings.TrimSpace(doc.Data.Message)
	if reason == "" {
		reason = strings.TrimSpace(doc.Msg)
	}
	if reason == "" {
		reason = strings.TrimSpace(msg)
	}
	if reason == "" {
		reason = "two_factor verify failed"
	}
	return false, reason
}

func parseThreeFactorResult(code int, msg string, raw string) (bool, string) {
	if code == 200 {
		return true, ""
	}
	var doc map[string]any
	_ = json.Unmarshal([]byte(raw), &doc)
	reason := strings.TrimSpace(anyToString(doc["msg"]))
	if reason == "" {
		reason = strings.TrimSpace(msg)
	}
	if reason == "" {
		reason = "three_factor verify failed"
	}
	return false, reason
}

func parseFaceStart(code int, msg string, raw string) (token string, redirectURL string, errMsg string) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return "", "", "invalid json response"
	}
	token = strings.TrimSpace(anyToString(doc["token"]))
	redirectURL = strings.TrimSpace(anyToString(doc["url"]))
	if code == 200 && token != "" {
		return token, redirectURL, ""
	}
	errMsg = strings.TrimSpace(anyToString(doc["msg"]))
	if errMsg == "" {
		errMsg = strings.TrimSpace(msg)
	}
	if errMsg == "" {
		errMsg = "face start failed"
	}
	return "", "", errMsg
}

func parseFaceQuery(code int, msg string, raw string, provider string) (statusStr, reason string) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return "PENDING", "invalid json response"
	}
	sm := int64(-1)
	switch v := doc["sm"].(type) {
	case float64:
		sm = int64(v)
	case int64:
		sm = v
	case int:
		sm = int64(v)
	}
	switch provider {
	case "wechat":
		// 0=pending, 1=verified, 3=failed
		switch sm {
		case 1:
			return "VERIFIED", ""
		case 3:
			return "FAILED", nonEmpty(anyToString(doc["msg"]), msg, "face verify failed")
		default:
			return "PENDING", ""
		}
	default:
		// baidu: 0=pending, 2=verified, 1=failed
		switch sm {
		case 2:
			return "VERIFIED", ""
		case 1:
			return "FAILED", nonEmpty(anyToString(doc["msg"]), msg, "face verify failed")
		default:
			return "PENDING", ""
		}
	}
}

func normalizeConfig(cfg *config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://e.mangzhuyun.cn"
	}
	switch strings.TrimSpace(strings.ToLower(cfg.AuthMode)) {
	case "two_factor", "three_factor", "face":
	default:
		cfg.AuthMode = "three_factor"
	}
	switch strings.TrimSpace(strings.ToLower(cfg.FaceProvider)) {
	case "wechat", "baidu":
	default:
		cfg.FaceProvider = "baidu"
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 10
	}
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return fmt.Sprintf("%.0f", t)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func newToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("sync-%d", time.Now().UnixNano())
	}
	return "sync-" + hex.EncodeToString(b[:])
}

func main() {
	core := &coreServer{}
	kyc := &kycServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore: &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyKYC:  &pluginsdk.KycGRPCPlugin{Impl: kyc},
	})
}
