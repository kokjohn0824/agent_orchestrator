package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "設定管理",
	Long: `顯示或管理 agent-orchestrator 設定。

範例:
  agent-orchestrator config           # 顯示目前設定
  agent-orchestrator config init      # 產生預設設定檔
  agent-orchestrator config path      # 顯示設定檔路徑`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "顯示目前設定",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := os.Stdout

		ui.PrintHeader(w, "目前設定")

		// Load config
		cfg, err := config.Load()
		if err != nil {
			ui.PrintError(w, "載入設定失敗: "+err.Error())
			return nil
		}

		table := ui.NewTable("設定項", "值")
		table.AddRow("Agent Command", cfg.AgentCommand)
		table.AddRow("Output Format", cfg.AgentOutputFormat)
		table.AddRow("Force Mode", fmt.Sprintf("%v", cfg.AgentForce))
		table.AddRow("Timeout", fmt.Sprintf("%d 秒", cfg.AgentTimeout))
		table.AddRow("Project Root", cfg.ProjectRoot)
		table.AddRow("Tickets Dir", cfg.TicketsDir)
		table.AddRow("Logs Dir", cfg.LogsDir)
		table.AddRow("Docs Dir", cfg.DocsDir)
		table.AddRow("Max Parallel", fmt.Sprintf("%d", cfg.MaxParallel))
		table.Render(w)

		ui.PrintInfo(w, "")
		ui.PrintInfo(w, "設定檔路徑: "+config.GetConfigFilePath())

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "產生預設設定檔",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := os.Stdout

		path := ".agent-orchestrator.yaml"

		// Check if already exists
		if _, err := os.Stat(path); err == nil {
			ui.PrintWarning(w, "設定檔已存在: "+path)
			prompt := ui.NewPrompt(os.Stdin, w)
			ok, err := prompt.Confirm("要覆蓋嗎？", false)
			if err != nil || !ok {
				return nil
			}
		}

		if err := config.GenerateDefaultConfigFile(path); err != nil {
			ui.PrintError(w, "產生設定檔失敗: "+err.Error())
			return nil
		}

		ui.PrintSuccess(w, "已產生設定檔: "+path)
		ui.PrintInfo(w, "你可以編輯此檔案來自訂設定")

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "顯示設定檔路徑",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.GetConfigFilePath())
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)

	// Default subcommand is show
	configCmd.RunE = configShowCmd.RunE
}
