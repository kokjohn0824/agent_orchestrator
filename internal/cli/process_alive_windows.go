//go:build windows

package cli

import "syscall"

// PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
const processQueryLimitedInformation = 0x1000

// IsProcessAlive reports whether the process with the given PID exists and is running.
// On Windows, uses OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION; success means the process exists.
func IsProcessAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil || h == 0 {
		return false
	}
	defer syscall.CloseHandle(h)
	return true
}
