package cli

import (
	"context"
	"os"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [goal]",
	Short: "互動式專案初始化，產生 milestone",
	Long: `透過一系列問題來了解專案需求，然後產生對應的 milestone 文件。

範例:
  agent-orchestrator init "建立一個 Log 分析工具，使用 Drain 演算法"
  agent-orchestrator init  # 互動模式輸入目標`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	// Get goal from args or prompt
	var goal string
	if len(args) > 0 {
		goal = args[0]
	} else {
		prompt := ui.NewPrompt(os.Stdin, w)
		var err error
		goal, err = prompt.Ask("請描述你的專案目標")
		if err != nil {
			return err
		}
	}

	ui.PrintHeader(w, "專案初始化")
	ui.PrintInfo(w, "專案目標: "+goal)

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

	initAgent := agent.NewInitAgent(caller, cfg.ProjectRoot, cfg.DocsDir)

	// Generate questions
	ui.PrintInfo(w, "讓我了解更多細節...")
	spinner := ui.NewSpinner("產生問題中...", w)
	spinner.Start()

	questions, err := initAgent.GenerateQuestions(ctx, goal)
	if err != nil {
		spinner.Fail("產生問題失敗")
		return err
	}
	spinner.Success("已產生問題")

	// Ask questions
	prompt := ui.NewPrompt(os.Stdin, w)
	answers := make([]string, 0, len(questions))

	for i, q := range questions {
		ui.PrintInfo(w, "")
		ui.PrintStep(w, i+1, len(questions), q)
		answer, err := prompt.Ask("")
		if err != nil {
			return err
		}
		answers = append(answers, answer)
	}

	// Generate milestone
	ui.PrintInfo(w, "")
	spinner = ui.NewSpinner("產生 milestone 文件中...", w)
	spinner.Start()

	milestonePath, err := initAgent.GenerateMilestone(ctx, goal, questions, answers)
	if err != nil {
		spinner.Fail("產生 milestone 失敗")
		return err
	}
	spinner.Success("已產生 milestone")

	ui.PrintSuccess(w, "已產生 milestone: "+milestonePath)

	// Ask if user wants to continue to plan
	continueOk, err := prompt.Confirm("要立即執行 plan 產生 tickets 嗎？", true)
	if err != nil {
		return err
	}

	if continueOk {
		return runPlanWithFile(ctx, milestonePath)
	}

	ui.PrintInfo(w, "你可以稍後執行: agent-orchestrator plan "+milestonePath)
	return nil
}
