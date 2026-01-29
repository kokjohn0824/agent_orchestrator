package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	runAnalyzeFirst bool
	runSkipTest     bool
	runSkipReview   bool
	runSkipCommit   bool
)

var runCmd = &cobra.Command{
	Use:   "run <milestone-file>",
	Short: "執行完整 pipeline",
	Long: `執行完整的開發 pipeline: plan -> work -> test -> review -> commit

範例:
  agent-orchestrator run docs/milestone.md
  agent-orchestrator run docs/milestone.md --analyze-first
  agent-orchestrator run docs/milestone.md --skip-test --skip-review`,
	Args: cobra.ExactArgs(1),
	RunE: runPipeline,
}

func init() {
	runCmd.Flags().BoolVar(&runAnalyzeFirst, "analyze-first", false, "先執行 analyze 分析現有專案")
	runCmd.Flags().BoolVar(&runSkipTest, "skip-test", false, "跳過測試步驟")
	runCmd.Flags().BoolVar(&runSkipReview, "skip-review", false, "跳過審查步驟")
	runCmd.Flags().BoolVar(&runSkipCommit, "skip-commit", false, "跳過提交步驟")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ui.PrintWarning(os.Stdout, "\n收到中斷信號，正在優雅關閉...")
		cancel()
	}()

	w := os.Stdout
	milestoneFile := args[0]

	// Check if milestone file exists
	if _, err := os.Stat(milestoneFile); os.IsNotExist(err) {
		ui.PrintError(w, "Milestone 檔案不存在: "+milestoneFile)
		return nil
	}

	ui.PrintHeader(w, "執行完整 Pipeline")
	ui.PrintInfo(w, "Milestone: "+milestoneFile)
	ui.PrintInfo(w, "")

	results := make(map[string]interface{})
	totalSteps := 5
	currentStep := 0

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

	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return err
	}

	// Step 0: Analyze (optional)
	if runAnalyzeFirst {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps+1, "Analyze - 分析現有專案...")
		
		analyzeAgent := agent.NewAnalyzeAgent(caller, cfg.ProjectRoot)
		scope := agent.AllScopes()
		
		issues, err := analyzeAgent.Analyze(ctx, scope)
		if err != nil {
			ui.PrintWarning(w, "分析失敗: "+err.Error())
		} else if issues.Count() > 0 {
			ui.PrintInfo(w, fmt.Sprintf("  發現 %d 個問題", issues.Count()))
			// Convert to tickets
			ticketList := issues.ToTickets()
			for _, t := range ticketList.Tickets {
				store.Save(t)
			}
			results["analyze"] = map[string]int{"issues": issues.Count()}
		}
		totalSteps++
	}

	// Step 1: Planning
	currentStep++
	ui.PrintStep(w, currentStep, totalSteps, "Planning - 分析 milestone 產生 tickets...")

	planningAgent := agent.NewPlanningAgent(caller, cfg.ProjectRoot, cfg.TicketsDir)
	tickets, err := planningAgent.Plan(ctx, milestoneFile)
	if err != nil {
		ui.PrintError(w, "  規劃失敗: "+err.Error())
		return err
	}

	for _, t := range tickets {
		store.Save(t)
	}
	ui.PrintSuccess(w, fmt.Sprintf("  產生 %d 個 tickets", len(tickets)))
	results["planning"] = map[string]int{"tickets_created": len(tickets)}

	// Check for cancellation
	select {
	case <-ctx.Done():
		ui.PrintWarning(w, "Pipeline 已中斷")
		return nil
	default:
	}

	// Step 2: Coding
	currentStep++
	ui.PrintStep(w, currentStep, totalSteps, "Coding - 處理 tickets...")

	codingAgent := agent.NewCodingAgent(caller, cfg.ProjectRoot)
	resolver := ticket.NewDependencyResolver(store)

	completed := 0
	failed := 0

	maxIterations := 20
	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		processable, _ := resolver.GetProcessable()
		if len(processable) == 0 {
			break
		}

		for _, t := range processable {
			t.MarkInProgress()
			store.Save(t)

			result, err := codingAgent.Execute(ctx, t)
			if err != nil || !result.Success {
				t.MarkFailed(fmt.Errorf("execution failed"))
				failed++
			} else {
				t.MarkCompleted(result.Output)
				completed++
			}
			store.Save(t)
		}
	}

	ui.PrintSuccess(w, fmt.Sprintf("  完成: %d, 失敗: %d", completed, failed))
	results["coding"] = map[string]int{"completed": completed, "failed": failed}

	// Check for cancellation
	select {
	case <-ctx.Done():
		ui.PrintWarning(w, "Pipeline 已中斷")
		return nil
	default:
	}

	// Step 3: Testing
	if !runSkipTest {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, "Testing - 執行測試...")

		testAgent := agent.NewTestAgent(caller, cfg.ProjectRoot)
		testResult, _, err := testAgent.RunTests(ctx)
		if err != nil {
			ui.PrintWarning(w, "  測試失敗: "+err.Error())
			results["testing"] = map[string]bool{"success": false}
		} else {
			ui.PrintSuccess(w, "  測試完成")
			results["testing"] = map[string]bool{"success": testResult.Success}
		}
	}

	// Step 4: Review
	if !runSkipReview {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, "Review - 程式碼審查...")

		files := getGitChangedFiles()
		if len(files) > 0 {
			reviewAgent := agent.NewReviewAgent(caller, cfg.ProjectRoot)
			reviewResult, _, err := reviewAgent.Review(ctx, files)
			if err != nil {
				ui.PrintWarning(w, "  審查失敗: "+err.Error())
				results["review"] = map[string]bool{"success": false}
			} else {
				ui.PrintSuccess(w, "  審查完成")
				results["review"] = map[string]bool{"success": reviewResult.Success}
			}
		} else {
			ui.PrintInfo(w, "  沒有檔案需要審查")
			results["review"] = map[string]bool{"success": true}
		}
	}

	// Step 5: Commit
	if !runSkipCommit {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, "Committing - 提交變更...")

		completedTickets, _ := store.LoadByStatus(ticket.StatusCompleted)
		commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)
		
		commitCount := 0
		for _, t := range completedTickets {
			changes := getGitStatus()
			if changes == "" {
				continue
			}
			
			result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes)
			if err == nil && result.Success {
				commitCount++
			}
		}

		ui.PrintSuccess(w, fmt.Sprintf("  提交 %d 個 commits", commitCount))
		results["committing"] = map[string]int{"commits": commitCount}
	}

	// Summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, "Pipeline 完成!")

	// Print final status
	counts, _ := store.Count()
	statusTable := ui.NewStatusTable()
	statusTable.SetCounts(
		counts[ticket.StatusPending],
		counts[ticket.StatusInProgress],
		counts[ticket.StatusCompleted],
		counts[ticket.StatusFailed],
	)
	statusTable.Render(w)

	return nil
}
