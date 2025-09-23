package tunnel

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type portChecker interface {
	findListener(port string) (*processInfo, error)
}

type processInfo struct {
	command string
	pid     int
}

func newSystemPortChecker() portChecker {
	return &systemPortChecker{}
}

type systemPortChecker struct{}

func (c *systemPortChecker) findListener(port string) (*processInfo, error) {
	if port == "" {
		return nil, nil
	}

	cmd := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			if exitErr.ExitCode() == 1 {
				return nil, nil
			}
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr == "" {
				stderr = exitErr.Error()
			}
			return nil, fmt.Errorf("failed to inspect port %s: %s", port, stderr)
		case errors.Is(err, exec.ErrNotFound):
			return nil, fmt.Errorf("required command lsof not found while checking port %s", port)
		default:
			return nil, fmt.Errorf("failed to inspect port %s: %w", port, err)
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return nil, nil
	}

	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		command := fields[0]
		if command == "" {
			command = "unknown"
		}

		return &processInfo{command: command, pid: pid}, nil
	}

	return nil, nil
}

func extractLocalPort(mapping string) (string, error) {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return "", fmt.Errorf("empty port mapping")
	}

	parts := strings.Split(mapping, ":")
	if len(parts) == 0 {
		return "", fmt.Errorf("unable to determine local port for mapping %q", mapping)
	}

	candidate := strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		if _, err := strconv.Atoi(candidate); err != nil {
			candidate = strings.TrimSpace(parts[1])
		}
	}

	if candidate == "" {
		return "", fmt.Errorf("unable to determine local port for mapping %q", mapping)
	}

	if _, err := strconv.Atoi(candidate); err != nil {
		return "", fmt.Errorf("invalid local port %q", candidate)
	}

	return candidate, nil
}
