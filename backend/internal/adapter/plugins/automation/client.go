package automation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"xiaoheiplay/internal/adapter/plugins/core"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

type PluginInstanceClient struct {
	mgr        *plugins.Manager
	pluginID   string
	instanceID string
	timeout    time.Duration
	settings   appports.SettingsRepository
	autoLogs   appports.AutomationLogRepository
}

func NewPluginInstanceClient(mgr *plugins.Manager, pluginID, instanceID string, settings appports.SettingsRepository, autoLogs appports.AutomationLogRepository) *PluginInstanceClient {
	return &PluginInstanceClient{
		mgr:        mgr,
		pluginID:   strings.TrimSpace(pluginID),
		instanceID: strings.TrimSpace(instanceID),
		timeout:    30 * time.Second,
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
		return nil, fmt.Errorf("plugin manager missing")
	}
	cli, _, err := c.mgr.GetAutomationClient(ctx, c.pluginID, c.instanceID)
	return cli, err
}

func (c *PluginInstanceClient) call(ctx context.Context, action string, req proto.Message, fn func(context.Context, pluginv1.AutomationServiceClient) (proto.Message, error)) (proto.Message, error) {
	cli, err := c.client(ctx)
	if err != nil {
		c.logRPC(ctx, action, req, nil, 0, err)
		return nil, err
	}
	start := time.Now()
	resp, rpcErr := c.invoke(ctx, cli, fn)

	// 自愈：插件进程退出/被外部结束后，go-plugin 仍缓存旧地址，
	// 表现为 codes.Unavailable（connection refused）。重启实例后重试一次。
	if isPluginUnavailable(rpcErr) && c.mgr != nil {
		restartCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		restartErr := c.mgr.RestartInstance(restartCtx, "automation", c.pluginID, c.instanceID)
		cancel()
		if restartErr == nil {
			if fresh, err := c.client(ctx); err == nil {
				resp, rpcErr = c.invoke(ctx, fresh, fn)
			}
		}
	}

	err = mapUnimplemented(rpcErr)
	err = mapRPCBusinessError(err)
	c.logRPC(ctx, action, req, resp, time.Since(start), err)
	return resp, err
}

// invoke 执行一次 RPC，仅做单次超时控制，保留原始错误供自愈判定。
func (c *PluginInstanceClient) invoke(ctx context.Context, cli pluginv1.AutomationServiceClient, fn func(context.Context, pluginv1.AutomationServiceClient) (proto.Message, error)) (proto.Message, error) {
	cctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return fn(cctx, cli)
}

// isPluginUnavailable 判断是否为传输层不可用（插件进程退出、连接被拒等）。
func isPluginUnavailable(err error) bool {
	if err == nil {
		return false
	}
	return status.Code(err) == codes.Unavailable
}

func ensureOpOK(resp proto.Message) error {
	if op, ok := resp.(*pluginv1.OperationResult); ok && op != nil {
		// Backward compatibility: old plugins may return legacy Empty payload and
		// decode into zero-value OperationResult on the host side.
		if !op.GetOk() && strings.TrimSpace(op.GetErrorCode()) == "" && strings.TrimSpace(op.GetErrorMessage()) == "" {
			return nil
		}
		if op.GetOk() {
			return nil
		}
		msg := strings.TrimSpace(op.GetErrorMessage())
		if msg == "" {
			msg = "operation failed"
		}
		code := strings.TrimSpace(op.GetErrorCode())
		if code != "" {
			return fmt.Errorf("%s (%s)", msg, code)
		}
		return fmt.Errorf("%s", msg)
	}
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
	return fmt.Errorf("%s", msg)
}

func (c *PluginInstanceClient) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	pb := &pluginv1.CreateInstanceRequest{
		LineId:        req.LineID,
		ImageId:       req.ImageID,
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
		return appshared.AutomationCreateHostResult{}, err
	}
	resp := respAny.(*pluginv1.CreateInstanceResponse)
	return appshared.AutomationCreateHostResult{HostID: resp.GetInstanceId(), Raw: map[string]any{"instance_id": resp.GetInstanceId()}}, nil
}

func (c *PluginInstanceClient) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	pb := &pluginv1.GetInstanceRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetInstance", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetInstance(cctx, pb)
	})
	if err != nil {
		return appshared.AutomationHostInfo{}, err
	}
	resp := respAny.(*pluginv1.GetInstanceResponse)
	inst := resp.GetInstance()
	var expire *time.Time
	if inst.GetExpireAtUnix() > 0 {
		t := time.Unix(inst.GetExpireAtUnix(), 0)
		expire = &t
	}
	return appshared.AutomationHostInfo{
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

func (c *PluginInstanceClient) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	pb := &pluginv1.ListInstancesSimpleRequest{SearchTag: strings.TrimSpace(searchTag)}
	respAny, err := c.call(ctx, "automation.ListInstancesSimple", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListInstancesSimple(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListInstancesSimpleResponse)
	out := make([]appshared.AutomationHostSimple, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationHostSimple{ID: it.GetId(), HostName: it.GetName(), IP: it.GetIp()})
	}
	return out, nil
}

func (c *PluginInstanceClient) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
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
