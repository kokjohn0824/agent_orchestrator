package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// EnhanceAgent uses the agent to enrich a ticket with more details (description, complexity,
// acceptance criteria, files to create/modify) based on the project structure.
type EnhanceAgent struct {
	caller     *Caller
	projectDir string
}

// NewEnhanceAgent creates an EnhanceAgent with the given Caller and project directory.
func NewEnhanceAgent(caller *Caller, projectDir string) *EnhanceAgent {
	return &EnhanceAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Enhance invokes the agent to analyze the ticket and project, then merges the AI output
// into a new ticket (description, estimated_complexity, acceptance_criteria, files_to_create/modify).
// Output is written to .tickets/enhance-result.json. On dry run, returns a mock-enhanced ticket.
func (ea *EnhanceAgent) Enhance(ctx context.Context, t *ticket.Ticket) (*ticket.Ticket, error) {
	prompt := ea.buildPrompt(t)

	outputFile := filepath.Join(ea.projectDir, ".tickets", "enhance-result.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0700); err != nil {
		return nil, fmt.Errorf(i18n.ErrAgentMkdirOutput, err)
	}

	result, jsonData, err := ea.caller.CallForJSON(ctx, prompt, outputFile,
		WithWorkingDir(ea.projectDir),
		WithTimeout(5*time.Minute),
	)

	if err != nil {
		if ea.caller.DryRun {
			return ea.createMockEnhanced(t), nil
		}
		return nil, fmt.Errorf(i18n.ErrAgentEnhanceFailed, err)
	}

	if !result.Success {
		return nil, fmt.Errorf(i18n.ErrAgentEnhanceOutput, result.Error)
	}

	return ea.applyEnhancements(t, jsonData)
}

// buildPrompt creates the prompt for enhancement
func (ea *EnhanceAgent) buildPrompt(t *ticket.Ticket) string {
	var sb strings.Builder

	sb.WriteString(i18n.AgentEnhanceIntro)
	sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceProjectDir, ea.projectDir))
	sb.WriteString(i18n.AgentEnhanceSection)
	sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceId, t.ID))
	sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceTitle, t.Title))
	sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceType, t.Type))
	sb.WriteString(fmt.Sprintf(i18n.AgentEnhancePriority, t.Priority))
	if t.Description != "" {
		sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceDesc, t.Description))
	}
	if len(t.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf(i18n.AgentEnhanceDeps, strings.Join(t.Dependencies, ", ")))
	}
	if len(t.AcceptanceCriteria) > 0 {
		sb.WriteString(i18n.AgentEnhanceCriteria)
		for _, c := range t.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("  - %s\n", c))
		}
	}
	sb.WriteString("\n")
	sb.WriteString(i18n.AgentEnhanceJSONBlock)

	return sb.String()
}

// mergeStringSlices merges new strings into existing, deduplicating by value and skipping empty strings.
// Existing items come first; new items are appended only if not already present.
func mergeStringSlices(existing, new []string) []string {
	seen := make(map[string]bool)
	for _, s := range existing {
		seen[s] = true
	}
	result := make([]string, len(existing), len(existing)+len(new))
	copy(result, existing)
	for _, s := range new {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
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
		enhanced.AcceptanceCriteria = mergeStringSlices(enhanced.AcceptanceCriteria, criteria)
	}

	// Apply files to create
	if files := jsonutil.GetStringSlice(data, "files_to_create"); len(files) > 0 {
		enhanced.FilesToCreate = mergeStringSlices(enhanced.FilesToCreate, files)
	}

	// Apply files to modify
	if files := jsonutil.GetStringSlice(data, "files_to_modify"); len(files) > 0 {
		enhanced.FilesToModify = mergeStringSlices(enhanced.FilesToModify, files)
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
