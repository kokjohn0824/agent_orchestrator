package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

// codeExtensions defines common code file extensions
var codeExtensions = map[string]bool{
	".go":    true,
	".py":    true,
	".js":    true,
	".ts":    true,
	".jsx":   true,
	".tsx":   true,
	".java":  true,
	".c":     true,
	".cpp":   true,
	".h":     true,
	".hpp":   true,
	".rs":    true,
	".rb":    true,
	".php":   true,
	".swift": true,
	".kt":    true,
	".scala": true,
	".cs":    true,
	".vue":   true,
	".svelte": true,
}

// excludeDirs defines directories to skip when scanning
var excludeDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".git":         true,
	".svn":         true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".idea":        true,
	".vscode":      true,
}

// hasExistingCode checks if the directory contains code files
func hasExistingCode(dir string) bool {
	codeFileCount := 0
	maxFilesToCheck := 100 // limit for performance

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip excluded directories
		if d.IsDir() {
			if excludeDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if codeExtensions[ext] {
			codeFileCount++
			if codeFileCount >= 3 { // found enough code files
				return filepath.SkipAll
			}
		}

		maxFilesToCheck--
		if maxFilesToCheck <= 0 {
			return filepath.SkipAll
		}

		return nil
	})

	return codeFileCount >= 3
}

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

	// Check if this is an existing project with code
	var summary *agent.ProjectSummary
	if hasExistingCode(cfg.ProjectRoot) {
		ui.PrintInfo(w, i18n.MsgDetectedExistingProject)

		// Scan project structure
		spinner := ui.NewSpinner(i18n.SpinnerScanningProject, w)
		spinner.Start()

		summary, err = initAgent.ScanProject(ctx)
		if err != nil {
			spinner.Fail("掃描專案失敗")
			// Continue without summary on error
			summary = nil
		} else {
			spinner.Success(i18n.MsgScanComplete)

			// Display project summary
			ui.PrintInfo(w, i18n.MsgProjectSummary)
			fmt.Fprint(w, summary.String())
		}
		ui.PrintInfo(w, "")
	}

	// Generate questions (with or without summary)
	spinner := ui.NewSpinner(i18n.SpinnerGeneratingQuestions, w)
	spinner.Start()

	questions, err := initAgent.GenerateQuestions(ctx, goal, summary)
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

	// Generate milestone (with or without summary)
	ui.PrintInfo(w, "")
	spinner = ui.NewSpinner(i18n.SpinnerGeneratingMilestone, w)
	spinner.Start()

	milestonePath, err := initAgent.GenerateMilestone(ctx, goal, questions, answers, summary)
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
