package tunnel

import (
	"context"
	"fmt"
	"sync"

	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
)

type Manager struct {
	executor executor.SSHExecutor
	display  *output.Display
	checker  portChecker
}

func NewManager(exec executor.SSHExecutor, display *output.Display) *Manager {
	return &Manager{
		executor: exec,
		display:  display,
		checker:  newSystemPortChecker(),
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
				if ctx.Err() == nil && m.display != nil {
					m.display.PrintError(n, err.Error())
				}
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
			if m.display != nil {
				m.display.UpdateStatus(tunnelName, mapping, fmt.Sprintf("error - %s", message))
			}
			return fmt.Errorf("%s", message)
		}
	}
	return nil
}
