package automation

import (
	"context"

	"google.golang.org/protobuf/proto"

	appshared "xiaoheiplay/internal/app/shared"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// 本文件承载「基础设施资源」类能力的 gRPC 客户端实现：
// 网卡 / 数据盘 / 光盘 / 反向代理 / 启动顺序 / PCI / USB 直通。
// 这些能力都是可选的，插件未实现时返回 codes.Unimplemented，
// 由 call() 统一转换为 appshared.ErrNotSupported。

// ---- 网卡 ----

func (c *PluginInstanceClient) ListNetworkCards(ctx context.Context, hostID int64) ([]appshared.AutomationNetworkCard, error) {
	pb := &pluginv1.ListNetworkCardsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListNetworkCards", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListNetworkCards(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListNetworkCardsResponse)
	out := make([]appshared.AutomationNetworkCard, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationNetworkCard{
			"nic_name":   it.GetNicName(),
			"nic_type":   it.GetNicType(),
			"mac_addr":   it.GetMacAddr(),
			"ip4_addr":   it.GetIp4Addr(),
			"ip6_addr":   it.GetIp6Addr(),
			"nic_gate":   it.GetNicGate(),
			"nic_mask":   it.GetNicMask(),
			"dns_addr":   it.GetDnsAddr(),
			"ip6_prefix": int(it.GetIp6Prefix()),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) CreateNetworkCard(ctx context.Context, hostID int64, card appshared.AutomationNetworkCardInput) error {
	pb := &pluginv1.CreateNetworkCardRequest{InstanceId: hostID, Card: networkCardToProto(card)}
	respAny, err := c.call(ctx, "automation.CreateNetworkCard", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.CreateNetworkCard(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) UpdateNetworkCard(ctx context.Context, hostID int64, nicName string, card appshared.AutomationNetworkCardInput) error {
	pb := &pluginv1.UpdateNetworkCardRequest{InstanceId: hostID, NicName: nicName, Card: networkCardToProto(card)}
	respAny, err := c.call(ctx, "automation.UpdateNetworkCard", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.UpdateNetworkCard(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteNetworkCard(ctx context.Context, hostID int64, nicName string) error {
	pb := &pluginv1.DeleteNetworkCardRequest{InstanceId: hostID, NicName: nicName}
	respAny, err := c.call(ctx, "automation.DeleteNetworkCard", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteNetworkCard(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func networkCardToProto(card appshared.AutomationNetworkCardInput) *pluginv1.AutomationNetworkCard {
	return &pluginv1.AutomationNetworkCard{
		NicName:   card.NicName,
		NicType:   card.NicType,
		MacAddr:   card.MacAddr,
		Ip4Addr:   card.IP4Addr,
		Ip6Addr:   card.IP6Addr,
		NicGate:   card.NicGate,
		NicMask:   card.NicMask,
		DnsAddr:   card.DNSAddr,
		Ip6Prefix: int32(card.IP6Prefix),
	}
}

// ---- 数据盘 ----

func (c *PluginInstanceClient) ListDataDisks(ctx context.Context, hostID int64) ([]appshared.AutomationDataDisk, error) {
	pb := &pluginv1.ListDataDisksRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListDataDisks", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListDataDisks(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListDataDisksResponse)
	out := make([]appshared.AutomationDataDisk, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationDataDisk{
			"hdd_name": it.GetHddName(),
			"hdd_size": int(it.GetHddSize()),
			"hdd_flag": int(it.GetHddFlag()),
			"hdd_type": int(it.GetHddType()),
			"hdd_path": it.GetHddPath(),
			"hdd_scsi": it.GetHddScsi(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) AttachDataDisk(ctx context.Context, hostID int64, sizeMB int, hddType int) error {
	pb := &pluginv1.AttachDataDiskRequest{InstanceId: hostID, HddSize: int32(sizeMB), HddType: int32(hddType)}
	respAny, err := c.call(ctx, "automation.AttachDataDisk", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.AttachDataDisk(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DetachDataDisk(ctx context.Context, hostID int64, hddName string) error {
	pb := &pluginv1.DetachDataDiskRequest{InstanceId: hostID, HddName: hddName}
	respAny, err := c.call(ctx, "automation.DetachDataDisk", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DetachDataDisk(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteDataDisk(ctx context.Context, hostID int64, hddName string) error {
	pb := &pluginv1.DeleteDataDiskRequest{InstanceId: hostID, HddName: hddName}
	respAny, err := c.call(ctx, "automation.DeleteDataDisk", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteDataDisk(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

// ---- 光盘 ----

func (c *PluginInstanceClient) ListISOs(ctx context.Context, hostID int64) ([]appshared.AutomationISO, []string, error) {
	pb := &pluginv1.ListISOsRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListISOs", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListISOs(cctx, pb)
	})
	if err != nil {
		return nil, nil, err
	}
	resp := respAny.(*pluginv1.ListISOsResponse)
	out := make([]appshared.AutomationISO, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationISO{
			"iso_name": it.GetIsoName(),
			"iso_file": it.GetIsoFile(),
			"iso_path": it.GetIsoPath(),
			"iso_hint": it.GetIsoHint(),
		})
	}
	return out, resp.GetAvailableIsos(), nil
}

func (c *PluginInstanceClient) MountISO(ctx context.Context, hostID int64, in appshared.AutomationISOInput) error {
	pb := &pluginv1.MountISORequest{
		InstanceId: hostID,
		IsoFile:    in.IsoFile,
		IsoPath:    in.IsoPath,
		IsoHint:    in.IsoHint,
	}
	respAny, err := c.call(ctx, "automation.MountISO", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.MountISO(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) UnmountISO(ctx context.Context, hostID int64, isoName string) error {
	pb := &pluginv1.UnmountISORequest{InstanceId: hostID, IsoName: isoName}
	respAny, err := c.call(ctx, "automation.UnmountISO", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.UnmountISO(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

// ---- 反向代理 ----

func (c *PluginInstanceClient) ListReverseProxies(ctx context.Context, hostID int64) ([]appshared.AutomationReverseProxy, error) {
	pb := &pluginv1.ListReverseProxiesRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListReverseProxies", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListReverseProxies(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListReverseProxiesResponse)
	out := make([]appshared.AutomationReverseProxy, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationReverseProxy{
			"id":           it.GetId(),
			"domain":       it.GetDomain(),
			"backend_ip":   it.GetBackendIp(),
			"backend_port": int(it.GetBackendPort()),
			"ssl_enabled":  it.GetSslEnabled(),
			"description":  it.GetDescription(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) CreateReverseProxy(ctx context.Context, hostID int64, in appshared.AutomationReverseProxyInput) error {
	pb := &pluginv1.CreateReverseProxyRequest{
		InstanceId:  hostID,
		Domain:      in.Domain,
		BackendIp:   in.BackendIP,
		BackendPort: int32(in.BackendPort),
		SslEnabled:  in.SSLEnabled,
		Description: in.Description,
	}
	respAny, err := c.call(ctx, "automation.CreateReverseProxy", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.CreateReverseProxy(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) UpdateReverseProxy(ctx context.Context, hostID int64, proxyIndex int64, in appshared.AutomationReverseProxyInput) error {
	pb := &pluginv1.UpdateReverseProxyRequest{
		InstanceId:  hostID,
		ProxyIndex:  proxyIndex,
		Domain:      in.Domain,
		BackendIp:   in.BackendIP,
		BackendPort: int32(in.BackendPort),
		SslEnabled:  in.SSLEnabled,
		Description: in.Description,
	}
	respAny, err := c.call(ctx, "automation.UpdateReverseProxy", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.UpdateReverseProxy(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) DeleteReverseProxy(ctx context.Context, hostID int64, proxyIndex int64) error {
	pb := &pluginv1.DeleteReverseProxyRequest{InstanceId: hostID, ProxyIndex: proxyIndex}
	respAny, err := c.call(ctx, "automation.DeleteReverseProxy", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.DeleteReverseProxy(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

// ---- 启动顺序 ----

func (c *PluginInstanceClient) GetBootOrder(ctx context.Context, hostID int64) ([]appshared.AutomationBootOption, error) {
	pb := &pluginv1.GetBootOrderRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.GetBootOrder", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.GetBootOrder(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.GetBootOrderResponse)
	out := make([]appshared.AutomationBootOption, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationBootOption{EfiType: it.GetEfiType(), EfiName: it.GetEfiName()})
	}
	return out, nil
}

func (c *PluginInstanceClient) SetBootOrder(ctx context.Context, hostID int64, items []appshared.AutomationBootOption) error {
	pbItems := make([]*pluginv1.AutomationBootOption, 0, len(items))
	for _, it := range items {
		pbItems = append(pbItems, &pluginv1.AutomationBootOption{EfiType: it.EfiType, EfiName: it.EfiName})
	}
	pb := &pluginv1.SetBootOrderRequest{InstanceId: hostID, Items: pbItems}
	respAny, err := c.call(ctx, "automation.SetBootOrder", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.SetBootOrder(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

// ---- PCI / USB 直通 ----

func (c *PluginInstanceClient) ListPCIDevices(ctx context.Context, hostID int64) ([]appshared.AutomationPCIDevice, error) {
	pb := &pluginv1.ListPCIDevicesRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListPCIDevices", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListPCIDevices(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListPCIDevicesResponse)
	out := make([]appshared.AutomationPCIDevice, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationPCIDevice{
			"pci_key":  it.GetPciKey(),
			"gpu_uuid": it.GetGpuUuid(),
			"gpu_mdev": it.GetGpuMdev(),
			"gpu_hint": it.GetGpuHint(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) SetupPCIDevice(ctx context.Context, hostID int64, in appshared.AutomationPassthroughInput) error {
	pb := &pluginv1.SetupPCIDeviceRequest{
		InstanceId: hostID,
		Device: &pluginv1.AutomationPCIDevice{
			PciKey:  in.Key,
			GpuUuid: in.GpuUUID,
			GpuMdev: in.GpuMdev,
			GpuHint: in.Hint,
		},
		Action: in.Action,
	}
	respAny, err := c.call(ctx, "automation.SetupPCIDevice", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.SetupPCIDevice(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}

func (c *PluginInstanceClient) ListUSBDevices(ctx context.Context, hostID int64) ([]appshared.AutomationUSBDevice, error) {
	pb := &pluginv1.ListUSBDevicesRequest{InstanceId: hostID}
	respAny, err := c.call(ctx, "automation.ListUSBDevices", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.ListUSBDevices(cctx, pb)
	})
	if err != nil {
		return nil, err
	}
	resp := respAny.(*pluginv1.ListUSBDevicesResponse)
	out := make([]appshared.AutomationUSBDevice, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, appshared.AutomationUSBDevice{
			"usb_key":  it.GetUsbKey(),
			"vid_uuid": it.GetVidUuid(),
			"pid_uuid": it.GetPidUuid(),
			"usb_hint": it.GetUsbHint(),
		})
	}
	return out, nil
}

func (c *PluginInstanceClient) SetupUSBDevice(ctx context.Context, hostID int64, in appshared.AutomationPassthroughInput) error {
	pb := &pluginv1.SetupUSBDeviceRequest{
		InstanceId: hostID,
		Device: &pluginv1.AutomationUSBDevice{
			UsbKey:  in.Key,
			VidUuid: in.VidUUID,
			PidUuid: in.PidUUID,
			UsbHint: in.Hint,
		},
		Action: in.Action,
	}
	respAny, err := c.call(ctx, "automation.SetupUSBDevice", pb, func(cctx context.Context, cli pluginv1.AutomationServiceClient) (proto.Message, error) {
		return cli.SetupUSBDevice(cctx, pb)
	})
	if err != nil {
		return err
	}
	return ensureOpOK(respAny)
}
