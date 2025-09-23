package tunnel

import (
	"context"
	"sync"

	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
)

type Manager struct {
	executor executor.SSHExecutor
	display  *output.Display
}

func NewManager(exec executor.SSHExecutor, display *output.Display) *Manager {
	return &Manager{
		executor: exec,
		display:  display,
	}
}

func (m *Manager) RunTunnels(ctx context.Context, tunnels map[string]config.Tunnel) error {
	var wg sync.WaitGroup

	for name, tunnel := range tunnels {
		wg.Add(1)
		go func(n string, t config.Tunnel) {
			defer wg.Done()
			if err := m.runTunnel(ctx, n, t); err != nil {
				if ctx.Err() == nil {
					m.display.PrintError(n, err.Error())
				}
			}
		}(name, tunnel)
	}

	// Wait for all tunnels to complete (they will block until context is cancelled)
	wg.Wait()
	return nil
}

func (m *Manager) runTunnel(ctx context.Context, name string, tunnel config.Tunnel) error {
	return m.executor.Execute(ctx, name, tunnel)
}
