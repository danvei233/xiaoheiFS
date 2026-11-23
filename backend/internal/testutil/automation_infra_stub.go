package testutil

import (
	"context"

	appshared "xiaoheiplay/internal/app/shared"
)

// AutomationInfraStub 提供 AutomationClient 中「基础设施资源」类可选能力的空实现：
// 网卡 / 数据盘 / 光盘 / 反向代理 / 启动顺序 / PCI / USB，以及防火墙的扩展方法。
//
// 这些能力都是后加的可选项，把默认实现集中在这里，测试用 fake 只要嵌入本类型
// 就能满足接口，无需为了新增一个可选能力在所有 fake 里补空方法。
type AutomationInfraStub struct{}

func (AutomationInfraStub) UpdateFirewallRule(ctx context.Context, hostID int64, ruleID int64, req appshared.AutomationFirewallRuleCreate) error {
	return nil
}

func (AutomationInfraStub) SetFirewallOptions(ctx context.Context, hostID int64, enable bool, policyIn string, policyOut string) error {
	return nil
}

func (AutomationInfraStub) ListNetworkCards(ctx context.Context, hostID int64) ([]appshared.AutomationNetworkCard, error) {
	return nil, nil
}

func (AutomationInfraStub) CreateNetworkCard(ctx context.Context, hostID int64, card appshared.AutomationNetworkCardInput) error {
	return nil
}

func (AutomationInfraStub) UpdateNetworkCard(ctx context.Context, hostID int64, nicName string, card appshared.AutomationNetworkCardInput) error {
	return nil
}

func (AutomationInfraStub) DeleteNetworkCard(ctx context.Context, hostID int64, nicName string) error {
	return nil
}

func (AutomationInfraStub) ListDataDisks(ctx context.Context, hostID int64) ([]appshared.AutomationDataDisk, error) {
	return nil, nil
}

func (AutomationInfraStub) AttachDataDisk(ctx context.Context, hostID int64, sizeMB int, hddType int) error {
	return nil
}

func (AutomationInfraStub) DetachDataDisk(ctx context.Context, hostID int64, hddName string) error {
	return nil
}

func (AutomationInfraStub) DeleteDataDisk(ctx context.Context, hostID int64, hddName string) error {
	return nil
}

func (AutomationInfraStub) ListISOs(ctx context.Context, hostID int64) ([]appshared.AutomationISO, []string, error) {
	return nil, nil, nil
}

func (AutomationInfraStub) MountISO(ctx context.Context, hostID int64, in appshared.AutomationISOInput) error {
	return nil
}

func (AutomationInfraStub) UnmountISO(ctx context.Context, hostID int64, isoName string) error {
	return nil
}

func (AutomationInfraStub) ListReverseProxies(ctx context.Context, hostID int64) ([]appshared.AutomationReverseProxy, error) {
	return nil, nil
}

func (AutomationInfraStub) CreateReverseProxy(ctx context.Context, hostID int64, in appshared.AutomationReverseProxyInput) error {
	return nil
}

func (AutomationInfraStub) UpdateReverseProxy(ctx context.Context, hostID int64, proxyIndex int64, in appshared.AutomationReverseProxyInput) error {
	return nil
}

func (AutomationInfraStub) DeleteReverseProxy(ctx context.Context, hostID int64, proxyIndex int64) error {
	return nil
}

func (AutomationInfraStub) GetBootOrder(ctx context.Context, hostID int64) ([]appshared.AutomationBootOption, error) {
	return nil, nil
}

func (AutomationInfraStub) SetBootOrder(ctx context.Context, hostID int64, items []appshared.AutomationBootOption) error {
	return nil
}

func (AutomationInfraStub) ListPCIDevices(ctx context.Context, hostID int64) ([]appshared.AutomationPCIDevice, error) {
	return nil, nil
}

func (AutomationInfraStub) SetupPCIDevice(ctx context.Context, hostID int64, in appshared.AutomationPassthroughInput) error {
	return nil
}

func (AutomationInfraStub) ListUSBDevices(ctx context.Context, hostID int64) ([]appshared.AutomationUSBDevice, error) {
	return nil, nil
}

func (AutomationInfraStub) SetupUSBDevice(ctx context.Context, hostID int64, in appshared.AutomationPassthroughInput) error {
	return nil
}
