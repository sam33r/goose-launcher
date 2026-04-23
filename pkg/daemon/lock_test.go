package daemon

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireLock_FirstWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	first, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	defer first.Release()

	second, err := AcquireLock(path)
	if err == nil {
		second.Release()
		t.Fatal("second AcquireLock succeeded; expected ErrAlreadyRunning")
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second AcquireLock err = %v, want ErrAlreadyRunning", err)
	}
}

func TestAcquireLock_ReleaseLetsAnotherIn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pid")

	first, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("first AcquireLock: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	second, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("second AcquireLock after release: %v", err)
	}
	second.Release()
}
