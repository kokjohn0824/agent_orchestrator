package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: i18n.CmdRetryShort,
	Long:  i18n.CmdRetryLong,
	RunE:  runRetry,
}

func runRetry(cmd *cobra.Command, args []string) error {
	w := os.Stdout

	store := ticket.NewStore(cfg.TicketsDir)

	// Get failed tickets
	failed, err := store.LoadByStatus(ticket.StatusFailed)
	if err != nil {
		return err
	}

	if len(failed) == 0 {
		ui.PrintInfo(w, i18n.MsgNoFailedToRetry)
		return nil
	}

	ui.PrintHeader(w, i18n.UIRetryFailed)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgFoundFailedTickets, len(failed)))

	// Move failed tickets to pending
	count, err := store.MoveFailed()
	if err != nil {
		return err
	}

	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgMovedToPending, count))
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, i18n.HintRunWork)

	return nil
}
