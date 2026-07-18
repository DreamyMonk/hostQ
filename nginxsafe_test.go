package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTTPProbeAddr(t *testing.T) {
	// A stand-in for Nginx: 200 only when the Host header matches the vhost.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "example.com" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	// Any HTTP response (even 404 for the wrong Host) means "Nginx answered".
	if !httpProbeAddr(addr, "example.com", false, 2*time.Second) {
		t.Fatal("probe should succeed against a live listener")
	}
	if !httpProbeAddr(addr, "other.com", false, 2*time.Second) {
		t.Fatal("probe should still succeed — a 404 is a valid response")
	}
	// A dead address (nothing listening) must read as not-serving → rollback.
	if httpProbeAddr("127.0.0.1:1", "example.com", false, 300*time.Millisecond) {
		t.Fatal("probe must fail when nothing is listening")
	}
}

func newTestApp(t *testing.T) (*App, string, string) {
	t.Helper()
	base := t.TempDir()
	sitesDir := filepath.Join(base, "nginx", "sites-available")
	if err := os.MkdirAll(sitesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "nginx", "sites-enabled"), 0755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(base, "hostq")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	app := &App{cfg: Config{DataDir: dataDir, NginxSitesDir: sitesDir}, cache: newMemCache()}
	return app, sitesDir, dataDir
}

func TestWriteFileAtomicReplacesWhole(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "conf")
	if err := os.WriteFile(p, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(p, []byte("new-content"), 0644); err != nil {
		t.Fatalf("atomic write: %v", err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "new-content" {
		t.Fatalf("got %q want %q", got, "new-content")
	}
	// No leftover temp files beside it.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected only the target file, found %d entries", len(entries))
	}
}

func TestIsBlankConf(t *testing.T) {
	for _, s := range []string{"", "   ", "\n\t\n"} {
		if !isBlankConf([]byte(s)) {
			t.Fatalf("expected %q to be blank", s)
		}
	}
	if isBlankConf([]byte("server {}")) {
		t.Fatal("non-empty config must not be blank")
	}
}

func TestSeedAndScanAndRestore(t *testing.T) {
	app, sitesDir, _ := newTestApp(t)
	// A healthy managed vhost on disk, no backup yet.
	good := "# hostQ managed - example.com\nserver { listen 80; }\n"
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	// A non-managed file must be ignored by seeding.
	if err := os.WriteFile(filepath.Join(sitesDir, "default"), []byte("server {}"), 0644); err != nil {
		t.Fatal(err)
	}

	app.seedConfBackups()
	if _, err := os.Stat(filepath.Join(app.nginxConfBackupDir(), "example.com")); err != nil {
		t.Fatalf("expected example.com to be backed up: %v", err)
	}
	if _, err := os.Stat(filepath.Join(app.nginxConfBackupDir(), "default")); err == nil {
		t.Fatal("non-managed vhost must not be backed up")
	}

	// Simulate the outage: truncate the live config to zero bytes.
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	fixable, unbacked := app.scanBrokenConfigs()
	if len(fixable) != 1 || fixable[0] != "example.com" {
		t.Fatalf("expected example.com fixable, got fixable=%v unbacked=%v", fixable, unbacked)
	}

	restored := app.restoreConfigs(fixable)
	if len(restored) != 1 {
		t.Fatalf("expected 1 restored, got %v", restored)
	}
	got, _ := os.ReadFile(filepath.Join(sitesDir, "example.com"))
	if string(got) != good {
		t.Fatalf("restore mismatch: got %q want %q", got, good)
	}
}

func TestSaveGoodConfRefusesBlank(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.saveGoodConf("example.com", []byte("   "))
	if _, err := os.Stat(filepath.Join(app.nginxConfBackupDir(), "example.com")); err == nil {
		t.Fatal("blank config must never be stored as a good backup")
	}
}

func TestSiteMetaRoundTrip(t *testing.T) {
	app, _, _ := newTestApp(t)
	m := SiteMeta{Domain: "example.com", Root: "/var/www/example.com/htdocs", Cache: true, PHPVersion: "8.3"}
	if err := app.saveSiteMeta(m); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok := app.loadSiteMeta("example.com")
	if !ok || got.Root != m.Root || got.Cache != m.Cache || got.PHPVersion != m.PHPVersion {
		t.Fatalf("load mismatch: %+v ok=%v", got, ok)
	}
	if list := app.listSiteMeta(); len(list) != 1 || list[0].Domain != "example.com" {
		t.Fatalf("list mismatch: %+v", list)
	}
	app.removeSiteMeta("example.com")
	if _, ok := app.loadSiteMeta("example.com"); ok {
		t.Fatal("meta should be gone after remove")
	}
}

func TestBackfillSiteMetaFromVhost(t *testing.T) {
	app, sitesDir, _ := newTestApp(t)
	vhost := "# hostQ managed - example.com\n# hostQ php: 8.3\nserver {\n  root /var/www/example.com/htdocs;\n  index index.php;\n}\n"
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(vhost), 0644); err != nil {
		t.Fatal(err)
	}
	app.backfillSiteMeta()
	m, ok := app.loadSiteMeta("example.com")
	if !ok {
		t.Fatal("expected metadata backfilled from existing vhost")
	}
	if m.Root != "/var/www/example.com/htdocs" || m.PHPVersion != "8.3" {
		t.Fatalf("backfill parsed wrong facts: %+v", m)
	}
}
