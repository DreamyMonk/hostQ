package main

import (
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller")
	}
	return filepath.Dir(file)
}

func TestLayoutTemplateParses(t *testing.T) {
	if _, err := template.New("hostq-test").Funcs(template.FuncMap{"now": func() any { return nil }}).Parse(layoutTemplate); err != nil {
		t.Fatalf("layout template must parse: %v", err)
	}
}

func readRepoFile(t *testing.T, parts ...string) string {
	data, err := os.ReadFile(filepath.Join(append([]string{repoRoot(t)}, parts...)...))
	if err != nil {
		t.Fatalf("read %v: %v", parts, err)
	}
	return string(data)
}

func TestDeploymentScripts(t *testing.T) {
	setup := readRepoFile(t, "install.sh")
	update := readRepoFile(t, "scripts", "hostq-update.sh")

	for name, content := range map[string]string{"install.sh": setup, "hostq-update.sh": update} {
		if !strings.Contains(content, "go build") {
			t.Fatalf("%s must build the panel", name)
		}
	}
	if !strings.Contains(setup, "hostq-panel.service") {
		t.Fatal("install script must install hostq-panel.service")
	}
	if !strings.Contains(update, "systemctl restart hostq-panel") {
		t.Fatal("hostq-update must restart the panel service")
	}
}

func TestPanelIncludesCoreHostingModules(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	source := b.String()
	required := []string{
		"func (a *App) wordpress",
		"func (a *App) php",
		"func (a *App) ssl",
		"CREATE DATABASE IF NOT EXISTS",
		"DROP DATABASE IF EXISTS",
		"removeBrokenNginxSSL",
		"blockedFileName",
		"hostq_session",
		"bcrypt.CompareHashAndPassword",
	}
	for _, needle := range required {
		if !strings.Contains(source, needle) {
			t.Fatalf("panel missing %q", needle)
		}
	}
}
