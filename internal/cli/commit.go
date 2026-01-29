package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitAll bool
)

var commitCmd = &cobra.Command{
	Use:   "commit [ticket-id]",
	Short: "提交變更",
	Long: `為完成的 ticket 建立 git commit。

範例:
  agent-orchestrator commit TICKET-001
  agent-orchestrator commit --all`,
	RunE: runCommit,
}

func init() {
	commitCmd.Flags().BoolVar(&commitAll, "all", false, "批次提交所有 completed tickets")
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
		ui.PrintError(w, "找不到 ticket: "+ticketID)
		return nil
	}

	if t.Status != ticket.StatusCompleted {
		ui.PrintWarning(w, fmt.Sprintf("Ticket %s 狀態為 %s，建議只提交已完成的 tickets", ticketID, t.Status))
	}

	// Get git changes
	changes := getGitStatus()
	if changes == "" {
		ui.PrintInfo(w, "沒有變更需要提交")
		return nil
	}

	ui.PrintHeader(w, "提交變更")
	ui.PrintInfo(w, "Ticket: "+ticketID+" - "+t.Title)
	ui.PrintInfo(w, "變更:")
	for _, line := range strings.Split(changes, "\n") {
		if line != "" {
			ui.PrintInfo(w, "  "+line)
		}
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

	commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)

	// Run commit
	spinner := ui.NewSpinner("產生並執行 commit...", w)
	spinner.Start()

	result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes)
	if err != nil {
		spinner.Fail("提交失敗")
		return err
	}

	if result.Success {
		spinner.Success("提交成功")
	} else {
		spinner.Fail("提交失敗: " + result.Error)
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
		ui.PrintInfo(w, "沒有 completed tickets 需要提交")
		return nil
	}

	ui.PrintHeader(w, "批次提交")
	ui.PrintInfo(w, fmt.Sprintf("準備提交 %d 個 tickets", len(completed)))

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

	commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)

	committed := 0
	failed := 0
	skipped := 0

	for i, t := range completed {
		ui.PrintStep(w, i+1, len(completed), fmt.Sprintf("提交 %s: %s", t.ID, t.Title))

		// Get current changes
		changes := getGitStatus()
		if changes == "" {
			ui.PrintInfo(w, "  沒有變更需要提交 (跳過)")
			skipped++
			continue
		}

		result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes)
		if err != nil || !result.Success {
			ui.PrintError(w, "  提交失敗")
			failed++
			continue
		}

		ui.PrintSuccess(w, "  提交成功")
		committed++
	}

	// Summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, "提交完成")
	ui.PrintSuccess(w, fmt.Sprintf("成功: %d", committed))
	if failed > 0 {
		ui.PrintError(w, fmt.Sprintf("失敗: %d", failed))
	}
	if skipped > 0 {
		ui.PrintWarning(w, fmt.Sprintf("跳過: %d", skipped))
	}

	return nil
}

func getGitStatus() string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
