package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// PlanningAgent analyzes milestone documents and generates tickets via the agent.
// It reads a milestone file, asks the agent to produce a JSON ticket list, and writes
// the result to ticketsDir/generated-tickets.json.
type PlanningAgent struct {
	caller     *Caller
	projectDir string
	ticketsDir string
}

// NewPlanningAgent creates a PlanningAgent with the given Caller, project directory, and tickets directory.
func NewPlanningAgent(caller *Caller, projectDir, ticketsDir string) *PlanningAgent {
	return &PlanningAgent{
		caller:     caller,
		projectDir: projectDir,
		ticketsDir: ticketsDir,
	}
}

// Plan reads the milestone file, invokes the agent to generate tickets, and returns the parsed list.
// Output is written to ticketsDir/generated-tickets.json. On dry run, returns mock tickets.
func (pa *PlanningAgent) Plan(ctx context.Context, milestoneFile string) ([]*ticket.Ticket, error) {
	// Read milestone file
	content, err := os.ReadFile(milestoneFile)
	if err != nil {
		return nil, fmt.Errorf(i18n.ErrAgentReadMilestone, err)
	}

	// Prepare output file
	outputFile := filepath.Join(pa.ticketsDir, "generated-tickets.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return nil, fmt.Errorf(i18n.ErrAgentMkdirOutput, err)
	}

	prompt := pa.buildPlanningPrompt(string(content), milestoneFile, outputFile)

	result, jsonData, err := pa.caller.CallForJSON(ctx, prompt, outputFile,
		WithContextFiles(milestoneFile),
		WithWorkingDir(pa.projectDir),
		WithTimeout(10*time.Minute),
	)

	if err != nil {
		// If dry run, create mock data
		if pa.caller.DryRun {
			return pa.createMockTickets(), nil
		}
		return nil, fmt.Errorf(i18n.ErrAgentPlanningFailed, err)
	}

	if !result.Success {
		return nil, fmt.Errorf(i18n.ErrAgentPlanningOutput, result.Error)
	}

	return pa.parseTickets(jsonData)
}

// buildPlanningPrompt creates the prompt for the planning agent
func (pa *PlanningAgent) buildPlanningPrompt(content, milestoneFile, outputFile string) string {
	return fmt.Sprintf(i18n.AgentPlanningPromptTemplate, milestoneFile, outputFile)
}

// parseTickets parses the JSON output into tickets
func (pa *PlanningAgent) parseTickets(data map[string]interface{}) ([]*ticket.Ticket, error) {
	ticketsData, ok := data["tickets"].([]interface{})
	if !ok {
		return nil, fmt.Errorf(i18n.ErrAgentInvalidTickets)
	}

	tickets := make([]*ticket.Ticket, 0)
	for _, td := range ticketsData {
		ticketMap, ok := td.(map[string]interface{})
		if !ok {
			continue
		}

		t := pa.mapToTicket(ticketMap)
		if t != nil {
			tickets = append(tickets, t)
		}
	}

	return tickets, nil
}

// mapToTicket converts a map to a ticket
func (pa *PlanningAgent) mapToTicket(data map[string]interface{}) *ticket.Ticket {
	id, _ := data["id"].(string)
	title, _ := data["title"].(string)
	description, _ := data["description"].(string)

	if id == "" || title == "" {
		return nil
	}

	t := ticket.NewTicket(id, title, description)

	if typeStr, ok := data["type"].(string); ok {
		t.Type = ticket.Type(typeStr)
	}

	if priority, ok := data["priority"].(float64); ok {
		t.Priority = int(priority)
	}

	if complexity, ok := data["estimated_complexity"].(string); ok {
		t.EstimatedComplexity = complexity
	}

	if deps := jsonutil.GetStringSlice(data, "dependencies"); deps != nil {
		t.Dependencies = deps
	}

	if criteria := jsonutil.GetStringSlice(data, "acceptance_criteria"); criteria != nil {
		t.AcceptanceCriteria = criteria
	}

	if files := jsonutil.GetStringSlice(data, "files_to_create"); files != nil {
		t.FilesToCreate = files
	}

	if files := jsonutil.GetStringSlice(data, "files_to_modify"); files != nil {
		t.FilesToModify = files
	}

	return t
}

