package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	orcherrors "github.com/anthropic/agent-orchestrator/internal/errors"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	analyzeScope   []string
	analyzeAutoGen bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: i18n.CmdAnalyzeShort,
	Long:  i18n.CmdAnalyzeLong,
	RunE:  runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringSliceVar(&analyzeScope, "scope", []string{"all"}, i18n.FlagScope)
	analyzeCmd.Flags().BoolVar(&analyzeAutoGen, "auto", false, i18n.FlagAuto)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	ui.PrintHeader(w, i18n.UIProjectAnalyze)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgAnalyzeProject, cfg.ProjectRoot))
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgAnalyzeScope, strings.Join(analyzeScope, ", ")))

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		return err
	}

	analyzeAgent := agent.NewAnalyzeAgent(caller, cfg.ProjectRoot)
	scope := agent.ParseScopes(analyzeScope)

	// Run analysis
	spinner := ui.NewSpinner(i18n.SpinnerAnalyzing, w)
	spinner.Start()

	issues, err := analyzeAgent.Analyze(ctx, scope)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailAnalysis)
		return err
	}
	spinner.Success(i18n.MsgAnalysisComplete)

	if issues.Count() == 0 {
		ui.PrintSuccess(w, i18n.MsgNoIssuesFound)
		return nil
	}

	// Display issues by category
	ui.PrintHeader(w, i18n.UIAnalysisReport)

	categories := []struct {
		name     string
		category string
	}{
		{i18n.CategoryPerformance, "performance"},
		{i18n.CategoryRefactor, "refactor"},
		{i18n.CategorySecurity, "security"},
		{i18n.CategoryTest, "test"},
		{i18n.CategoryDocs, "docs"},
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
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgFoundIssues, issues.Count()))

	// Ask to generate tickets
	generateTickets := analyzeAutoGen
	if !generateTickets && !cfg.Quiet {
		prompt := ui.NewPrompt(os.Stdin, w)
		var err error
		generateTickets, err = prompt.Confirm(i18n.PromptGenerateTickets, true)
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
		// Store initialization is fatal
		return orcherrors.ErrStoreInit(err)
	}

	for _, t := range ticketList.Tickets {
		if err := store.Save(t); err != nil {
			// Ticket save failure is recoverable - log and continue
			recErr := orcherrors.ErrSaveTicket(t.ID, err)
			ui.PrintWarning(w, recErr.Error())
			continue
		}
		ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgTicketCreated, t.ID, t.Title))
	}

	ui.PrintInfo(w, "")
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgToDirectory, ticketList.Count(), cfg.TicketsDir))
	ui.PrintInfo(w, i18n.HintRunWork)

	return nil
}
