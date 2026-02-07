package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	cloudauth "github.com/alibabacloud-go/cloudauth-20200618/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	util "github.com/alibabacloud-go/tea-utils/service"
	"github.com/alibabacloud-go/tea/tea"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type config struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	SceneID         int64  `json:"scene_id"`
	Mode            string `json:"mode"`
	H5BaseURL       string `json:"h5_base_url"`
	TimeoutSec      int    `json:"timeout_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg      config
	instance string
	client   *cloudauth.Client
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "aliyun_kyc",
		Name:        "Aliyun CloudAuth (eKYC/Real-Name)",
		Version:     "1.0.0",
		Description: "Aliyun CloudAuth InitSmartVerify/DescribeSmartVerify flow.",
		Kyc:         &pluginv1.KycCapability{Start: true, QueryResult: true},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Aliyun CloudAuth",
  "type": "object",
  "properties": {
    "access_key_id": { "type": "string", "title": "AccessKey ID" },
    "access_key_secret": { "type": "string", "title": "AccessKey Secret", "format": "password" },
    "region": { "type": "string", "title": "Region", "default": "cn-hangzhou" },
    "endpoint": { "type": "string", "title": "Endpoint", "default": "cloudauth.cn-hangzhou.aliyuncs.com" },
    "scene_id": { "type": "integer", "title": "SceneId" },
    "mode": { "type": "string", "title": "Mode", "default": "FULL" },
    "h5_base_url": { "type": "string", "title": "H5 Base URL (optional)", "default": "" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["access_key_id","access_key_secret","scene_id"]
}`,
		UiSchema: `{
  "access_key_secret": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(ctx context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	_ = ctx
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" || cfg.SceneID <= 0 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "access_key_id/access_key_secret/scene_id required"}, nil
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
		cfg.Region = "cn-hangzhou"
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		cfg.Endpoint = "cloudauth.cn-hangzhou.aliyuncs.com"
	}
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = "FULL"
	}
	client, err := newAliyunCloudAuthClient(cfg)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "init aliyun cloudauth client failed: " + err.Error()}, nil
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
	name := strings.TrimSpace(req.GetParams()["name"])
	idNumber := strings.TrimSpace(req.GetParams()["id_number"])
	metaInfo := strings.TrimSpace(req.GetParams()["meta_info"])
	callbackURL := strings.TrimSpace(req.GetParams()["callback_url"])
	outerOrderNo := strings.TrimSpace(req.GetParams()["outer_order_no"])
	if name == "" || idNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "params.name/params.id_number required")
	}
	if outerOrderNo == "" {
		outerOrderNo = fmt.Sprintf("KYC-%s-%d", strings.TrimSpace(req.GetUserId()), time.Now().Unix())
	}
	runtime := &util.RuntimeOptions{}
	runtime.SetReadTimeout(k.core.cfg.TimeoutSec * 1000)
	runtime.SetConnectTimeout(k.core.cfg.TimeoutSec * 1000)
	initReq := &cloudauth.InitSmartVerifyRequest{}
	initReq.SetSceneId(k.core.cfg.SceneID)
	initReq.SetOuterOrderNo(outerOrderNo)
	initReq.SetMode(k.core.cfg.Mode)
	initReq.SetCertType("IDENTITY_CARD")
	initReq.SetCertName(name)
	initReq.SetCertNo(idNumber)
	if strings.TrimSpace(req.GetUserId()) != "" {
		initReq.SetUserId(strings.TrimSpace(req.GetUserId()))
	}
	if metaInfo != "" {
		initReq.SetMetaInfo(metaInfo)
	}
	if callbackURL != "" {
		initReq.SetCallbackUrl(callbackURL)
	}
	resp, err := k.core.client.InitSmartVerifyWithOptions(initReq, runtime)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "aliyun init_smart_verify failed: "+sanitizeErr(err))
	}
	if resp == nil || resp.Body == nil || resp.Body.ResultObject == nil || resp.Body.ResultObject.CertifyId == nil {
		return nil, status.Error(codes.Unavailable, "aliyun empty response")
	}
	certifyID := strings.TrimSpace(tea.StringValue(resp.Body.ResultObject.CertifyId))
	u := ""
	if strings.TrimSpace(k.core.cfg.H5BaseURL) != "" && certifyID != "" {
		u = strings.TrimRight(strings.TrimSpace(k.core.cfg.H5BaseURL), "?&") + "?certifyId=" + certifyID
	}
	next := "query_result"
	if u != "" {
		next = "redirect"
	}
	return &pluginv1.KycStartResponse{
		Ok:       true,
		Token:    certifyID,
		Url:      u,
		NextStep: next,
	}, nil
}

func (k *kycServer) QueryResult(ctx context.Context, req *pluginv1.KycQueryRequest) (*pluginv1.KycQueryResponse, error) {
	if k.core == nil || k.core.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	if req == nil || strings.TrimSpace(req.GetToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "token required")
	}
	runtime := &util.RuntimeOptions{}
	runtime.SetReadTimeout(k.core.cfg.TimeoutSec * 1000)
	runtime.SetConnectTimeout(k.core.cfg.TimeoutSec * 1000)
	q := &cloudauth.DescribeSmartVerifyRequest{}
	q.SetSceneId(k.core.cfg.SceneID)
	q.SetCertifyId(strings.TrimSpace(req.GetToken()))
	resp, err := k.core.client.DescribeSmartVerifyWithOptions(q, runtime)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "aliyun describe_smart_verify failed: "+sanitizeErr(err))
	}
	raw, _ := json.Marshal(resp)
	statusStr := "PENDING"
	reason := ""
	if resp != nil && resp.Body != nil && resp.Body.ResultObject != nil && resp.Body.ResultObject.Passed != nil {
		switch strings.ToUpper(strings.TrimSpace(tea.StringValue(resp.Body.ResultObject.Passed))) {
		case "T", "TRUE", "Y", "YES":
			statusStr = "VERIFIED"
		case "F", "FALSE", "N", "NO":
			statusStr = "FAILED"
			reason = strings.TrimSpace(tea.StringValue(resp.Body.ResultObject.SubCode))
			if reason == "" {
				reason = strings.TrimSpace(tea.StringValue(resp.Body.Message))
			}
		}
	}
	return &pluginv1.KycQueryResponse{
		Ok:      true,
		Status:  statusStr,
		Reason:  reason,
		RawJson: string(raw),
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

func newAliyunCloudAuthClient(cfg config) (*cloudauth.Client, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, errors.New("access_key_id/access_key_secret required")
	}
	conf := &openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(cfg.AccessKeyID)),
		AccessKeySecret: tea.String(strings.TrimSpace(cfg.AccessKeySecret)),
		Endpoint:        tea.String(strings.TrimSpace(cfg.Endpoint)),
		RegionId:        tea.String(strings.TrimSpace(cfg.Region)),
	}
	return cloudauth.NewClient(conf)
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
