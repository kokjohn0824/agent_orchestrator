package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan <milestone-file>",
	Short: "分析 milestone 並產生 tickets",
	Long: `分析 milestone 文件，將其分解為可執行的 tickets。

範例:
  agent-orchestrator plan docs/milestone-001.md
  agent-orchestrator plan docs/milestone.md --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runPlan,
}

func runPlan(cmd *cobra.Command, args []string) error {
	return runPlanWithFile(context.Background(), args[0])
}

func runPlanWithFile(ctx context.Context, milestoneFile string) error {
	w := os.Stdout

	// Check if milestone file exists
	if _, err := os.Stat(milestoneFile); os.IsNotExist(err) {
		ui.PrintError(w, "Milestone 檔案不存在: "+milestoneFile)
		return nil
	}

	ui.PrintHeader(w, "規劃階段")
	ui.PrintInfo(w, "分析 Milestone: "+milestoneFile)

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

	planningAgent := agent.NewPlanningAgent(caller, cfg.ProjectRoot, cfg.TicketsDir)

	// Run planning
	spinner := ui.NewSpinner("分析並產生 tickets...", w)
	spinner.Start()

	tickets, err := planningAgent.Plan(ctx, milestoneFile)
	if err != nil {
		spinner.Fail("規劃失敗")
		return err
	}
	spinner.Success("規劃完成")

	if len(tickets) == 0 {
		ui.PrintWarning(w, "沒有產生任何 tickets")
		return nil
	}

	// Initialize store and save tickets
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf("初始化 ticket store 失敗: %w", err)
	}

	// Validate dependencies
	resolver := ticket.NewDependencyResolver(store)
	if err := resolver.ValidateDependencies(tickets); err != nil {
		ui.PrintWarning(w, "依賴驗證警告: "+err.Error())
	}

	// Check for circular dependencies
	if resolver.HasCircularDependency(tickets) {
		ui.PrintWarning(w, "警告: 發現循環依賴")
	}

	// Save tickets
	for _, t := range tickets {
		if err := store.Save(t); err != nil {
			ui.PrintError(w, "儲存 ticket 失敗: "+t.ID)
			continue
		}
	}

	// Display results
	ui.PrintInfo(w, "")
	ui.PrintSuccess(w, fmt.Sprintf("已產生 %d 個 tickets", len(tickets)))
	ui.PrintInfo(w, "")

	// Show ticket list
	table := ui.NewTable("Priority", "ID", "Title", "Type", "Complexity")
	for _, t := range tickets {
		priority := ui.PriorityStyle(t.Priority).Render(fmt.Sprintf("P%d", t.Priority))
		table.AddRow(priority, t.ID, truncateTitle(t.Title, 40), string(t.Type), t.EstimatedComplexity)
	}
	table.Render(w)

	ui.PrintInfo(w, "")
	ui.PrintInfo(w, "執行 'agent-orchestrator work' 開始處理 tickets")
	ui.PrintInfo(w, "執行 'agent-orchestrator status' 查看狀態")

	return nil
}

func truncateTitle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
