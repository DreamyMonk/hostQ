package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DBCred is a remembered MariaDB user credential used for phpMyAdmin single
// sign-on. Only users whose passwords were created or reset through the panel
// are stored — anything generated manually via the mysql CLI is invisible.
type DBCred struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Domain   string `json:"domain"`
	Updated  string `json:"updated"`
}

var credsMu sync.Mutex

func (a *App) credsPath() string {
	return filepath.Join(a.cfg.DataDir, "db-creds.json")
}

func (a *App) loadCreds() map[string]DBCred {
	credsMu.Lock()
	defer credsMu.Unlock()
	data, err := os.ReadFile(a.credsPath())
	if err != nil {
		return map[string]DBCred{}
	}
	out := map[string]DBCred{}
	_ = json.Unmarshal(data, &out)
	return out
}

func (a *App) saveCreds(creds map[string]DBCred) error {
	credsMu.Lock()
	defer credsMu.Unlock()
	if err := os.MkdirAll(a.cfg.DataDir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(creds, "", "  ")
	return os.WriteFile(a.credsPath(), data, 0600)
}

func (a *App) rememberCred(user, password, domain string) {
	if user == "" || password == "" {
		return
	}
	creds := a.loadCreds()
	creds[user] = DBCred{
		User:     user,
		Password: password,
		Domain:   domain,
		Updated:  time.Now().Format(time.RFC3339),
	}
	_ = a.saveCreds(creds)
}

func (a *App) forgetCred(user string) {
	if user == "" {
		return
	}
	creds := a.loadCreds()
	delete(creds, user)
	_ = a.saveCreds(creds)
}

func (a *App) lookupCred(user string) (DBCred, error) {
	creds := a.loadCreds()
	if c, ok := creds[user]; ok {
		return c, nil
	}
	return DBCred{}, errors.New("no remembered password for user " + user)
}
