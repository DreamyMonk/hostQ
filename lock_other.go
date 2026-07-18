//go:build !linux

package main

// withConfigLock is a no-op on non-Linux platforms (development builds). The
// panel runs on Linux in production, where lock_linux.go provides the real
// flock-based implementation.
func (a *App) withConfigLock(fn func()) { fn() }
