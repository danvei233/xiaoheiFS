package system

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	appshared "xiaoheiplay/internal/app/shared"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Status(ctx context.Context) (appshared.ServerStatus, error) {
	info, _ := host.InfoWithContext(ctx)
	cpuInfo, _ := cpu.InfoWithContext(ctx)
	cpuCores, _ := cpu.CountsWithContext(ctx, true)
	// interval=0 is often 0/unstable on first call; sample briefly for a meaningful value.
	usage, _ := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	vm, _ := mem.VirtualMemoryWithContext(ctx)

	diskPath := defaultDiskPath()
	du, _ := disk.UsageWithContext(ctx, diskPath)

	status := appshared.ServerStatus{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		KernelVersion:   info.KernelVersion,
		UptimeSeconds:   info.Uptime,
		MemTotal:        vm.Total,
		MemUsed:         vm.Used,
		MemUsedPercent:  vm.UsedPercent,
		DiskTotal:       du.Total,
		DiskUsed:        du.Used,
		DiskUsedPercent: du.UsedPercent,
	}
	if len(cpuInfo) > 0 {
		status.CPUModel = cpuInfo[0].ModelName
		if cpuCores > 0 {
			status.CPUCores = cpuCores
		} else {
			status.CPUCores = int(cpuInfo[0].Cores)
		}
	} else if cpuCores > 0 {
		status.CPUCores = cpuCores
	}
	if len(usage) > 0 {
		status.CPUUsagePercent = usage[0]
	}
	return status, nil
}

func defaultDiskPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return "/"
	}
	vol := filepath.VolumeName(wd)
	if vol != "" && strings.HasSuffix(vol, "\\") {
		return vol
	}
	if vol != "" {
		return vol + "\\"
	}
	return "/"
}
