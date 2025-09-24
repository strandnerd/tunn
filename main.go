package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/strandnerd/tunn/cli"
	"github.com/strandnerd/tunn/config"
	"github.com/strandnerd/tunn/daemon"
	"github.com/strandnerd/tunn/executor"
	"github.com/strandnerd/tunn/output"
	"github.com/strandnerd/tunn/status"
	"github.com/strandnerd/tunn/tunnel"
)

const daemonPreviewDuration = 2 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	opts, err := cli.Parse(args)
	if err != nil {
		return err
	}

	paths, err := daemon.ResolvePaths()
	if err != nil {
		return err
	}

	switch opts.Command {
	case cli.CommandStatus:
		return runStatusCommand(paths)
	case cli.CommandStop:
		return runStopCommand(paths)
	case cli.CommandStart:
		if opts.InternalDaemon {
			return runDaemonCommand(paths, opts.TunnelNames)
		}
		return runStartCommand(paths, opts)
	default:
		return fmt.Errorf("unknown command")
	}
}

func runStartCommand(paths daemon.Paths, opts *cli.Options) error {
	pid, running, err := daemon.CheckRunning(paths)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("tunn daemon already running (pid %d); use 'tunn status' to inspect or stop it before launching in the foreground", pid)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	selected := cfg.FilterTunnels(opts.TunnelNames)
	if len(selected) == 0 {
		if len(opts.TunnelNames) > 0 {
			return fmt.Errorf("no tunnels found matching: %v", opts.TunnelNames)
		}
		return fmt.Errorf("no tunnels defined in configuration")
	}

	if opts.Detach {
		return launchDaemon(paths, opts.TunnelNames)
	}

	if err := runForeground(selected); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("Exiting...")
			return nil
		}
		return err
	}
	return nil
}

func launchDaemon(paths daemon.Paths, tunnelNames []string) error {
	pid, running, err := daemon.CheckRunning(paths)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("tunn daemon already running (pid %d); use 'tunn status' or stop it before relaunching", pid)
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate executable: %w", err)
	}

	args := []string{"--internal-daemon"}
	args = append(args, tunnelNames...)

	cmd := exec.Command(executable, args...)
	cmd.Env = os.Environ()

	logFile, err := os.OpenFile(paths.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open daemon log: %w", err)
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	childPID := cmd.Process.Pid
	if err := daemon.WritePID(paths, childPID); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return err
	}

	if err := daemon.WaitForSocket(paths, 3*time.Second); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		daemon.Cleanup(paths)
		return fmt.Errorf("daemon failed to expose control socket: %w", err)
	}

	statusCtx, cancelStatus := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelStatus()
	resp, statusErr := daemon.QueryStatus(statusCtx, paths)
	if statusErr != nil || resp == nil || !resp.Running {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		daemon.Cleanup(paths)
		if statusErr != nil {
			msg, logErr := tailLogMessage(paths.LogFile)
			if logErr == nil && msg != "" {
				return fmt.Errorf("daemon failed to start: %s", msg)
			}
			return fmt.Errorf("daemon failed to start: %w", statusErr)
		}
		return fmt.Errorf("daemon failed to start")
	}

	if err := cmd.Process.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to release daemon process: %v\n", err)
	}

	encounteredErrors, previewErr := monitorDaemonStartup(paths, daemonPreviewDuration, isTerminal(os.Stdout))
	if previewErr != nil {
		fmt.Fprintf(os.Stderr, "warning: unable to preview daemon startup: %v\n", previewErr)
	}

	fmt.Printf("tunn daemon started (pid %d)\n", childPID)
	if encounteredErrors {
		fmt.Println("Some tunnels reported errors during startup. Run 'tunn status' for details.")
	}
	return nil
}

func runForeground(tunnels map[string]config.Tunnel) error {
	ctx, cancel := context.WithCancel(context.Background())
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(cancel)
	}
	defer shutdown()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)
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

	return manager.RunTunnels(ctx, tunnels)
}

