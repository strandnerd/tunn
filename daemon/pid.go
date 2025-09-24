package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadPID loads the stored daemon PID if present.
func ReadPID(paths Paths) (int, error) {
	data, err := os.ReadFile(paths.PIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read pid file: %w", err)
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return 0, nil
	}

	pid, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("invalid pid file contents: %w", err)
	}
	return pid, nil
}

// WritePID persists the daemon PID with restrictive permissions.
func WritePID(paths Paths, pid int) error {
	content := []byte(strconv.Itoa(pid) + "\n")
	if err := os.WriteFile(paths.PIDFile, content, 0o600); err != nil {
		return fmt.Errorf("failed to write pid file: %w", err)
	}
	return nil
}

// RemovePID clears the stored PID file if present.
func RemovePID(paths Paths) error {
	if err := os.Remove(paths.PIDFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pid file: %w", err)
	}
	return nil
}

// RemoveSocket unlinks the daemon's control socket.
func RemoveSocket(paths Paths) error {
	if err := os.Remove(paths.SocketFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove socket: %w", err)
	}
	return nil
}

// Cleanup removes daemon runtime files.
func Cleanup(paths Paths) {
	_ = RemovePID(paths)
	_ = RemoveSocket(paths)
}

// CheckRunning inspects the pid file and returns the running state.
func CheckRunning(paths Paths) (pid int, running bool, err error) {
	pid, err = ReadPID(paths)
	if err != nil {
		return 0, false, err
	}
	if pid == 0 {
		return 0, false, nil
	}
	if isProcessRunning(pid) {
		return pid, true, nil
	}

	Cleanup(paths)
	return 0, false, nil
}
