package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "顯示 tickets 狀態",
	Long: `顯示所有 tickets 的狀態統計和列表。

範例:
  agent-orchestrator status`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	w := os.Stdout

	store := ticket.NewStore(cfg.TicketsDir)

	// Get counts
	counts, err := store.Count()
	if err != nil {
		return err
	}

	total := 0
	for _, c := range counts {
		total += c
	}

	if total == 0 {
		ui.PrintInfo(w, "沒有任何 tickets")
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, "使用以下指令開始:")
		ui.PrintInfo(w, "  agent-orchestrator init \"專案目標\"   # 互動式初始化")
		ui.PrintInfo(w, "  agent-orchestrator plan <milestone>  # 從 milestone 產生 tickets")
		ui.PrintInfo(w, "  agent-orchestrator analyze           # 分析現有專案")
		return nil
	}

	ui.PrintHeader(w, "Tickets 狀態")

	// Status summary table
	statusTable := ui.NewStatusTable()
	statusTable.SetCounts(
		counts[ticket.StatusPending],
		counts[ticket.StatusInProgress],
		counts[ticket.StatusCompleted],
		counts[ticket.StatusFailed],
	)
	statusTable.Render(w)

	// List tickets by status
	statuses := []struct {
		status ticket.Status
		name   string
		style  func(...string) string
	}{
		{ticket.StatusPending, "Pending", ui.StyleWarning.Render},
		{ticket.StatusInProgress, "In Progress", ui.StyleInfo.Render},
		{ticket.StatusCompleted, "Completed", ui.StyleSuccess.Render},
		{ticket.StatusFailed, "Failed", ui.StyleError.Render},
	}

	for _, s := range statuses {
		tickets, err := store.LoadByStatus(s.status)
		if err != nil {
			continue
		}
		if len(tickets) == 0 {
			continue
		}

		ui.PrintInfo(w, "")
		ui.PrintInfo(w, s.style(fmt.Sprintf("%s (%d):", s.name, len(tickets))))

		for _, t := range tickets {
			priority := ui.PriorityStyle(t.Priority).Render(fmt.Sprintf("P%d", t.Priority))
			ui.PrintInfo(w, fmt.Sprintf("  %s %s: %s", priority, t.ID, truncateTitle(t.Title, 50)))
			
			// Show dependencies if any
			if len(t.Dependencies) > 0 {
				ui.PrintInfo(w, ui.StyleMuted.Render(fmt.Sprintf("      依賴: %v", t.Dependencies)))
			}
			
			// Show error if failed
			if s.status == ticket.StatusFailed && t.Error != "" {
				ui.PrintInfo(w, ui.StyleError.Render(fmt.Sprintf("      錯誤: %s", truncateTitle(t.Error, 60))))
			}
		}
	}

	// Show helpful commands
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, ui.StyleMuted.Render("常用指令:"))
	if counts[ticket.StatusPending] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  agent-orchestrator work        # 處理 pending tickets"))
	}
	if counts[ticket.StatusFailed] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  agent-orchestrator retry       # 重試失敗的 tickets"))
	}
	if counts[ticket.StatusCompleted] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  agent-orchestrator commit --all  # 提交所有完成的 tickets"))
	}

	return nil
}
