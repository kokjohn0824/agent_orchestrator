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

var initCmd = &cobra.Command{
	Use:   "init [goal]",
	Short: i18n.CmdInitShort,
	Long:  i18n.CmdInitLong,
	RunE:  runInit,
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
		goal, err = prompt.Ask(i18n.PromptProjectGoal)
		if err != nil {
			return err
		}
	}

	ui.PrintHeader(w, i18n.UIProjectInit)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgProjectGoal, goal))

	// Create agent caller
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentNotFound)
		return nil
	}

	initAgent := agent.NewInitAgent(caller, cfg.ProjectRoot, cfg.DocsDir)

	// Generate questions
	spinner := ui.NewSpinner(i18n.SpinnerGeneratingQuestions, w)
	spinner.Start()

	questions, err := initAgent.GenerateQuestions(ctx, goal)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailQuestions)
		return err
	}
	spinner.Success(i18n.MsgQuestionsGenerated)

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
	spinner = ui.NewSpinner(i18n.SpinnerGeneratingMilestone, w)
	spinner.Start()

	milestonePath, err := initAgent.GenerateMilestone(ctx, goal, questions, answers)
	if err != nil {
		spinner.Fail(i18n.SpinnerFailMilestone)
		return err
	}
	spinner.Success(i18n.MsgMilestoneGenerated)

	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgMilestoneCreated, milestonePath))

	// Ask if user wants to continue to plan
	continueOk, err := prompt.Confirm(i18n.PromptContinuePlan, true)
	if err != nil {
		return err
	}

	if continueOk {
		return runPlanWithFile(ctx, milestonePath)
	}

	ui.PrintInfo(w, fmt.Sprintf(i18n.HintRunPlanLater, milestonePath))
	return nil
}
