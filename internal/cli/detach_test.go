package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestWriteWorkPIDFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Path under existing dir: .tickets/.work.pid style
	path := filepath.Join(tmpDir, ".tickets", ".work.pid")
	if err := WriteWorkPIDFile(path); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("pid not an integer: %q", pidStr)
	}
	expected := os.Getpid()
	if pid != expected {
		t.Errorf("pid file content = %d, want %d", pid, expected)
	}
}

func TestWriteWorkPIDFile_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Path with non-existent parent
	path := filepath.Join(tmpDir, "nested", "dir", ".work.pid")
	if err := WriteWorkPIDFile(path); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("pid file should exist: %v", err)
	}
	dir := filepath.Dir(path)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("parent dir should exist and be directory: %v", err)
	}
}

func TestRemoveWorkPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".work.pid")
	if err := WriteWorkPIDFile(path); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("pid file should exist before remove: %v", err)
	}
	RemoveWorkPIDFile(path)
	if _, err := os.Stat(path); err == nil {
		t.Fatal("pid file should be removed")
	}
}

func TestRemoveWorkPIDFile_EmptyPath(t *testing.T) {
	RemoveWorkPIDFile("") // should not panic
}

func TestRemoveWorkPIDFile_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.pid")
	RemoveWorkPIDFile(path) // should not panic; idempotent
}

func TestReadWorkPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".work.pid")
	if err := WriteWorkPIDFile(path); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	pid, err := ReadWorkPIDFile(path)
	if err != nil {
		t.Fatalf("ReadWorkPIDFile: %v", err)
	}
	if pid != os.Getpid() {
		t.Errorf("ReadWorkPIDFile() pid = %d, want %d", pid, os.Getpid())
	}
}

func TestReadWorkPIDFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.pid")
	_, err := ReadWorkPIDFile(path)
	if err == nil {
		t.Fatal("ReadWorkPIDFile expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("ReadWorkPIDFile error = %v, want os.ErrNotExist", err)
	}
}

func TestReadWorkPIDFile_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".work.pid")
	if err := os.WriteFile(path, []byte("not-a-number\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	pid, err := ReadWorkPIDFile(path)
	if err == nil {
		t.Fatalf("ReadWorkPIDFile expected error for invalid content, got pid=%d", pid)
	}
}

func TestErrIfBackgroundWorkRunning_NilConfig(t *testing.T) {
	old := cfg
	defer func() { cfg = old }()
	cfg = nil
	if err := ErrIfBackgroundWorkRunning(); err != nil {
		t.Errorf("ErrIfBackgroundWorkRunning() with nil config = %v, want nil", err)
	}
}

func TestErrIfBackgroundWorkRunning_NoPidFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, ".work.pid")
	old := cfg
	defer func() { cfg = old }()
	cfg = &config.Config{WorkPIDFile: pidPath}
	if err := ErrIfBackgroundWorkRunning(); err != nil {
		t.Errorf("ErrIfBackgroundWorkRunning() with no PID file = %v, want nil", err)
	}
}

func TestErrIfBackgroundWorkRunning_PidAlive(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, ".work.pid")
	if err := WriteWorkPIDFile(pidPath); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	old := cfg
	defer func() { cfg = old }()
	cfg = &config.Config{WorkPIDFile: pidPath}
	err := ErrIfBackgroundWorkRunning()
	if err == nil {
		t.Fatal("ErrIfBackgroundWorkRunning() with alive PID file want error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "背景 work 執行中") {
		t.Errorf("ErrIfBackgroundWorkRunning() error = %v, want message containing 背景 work 執行中", err)
	}
}

func TestErrIfBackgroundWorkRunning_PidDead(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, ".work.pid")
	// Use a PID that is very unlikely to exist (e.g. 99999999)
	if err := os.WriteFile(pidPath, []byte("99999999\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	old := cfg
	defer func() { cfg = old }()
	cfg = &config.Config{WorkPIDFile: pidPath}
	if err := ErrIfBackgroundWorkRunning(); err != nil {
		t.Errorf("ErrIfBackgroundWorkRunning() with dead PID = %v, want nil", err)
	}
}
