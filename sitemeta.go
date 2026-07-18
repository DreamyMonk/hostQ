package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Source of truth. Before this file, the only record that a site existed with a
// given docroot / PHP version / cache setting was the generated Nginx vhost
// itself — so when those vhosts went to zero bytes in the 2026-07-18 incident,
// the panel had nothing to regenerate from. SiteMeta fixes that: every site is
// described by a small JSON record under /etc/hostq/sites/<domain>.json, and
// the Nginx config is a disposable artifact rebuilt from it on demand.
//
// The other inputs writeNginxSite needs (backend, aliases, per-site php.ini,
// custom Nginx snippet, SSL certs) are already persisted as their own sidecar
// files under /etc/hostq or /etc/letsencrypt, so {domain, root, cache, php}
// plus those sidecars fully reconstruct the vhost — making generation
// idempotent and recoverable from metadata alone.

type SiteMeta struct {
	Domain     string `json:"domain"`
	Root       string `json:"root"`
	Cache      bool   `json:"cache"`
	PHPVersion string `json:"php"`
	Updated    string `json:"updated,omitempty"`
}

// siteMetaPath returns /etc/hostq/sites/<domain>.json for a valid domain.
func (a *App) siteMetaPath(domain string) string {
	if !domainRe.MatchString(domain) {
		return ""
	}
	return filepath.Join(a.cfg.DataDir, "sites", domain+".json")
}

// saveSiteMeta writes the authoritative record for a site. It is called at the
// start of every writeNginxSite so the metadata is committed before (and
// independent of) the generated artifact.
func (a *App) saveSiteMeta(m SiteMeta) error {
	p := a.siteMetaPath(m.Domain)
	if p == "" {
		return fmt.Errorf("invalid domain %q", m.Domain)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write so a crash mid-save never leaves a truncated metadata file.
	return writeFileAtomic(p, data, 0640)
}

// loadSiteMeta reads a single site's record.
func (a *App) loadSiteMeta(domain string) (SiteMeta, bool) {
	p := a.siteMetaPath(domain)
	if p == "" {
		return SiteMeta{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return SiteMeta{}, false
	}
	var m SiteMeta
	if json.Unmarshal(data, &m) != nil || m.Domain == "" {
		return SiteMeta{}, false
	}
	return m, true
}

// removeSiteMeta deletes a site's record (called on site deletion).
func (a *App) removeSiteMeta(domain string) {
	if p := a.siteMetaPath(domain); p != "" {
		_ = os.Remove(p)
	}
}

// listSiteMeta returns every site record, sorted by domain.
func (a *App) listSiteMeta() []SiteMeta {
	dir := filepath.Join(a.cfg.DataDir, "sites")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	metas := []SiteMeta{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".json")
		if m, ok := a.loadSiteMeta(domain); ok {
			metas = append(metas, m)
		}
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Domain < metas[j].Domain })
	return metas
}

// backfillSiteMeta seeds metadata for any managed vhost that predates the
// metadata store, parsing the site facts back out of the current generated
// config. Runs once at startup so `hostq rebuild` works for existing sites.
func (a *App) backfillSiteMeta() {
	for _, s := range a.listSites() {
		if _, ok := a.loadSiteMeta(s.Domain); ok {
			continue
		}
		if s.Root == "" {
			continue
		}
		_ = a.saveSiteMeta(SiteMeta{
			Domain:     s.Domain,
			Root:       s.Root,
			Cache:      s.Cache,
			PHPVersion: s.PHPVersion,
		})
	}
}

// RebuildResult records the outcome of regenerating one site from metadata.
type RebuildResult struct {
	Domain string
	OK     bool
	Err    string
}

// rebuildFromMetadata regenerates every site's Nginx config from its JSON
// record. Generation is idempotent — running it repeatedly yields identical
// output — because it always derives from metadata + sidecars, never from the
// previously generated file. Each site goes through the safe writer, so a bad
// one is rolled back individually without aborting the whole rebuild.
func (a *App) rebuildFromMetadata() []RebuildResult {
	results := []RebuildResult{}
	for _, m := range a.listSiteMeta() {
		php := m.PHPVersion
		if !phpVersionRe.MatchString(php) {
			php = "8.4"
		}
		err := a.writeNginxSite(m.Domain, m.Root, m.Cache, php)
		r := RebuildResult{Domain: m.Domain, OK: err == nil}
		if err != nil {
			r.Err = err.Error()
		}
		results = append(results, r)
	}
	return results
}
