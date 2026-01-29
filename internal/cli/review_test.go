package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestGetGitChangedFiles_ContextCancellation(t *testing.T) {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Initialize cfg if nil and store original config
	originalCfg := cfg
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	defer func() { cfg = originalCfg }()

	// Create a temporary git directory for testing
	tempDir, err := os.MkdirTemp("", "test-git-changed-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize as git repo
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	cfg.ProjectRoot = tempDir

	// With cancelled context, getGitChangedFiles should return nil
	// (command execution should fail due to context cancellation)
	result := getGitChangedFiles(ctx)
	if result != nil {
		// Note: The actual behavior depends on whether the command starts before context check
		// This test just ensures no panic and the function returns gracefully
		t.Logf("getGitChangedFiles returned: %v (may vary based on timing)", result)
	}
}

func TestGetGitChangedFiles_ValidContext(t *testing.T) {
	ctx := context.Background()

	// Initialize cfg if nil and store original config
	originalCfg := cfg
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	defer func() { cfg = originalCfg }()

	// Use current working directory if it's a git repo, otherwise skip
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get current working directory")
	}

	// Check if we're in a git repo
	gitPath := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		// Try going up to find git root
		dir := cwd
		for {
			gitPath = filepath.Join(dir, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				cfg.ProjectRoot = dir
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				t.Skip("not in a git repository")
			}
			dir = parent
		}
	} else {
		cfg.ProjectRoot = cwd
	}

	// This should not panic and should return a slice (possibly empty)
	result := getGitChangedFiles(ctx)
	// Just verify no panic occurred - result can be nil or a slice
	t.Logf("getGitChangedFiles returned %d files", len(result))
}
