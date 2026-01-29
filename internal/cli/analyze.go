package cli

import (
	"context"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	analyzeScope     []string
	analyzeAutoGen   bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "分析現有專案，產生改進 issues 和 tickets",
	Long: `分析現有專案的程式碼，找出可改進的地方，包括效能問題、重構建議、安全性問題等。

範例:
  agent-orchestrator analyze
  agent-orchestrator analyze --scope performance,refactor
  agent-orchestrator analyze --scope security --auto`,
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringSliceVar(&analyzeScope, "scope", []string{"all"}, 
		"分析範圍: all, performance, refactor, security, test, docs (可用逗號分隔多個)")
	analyzeCmd.Flags().BoolVar(&analyzeAutoGen, "auto", false, "自動產生 tickets 不詢問")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	ui.PrintHeader(w, "專案分析")
	ui.PrintInfo(w, "分析專案: "+cfg.ProjectRoot)
	ui.PrintInfo(w, "分析範圍: "+strings.Join(analyzeScope, ", "))

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

	analyzeAgent := agent.NewAnalyzeAgent(caller, cfg.ProjectRoot)
	scope := agent.ParseScopes(analyzeScope)

	// Run analysis
	spinner := ui.NewSpinner("分析專案中...", w)
	spinner.Start()

	issues, err := analyzeAgent.Analyze(ctx, scope)
	if err != nil {
		spinner.Fail("分析失敗")
		return err
	}
	spinner.Success("分析完成")

	if issues.Count() == 0 {
		ui.PrintSuccess(w, "沒有發現問題！")
		return nil
	}

	// Display issues by category
	ui.PrintHeader(w, "分析報告")

	categories := []struct {
		name     string
		category string
	}{
		{"效能問題", "performance"},
		{"重構建議", "refactor"},
		{"安全性問題", "security"},
		{"測試覆蓋", "test"},
		{"文件缺失", "docs"},
	}

	for _, cat := range categories {
		filtered := issues.FilterByCategory(cat.category)
		if len(filtered) > 0 {
			table := ui.NewIssueTable(cat.name)
			for _, issue := range filtered {
				table.AddIssue(issue.Severity, issue.Title, issue.Location)
			}
			table.Render(w)
		}
	}

	ui.PrintInfo(w, "")
	ui.PrintInfo(w, "共發現 "+string(rune('0'+issues.Count()))+" 個問題")

	// Ask to generate tickets
	generateTickets := analyzeAutoGen
	if !generateTickets && !cfg.Quiet {
		prompt := ui.NewPrompt(os.Stdin, w)
		var err error
		generateTickets, err = prompt.Confirm("要產生對應的 tickets 嗎？", true)
		if err != nil {
			return err
		}
	}

	if generateTickets {
		return generateTicketsFromIssues(issues)
	}

	return nil
}

func generateTicketsFromIssues(issues *ticket.IssueList) error {
	w := os.Stdout

	// Convert issues to tickets
	ticketList := issues.ToTickets()

	// Save tickets
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return err
	}

	for _, t := range ticketList.Tickets {
		if err := store.Save(t); err != nil {
			ui.PrintError(w, "儲存 ticket 失敗: "+t.ID)
			continue
		}
		ui.PrintSuccess(w, "建立 ticket: "+t.ID+" - "+t.Title)
	}

	ui.PrintInfo(w, "")
	ui.PrintSuccess(w, "已產生 "+string(rune('0'+ticketList.Count()))+" 個 tickets 到 "+cfg.TicketsDir)
	ui.PrintInfo(w, "執行 'agent-orchestrator work' 開始處理 tickets")

	return nil
}
