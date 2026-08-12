package plugins

import (
	"context"
	"strings"
	"time"

	"fmt"
	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// defaultSMSOperationTimeout bounds a single plugin Send RPC so a hung plugin
// cannot block a caller (e.g. the admin SMS test endpoint) indefinitely.
const defaultSMSOperationTimeout = 10 * time.Second

type SMSSender struct {
	manager *Manager
	timeout time.Duration
}

func NewSMSSender(manager *Manager) *SMSSender {
	return &SMSSender{manager: manager, timeout: defaultSMSOperationTimeout}
}

func (s *SMSSender) Send(ctx context.Context, pluginID, instanceID string, msg appshared.SMSMessage) (appshared.SMSDelivery, error) {
	if s == nil || s.manager == nil {
		return appshared.SMSDelivery{}, fmt.Errorf("plugin manager unavailable")
	}
	// Structural guardrail: never hand an unvalidated message to the plugin.
	if err := msg.Validate(); err != nil {
		return appshared.SMSDelivery{}, err
	}
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = DefaultInstanceID
	}
	if pluginID == "" {
		return appshared.SMSDelivery{}, fmt.Errorf("sms plugin not configured")
	}
	if _, err := s.manager.EnsureRunning(ctx, "sms", pluginID, instanceID); err != nil {
		return appshared.SMSDelivery{}, err
	}
	client, ok := s.manager.GetSMSClient("sms", pluginID, instanceID)
	if !ok || client == nil {
		return appshared.SMSDelivery{}, fmt.Errorf("sms plugin not running")
	}
	req := &pluginv1.SendSmsRequest{
		TemplateId: strings.TrimSpace(msg.TemplateID),
		Content:    strings.TrimSpace(msg.Content),
		Vars:       msg.Vars,
		Phones:     msg.Phones,
	}
	timeout := s.timeout
	if timeout <= 0 {
		timeout = defaultSMSOperationTimeout
	}
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := client.Send(sctx, req)
	if err != nil {
		return appshared.SMSDelivery{}, MapRPCError(err, "sms plugin")
	}
	if resp == nil || !resp.Ok {
		errMsg := "sms send failed"
		errCode := ""
		if resp != nil && strings.TrimSpace(resp.Error) != "" {
			errMsg = SanitizePluginMessage(strings.TrimSpace(resp.Error))
		}
		if resp != nil {
			errCode = strings.TrimSpace(resp.ErrorCode)
		}
		if errCode != "" {
			return appshared.SMSDelivery{}, fmt.Errorf("%s (%s)", errMsg, errCode)
		}
		return appshared.SMSDelivery{}, fmt.Errorf("%s", errMsg)
	}
	return appshared.SMSDelivery{MessageID: strings.TrimSpace(resp.MessageId)}, nil
}
