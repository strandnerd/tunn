package tunnel

import (
	"context"
	"testing"
	"time"

	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
)

func TestManagerRunTunnels(t *testing.T) {
	mock := &executor.MockSSHExecutor{}
	display := output.NewDisplay()
	manager := NewManager(mock, display)

	tunnels := map[string]config.Tunnel{
		"api": {
			Host:  "server1",
			Ports: []string{"3000:3000"},
		},
		"db": {
			Host:  "server2",
			Ports: []string{"5432:5432"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := manager.RunTunnels(ctx, tunnels)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}

	if len(mock.Commands) != 2 {
		t.Errorf("Expected 2 commands (one per tunnel), got %d", len(mock.Commands))
	}

	foundHosts := make(map[string]bool)
	for _, cmd := range mock.Commands {
		for _, arg := range cmd {
			if arg == "server1" || arg == "server2" {
				foundHosts[arg] = true
			}
		}
	}

	if len(foundHosts) != 2 {
		t.Error("Expected both server1 and server2 to be in commands")
	}
}

func TestManagerRunSingleTunnel(t *testing.T) {
	mock := &executor.MockSSHExecutor{}
	display := output.NewDisplay()
	manager := NewManager(mock, display)

	tunnels := map[string]config.Tunnel{
		"api": {
			Host:  "server1",
			Ports: []string{"3000:3000", "4000:4000"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := manager.RunTunnels(ctx, tunnels)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context deadline exceeded, got %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(mock.Commands))
	}
}

func TestManagerCancellation(t *testing.T) {
	mock := &executor.MockSSHExecutor{}
	display := output.NewDisplay()
	manager := NewManager(mock, display)

	tunnels := map[string]config.Tunnel{
		"api": {
			Host:  "server1",
			Ports: []string{"3000:3000"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := manager.RunTunnels(ctx, tunnels)
	if err != context.Canceled {
		t.Errorf("Expected context canceled, got %v", err)
	}
}