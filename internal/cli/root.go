// Package cli provides the command-line interface
package cli

import (
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Version information (set at build time)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// Global flags
	cfgFile     string
	dryRun      bool
	verbose     bool
	debug       bool
	quiet       bool
	outputFormat string

	// Global config
	cfg *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "agent-orchestrator",
	Short: "協調多個 Cursor Agent 的 CLI 工具",
	Long: `Agent Orchestrator - 使用 Cursor Agent (Headless Mode) 作為 Subagents

這個工具可以幫助你：
  • 透過互動式問答初始化專案規劃 (init)
  • 分析現有專案並產生改進建議 (analyze)
  • 將 milestone 分解為可執行的 tickets (plan)
  • 自動執行 coding、review、test、commit 等任務

參考文件: https://cursor.com/docs/cli/headless`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for some commands
		if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("載入設定失敗: %w", err)
		}

		// Override with flags
		if dryRun {
			cfg.DryRun = true
		}
		if verbose {
			cfg.Verbose = true
		}
		if debug {
			cfg.Debug = true
			cfg.Verbose = true // debug implies verbose
		}
		if quiet {
			cfg.Quiet = true
			cfg.Verbose = false
		}
		if outputFormat != "" {
			cfg.AgentOutputFormat = outputFormat
		}

		return cfg.Validate()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "設定檔路徑 (預設: .agent-orchestrator.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "不實際執行 agent，只顯示會做什麼")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細輸出")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "除錯模式")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "安靜模式，只顯示錯誤")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "Agent 輸出格式: text, json, stream-json")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(retryCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(configCmd)
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "顯示版本資訊",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Agent Orchestrator %s\n", Version)
		fmt.Printf("  Commit: %s\n", Commit)
		fmt.Printf("  Built:  %s\n", BuildDate)
	},
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}
