package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	addTitle       string
	addType        string
	addPriority    int
	addDescription string
	addDeps        string
	addCriteria    string
	addEnhance     bool
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: i18n.CmdAddShort,
	Long:  i18n.CmdAddLong,
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", i18n.FlagTitle)
	addCmd.Flags().StringVar(&addType, "type", "feature", i18n.FlagType)
	addCmd.Flags().IntVar(&addPriority, "priority", 3, i18n.FlagPriority)
	addCmd.Flags().StringVar(&addDescription, "description", "", i18n.FlagDescription)
	addCmd.Flags().StringVar(&addDeps, "deps", "", i18n.FlagDeps)
	addCmd.Flags().StringVar(&addCriteria, "criteria", "", i18n.FlagCriteria)
	addCmd.Flags().BoolVar(&addEnhance, "enhance", false, i18n.FlagEnhance)
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout

	ui.PrintHeader(w, i18n.UIAddTicket)

	// Initialize store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf(i18n.ErrInitStoreFailed, err)
	}

	var t *ticket.Ticket
	var err error

	// Check if interactive mode (no title provided)
	if addTitle == "" {
		t, err = collectTicketInteractive(w)
		if err != nil {
			return err
		}
	} else {
		t, err = createTicketFromFlags()
		if err != nil {
			return err
		}
	}

	// AI enhancement if requested
	if addEnhance {
		t, err = enhanceTicket(ctx, w, t)
		if err != nil {
			ui.PrintWarning(w, fmt.Sprintf("AI 預處理失敗: %s，使用原始內容", err.Error()))
		}
	}

	// Validate
	if err := t.Validate(); err != nil {
		ui.PrintError(w, fmt.Sprintf("Ticket 驗證失敗: %s", err.Error()))
		return nil
	}

	// Save
	if err := store.Save(t); err != nil {
		ui.PrintError(w, fmt.Sprintf(i18n.ErrSaveTicketFailed, t.ID))
		return nil
	}

	// Display result
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgTicketAdded, t.ID))
	ui.PrintInfo(w, "")
	displayTicketDetails(w, t)

	return nil
}

func collectTicketInteractive(w *os.File) (*ticket.Ticket, error) {
	prompt := ui.NewPrompt(os.Stdin, w)

	// Title
	title, err := prompt.Ask(i18n.PromptTicketTitle)
	if err != nil {
		return nil, err
	}
	if title == "" {
		ui.PrintError(w, "標題不能為空")
		return nil, fmt.Errorf("empty title")
	}

	// Description
	descLines, err := prompt.AskMultiline(i18n.PromptTicketDesc)
	if err != nil {
		return nil, err
	}
	description := strings.Join(descLines, "\n")

	// Type
	typeOptions := []string{
		"feature - 新功能",
		"bugfix - 錯誤修復",
		"refactor - 重構",
		"test - 測試",
		"docs - 文件",
		"performance - 效能優化",
		"security - 安全性",
	}
	typeIdx, err := prompt.Select(i18n.PromptTicketType, typeOptions)
	if err != nil {
		return nil, err
	}
	ticketTypes := []ticket.Type{
		ticket.TypeFeature,
		ticket.TypeBugfix,
		ticket.TypeRefactor,
		ticket.TypeTest,
		ticket.TypeDocs,
		ticket.TypePerf,
		ticket.TypeSecurity,
	}
	selectedType := ticketTypes[typeIdx]

	// Priority
	priorityStr, err := prompt.Ask(i18n.PromptTicketPriority)
	if err != nil {
		return nil, err
	}
	priority := 3
	if priorityStr != "" {
		fmt.Sscanf(priorityStr, "%d", &priority)
		if priority < 1 || priority > 5 {
			priority = 3
		}
	}

	// Dependencies
	depsStr, err := prompt.Ask(i18n.PromptTicketDeps)
	if err != nil {
		return nil, err
	}
	var deps []string
	if depsStr != "" {
		for _, d := range strings.Split(depsStr, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				deps = append(deps, d)
			}
		}
	}

	// Acceptance Criteria
	criteriaLines, err := prompt.AskMultiline(i18n.PromptTicketCriteria)
	if err != nil {
		return nil, err
	}

	// Generate ID
	id := generateTicketID()

	t := ticket.NewTicket(id, title, description)
	t.Type = selectedType
	t.Priority = priority
	t.Dependencies = deps
	t.AcceptanceCriteria = criteriaLines

	return t, nil
}

