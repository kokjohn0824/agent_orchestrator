//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// setDetachSysProcAttr sets process attributes so the child detaches from the terminal.
// On Unix, uses setsid so the child becomes session leader and is not killed when the terminal closes.
func setDetachSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
