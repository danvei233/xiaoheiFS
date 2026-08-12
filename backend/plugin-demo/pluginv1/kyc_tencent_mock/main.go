package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	faceid "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/faceid/v20180301"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	SecretID   string `json:"secret_id"`
	SecretKey  string `json:"secret_key"`
	Region     string `json:"region"`
	RuleID     string `json:"rule_id"`
	TimeoutSec int    `json:"timeout_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg      config
	instance string
	client   *faceid.Client
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "tencent_kyc",
		Name:        "Tencent Cloud FaceID",
		Version:     "1.0.0",
		Description: "Tencent Cloud FaceID DetectAuth/GetDetectInfo flow (H5).",
		Kyc:         &pluginv1.KycCapability{Start: true, QueryResult: true},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Tencent Cloud FaceID",
  "type": "object",
  "properties": {
    "secret_id": { "type": "string", "title": "SecretId" },
    "secret_key": { "type": "string", "title": "SecretKey", "format": "password" },
    "region": { "type": "string", "title": "Region", "default": "ap-guangzhou" },
    "rule_id": { "type": "string", "title": "RuleId" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["secret_id","secret_key","rule_id"]
}`,
		UiSchema: `{
  "secret_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.SecretID) == "" || strings.TrimSpace(cfg.SecretKey) == "" || strings.TrimSpace(cfg.RuleID) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "secret_id/secret_key/rule_id required"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(ctx context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "invalid config"}, nil
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 10
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = "ap-guangzhou"
	}
	cred := common.NewCredential(strings.TrimSpace(cfg.SecretID), strings.TrimSpace(cfg.SecretKey))
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqTimeout = cfg.TimeoutSec
	client, err := faceid.NewClient(cred, cfg.Region, cpf)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "init tencent faceid client failed: " + err.Error()}, nil
	}
	s.cfg = cfg
	s.client = client
	s.instance = req.GetInstanceId()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(ctx context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	_, err := s.Init(ctx, &pluginv1.InitRequest{InstanceId: s.instance, ConfigJson: req.GetConfigJson()})
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
}

func (k *kycServer) Start(ctx context.Context, req *pluginv1.KycStartRequest) (*pluginv1.KycStartResponse, error) {
	if k.core == nil || k.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	idNumber := strings.TrimSpace(req.GetParams()["id_number"])
	name := strings.TrimSpace(req.GetParams()["name"])
	redirectURL := strings.TrimSpace(req.GetParams()["redirect_url"])
	extra := strings.TrimSpace(req.GetParams()["extra"])
	imageBase64 := strings.TrimSpace(req.GetParams()["image_base64"])
	if idNumber == "" || name == "" {
		return nil, status.Error(codes.InvalidArgument, "params.name/params.id_number required")
	}
	// Only allow absolute http(s) redirect URLs; an end-user-injectable
	// redirect_url would otherwise let the provider send the certified result
	// (with PII) to an arbitrary third party (M12).
	if redirectURL != "" && !validCallbackURL(redirectURL) {
		return nil, status.Error(codes.InvalidArgument, "invalid redirect_url: must be an absolute http(s) URL with a host")
	}
	r := faceid.NewDetectAuthRequest()
	r.RuleId = common.StringPtr(strings.TrimSpace(k.core.cfg.RuleID))
	r.IdCard = common.StringPtr(idNumber)
	r.Name = common.StringPtr(name)
	if redirectURL != "" {
		r.RedirectUrl = common.StringPtr(redirectURL)
	}
	if extra != "" {
		r.Extra = common.StringPtr(extra)
	} else if strings.TrimSpace(req.GetUserId()) != "" {
		r.Extra = common.StringPtr("user_id=" + strings.TrimSpace(req.GetUserId()))
	}
	if imageBase64 != "" {
		r.ImageBase64 = common.StringPtr(imageBase64)
	}

	resp, err := k.core.client.DetectAuth(r)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "tencent faceid detect_auth failed: "+sanitizeErr(err))
	}
	if resp == nil || resp.Response == nil || resp.Response.BizToken == nil {
		return nil, status.Error(codes.Unavailable, "tencent faceid empty response")
	}
	url := ""
	if resp.Response.Url != nil {
		url = *resp.Response.Url
	}
	return &pluginv1.KycStartResponse{
		Ok:       true,
		Token:    strings.TrimSpace(*resp.Response.BizToken),
		Url:      url,
		NextStep: "redirect",
	}, nil
}

func (k *kycServer) QueryResult(ctx context.Context, req *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
	if k.core == nil || k.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	if req == nil || strings.TrimSpace(req.GetToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	r := faceid.NewGetDetectInfoRequest()
	r.BizToken = common.StringPtr(strings.TrimSpace(req.GetToken()))
	resp, err := k.core.client.GetDetectInfo(r)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "tencent faceid get_detect_info failed: "+sanitizeErr(err))
	}
	if resp == nil || resp.Response == nil || resp.Response.DetectInfo == nil {
		return nil, status.Error(codes.Unavailable, "tencent faceid empty response")
	}
	raw := strings.TrimSpace(*resp.Response.DetectInfo)
	statusStr, reason := parseTencentDetectInfo(raw)
	return &pluginv1.KycQueryResponse{
		Ok:      true,
		Status:  statusStr,
		Reason:  reason,
		RawJson: raw,
	}, nil
}

func main() {
	core := &coreServer{}
	kyc := &kycServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore: &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyKYC:  &pluginsdk.KycGRPCPlugin{Impl: kyc},
	})
}

// validCallbackURL reports whether a callback/redirect URL is acceptable to
// forward to the KYC provider: an absolute http(s) URL with a non-empty host.
// It rejects javascript:/data:/file: and protocol-relative URLs.
func validCallbackURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	return strings.TrimSpace(u.Host) != ""
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

func parseTencentDetectInfo(raw string) (status string, reason string) {
	if strings.TrimSpace(raw) == "" {
		return "PENDING", ""
	}
	var doc struct {
		Text struct {
			ErrCode *int   `json:"ErrCode"`
			ErrMsg  string `json:"ErrMsg"`
		} `json:"Text"`
	}
	_ = json.Unmarshal([]byte(raw), &doc)
	if doc.Text.ErrCode == nil {
		return "PENDING", ""
	}
	if *doc.Text.ErrCode == 0 {
		return "VERIFIED", ""
	}
	r := strings.TrimSpace(doc.Text.ErrMsg)
	if r == "" {
		r = fmt.Sprintf("ErrCode=%d", *doc.Text.ErrCode)
	}
	return "FAILED", r
}