// createMockTickets creates mock tickets for dry run
func (pa *PlanningAgent) createMockTickets() []*ticket.Ticket {
	return []*ticket.Ticket{
		{
			ID:                  "TICKET-001-setup",
			Title:               "設定專案結構",
			Description:         "建立專案結構與基本設定",
			Type:                ticket.TypeFeature,
			Priority:            1,
			Status:              ticket.StatusPending,
			EstimatedComplexity: "low",
			Dependencies:        []string{},
			AcceptanceCriteria:  []string{"專案可編譯", "基本設定完成"},
			FilesToCreate:       []string{},
			FilesToModify:       []string{},
			CreatedAt:           time.Now(),
		},
		{
			ID:                  "TICKET-002-core",
			Title:               "實作核心功能",
			Description:         "實作核心業務邏輯",
			Type:                ticket.TypeFeature,
			Priority:            2,
			Status:              ticket.StatusPending,
			EstimatedComplexity: "medium",
			Dependencies:        []string{"TICKET-001-setup"},
			AcceptanceCriteria:  []string{"核心功能可運作", "有單元測試"},
			FilesToCreate:       []string{},
			FilesToModify:       []string{},
			CreatedAt:           time.Now(),
		},
		{
			ID:                  "TICKET-003-test",
			Title:               "新增測試",
			Description:         "為核心功能新增完整測試",
			Type:                ticket.TypeTest,
			Priority:            3,
			Status:              ticket.StatusPending,
			EstimatedComplexity: "medium",
			Dependencies:        []string{"TICKET-002-core"},
			AcceptanceCriteria:  []string{"測試覆蓋率達 80%"},
			FilesToCreate:       []string{},
			FilesToModify:       []string{},
			CreatedAt:           time.Now(),
		},
	}
}

// ProjectSummary holds analyzed information about an existing project (language, framework,
// structure, main files, tests/docs presence, and a short description). Used by InitAgent
// for interactive project initialization.
type ProjectSummary struct {
	Language    string   // Primary programming language
	Framework   string   // Framework if detected
	Structure   string   // Project structure description
	MainFiles   []string // Key files in the project
	HasTests    bool     // Whether project has tests
	HasDocs     bool     // Whether project has documentation
	Description string   // AI-generated project description
}

// String returns a formatted string representation of the summary (language, framework, structure, etc.).
func (ps *ProjectSummary) String() string {
	var sb strings.Builder
	if ps.Language != "" {
		sb.WriteString(fmt.Sprintf("  - 語言: %s\n", ps.Language))
	}
	if ps.Framework != "" {
		sb.WriteString(fmt.Sprintf("  - 框架: %s\n", ps.Framework))
	}
	if ps.Structure != "" {
		sb.WriteString(fmt.Sprintf("  - 結構: %s\n", ps.Structure))
	}
	if ps.HasTests {
		sb.WriteString("  - 已有測試: 是\n")
	}
	if ps.HasDocs {
		sb.WriteString("  - 已有文件: 是\n")
	}
	return sb.String()
}

// InitAgent handles interactive project initialization: scanning the project, generating
// questions, and producing milestone documents. It uses the agent to analyze the codebase
// and to generate Q&A and milestone content.
type InitAgent struct {
	caller     *Caller
	projectDir string
	docsDir    string
}

// NewInitAgent creates an InitAgent with the given Caller, project directory, and docs directory.
func NewInitAgent(caller *Caller, projectDir, docsDir string) *InitAgent {
	return &InitAgent{
		caller:     caller,
		projectDir: projectDir,
		docsDir:    docsDir,
	}
}

// ScanProject invokes the agent to analyze the project and returns a ProjectSummary.
// On dry run or parse error, returns a mock summary.
func (ia *InitAgent) ScanProject(ctx context.Context) (*ProjectSummary, error) {
	prompt := fmt.Sprintf(i18n.AgentInitScanIntro, ia.projectDir)

	result, err := ia.caller.Call(ctx, prompt,
		WithWorkingDir(ia.projectDir),
		WithTimeout(3*time.Minute),
	)

	if err != nil {
		if ia.caller.DryRun {
			return ia.createMockSummary(), nil
		}
		return nil, fmt.Errorf(i18n.ErrAgentScanFailed, err)
	}

	summary, err := ia.parseSummary(result.Output)
	if err != nil {
		// Return a basic summary on parse error
		return ia.createMockSummary(), nil
	}

	return summary, nil
}

