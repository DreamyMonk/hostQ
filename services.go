package main

import (
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

func (a *App) services(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		action := r.FormValue("action")
		if allowedServiceAction(id, action) {
			_ = exec.Command("systemctl", action, serviceMap()[id]).Run()
			a.cache.invalidate("services")
			a.audit("service."+action, "success", id)
		}
		http.Redirect(w, r, "/services", http.StatusSeeOther)
		return
	}
	a.render(w, "services", map[string]any{"Title": "Services", "Services": a.listServices()})
}

func serviceMap() map[string]string {
	return map[string]string{
		"nginx": "nginx", "mariadb": "mariadb", "redis": "redis-server",
		"php84": "php8.4-fpm", "pureftpd": "pure-ftpd",
	}
}

func allowedServiceAction(id, action string) bool {
	if _, ok := serviceMap()[id]; !ok {
		return false
	}
	return action == "start" || action == "stop" || action == "restart"
}

func (a *App) listServices() []Service {
	if v, ok := a.cache.get("services"); ok {
		return v.([]Service)
	}
	labels := map[string]string{"nginx": "Nginx", "mariadb": "MariaDB", "redis": "Redis", "php84": "PHP 8.4-FPM", "pureftpd": "Pure-FTPd"}
	out := []Service{}
	for id, systemd := range serviceMap() {
		status := "unknown"
		if data, err := exec.Command("systemctl", "is-active", systemd).Output(); err == nil {
			status = strings.TrimSpace(string(data))
		}
		out = append(out, Service{ID: id, Name: labels[id], Systemd: systemd, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	a.cache.set("services", out, 3*time.Second)
	return out
}
