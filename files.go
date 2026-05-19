package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func (a *App) files(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.fileAction(w, r)
		return
	}
	reqPath := filepath.Clean("/" + strings.TrimPrefix(r.URL.Query().Get("path"), "/"))
	full := a.safeWebPath(reqPath)
	entries, _ := os.ReadDir(full)
	items := []FileItem{}
	for _, entry := range entries {
		if blockedFileName(entry.Name()) {
			continue
		}
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		items = append(items, FileItem{Name: entry.Name(), Kind: kind, Path: filepath.Join(reqPath, entry.Name())})
	}
	a.render(w, "files", map[string]any{"Title": "Files", "Path": reqPath, "Items": items})
}

func (a *App) safeWebPath(reqPath string) string {
	clean := filepath.Clean("/" + strings.TrimPrefix(reqPath, "/"))
	full := filepath.Join(a.cfg.WebRoot, clean)
	root := filepath.Clean(a.cfg.WebRoot)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return root
	}
	return full
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, ". ")
}

func blockedFileName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, ".env") ||
		strings.Contains(lower, ".key") ||
		strings.Contains(lower, "id_rsa") ||
		strings.Contains(lower, "id_ed25519") ||
		strings.HasSuffix(lower, ".pem") ||
		strings.HasSuffix(lower, ".p12") ||
		strings.HasSuffix(lower, ".pfx")
}

func (a *App) canMutateWebPath(path string) bool {
	root := filepath.Clean(a.cfg.WebRoot)
	clean := filepath.Clean(path)
	if clean == root || !strings.HasPrefix(clean, root+string(os.PathSeparator)) {
		return false
	}
	return !blockedFileName(filepath.Base(clean))
}

func (a *App) fileAction(w http.ResponseWriter, r *http.Request) {
	basePath := r.FormValue("path")
	full := a.safeWebPath(basePath)
	action := r.FormValue("action")
	name := safeName(r.FormValue("name"))
	target := filepath.Join(full, name)
	if name == "" && (action == "mkdir" || action == "touch") {
		http.Redirect(w, r, "/files?path="+basePath, http.StatusSeeOther)
		return
	}
	switch action {
	case "mkdir":
		if !a.canMutateWebPath(target) {
			break
		}
		_ = os.MkdirAll(target, 0755)
		a.audit("file.mkdir", "success", target)
	case "touch":
		if !a.canMutateWebPath(target) {
			break
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			_ = f.Close()
		}
		a.audit("file.create", "success", target)
	case "delete":
		deletePath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(deletePath) {
			break
		}
		_ = os.RemoveAll(deletePath)
		a.audit("file.delete", "success", deletePath)
	case "chmod":
		chmodPath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(chmodPath) {
			break
		}
		mode := r.FormValue("mode")
		if regexp.MustCompile(`^[0-7]{3,4}$`).MatchString(mode) {
			parsed, _ := strconv.ParseUint(mode, 8, 32)
			_ = os.Chmod(chmodPath, os.FileMode(parsed))
			a.audit("file.chmod", "success", chmodPath)
		}
	case "move", "copy":
		from := a.safeWebPath(r.FormValue("target"))
		to := a.safeWebPath(r.FormValue("dest"))
		if !a.canMutateWebPath(from) || !a.canMutateWebPath(to) || blockedFileName(filepath.Base(to)) {
			break
		}
		_ = os.MkdirAll(filepath.Dir(to), 0755)
		if action == "move" {
			_ = os.Rename(from, to)
			a.audit("file.move", "success", from+" -> "+to)
			break
		}
		if info, err := os.Stat(from); err == nil && !info.IsDir() && copyFile(from, to) == nil {
			a.audit("file.copy", "success", from+" -> "+to)
		}
	}
	http.Redirect(w, r, "/files?path="+basePath, http.StatusSeeOther)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
