package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	// Download intercept: ?download=<path>
	if dl := strings.TrimSpace(r.URL.Query().Get("download")); dl != "" {
		a.fileDownload(w, r, dl)
		return
	}
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
		info, _ := entry.Info()
		size := "-"
		modTime := "-"
		mode := "-"
		if info != nil {
			if !entry.IsDir() {
				size = humanSize(info.Size())
			}
			modTime = info.ModTime().Format("2006-01-02 15:04")
			mode = fmt.Sprintf("%o", info.Mode().Perm())
		}
		items = append(items, FileItem{
			Name: entry.Name(), Kind: kind,
			Path: filepath.ToSlash(filepath.Join(reqPath, entry.Name())),
			Size: size, Mode: mode, ModTime: modTime,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind == "dir"
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	a.render(w, "files", map[string]any{
		"Title":  "File Manager",
		"Path":   reqPath,
		"Items":  items,
		"Crumbs": breadcrumbs(reqPath),
		"Output": r.URL.Query().Get("output"),
	})
}

func breadcrumbs(p string) []Crumb {
	out := []Crumb{{Name: "/", Path: "/"}}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	cur := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur += "/" + part
		out = append(out, Crumb{Name: part, Path: cur})
	}
	return out
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

func (a *App) fileDownload(w http.ResponseWriter, r *http.Request, reqPath string) {
	full := a.safeWebPath(reqPath)
	root := filepath.Clean(a.cfg.WebRoot)
	if full == root || blockedFileName(filepath.Base(full)) {
		http.Error(w, "blocked", http.StatusForbidden)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(full)+"\"")
	a.audit("file.download", "success", full)
	http.ServeFile(w, r, full)
}

func (a *App) fileAction(w http.ResponseWriter, r *http.Request) {
	// Multipart upload has its own size limit + parsing
	action := r.FormValue("action")
	if action == "upload" {
		a.fileUpload(w, r)
		return
	}
	basePath := r.FormValue("path")
	full := a.safeWebPath(basePath)
	name := safeName(r.FormValue("name"))
	output := ""
	switch action {
	case "mkdir":
		if name == "" {
			output = "Folder name required"
			break
		}
		target := filepath.Join(full, name)
		if !a.canMutateWebPath(target) {
			output = "Path is not writable"
			break
		}
		if err := os.MkdirAll(target, 0755); err != nil {
			output = "mkdir failed: " + err.Error()
			break
		}
		output = "Folder created: " + name
		a.audit("file.mkdir", "success", target)
	case "touch":
		if name == "" {
			output = "File name required"
			break
		}
		target := filepath.Join(full, name)
		if !a.canMutateWebPath(target) {
			output = "Path is not writable"
			break
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			output = "create failed: " + err.Error()
			break
		}
		_ = f.Close()
		output = "File created: " + name
		a.audit("file.create", "success", target)
	case "delete":
		deletePath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(deletePath) {
			output = "Cannot delete this path"
			break
		}
		if err := os.RemoveAll(deletePath); err != nil {
			output = "delete failed: " + err.Error()
			break
		}
		output = "Deleted: " + filepath.Base(deletePath)
		a.audit("file.delete", "success", deletePath)
	case "chmod":
		chmodPath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(chmodPath) {
			output = "Cannot chmod this path"
			break
		}
		mode := r.FormValue("mode")
		if !regexp.MustCompile(`^[0-7]{3,4}$`).MatchString(mode) {
			output = "Invalid mode: use octal like 755 or 644"
			break
		}
		parsed, _ := strconv.ParseUint(mode, 8, 32)
		if err := os.Chmod(chmodPath, os.FileMode(parsed)); err != nil {
			output = "chmod failed: " + err.Error()
			break
		}
		output = "Permissions updated to " + mode
		a.audit("file.chmod", "success", chmodPath)
	case "rename":
		from := a.safeWebPath(r.FormValue("target"))
		newName := safeName(r.FormValue("dest"))
		if newName == "" || !a.canMutateWebPath(from) {
			output = "Cannot rename"
			break
		}
		to := filepath.Join(filepath.Dir(from), newName)
		if !a.canMutateWebPath(to) {
			output = "New name is blocked"
			break
		}
		if err := os.Rename(from, to); err != nil {
			output = "rename failed: " + err.Error()
			break
		}
		output = "Renamed to " + newName
		a.audit("file.rename", "success", from+" -> "+to)
	case "move", "copy":
		from := a.safeWebPath(r.FormValue("target"))
		to := a.safeWebPath(r.FormValue("dest"))
		if !a.canMutateWebPath(from) || !a.canMutateWebPath(to) || blockedFileName(filepath.Base(to)) {
			output = "Cannot " + action + " this path"
			break
		}
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			output = "mkdir parent failed: " + err.Error()
			break
		}
		if action == "move" {
			if err := os.Rename(from, to); err != nil {
				output = "move failed: " + err.Error()
				break
			}
			output = "Moved " + filepath.Base(from)
			a.audit("file.move", "success", from+" -> "+to)
			break
		}
		if err := copyAny(from, to); err != nil {
			output = "copy failed: " + err.Error()
			break
		}
		output = "Copied " + filepath.Base(from)
		a.audit("file.copy", "success", from+" -> "+to)
	}
	http.Redirect(w, r, "/files?path="+basePath+"&output="+queryEscape(output), http.StatusSeeOther)
}

func (a *App) fileUpload(w http.ResponseWriter, r *http.Request) {
	basePath := r.FormValue("path")
	full := a.safeWebPath(basePath)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Redirect(w, r, "/files?path="+basePath+"&output="+queryEscape("upload parse failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	files := r.MultipartForm.File["upload"]
	saved := 0
	output := ""
	for _, fh := range files {
		name := safeName(fh.Filename)
		if name == "" || blockedFileName(name) {
			continue
		}
		target := filepath.Join(full, name)
		if !a.canMutateWebPath(target) {
			continue
		}
		src, err := fh.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(target)
		if err != nil {
			_ = src.Close()
			continue
		}
		_, copyErr := io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()
		if copyErr == nil {
			saved++
			a.audit("file.upload", "success", target)
		}
	}
	if saved == 0 {
		output = "No files uploaded (blocked or empty)"
	} else {
		output = fmt.Sprintf("Uploaded %d file(s)", saved)
	}
	http.Redirect(w, r, "/files?path="+basePath+"&output="+queryEscape(output), http.StatusSeeOther)
}

func copyAny(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if blockedFileName(entry.Name()) {
			continue
		}
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(s, d); err != nil {
			return err
		}
	}
	return nil
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

func queryEscape(s string) string {
	r := strings.NewReplacer(" ", "+", "&", "%26", "?", "%3F", "#", "%23", "=", "%3D", "/", "%2F")
	return r.Replace(s)
}
