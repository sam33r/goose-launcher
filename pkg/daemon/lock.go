package daemon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ErrAlreadyRunning is returned by AcquireLock when another daemon already
// holds the lock.
var ErrAlreadyRunning = errors.New("daemon: another instance is already running")

// LockFile is an exclusive file lock that auto-releases when the process
// exits (kernel-managed advisory lock — no stale-PID concerns even after
// SIGKILL). Hold it for the lifetime of the daemon.
type LockFile struct {
	f *os.File
}

// AcquireLock takes an exclusive flock on the given path. Creates parent
// dirs if needed. Writes the daemon's pid to the file (informational; the
// flock is what enforces single-instance, not the pid).
//
// Returns ErrAlreadyRunning if another process holds the lock.
func AcquireLock(path string) (*LockFile, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("mkdir lockfile dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lockfile %s: %w", path, err)
	}

	// LOCK_EX (exclusive) + LOCK_NB (don't block — fail fast if held).
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("flock %s: %w", path, err)
	}

	// Truncate + write our pid for human-readability. Failing here is
	// non-fatal — the lock is what matters.
	_ = f.Truncate(0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())

	return &LockFile{f: f}, nil
}

// Release closes the file (which releases the kernel lock).
func (l *LockFile) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	return err
}

// DefaultLockPath is the canonical pidfile location.
func DefaultLockPath() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "goose-launcher.pid")
	}
	return filepath.Join(cache, "goose-launcher.pid")
}
