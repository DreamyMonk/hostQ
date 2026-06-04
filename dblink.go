package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// site → list of explicitly attached DBs, persisted at
// /etc/hostq/sites/<domain>.dbs.json. Used so the per-site Database tab can
// show databases whose names don't happen to match the auto-derived
// hostq_<domain_underscored>_ prefix.

func (a *App) siteDBLinksPath(domain string) string {
	return filepath.Join(a.cfg.DataDir, "sites", domain+".dbs.json")
}

func (a *App) siteAttachedDBs(domain string) map[string]bool {
	set := map[string]bool{}
	data, err := os.ReadFile(a.siteDBLinksPath(domain))
	if err != nil {
		return set
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return set
	}
	for _, n := range list {
		set[n] = true
	}
	return set
}

func (a *App) writeAttachedDBs(domain string, set map[string]bool) error {
	list := make([]string, 0, len(set))
	for n := range set {
		list = append(list, n)
	}
	sort.Strings(list)
	path := a.siteDBLinksPath(domain)
	if len(list) == 0 {
		_ = os.Remove(path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(path, data, 0600)
}

func (a *App) attachDBToSite(domain, db string) error {
	if !domainRe.MatchString(domain) {
		return errors.New("bad domain")
	}
	if safeDBName(db) == "" {
		return errors.New("bad db")
	}
	set := a.siteAttachedDBs(domain)
	set[db] = true
	return a.writeAttachedDBs(domain, set)
}

func (a *App) detachDBFromSite(domain, db string) error {
	set := a.siteAttachedDBs(domain)
	delete(set, db)
	return a.writeAttachedDBs(domain, set)
}

// detectSiteDBs scans the most common config files in the site's docroot for
// DB names the app is configured against, so the Database tab can surface
// them even when they weren't created through the panel.
var (
	wpConfigDBRe = regexp.MustCompile(`(?i)define\s*\(\s*['"]DB_NAME['"]\s*,\s*['"]([^'"]+)['"]`)
	envDBKeyRe   = regexp.MustCompile(`^(DB_NAME|DB_DATABASE|MYSQL_DATABASE)\s*=\s*['"]?([^'"\s#]+)`)
	phpDBRe      = regexp.MustCompile(`(?i)(?:\$db_name|['"]database['"]|['"]dbname['"])\s*[=:]\s*>?\s*['"]([^'"]+)['"]`)
)

func (a *App) detectSiteDBs(site Site) map[string]bool {
	found := map[string]bool{}
	if site.Root == "" {
		return found
	}
	probe := func(name string, fn func(string)) {
		data, err := os.ReadFile(filepath.Join(site.Root, name))
		if err == nil {
			fn(string(data))
		}
	}
	probe("wp-config.php", func(s string) {
		if m := wpConfigDBRe.FindStringSubmatch(s); m != nil {
			found[m[1]] = true
		}
	})
	probe(".env", func(s string) {
		for _, line := range strings.Split(s, "\n") {
			line = strings.TrimSpace(line)
			if m := envDBKeyRe.FindStringSubmatch(line); m != nil {
				found[m[2]] = true
			}
		}
	})
	for _, name := range []string{"config.php", "includes/config.php", "config/database.php"} {
		probe(name, func(s string) {
			if m := phpDBRe.FindStringSubmatch(s); m != nil {
				found[m[1]] = true
			}
		})
	}
	return found
}
