package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
)

func sendRequest(ctx context.Context, paths Paths, command string) (*StatusResponse, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", paths.SocketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon socket: %w", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(StatusRequest{Command: command}); err != nil {
		return nil, fmt.Errorf("failed to send %s request: %w", command, err)
	}

	var resp StatusResponse
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode daemon response: %w", err)
	}
	return &resp, nil
}

// QueryStatus contacts the daemon control socket for a status snapshot.
func QueryStatus(ctx context.Context, paths Paths) (*StatusResponse, error) {
	return sendRequest(ctx, paths, "status")
}

// SendStop requests the daemon to initiate shutdown.
func SendStop(ctx context.Context, paths Paths) (*StatusResponse, error) {
	return sendRequest(ctx, paths, "stop")
}
