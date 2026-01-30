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
	cleanForce bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: i18n.CmdCleanShort,
	Long:  i18n.CmdCleanLong,
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, i18n.FlagForce)
}

func runClean(cmd *cobra.Command, args []string) error {
	w := os.Stdout

	store := ticket.NewStore(cfg.TicketsDir)

	// Get current counts
	counts, err := store.Count()
	if err != nil {
		// Directory might not exist, that's ok
		counts = make(map[ticket.Status]int)
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	if total == 0 {
		ui.PrintInfo(w, i18n.MsgNoDataToClean)
		return nil
	}

	ui.PrintHeader(w, i18n.UICleanData)
	ui.PrintWarning(w, i18n.MsgAboutToDelete)
	ui.PrintInfo(w, "  - "+i18n.MsgTicketsDir+cfg.TicketsDir)
	ui.PrintInfo(w, "  - "+i18n.MsgLogsDir+cfg.LogsDir)
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, i18n.MsgCurrentStatus)

	for status, count := range counts {
		if count > 0 {
			ui.PrintInfo(w, fmt.Sprintf("  - %s: %d", status, count))
		}
	}

	// Confirm
	if !cleanForce {
		prompt := ui.NewPrompt(os.Stdin, w)
		ok, err := prompt.Confirm(i18n.PromptConfirmClean, false)
		if err != nil {
			return err
		}
		if !ok {
			ui.PrintInfo(w, i18n.MsgCancelled)
			return nil
		}
	}

	// Clean tickets
	if err := store.Clean(); err != nil {
		ui.PrintError(w, i18n.ErrCleanTicketsFailed+err.Error())
	}

	// Clean logs
	if err := os.RemoveAll(cfg.LogsDir); err != nil {
		ui.PrintError(w, i18n.ErrCleanLogsFailed+err.Error())
	}

	// Re-init store
	store.Init()

	ui.PrintSuccess(w, i18n.MsgDataCleared)

	return nil
}
