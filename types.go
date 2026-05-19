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
	Name string
	Kind string
	Path string
}

type DatabaseInfo struct {
	Name string
}

type CertInfo struct {
	Domain string
	Expiry string
	Days   int
	Status string
}

type WordPressInfo struct {
	Domain string
	Path   string
	Status string
}

type PHPInfo struct {
	Version string
	Service string
	Status  string
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
	cfg Config
	tpl *template.Template
}

var domainRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phpVersionRe = regexp.MustCompile(`^(8\.2|8\.3|8\.4|8\.5)$`)
