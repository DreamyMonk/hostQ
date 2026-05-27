package main

import (
	"html/template"
	"regexp"
)

type Config struct {
	Addr          string
	DataDir       string
	WebRoot       string
	NginxSitesDir string
	JWTSecret     string
}

type Account struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Role         string `json:"role"`
}

type Site struct {
	Domain     string
	Root       string
	Enabled    bool
	SSL        bool
	Cache      bool
	PHPVersion string
}

type FileItem struct {
	Name    string
	Kind    string
	Path    string
	Size    string
	Mode    string
	ModTime string
}

type Crumb struct {
	Name string
	Path string
}

type SystemStats struct {
	LoadAvg     string
	Uptime      string
	MemUsed     string
	MemTotal    string
	MemPercent  int
	DiskUsed    string
	DiskTotal   string
	DiskPercent int
	Hostname    string
	CPUCount    int
}

type AuditEntry struct {
	Timestamp string
	Action    string
	Status    string
	Target    string
}

type DatabaseInfo struct {
	Name  string
	Users []DBUser
	Size  string
}

type DBUser struct {
	Login string
	Host  string
}

type CertInfo struct {
	Domain string
	Expiry string
	Days   int
	Status string
}

type WordPressInfo struct {
	Domain  string
	Path    string
	Status  string
	Version string
	SiteURL string
}

type WordPressUser struct {
	ID    string
	Login string
	Email string
	Roles string
}

type PHPInfo struct {
	Version string
	Service string
	Status  string
}

type PHPExtension struct {
	Name      string // module name as PHP sees it (e.g. "gd", "redis")
	Apt       string // apt package providing it (e.g. "php8.4-gd")
	Installed bool   // mods-available/<name>.ini present
	Enabled   bool   // symlink in fpm/conf.d/
	Loaded    bool   // actually reported by `php -m`
	Common    bool   // surfaced as installable even when not yet on disk
}

type Service struct {
	ID      string
	Name    string
	Systemd string
	Status  string
}

type BackupInfo struct {
	Name    string
	Domain  string
	Path    string
	Size    string
	Created string
}

type BackupPolicy struct {
	Domain    string `json:"domain"`
	Frequency string `json:"frequency"`
	Keep      int    `json:"keep"`
	Hour      int    `json:"hour"`
	MaxLoad   string `json:"maxLoad"`
	LastRun   string `json:"lastRun"`
}

type CronJob struct {
	ID       string
	Name     string
	Schedule string
	User     string
	Command  string
	Source   string
}

type App struct {
	cfg   Config
	tpl   *template.Template
	cache *memCache
}

type RedisStats struct {
	Active        bool
	Version       string
	UptimeDays    string
	UsedMemory    string
	PeakMemory    string
	Clients       string
	OpsPerSec     string
	TotalKeys     int
	HitRate       string
	EvictedKeys   string
	Service       string
}

var domainRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phpVersionRe = regexp.MustCompile(`^(8\.2|8\.3|8\.4|8\.5)$`)
