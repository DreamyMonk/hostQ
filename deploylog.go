package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// A dedicated, append-only journal of every configuration deployment. This is
// separate from the panel's user-facing audit log: it records the transactional
// outcome of each generate → validate → reload cycle (apply / rollback /
// restore, and why), so an operator can reconstruct exactly what the
// configuration layer did around an incident.

type DeployLogEntry struct {
	Time   string `json:"time"`
	Domain string `json:"domain"`
	Action string `json:"action"` // apply, rollback, restore, backup, rebuild
	Result string `json:"result"` // ok, validate-failed, reload-failed, serve-failed, rolled-back, write-failed
	Detail string `json:"detail,omitempty"`
}

// maxDeployLogLines caps the journal; when exceeded it is trimmed to the newest
// half so it never grows without bound.
const maxDeployLogLines = 2000

func (a *App) deployLogPath() string {
	return filepath.Join(a.cfg.DataDir, "deploy-log.jsonl")
}

// deployLog appends one JSONL record describing a deployment outcome. It is
// best-effort: journalling must never break a deployment.
func (a *App) deployLog(domain, action, result, detail string) {
	entry := DeployLogEntry{
		Time:   time.Now().UTC().Format(time.RFC3339),
		Domain: domain,
		Action: action,
		Result: result,
		Detail: tail(detail, 300),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := a.deployLogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return
	}
	_, _ = f.Write(append(data, '\n'))
	_ = f.Close()
	a.trimDeployLog()
}

// trimDeployLog keeps the journal bounded by rewriting it with only its newest
// records once it grows past maxDeployLogLines.
func (a *App) trimDeployLog() {
	lines, err := a.readDeployLog()
	if err != nil || len(lines) <= maxDeployLogLines {
		return
	}
	keep := lines[len(lines)-maxDeployLogLines/2:]
	blob := []byte{}
	for _, l := range keep {
		blob = append(blob, []byte(l+"\n")...)
	}
	_ = writeFileAtomic(a.deployLogPath(), blob, 0640)
}

// readDeployLog returns every journal line in order.
func (a *App) readDeployLog() ([]string, error) {
	f, err := os.Open(a.deployLogPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, sc.Err()
}

// tailDeployLog returns the newest n journal lines.
func (a *App) tailDeployLog(n int) []string {
	lines, err := a.readDeployLog()
	if err != nil {
		return nil
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
