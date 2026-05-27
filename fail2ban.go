package main

import (
	"net/http"
	"os/exec"
	"strings"
)

// Fail2banJail captures one jail's runtime state — name, total/current
// banned IPs, plus the live list. We surface every jail fail2ban knows about
// so the operator can see what's actively protecting them (sshd, nginx,
// recidive, …) instead of guessing.
type Fail2banJail struct {
	Name      string
	Enabled   bool
	Currently int
	Total     int
	BannedIPs []string
	Error     string
}

// Fail2banState is the top-level page model.
type Fail2banState struct {
	Installed bool
	Running   bool
	Jails     []Fail2banJail
	Error     string
}

func fail2banInstalled() bool {
	_, err := exec.LookPath("fail2ban-client")
	return err == nil
}

func fail2banRunning() bool {
	out, _ := exec.Command("systemctl", "is-active", "fail2ban").Output()
	return strings.TrimSpace(string(out)) == "active"
}

// listFail2banJails calls fail2ban-client twice: once for the top-level jail
// list, then once per jail to pull live ban counts and banned IPs. We swallow
// per-jail errors into the struct so one stuck jail doesn't blank the page.
func listFail2banJails() []Fail2banJail {
	out, err := exec.Command("fail2ban-client", "status").CombinedOutput()
	if err != nil {
		return nil
	}
	var jailNames []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "`- Jail list:") && !strings.HasPrefix(line, "|- Jail list:") &&
			!strings.HasPrefix(line, "Jail list:") {
			continue
		}
		_, after, _ := strings.Cut(line, "Jail list:")
		for _, raw := range strings.Split(after, ",") {
			n := strings.TrimSpace(raw)
			if n != "" {
				jailNames = append(jailNames, n)
			}
		}
	}
	jails := make([]Fail2banJail, 0, len(jailNames))
	for _, name := range jailNames {
		jails = append(jails, fail2banJailDetails(name))
	}
	return jails
}

func fail2banJailDetails(name string) Fail2banJail {
	j := Fail2banJail{Name: name, Enabled: true}
	out, err := exec.Command("fail2ban-client", "status", name).CombinedOutput()
	if err != nil {
		j.Error = tail(string(out), 200)
		return j
	}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.Contains(line, "Currently banned:"):
			j.Currently = parseIntField(line)
		case strings.Contains(line, "Total banned:"):
			j.Total = parseIntField(line)
		case strings.Contains(line, "Banned IP list:"):
			_, after, _ := strings.Cut(line, "Banned IP list:")
			for _, ip := range strings.Fields(after) {
				if ip != "" {
					j.BannedIPs = append(j.BannedIPs, ip)
				}
			}
		}
	}
	return j
}

// parseIntField pulls the trailing integer from a "Label: N" status line.
func parseIntField(line string) int {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return 0
	}
	n := 0
	for _, c := range parts[len(parts)-1] {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func unbanFail2ban(jail, ip string) (string, error) {
	if jail == "" || ip == "" {
		return "Missing jail or IP", nil
	}
	out, err := exec.Command("fail2ban-client", "set", jail, "unbanip", ip).CombinedOutput()
	if err != nil {
		return tail(string(out), 200), err
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *App) firewall(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.firewallAction(w, r)
		return
	}
	state := Fail2banState{
		Installed: fail2banInstalled(),
		Running:   fail2banRunning(),
	}
	if state.Installed && state.Running {
		state.Jails = listFail2banJails()
	}
	a.render(w, "firewall", map[string]any{
		"Title":  "Firewall · fail2ban",
		"State":  state,
		"Output": r.URL.Query().Get("output"),
	})
}

func (a *App) firewallAction(w http.ResponseWriter, r *http.Request) {
	output := ""
	switch r.FormValue("action") {
	case "install":
		out, err := aptInstall("fail2ban")
		if err != nil {
			output = "Install failed: " + tail(string(out), 240)
			a.audit("firewall.install", "failure", "")
		} else {
			_ = exec.Command("systemctl", "enable", "--now", "fail2ban").Run()
			output = "fail2ban installed and started."
			a.audit("firewall.install", "success", "")
		}
	case "start":
		if err := exec.Command("systemctl", "enable", "--now", "fail2ban").Run(); err != nil {
			output = "Start failed: " + err.Error()
			a.audit("firewall.start", "failure", "")
		} else {
			output = "fail2ban started."
			a.audit("firewall.start", "success", "")
		}
	case "stop":
		if err := exec.Command("systemctl", "stop", "fail2ban").Run(); err != nil {
			output = "Stop failed: " + err.Error()
			a.audit("firewall.stop", "failure", "")
		} else {
			output = "fail2ban stopped."
			a.audit("firewall.stop", "success", "")
		}
	case "reload":
		out, err := exec.Command("fail2ban-client", "reload").CombinedOutput()
		if err != nil {
			output = "Reload failed: " + tail(string(out), 200)
			a.audit("firewall.reload", "failure", "")
		} else {
			output = "Configuration reloaded."
			a.audit("firewall.reload", "success", "")
		}
	case "unban":
		jail := r.FormValue("jail")
		ip := r.FormValue("ip")
		msg, err := unbanFail2ban(jail, ip)
		if err != nil {
			output = "Unban failed: " + msg
			a.audit("firewall.unban", "failure", jail+" "+ip)
		} else {
			output = "Unbanned " + ip + " from " + jail + ": " + msg
			a.audit("firewall.unban", "success", jail+" "+ip)
		}
	default:
		output = "Unknown action."
	}
	http.Redirect(w, r, "/firewall?output="+queryEscape(output), http.StatusSeeOther)
}
