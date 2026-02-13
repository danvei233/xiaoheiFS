package collector

import (
	"context"
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

func Snapshot(ctx context.Context, hostnameAlias string) map[string]any {
	info, _ := host.InfoWithContext(ctx)
	cpuInfo, _ := cpu.InfoWithContext(ctx)
	cpuCores, _ := cpu.CountsWithContext(ctx, true)
	cpuUsage, _ := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	vm, _ := mem.VirtualMemoryWithContext(ctx)
	partitions, _ := disk.PartitionsWithContext(ctx, false)
	conns, _ := gnet.ConnectionsWithContext(ctx, "inet")

	hostName := info.Hostname
	if hostnameAlias != "" {
		hostName = hostnameAlias
	}
	system := map[string]any{
		"hostname": hostName,
		"os":       info.OS,
		"platform": info.Platform,
		"kernel":   info.KernelVersion,
		"uptime":   info.Uptime,
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
	memMap := map[string]any{
		"total":         vm.Total,
		"used":          vm.Used,
		"usage_percent": vm.UsedPercent,
	}
	disks := make([]map[string]any, 0, len(partitions))
	for _, p := range partitions {
		u, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
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
	}
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
