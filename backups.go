package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) backups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.backupAction(w, r)
		return
	}
	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("site")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	if name := strings.TrimSpace(r.URL.Query().Get("download")); name != "" {
		a.downloadBackup(w, r, site, name)
		return
	}
	a.render(w, "backups", map[string]any{
		"Title":   "Backups",
		"Site":    site,
		"Backups": a.listBackups(site.Domain),
		"Policy":  a.backupPolicy(site.Domain),
		"Output":  r.URL.Query().Get("output"),
	})
}

func (a *App) backupAction(w http.ResponseWriter, r *http.Request) {
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	site, ok := a.findSite(domain)
	if !ok {
		http.Redirect(w, r, "/sites", http.StatusSeeOther)
		return
	}
	action := r.FormValue("action")
	output := ""
	switch action {
	case "create":
		path, err := a.createSiteBackup(site)
		if err != nil {
			output = "Backup failed: " + err.Error()
			a.audit("backup.create", "failure", site.Domain)
			break
		}
		output = "Backup created: " + filepath.Base(path)
		a.audit("backup.create", "success", site.Domain)
	case "policy":
		policy := BackupPolicy{
			Domain:    site.Domain,
			Frequency: cleanBackupFrequency(r.FormValue("frequency")),
			Keep:      cleanInt(r.FormValue("keep"), 7, 1, 365),
			Hour:      cleanInt(r.FormValue("hour"), 3, 0, 23),
			MaxLoad:   strings.TrimSpace(r.FormValue("max_load")),
		}
		if policy.MaxLoad == "" {
			policy.MaxLoad = "1.50"
		}
		if err := a.saveBackupPolicy(policy); err != nil {
			output = "Policy save failed: " + err.Error()
		} else {
			output = "Automatic backup policy saved."
		}
	case "delete":
		name := filepath.Base(r.FormValue("name"))
		if err := os.Remove(a.backupPath(site.Domain, name)); err != nil {
			output = "Delete failed: " + err.Error()
		} else {
			output = "Backup deleted."
			a.audit("backup.delete", "success", site.Domain+"/"+name)
		}
	case "restore":
		mode := r.FormValue("mode")
		name := filepath.Base(r.FormValue("name"))
		err := a.restoreBackup(site, a.backupPath(site.Domain, name), mode)
		if err != nil {
			output = "Restore failed: " + err.Error()
			a.audit("backup.restore", "failure", site.Domain+"/"+mode)
		} else {
			output = "Restore complete: " + mode
			a.audit("backup.restore", "success", site.Domain+"/"+mode)
		}
	}
	http.Redirect(w, r, "/backups?site="+site.Domain+"&output="+template.URLQueryEscaper(output), http.StatusSeeOther)
}

func (a *App) backupRoot(domain string) string {
	return filepath.Join("/var/backups/hostq/sites", domain)
}

func (a *App) backupPath(domain, name string) string {
	return filepath.Join(a.backupRoot(domain), filepath.Base(name))
}

func (a *App) createSiteBackup(site Site) (string, error) {
	if err := os.MkdirAll(a.backupRoot(site.Domain), 0750); err != nil {
		return "", err
	}
	target := filepath.Join(a.backupRoot(site.Domain), fmt.Sprintf("%s-%s.zip", site.Domain, time.Now().Format("2006-01-02-150405")))
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	defer zw.Close()
	manifest := map[string]string{
		"domain":    site.Domain,
		"root":      site.Root,
		"createdAt": time.Now().Format(time.RFC3339),
		"format":    "hostq-site-backup-v1",
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if err := addZipBytes(zw, "manifest.json", data); err != nil {
		return "", err
	}
	if err := addZipDirectory(zw, site.Root, "files"); err != nil {
		return "", err
	}
	dbName := safeDBName(strings.ReplaceAll(site.Domain, ".", "_"))
	if out, err := exec.Command("mysqldump", "--single-transaction", "--skip-lock-tables", "--databases", dbName).Output(); err == nil && len(out) > 0 {
		if err := addZipBytes(zw, "database.sql", out); err != nil {
			return "", err
		}
	}
	a.pruneBackups(site.Domain, a.backupPolicy(site.Domain).Keep)
	return target, nil
}

func addZipDirectory(zw *zip.Writer, root, prefix string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if blockedFileName(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		name := prefix + "/" + rel
		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		writer, err := zw.Create(name)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer src.Close()
		_, err = io.Copy(writer, src)
		return err
	})
}

func addZipBytes(zw *zip.Writer, name string, data []byte) error {
	writer, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

func (a *App) listBackups(domain string) []BackupInfo {
	entries, err := os.ReadDir(a.backupRoot(domain))
	if err != nil {
		return nil
	}
	backups := []BackupInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Name:    entry.Name(),
			Domain:  domain,
			Path:    filepath.Join(a.backupRoot(domain), entry.Name()),
			Size:    humanSize(info.Size()),
			Created: info.ModTime().Format("2006-01-02 15:04"),
		})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].Created > backups[j].Created })
	return backups
}