func runDaemonCommand(paths daemon.Paths, tunnelNames []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	selected := cfg.FilterTunnels(tunnelNames)
	if len(selected) == 0 {
		if len(tunnelNames) > 0 {
			return fmt.Errorf("no tunnels found matching: %v", tunnelNames)
		}
		return fmt.Errorf("no tunnels defined in configuration")
	}

	logFile, err := os.OpenFile(paths.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open daemon log: %w", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("tunn daemon starting (pid %d)", os.Getpid())

	store := status.NewStore()
	for name, tun := range selected {
		store.EnsureTunnel(name, tun.Ports)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(cancel)
	}
	defer shutdown()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		sig := <-sigChan
		logger.Printf("received signal %v, initiating shutdown", sig)
		shutdown()
	}()

	server := daemon.NewServer(paths, store, os.Getpid(), shutdown)
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Run(ctx)
	}()

	sshExec := &executor.RealSSHExecutor{
		OnStatusChange: store.Update,
	}

	manager := tunnel.NewManager(sshExec, nil)
	managerErrCh := make(chan error, 1)
	go func() {
		managerErrCh <- manager.RunTunnels(ctx, selected)
	}()

	var managerErr error
	var srvErr error
	managerDone := false
	serverDone := false
	cancelled := false
	serverClosed := false

	for !managerDone || !serverDone {
		select {
		case err := <-managerErrCh:
			managerErr = err
			managerDone = true
		case err := <-serverErrCh:
			srvErr = err
			serverDone = true
		}
		if !cancelled {
			shutdown()
			cancelled = true
		}
		if !serverClosed {
			server.Close()
			serverClosed = true
		}
	}

	daemon.Cleanup(paths)

	if srvErr != nil && !errors.Is(srvErr, context.Canceled) {
		logger.Printf("ipc server error: %v", srvErr)
	}

	if managerErr != nil && !errors.Is(managerErr, context.Canceled) {
		logger.Printf("tunnel manager exited with error: %v", managerErr)
		return managerErr
	}

	logger.Printf("tunn daemon stopped")
	return nil
}

func runStatusCommand(paths daemon.Paths) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := daemon.QueryStatus(ctx, paths)
	if err != nil {
		pid, running, checkErr := daemon.CheckRunning(paths)
		if checkErr != nil {
			return fmt.Errorf("failed to check daemon status: %w", checkErr)
		}
		if running {
			return fmt.Errorf("daemon (pid %d) is unreachable: %v", pid, err)
		}
		fmt.Println("tunn daemon not running")
		return nil
	}

	state := "running"
	if !resp.Running {
		state = "stopped"
	}

	if !isTerminal(os.Stdout) {
		fmt.Printf("Daemon: %s (pid %d, mode %s)\n", state, resp.PID, resp.Mode)
		if len(resp.Tunnels) == 0 {
			fmt.Println("No tunnels managed by daemon")
			return nil
		}

		sort.Slice(resp.Tunnels, func(i, j int) bool {
			return resp.Tunnels[i].Name < resp.Tunnels[j].Name
		})

		fmt.Println("Tunnels:")
		for _, tun := range resp.Tunnels {
			fmt.Printf("  %s\n", tun.Name)
			ports := make([]string, 0, len(tun.Ports))
			for port := range tun.Ports {
				ports = append(ports, port)
			}
			sort.Strings(ports)
			for _, port := range ports {
				fmt.Printf("    %s - %s\n", port, tun.Ports[port])
			}
		}
		return nil
	}

	display := output.NewDisplay()
	cache := make(map[string]string)
	hasErrors := applySnapshotToDisplay(display, resp, cache)
	summary := fmt.Sprintf("Daemon: %s (pid %d, mode %s)", state, resp.PID, resp.Mode)
	if len(resp.Tunnels) == 0 {
		summary += " — no tunnels managed"
	}
	if hasErrors {
		summary += " — errors detected"
	}
	display.SetFooter(summary)
	return nil
}

