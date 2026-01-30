package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	dropForce bool
)

var dropCmd = &cobra.Command{
	Use:   "drop <ticket-id>",
	Short: i18n.CmdDropShort,
	Long:  i18n.CmdDropLong,
	Args:  cobra.ExactArgs(1),
	RunE:  runDrop,
}

func init() {
	dropCmd.Flags().BoolVar(&dropForce, "force", false, i18n.FlagForce)
}

func runDrop(cmd *cobra.Command, args []string) error {
	w := os.Stdout
	ticketID := args[0]

	ui.PrintHeader(w, i18n.UIDropTicket)

	// Initialize store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf(i18n.ErrInitStoreFailed, err)
	}

	// Load existing ticket to show info
	t, err := store.Load(ticketID)
	if err != nil {
		return fmt.Errorf(i18n.ErrTicketNotFound, ticketID)
	}

	// Show ticket info
	ui.PrintInfo(w, "即將刪除的 Ticket:")
	ui.PrintInfo(w, fmt.Sprintf("  ID: %s", t.ID))
	ui.PrintInfo(w, fmt.Sprintf("  標題: %s", t.Title))
	ui.PrintInfo(w, fmt.Sprintf("  類型: %s", t.Type))
	ui.PrintInfo(w, fmt.Sprintf("  狀態: %s", t.Status))
	ui.PrintInfo(w, "")

	// Confirm deletion unless force flag is set
	if !dropForce {
		prompt := ui.NewPrompt(os.Stdin, w)
		confirmed, err := prompt.Confirm(fmt.Sprintf(i18n.PromptConfirmDrop, ticketID), false)
		if err != nil {
			return err
		}
		if !confirmed {
			ui.PrintInfo(w, i18n.MsgCancelled)
			return nil
		}
	}

	// Delete the ticket
	if err := store.Delete(ticketID); err != nil {
		return fmt.Errorf("%s: %w", i18n.ErrDeleteTicketFailed, err)
	}

	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgTicketDropped, ticketID))

	return nil
}
