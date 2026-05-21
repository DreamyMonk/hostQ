package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ScanFinding is a single suspicious result in a site scan.
type ScanFinding struct {
	Path     string `json:"path"`     // path relative to site docroot
	AbsPath  string `json:"abs"`      // absolute path on disk
	Rule     string `json:"rule"`     // short rule name
	Detail   string `json:"detail"`   // human description
	Severity string `json:"severity"` // critical | high | medium | low
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Match    string `json:"match"` // small snippet from the file (max 120 chars)
}

// ScanReport is what we persist on disk and render on the Security tab.
type ScanReport struct {
	Domain   string        `json:"domain"`
	Root     string        `json:"root"`
	When     string        `json:"when"`
	Scanned  int           `json:"scanned"`
	Findings []ScanFinding `json:"findings"`
	Took     string        `json:"took"`
	Critical int           `json:"critical"`
	High     int           `json:"high"`
	Medium   int           `json:"medium"`
	Low      int           `json:"low"`
}

type scanRule struct {
	name     string
	severity string
	detail   string
	re       *regexp.Regexp
}

// rules are intentionally a curated, lightweight set. Goal is high-signal hits,
// not a full antivirus database. Mirrors what maldet/CXS commonly flag.
var scanRules = []scanRule{
	{"webshell.signature", "critical", "Known webshell name (c99/r57/WSO/FilesMan/b374k/alfa/indoxploit)",
		regexp.MustCompile(`(?i)\b(c99shell|r57shell|WSO\s*\d*\s*shell|FilesMan|b374k|alfa[-_]?(team|shell)|indoxploit|mini[-_]?shell)\b`)},
	{"eval.encoded", "critical", "eval() of decoded payload",
		regexp.MustCompile(`(?i)eval\s*\(\s*(base64_decode|gzinflate|gzuncompress|str_rot13|hex2bin)`)},
	{"input.callback", "critical", "User input invoked as a function (RCE pattern)",
		regexp.MustCompile(`\$_(GET|POST|REQUEST|COOKIE|SERVER)\s*\[[^\]]+\]\s*\(`)},
	{"assert.input", "critical", "assert() with user input (RCE)",
		regexp.MustCompile(`(?i)assert\s*\(\s*\$_(GET|POST|REQUEST|COOKIE)`)},
	{"preg.e_modifier", "high", "preg_replace with /e modifier (deprecated RCE vector)",
		regexp.MustCompile(`preg_replace\s*\([^)]*['"][^'"]*[a-zA-Z]*e[a-zA-Z]*['"]`)},
	{"create_function", "high", "create_function() — commonly used to hide eval",
		regexp.MustCompile(`(?i)\bcreate_function\s*\(`)},
	{"long.b64", "medium", "Long base64-looking string assigned to a variable (~200+ chars)",
		regexp.MustCompile(`['"][A-Za-z0-9+/]{200,}={0,2}['"]`)},
	{"shell.exec", "medium", "Shell command execution (exec/system/shell_exec/passthru/popen/proc_open)",
		regexp.MustCompile(`(?i)\b(exec|system|shell_exec|passthru|popen|proc_open)\s*\(`)},
	{"obfuscated.chr", "medium", "Many chained chr() calls — common obfuscation",
		regexp.MustCompile(`(chr\(\d+\)\s*\.\s*){8,}`)},
	{"iframe.hidden", "medium", "Hidden iframe (frequent SEO/redirect malware payload)",
		regexp.MustCompile(`(?i)<iframe[^>]*(display\s*:\s*none|width\s*=\s*["']?0|height\s*=\s*["']?0)`)},
	{"htaccess.handler", "medium", ".htaccess re-maps a non-PHP extension to PHP handler",
		regexp.MustCompile(`(?i)AddHandler\s+application/x-httpd-php\s+\.(jpg|png|gif|txt|ico|css|js)`)},
	{"php.in.uploads", "high", "PHP file living under an uploads/cache directory",
		nil /* matched by path, not content */},
}

// extensions that get content-scanned
var scanContentExts = map[string]bool{
	".php": true, ".phtml": true, ".phar": true,
	".html": true, ".htm": true,
	".js":  true,
	".inc": true,
}

// excluded directory bases (we still scan WP plugin/theme code; only skip large noise)
var scanSkipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	".svn":         true,
	"cache":        false, // intentional — cache directories are a favorite hiding spot
}

const scanMaxBytes int64 = 5 * 1024 * 1024 // skip files larger than 5MB