func runStopCommand(paths daemon.Paths) error {
	pid, running, err := daemon.CheckRunning(paths)
	if err != nil {
		return err
	}
	if !running {
		fmt.Println("tunn daemon not running")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := daemon.SendStop(ctx, paths)
	if err != nil {
		return fmt.Errorf("failed to send stop command: %w", err)
	}

	statusLine := fmt.Sprintf("stopping... (pid %d)", pid)
	if resp.Message != "" && resp.Message != "stopping" {
		statusLine = fmt.Sprintf("%s (%s)", statusLine, resp.Message)
	}
	fmt.Println(statusLine)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		_, stillRunning, checkErr := daemon.CheckRunning(paths)
		if checkErr != nil {
			return checkErr
		}
		if !stillRunning {
			fmt.Println("tunn daemon stopped")
			return nil
		}
	}

	fmt.Println("tunn daemon stopping — run 'tunn status' to verify if needed")
	return nil
}

func monitorDaemonStartup(paths daemon.Paths, duration time.Duration, render bool) (bool, error) {
	if duration <= 0 {
		return false, nil
	}

	if render {
		return renderDaemonPreview(paths, duration)
	}

	deadline := time.Now().Add(duration)
	interval := 250 * time.Millisecond
	for {
		resp, err := queryDaemonStatus(paths)
		if err != nil {
			return false, err
		}
		if statusHasError(resp) {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(interval)
	}
}

func renderDaemonPreview(paths daemon.Paths, duration time.Duration) (bool, error) {
	display := output.NewDisplay()
	cache := make(map[string]string)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	defer fmt.Print("\033[H\033[2J")

	autoCloseAt := time.Now().Add(duration)
	encounteredError := false
	var exitCh <-chan struct{}
	var errorDeadline time.Time

	for {
		resp, err := queryDaemonStatus(paths)
		if err != nil {
			return encounteredError, err
		}

		hasErr := applySnapshotToDisplay(display, resp, cache)
		if hasErr {
			if !encounteredError {
				encounteredError = true
				display.SetFooter("Errors detected while starting tunnels. Press Enter to exit preview.")
				if isTerminal(os.Stdin) {
					exitCh = waitForEnterAsync()
					errorDeadline = time.Time{}
				} else {
					errorDeadline = time.Now().Add(duration)
				}
			}
		}

		if encounteredError {
			if exitCh != nil {
				select {
				case <-exitCh:
					display.SetFooter("")
					return true, nil
				case <-ticker.C:
					continue
				}
			}
			if !errorDeadline.IsZero() && time.Now().After(errorDeadline) {
				display.SetFooter("")
				return true, nil
			}
			<-ticker.C
			continue
		}

		if time.Now().After(autoCloseAt) {
			display.SetFooter("")
			return false, nil
		}

		<-ticker.C
	}
}

func queryDaemonStatus(paths daemon.Paths) (*daemon.StatusResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	return daemon.QueryStatus(ctx, paths)
}

func applySnapshotToDisplay(display *output.Display, resp *daemon.StatusResponse, cache map[string]string) bool {
	if resp == nil {
		return false
	}

	tunnels := append([]status.Tunnel(nil), resp.Tunnels...)
	sort.Slice(tunnels, func(i, j int) bool {
		return tunnels[i].Name < tunnels[j].Name
	})

	hasError := false
	for _, tun := range tunnels {
		ports := make([]string, 0, len(tun.Ports))
		for port := range tun.Ports {
			ports = append(ports, port)
		}
		sort.Strings(ports)
		for _, port := range ports {
			state := tun.Ports[port]
			key := tun.Name + "|" + port
			if prev, ok := cache[key]; !ok || prev != state {
				display.UpdateStatus(tun.Name, port, state)
				cache[key] = state
			}
			if !hasError && strings.HasPrefix(strings.ToLower(state), "error") {
				hasError = true
			}
		}
	}

	return hasError
}

func statusHasError(resp *daemon.StatusResponse) bool {
	if resp == nil {
		return false
	}
	for _, tun := range resp.Tunnels {
		for _, state := range tun.Ports {
			if strings.HasPrefix(strings.ToLower(state), "error") {
				return true
			}
		}
	}
	return false
}

func waitForEnterAsync() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
		close(ch)
	}()
	return ch
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func tailLogMessage(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line, nil
		}
	}
	return "", nil
}
