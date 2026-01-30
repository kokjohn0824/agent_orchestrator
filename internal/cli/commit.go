package cli

import (
	"context"
	"fmt"
	"os"
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

// filesForTicket returns the subset of changedFiles that belong to this ticket
// (intersection with FilesToModify and FilesToCreate). If the ticket has no
// plan files, returns nil so callers can fall back to full changes.
func filesForTicket(t *ticket.Ticket, changedFiles []string) []string {
	planSet := make(map[string]struct{})
	for _, p := range t.FilesToModify {
		planSet[p] = struct{}{}
	}
	for _, p := range t.FilesToCreate {
		planSet[p] = struct{}{}
	}
	if len(planSet) == 0 {
		return nil
	}
	var out []string
	for _, f := range changedFiles {
		if _, ok := planSet[f]; ok {
			out = append(out, f)
		}
	}
	return out
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

	changedFiles := getGitChangedFiles(ctx)
	if len(changedFiles) == 0 {
		ui.PrintInfo(w, i18n.MsgNoChangesToCommit)
		return nil
	}

	filesToStage := filesForTicket(t, changedFiles)
	if filesToStage == nil {
		filesToStage = changedFiles
	}
	if len(filesToStage) == 0 {
		ui.PrintInfo(w, i18n.MsgNoChangesToCommit)
		return nil
	}

	changes := getGitStatusForFiles(ctx, filesToStage)
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

	result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes, filesToStage)
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

		changedFiles := getGitChangedFiles(ctx)
		if len(changedFiles) == 0 {
			ui.PrintInfo(w, "  "+i18n.MsgSkipNoChanges)
			skipped++
			continue
		}

		filesToStage := filesForTicket(t, changedFiles)
		if filesToStage == nil {
			filesToStage = changedFiles
		}
		if len(filesToStage) == 0 {
			ui.PrintInfo(w, "  "+i18n.MsgSkipNoChanges)
			skipped++
			continue
		}

		changes := getGitStatusForFiles(ctx, filesToStage)
		if changes == "" {
			ui.PrintInfo(w, "  "+i18n.MsgSkipNoChanges)
			skipped++
			continue
		}

		result, err := commitAgent.Commit(ctx, t.ID, t.Title, changes, filesToStage)
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
