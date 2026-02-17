package realname

import (
	"context"
	"strings"
	"time"

	plugins "xiaoheiplay/internal/adapter/plugins"
	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type kycPluginProvider struct {
	mgr        *plugins.Manager
	pluginID   string
	instanceID string
	name       string
	canQuery   bool
}

func (p *kycPluginProvider) Key() string {
	return pluginProviderKey(p.pluginID, p.instanceID)
}

func (p *kycPluginProvider) Name() string {
	if strings.TrimSpace(p.name) == "" {
		return p.Key()
	}
	return p.name
}

func (p *kycPluginProvider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	return p.VerifyWithInput(ctx, appshared.RealNameVerifyInput{
		RealName: realName,
		IDNumber: idNumber,
	})
}

func (p *kycPluginProvider) VerifyWithInput(ctx context.Context, in appshared.RealNameVerifyInput) (bool, string, error) {
	if p.mgr == nil {
		return false, "plugin manager missing", nil
	}
	client, ok := p.mgr.GetKYCClient("kyc", p.pluginID, p.instanceID)
	if !ok || client == nil {
		return false, "plugin not loaded", nil
	}
	params := map[string]string{
		"real_name": strings.TrimSpace(in.RealName),
		"id_number": strings.TrimSpace(in.IDNumber),
		"phone":     strings.TrimSpace(in.Phone),
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	start, err := client.Start(cctx, &pluginv1.KycStartRequest{
		UserId: "",
		Params: params,
	})
	if err != nil {
		return false, "", err
	}
	if start != nil && !start.Ok {
		return false, strings.TrimSpace(start.Error), nil
	}
	token := ""
	if start != nil {
		token = strings.TrimSpace(start.Token)
	}
	if token == "" {
		return true, "", nil
	}
	if !p.canQuery {
		return false, "pending:" + token, nil
	}
	query, qerr := client.QueryResult(cctx, &pluginv1.KycQueryRequest{Token: token})
	if qerr != nil {
		return false, "", qerr
	}
	if query != nil && !query.Ok {
		reason := strings.TrimSpace(query.Error)
		if reason == "" {
			reason = strings.TrimSpace(query.Reason)
		}
		return false, reason, nil
	}
	status := strings.ToLower(strings.TrimSpace(query.GetStatus()))
	switch status {
	case "verified", "approved", "success", "passed", "pass":
		return true, "", nil
	case "pending", "processing", "reviewing":
		return false, "pending:" + token, nil
	default:
		reason := strings.TrimSpace(query.GetReason())
		if reason == "" {
			reason = status
		}
		return false, reason, nil
	}
}

func (p *kycPluginProvider) QueryPending(ctx context.Context, token string, provider string) (string, string, error) {
	if p.mgr == nil {
		return "", "plugin manager missing", nil
	}
	client, ok := p.mgr.GetKYCClient("kyc", p.pluginID, p.instanceID)
	if !ok || client == nil {
		return "", "plugin not loaded", nil
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	query, err := client.QueryResult(cctx, &pluginv1.KycQueryRequest{Token: strings.TrimSpace(token)})
	if err != nil {
		return "", "", err
	}
	if query != nil && !query.Ok {
		reason := strings.TrimSpace(query.Error)
		if reason == "" {
			reason = strings.TrimSpace(query.Reason)
		}
		return "failed", reason, nil
	}
	status := strings.ToLower(strings.TrimSpace(query.GetStatus()))
	switch status {
	case "verified", "approved", "success", "passed", "pass":
		return "verified", "", nil
	case "failed", "reject", "rejected", "deny", "denied":
		return "failed", strings.TrimSpace(query.GetReason()), nil
	default:
		return "pending", strings.TrimSpace(query.GetReason()), nil
	}
}

func pluginProviderKey(pluginID, instanceID string) string {
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = plugins.DefaultInstanceID
	}
	return "plugin/" + pluginID + "/" + instanceID
}

func parsePluginProviderKey(key string) (pluginID, instanceID string, ok bool) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "plugin/") {
		return "", "", false
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", "", false
	}
	pluginID = strings.TrimSpace(parts[1])
	instanceID = strings.TrimSpace(parts[2])
	if pluginID == "" || instanceID == "" {
		return "", "", false
	}
	return pluginID, instanceID, true
}

var _ appshared.RealNameProvider = (*kycPluginProvider)(nil)
var _ appshared.RealNameProviderWithInput = (*kycPluginProvider)(nil)
var _ appshared.RealNameProviderPendingPoller = (*kycPluginProvider)(nil)
