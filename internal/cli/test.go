package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: i18n.CmdTestShort,
	Long:  i18n.CmdTestLong,
	RunE:  runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	ui.PrintHeader(w, i18n.UIRunTests)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgProjectDir, cfg.ProjectRoot))

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	testAgent := agent.NewTestAgent(caller, cfg.ProjectRoot)

	// Run tests
	spinner := ui.NewSpinner(i18n.SpinnerTesting, w)
	spinner.Start()

	result, testResult, err := testAgent.RunTests(ctx)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailTest)
		return err
	}

	if result.Success {
		spinner.Success(i18n.MsgTestComplete)
	} else {
		spinner.Fail(i18n.SpinnerFailTestHas)
	}

	// Print test result summary
	if testResult != nil {
		ui.PrintInfo(w, "")
		if testResult.Passed > 0 || testResult.Failed > 0 {
			ui.PrintInfo(w, i18n.MsgTestResult)
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
			ui.PrintInfo(w, fmt.Sprintf(i18n.MsgSummary, testResult.Summary))
		}
	}

	// Print full output if verbose
	if cfg.Verbose && result != nil {
		ui.PrintInfo(w, "")
		ui.PrintInfo(w, i18n.MsgFullOutput)
		ui.PrintInfo(w, result.Output)
	}

	return nil
}
