package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestParsePorcelainLinePath(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{" M internal/cli/commit.go", "internal/cli/commit.go"},
		{"M  staged.go", "staged.go"},
		{"?? untracked.go", "untracked.go"},
		{"?? docs/", "docs/"},
		{"R  old.go -> new.go", "new.go"},
		{"A  added.go", "added.go"},
		{" D deleted.go", "deleted.go"},
		{"", ""},
		{"xy", ""},
		{"ab", ""},
		{"  ", ""},
		{" M ", ""},
	}
	for _, tt := range tests {
		got := parsePorcelainLinePath(tt.line)
		if got != tt.want {
			t.Errorf("parsePorcelainLinePath(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestGetGitChangedFiles_IncludesUntracked(t *testing.T) {
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "test-git-untracked-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Create an untracked file (git diff HEAD would not list it)
	untrackedPath := filepath.Join(tempDir, "newfile.txt")
	if err := os.WriteFile(untrackedPath, []byte("untracked"), 0644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}
	relPath := "newfile.txt"

	originalCfg := cfg
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	cfg.ProjectRoot = tempDir
	defer func() { cfg = originalCfg }()

	result := getGitChangedFiles(ctx)
	if result == nil {
		t.Fatal("getGitChangedFiles() = nil, want list including untracked file")
	}
	found := false
	for _, p := range result {
		if p == relPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("getGitChangedFiles() = %v, want to include %q (untracked)", result, relPath)
	}
}

func TestGetGitStatusForFiles_MatchesRenamedPath(t *testing.T) {
	// getGitStatusForFiles uses parsePorcelainLinePath, so when files contain
	// the "new" path from a rename line "R  old -> new", that line should be
	// included. We test this by verifying parsePorcelainLinePath("R  a -> b") == "b".
	// Full integration would require a repo with a rename; the helper behavior
	// is already covered by TestParsePorcelainLinePath.
	if got := parsePorcelainLinePath("R  old.go -> new.go"); got != "new.go" {
		t.Errorf("rename path: got %q, want new.go", got)
	}
}
