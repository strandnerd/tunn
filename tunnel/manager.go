package tunnel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
)

type Manager struct {
	executor executor.SSHExecutor
	display  *output.Display
	checker  portChecker
	notify   func(string, string, string)
}

func NewManager(exec executor.SSHExecutor, display *output.Display, notifier func(string, string, string)) *Manager {
	return &Manager{
		executor: exec,
		display:  display,
		checker:  newSystemPortChecker(),
		notify:   notifier,
	}
}

func (m *Manager) RunTunnels(ctx context.Context, tunnels map[string]config.Tunnel) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(tunnels))

	for name, tunnel := range tunnels {
		wg.Add(1)
		go func(n string, t config.Tunnel) {
			defer wg.Done()
			if err := m.runTunnel(ctx, n, t); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(name, tunnel)
	}

	// Wait for all tunnels to complete (they will block until context is cancelled)
	wg.Wait()
	select {
	case err := <-errCh:
		return err
	default:
	}
	return ctx.Err()
}

func (m *Manager) runTunnel(ctx context.Context, name string, tunnel config.Tunnel) error {
	if err := m.ensurePortsAvailable(name, tunnel); err != nil {
		return err
	}
	return m.executor.Execute(ctx, name, tunnel)
}

func (m *Manager) ensurePortsAvailable(tunnelName string, tunnel config.Tunnel) error {
	conflicts := make(map[string]string)
	var conflictMessages []string

	for _, mapping := range tunnel.Ports {
		localPort, err := extractLocalPort(mapping)
		if err != nil {
			return fmt.Errorf("invalid port mapping %q: %w", mapping, err)
		}

		process, err := m.checker.findListener(localPort)
		if err != nil {
			return err
		}
		if process != nil {
			message := fmt.Sprintf("port %s is being used by \"%s\" (pid: %d)", localPort, process.command, process.pid)
			conflicts[mapping] = message
			conflictMessages = append(conflictMessages, message)
		}
	}

	if len(conflicts) > 0 {
		for _, mapping := range tunnel.Ports {
			status := "stopped"
			if msg, ok := conflicts[mapping]; ok {
				status = fmt.Sprintf("error - %s", msg)
			}
			m.reportStatus(tunnelName, mapping, status)
		}
		return fmt.Errorf("%s", strings.Join(conflictMessages, "; "))
	}

	return nil
}

func (m *Manager) reportStatus(tunnelName, mapping, status string) {
	if m.display != nil {
		m.display.UpdateStatus(tunnelName, mapping, status)
	}
	if m.notify != nil {
		m.notify(tunnelName, mapping, status)
	}
}
