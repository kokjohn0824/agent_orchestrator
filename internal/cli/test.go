package cli

import (
	"context"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "執行專案測試",
	Long: `執行專案的測試並分析結果。

範例:
  agent-orchestrator test`,
	RunE: runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	ui.PrintHeader(w, "執行測試")
	ui.PrintInfo(w, "專案目錄: "+cfg.ProjectRoot)

	// Create agent caller
	caller := agent.NewCaller(
		cfg.AgentCommand,
		cfg.AgentForce,
		cfg.AgentOutputFormat,
		cfg.LogsDir,
	)
	caller.SetDryRun(cfg.DryRun)
	caller.SetVerbose(cfg.Verbose)

	if !caller.IsAvailable() && !cfg.DryRun {
		ui.PrintError(w, "找不到 agent 指令，請確保已安裝 Cursor CLI")
		return nil
	}

	testAgent := agent.NewTestAgent(caller, cfg.ProjectRoot)

	// Run tests
	spinner := ui.NewSpinner("執行測試中...", w)
	spinner.Start()

	result, testResult, err := testAgent.RunTests(ctx)
	if err != nil {
		spinner.Fail("測試執行失敗")
		return err
	}

	if result.Success {
		spinner.Success("測試完成")
	} else {
		spinner.Fail("測試有失敗")
	}

	// Print test result summary
	if testResult != nil {
		ui.PrintInfo(w, "")
		if testResult.Passed > 0 || testResult.Failed > 0 {
			ui.PrintInfo(w, "測試結果:")
			ui.PrintSuccess(w, "  通過: "+string(rune('0'+testResult.Passed)))
			if testResult.Failed > 0 {
				ui.PrintError(w, "  失敗: "+string(rune('0'+testResult.Failed)))
			}
			if testResult.Skipped > 0 {
				ui.PrintWarning(w, "  跳過: "+string(rune('0'+testResult.Skipped)))
			}
		}
		if testResult.Summary != "" {
			ui.PrintInfo(w, "")
			ui.PrintInfo(w, "摘要: "+testResult.Summary)
		}
	}

	// Print full output if verbose
	if cfg.Verbose && result != nil {
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, "完整輸出:")
		ui.PrintInfo(w, result.Output)
	}

	return nil
}
