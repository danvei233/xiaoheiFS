package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v2/client"
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
	SignName        string `json:"sign_name"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	TimeoutSec      int    `json:"timeout_sec"`
}

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg       config
	instance  string
	smsClient *dysmsapi.Client
}

func (s *coreServer) GetManifest(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	_ = ctx
	return &pluginv1.Manifest{
		PluginId:    "alisms",
		Name:        "Aliyun SMS",
		Version:     "1.0.0",
		Description: "Aliyun SMS via official Dysmsapi SDK.",
		Sms:         &pluginv1.SmsCapability{Send: true},
	}, nil
}

func (s *coreServer) GetConfigSchema(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	_ = ctx
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "Aliyun SMS",
  "type": "object",
  "properties": {
    "access_key_id": { "type": "string", "title": "AccessKey ID" },
    "access_key_secret": { "type": "string", "title": "AccessKey Secret", "format": "password" },
    "sign_name": { "type": "string", "title": "Sign Name" },
    "region": { "type": "string", "title": "Region", "default": "cn-hangzhou" },
    "endpoint": { "type": "string", "title": "Endpoint", "default": "dysmsapi.aliyuncs.com" },
    "timeout_sec": { "type": "integer", "title": "Request Timeout (sec)", "default": 10, "minimum": 1, "maximum": 60 }
  },
  "required": ["access_key_id","access_key_secret","sign_name"]
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
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" || strings.TrimSpace(cfg.SignName) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "access_key_id/access_key_secret/sign_name required"}, nil
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
		cfg.Endpoint = "dysmsapi.aliyuncs.com"
	}
	client, err := newAliyunDysmsapiClient(cfg)
	if err != nil {
		return &pluginv1.InitResponse{Ok: false, Error: "init aliyun sms client failed: " + err.Error()}, nil
	}
	s.cfg = cfg
	s.smsClient = client
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

type smsServer struct {
	pluginv1.UnimplementedSmsServiceServer
	core *coreServer
}

func (s *smsServer) Send(ctx context.Context, req *pluginv1.SendSmsRequest) (*pluginv1.SendSmsResponse, error) {
	if s.core == nil || s.core.smsClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "plugin not initialized")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if strings.TrimSpace(req.GetTemplateId()) == "" {
		if strings.TrimSpace(req.GetContent()) != "" {
			return nil, status.Error(codes.InvalidArgument, "aliyun sms requires template_id (content-only not supported)")
		}
		return nil, status.Error(codes.InvalidArgument, "template_id required")
	}
	if len(req.GetPhones()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "phones required")
	}
	varsJSON, err := json.Marshal(req.GetVars())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid vars")
	}
	phones := strings.Join(req.GetPhones(), ",")

	runtime := &util.RuntimeOptions{}
	runtime.SetReadTimeout(s.core.cfg.TimeoutSec * 1000)
	runtime.SetConnectTimeout(s.core.cfg.TimeoutSec * 1000)
	resp, err := s.core.smsClient.SendSmsWithOptions(&dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phones),
		SignName:      tea.String(strings.TrimSpace(s.core.cfg.SignName)),
		TemplateCode:  tea.String(strings.TrimSpace(req.GetTemplateId())),
		TemplateParam: tea.String(string(varsJSON)),
	}, runtime)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "aliyun sms send failed: "+sanitizeErr(err))
	}
	if resp == nil || resp.Body == nil {
		return nil, status.Error(codes.Unavailable, "aliyun sms empty response")
	}
	if tea.StringValue(resp.Body.Code) != "OK" {
		msg := tea.StringValue(resp.Body.Message)
		if msg == "" {
			msg = tea.StringValue(resp.Body.Code)
		}
		return nil, status.Error(codes.FailedPrecondition, "aliyun sms rejected: "+msg)
	}
	return &pluginv1.SendSmsResponse{
		Ok:        true,
		MessageId: tea.StringValue(resp.Body.BizId),
	}, nil
}

func main() {
	core := &coreServer{}
	sms := &smsServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore: &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeySMS:  &pluginsdk.SmsGRPCPlugin{Impl: sms},
	})
}

func newAliyunDysmsapiClient(cfg config) (*dysmsapi.Client, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, fmt.Errorf("access_key_id/access_key_secret required")
	}
	conf := &openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(cfg.AccessKeyID)),
		AccessKeySecret: tea.String(strings.TrimSpace(cfg.AccessKeySecret)),
		Endpoint:        tea.String(strings.TrimSpace(cfg.Endpoint)),
		RegionId:        tea.String(strings.TrimSpace(cfg.Region)),
	}
	return dysmsapi.NewClient(conf)
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
