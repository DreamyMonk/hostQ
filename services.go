package main

import (
	"net/http"
	"os/exec"
	"time"
)

// services renders the unified Services & Packages page. Every entry in
// managedPackages() shows up here with the right action set based on its
// current state (not installed → Install; installed + has service → start/
// stop/restart + Uninstall; installed + no service → Uninstall only).
func (a *App) services(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.servicesAction(w, r)
		return
	}
	a.render(w, "services", map[string]any{
		"Title":    "Services & Packages",
		"Services": a.listManagedUnits(),
	})
}

// listManagedUnits is the new single source of truth. It's also what the
// dashboard preview pulls — both go through the same cache key.
func (a *App) listManagedUnits() []PackageInfo {
	if v, ok := a.cache.get("services"); ok {
		return v.([]PackageInfo)
	}
	out := a.listPackagesInfo()
	a.cache.set("services", out, 3*time.Second)
	return out
}

// listServices keeps the old name available so the dashboard handler and
// install.sh smoke tests don't need to change. It just delegates.
func (a *App) listServices() []PackageInfo { return a.listManagedUnits() }

func (a *App) servicesAction(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	action := r.FormValue("action")
	def, ok := a.findPackageDef(id)
	if !ok {
		http.Redirect(w, r, "/services?output="+queryEscape("Unknown unit "+id), http.StatusSeeOther)
		return
	}
	output := ""
	switch action {
	case "install":
		if def.Apt == "" {
			output = def.Name + " is not managed by apt."
			break
		}
		if def.ID == "phpmyadmin" {
			pmaPreseed()
		}
		out, err := aptInstall(def.Apt)
		if err != nil {
			output = "Install failed: " + tail(string(out), 240)
			a.audit("packages.install", "failure", id)
			break
		}
		if def.Service != "" {
			_ = exec.Command("systemctl", "enable", "--now", def.Service).Run()
		}
		output = def.Name + " installed."
		a.audit("packages.install", "success", id)
		a.cache.invalidate("services", "php")
	case "uninstall":
		if def.Apt == "" {
			output = def.Name + " is not managed by apt."
			break
		}
		out, err := aptRemove(def.Apt)
		if err != nil {
			output = "Uninstall failed: " + tail(string(out), 240)
			a.audit("packages.uninstall", "failure", id)
			break
		}
		output = def.Name + " uninstalled."
		a.audit("packages.uninstall", "success", id)
		a.cache.invalidate("services", "php")
	case "start", "restart":
		if def.Service == "" {
			output = def.Name + " has no systemd service."
			break
		}
		// Auto-install when the unit isn't on disk yet (matches the v0.10.0
		// Redis convenience and generalises it to every unit).
		if def.Apt != "" && !isAptInstalled(def.Apt) {
			if def.ID == "phpmyadmin" {
				pmaPreseed()
			}
			out, err := aptInstall(def.Apt)
			if err != nil {
				output = "auto-install failed: " + tail(string(out), 200)
				a.audit("packages.install", "failure", id)
				break
			}
			a.audit("packages.install", "success", id)
		}
		if err := exec.Command("systemctl", "enable", "--now", def.Service).Run(); err != nil {
			output = "systemctl " + action + " failed: " + err.Error()
			a.audit("service."+action, "failure", id)
			break
		}
		if action == "restart" {
			_ = exec.Command("systemctl", "restart", def.Service).Run()
		}
		output = def.Name + ": " + action + " ok"
		a.audit("service."+action, "success", id)
		a.cache.invalidate("services", "php")
	case "stop":
		if def.Service == "" {
			output = def.Name + " has no systemd service."
			break
		}
		if err := exec.Command("systemctl", "stop", def.Service).Run(); err != nil {
			output = "systemctl stop failed: " + err.Error()
			a.audit("service.stop", "failure", id)
			break
		}
		output = def.Name + " stopped."
		a.audit("service.stop", "success", id)
		a.cache.invalidate("services", "php")
	default:
		output = "Unknown action " + action
	}
	http.Redirect(w, r, "/services?output="+queryEscape(output), http.StatusSeeOther)
}
