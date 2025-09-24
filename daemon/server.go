package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/strandnerd/tunn/status"
)

// StatusRequest represents an IPC command from the CLI.
type StatusRequest struct {
	Command string `json:"command"`
}

// StatusResponse captures the daemon state for CLI consumption.
type StatusResponse struct {
	Running bool            `json:"running"`
	Mode    string          `json:"mode"`
	PID     int             `json:"pid"`
	Message string          `json:"message,omitempty"`
	Tunnels []status.Tunnel `json:"tunnels,omitempty"`
}

// Server handles IPC communication with CLI clients.
type Server struct {
	paths  Paths
	store  *status.Store
	pid    int
	mu     sync.Mutex
	ln     net.Listener
	stopFn func()
}

// NewServer constructs a server bound to the given socket and status store.
func NewServer(paths Paths, store *status.Store, pid int, stopFn func()) *Server {
	return &Server{
		paths:  paths,
		store:  store,
		pid:    pid,
		stopFn: stopFn,
	}
}

// Run starts the IPC server and blocks until the context is cancelled or the listener fails.
func (s *Server) Run(ctx context.Context) error {
	if err := os.Remove(s.paths.SocketFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	ln, err := net.Listen("unix", s.paths.SocketFile)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}
	if err := os.Chmod(s.paths.SocketFile, 0o600); err != nil {
		ln.Close()
		return fmt.Errorf("failed to secure socket permissions: %w", err)
	}

	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	defer func() {
		ln.Close()
		_ = os.Remove(s.paths.SocketFile)
	}()

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handleConnection(conn)
	}
}

// Close terminates the listener if active.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		_ = s.ln.Close()
		s.ln = nil
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	var req StatusRequest
	if err := decoder.Decode(&req); err != nil {
		return
	}

	switch req.Command {
	case "status":
		s.handleStatus(conn)
	case "stop":
		s.handleStop(conn)
	default:
		return
	}
}

func (s *Server) handleStatus(conn net.Conn) {
	encoder := json.NewEncoder(conn)
	snapshot := s.store.Snapshot()
	resp := StatusResponse{
		Running: true,
		Mode:    "daemon",
		PID:     s.pid,
		Tunnels: snapshot,
	}
	_ = encoder.Encode(resp)
}

func (s *Server) handleStop(conn net.Conn) {
	encoder := json.NewEncoder(conn)
	resp := StatusResponse{
		Running: false,
		Mode:    "daemon",
		PID:     s.pid,
		Message: "stopping",
		Tunnels: s.store.Snapshot(),
	}
	_ = encoder.Encode(resp)
	if s.stopFn != nil {
		s.stopFn()
	}
}
