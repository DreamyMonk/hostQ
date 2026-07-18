package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestConfRevisionHistoryRotates(t *testing.T) {
	app, _, _ := newTestApp(t)
	// Save more than the cap, each with distinct content.
	for i := 0; i < maxConfRevisions+5; i++ {
		app.saveConfRevision("example.com", []byte(fmt.Sprintf("# hostQ managed - example.com\n# rev %d\n", i)))
	}
	revs := app.listConfRevisions("example.com")
	if len(revs) != maxConfRevisions {
		t.Fatalf("expected history capped at %d, got %d", maxConfRevisions, len(revs))
	}
	// Newest revision (#1) must hold the last content written.
	rev, ok := app.revisionByIndex("example.com", 1)
	if !ok {
		t.Fatal("expected a newest revision")
	}
	data, _ := os.ReadFile(filepath.Join(app.confHistoryDir("example.com"), rev))
	want := fmt.Sprintf("# rev %d", maxConfRevisions+4)
	if !contains(string(data), want) {
		t.Fatalf("newest revision should contain %q, got %q", want, data)
	}
}

func TestConfRevisionDedupesIdentical(t *testing.T) {
	app, _, _ := newTestApp(t)
	same := []byte("# hostQ managed - example.com\nserver{}\n")
	app.saveConfRevision("example.com", same)
	app.saveConfRevision("example.com", same)
	app.saveConfRevision("example.com", same)
	if n := len(app.listConfRevisions("example.com")); n != 1 {
		t.Fatalf("identical saves should dedupe to 1 revision, got %d", n)
	}
}

func TestBackupAndRestoreRoundTrip(t *testing.T) {
	app, sitesDir, _ := newTestApp(t)
	good := "# hostQ managed - example.com\nserver { listen 80; }\n"
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	if saved := app.runBackup(); len(saved) != 1 || saved[0] != "example.com" {
		t.Fatalf("backup should capture example.com, got %v", saved)
	}
	// Outage: truncate the live config, then restore from backup.
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	res := app.runRestore("example.com")
	if len(res.Restored) != 1 {
		t.Fatalf("expected example.com restored, got %+v", res)
	}
	got, _ := os.ReadFile(filepath.Join(sitesDir, "example.com"))
	if string(got) != good {
		t.Fatalf("restore mismatch: got %q want %q", got, good)
	}
}

func TestRestoreSpecificRevision(t *testing.T) {
	app, sitesDir, _ := newTestApp(t)
	v1 := "# hostQ managed - example.com\n# version one\n"
	v2 := "# hostQ managed - example.com\n# version two\n"
	app.saveGoodConf("example.com", []byte(v1))
	app.saveGoodConf("example.com", []byte(v2))
	// #2 is the older revision (v1). Restore it.
	rev, ok := app.revisionByIndex("example.com", 2)
	if !ok {
		t.Fatal("expected revision #2 to exist")
	}
	if !app.restoreRevision("example.com", rev) {
		t.Fatal("restoreRevision failed")
	}
	got, _ := os.ReadFile(filepath.Join(sitesDir, "example.com"))
	if string(got) != v1 {
		t.Fatalf("expected older revision v1 restored, got %q", got)
	}
}

func TestDeployLogAppendAndTail(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.deployLog("example.com", "apply", "ok", "")
	app.deployLog("example.com", "apply", "validate-failed", "bad directive")
	app.deployLog("other.com", "restore", "ok", "latest")
	lines := app.tailDeployLog(2)
	if len(lines) != 2 {
		t.Fatalf("expected 2 tailed lines, got %d", len(lines))
	}
	var last DeployLogEntry
	if err := json.Unmarshal([]byte(lines[1]), &last); err != nil {
		t.Fatalf("journal line must be valid JSON: %v", err)
	}
	if last.Domain != "other.com" || last.Action != "restore" {
		t.Fatalf("unexpected last entry: %+v", last)
	}
}

func TestListSitesSkipsSidecars(t *testing.T) {
	app, sitesDir, _ := newTestApp(t)
	managed := "# hostQ managed - example.com\n# hostQ php: 8.4\nserver {\n  root /var/www/example.com/htdocs;\n}\n"
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com"), []byte(managed), 0644); err != nil {
		t.Fatal(err)
	}
	// A .prev rollback copy also contains "hostQ managed" — it must NOT appear
	// as a second site.
	if err := os.WriteFile(filepath.Join(sitesDir, "example.com.prev"), []byte(managed), 0644); err != nil {
		t.Fatal(err)
	}
	sites := app.listSites()
	if len(sites) != 1 || sites[0].Domain != "example.com" {
		t.Fatalf("expected exactly one site (sidecar skipped), got %+v", sites)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
