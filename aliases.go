package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Per-site alias list — stored at /etc/hostq/sites/<domain>.aliases.json.
// Each entry is just another hostname (e.g. "www2.example.com" or
// "example.org") that the panel adds to the site's server_name list so the
// existing vhost serves both. Subdomains with their own document root and
// addon domains are managed as separate sites — alias is the "share the same
// content" case.

func (a *App) siteAliasesPath(domain string) string {
	return filepath.Join(a.cfg.DataDir, "sites", domain+".aliases.json")
}

func (a *App) loadAliases(domain string) []string {
	data, err := os.ReadFile(a.siteAliasesPath(domain))
	if err != nil {
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil
	}
	return list
}

func (a *App) saveAliases(domain string, list []string) error {
	out := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, n := range list {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" || n == domain || seen[n] {
			continue
		}
		if !domainRe.MatchString(n) {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	path := a.siteAliasesPath(domain)
	if len(out) == 0 {
		_ = os.Remove(path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func (a *App) addAlias(domain, alias string) error {
	if !domainRe.MatchString(alias) {
		return errors.New("invalid alias")
	}
	list := a.loadAliases(domain)
	list = append(list, alias)
	return a.saveAliases(domain, list)
}

func (a *App) removeAlias(domain, alias string) error {
	cur := a.loadAliases(domain)
	out := make([]string, 0, len(cur))
	for _, n := range cur {
		if n != alias {
			out = append(out, n)
		}
	}
	return a.saveAliases(domain, out)
}
