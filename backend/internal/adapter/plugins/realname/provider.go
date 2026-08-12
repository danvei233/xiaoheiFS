package realnameplugin

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	plugins "xiaoheiplay/internal/adapter/plugins/core"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// defaultKYCRequestTimeout bounds each KYC RPC (Start, QueryResult) individually.
const defaultKYCRequestTimeout = 10 * time.Second

type Provider struct {
	mgr        *plugins.Manager
	pluginID   string
	instanceID string
	name       string
	canQuery   bool
	timeout    time.Duration
}

func NewProvider(mgr *plugins.Manager, pluginID, instanceID, name string, canQuery bool) *Provider {
	return &Provider{
		mgr:        mgr,
		pluginID:   strings.TrimSpace(pluginID),
		instanceID: strings.TrimSpace(instanceID),
		name:       strings.TrimSpace(name),
		canQuery:   canQuery,
		timeout:    defaultKYCRequestTimeout,
	}
}

func (p *Provider) requestTimeout() time.Duration {
	if p.timeout > 0 {
		return p.timeout
	}
	return defaultKYCRequestTimeout
}

func (p *Provider) Key() string {
	return ProviderKey(p.pluginID, p.instanceID)
}

func (p *Provider) Name() string {
	if strings.TrimSpace(p.name) == "" {
		return p.Key()
	}
	return p.name
}

func (p *Provider) Verify(ctx context.Context, realName string, idNumber string) (bool, string, error) {
	return p.VerifyWithInput(ctx, appshared.RealNameVerifyInput{RealName: realName, IDNumber: idNumber})
}

func (p *Provider) VerifyWithInput(ctx context.Context, in appshared.RealNameVerifyInput) (bool, string, error) {
	if p.mgr == nil {
		// Return an error rather than a (false, reason, nil) so the service does
		// not persist a permanent failed record for a transient condition.
		return false, "", fmt.Errorf("kyc plugin error: plugin manager missing")
	}
	client, ok := p.mgr.GetKYCClient("kyc", p.pluginID, p.instanceID)
	if !ok || client == nil {
		return false, "", fmt.Errorf("kyc plugin error: plugin not loaded")
	}
	return p.verifyWithClient(ctx, client, in)
}

