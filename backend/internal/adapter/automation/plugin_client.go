package automation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type PluginInstanceClient struct {
	mgr        *plugins.Manager
	pluginID   string
	instanceID string
	timeout    time.Duration
	settings   usecase.SettingsRepository
	autoLogs   usecase.AutomationLogRepository
}

func NewPluginInstanceClient(mgr *plugins.Manager, pluginID, instanceID string, settings usecase.SettingsRepository, autoLogs usecase.AutomationLogRepository) *PluginInstanceClient {
	return &PluginInstanceClient{
		mgr:        mgr,
		pluginID:   strings.TrimSpace(pluginID),
		instanceID: strings.TrimSpace(instanceID),
		timeout:    12 * time.Second,
		settings:   settings,
		autoLogs:   autoLogs,
	}
}

func (c *PluginInstanceClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *PluginInstanceClient) client(ctx context.Context) (pluginv1.AutomationServiceClient, error) {
	if c.mgr == nil {
		return nil, errors.New("plugin manager missing")
	}
	cli, _, err := c.mgr.GetAutomationClient(ctx, c.pluginID, c.instanceID)
	return cli, err
}

type automationLogConfig struct {
	debugEnabled  bool
	retentionDays int
}

func (c *PluginInstanceClient) loadLogConfig(ctx context.Context) automationLogConfig {
	cfg := automationLogConfig{}
	if c.settings == nil {
		return cfg
	}
	if setting, err := c.settings.GetSetting(ctx, "debug_enabled"); err == nil && setting.ValueJSON != "" {
		cfg.debugEnabled = strings.ToLower(setting.ValueJSON) == "true"
	}
	if setting, err := c.settings.GetSetting(ctx, "automation_log_retention_days"); err == nil && setting.ValueJSON != "" {
		if v, err := strconv.Atoi(setting.ValueJSON); err == nil && v > 0 {
			cfg.retentionDays = v
		}
	}
	return cfg
}

func (c *PluginInstanceClient) logRPC(ctx context.Context, action string, req any, resp any, duration time.Duration, err error) {
	if c.autoLogs == nil {
		return
	}
	cfg := c.loadLogConfig(ctx)
	if !cfg.debugEnabled && err == nil {
		return
	}
	trace, _ := usecase.GetAutomationLogContext(ctx)
	if cfg.retentionDays > 0 {
		before := time.Now().AddDate(0, 0, -cfg.retentionDays)
		_ = c.autoLogs.PurgeAutomationLogs(ctx, before)
	}
	requestMeta := c.grpcRequestMeta(action)
	reqPayload := map[string]any{
		"method":  "GRPC",
		"url":     requestMeta.URL,
		"headers": requestMeta.Headers,
		"body":    sanitizePayload(toLogPayload(req)),
	}
	respPayload := map[string]any{
		"status":      200,
		"headers":     map[string]string{},
		"duration_ms": duration.Milliseconds(),
	}
	if err != nil {
		respPayload["status"] = 500
		respPayload["body"] = err.Error()
		respPayload["format"] = "text"
		if traceReq, traceResp, traceAction, traceMsg, ok := extractHTTPTrace(err); ok {
			if traceReq != nil {
				reqPayload = traceReq
			}
			reqPayload = mergeRequestMeta(reqPayload, requestMeta)
			if traceResp != nil {
				respPayload = traceResp
			}
			if strings.TrimSpace(traceAction) != "" {
				action = traceAction
			}
			if strings.TrimSpace(traceMsg) != "" {
				err = errors.New(traceMsg)
			}
		}
	} else {
		body := sanitizePayload(toLogPayload(resp))
		respPayload["body"] = body
		respPayload["format"] = "json"
		respPayload["body_json"] = body
	}
	logEntry := domain.AutomationLog{
		OrderID:      trace.OrderID,
		OrderItemID:  trace.OrderItemID,
		Action:       action,
		RequestJSON:  mustJSON(reqPayload),
		ResponseJSON: mustJSON(respPayload),
		Success:      err == nil,
		Message:      messageFromErr(err),
	}
	_ = c.autoLogs.CreateAutomationLog(ctx, &logEntry)
}

