package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.CmdConfigShort,
	Long:  i18n.CmdConfigLong,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: i18n.CmdConfigShowShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := os.Stdout

		ui.PrintHeader(w, i18n.UICurrentConfig)

		// Load config
		cfg, err := config.Load()
		if err != nil {
			ui.PrintError(w, fmt.Sprintf(i18n.ErrLoadConfigFailed, err.Error()))
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
		ui.PrintInfo(w, fmt.Sprintf(i18n.MsgConfigFilePath, config.GetConfigFilePath()))

		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: i18n.CmdConfigInitShort,
	RunE: func(cmd *cobra.Command, args []string) error {
		w := os.Stdout

		path := ".agent-orchestrator.yaml"

		// Check if already exists
		if _, err := os.Stat(path); err == nil {
			ui.PrintWarning(w, fmt.Sprintf(i18n.MsgConfigExists, path))
			prompt := ui.NewPrompt(os.Stdin, w)
			ok, err := prompt.Confirm(i18n.PromptOverwrite, false)
			if err != nil || !ok {
				return nil
			}
		}

		if err := config.GenerateDefaultConfigFile(path); err != nil {
			ui.PrintError(w, fmt.Sprintf(i18n.ErrGenerateConfigFailed, err.Error()))
			return nil
		}

		ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgConfigGenerated, path))
		ui.PrintInfo(w, i18n.MsgEditConfigHint)

		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: i18n.CmdConfigPathShort,
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
