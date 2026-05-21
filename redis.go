package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

func (a *App) redis(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.redisAction(w, r)
		return
	}
	a.render(w, "redis", map[string]any{
		"Title": "Redis",
		"Stats": a.redisStats(),
	})
}

func (a *App) redisAction(w http.ResponseWriter, r *http.Request) {
	action := r.FormValue("action")
	output := ""
	switch action {
	case "flush":
		if err := exec.Command("redis-cli", "FLUSHALL").Run(); err != nil {
			output = "flush failed: " + err.Error()
		} else {
			output = "Redis cache flushed."
			a.audit("redis.flush", "success", "")
		}
	case "restart":
		if err := exec.Command("systemctl", "restart", "redis-server").Run(); err != nil {
			output = "restart failed: " + err.Error()
		} else {
			output = "Redis restarted."
			a.audit("redis.restart", "success", "")
		}
	case "start":
		if err := exec.Command("systemctl", "start", "redis-server").Run(); err != nil {
			output = "start failed: " + err.Error()
		} else {
			output = "Redis started."
			a.audit("redis.start", "success", "")
		}
	case "stop":
		if err := exec.Command("systemctl", "stop", "redis-server").Run(); err != nil {
			output = "stop failed: " + err.Error()
		} else {
			output = "Redis stopped."
			a.audit("redis.stop", "success", "")
		}
	}
	http.Redirect(w, r, "/redis?output="+queryEscape(output), http.StatusSeeOther)
}

// redisStats shells out to redis-cli once for INFO and once for DBSIZE. Both
// return immediately when redis is down so the page still renders.
func (a *App) redisStats() RedisStats {
	stats := RedisStats{Service: "redis-server", HitRate: "n/a", UptimeDays: "n/a"}
	if data, err := exec.Command("systemctl", "is-active", "redis-server").Output(); err == nil {
		stats.Active = strings.TrimSpace(string(data)) == "active"
	}
	if !stats.Active {
		return stats
	}
	out, err := exec.Command("redis-cli", "INFO").Output()
	if err != nil {
		return stats
	}
	info := parseRedisInfo(string(out))
	stats.Version = info["redis_version"]
	stats.UptimeDays = info["uptime_in_days"]
	stats.UsedMemory = info["used_memory_human"]
	stats.PeakMemory = info["used_memory_peak_human"]
	stats.Clients = info["connected_clients"]
	stats.OpsPerSec = info["instantaneous_ops_per_sec"]
	stats.EvictedKeys = info["evicted_keys"]
	hits, _ := strconv.ParseFloat(info["keyspace_hits"], 64)
	misses, _ := strconv.ParseFloat(info["keyspace_misses"], 64)
	if hits+misses > 0 {
		stats.HitRate = strconv.FormatFloat(100*hits/(hits+misses), 'f', 1, 64) + "%"
	}
	if dbs, err := exec.Command("redis-cli", "DBSIZE").Output(); err == nil {
		n, _ := strconv.Atoi(strings.TrimSpace(string(dbs)))
		stats.TotalKeys = n
	}
	return stats
}

func parseRedisInfo(s string) map[string]string {
	info := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		info[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return info
}
