package cli

import (
	"os"

	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanForce bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "清除所有 tickets 和 logs",
	Long: `清除所有 tickets 和 agent 執行日誌。

範例:
  agent-orchestrator clean
  agent-orchestrator clean --force  # 不詢問直接清除`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "不詢問直接清除")
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
		ui.PrintInfo(w, "沒有資料需要清除")
		return nil
	}

	ui.PrintHeader(w, "清除資料")
	ui.PrintWarning(w, "即將刪除以下資料:")
	ui.PrintInfo(w, "  - Tickets 目錄: "+cfg.TicketsDir)
	ui.PrintInfo(w, "  - Logs 目錄: "+cfg.LogsDir)
	ui.PrintInfo(w, "")
	ui.PrintInfo(w, "目前狀態:")

	for status, count := range counts {
		if count > 0 {
			ui.PrintInfo(w, "  - "+string(status)+": "+string(rune('0'+count)))
		}
	}

	// Confirm
	if !cleanForce {
		prompt := ui.NewPrompt(os.Stdin, w)
		ok, err := prompt.Confirm("確定要清除所有資料嗎？", false)
		if err != nil {
			return err
		}
		if !ok {
			ui.PrintInfo(w, "已取消")
			return nil
		}
	}

	// Clean tickets
	if err := store.Clean(); err != nil {
		ui.PrintError(w, "清除 tickets 失敗: "+err.Error())
	}

	// Clean logs
	if err := os.RemoveAll(cfg.LogsDir); err != nil {
		ui.PrintError(w, "清除 logs 失敗: "+err.Error())
	}

	// Re-init store
	store.Init()

	ui.PrintSuccess(w, "已清除所有資料")

	return nil
}