type grpcRequestMetadata struct {
	URL     string
	Headers map[string]string
}

func (c *PluginInstanceClient) grpcRequestMeta(action string) grpcRequestMetadata {
	pluginID := strings.TrimSpace(c.pluginID)
	if pluginID == "" {
		pluginID = "-"
	}
	instanceID := strings.TrimSpace(c.instanceID)
	if instanceID == "" {
		instanceID = "-"
	}
	rpcAction := strings.TrimSpace(action)
	if rpcAction == "" {
		rpcAction = "unknown"
	}
	return grpcRequestMetadata{
		URL: fmt.Sprintf("grpc://automation/%s/%s/%s", pluginID, instanceID, rpcAction),
		Headers: map[string]string{
			"x-transport":          "grpc",
			"x-plugin-category":    "automation",
			"x-plugin-id":          pluginID,
			"x-plugin-instance-id": instanceID,
			"x-rpc-action":         rpcAction,
		},
	}
}

func mergeRequestMeta(payload map[string]any, meta grpcRequestMetadata) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	method := strings.ToUpper(strings.TrimSpace(fmt.Sprint(payload["method"])))
	if method == "" || method == "<NIL>" {
		payload["method"] = "GRPC"
	}
	urlText := strings.TrimSpace(fmt.Sprint(payload["url"]))
	if urlText == "" || strings.EqualFold(urlText, "<nil>") {
		payload["url"] = meta.URL
	}
	headers := map[string]string{}
	switch raw := payload["headers"].(type) {
	case map[string]string:
		for k, v := range raw {
			headers[k] = v
		}
	case map[string]any:
		for k, v := range raw {
			headers[k] = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	for k, v := range meta.Headers {
		if strings.TrimSpace(v) == "" {
			continue
		}
		headers[k] = v
	}
	payload["headers"] = headers
	return payload
}

func extractHTTPTrace(err error) (map[string]any, map[string]any, string, string, bool) {
	if err == nil {
		return nil, nil, "", "", false
	}
	message := err.Error()
	const marker = "http_trace="
	index := strings.LastIndex(message, marker)
	if index < 0 {
		return nil, nil, "", "", false
	}
	raw := strings.TrimSpace(message[index+len(marker):])
	if raw == "" {
		return nil, nil, "", "", false
	}
	if decoded, decodeErr := base64.StdEncoding.DecodeString(raw); decodeErr == nil && len(decoded) > 0 {
		raw = string(decoded)
	}
	var trace struct {
		Action   string         `json:"action"`
		Request  map[string]any `json:"request"`
		Response map[string]any `json:"response"`
		Success  bool           `json:"success"`
		Message  string         `json:"message"`
	}
	if json.Unmarshal([]byte(raw), &trace) != nil {
		return nil, nil, "", "", false
	}
	return trace.Request, trace.Response, trace.Action, trace.Message, true
}

func messageFromErr(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func mustJSON(payload any) string {
	if payload == nil {
		return ""
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func toLogPayload(payload any) any {
	if payload == nil {
		return nil
	}
	if msg, ok := payload.(proto.Message); ok {
		raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(msg)
		if err != nil {
			return map[string]any{"_error": err.Error()}
		}
		var out any
		if err := json.Unmarshal(raw, &out); err == nil {
			return out
		}
		return string(raw)
	}
	return payload
}

func sanitizePayload(payload any) any {
	switch v := payload.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			if isSensitiveKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = sanitizePayload(val)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, sanitizePayload(item))
		}
		return out
	default:
		return payload
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "api_key", "apikey", "secret", "token", "access_key", "access_key_id", "access_key_secret":
		return true
	default:
		return strings.Contains(normalized, "secret")
	}
}

func (c *PluginInstanceClient) call(ctx context.Context, action string, req proto.Message, fn func(context.Context, pluginv1.AutomationServiceClient) (proto.Message, error)) (proto.Message, error) {
	cli, err := c.client(ctx)
	if err != nil {
		c.logRPC(ctx, action, req, nil, 0, err)
		return nil, err
	}
	cctx, cancel := c.withTimeout(ctx)
	defer cancel()
	start := time.Now()
	resp, err := fn(cctx, cli)
	err = mapUnimplemented(err)
	err = mapRPCBusinessError(err)
	c.logRPC(ctx, action, req, resp, time.Since(start), err)
	return resp, err
}

func mapUnimplemented(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	if st.Code() == codes.Unimplemented {
		msg := strings.TrimSpace(st.Message())
		if msg == "" {
			msg = "not supported"
		}
		return fmt.Errorf("%w: %s", usecase.ErrNotSupported, msg)
	}
	return err
}

func mapRPCBusinessError(err error) error {
	if err == nil {
		return nil
	}
	msg := extractRPCErrorMessage(err.Error())
	if strings.TrimSpace(msg) == "" || msg == err.Error() {
		return err
	}
	return errors.New(msg)
}

func extractRPCErrorMessage(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed := parseRPCErrorJSON(trimmed); parsed != "" {
		return parsed
	}
	if idx := strings.Index(trimmed, "{"); idx >= 0 {
		if parsed := parseRPCErrorJSON(trimmed[idx:]); parsed != "" {
			return parsed
		}
	}
	re := regexp.MustCompile(`msg":"([^"]+)"`)
	matches := re.FindStringSubmatch(trimmed)
	if len(matches) == 2 && strings.TrimSpace(matches[1]) != "" {
		return matches[1]
	}
	return ""
}

func parseRPCErrorJSON(raw string) string {
	var obj map[string]any
	if json.Unmarshal([]byte(raw), &obj) != nil {
		return ""
	}
	for _, key := range []string{"msg", "message", "error"} {
		if v, ok := obj[key]; ok {
			msg := strings.TrimSpace(fmt.Sprint(v))
			if msg != "" && msg != "<nil>" {
				return msg
			}
		}
	}
	if other, ok := obj["other"].(map[string]any); ok {
		if v, ok := other["msg"]; ok {
			msg := strings.TrimSpace(fmt.Sprint(v))
			if msg != "" && msg != "<nil>" {
				return msg
			}
		}
	}
	return ""
}

func ensureOpOK(resp proto.Message) error {
	empty, ok := resp.(*pluginv1.Empty)
	if !ok || empty == nil {
		return nil
	}
	status := strings.ToLower(strings.TrimSpace(empty.GetStatus()))
	if status == "" || status == "ok" || status == "success" || status == "succeeded" || status == "1" || status == "200" {
		return nil
	}
	msg := strings.TrimSpace(empty.GetMsg())
	if msg == "" {
		msg = "operation failed"
	}
	if other := strings.TrimSpace(empty.GetOther()); other != "" {
		return fmt.Errorf("%s (%s)", msg, other)
	}
	return errors.New(msg)
}

func (c *PluginInstanceClient) CreateHost(ctx context.Context, req usecase.AutomationCreateHostRequest) (usecase.AutomationCreateHostResult, error) {
	pb := &pluginv1.CreateInstanceRequest{
		LineId:        req.LineID,
		Os:            req.OS,
		Name:          req.HostName,
		Password:      req.SysPwd,
		VncPassword:   req.VNCPwd,
		ExpireAtUnix:  req.ExpireTime.Unix(),
		PortNum:       int32(req.PortNum),
		Cpu:           int32(req.CPU),
		MemoryGb:      int32(req.MemoryGB),
		DiskGb:        int32(req.DiskGB),
		BandwidthMbps: int32(req.Bandwidth),
	}
	respAny, err := c.call(ctx, "automation.CreateInstance", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.CreateInstance(cctx, pb)
	})
	if err != nil {
		return usecase.AutomationCreateHostResult{}, err
	}
	resp := respAny.(*pluginv1.CreateInstanceResponse)
	return usecase.AutomationCreateHostResult{HostID: resp.GetInstanceId(), Raw: map[string]any{"instance_id": resp.GetInstanceId()}}, nil
}