// parseSummary extracts ProjectSummary from the agent output
func (ia *InitAgent) parseSummary(output string) (*ProjectSummary, error) {
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found")
	}

	jsonStr := output[start : end+1]
	var data struct {
		Language    string   `json:"language"`
		Framework   string   `json:"framework"`
		Structure   string   `json:"structure"`
		MainFiles   []string `json:"main_files"`
		HasTests    bool     `json:"has_tests"`
		HasDocs     bool     `json:"has_docs"`
		Description string   `json:"description"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	return &ProjectSummary{
		Language:    data.Language,
		Framework:   data.Framework,
		Structure:   data.Structure,
		MainFiles:   data.MainFiles,
		HasTests:    data.HasTests,
		HasDocs:     data.HasDocs,
		Description: data.Description,
	}, nil
}

// createMockSummary creates a mock summary for dry run or errors
func (ia *InitAgent) createMockSummary() *ProjectSummary {
	return &ProjectSummary{
		Language:    "[DRY RUN] 未知",
		Framework:   "[DRY RUN] 未知",
		Structure:   "[DRY RUN] 未知",
		MainFiles:   []string{},
		HasTests:    false,
		HasDocs:     false,
		Description: "[DRY RUN] AI 會分析專案結構並產生摘要",
	}
}

// GenerateQuestions uses the agent to generate 5–7 questions based on the goal and optional
// project summary. For an existing project, questions focus on integration and compatibility;
// for a new project, on tech stack and requirements.
func (ia *InitAgent) GenerateQuestions(ctx context.Context, goal string, summary *ProjectSummary) ([]string, error) {
	var prompt string

	if summary != nil {
		// Existing project - generate targeted questions
		prompt = fmt.Sprintf(i18n.AgentInitQuestionsExisting,
			goal,
			summary.Language,
			summary.Framework,
			summary.Structure,
			summary.Description,
			summary.HasTests,
			summary.HasDocs,
		)
	} else {
		// New project - generate general questions
		prompt = fmt.Sprintf(i18n.AgentInitQuestionsNew, goal)
	}

	result, err := ia.caller.Call(ctx, prompt, WithTimeout(2*time.Minute))
	if err != nil {
		return nil, err
	}

	// Parse questions from output
	questions, err := ia.parseQuestions(result.Output)
	if err != nil {
		// Return default questions
		if summary != nil {
			return ia.defaultQuestionsExisting(), nil
		}
		return ia.defaultQuestions(), nil
	}

	return questions, nil
}

// parseQuestions extracts questions from the agent output
func (ia *InitAgent) parseQuestions(output string) ([]string, error) {
	// Try to find JSON in the output
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found")
	}

	jsonStr := output[start : end+1]
	var data struct {
		Questions []string `json:"questions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	return data.Questions, nil
}

// defaultQuestions returns default questions for new projects
func (ia *InitAgent) defaultQuestions() []string {
	return []string{
		"這個專案使用什麼程式語言？",
		"主要的目標使用者是誰？",
		"有什麼關鍵功能需求？",
		"有沒有效能或規模上的需求？",
		"需要什麼輸出格式或介面？",
	}
}

// defaultQuestionsExisting returns default questions for existing projects
func (ia *InitAgent) defaultQuestionsExisting() []string {
	return []string{
		"這個新功能如何與現有架構整合？",
		"是否需要修改現有的模組或 API？",
		"有沒有相容性的考量？",
		"需要新增哪些測試？",
		"是否需要更新文件？",
	}
}

// GenerateMilestone invokes the agent to produce a milestone Markdown file from the goal,
// Q&A (questions and answers), and optional project summary. Returns the path to the written file.
func (ia *InitAgent) GenerateMilestone(ctx context.Context, goal string, questions []string, answers []string, summary *ProjectSummary) (string, error) {
	// Build Q&A section
	var qaSection strings.Builder
	for i, q := range questions {
		if i < len(answers) {
			qaSection.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", q, answers[i]))
		}
	}

	// Create output file path
	timestamp := time.Now().Format("20060102")
	filename := fmt.Sprintf("milestone-%s-generated.md", timestamp)
	outputPath := filepath.Join(ia.docsDir, filename)

	if err := os.MkdirAll(ia.docsDir, 0755); err != nil {
		return "", fmt.Errorf(i18n.ErrAgentMkdirDocs, err)
	}

	var prompt string

	if summary != nil {
		// Existing project - generate enhancement milestone
		prompt = fmt.Sprintf(i18n.AgentInitMilestoneExisting,
			goal,
			summary.Language,
			summary.Framework,
			summary.Structure,
			summary.Description,
			summary.HasTests,
			summary.HasDocs,
			qaSection.String(),
			outputPath,
		)
	} else {
		// New project - generate standard milestone
		prompt = fmt.Sprintf(i18n.AgentInitMilestoneNew, goal, qaSection.String(), outputPath)
	}

	result, err := ia.caller.Call(ctx, prompt,
		WithWorkingDir(ia.projectDir),
		WithTimeout(5*time.Minute),
	)

	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf(i18n.ErrAgentCreateMilestone, result.Error)
	}

	// Check if file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		// Try to extract content from output and write it
		if err := os.WriteFile(outputPath, []byte(result.Output), 0644); err != nil {
			return "", fmt.Errorf(i18n.ErrAgentWriteMilestone, err)
		}
	}

	return outputPath, nil
}
