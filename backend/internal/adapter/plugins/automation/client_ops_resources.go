package automation

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

func (c *PluginInstanceClient) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	pb := &pluginv1.ListSnapshotsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListSnapshots", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListSnapshots(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListSnapshotsResponse)
	out := make([]appshared.AutomationSnapshot, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationSnapshot{
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

func (c *PluginInstanceClient) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	pb := &pluginv1.ListBackupsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListBackups", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListBackups(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListBackupsResponse)
	out := make([]appshared.AutomationBackup, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationBackup{
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

func (c *PluginInstanceClient) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	pb := &pluginv1.ListFirewallRulesRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListFirewallRules", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListFirewallRules(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListFirewallRulesResponse)
	out := make([]appshared.AutomationFirewallRule, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationFirewallRule{
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

func (c *PluginInstanceClient) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
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

func (c *PluginInstanceClient) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	pb := &pluginv1.ListPortMappingsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListPortMappings", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListPortMappings(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListPortMappingsResponse)
	out := make([]appshared.AutomationPortMapping, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationPortMapping{
			"id":    it.GetId(),
			"name":  it.GetName(),
			"sport": it.GetSport(),
			"dport": it.GetDport(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
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

func (c *PluginInstanceClient) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	pb := &pluginv1.Empty{}
	respAny, err := c.call(ctx, "automation.ListAreas", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListAreas(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListAreasResponse)
	out := make([]appshared.AutomationArea, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationArea{ID: it.GetId(), Name: it.GetName(), State: int(it.GetState())})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	pb := &pluginv1.ListImagesRequest{LineId: lineID}
	respAny, err := c.call(ctx, "automation.ListImages", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListImages(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListImagesResponse)
	out := make([]appshared.AutomationImage, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationImage{ImageID: it.GetId(), Name: it.GetName(), Type: it.GetType()})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	pb := &pluginv1.Empty{}
	respAny, err := c.call(ctx, "automation.ListLines", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListLines(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListLinesResponse)
	out := make([]appshared.AutomationLine, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationLine{ID: it.GetId(), Name: it.GetName(), AreaID: it.GetAreaId(), State: int(it.GetState())})
	}
	return out, nil
}

func (c *PluginInstanceClient) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	pb := &pluginv1.ListPackagesRequest{LineId: lineID}
	respAny, err := c.call(ctx, "automation.ListPackages", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListPackages(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListPackagesResponse)
	out := make([]appshared.AutomationProduct, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationProduct{
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

func (c *PluginInstanceClient) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	pb := &pluginv1.GetMonitorRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetMonitor", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetMonitor(cctx, pb)
	})
	if err != nil {
		return appshared.AutomationMonitor{}, err
	}
	resp := respAny.(*pluginv1.GetMonitorResponse)
	if strings.TrimSpace(resp.GetRawJson()) == "" {
		return appshared.AutomationMonitor{}, nil
	}
	var raw struct {
		StorageStats float64         `json:"StorageStats"`
		NetworkStats json.RawMessage `json:"NetworkStats"`
		CpuStats     float64         `json:"CpuStats"`
		MemoryStats  float64         `json:"MemoryStats"`
	}
	if err := json.Unmarshal([]byte(resp.GetRawJson()), &raw); err != nil {
		return appshared.AutomationMonitor{}, err
	}
	bytesIn, bytesOut := parseNetworkStats(raw.NetworkStats)
	return appshared.AutomationMonitor{
		CPUPercent:     int(math.Round(raw.CpuStats)),
		MemoryPercent:  int(math.Round(raw.MemoryStats)),
		StoragePercent: int(math.Round(raw.StorageStats)),
		BytesIn:        bytesIn,
		BytesOut:       bytesOut,
	}, nil
}
