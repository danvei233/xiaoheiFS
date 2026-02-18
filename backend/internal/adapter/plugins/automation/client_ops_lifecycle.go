package automation

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

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

var _ appshared.AutomationClient = (*PluginInstanceClient)(nil)
