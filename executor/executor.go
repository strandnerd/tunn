package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/strandnerd/tunn/config"
)

type SSHExecutor interface {
	Execute(ctx context.Context, name string, tunnel config.Tunnel) error
}

type RealSSHExecutor struct {
	OnStatusChange func(tunnelName string, port string, status string)
}

func (e *RealSSHExecutor) Execute(ctx context.Context, name string, tunnel config.Tunnel) error {
	var wg sync.WaitGroup

	// Update all ports to connecting status synchronously
	for _, portMapping := range tunnel.Ports {
		if e.OnStatusChange != nil {
			e.OnStatusChange(name, portMapping, "connecting")
		}
	}

	// Start SSH processes for each port
	for _, portMapping := range tunnel.Ports {
		wg.Add(1)
		go func(port string) {
			defer wg.Done()
			e.executePortSSH(ctx, name, tunnel, port)
		}(portMapping)
	}

	// Wait for context cancellation (tunnels run until cancelled)
	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (e *RealSSHExecutor) executePortSSH(ctx context.Context, tunnelName string, tunnel config.Tunnel, portMapping string) error {
	// Build SSH command for this specific port
	args := []string{"-N"}

	ports := expandPort(portMapping, ":")
	local, remote := ports[0], ports[1]
	args = append(args, "-L", fmt.Sprintf("%s:localhost:%s", local, remote))

	if tunnel.IdentityFile != "" {
		args = append(args, "-i", os.ExpandEnv(tunnel.IdentityFile))
	}

	if tunnel.User != "" {
		args = append(args, "-l", tunnel.User)
	}

	args = append(args, tunnel.Host)

	cmd := exec.Command("ssh", args...)

	// Start the SSH command
	if err := cmd.Start(); err != nil {
		if e.OnStatusChange != nil {
			e.OnStatusChange(tunnelName, portMapping, fmt.Sprintf("error - %s", err.Error()))
		}
		return err
	}

	activeTimer := time.NewTimer(500 * time.Millisecond)
	activeC := activeTimer.C

	stopActiveTimer := func() {
		if activeC == nil {
			return
		}
		if !activeTimer.Stop() {
			select {
			case <-activeC:
			default:
			}
		}
		activeC = nil
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	for {
		select {
		case <-activeC:
			if e.OnStatusChange != nil {
				e.OnStatusChange(tunnelName, portMapping, "active")
			}
			activeC = nil
		case err := <-done:
			stopActiveTimer()
			if err != nil {
				if e.OnStatusChange != nil {
					e.OnStatusChange(tunnelName, portMapping, fmt.Sprintf("error - %s", err.Error()))
				}
				return err
			}
			if e.OnStatusChange != nil {
				e.OnStatusChange(tunnelName, portMapping, "stopped")
			}
			return nil
		case <-ctx.Done():
			stopActiveTimer()
			if e.OnStatusChange != nil {
				e.OnStatusChange(tunnelName, portMapping, "stopping")
			}
			if cmd.Process != nil {
				_ = cmd.Process.Signal(os.Interrupt)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				<-done
			}
			if e.OnStatusChange != nil {
				e.OnStatusChange(tunnelName, portMapping, "stopped")
			}
			return ctx.Err()
		}
	}
}

func expandPort(mapping string, sep string) []string {
	parts := strings.Split(mapping, sep)
	if len(parts) == 1 {
		return []string{parts[0], parts[0]}
	}
	return parts[:2]
}

type MockSSHExecutor struct {
	Commands       [][]string
	OnStatusChange func(tunnelName string, port string, status string)
}

func (m *MockSSHExecutor) Execute(ctx context.Context, name string, tunnel config.Tunnel) error {
	args := []string{"ssh", "-N"}
	for _, portMapping := range tunnel.Ports {
		ports := expandPort(portMapping, ":")
		local, remote := ports[0], ports[1]
		args = append(args, "-L", fmt.Sprintf("%s:localhost:%s", local, remote))
	}

	if tunnel.IdentityFile != "" {
		args = append(args, "-i", os.ExpandEnv(tunnel.IdentityFile))
	}

	if tunnel.User != "" {
		args = append(args, "-l", tunnel.User)
	}

	args = append(args, tunnel.Host)
	m.Commands = append(m.Commands, args)

	if m.OnStatusChange != nil {
		for _, portMapping := range tunnel.Ports {
			m.OnStatusChange(name, portMapping, "connecting")
			m.OnStatusChange(name, portMapping, "active")
		}
	}

	<-ctx.Done()
	return ctx.Err()
}
