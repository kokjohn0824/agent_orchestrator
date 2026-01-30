package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// EnhanceAgent enhances ticket content using AI
type EnhanceAgent struct {
	caller     *Caller
	projectDir string
}

// NewEnhanceAgent creates a new enhance agent
func NewEnhanceAgent(caller *Caller, projectDir string) *EnhanceAgent {
	return &EnhanceAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Enhance analyzes the ticket and project to add more details
func (ea *EnhanceAgent) Enhance(ctx context.Context, t *ticket.Ticket) (*ticket.Ticket, error) {
	prompt := ea.buildPrompt(t)

	outputFile := filepath.Join(ea.projectDir, ".tickets", "enhance-result.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
		return nil, fmt.Errorf("無法建立輸出目錄: %w", err)
	}

	result, jsonData, err := ea.caller.CallForJSON(ctx, prompt, outputFile,
		WithWorkingDir(ea.projectDir),
		WithTimeout(5*time.Minute),
	)

	if err != nil {
		if ea.caller.DryRun {
			return ea.createMockEnhanced(t), nil
		}
		return nil, fmt.Errorf("AI 預處理失敗: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("AI 預處理失敗: %s", result.Error)
	}

	return ea.applyEnhancements(t, jsonData)
}

// buildPrompt creates the prompt for enhancement
func (ea *EnhanceAgent) buildPrompt(t *ticket.Ticket) string {
	var sb strings.Builder

	sb.WriteString("你是一個專案分析專家。請根據以下 ticket 資訊和專案結構，補充更詳細的實作細節。\n\n")
	sb.WriteString(fmt.Sprintf("專案目錄: %s\n\n", ea.projectDir))

	sb.WriteString("## 原始 Ticket 資訊\n")
	sb.WriteString(fmt.Sprintf("- ID: %s\n", t.ID))
	sb.WriteString(fmt.Sprintf("- 標題: %s\n", t.Title))
	sb.WriteString(fmt.Sprintf("- 類型: %s\n", t.Type))
	sb.WriteString(fmt.Sprintf("- 優先級: P%d\n", t.Priority))
	if t.Description != "" {
		sb.WriteString(fmt.Sprintf("- 描述: %s\n", t.Description))
	}
	if len(t.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("- 依賴: %s\n", strings.Join(t.Dependencies, ", ")))
	}
	if len(t.AcceptanceCriteria) > 0 {
		sb.WriteString("- 驗收條件:\n")
		for _, c := range t.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
	}
	sb.WriteString("\n")

	sb.WriteString(`## 請分析專案結構並補充以下資訊

請以 JSON 格式輸出分析結果：
{
  "description": "補充或改進的詳細描述",
  "estimated_complexity": "low|medium|high",
  "acceptance_criteria": ["驗收條件1", "驗收條件2"],
  "files_to_create": ["可能需要建立的檔案路徑"],
  "files_to_modify": ["可能需要修改的檔案路徑"],
  "implementation_hints": ["實作建議1", "實作建議2"]
}

分析要點:
1. 根據專案結構推斷需要修改或建立的檔案
2. 評估實作複雜度 (low/medium/high)
3. 補充具體可測試的驗收條件
4. 提供實作建議

請將結果寫入 .tickets/enhance-result.json`)

	return sb.String()
}

// applyEnhancements applies the AI suggestions to the ticket
func (ea *EnhanceAgent) applyEnhancements(t *ticket.Ticket, data map[string]interface{}) (*ticket.Ticket, error) {
	// Create a copy to avoid modifying the original
	enhanced := &ticket.Ticket{
		ID:                  t.ID,
		Title:               t.Title,
		Description:         t.Description,
		Type:                t.Type,
		Priority:            t.Priority,
		Status:              t.Status,
		EstimatedComplexity: t.EstimatedComplexity,
		Dependencies:        t.Dependencies,
		AcceptanceCriteria:  t.AcceptanceCriteria,
		FilesToCreate:       t.FilesToCreate,
		FilesToModify:       t.FilesToModify,
		CreatedAt:           t.CreatedAt,
	}

	// Apply description enhancement
	if desc := jsonutil.GetString(data, "description"); desc != "" {
		if enhanced.Description == "" {
			enhanced.Description = desc
		} else {
			// Append AI suggestions if original exists
			enhanced.Description = enhanced.Description + "\n\n## AI 補充說明\n" + desc
		}
	}

	// Apply complexity
	if complexity := jsonutil.GetString(data, "estimated_complexity"); complexity != "" {
		enhanced.EstimatedComplexity = complexity
	}

	// Apply acceptance criteria
	if criteria := jsonutil.GetStringSlice(data, "acceptance_criteria"); len(criteria) > 0 {
		// Merge with existing criteria
		existing := make(map[string]bool)
		for _, c := range enhanced.AcceptanceCriteria {
			existing[c] = true
		}
		for _, c := range criteria {
			if !existing[c] && c != "" {
				enhanced.AcceptanceCriteria = append(enhanced.AcceptanceCriteria, c)
			}
		}
	}

	// Apply files to create
	if files := jsonutil.GetStringSlice(data, "files_to_create"); len(files) > 0 {
		existing := make(map[string]bool)
		for _, f := range enhanced.FilesToCreate {
			existing[f] = true
		}
		for _, f := range files {
			if !existing[f] && f != "" {
				enhanced.FilesToCreate = append(enhanced.FilesToCreate, f)
			}
		}
	}

	// Apply files to modify
	if files := jsonutil.GetStringSlice(data, "files_to_modify"); len(files) > 0 {
		existing := make(map[string]bool)
		for _, f := range enhanced.FilesToModify {
			existing[f] = true
		}
		for _, f := range files {
			if !existing[f] && f != "" {
				enhanced.FilesToModify = append(enhanced.FilesToModify, f)
			}
		}
	}

	return enhanced, nil
}

// createMockEnhanced creates mock enhanced ticket for dry run
func (ea *EnhanceAgent) createMockEnhanced(t *ticket.Ticket) *ticket.Ticket {
	enhanced := &ticket.Ticket{
		ID:                  t.ID,
		Title:               t.Title,
		Description:         t.Description,
		Type:                t.Type,
		Priority:            t.Priority,
		Status:              t.Status,
		EstimatedComplexity: "medium",
		Dependencies:        t.Dependencies,
		AcceptanceCriteria:  t.AcceptanceCriteria,
		FilesToCreate:       t.FilesToCreate,
		FilesToModify:       t.FilesToModify,
		CreatedAt:           t.CreatedAt,
	}

	if enhanced.Description == "" {
		enhanced.Description = "[DRY RUN] AI 會根據專案結構分析並補充描述"
	}

	if len(enhanced.AcceptanceCriteria) == 0 {
		enhanced.AcceptanceCriteria = []string{
			"[DRY RUN] 功能正確實作",
			"[DRY RUN] 通過單元測試",
			"[DRY RUN] 程式碼符合專案規範",
		}
	}

	if len(enhanced.FilesToModify) == 0 {
		enhanced.FilesToModify = []string{
			"[DRY RUN] AI 會分析專案結構推薦檔案",
		}
	}

	return enhanced
}
