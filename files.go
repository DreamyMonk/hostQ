package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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
	// Trim trailing dots/spaces only — leading dots are valid for hidden
	// files (.htaccess, .gitignore, .user.ini). Reject the bare path
	// indicators "." and ".." explicitly.
	name = strings.TrimRight(name, ". ")
	if name == "." || name == ".." {
		return ""
	}
	return name
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

// apiDirs lists subdirectories under the requested path so the browser-side
// folder picker can navigate the tree without needing a separate /files
// render. JSON shape: {path, up?, items:[{name, path}]}.
func (a *App) apiDirs(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if reqPath == "" {
		reqPath = "/"
	}
	reqPath = filepath.ToSlash(filepath.Clean("/" + strings.TrimPrefix(reqPath, "/")))
	full := a.safeWebPath(reqPath)
	type item struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	resp := struct {
		Path  string `json:"path"`
		Up    string `json:"up,omitempty"`
		Items []item `json:"items"`
	}{Path: reqPath, Items: []item{}}
	if reqPath != "/" {
		resp.Up = filepath.ToSlash(filepath.Dir(reqPath))
	}
	if entries, err := os.ReadDir(full); err == nil {
		for _, e := range entries {
			if !e.IsDir() || blockedFileName(e.Name()) {
				continue
			}
			resp.Items = append(resp.Items, item{
				Name: e.Name(),
				Path: filepath.ToSlash(filepath.Join(reqPath, e.Name())),
			})
		}
	}
	sort.Slice(resp.Items, func(i, j int) bool {
		return strings.ToLower(resp.Items[i].Name) < strings.ToLower(resp.Items[j].Name)
	})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

// looksTextual is a cheap heuristic: a NUL byte in the first 8 KB means binary.
// Good enough to keep the in-browser editor from being handed an image or zip.
func looksTextual(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return false
		}
	}
	return true
}

// fileEditMaxBytes caps in-browser editing. Bigger files should be downloaded
// and edited locally. The cap also matches Go's default form-parse budget
// (10 MB) with a comfortable margin.
const fileEditMaxBytes = 2 * 1024 * 1024

// fileEdit renders the in-browser text editor, or saves a POSTed edit.
// Handles dot-files like .htaccess natively — the only files it refuses are
// the secret patterns blockedFileName() already filters everywhere.
func (a *App) fileEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.fileEditSave(w, r)
		return
	}
	reqPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if reqPath == "" {
		http.Redirect(w, r, "/files?path=/", http.StatusSeeOther)
		return
	}
	full := a.safeWebPath(reqPath)
	parent := filepath.ToSlash(filepath.Dir(reqPath))
	if !a.canMutateWebPath(full) {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("Cannot edit that path"), http.StatusSeeOther)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("Not a file"), http.StatusSeeOther)
		return
	}
	if info.Size() > fileEditMaxBytes {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("File too large to edit in browser (>2 MB) — download instead"), http.StatusSeeOther)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("read failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if !looksTextual(data) {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("Looks like a binary file — download and edit locally"), http.StatusSeeOther)
		return
	}
	a.render(w, "fileedit", map[string]any{
		"Title":   "Edit " + info.Name(),
		"Path":    reqPath,
		"Parent":  parent,
		"Name":    info.Name(),
		"Content": string(data),
		"Mode":    fmt.Sprintf("%o", info.Mode().Perm()),
		"Size":    humanSize(info.Size()),
		"ModTime": info.ModTime().Format("2006-01-02 15:04"),
		"Output":  r.URL.Query().Get("output"),
	})
}

