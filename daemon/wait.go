package daemon

import (
	"errors"
	"os"
	"time"
)

// WaitForSocket waits for the daemon to create its control socket within the timeout.
func WaitForSocket(paths Paths, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(paths.SocketFile); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if time.Now().After(deadline) {
			return errors.New("daemon did not create control socket in time")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
