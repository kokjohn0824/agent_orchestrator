package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestHasExistingCode_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if hasExistingCode(tmpDir) {
		t.Error("hasExistingCode(empty dir) should be false")
	}
}

func TestHasExistingCode_WithCodeFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create at least 3 code files
	for _, name := range []string{"main.go", "util.go", "handler.go"} {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
	}

	if !hasExistingCode(tmpDir) {
		t.Error("hasExistingCode(dir with 3 .go files) should be true")
	}
}

func TestHasExistingCode_ExcludedDirsSkipped(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Only code in node_modules - should be skipped
	nodeMod := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeMod, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		path := filepath.Join(nodeMod, name)
		if err := os.WriteFile(path, []byte("package x"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Root has no code files, so hasExistingCode should be false
	if hasExistingCode(tmpDir) {
		t.Error("hasExistingCode should not count files inside node_modules")
	}
}

func TestRunInit_WithGoalArg(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		DocsDir:          docsDir,
		AgentCommand:     "nonexistent-agent-12345",
		AgentForce:       true,
		AgentOutputFormat: "text",
		DryRun:           false, // so CreateAgentCaller fails
		MaxParallel:      3,
	}

	// runInit with goal in args; CreateAgentCaller will fail (no dry run)
	err = runInit(nil, []string{"my project goal"})
	if err != nil {
		t.Fatalf("runInit returns nil on agent error (prints message), got: %v", err)
	}
}

func TestRunInit_AgentNotFound_PrintsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	docsDir := filepath.Join(tmpDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		DocsDir:          docsDir,
		AgentCommand:     "nonexistent-agent-xyz",
		AgentForce:       true,
		AgentOutputFormat: "text",
		DryRun:           false,
		MaxParallel:      3,
	}

	// Capture stdout to verify error is printed
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err = runInit(nil, []string{"goal"})
	w.Close()
	if err != nil {
		t.Fatalf("runInit should return nil when agent not found (error is printed): %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if out == "" {
		t.Error("runInit should print something when agent not found")
	}
}

func TestInitCmd_UseAndArgs(t *testing.T) {
	if initCmd.Use != "init [goal]" {
		t.Errorf("initCmd.Use = %q, want init [goal]", initCmd.Use)
	}
	if initCmd.Short == "" {
		t.Error("initCmd.Short should be set")
	}
}
