package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/strandnerd/tunn/config"
)

func TestMockSSHExecutorWithUser(t *testing.T) {
	mock := &MockSSHExecutor{}

	tunnel := config.Tunnel{
		Host:  "testserver",
		Ports: []string{"8080:8080"},
		User:  "testuser",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mock.Execute(ctx, "test", tunnel)

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	cmdStr := strings.Join(cmd, " ")

	if !strings.Contains(cmdStr, "-l testuser") {
		t.Error("Command should contain '-l testuser' for user specification")
	}

	foundUser := false
	for i, arg := range cmd {
		if arg == "-l" && i+1 < len(cmd) {
			if cmd[i+1] == "testuser" {
				foundUser = true
				break
			}
		}
	}

	if !foundUser {
		t.Error("Command should contain the user parameter")
	}
}

func TestMockSSHExecutorWithUserAndIdentityFile(t *testing.T) {
	mock := &MockSSHExecutor{}

	tunnel := config.Tunnel{
		Host:         "testserver",
		Ports:        []string{"8080:8080"},
		User:         "deployuser",
		IdentityFile: "~/.ssh/deploy_key",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mock.Execute(ctx, "test", tunnel)

	if len(mock.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(mock.Commands))
	}

	cmd := mock.Commands[0]
	cmdStr := strings.Join(cmd, " ")

	if !strings.Contains(cmdStr, "-l deployuser") {
		t.Error("Command should contain '-l deployuser'")
	}

	if !strings.Contains(cmdStr, "-i") {
		t.Error("Command should contain '-i' flag for identity file")
	}

	foundUser := false
	foundIdentity := false

	for i, arg := range cmd {
		if arg == "-l" && i+1 < len(cmd) {
			if cmd[i+1] == "deployuser" {
				foundUser = true
			}
		}
		if arg == "-i" && i+1 < len(cmd) {
			if strings.Contains(cmd[i+1], "deploy_key") {
				foundIdentity = true
			}
		}
	}

	if !foundUser {
		t.Error("Command should contain the user parameter")
	}

	if !foundIdentity {
		t.Error("Command should contain the identity file parameter")
	}
}