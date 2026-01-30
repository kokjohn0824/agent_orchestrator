package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	orcherrors "github.com/anthropic/agent-orchestrator/internal/errors"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

// Note: agent import is still needed for NewPlanningAgent, NewCodingAgent, etc.

var (
	runAnalyzeFirst bool
	runSkipTest     bool
	runSkipReview   bool
	runSkipCommit   bool
)

var runCmd = &cobra.Command{
	Use:   "run <milestone-file>",
	Short: i18n.CmdRunShort,
	Long:  i18n.CmdRunLong,
	Args:  cobra.ExactArgs(1),
	RunE:  runPipeline,
}

func init() {
	runCmd.Flags().BoolVar(&runAnalyzeFirst, "analyze-first", false, i18n.FlagAnalyzeFirst)
	runCmd.Flags().BoolVar(&runSkipTest, "skip-test", false, i18n.FlagSkipTest)
	runCmd.Flags().BoolVar(&runSkipReview, "skip-review", false, i18n.FlagSkipReview)
	runCmd.Flags().BoolVar(&runSkipCommit, "skip-commit", false, i18n.FlagSkipCommit)
}

func runPipeline(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ui.PrintWarning(os.Stdout, i18n.MsgInterruptSignal)
		cancel()
	}()

	w := os.Stdout
	milestoneFile := args[0]

	// Check if milestone file exists
	if _, err := os.Stat(milestoneFile); os.IsNotExist(err) {
		return orcherrors.ErrFileNotFound(milestoneFile)
	}

	ui.PrintHeader(w, i18n.UIFullPipeline)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgMilestone, milestoneFile))
	ui.PrintInfo(w, "")

	results := make(map[string]interface{})
	totalSteps := 5
	currentStep := 0

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		return err
	}

	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return orcherrors.ErrStoreInit(err)
	}

	// Step 0: Analyze (optional)
	if runAnalyzeFirst {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps+1, i18n.StepAnalyze)

		analyzeAgent := agent.NewAnalyzeAgent(caller, cfg.ProjectRoot)
		scope := agent.AllScopes()

		issues, err := analyzeAgent.Analyze(ctx, scope)
		if err != nil {
			// Analysis failure is recoverable - log and continue
			recErr := orcherrors.ErrAnalysis(err)
			ui.PrintWarning(w, recErr.Error())
		} else if issues.Count() > 0 {
			ui.PrintInfo(w, fmt.Sprintf(i18n.MsgFoundIssues, issues.Count()))
			// Convert to tickets
			ticketList := issues.ToTickets()
			for _, t := range ticketList.Tickets {
				if err := store.Save(t); err != nil {
					// Ticket save failure is recoverable - log and continue
					recErr := orcherrors.ErrSaveTicket(t.ID, err)
					ui.PrintWarning(w, recErr.Error())
				}
			}
			results["analyze"] = map[string]int{"issues": issues.Count()}
		}
		totalSteps++
	}

	// Step 1: Planning
	currentStep++
	ui.PrintStep(w, currentStep, totalSteps, i18n.StepPlanning)

	planningAgent := agent.NewPlanningAgent(caller, cfg.ProjectRoot, cfg.TicketsDir)
	tickets, err := planningAgent.Plan(ctx, milestoneFile)
	if err != nil {
		// Planning failure is fatal - must return error
		return orcherrors.ErrPlanning(err)
	}

	for _, t := range tickets {
		if err := store.Save(t); err != nil {
			// Ticket save failure is recoverable - log and continue
			recErr := orcherrors.ErrSaveTicket(t.ID, err)
			ui.PrintWarning(w, recErr.Error())
		}
	}
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgGeneratedTickets, len(tickets)))
	results["planning"] = map[string]int{"tickets_created": len(tickets)}

	// Check for cancellation
	select {
	case <-ctx.Done():
		ui.PrintWarning(w, i18n.MsgPipelineInterrupted)
		return nil
	default:
	}

	// Step 2: Coding
	currentStep++
	ui.PrintStep(w, currentStep, totalSteps, i18n.StepCoding)

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
			if err := store.Save(t); err != nil {
				recErr := orcherrors.ErrSaveTicket(t.ID, err)
				ui.PrintWarning(w, recErr.Error())
			}

			result, err := codingAgent.Execute(ctx, t)
			if err != nil || !result.Success {
				t.MarkFailed(fmt.Errorf("execution failed"))
				failed++
			} else {
				t.MarkCompleted(result.Output)
				completed++
			}
			if err := store.Save(t); err != nil {
				recErr := orcherrors.ErrSaveTicket(t.ID, err)
				ui.PrintWarning(w, recErr.Error())
			}
		}
	}

	ui.PrintSuccess(w, fmt.Sprintf("  "+i18n.MsgCountCompleted+", "+i18n.MsgCountFailed, completed, failed))
	results["coding"] = map[string]int{"completed": completed, "failed": failed}

	// Check for cancellation
	select {
	case <-ctx.Done():
		ui.PrintWarning(w, i18n.MsgPipelineInterrupted)
		return nil
	default:
	}

	// Step 3: Testing
	if !runSkipTest {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, i18n.StepTesting)

		testAgent := agent.NewTestAgent(caller, cfg.ProjectRoot)
		testResult, _, err := testAgent.RunTests(ctx)
		if err != nil {
			// Test failure is recoverable - log and continue
			recErr := orcherrors.ErrTest(err)
			ui.PrintWarning(w, recErr.Error())
			results["testing"] = map[string]bool{"success": false}
		} else {
			ui.PrintSuccess(w, "  "+i18n.MsgTestComplete)
			results["testing"] = map[string]bool{"success": testResult.Success}
		}
	}

	// Step 4: Review
	if !runSkipReview {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, i18n.StepReview)

		files := getGitChangedFiles(ctx)
		if len(files) > 0 {
			reviewAgent := agent.NewReviewAgent(caller, cfg.ProjectRoot)
			result, reviewResult, err := reviewAgent.Review(ctx, files)
			if err != nil {
				// Review failure is recoverable - log and continue
				recErr := orcherrors.ErrReview(err)
				ui.PrintWarning(w, recErr.Error())
				results["review"] = map[string]bool{"success": false}
			} else {
				ui.PrintSuccess(w, "  "+i18n.MsgReviewComplete)
				results["review"] = map[string]bool{"success": result.Success}
				// 若需區分審查通過/需修改，可依 reviewResult.Status 使用
				_ = reviewResult
			}
		} else {
			ui.PrintInfo(w, "  "+i18n.MsgNoFilesToReview)
			results["review"] = map[string]bool{"success": true}
		}
	}

	// Step 5: Commit
	if !runSkipCommit {
		currentStep++
		ui.PrintStep(w, currentStep, totalSteps, i18n.StepCommitting)

		completedTickets, _ := store.LoadByStatus(ticket.StatusCompleted)
		commitAgent := agent.NewCommitAgent(caller, cfg.ProjectRoot)

		commitCount := 0
		for _, t := range completedTickets {
			changedFiles := getGitChangedFiles(ctx)
			if len(changedFiles) == 0 {
				break
			}
			filesToStage := filesForTicket(t, changedFiles)
			if filesToStage == nil {
				filesToStage = changedFiles
			}
			if len(filesToStage) == 0 {
				continue
			}
			changes := getGitStatusForFiles(ctx, filesToStage)
			if changes == "" {
				continue
			}
			result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes, filesToStage)
			if err == nil && result.Success {
				commitCount++
			}
		}

		ui.PrintSuccess(w, fmt.Sprintf("  "+i18n.MsgCommitCount, commitCount))
		results["committing"] = map[string]int{"commits": commitCount}
	}

	// Summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, i18n.UIPipelineComplete)

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
