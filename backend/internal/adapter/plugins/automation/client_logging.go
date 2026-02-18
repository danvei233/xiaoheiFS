package automation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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
	trace, _ := appshared.GetAutomationLogContext(ctx)
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
				err = fmt.Errorf("%s", traceMsg)
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
