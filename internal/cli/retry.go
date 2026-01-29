package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "重試失敗的 tickets",
	Long: `將所有失敗的 tickets 移回 pending 狀態，以便重新處理。

範例:
  agent-orchestrator retry
  agent-orchestrator retry && agent-orchestrator work`,
	RunE: runRetry,
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
		ui.PrintInfo(w, "沒有失敗的 tickets 需要重試")
		return nil
	}

	ui.PrintHeader(w, "重試失敗的 Tickets")
	ui.PrintInfo(w, fmt.Sprintf("找到 %d 個失敗的 tickets", len(failed)))

	// Move failed tickets to pending
	count, err := store.MoveFailed()
	if err != nil {
		return err
	}

	ui.PrintSuccess(w, fmt.Sprintf("已將 %d 個 tickets 移回 pending", count))
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, "執行 'agent-orchestrator work' 開始重新處理")

	return nil
}
