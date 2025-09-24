//go:build !unix

package daemon

func isProcessRunning(pid int) bool {
	return false
}
