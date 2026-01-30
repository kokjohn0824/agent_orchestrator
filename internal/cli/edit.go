package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	editTitle       string
	editType        string
	editPriority    int
	editDescription string
	editDeps        string
	editCriteria    string
	editEnhance     bool
)

var editCmd = &cobra.Command{
	Use:   "edit <ticket-id>",
	Short: i18n.CmdEditShort,
	Long:  i18n.CmdEditLong,
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", i18n.FlagTitle)
	editCmd.Flags().StringVar(&editType, "type", "", i18n.FlagType)
	editCmd.Flags().IntVar(&editPriority, "priority", 0, i18n.FlagPriority)
	editCmd.Flags().StringVar(&editDescription, "description", "", i18n.FlagDescription)
	editCmd.Flags().StringVar(&editDeps, "deps", "", i18n.FlagDeps)
	editCmd.Flags().StringVar(&editCriteria, "criteria", "", i18n.FlagCriteria)
	editCmd.Flags().BoolVar(&editEnhance, "enhance", false, i18n.FlagEnhance)
}

func runEdit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := os.Stdout
	ticketID := args[0]

	ui.PrintHeader(w, i18n.UIEditTicket)

	// Initialize store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf(i18n.ErrInitStoreFailed, err)
	}

	// Load existing ticket
	t, err := store.Load(ticketID)
	if err != nil {
		ui.PrintError(w, fmt.Sprintf(i18n.ErrTicketNotFound, ticketID))
		return nil
	}

	// Check if any flags provided for direct edit
	hasFlags := editTitle != "" || editType != "" || editPriority != 0 ||
		editDescription != "" || editDeps != "" || editCriteria != ""

	if hasFlags {
		// Direct edit mode
		applyEditFlags(t)
	} else if !editEnhance {
		// Interactive edit mode
		var editErr error
		t, editErr = editTicketInteractive(w, t)
		if editErr != nil {
			return editErr
		}
	}

	// AI enhancement if requested
	if editEnhance {
		caller, err := CreateAgentCaller()
		if err == nil {
			enhancer := agent.NewEnhanceAgent(caller, cfg.ProjectRoot)

			spinner := ui.NewSpinner(i18n.SpinnerEnhancing, w)
			spinner.Start()

			enhanced, err := enhancer.Enhance(ctx, t)
			if err != nil {
				spinner.Fail("AI 預處理失敗")
				ui.PrintWarning(w, fmt.Sprintf("AI 預處理失敗: %s，保留原始內容", err.Error()))
			} else {
				spinner.Success(i18n.MsgEnhanceComplete)
				t = enhanced
			}
		} else {
			ui.PrintWarning(w, "無法使用 AI 預處理: agent 不可用")
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
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgTicketUpdated, t.ID))
	ui.PrintInfo(w, "")
	displayTicketDetails(w, t)

	return nil
}

func applyEditFlags(t *ticket.Ticket) {
	if editTitle != "" {
		t.Title = editTitle
	}

	if editType != "" {
		switch strings.ToLower(editType) {
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
		}
	}

	if editPriority >= 1 && editPriority <= 5 {
		t.Priority = editPriority
	}

	if editDescription != "" {
		t.Description = editDescription
	}

	if editDeps != "" {
		t.Dependencies = []string{}
		for _, d := range strings.Split(editDeps, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				t.Dependencies = append(t.Dependencies, d)
			}
		}
	}

	if editCriteria != "" {
		t.AcceptanceCriteria = []string{}
		for _, c := range strings.Split(editCriteria, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				t.AcceptanceCriteria = append(t.AcceptanceCriteria, c)
			}
		}
	}
}

func editTicketInteractive(w *os.File, t *ticket.Ticket) (*ticket.Ticket, error) {
	prompt := ui.NewPrompt(os.Stdin, w)

	// Show current ticket info
	ui.PrintInfo(w, "目前 Ticket 資訊:")
	displayTicketDetails(w, t)
	ui.PrintInfo(w, "")

	// Select field to edit
	editOptions := []string{
		"標題",
		"描述",
		"類型",
		"優先級",
		"依賴",
		"驗收條件",
		"完成編輯",
	}

	for {
		idx, err := prompt.Select(i18n.PromptEditField, editOptions)
		if err != nil {
			return nil, err
		}

		switch idx {
		case 0: // Title
			newTitle, err := prompt.Ask(fmt.Sprintf("新標題 (目前: %s)", t.Title))
			if err != nil {
				return nil, err
			}
			if newTitle != "" {
				t.Title = newTitle
			}

		case 1: // Description
			ui.PrintInfo(w, fmt.Sprintf("目前描述: %s", t.Description))
			descLines, err := prompt.AskMultiline("新描述")
			if err != nil {
				return nil, err
			}
			if len(descLines) > 0 {
				t.Description = strings.Join(descLines, "\n")
			}

		case 2: // Type
			typeOptions := []string{
				"feature - 新功能",
				"bugfix - 錯誤修復",
				"refactor - 重構",
				"test - 測試",
				"docs - 文件",
				"performance - 效能優化",
				"security - 安全性",
			}
			typeIdx, err := prompt.Select(fmt.Sprintf("選擇類型 (目前: %s)", t.Type), typeOptions)
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
			t.Type = ticketTypes[typeIdx]

		case 3: // Priority
			priorityStr, err := prompt.Ask(fmt.Sprintf("新優先級 1-5 (目前: %d)", t.Priority))
			if err != nil {
				return nil, err
			}
			if priorityStr != "" {
				var newPriority int
				fmt.Sscanf(priorityStr, "%d", &newPriority)
				if newPriority >= 1 && newPriority <= 5 {
					t.Priority = newPriority
				}
			}

		case 4: // Dependencies
			ui.PrintInfo(w, fmt.Sprintf("目前依賴: %s", strings.Join(t.Dependencies, ", ")))
			depsStr, err := prompt.Ask("新依賴 (逗號分隔，留空清除)")
			if err != nil {
				return nil, err
			}
			t.Dependencies = []string{}
			if depsStr != "" {
				for _, d := range strings.Split(depsStr, ",") {
					d = strings.TrimSpace(d)
					if d != "" {
						t.Dependencies = append(t.Dependencies, d)
					}
				}
			}

		case 5: // Acceptance Criteria
			if len(t.AcceptanceCriteria) > 0 {
				ui.PrintInfo(w, "目前驗收條件:")
				for _, c := range t.AcceptanceCriteria {
					fmt.Fprintf(w, "  - %s\n", c)
				}
			}
			criteriaLines, err := prompt.AskMultiline("新驗收條件 (每行一條)")
			if err != nil {
				return nil, err
			}
			if len(criteriaLines) > 0 {
				t.AcceptanceCriteria = criteriaLines
			}

		case 6: // Done
			return t, nil
		}
	}
}