func (c *PluginInstanceClient) GetHostInfo(ctx context.Context, hostID int64) (usecase.AutomationHostInfo, error) {
	pb := &pluginv1.GetInstanceRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetInstance", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetInstance(cctx, pb)
	})
	if err != nil {
		return usecase.AutomationHostInfo{}, err
	}
	resp := respAny.(*pluginv1.GetInstanceResponse)
	inst := resp.GetInstance()
	var expire *time.Time
	if inst.GetExpireAtUnix() > 0 {
		t := time.Unix(inst.GetExpireAtUnix(), 0)
		expire = &t
	}
	return usecase.AutomationHostInfo{
		HostID:        inst.GetId(),
		HostName:      inst.GetName(),
		State:         int(inst.GetState()),
		CPU:           int(inst.GetCpu()),
		MemoryGB:      int(inst.GetMemoryGb()),
		DiskGB:        int(inst.GetDiskGb()),
		Bandwidth:     int(inst.GetBandwidthMbps()),
		PanelPassword: inst.GetPanelPassword(),
		VNCPassword:   inst.GetVncPassword(),
		OSPassword:    inst.GetOsPassword(),
		RemoteIP:      inst.GetRemoteIp(),
		ExpireAt:      expire,
	}, nil
}

func (c *PluginInstanceClient) ListHostSimple(ctx context.Context, searchTag string) ([]usecase.AutomationHostSimple, error) {
	pb := &pluginv1.ListInstancesSimpleRequest{SearchTag: strings.TrimSpace(searchTag)}
	respAny, err := c.call(ctx, "automation.ListInstancesSimple", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListInstancesSimple(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListInstancesSimpleResponse)
	out := make([]usecase.AutomationHostSimple, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationHostSimple{ID: it.GetId(), HostName: it.GetName(), IP: it.GetIp()})
	}
	return out, nil
}

