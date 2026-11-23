package vps

import (
	"context"
	"strings"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// 本文件承载实例「基础设施资源」的编排：
// 网卡 / 数据盘 / 光盘 / 反向代理 / 启动顺序 / PCI / USB 直通。
// 全部走插件 gRPC 能力，插件未实现时返回 appshared.ErrNotSupported（HTTP 501）。

func (s *Service) infraClient(ctx context.Context, inst domain.VPSInstance) (AutomationClient, int64, error) {
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return nil, 0, appshared.ErrInvalidInput
	}
	cli, err := s.client(ctx, inst.GoodsTypeID)
	if err != nil {
		return nil, 0, err
	}
	return cli, hostID, nil
}

// ---- 网卡 ----

func (s *Service) ListNetworkCards(ctx context.Context, inst domain.VPSInstance) ([]AutomationNetworkCard, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.ListNetworkCards(ctx, hostID)
}

func (s *Service) CreateNetworkCard(ctx context.Context, inst domain.VPSInstance, card AutomationNetworkCardInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	card.NicName = strings.TrimSpace(card.NicName)
	card.NicType = strings.TrimSpace(card.NicType)
	return cli.CreateNetworkCard(ctx, hostID, card)
}

func (s *Service) UpdateNetworkCard(ctx context.Context, inst domain.VPSInstance, nicName string, card AutomationNetworkCardInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	nicName = strings.TrimSpace(nicName)
	if nicName == "" {
		return appshared.ErrInvalidInput
	}
	card.NicName = nicName
	card.NicType = strings.TrimSpace(card.NicType)
	return cli.UpdateNetworkCard(ctx, hostID, nicName, card)
}

func (s *Service) DeleteNetworkCard(ctx context.Context, inst domain.VPSInstance, nicName string) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	nicName = strings.TrimSpace(nicName)
	if nicName == "" {
		return appshared.ErrInvalidInput
	}
	return cli.DeleteNetworkCard(ctx, hostID, nicName)
}

// ---- 数据盘 ----

func (s *Service) ListDataDisks(ctx context.Context, inst domain.VPSInstance) ([]AutomationDataDisk, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.ListDataDisks(ctx, hostID)
}

func (s *Service) AttachDataDisk(ctx context.Context, inst domain.VPSInstance, sizeMB int, hddType int) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	if sizeMB < 1024 || sizeMB > 1024*1024 {
		return appshared.ErrInvalidInput
	}
	return cli.AttachDataDisk(ctx, hostID, sizeMB, hddType)
}

func (s *Service) DetachDataDisk(ctx context.Context, inst domain.VPSInstance, hddName string) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	hddName = strings.TrimSpace(hddName)
	if hddName == "" {
		return appshared.ErrInvalidInput
	}
	return cli.DetachDataDisk(ctx, hostID, hddName)
}

func (s *Service) DeleteDataDisk(ctx context.Context, inst domain.VPSInstance, hddName string) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	hddName = strings.TrimSpace(hddName)
	if hddName == "" {
		return appshared.ErrInvalidInput
	}
	return cli.DeleteDataDisk(ctx, hostID, hddName)
}

// ---- 光盘 ----

func (s *Service) ListISOs(ctx context.Context, inst domain.VPSInstance) ([]AutomationISO, []string, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, nil, err
	}
	return cli.ListISOs(ctx, hostID)
}

func (s *Service) MountISO(ctx context.Context, inst domain.VPSInstance, in AutomationISOInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	in.IsoFile = strings.TrimSpace(in.IsoFile)
	in.IsoPath = strings.TrimSpace(in.IsoPath)
	in.IsoHint = strings.TrimSpace(in.IsoHint)
	if in.IsoFile == "" && in.IsoPath == "" {
		return appshared.ErrInvalidInput
	}
	return cli.MountISO(ctx, hostID, in)
}

