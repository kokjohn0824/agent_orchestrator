//go:build windows

package cli

import (
	"os/exec"
	"syscall"
)

// DETACHED_PROCESS: new process has no console and does not inherit the parent's.
const detachedProcess = 0x00000008

// setDetachSysProcAttr sets process attributes so the child detaches from the terminal.
// On Windows, uses DETACHED_PROCESS so the child is not attached to any console.
func setDetachSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
}
