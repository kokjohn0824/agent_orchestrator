package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitAll bool
)

var commitCmd = &cobra.Command{
	Use:   "commit [ticket-id]",
	Short: i18n.CmdCommitShort,
	Long:  i18n.CmdCommitLong,
	RunE:  runCommit,
}

func init() {
	commitCmd.Flags().BoolVar(&commitAll, "all", false, i18n.FlagCommitAll)
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	store := ticket.NewStore(cfg.TicketsDir)

	if commitAll {
		return commitAllTickets(ctx, store)
	}

	if len(args) == 0 {
		ui.PrintError(w, "請提供 ticket ID 或使用 --all")
		return nil
	}

	return commitSingleTicket(ctx, store, args[0])
}

func commitSingleTicket(ctx context.Context, store *ticket.Store, ticketID string) error {
	w := os.Stdout

	t, err := store.Load(ticketID)
	if err != nil {
		ui.PrintError(w, fmt.Sprintf(i18n.ErrTicketNotFound, ticketID))
		return nil
	}

	if t.Status != ticket.StatusCompleted {
		ui.PrintWarning(w, fmt.Sprintf(i18n.MsgTicketStatusWarning, ticketID, t.Status))
	}

	// Get git changes
	changes := getGitStatus(ctx)
	if changes == "" {
		ui.PrintInfo(w, i18n.MsgNoChangesToCommit)
		return nil
	}

	ui.PrintHeader(w, i18n.UICommitChanges)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgTicket, ticketID, t.Title))
	ui.PrintInfo(w, i18n.MsgChanges)
	for _, line := range strings.Split(changes, "\n") {
		if line != "" {
			ui.PrintInfo(w, "  "+line)
		}
	}

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)

	// Run commit
	spinner := ui.NewSpinner(i18n.SpinnerCommitting, w)
	spinner.Start()

	result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailCommit)
		return err
	}

	if result.Success {
		spinner.Success(i18n.MsgCommitSuccess)
	} else {
		spinner.Fail(i18n.SpinnerFailCommit + ": " + result.Error)
	}

	return nil
}

func commitAllTickets(ctx context.Context, store *ticket.Store) error {
	w := os.Stdout

	completed, err := store.LoadByStatus(ticket.StatusCompleted)
	if err != nil {
		return err
	}

	if len(completed) == 0 {
		ui.PrintInfo(w, i18n.MsgNoCompletedCommit)
		return nil
	}

	ui.PrintHeader(w, i18n.UIBatchCommit)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgPrepareCommit, len(completed)))

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)

	committed := 0
	failed := 0
	skipped := 0

	for i, t := range completed {
		ui.PrintStep(w, i+1, len(completed), fmt.Sprintf("提交 %s: %s", t.ID, t.Title))

		// Get current changes
		changes := getGitStatus(ctx)
		if changes == "" {
			ui.PrintInfo(w, "  "+i18n.MsgSkipNoChanges)
			skipped++
			continue
		}

		result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes)
		if err != nil || !result.Success {
			ui.PrintError(w, "  "+i18n.SpinnerFailCommit)
			failed++
			continue
		}

		ui.PrintSuccess(w, "  "+i18n.MsgCommitSuccess)
		committed++
	}

	// Summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, i18n.UICommitComplete)
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgCountSuccess, committed))
	if failed > 0 {
		ui.PrintError(w, fmt.Sprintf(i18n.MsgCountFailed, failed))
	}
	if skipped > 0 {
		ui.PrintWarning(w, fmt.Sprintf(i18n.MsgCountSkipped, skipped))
	}

	return nil
}

// validateProjectRoot validates that the project root is a safe and valid git repository.
// It checks for:
// 1. Path traversal attacks (../)
// 2. Dangerous special characters
// 3. The path being an absolute path
// 4. The directory being a valid git repository
func validateProjectRoot(projectRoot string) error {
	// Check for empty path
	if projectRoot == "" {
		return fmt.Errorf("project root is empty")
	}

	// Check for path traversal
	if strings.Contains(projectRoot, "..") {
		return fmt.Errorf("project root contains path traversal sequence (..): %s", projectRoot)
	}

	// Check for dangerous special characters that could be used for command injection
	// Allow only alphanumeric, dash, underscore, dot, forward slash (for paths)
	// This regex matches any character that is NOT safe
	unsafePattern := regexp.MustCompile(`[^a-zA-Z0-9_\-./]`)
	if unsafePattern.MatchString(projectRoot) {
		return fmt.Errorf("project root contains unsafe characters: %s", projectRoot)
	}

	// Ensure the path is absolute
	if !filepath.IsAbs(projectRoot) {
		return fmt.Errorf("project root must be an absolute path: %s", projectRoot)
	}

	// Clean the path and verify it doesn't change (catches normalized traversal)
	cleanPath := filepath.Clean(projectRoot)
	if cleanPath != projectRoot {
		// Allow trailing slash difference
		if strings.TrimSuffix(projectRoot, "/") != cleanPath {
			return fmt.Errorf("project root contains non-canonical path: %s", projectRoot)
		}
	}

	// Verify the directory exists
	info, err := os.Stat(projectRoot)
	if err != nil {
		return fmt.Errorf("project root does not exist: %s", projectRoot)
	}
	if !info.IsDir() {
		return fmt.Errorf("project root is not a directory: %s", projectRoot)
	}

	// Verify it's a git repository by checking for .git directory or file
	gitPath := filepath.Join(projectRoot, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return fmt.Errorf("project root is not a git repository (missing .git): %s", projectRoot)
	}

	return nil
}

func getGitStatus(ctx context.Context) string {
	// Validate project root before executing git command
	if err := validateProjectRoot(cfg.ProjectRoot); err != nil {
		// Log the security validation error but return empty to maintain existing behavior
		// In production, this should be logged to a security audit log
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