func (c *PluginInstanceClient) ElasticUpdate(ctx context.Context, req usecase.AutomationElasticUpdateRequest) error {
	pb := &pluginv1.ElasticUpdateRequest{InstanceId: req.HostID}
	if req.CPU != nil {
		pb.Cpu = ptrInt32(int32(*req.CPU))
	}
	if req.MemoryGB != nil {
		pb.MemoryGb = ptrInt32(int32(*req.MemoryGB))
	}
	if req.DiskGB != nil {
		pb.DiskGb = ptrInt32(int32(*req.DiskGB))
	}
	if req.Bandwidth != nil {
		pb.BandwidthMbps = ptrInt32(int32(*req.Bandwidth))
	}
	if req.PortNum != nil {
		pb.PortNum = ptrInt32(int32(*req.PortNum))
	}
	respAny, err := c.call(ctx, "automation.ElasticUpdate", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ElasticUpdate(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func ptrInt32(v int32) *int32 { return &v }

func (c *PluginInstanceClient) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	pb := &pluginv1.RenewRequest{InstanceId: hostID, NextDueAtUnix: nextDueDate.Unix()}
	respAny, err := c.call(ctx, "automation.Renew", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Renew(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) LockHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.LockRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Lock", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Lock(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) UnlockHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.UnlockRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Unlock", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Unlock(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.DestroyRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Destroy", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Destroy(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) StartHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.StartRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Start", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Start(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ShutdownHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.ShutdownRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Shutdown", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Shutdown(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) RebootHost(ctx context.Context, hostID int64) error {
	pb := &pluginv1.RebootRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.Reboot", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Reboot(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	pb := &pluginv1.RebuildRequest{InstanceId: hostID, ImageId: templateID, Password: password}
	respAny, err := c.call(ctx, "automation.Rebuild", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.Rebuild(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	pb := &pluginv1.ResetPasswordRequest{InstanceId: hostID, Password: password}
	respAny, err := c.call(ctx, "automation.ResetPassword", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ResetPassword(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ListSnapshots(ctx context.Context, hostID int64) ([]usecase.AutomationSnapshot, error) {
	pb := &pluginv1.ListSnapshotsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListSnapshots", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListSnapshots(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListSnapshotsResponse)
	out := make([]usecase.AutomationSnapshot, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationSnapshot{
			"id":              it.GetId(),
			"name":            it.GetName(),
			"created_at_unix": it.GetCreatedAtUnix(),
			"created_at":      time.Unix(it.GetCreatedAtUnix(), 0).Format(time.RFC3339),
			"state":           int(it.GetState()),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) CreateSnapshot(ctx context.Context, hostID int64) error {
	pb := &pluginv1.CreateSnapshotRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.CreateSnapshot", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.CreateSnapshot(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	pb := &pluginv1.DeleteSnapshotRequest{InstanceId: hostID, SnapshotId: snapshotID}
	respAny, err := c.call(ctx, "automation.DeleteSnapshot", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteSnapshot(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	pb := &pluginv1.RestoreSnapshotRequest{InstanceId: hostID, SnapshotId: snapshotID}
	respAny, err := c.call(ctx, "automation.RestoreSnapshot", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.RestoreSnapshot(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ListBackups(ctx context.Context, hostID int64) ([]usecase.AutomationBackup, error) {
	pb := &pluginv1.ListBackupsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListBackups", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListBackups(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListBackupsResponse)
	out := make([]usecase.AutomationBackup, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationBackup{
			"id":              it.GetId(),
			"name":            it.GetName(),
			"created_at_unix": it.GetCreatedAtUnix(),
			"created_at":      time.Unix(it.GetCreatedAtUnix(), 0).Format(time.RFC3339),
			"state":           int(it.GetState()),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) CreateBackup(ctx context.Context, hostID int64) error {
	pb := &pluginv1.CreateBackupRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.CreateBackup", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.CreateBackup(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	pb := &pluginv1.DeleteBackupRequest{InstanceId: hostID, BackupId: backupID}
	respAny, err := c.call(ctx, "automation.DeleteBackup", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteBackup(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	pb := &pluginv1.RestoreBackupRequest{InstanceId: hostID, BackupId: backupID}
	respAny, err := c.call(ctx, "automation.RestoreBackup", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.RestoreBackup(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ListFirewallRules(ctx context.Context, hostID int64) ([]usecase.AutomationFirewallRule, error) {
	pb := &pluginv1.ListFirewallRulesRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListFirewallRules", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListFirewallRules(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListFirewallRulesResponse)
	out := make([]usecase.AutomationFirewallRule, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationFirewallRule{
			"id":        it.GetId(),
			"direction": it.GetDirection(),
			"protocol":  it.GetProtocol(),
			"method":    it.GetMethod(),
			"port":      it.GetPort(),
			"ip":        it.GetIp(),
			"priority":  int(it.GetPriority()),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) AddFirewallRule(ctx context.Context, req usecase.AutomationFirewallRuleCreate) error {
	pb := &pluginv1.AddFirewallRuleRequest{
		InstanceId: req.HostID,
		Direction:  req.Direction,
		Protocol:   req.Protocol,
		Method:     req.Method,
		Port:       req.Port,
		Ip:         req.IP,
		Priority:   int32(req.Priority),
	}
	respAny, err := c.call(ctx, "automation.AddFirewallRule", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.AddFirewallRule(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	pb := &pluginv1.DeleteFirewallRuleRequest{InstanceId: hostID, RuleId: ruleID}
	respAny, err := c.call(ctx, "automation.DeleteFirewallRule", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteFirewallRule(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ListPortMappings(ctx context.Context, hostID int64) ([]usecase.AutomationPortMapping, error) {
	pb := &pluginv1.ListPortMappingsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListPortMappings", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListPortMappings(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListPortMappingsResponse)
	out := make([]usecase.AutomationPortMapping, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationPortMapping{
			"id":    it.GetId(),
			"name":  it.GetName(),
			"sport": it.GetSport(),
			"dport": it.GetDport(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) AddPortMapping(ctx context.Context, req usecase.AutomationPortMappingCreate) error {
	pb := &pluginv1.AddPortMappingRequest{
		InstanceId: req.HostID,
		Name:       req.Name,
		Sport:      req.Sport,
		Dport:      req.Dport,
	}
	respAny, err := c.call(ctx, "automation.AddPortMapping", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.AddPortMapping(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	pb := &pluginv1.DeletePortMappingRequest{InstanceId: hostID, MappingId: mappingID}
	respAny, err := c.call(ctx, "automation.DeletePortMapping", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeletePortMapping(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	pb := &pluginv1.FindPortCandidatesRequest{InstanceId: hostID, Keywords: strings.TrimSpace(keywords)}
	respAny, err := c.call(ctx, "automation.FindPortCandidates", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.FindPortCandidates(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.FindPortCandidatesResponse)
	return resp.GetPorts(), nil
}

func (c *PluginInstanceClient) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	pb := &pluginv1.GetPanelURLRequest{InstanceName: hostName, PanelPassword: panelPassword}
	respAny, err := c.call(ctx, "automation.GetPanelURL", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetPanelURL(cctx, pb)
	})
	if err != nil {
		return "", err
	}
	resp := respAny.(*pluginv1.GetPanelURLResponse)
	return resp.GetUrl(), nil
}

func (c *PluginInstanceClient) ListAreas(ctx context.Context) ([]usecase.AutomationArea, error) {
	pb := &pluginv1.Empty{}
	respAny, err := c.call(ctx, "automation.ListAreas", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListAreas(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListAreasResponse)
	out := make([]usecase.AutomationArea, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationArea{ID: it.GetId(), Name: it.GetName(), State: int(it.GetState())})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListImages(ctx context.Context, lineID int64) ([]usecase.AutomationImage, error) {
	pb := &pluginv1.ListImagesRequest{LineId: lineID}
	respAny, err := c.call(ctx, "automation.ListImages", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListImages(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListImagesResponse)
	out := make([]usecase.AutomationImage, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationImage{ImageID: it.GetId(), Name: it.GetName(), Type: it.GetType()})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListLines(ctx context.Context) ([]usecase.AutomationLine, error) {
	pb := &pluginv1.Empty{}
	respAny, err := c.call(ctx, "automation.ListLines", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListLines(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListLinesResponse)
	out := make([]usecase.AutomationLine, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationLine{ID: it.GetId(), Name: it.GetName(), AreaID: it.GetAreaId(), State: int(it.GetState())})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListProducts(ctx context.Context, lineID int64) ([]usecase.AutomationProduct, error) {
	pb := &pluginv1.ListPackagesRequest{LineId: lineID}
	respAny, err := c.call(ctx, "automation.ListPackages", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListPackages(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListPackagesResponse)
	out := make([]usecase.AutomationProduct, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, usecase.AutomationProduct{
			ID:        it.GetId(),
			Name:      it.GetName(),
			CPU:       int(it.GetCpu()),
			MemoryGB:  int(it.GetMemoryGb()),
			DiskGB:    int(it.GetDiskGb()),
			Bandwidth: int(it.GetBandwidthMbps()),
			Price:     it.GetMonthlyPrice(),
			PortNum:   int(it.GetPortNum()),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) GetMonitor(ctx context.Context, hostID int64) (usecase.AutomationMonitor, error) {
	pb := &pluginv1.GetMonitorRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetMonitor", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetMonitor(cctx, pb)
	})
	if err != nil {
		return usecase.AutomationMonitor{}, err
	}
	resp := respAny.(*pluginv1.GetMonitorResponse)
	if strings.TrimSpace(resp.GetRawJson()) == "" {
		return usecase.AutomationMonitor{}, nil
	}
	var raw struct {
		StorageStats float64         `json:"StorageStats"`
		NetworkStats json.RawMessage `json:"NetworkStats"`
		CpuStats     float64         `json:"CpuStats"`
		MemoryStats  float64         `json:"MemoryStats"`
	}
	if err := json.Unmarshal([]byte(resp.GetRawJson()), &raw); err != nil {
		return usecase.AutomationMonitor{}, err
	}
	bytesIn, bytesOut := parseNetworkStats(raw.NetworkStats)
	return usecase.AutomationMonitor{
		CPUPercent:     int(math.Round(raw.CpuStats)),
		MemoryPercent:  int(math.Round(raw.MemoryStats)),
		StoragePercent: int(math.Round(raw.StorageStats)),
		BytesIn:        bytesIn,
		BytesOut:       bytesOut,
	}, nil
}

func (c *PluginInstanceClient) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	pb := &pluginv1.GetVNCURLRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetVNCURL", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetVNCURL(cctx, pb)
	})
	if err != nil {
		return "", err
	}
	resp := respAny.(*pluginv1.GetVNCURLResponse)
	return resp.GetUrl(), nil
}
