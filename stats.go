//go:build linux

package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (a *App) systemStats() SystemStats {
	stats := SystemStats{
		LoadAvg:  "n/a",
		Uptime:   "n/a",
		Hostname: hostname(),
		CPUCount: runtime.NumCPU(),
	}
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			stats.LoadAvg = strings.Join(fields[:3], " ")
		}
	}
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			if sec, err := strconv.ParseFloat(fields[0], 64); err == nil {
				stats.Uptime = humanDuration(time.Duration(sec) * time.Second)
			}
		}
	}
	if memTotal, memFree, ok := readMemKB(); ok {
		used := memTotal - memFree
		stats.MemTotal = humanSize(memTotal * 1024)
		stats.MemUsed = humanSize(used * 1024)
		if memTotal > 0 {
			stats.MemPercent = int(used * 100 / memTotal)
		}
	}
	if used, total, ok := diskUsage(a.cfg.WebRoot); ok {
		stats.DiskTotal = humanSize(int64(total))
		stats.DiskUsed = humanSize(int64(used))
		if total > 0 {
			stats.DiskPercent = int(used * 100 / total)
		}
	}
	return stats
}

func readMemKB() (int64, int64, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	var total, available int64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseInt(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			available = v
		}
	}
	return total, available, total > 0
}

func diskUsage(path string) (uint64, uint64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	if total == 0 {
		return 0, 0, false
	}
	return total - free, total, true
}

func humanDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "server"
	}
	return h
}
