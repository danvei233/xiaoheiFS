package plugins

import (
	"context"
	"strings"

	"fmt"
	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type SMSSender struct {
	manager *Manager
}

func NewSMSSender(manager *Manager) *SMSSender {
	return &SMSSender{manager: manager}
}

func (s *SMSSender) Send(ctx context.Context, pluginID, instanceID string, msg appshared.SMSMessage) (appshared.SMSDelivery, error) {
	if s == nil || s.manager == nil {
		return appshared.SMSDelivery{}, fmt.Errorf("plugin manager unavailable")
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
	resp, err := client.Send(ctx, req)
	if err != nil {
		return appshared.SMSDelivery{}, err
	}
	if resp == nil || !resp.Ok {
		errMsg := "sms send failed"
		if resp != nil && strings.TrimSpace(resp.Error) != "" {
			errMsg = strings.TrimSpace(resp.Error)
		}
		return appshared.SMSDelivery{}, fmt.Errorf("%s", errMsg)
	}
	return appshared.SMSDelivery{MessageID: strings.TrimSpace(resp.MessageId)}, nil
}