func (a *App) runSiteScan(site Site) ScanReport {
	start := time.Now()
	report := ScanReport{
		Domain: site.Domain,
		Root:   site.Root,
		When:   start.Format(time.RFC3339),
	}
	if site.Root == "" {
		return report
	}
	_ = filepath.Walk(site.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if scanSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		// .htaccess has no extension on the dot side; check name
		name := strings.ToLower(info.Name())
		if name == ".htaccess" {
			ext = ".htaccess"
		}
		// Path-based rule: PHP file under uploads/cache/runtime is suspicious
		rel, _ := filepath.Rel(site.Root, path)
		rel = filepath.ToSlash(rel)
		lowerRel := strings.ToLower(rel)
		if ext == ".php" || ext == ".phtml" || ext == ".phar" {
			for _, marker := range []string{"/uploads/", "/wp-content/uploads/", "/cache/", "/tmp/", "/runtime/"} {
				if strings.Contains(lowerRel, marker) {
					report.Findings = append(report.Findings, ScanFinding{
						Path: rel, AbsPath: path, Rule: "php.in.uploads", Severity: "high",
						Detail: "PHP file located under " + strings.Trim(marker, "/") + " — usually a dropper",
						Size:   info.Size(), Modified: info.ModTime().Format("2006-01-02 15:04"),
					})
				}
			}
		}
		if ext != ".htaccess" && !scanContentExts[ext] {
			return nil
		}
		if info.Size() > scanMaxBytes {
			return nil
		}
		report.Scanned++
		data, err := readFirstBytes(path, 256*1024)
		if err != nil || len(data) == 0 {
			return nil
		}
		for _, rule := range scanRules {
			if rule.re == nil {
				continue
			}
			if loc := rule.re.FindIndex(data); loc != nil {
				report.Findings = append(report.Findings, ScanFinding{
					Path: rel, AbsPath: path, Rule: rule.name, Severity: rule.severity,
					Detail: rule.detail, Size: info.Size(),
					Modified: info.ModTime().Format("2006-01-02 15:04"),
					Match:    snippet(data, loc[0], loc[1]),
				})
			}
		}
		return nil
	})
	for _, f := range report.Findings {
		switch f.Severity {
		case "critical":
			report.Critical++
		case "high":
			report.High++
		case "medium":
			report.Medium++
		default:
			report.Low++
		}
	}
	// Sort: critical first, then path
	sevOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
	sort.Slice(report.Findings, func(i, j int) bool {
		if sevOrder[report.Findings[i].Severity] != sevOrder[report.Findings[j].Severity] {
			return sevOrder[report.Findings[i].Severity] < sevOrder[report.Findings[j].Severity]
		}
		return report.Findings[i].Path < report.Findings[j].Path
	})
	report.Took = time.Since(start).Round(time.Millisecond).String()
	_ = a.saveScanReport(report)
	return report
}

func readFirstBytes(path string, n int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := bytes.Buffer{}
	_, err = io.CopyN(&buf, f, n)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf.Bytes(), nil
}

func snippet(data []byte, start, end int) string {
	pad := 30
	a := start - pad
	if a < 0 {
		a = 0
	}
	b := end + pad
	if b > len(data) {
		b = len(data)
	}
	s := string(data[a:b])
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	if len(s) > 160 {
		s = s[:160] + "…"
	}
	return strings.TrimSpace(s)
}

func (a *App) scanReportPath(domain string) string {
	return filepath.Join(a.cfg.DataDir, "scans", domain+".json")
}

func (a *App) saveScanReport(report ScanReport) error {
	dir := filepath.Join(a.cfg.DataDir, "scans")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(report, "", "  ")
	return os.WriteFile(a.scanReportPath(report.Domain), data, 0600)
}

func (a *App) loadScanReport(domain string) (*ScanReport, error) {
	data, err := os.ReadFile(a.scanReportPath(domain))
	if err != nil {
		return nil, err
	}
	var r ScanReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (a *App) quarantinePath(domain string) string {
	return filepath.Join("/var/backups/hostq/quarantine", domain)
}

// quarantineFile moves a file into a per-domain quarantine directory, preserving
// the original relative path. Returns the new path.
func (a *App) quarantineFile(domain, abs string) (string, error) {
	site, ok := a.findSite(domain)
	if !ok {
		return "", fmt.Errorf("unknown domain")
	}
	if !a.canMutateWebPath(abs) {
		return "", fmt.Errorf("path is not mutable")
	}
	rel, err := filepath.Rel(site.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes site root")
	}
	stamp := time.Now().Format("20060102-150405")
	dst := filepath.Join(a.quarantinePath(domain), stamp, rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return "", err
	}
	if err := os.Rename(abs, dst); err != nil {
		// cross-device fallback: copy + remove
		if copyErr := copyFile(abs, dst); copyErr != nil {
			return "", copyErr
		}
		_ = os.Remove(abs)
	}
	return dst, nil
}