func createTicketFromFlags() (*ticket.Ticket, error) {
	id := generateTicketID()

	t := ticket.NewTicket(id, addTitle, addDescription)

	// Parse type
	switch strings.ToLower(addType) {
	case "feature":
		t.Type = ticket.TypeFeature
	case "bugfix":
		t.Type = ticket.TypeBugfix
	case "refactor":
		t.Type = ticket.TypeRefactor
	case "test":
		t.Type = ticket.TypeTest
	case "docs":
		t.Type = ticket.TypeDocs
	case "performance", "perf":
		t.Type = ticket.TypePerf
	case "security":
		t.Type = ticket.TypeSecurity
	default:
		t.Type = ticket.TypeFeature
	}

	// Priority
	if addPriority >= 1 && addPriority <= 5 {
		t.Priority = addPriority
	}

	// Dependencies
	if addDeps != "" {
		for _, d := range strings.Split(addDeps, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				t.Dependencies = append(t.Dependencies, d)
			}
		}
	}

	// Acceptance Criteria
	if addCriteria != "" {
		for _, c := range strings.Split(addCriteria, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				t.AcceptanceCriteria = append(t.AcceptanceCriteria, c)
			}
		}
	}

	return t, nil
}

func generateTicketID() string {
	return fmt.Sprintf("TICKET-%d", time.Now().UnixNano()/1000000)
}

func enhanceTicket(ctx context.Context, w *os.File, t *ticket.Ticket) (*ticket.Ticket, error) {
	caller, err := CreateAgentCaller()
	if err != nil {
		return t, err
	}

	enhancer := agent.NewEnhanceAgent(caller, cfg.ProjectRoot)

	spinner := ui.NewSpinner(i18n.SpinnerEnhancing, w)
	spinner.Start()

	enhanced, err := enhancer.Enhance(ctx, t)
	if err != nil {
		spinner.Fail("AI 預處理失敗")
		return t, err
	}

	spinner.Success(i18n.MsgEnhanceComplete)
	return enhanced, nil
}

func displayTicketDetails(w *os.File, t *ticket.Ticket) {
	ui.PrintInfo(w, fmt.Sprintf("ID: %s", t.ID))
	ui.PrintInfo(w, fmt.Sprintf("標題: %s", t.Title))
	ui.PrintInfo(w, fmt.Sprintf("類型: %s", t.Type))
	ui.PrintInfo(w, fmt.Sprintf("優先級: P%d", t.Priority))
	ui.PrintInfo(w, fmt.Sprintf("狀態: %s", t.Status))

	if t.Description != "" {
		ui.PrintInfo(w, fmt.Sprintf("描述: %s", t.Description))
	}

	if len(t.Dependencies) > 0 {
		ui.PrintInfo(w, fmt.Sprintf("依賴: %s", strings.Join(t.Dependencies, ", ")))
	}

	if len(t.AcceptanceCriteria) > 0 {
		ui.PrintInfo(w, "驗收條件:")
		for _, c := range t.AcceptanceCriteria {
			fmt.Fprintf(w, "  - %s\n", c)
		}
	}

	if len(t.FilesToModify) > 0 {
		ui.PrintInfo(w, fmt.Sprintf("要修改的檔案: %s", strings.Join(t.FilesToModify, ", ")))
	}

	if len(t.FilesToCreate) > 0 {
		ui.PrintInfo(w, fmt.Sprintf("要建立的檔案: %s", strings.Join(t.FilesToCreate, ", ")))
	}
}
