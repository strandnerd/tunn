package daemon

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths represents the filesystem anchor for daemon metadata.
type Paths struct {
	RuntimeDir string
	PIDFile    string
	SocketFile string
	LogFile    string
}

// ResolvePaths determines the directory for daemon runtime artifacts and ensures it exists.
func ResolvePaths() (Paths, error) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		runtimeDir = filepath.Join(runtimeDir, "tunn")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("failed to determine home directory: %w", err)
		}
		runtimeDir = filepath.Join(home, ".cache", "tunn")
	}

	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return Paths{}, fmt.Errorf("failed to create runtime directory: %w", err)
	}
	if err := os.Chmod(runtimeDir, 0o700); err != nil && !os.IsPermission(err) {
		return Paths{}, fmt.Errorf("failed to enforce runtime directory permissions: %w", err)
	}

	return Paths{
		RuntimeDir: runtimeDir,
		PIDFile:    filepath.Join(runtimeDir, "daemon.pid"),
		SocketFile: filepath.Join(runtimeDir, "daemon.sock"),
		LogFile:    filepath.Join(runtimeDir, "daemon.log"),
	}, nil
}
