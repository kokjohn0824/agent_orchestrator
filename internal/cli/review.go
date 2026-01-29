package cli

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review [files...]",
	Short: "執行程式碼審查",
	Long: `對變更的檔案執行程式碼審查。如果沒有指定檔案，會自動取得 git 變更的檔案。

範例:
  agent-orchestrator review
  agent-orchestrator review src/main.go src/util.go`,
	RunE: runReview,
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
		files = getGitChangedFiles()
	}

	if len(files) == 0 {
		ui.PrintInfo(w, "沒有檔案需要審查")
		return nil
	}

	ui.PrintHeader(w, "程式碼審查")
	ui.PrintInfo(w, "審查檔案:")
	for _, f := range files {
		ui.PrintInfo(w, "  - "+f)
	}

	// Create agent caller
	caller := agent.NewCaller(
		cfg.AgentCommand,
		cfg.AgentForce,
		cfg.AgentOutputFormat,
		cfg.LogsDir,
	)
	caller.SetDryRun(cfg.DryRun)
	caller.SetVerbose(cfg.Verbose)

	if !caller.IsAvailable() && !cfg.DryRun {
		ui.PrintError(w, "找不到 agent 指令，請確保已安裝 Cursor CLI")
		return nil
	}

	reviewAgent := agent.NewReviewAgent(caller, cfg.ProjectRoot)

	// Run review
	spinner := ui.NewSpinner("審查程式碼中...", w)
	spinner.Start()

	result, reviewResult, err := reviewAgent.Review(ctx, files)
	if err != nil {
		spinner.Fail("審查失敗")
		return err
	}

	if reviewResult != nil {
		if reviewResult.Status == "APPROVED" {
			spinner.Success("審查通過")
		} else if reviewResult.Status == "CHANGES_REQUESTED" {
			spinner.Fail("審查需要修改")
		} else {
			spinner.Info("審查完成")
		}

		if reviewResult.Summary != "" {
			ui.PrintInfo(w, "")
			ui.PrintInfo(w, "摘要: "+reviewResult.Summary)
		}
	} else {
		spinner.Success("審查完成")
	}

	// Print full output if verbose
	if cfg.Verbose && result != nil {
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, "完整輸出:")
		ui.PrintInfo(w, result.Output)
	}

	return nil
}

func getGitChangedFiles() []string {
	// Try git diff --name-only HEAD
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.Output()
	if err != nil {
		// Try git status --porcelain
		cmd = exec.Command("git", "status", "--porcelain")
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