func (p *Provider) verifyWithClient(ctx context.Context, client pluginv1.KycServiceClient, in appshared.RealNameVerifyInput) (bool, string, error) {
	params := map[string]string{
		"name":      strings.TrimSpace(in.RealName),
		"real_name": strings.TrimSpace(in.RealName),
		"id_number": strings.TrimSpace(in.IDNumber),
		"phone":     strings.TrimSpace(in.Phone),
		"mobile":    strings.TrimSpace(in.Phone),
	}
	if callbackURL := strings.TrimSpace(in.CallbackURL); callbackURL != "" {
		params["callback_url"] = callbackURL
	}
	cctx, cancel := context.WithTimeout(ctx, p.requestTimeout())
	defer cancel()
	start, err := client.Start(cctx, &pluginv1.KycStartRequest{UserId: "", Params: params})
	if err != nil {
		return false, "", mapKycRPCError(err)
	}
	if start == nil {
		return false, "", fmt.Errorf("kyc plugin error: empty start response")
	}
	if !start.Ok {
		reason := strings.TrimSpace(start.Error)
		code := strings.TrimSpace(start.ErrorCode)
		if code != "" {
			if reason == "" {
				reason = "kyc start failed"
			}
			reason = reason + " (" + code + ")"
		}
		return false, reason, nil
	}
	token := strings.TrimSpace(start.Token)
	nextStep := strings.ToLower(strings.TrimSpace(start.GetNextStep()))
	redirectURL := strings.TrimSpace(start.GetUrl())
	// A redirect/face-flow answer is only resolvable by polling when it carries a
	// token; without one it must not be treated as verified nor as a pending we
	// can later poll, so it falls through to the empty-token error below.
	if (nextStep == "redirect" || redirectURL != "") && token != "" {
		reason := "pending_face:baidu:" + token
		if redirectURL != "" {
			encoded := base64.RawURLEncoding.EncodeToString([]byte(redirectURL))
			reason += ":" + encoded
		}
		return false, reason, nil
	}
	if token == "" {
		return false, "", fmt.Errorf("kyc plugin error: empty token")
	}
	if !p.canQuery {
		return false, "pending:" + token, nil
	}
	// QueryResult gets its own budget: a Start that consumed the whole Start
	// timeout must not starve the follow-up query of an already expired context.
	qctx, qcancel := context.WithTimeout(ctx, p.requestTimeout())
	defer qcancel()
	query, qerr := client.QueryResult(qctx, &pluginv1.KycQueryRequest{Token: token})
	if qerr != nil {
		return false, "", mapKycRPCError(qerr)
	}
	if query == nil {
		return false, "", fmt.Errorf("kyc plugin error: empty query response")
	}
	if !query.Ok {
		reason := strings.TrimSpace(query.Error)
		code := strings.TrimSpace(query.ErrorCode)
		if reason == "" {
			reason = strings.TrimSpace(query.Reason)
		}
		if code != "" {
			if reason == "" {
				reason = "kyc query failed"
			}
			reason = reason + " (" + code + ")"
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

func (p *Provider) QueryPending(ctx context.Context, token string, provider string) (string, string, error) {
	// A provider that cannot QueryResult can never resolve its own pending
	// records; resolve them to failed so polling does not leave them pending forever.
	if !p.canQuery {
		return "failed", "provider does not support result querying", nil
	}
	if p.mgr == nil {
		return "", "plugin manager missing", nil
	}
	client, ok := p.mgr.GetKYCClient("kyc", p.pluginID, p.instanceID)
	if !ok || client == nil {
		return "", "plugin not loaded", nil
	}
	return p.queryPendingWithClient(ctx, client, token)
}

func (p *Provider) queryPendingWithClient(ctx context.Context, client pluginv1.KycServiceClient, token string) (string, string, error) {
	cctx, cancel := context.WithTimeout(ctx, p.requestTimeout())
	defer cancel()
	query, err := client.QueryResult(cctx, &pluginv1.KycQueryRequest{Token: strings.TrimSpace(token)})
	if err != nil {
		return "", "", mapKycRPCError(err)
	}
	if query == nil {
		return "", "", fmt.Errorf("kyc plugin error: empty query response")
	}
	if !query.Ok {
		reason := strings.TrimSpace(query.Error)
		code := strings.TrimSpace(query.ErrorCode)
		if reason == "" {
			reason = strings.TrimSpace(query.Reason)
		}
		if code != "" {
			if reason == "" {
				reason = "kyc query failed"
			}
			reason = reason + " (" + code + ")"
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

func ProviderKey(pluginID, instanceID string) string {
	pluginID = strings.TrimSpace(pluginID)
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = plugins.DefaultInstanceID
	}
	return "plugin/" + pluginID + "/" + instanceID
}

func ParseProviderKey(key string) (pluginID, instanceID string, ok bool) {
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

func mapKycRPCError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	msg := strings.TrimSpace(st.Message())
	lower := strings.ToLower(msg)
	if st.Code() == codes.InvalidArgument {
		if strings.Contains(lower, "params.mobile required") ||
			strings.Contains(lower, "mobile required") ||
			strings.Contains(lower, "phone required") {
			return domain.ErrPhoneRequired
		}
		if msg == "" {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, msg)
	}
	if msg == "" {
		msg = st.Code().String()
	}
	return fmt.Errorf("kyc plugin error: %s", msg)
}

var _ appshared.RealNameProvider = (*Provider)(nil)
var _ appshared.RealNameProviderWithInput = (*Provider)(nil)
var _ appshared.RealNameProviderPendingPoller = (*Provider)(nil)
