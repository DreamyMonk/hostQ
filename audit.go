package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (a *App) audit(action, status, target string) {
	_ = os.MkdirAll(a.cfg.DataDir, 0700)
	file := filepath.Join(a.cfg.DataDir, "audit.log")
	line, _ := json.Marshal(map[string]string{"ts": time.Now().Format(time.RFC3339), "action": action, "status": status, "target": target})
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer f.Close()
		_, _ = f.Write(append(line, '\n'))
	}
}

func (a *App) render(w http.ResponseWriter, view string, data map[string]any) {
	data["View"] = view
	if _, ok := data["PaletteSites"]; !ok && view != "login" {
		data["PaletteSites"] = a.listSites()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
