//go:build linux

package main

import (
	"os"
	"path/filepath"
	"syscall"
)

// withConfigLock runs fn while holding an exclusive advisory lock on
// /etc/hostq/nginx-config.lock, so a CLI maintenance command (repair/rebuild/
// doctor) and, say, an SSL renewal can't regenerate Nginx concurrently. The
// lock is released automatically if the process dies, so a crash never leaves
// it stuck. If locking is unavailable we still run — atomic writes keep the
// worst case safe.
func (a *App) withConfigLock(fn func()) {
	_ = os.MkdirAll(a.cfg.DataDir, 0700)
	f, err := os.OpenFile(filepath.Join(a.cfg.DataDir, "nginx-config.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fn()
		return
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		fn()
		return
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	fn()
}
