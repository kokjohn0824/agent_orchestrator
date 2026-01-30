package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// getGitChangedFiles returns the list of file paths that have been modified
// in the working tree (compared to HEAD), or nil if the project root is invalid
// or the git command fails. Used by run and review commands.
func getGitChangedFiles(ctx context.Context) []string {
	if err := validateProjectRoot(cfg.ProjectRoot); err != nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
		cmd.Dir = cfg.ProjectRoot
		output, err = cmd.Output()
		if err != nil {
			return nil
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 3 && line[2] == ' ' {
			line = strings.TrimSpace(line[3:])
		}
		files = append(files, line)
	}

	return files
}

// getGitStatusForFiles returns only the "git status --porcelain" lines whose
// path is in files. Used by commit to restrict which changes are shown/staged
// per ticket. Returns empty string if files is empty or no matching lines.
func getGitStatusForFiles(ctx context.Context, files []string) string {
	if len(files) == 0 {
		return ""
	}
	filesSet := make(map[string]struct{}, len(files))
	for _, f := range files {
		filesSet[f] = struct{}{}
	}
	full := getGitStatus(ctx)
	if full == "" {
		return ""
	}
	var out []string
	for _, line := range strings.Split(full, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var path string
		if len(line) > 3 && line[2] == ' ' {
			path = strings.TrimSpace(line[3:])
		} else {
			path = line
		}
		if _, ok := filesSet[path]; ok {
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// getGitStatus returns the output of "git status --porcelain" for the project
// root, or empty string if the root is invalid or the command fails. Used by
// run and commit commands.
func getGitStatus(ctx context.Context) string {
	if err := validateProjectRoot(cfg.ProjectRoot); err != nil {
		return ""
	}

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// validateProjectRoot checks that the project root is a safe and valid git
// repository (no path traversal, no dangerous characters, absolute path, .git
// present). Used by git helpers before running any git command.
func validateProjectRoot(projectRoot string) error {
	if projectRoot == "" {
		return fmt.Errorf("project root is empty")
	}
	if strings.Contains(projectRoot, "..") {
		return fmt.Errorf("project root contains path traversal sequence (..): %s", projectRoot)
	}
	unsafePattern := regexp.MustCompile(`[^a-zA-Z0-9_\-./]`)
	if unsafePattern.MatchString(projectRoot) {
		return fmt.Errorf("project root contains unsafe characters: %s", projectRoot)
	}
	if !filepath.IsAbs(projectRoot) {
		return fmt.Errorf("project root must be an absolute path: %s", projectRoot)
	}
	cleanPath := filepath.Clean(projectRoot)
	if cleanPath != projectRoot {
		if strings.TrimSuffix(projectRoot, "/") != cleanPath {
			return fmt.Errorf("project root contains non-canonical path: %s", projectRoot)
		}
	}
	info, err := os.Stat(projectRoot)
	if err != nil {
		return fmt.Errorf("project root does not exist: %s", projectRoot)
	}
	if !info.IsDir() {
		return fmt.Errorf("project root is not a directory: %s", projectRoot)
	}
	gitPath := filepath.Join(projectRoot, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return fmt.Errorf("project root is not a git repository (missing .git): %s", projectRoot)
	}
	return nil
}
