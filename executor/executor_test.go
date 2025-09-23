package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/strandnerd/tunn/config"
)

func TestMockSSHExecutor(t *testing.T) {
	mock := &MockSSHExecutor{}

	tunnel := config.Tunnel{
		Host:  "testserver",
		Ports: []string{"8080:8080", "9090:9091"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := mock.Execute(ctx, "test", tunnel)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	cmdStr := strings.Join(cmd, " ")

	if !strings.Contains(cmdStr, "ssh -N") {
		t.Error("Command should contain 'ssh -N'")
	}

	if !strings.Contains(cmdStr, "-L 8080:localhost:8080") {
		t.Error("Command should contain port mapping for 8080")
	}

	if !strings.Contains(cmdStr, "-L 9090:localhost:9091") {
		t.Error("Command should contain port mapping for 9090")
	}

	if !strings.Contains(cmdStr, "testserver") {
		t.Error("Command should contain the host")
	}
}

func TestMockSSHExecutorWithIdentityFile(t *testing.T) {
	mock := &MockSSHExecutor{}

	tunnel := config.Tunnel{
		Host:         "testserver",
		Ports:        []string{"8080:8080"},
		IdentityFile: "~/.ssh/custom_key",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mock.Execute(ctx, "test", tunnel)

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	cmdStr := strings.Join(cmd, " ")

	if !strings.Contains(cmdStr, "-i") {
		t.Error("Command should contain '-i' flag for identity file")
	}

	foundIdentityFile := false
	for i, arg := range cmd {
		if arg == "-i" && i+1 < len(cmd) {
			if strings.Contains(cmd[i+1], "custom_key") {
				foundIdentityFile = true
				break
			}
		}
	}

	if !foundIdentityFile {
		t.Error("Command should contain the identity file path")
	}
}

func TestMockSSHExecutorStatusCallbacks(t *testing.T) {
	statusChanges := []struct {
		tunnelName string
		port       string
		status     string
	}{}

	mock := &MockSSHExecutor{
		OnStatusChange: func(name, port, status string) {
			statusChanges = append(statusChanges, struct {
				tunnelName string
				port       string
				status     string
			}{name, port, status})
		},
	}

	tunnel := config.Tunnel{
		Host:  "testserver",
		Ports: []string{"8080:8080", "9090:9091"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mock.Execute(ctx, "test", tunnel)

	expectedChanges := 4
	if len(statusChanges) != expectedChanges {
		t.Errorf("Expected %d status changes, got %d", expectedChanges, len(statusChanges))
	}

	expectedStatuses := []string{"connecting", "active", "connecting", "active"}
	for i, expected := range expectedStatuses {
		if i < len(statusChanges) && statusChanges[i].status != expected {
			t.Errorf("Status change %d: expected %s, got %s", i, expected, statusChanges[i].status)
		}
	}
}

func TestExpandPort(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"8080:8081", []string{"8080", "8081"}},
		{"3000", []string{"3000", "3000"}},
		{"5432:5433", []string{"5432", "5433"}},
	}

	for _, tt := range tests {
		result := expandPort(tt.input, ":")
		if len(result) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(result))
			continue
		}
		if result[0] != tt.expected[0] || result[1] != tt.expected[1] {
			t.Errorf("For %s: expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}