package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan <milestone-file>",
	Short: i18n.CmdPlanShort,
	Long:  i18n.CmdPlanLong,
	Args:  cobra.ExactArgs(1),
	RunE:  runPlan,
}

func runPlan(cmd *cobra.Command, args []string) error {
	return runPlanWithFile(context.Background(), args[0])
}

func runPlanWithFile(ctx context.Context, milestoneFile string) error {
	w := os.Stdout

	// Check if milestone file exists
	if _, err := os.Stat(milestoneFile); os.IsNotExist(err) {
		ui.PrintError(w, fmt.Sprintf(i18n.ErrMilestoneNotFound, milestoneFile))
		return nil
	}

	ui.PrintHeader(w, i18n.UIPlanning)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgAnalyzeMilestone, milestoneFile))

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	planningAgent := agent.NewPlanningAgent(caller, cfg.ProjectRoot, cfg.TicketsDir)

	// Run planning
	spinner := ui.NewSpinner(i18n.SpinnerPlanning, w)
	spinner.Start()

	tickets, err := planningAgent.Plan(ctx, milestoneFile)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailPlanning)
		return err
	}
	spinner.Success(i18n.MsgPlanningComplete)

	if len(tickets) == 0 {
		ui.PrintWarning(w, i18n.MsgNoTicketsGenerated)
		return nil
	}

	// Initialize store and save tickets
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf(i18n.ErrInitStoreFailed, err)
	}

	// Validate dependencies
	resolver := ticket.NewDependencyResolver(store)
	if err := resolver.ValidateDependencies(tickets); err != nil {
		ui.PrintWarning(w, fmt.Sprintf(i18n.MsgDependencyWarning, err.Error()))
	}

	// Check for circular dependencies
	if resolver.HasCircularDependency(tickets) {
		ui.PrintWarning(w, i18n.MsgCircularDependency)
	}

	// Save tickets
	for _, t := range tickets {
		if err := store.Save(t); err != nil {
			ui.PrintError(w, fmt.Sprintf(i18n.ErrSaveTicketFailed, t.ID))
			continue
		}
	}

	// Display results
	ui.PrintInfo(w, "")
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgGeneratedTickets, len(tickets)))
	ui.PrintInfo(w, "")

	// Show ticket list
	table := ui.NewTable("Priority", "ID", "Title", "Type", "Complexity")
	for _, t := range tickets {
		priority := ui.PriorityStyle(t.Priority).Render(fmt.Sprintf("P%d", t.Priority))
		table.AddRow(priority, t.ID, ui.Truncate(t.Title, 40), string(t.Type), t.EstimatedComplexity)
	}
	table.Render(w)

	ui.PrintInfo(w, "")
	ui.PrintInfo(w, i18n.HintRunWork)
	ui.PrintInfo(w, i18n.HintRunStatus)

	return nil
}
