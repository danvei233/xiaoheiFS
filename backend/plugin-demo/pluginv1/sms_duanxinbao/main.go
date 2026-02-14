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
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	Username    string `json:"username"`
	Passwd      string `json:"passwd"`
	GoodsID     string `json:"goods_id"`
	Endpoint    string `json:"endpoint"`
	TimeoutSec  int    `json:"timeout_sec"`
	PasswdIsMD5 bool   `json:"passwd_is_md5"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg      config
	instance string
	client   *http.Client
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "duanxinbao",
		Name:        "Duanxinbao SMS",
		Version:     "1.0.0",
		Description: "短信宝短信发送插件（国内短信 API）",
		Sms:         &pluginv1.SmsCapability{Send: true},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Duanxinbao SMS",
  "type": "object",
  "properties": {
    "username": { "type": "string", "title": "Username" },
    "passwd": { "type": "string", "title": "Password / MD5", "format": "password" },
    "passwd_is_md5": { "type": "boolean", "title": "Password Is MD5", "default": true },
    "goods_id": { "type": "string", "title": "Goods ID (optional)" },
    "endpoint": { "type": "string", "title": "Endpoint", "default": "https://api.smsbao.com/sms" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["username", "passwd"]
}`,
		UiSchema: `{
  "passwd": { "ui:widget": "password", "ui:help": "可填写明文密码或 32 位 MD5 值" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.Passwd) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "username/passwd required"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "invalid config"}, nil
	}
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.Passwd = strings.TrimSpace(cfg.Passwd)
	cfg.GoodsID = strings.TrimSpace(cfg.GoodsID)
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 10
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		cfg.Endpoint = "https://api.smsbao.com/sms"
	}
	if cfg.Username == "" || cfg.Passwd == "" {
		return &pluginv1.InitResponse{Ok: false, Error: "username/passwd required"}, nil
	}
	if !cfg.PasswdIsMD5 {
		cfg.Passwd = md5Hex(cfg.Passwd)
	}
	s.cfg = cfg
	s.instance = strings.TrimSpace(req.GetInstanceId())
	s.client = &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(ctx context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	_, _ = s.Init(ctx, &pluginv1.InitRequest{InstanceId: s.instance, ConfigJson: req.GetConfigJson()})
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

type smsServer struct {
	pluginv1.UnimplementedSmsServiceServer
	core *coreServer
}

func (s *smsServer) Send(ctx context.Context, req *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error) {
	if s.core == nil || s.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if len(req.GetPhones()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "phones required")
	}
	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return nil, status.Error(codes.InvalidArgument, "content required")
	}

	phones := make([]string, 0, len(req.GetPhones()))
	for _, p := range req.GetPhones() {
		p = strings.TrimSpace(p)
		if p != "" {
			phones = append(phones, p)
		}
	}
	if len(phones) == 0 {
		return nil, status.Error(codes.InvalidArgument, "phones required")
	}

	goodsID := strings.TrimSpace(s.core.cfg.GoodsID)
	if strings.TrimSpace(req.GetTemplateId()) != "" {
		goodsID = strings.TrimSpace(req.GetTemplateId())
	}

	q := url.Values{}
	q.Set("u", s.core.cfg.Username)
	q.Set("p", s.core.cfg.Passwd)
	q.Set("m", strings.Join(phones, ","))
	q.Set("c", content)
	if goodsID != "" {
		q.Set("g", goodsID)
	}

	u := strings.TrimRight(s.core.cfg.Endpoint, "?") + "?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "build request failed")
	}
	resp, err := s.core.client.Do(httpReq)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "smsbao request failed: "+sanitizeErr(err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	result := strings.TrimSpace(string(body))
	if resp.StatusCode >= 400 {
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("smsbao http %d: %s", resp.StatusCode, result))
	}
	if result != "0" {
		return nil, status.Error(codes.FailedPrecondition, "smsbao rejected: "+smsbaoError(result))
	}
	return &pluginv1.SendSmsResponse{Ok: true, MessageId: fmt.Sprintf("smsbao-%d", time.Now().UnixNano())}, nil
}

func main() {
	core := &coreServer{}
	sms := &smsServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore: &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeySMS:  &pluginsdk.SmsGRPCPlugin{Impl: sms},
	})
}

func md5Hex(raw string) string {
	sum := md5.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func smsbaoError(code string) string {
	switch strings.TrimSpace(code) {
	case "30":
		return "30 错误密码"
	case "40":
		return "40 账号不存在"
	case "41":
		return "41 余额不足"
	case "43":
		return "43 IP地址限制"
	case "50":
		return "50 内容含有敏感词"
	case "51":
		return "51 手机号码不正确"
	default:
		if strings.TrimSpace(code) == "" {
			return "unknown error"
		}
		return code
	}
}

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	s := strings.ReplaceAll(err.Error(), "\n", " ")
	if len(s) > 400 {
		s = s[:400]
	}
	return s
}