func (a *App) downloadBackup(w http.ResponseWriter, r *http.Request, site Site, name string) {
	path := a.backupPath(site.Domain, name)
	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(path)+"\"")
	http.ServeFile(w, r, path)
}

func (a *App) restoreBackup(site Site, archivePath, mode string) error {
	if mode != "full" && mode != "files" && mode != "database" {
		return fmt.Errorf("invalid restore mode")
	}
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()
	if mode == "full" || mode == "files" {
		for _, file := range zr.File {
			if !strings.HasPrefix(file.Name, "files/") || strings.HasSuffix(file.Name, "/") {
				continue
			}
			rel := strings.TrimPrefix(file.Name, "files/")
			if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(rel) || blockedFileName(filepath.Base(rel)) {
				return fmt.Errorf("unsafe file in backup: %s", file.Name)
			}
			target := filepath.Join(site.Root, filepath.FromSlash(rel))
			if !a.canMutateWebPath(target) {
				return fmt.Errorf("restore target blocked: %s", rel)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			src, err := file.Open()
			if err != nil {
				return err
			}
			dst, err := os.Create(target)
			if err != nil {
				_ = src.Close()
				return err
			}
			_, copyErr := io.Copy(dst, src)
			_ = src.Close()
			_ = dst.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	if mode == "full" || mode == "database" {
		for _, file := range zr.File {
			if file.Name != "database.sql" {
				continue
			}
			src, err := file.Open()
			if err != nil {
				return err
			}
			cmd := exec.Command("mysql")
			cmd.Stdin = src
			err = cmd.Run()
			_ = src.Close()
			return err
		}
		if mode == "database" {
			return fmt.Errorf("backup has no database.sql")
		}
	}
	return nil
}

func (a *App) backupPolicy(domain string) BackupPolicy {
	policies := a.readBackupPolicies()
	if policy, ok := policies[domain]; ok {
		if policy.Keep < 1 {
			policy.Keep = 7
		}
		return policy
	}
	return BackupPolicy{Domain: domain, Frequency: "daily", Keep: 7, Hour: 3, MaxLoad: "1.50"}
}

func (a *App) readBackupPolicies() map[string]BackupPolicy {
	path := filepath.Join(a.cfg.DataDir, "backup-policies.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]BackupPolicy{}
	}
	var policies map[string]BackupPolicy
	if json.Unmarshal(data, &policies) != nil {
		return map[string]BackupPolicy{}
	}
	return policies
}

func (a *App) saveBackupPolicy(policy BackupPolicy) error {
	policies := a.readBackupPolicies()
	policies[policy.Domain] = policy
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(policies, "", "  ")
	return os.WriteFile(filepath.Join(a.cfg.DataDir, "backup-policies.json"), data, 0600)
}

func (a *App) runScheduledBackups() error {
	for _, site := range a.listSites() {
		policy := a.backupPolicy(site.Domain)
		if !backupDue(policy, time.Now()) || !loadIsLow(policy.MaxLoad) {
			continue
		}
		if _, err := a.createSiteBackup(site); err != nil {
			a.audit("backup.auto", "failure", site.Domain)
			continue
		}
		policy.LastRun = time.Now().Format(time.RFC3339)
		_ = a.saveBackupPolicy(policy)
		a.audit("backup.auto", "success", site.Domain)
	}
	return nil
}

func backupDue(policy BackupPolicy, now time.Time) bool {
	if now.Hour() != policy.Hour {
		return false
	}
	last, _ := time.Parse(time.RFC3339, policy.LastRun)
	switch policy.Frequency {
	case "weekly":
		return last.IsZero() || now.Sub(last) >= 7*24*time.Hour
	case "monthly":
		return last.IsZero() || now.Month() != last.Month() || now.Year() != last.Year()
	default:
		return last.IsZero() || now.Sub(last) >= 24*time.Hour
	}
}

func loadIsLow(maxLoad string) bool {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return true
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return true
	}
	current, err1 := strconv.ParseFloat(fields[0], 64)
	max, err2 := strconv.ParseFloat(maxLoad, 64)
	return err1 != nil || err2 != nil || current <= max
}

func (a *App) pruneBackups(domain string, keep int) {
	if keep < 1 {
		keep = 7
	}
	backups := a.listBackups(domain)
	for i := keep; i < len(backups); i++ {
		_ = os.Remove(backups[i].Path)
	}
}

func cleanBackupFrequency(value string) string {
	switch value {
	case "weekly", "monthly":
		return value
	default:
		return "daily"
	}
}

func cleanInt(value string, fallback, min, max int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func humanSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	value := float64(size)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value = value / 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
