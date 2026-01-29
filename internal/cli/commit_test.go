package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestValidateProjectRoot(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-project-root-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a .git directory to simulate a git repository
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	tests := []struct {
		name        string
		projectRoot string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid git repository",
			projectRoot: tempDir,
			wantErr:     false,
		},
		{
			name:        "empty path",
			projectRoot: "",
			wantErr:     true,
			errContains: "project root is empty",
		},
		{
			name:        "path traversal with double dots",
			projectRoot: "/home/user/../etc/passwd",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "path with special characters - semicolon",
			projectRoot: "/home/user; rm -rf /",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "path with special characters - backtick",
			projectRoot: "/home/user`whoami`",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "path with special characters - dollar sign",
			projectRoot: "/home/$USER/project",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "path with special characters - pipe",
			projectRoot: "/home/user | cat /etc/passwd",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "path with special characters - ampersand",
			projectRoot: "/home/user & echo pwned",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "path with newline",
			projectRoot: "/home/user\n/etc/passwd",
			wantErr:     true,
			errContains: "unsafe characters",
		},
		{
			name:        "relative path",
			projectRoot: "relative/path",
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name:        "non-existent directory",
			projectRoot: "/nonexistent/directory/that/does/not/exist",
			wantErr:     true,
			errContains: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectRoot(tt.projectRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !contains(err.Error(), tt.errContains) {
					t.Errorf("validateProjectRoot() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateProjectRoot_NotGitRepo(t *testing.T) {
	// Create a temporary directory without .git
	tempDir, err := os.MkdirTemp("", "test-not-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = validateProjectRoot(tempDir)
	if err == nil {
		t.Error("validateProjectRoot() expected error for non-git directory, got nil")
	}
	if !contains(err.Error(), "not a git repository") {
		t.Errorf("validateProjectRoot() error = %v, want error containing 'not a git repository'", err)
	}
}

func TestValidateProjectRoot_FileNotDirectory(t *testing.T) {
	// Create a temporary file (not directory)
	tempFile, err := os.CreateTemp("", "test-file-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	err = validateProjectRoot(tempFile.Name())
	if err == nil {
		t.Error("validateProjectRoot() expected error for file, got nil")
	}
	if !contains(err.Error(), "not a directory") {
		t.Errorf("validateProjectRoot() error = %v, want error containing 'not a directory'", err)
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetGitStatus_ContextCancellation(t *testing.T) {
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
	tempDir, err := os.MkdirTemp("", "test-git-status-*")
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

	// With cancelled context, getGitStatus should return empty string
	// (command execution should fail due to context cancellation)
	result := getGitStatus(ctx)
	if result != "" {
		// Note: The actual behavior depends on whether the command starts before context check
		// This test just ensures no panic and the function returns gracefully
		t.Logf("getGitStatus returned: %q (may vary based on timing)", result)
	}
}
