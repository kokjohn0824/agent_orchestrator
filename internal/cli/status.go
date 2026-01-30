package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: i18n.CmdStatusShort,
	Long:  i18n.CmdStatusLong,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	// status 為僅讀（查詢）指令，不呼叫 ErrIfBackgroundWorkRunning，可與背景 work 並存（TICKET-019）。
	// 僅「會寫入 store」的指令（plan, work, run 等）受並行策略限制。
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
		ui.PrintInfo(w, i18n.MsgNoTickets)
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, i18n.MsgGettingStarted)
		ui.PrintInfo(w, i18n.MsgGettingStartedInit)
		ui.PrintInfo(w, i18n.MsgGettingStartedPlan)
		ui.PrintInfo(w, i18n.MsgGettingStartedAnalyze)
		ui.PrintInfo(w, i18n.MsgGettingStartedAdd)
		return nil
	}

	ui.PrintHeader(w, i18n.UITicketStatus)

	// Status summary table
	statusTable := ui.NewStatusTable()
	statusTable.SetCounts(
		counts[ticket.StatusPending],
		counts[ticket.StatusInProgress],
		counts[ticket.StatusCompleted],
		counts[ticket.StatusFailed],
	)
	statusTable.Render(w)

	// Show background work if PID file exists and process is alive.
	// If PID file exists but process is dead, treat as stale and remove the PID file (do not show as running).
	if cfg != nil {
		pidPath := cfg.WorkPIDFilePath()
		pid, err := ReadWorkPIDFile(pidPath)
		if err == nil {
			if !IsProcessAlive(pid) {
				RemoveWorkPIDFile(pidPath)
			} else {
				ui.PrintInfo(w, "")
				ui.PrintInfo(w, fmt.Sprintf(i18n.MsgBackgroundWorkRunningPid, pid))
				logDir := cfg.LogsDir
				if cfg.WorkDetachLogDir != "" {
					logDir = cfg.WorkDetachLogDir
				}
				ui.PrintInfo(w, ui.StyleMuted.Render(fmt.Sprintf(i18n.MsgLogPath, logDir)))
			}
		}
	}

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
			ui.PrintInfo(w, fmt.Sprintf("  %s %s: %s", priority, t.ID, ui.Truncate(t.Title, 50)))

			// Show dependencies if any
			if len(t.Dependencies) > 0 {
				ui.PrintInfo(w, ui.StyleMuted.Render(fmt.Sprintf(i18n.MsgDependencies, t.Dependencies)))
			}

			// Show full error and log path if failed
			if s.status == ticket.StatusFailed {
				if t.Error != "" {
					// Show full error (up to 200 chars for readability)
					errDisplay := t.Error
					if len(errDisplay) > 200 {
						errDisplay = errDisplay[:200] + "..."
					}
					ui.PrintInfo(w, ui.StyleError.Render(fmt.Sprintf(i18n.MsgErrorDetail, errDisplay)))
				}
				if t.ErrorLog != "" {
					ui.PrintInfo(w, ui.StyleMuted.Render(fmt.Sprintf(i18n.MsgErrorLog, t.ErrorLog)))
				}
			}
		}
	}

	// Show helpful commands
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, ui.StyleMuted.Render(i18n.UICommonCommands))
	if counts[ticket.StatusPending] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  "+i18n.HintRunWorkCmd))
	}
	if counts[ticket.StatusFailed] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  "+i18n.HintRunRetryCmd))
	}
	if counts[ticket.StatusCompleted] > 0 {
		ui.PrintInfo(w, ui.StyleMuted.Render("  "+i18n.HintRunCommitCmd))
	}

	return nil
}
