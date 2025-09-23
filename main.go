package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
	"github.com/strandnerd/tunn/tunnel"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	selectedTunnels := cfg.FilterTunnels(os.Args[1:])

	if len(selectedTunnels) == 0 {
		if len(os.Args) > 1 {
			return fmt.Errorf("no tunnels found matching: %v", os.Args[1:])
		}
		return fmt.Errorf("no tunnels defined in configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down tunnels...")
		cancel()
	}()

	display := output.NewDisplay()

	sshExec := &executor.RealSSHExecutor{
		OnStatusChange: display.UpdateStatus,
	}

	manager := tunnel.NewManager(sshExec, display)

	return manager.RunTunnels(ctx, selectedTunnels)
}
