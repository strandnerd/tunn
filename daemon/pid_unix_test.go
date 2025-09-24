//go:build unix

package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPIDLifecycle(t *testing.T) {
	dir := t.TempDir()
	paths := Paths{
		RuntimeDir: dir,
		PIDFile:    filepath.Join(dir, "daemon.pid"),
		SocketFile: filepath.Join(dir, "daemon.sock"),
	}

	if err := WritePID(paths, os.Getpid()); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}

	pid, err := ReadPID(paths)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("expected pid %d, got %d", os.Getpid(), pid)
	}

	pid, running, err := CheckRunning(paths)
	if err != nil {
		t.Fatalf("CheckRunning failed: %v", err)
	}
	if !running {
		t.Fatalf("expected current process to be considered running")
	}
	if pid != os.Getpid() {
		t.Fatalf("expected pid %d, got %d", os.Getpid(), pid)
	}

	// Write a clearly invalid PID.
	if err := WritePID(paths, 999999); err != nil {
		t.Fatalf("WritePID failed: %v", err)
	}
	if err := os.WriteFile(paths.SocketFile, []byte("stub"), 0o600); err != nil {
		t.Fatalf("failed to write socket stub: %v", err)
	}

	pid, running, err = CheckRunning(paths)
	if err != nil {
		t.Fatalf("CheckRunning failed: %v", err)
	}
	if running {
		t.Fatalf("expected stale pid to be treated as not running")
	}
	if pid != 0 {
		t.Fatalf("expected pid to be cleared, got %d", pid)
	}

	if _, err := os.Stat(paths.PIDFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file to be removed, got err %v", err)
	}
	if _, err := os.Stat(paths.SocketFile); !os.IsNotExist(err) {
		t.Fatalf("expected socket file to be removed, got err %v", err)
	}
}