func (s *Service) UnmountISO(ctx context.Context, inst domain.VPSInstance, isoName string) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	isoName = strings.TrimSpace(isoName)
	if isoName == "" {
		return appshared.ErrInvalidInput
	}
	return cli.UnmountISO(ctx, hostID, isoName)
}

// ---- 反向代理 ----

func (s *Service) ListReverseProxies(ctx context.Context, inst domain.VPSInstance) ([]AutomationReverseProxy, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.ListReverseProxies(ctx, hostID)
}

func (s *Service) CreateReverseProxy(ctx context.Context, inst domain.VPSInstance, in AutomationReverseProxyInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	in.Domain = strings.TrimSpace(in.Domain)
	if in.Domain == "" {
		return appshared.ErrInvalidInput
	}
	if in.BackendPort <= 0 {
		in.BackendPort = 80
	}
	return cli.CreateReverseProxy(ctx, hostID, in)
}

func (s *Service) UpdateReverseProxy(ctx context.Context, inst domain.VPSInstance, proxyIndex int64, in AutomationReverseProxyInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	if proxyIndex < 0 {
		return appshared.ErrInvalidInput
	}
	in.Domain = strings.TrimSpace(in.Domain)
	if in.Domain == "" {
		return appshared.ErrInvalidInput
	}
	if in.BackendPort <= 0 {
		in.BackendPort = 80
	}
	return cli.UpdateReverseProxy(ctx, hostID, proxyIndex, in)
}

func (s *Service) DeleteReverseProxy(ctx context.Context, inst domain.VPSInstance, proxyIndex int64) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	if proxyIndex < 0 {
		return appshared.ErrInvalidInput
	}
	return cli.DeleteReverseProxy(ctx, hostID, proxyIndex)
}

// ---- 启动顺序 ----

func (s *Service) GetBootOrder(ctx context.Context, inst domain.VPSInstance) ([]AutomationBootOption, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.GetBootOrder(ctx, hostID)
}

func (s *Service) SetBootOrder(ctx context.Context, inst domain.VPSInstance, items []AutomationBootOption) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return appshared.ErrInvalidInput
	}
	cleaned := make([]AutomationBootOption, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.EfiName)
		if name == "" {
			continue
		}
		cleaned = append(cleaned, AutomationBootOption{EfiType: item.EfiType, EfiName: name})
	}
	if len(cleaned) == 0 {
		return appshared.ErrInvalidInput
	}
	return cli.SetBootOrder(ctx, hostID, cleaned)
}

// ---- PCI / USB 直通 ----

func (s *Service) ListPCIDevices(ctx context.Context, inst domain.VPSInstance) ([]AutomationPCIDevice, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.ListPCIDevices(ctx, hostID)
}

func (s *Service) SetupPCIDevice(ctx context.Context, inst domain.VPSInstance, in AutomationPassthroughInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	in.Key = strings.TrimSpace(in.Key)
	if in.Key == "" {
		return appshared.ErrInvalidInput
	}
	in.Action = normalizePassthroughAction(in.Action)
	return cli.SetupPCIDevice(ctx, hostID, in)
}

func (s *Service) ListUSBDevices(ctx context.Context, inst domain.VPSInstance) ([]AutomationUSBDevice, error) {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return nil, err
	}
	return cli.ListUSBDevices(ctx, hostID)
}

func (s *Service) SetupUSBDevice(ctx context.Context, inst domain.VPSInstance, in AutomationPassthroughInput) error {
	cli, hostID, err := s.infraClient(ctx, inst)
	if err != nil {
		return err
	}
	in.Key = strings.TrimSpace(in.Key)
	if in.Key == "" {
		return appshared.ErrInvalidInput
	}
	in.Action = normalizePassthroughAction(in.Action)
	return cli.SetupUSBDevice(ctx, hostID, in)
}

// normalizePassthroughAction 归一化直通动作：只接受 add / remove，缺省按 add。
func normalizePassthroughAction(action string) string {
	if strings.EqualFold(strings.TrimSpace(action), "remove") {
		return "remove"
	}
	return "add"
}
