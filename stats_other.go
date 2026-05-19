//go:build !linux

package main

import (
	"os"
	"runtime"
)

func (a *App) systemStats() SystemStats {
	h, _ := os.Hostname()
	if h == "" {
		h = "server"
	}
	return SystemStats{
		LoadAvg: "n/a", Uptime: "n/a",
		MemTotal: "n/a", MemUsed: "n/a", MemPercent: 0,
		DiskTotal: "n/a", DiskUsed: "n/a", DiskPercent: 0,
		Hostname: h, CPUCount: runtime.NumCPU(),
	}
}
