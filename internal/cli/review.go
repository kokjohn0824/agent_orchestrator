package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [files...]",
	Short: i18n.CmdReviewShort,
	Long:  i18n.CmdReviewLong,
	RunE:  runReview,
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	// Get files to review
	var files []string
	if len(args) > 0 {
		files = args
	} else {
		// Get changed files from git
		files = getGitChangedFiles(ctx)
	}

	if len(files) == 0 {
		ui.PrintInfo(w, i18n.MsgNoFilesToReview)
		return nil
	}

	ui.PrintHeader(w, i18n.UICodeReview)
	ui.PrintInfo(w, i18n.MsgReviewFiles)
	for _, f := range files {
		ui.PrintInfo(w, "  - "+f)
	}

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	reviewAgent := agent.NewReviewAgent(caller, cfg.ProjectRoot)

	// Run review
	spinner := ui.NewSpinner(i18n.SpinnerReviewing, w)
	spinner.Start()

	result, reviewResult, err := reviewAgent.Review(ctx, files)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailReview)
		return err
	}

	if reviewResult != nil {
		if reviewResult.Status == "APPROVED" {
			spinner.Success(i18n.MsgReviewApproved)
		} else if reviewResult.Status == "CHANGES_REQUESTED" {
			spinner.Fail(i18n.SpinnerFailReviewNeeds)
		} else {
			spinner.Info(i18n.MsgReviewComplete)
		}

		if reviewResult.Summary != "" {
			ui.PrintInfo(w, "")
			ui.PrintInfo(w, fmt.Sprintf(i18n.MsgSummary, reviewResult.Summary))
		}
	} else {
		spinner.Success(i18n.MsgReviewComplete)
	}

	// Print full output if verbose
	if cfg.Verbose && result != nil {
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, i18n.MsgFullOutput)
		ui.PrintInfo(w, result.Output)
	}

	return nil
}

func getGitChangedFiles(ctx context.Context) []string {
	// Try git diff --name-only HEAD
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.Output()
	if err != nil {
		// Check if context was cancelled
		if ctx.Err() != nil {
			return nil
		}
		// Try git status --porcelain
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
		// Handle git status --porcelain format (e.g., "M  file.go")
		if len(line) > 3 && line[2] == ' ' {
			line = strings.TrimSpace(line[3:])
		}
		files = append(files, line)
	}

	return files
}