func (a *App) fileEditSave(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.TrimSpace(r.FormValue("path"))
	content := r.FormValue("content")
	full := a.safeWebPath(reqPath)
	parent := filepath.ToSlash(filepath.Dir(reqPath))
	if !a.canMutateWebPath(full) {
		http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("Cannot save to that path"), http.StatusSeeOther)
		return
	}
	var mode os.FileMode = 0644
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(full, []byte(content), mode); err != nil {
		http.Redirect(w, r, "/file-edit?path="+url.QueryEscape(reqPath)+"&output="+queryEscape("save failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	a.audit("file.edit", "success", full)
	http.Redirect(w, r, "/files?path="+url.QueryEscape(parent)+"&output="+queryEscape("Saved "+filepath.Base(full)), http.StatusSeeOther)
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
	// Multipart uploads need a much larger parse budget than the rest of
	// the actions. Dispatch *before* FormValue triggers the default 32 MB
	// ParseMultipartForm — once the body is parsed at that limit, calling
	// ParseMultipartForm again with a larger budget is a no-op.
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		a.fileUpload(w, r)
		return
	}
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
	case "writable":
		targetPath := a.safeWebPath(r.FormValue("target"))
		if !a.canMutateWebPath(targetPath) {
			output = "Cannot make that path writable"
			break
		}
		if err := makePathWritable(targetPath); err != nil {
			output = "make writable failed: " + err.Error()
			a.audit("file.writable", "failure", targetPath)
			break
		}
		output = "Made writable for www-data: " + filepath.Base(targetPath)
		a.audit("file.writable", "success", targetPath)
	case "bulk-writable":
		_ = r.ParseForm()
		count, failed := 0, 0
		for _, t := range r.PostForm["target"] {
			p := a.safeWebPath(t)
			if !a.canMutateWebPath(p) {
				failed++
				continue
			}
			if err := makePathWritable(p); err != nil {
				failed++
				continue
			}
			count++
			a.audit("file.writable", "success", p)
		}
		output = fmt.Sprintf("Made %d item(s) writable for www-data", count)
		if failed > 0 {
			output += fmt.Sprintf(" · %d failed", failed)
		}
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
	case "bulk-delete":
		_ = r.ParseForm()
		count, failed := 0, 0
		for _, t := range r.PostForm["target"] {
			p := a.safeWebPath(t)
			if !a.canMutateWebPath(p) {
				failed++
				continue
			}
			if err := os.RemoveAll(p); err == nil {
				count++
				a.audit("file.delete", "success", p)
			} else {
				failed++
			}
		}
		output = fmt.Sprintf("Deleted %d item(s)", count)
		if failed > 0 {
			output += fmt.Sprintf(" · %d blocked or failed", failed)
		}
	case "bulk-chmod":
		_ = r.ParseForm()
		mode := r.FormValue("mode")
		if !regexp.MustCompile(`^[0-7]{3,4}$`).MatchString(mode) {
			output = "Invalid mode: use octal like 755 or 644"
			break
		}
		parsed, _ := strconv.ParseUint(mode, 8, 32)
		count := 0
		for _, t := range r.PostForm["target"] {
			p := a.safeWebPath(t)
			if !a.canMutateWebPath(p) {
				continue
			}
			if err := os.Chmod(p, os.FileMode(parsed)); err == nil {
				count++
				a.audit("file.chmod", "success", p)
			}
		}
		output = fmt.Sprintf("Set mode %s on %d item(s)", mode, count)
	case "bulk-move":
		_ = r.ParseForm()
		dest := strings.TrimSpace(r.FormValue("dest"))
		if dest == "" {
			output = "Destination required"
			break
		}
		dp := a.safeWebPath(dest)
		if info, err := os.Stat(dp); err != nil || !info.IsDir() {
			output = "Destination must be an existing directory under /var/www"
			break
		}
		count := 0
		for _, t := range r.PostForm["target"] {
			from := a.safeWebPath(t)
			if !a.canMutateWebPath(from) {
				continue
			}
			to := filepath.Join(dp, filepath.Base(from))
			if err := os.Rename(from, to); err == nil {
				count++
				a.audit("file.move", "success", from+" -> "+to)
			}
		}
		output = fmt.Sprintf("Moved %d item(s) into %s", count, dest)
	case "bulk-copy":
		_ = r.ParseForm()
		dest := strings.TrimSpace(r.FormValue("dest"))
		if dest == "" {
			output = "Destination required"
			break
		}
		dp := a.safeWebPath(dest)
		if info, err := os.Stat(dp); err != nil || !info.IsDir() {
			output = "Destination must be an existing directory under /var/www"
			break
		}
		count, failed := 0, 0
		for _, t := range r.PostForm["target"] {
			from := a.safeWebPath(t)
			if !a.canMutateWebPath(from) {
				failed++
				continue
			}
			to := filepath.Join(dp, filepath.Base(from))
			if err := copyAny(from, to); err == nil {
				count++
				a.audit("file.copy", "success", from+" -> "+to)
			} else {
				failed++
			}
		}
		output = fmt.Sprintf("Copied %d item(s) into %s", count, dest)
		if failed > 0 {
			output += fmt.Sprintf(" · %d failed", failed)
		}
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
	log.Printf("upload: start path=%q content-length=%d ua=%q", basePath, r.ContentLength, r.Header.Get("User-Agent"))
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		log.Printf("upload: parse failed path=%q err=%v", basePath, err)
		http.Redirect(w, r, "/files?path="+url.QueryEscape(basePath)+"&output="+queryEscape("upload parse failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if r.MultipartForm == nil {
		log.Printf("upload: no multipart form path=%q", basePath)
		http.Redirect(w, r, "/files?path="+url.QueryEscape(basePath)+"&output="+queryEscape("upload received no form data"), http.StatusSeeOther)
		return
	}
	files := r.MultipartForm.File["upload"]
	log.Printf("upload: received %d file part(s) for %q", len(files), basePath)
	saved, dirs := 0, 0
	skipped := 0
	output := ""
	for _, fh := range files {
		// With <input webkitdirectory>, browsers send the file's full path
		// relative to the picked folder, e.g. "site/css/app.css", in the
		// multipart part's Content-Disposition filename parameter.
		//
		// Go's mime/multipart, however, runs filepath.Base() on the filename
		// it exposes as fh.Filename, so any directory tree is stripped before
		// our handler sees it. Parse the raw Content-Disposition header
		// ourselves to recover the original path. Falls back to fh.Filename
		// for clients that don't include a directory.
		raw := fh.Filename
		if cd := fh.Header.Get("Content-Disposition"); cd != "" {
			if _, params, err := mime.ParseMediaType(cd); err == nil {
				if name := params["filename"]; name != "" {
					raw = name
				}
			}
		}
		raw = strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
		raw = strings.TrimPrefix(raw, "/")
		if raw == "" {
			skipped++
			continue
		}
		segments := strings.Split(raw, "/")
		cleaned := make([]string, 0, len(segments))
		blocked := false
		for _, seg := range segments {
			s := safeName(seg)
			if s == "" || s == "." || s == ".." || blockedFileName(s) {
				blocked = true
				break
			}
			cleaned = append(cleaned, s)
		}
		if blocked || len(cleaned) == 0 {
			skipped++
			continue
		}
		rel := filepath.Join(cleaned...)
		target := filepath.Join(full, rel)
		if !a.canMutateWebPath(target) {
			skipped++
			continue
		}
		if len(cleaned) > 1 {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				skipped++
				continue
			}
			dirs++
		}
		src, err := fh.Open()
		if err != nil {
			skipped++
			continue
		}
		dst, err := os.Create(target)
		if err != nil {
			_ = src.Close()
			skipped++
			continue
		}
		_, copyErr := io.Copy(dst, src)
		_ = src.Close()
		_ = dst.Close()
		if copyErr == nil {
			saved++
			a.audit("file.upload", "success", target)
		} else {
			skipped++
		}
	}
	switch {
	case saved == 0 && skipped == 0:
		output = "No files in upload"
	case saved == 0:
		output = "No files uploaded (blocked, empty, or write failed)"
	default:
		output = fmt.Sprintf("Uploaded %d file(s)", saved)
		if dirs > 0 {
			output += " across nested folders"
		}
		if skipped > 0 {
			output += fmt.Sprintf(" · %d skipped", skipped)
		}
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

// makePathWritable hands ownership of a file or directory tree to www-data and
// sets a g+w-friendly mode so PHP-FPM (which runs as www-data) can write into
// it. This is the standard "config/, assets/, uploads/, storage/" treatment
// most PHP apps need before their installer wizard will let you continue.
//
// Files end up at 664 (owner+group rw, world r). Directories at 775 (owner+
// group rwx, world rx) so PHP can create new files inside. The site's other
// owner (likely the panel's root) still has full access.
func makePathWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if out, err := exec.Command("chown", "-R", "www-data:www-data", path).CombinedOutput(); err != nil {
		return fmt.Errorf("chown: %s", strings.TrimSpace(string(out)))
	}
	if info.IsDir() {
		if out, err := exec.Command("find", path, "-type", "d", "-exec", "chmod", "775", "{}", "+").CombinedOutput(); err != nil {
			return fmt.Errorf("chmod dirs: %s", strings.TrimSpace(string(out)))
		}
		if out, err := exec.Command("find", path, "-type", "f", "-exec", "chmod", "664", "{}", "+").CombinedOutput(); err != nil {
			return fmt.Errorf("chmod files: %s", strings.TrimSpace(string(out)))
		}
		return nil
	}
	return os.Chmod(path, 0664)
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

// queryEscape escapes an arbitrary string so it is safe to drop into a
// "?output=" query parameter. The previous homebrew Replacer left "%" and
// other special characters un-escaped, which broke browser-side URL parsing
// (and made some XHR redirects look like "network error"). net/url does the
// right thing for every byte.
func queryEscape(s string) string {
	return url.QueryEscape(s)
}
