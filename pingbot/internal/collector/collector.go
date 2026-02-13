package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
)

func Snapshot(ctx context.Context, hostnameAlias string) (map[string]any, []string) {
	warnings := make([]string, 0)

	info, err := host.InfoWithContext(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("host info failed: %v", err))
	}
	cpuInfo, err := cpu.InfoWithContext(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cpu info failed: %v", err))
	}
	cpuCores, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cpu core count failed: %v", err))
	}
	cpuUsage, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("cpu usage failed: %v", err))
	}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("memory usage failed: %v", err))
	}
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("disk partitions failed: %v", err))
	}
	conns, err := gnet.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("network connections failed: %v", err))
	}

	hostName := ""
	if info != nil {
		hostName = info.Hostname
	}
	if hostnameAlias != "" {
		hostName = hostnameAlias
	}
	systemOS := ""
	systemPlatform := ""
	systemKernel := ""
	systemUptime := uint64(0)
	if info != nil {
		systemOS = info.OS
		systemPlatform = info.Platform
		systemKernel = info.KernelVersion
		systemUptime = info.Uptime
	}
	system := map[string]any{
		"hostname": hostName,
		"os":       systemOS,
		"platform": systemPlatform,
		"kernel":   systemKernel,
		"uptime":   systemUptime,
	}
	cpuMap := map[string]any{
		"model":         "",
		"cores":         cpuCores,
		"usage_percent": 0.0,
	}
	if len(cpuInfo) > 0 {
		cpuMap["model"] = cpuInfo[0].ModelName
	}
	if len(cpuUsage) > 0 {
		cpuMap["usage_percent"] = cpuUsage[0]
	}
	memTotal := uint64(0)
	memUsed := uint64(0)
	memUsagePercent := 0.0
	if vm != nil {
		memTotal = vm.Total
		memUsed = vm.Used
		memUsagePercent = vm.UsedPercent
	}
	memMap := map[string]any{
		"total":         memTotal,
		"used":          memUsed,
		"usage_percent": memUsagePercent,
	}
	disks := make([]map[string]any, 0, len(partitions))
	for _, p := range partitions {
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("disk usage failed mount=%s err=%v", p.Mountpoint, err))
			continue
		}
		disks = append(disks, map[string]any{
			"mount":         p.Mountpoint,
			"fs":            p.Fstype,
			"total":         u.Total,
			"used":          u.Used,
			"usage_percent": u.UsedPercent,
		})
	}
	ports := make([]map[string]any, 0, len(conns))
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		ports = append(ports, map[string]any{
			"port":         c.Laddr.Port,
			"proto":        "tcp",
			"listen":       true,
			"process_name": "",
		})
	}
	sort.Slice(ports, func(i, j int) bool {
		pi, _ := ports[i]["port"].(uint32)
		pj, _ := ports[j]["port"].(uint32)
		return pi < pj
	})
	return map[string]any{
		"system": system,
		"cpu":    cpuMap,
		"memory": memMap,
		"disks":  disks,
		"ports":  ports,
	}, warnings
}

func DefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\pingbot\config.yaml`
	}
	return "/etc/pingbot/config.yaml"
}

func DefaultLogsPath() string {
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "logs")
	}
	return "logs"
}
