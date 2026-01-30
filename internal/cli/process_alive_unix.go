//go:build !windows

package cli

import "syscall"

// IsProcessAlive reports whether the process with the given PID exists and is running.
// On Unix, uses kill(pid, 0) which sends no signal but checks process existence.
func IsProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
