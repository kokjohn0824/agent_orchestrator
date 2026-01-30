package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
)

// WriteWorkPIDFile writes the current process PID to path, creating the parent directory if needed.
// Used by the work detach-child process so that the PID file exists before entering work logic.
// Path is typically from config.WorkPIDFilePath() (e.g. .tickets/.work.pid).
func WriteWorkPIDFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("create pid dir: %w", err)
		}
	}
	pid := []byte(fmt.Sprintf("%d\n", os.Getpid()))
	if err := os.WriteFile(path, pid, 0600); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	return nil
}

// ReadWorkPIDFile reads the PID from the work PID file at path.
// Returns the PID and nil if the file exists and contains a valid PID;
// otherwise returns 0 and a non-nil error (e.g. os.ErrNotExist, parse error).
func ReadWorkPIDFile(path string) (int, error) {
	if path == "" {
		return 0, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	line := strings.TrimSpace(string(data))
	if line == "" {
		return 0, fmt.Errorf("empty pid file")
	}
	pid, err := strconv.Atoi(line)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid in file: %s", line)
	}
	return pid, nil
}

// RemoveWorkPIDFile removes the PID file at path. Safe to call when the file
// does not exist (e.g. already removed). Used by work detach-child on exit
// and on SIGTERM/SIGINT so the PID file is always cleaned up.
func RemoveWorkPIDFile(path string) {
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

// ErrIfBackgroundWorkRunning returns an error if the work PID file exists and
// the process is alive (i.e. background work is running). Call this at the
// start of CLI commands that write to the store (plan, work, run, add, etc.).
// When running as detach-child, the caller should skip this check (we are the
// background work). See docs/ticket-store-concurrency.md (TICKET-018).
func ErrIfBackgroundWorkRunning() error {
	if cfg == nil {
		return nil
	}
	pidPath := cfg.WorkPIDFilePath()
	pid, err := ReadWorkPIDFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil // invalid or missing PID: no running background work
	}
	if pid <= 0 {
		return nil
	}
	if IsProcessAlive(pid) {
		return fmt.Errorf(i18n.ErrBackgroundWorkRunning, pid)
	}
	return nil
}
