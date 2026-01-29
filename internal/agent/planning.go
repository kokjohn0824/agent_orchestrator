package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// PlanningAgent analyzes milestones and generates tickets
type PlanningAgent struct {
	caller     *Caller
	projectDir string
	ticketsDir string
}

// NewPlanningAgent creates a new planning agent
func NewPlanningAgent(caller *Caller, projectDir, ticketsDir string) *PlanningAgent {
	return &PlanningAgent{
		caller:     caller,
		projectDir: projectDir,
		ticketsDir: ticketsDir,
	}
}

// Plan analyzes a milestone file and generates tickets
func (pa *PlanningAgent) Plan(ctx context.Context, milestoneFile string) ([]*ticket.Ticket, error) {
	// Read milestone file
	content, err := os.ReadFile(milestoneFile)
	if err != nil {
		return nil, fmt.Errorf("無法讀取 milestone 檔案: %w", err)
	}

	// Prepare output file
	outputFile := filepath.Join(pa.ticketsDir, "generated-tickets.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return nil, fmt.Errorf("無法建立輸出目錄: %w", err)
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
		return nil, fmt.Errorf("規劃失敗: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("規劃失敗: %s", result.Error)
	}

	return pa.parseTickets(jsonData)
}

// buildPlanningPrompt creates the prompt for the planning agent
func (pa *PlanningAgent) buildPlanningPrompt(content, milestoneFile, outputFile string) string {
	return fmt.Sprintf(`你是一個專案規劃 Agent。請分析 milestone 文件並產生 tickets。

請讀取檔案 %s 的內容，然後產生 JSON 格式的 tickets 列表。

每個 ticket 包含:
- id: 唯一識別碼 (格式: TICKET-xxx-描述)
- title: 簡短標題
- description: 詳細描述
- type: 類型 (feature/test/refactor/docs/bugfix/performance/security)
- priority: 優先級 (1-5, 1最高)
- estimated_complexity: 複雜度 (low/medium/high)
- dependencies: 依賴的其他 ticket ID 列表
- acceptance_criteria: 驗收標準列表
- files_to_create: 需要建立的檔案
- files_to_modify: 需要修改的檔案

請確保：
1. Tickets 之間的依賴關係正確
2. 每個 ticket 都是獨立可完成的工作單元
3. 複雜的任務要拆分成多個小 tickets
4. 按照優先級排序

請將結果以 JSON 格式寫入檔案: %s
格式為: {"tickets": [...]}`, milestoneFile, outputFile)
}

// parseTickets parses the JSON output into tickets
func (pa *PlanningAgent) parseTickets(data map[string]interface{}) ([]*ticket.Ticket, error) {
	ticketsData, ok := data["tickets"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("無效的 tickets 格式")
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

	if deps, ok := data["dependencies"].([]interface{}); ok {
		t.Dependencies = toStringSlice(deps)
	}

	if criteria, ok := data["acceptance_criteria"].([]interface{}); ok {
		t.AcceptanceCriteria = toStringSlice(criteria)
	}

	if files, ok := data["files_to_create"].([]interface{}); ok {
		t.FilesToCreate = toStringSlice(files)
	}

	if files, ok := data["files_to_modify"].([]interface{}); ok {
		t.FilesToModify = toStringSlice(files)
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

func toStringSlice(slice []interface{}) []string {
	result := make([]string, 0)
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// InitAgent handles interactive project initialization
type InitAgent struct {
	caller     *Caller
	projectDir string
	docsDir    string
}

// NewInitAgent creates a new init agent
func NewInitAgent(caller *Caller, projectDir, docsDir string) *InitAgent {
	return &InitAgent{
		caller:     caller,
		projectDir: projectDir,
		docsDir:    docsDir,
	}
}

// GenerateQuestions generates questions based on the initial goal
func (ia *InitAgent) GenerateQuestions(ctx context.Context, goal string) ([]string, error) {
	prompt := fmt.Sprintf(`你是一個專案規劃助手。使用者想要建立以下專案：

"%s"

請產生 5-7 個關鍵問題，幫助我了解更多細節以便產生完整的 milestone。
問題應該涵蓋：
1. 技術選型（程式語言、框架等）
2. 目標使用者
3. 關鍵功能需求
4. 效能/規模需求
5. 部署環境
6. 整合需求

請以 JSON 格式輸出：{"questions": ["問題1", "問題2", ...]}`, goal)

	result, err := ia.caller.Call(ctx, prompt, WithTimeout(2*time.Minute))
	if err != nil {
		return nil, err
	}

	// Parse questions from output
	questions, err := ia.parseQuestions(result.Output)
	if err != nil {
		// Return default questions
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

// defaultQuestions returns default questions
func (ia *InitAgent) defaultQuestions() []string {
	return []string{
		"這個專案使用什麼程式語言？",
		"主要的目標使用者是誰？",
		"有什麼關鍵功能需求？",
		"有沒有效能或規模上的需求？",
		"需要什麼輸出格式或介面？",
	}
}

// GenerateMilestone generates a milestone document based on goal and answers
func (ia *InitAgent) GenerateMilestone(ctx context.Context, goal string, questions []string, answers []string) (string, error) {
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
		return "", fmt.Errorf("無法建立文件目錄: %w", err)
	}

	prompt := fmt.Sprintf(`你是一個專案規劃專家。請根據以下資訊產生詳細的 milestone 文件。

## 專案目標
%s

## 需求細節
%s

請產生一個 Markdown 格式的 milestone 文件，包含：
1. 專案概述
2. 技術架構
3. 功能需求清單
4. 實作階段規劃（分成多個 phase）
5. 每個階段的具體任務
6. 驗收標準

請將結果寫入檔案: %s`, goal, qaSection.String(), outputPath)

	result, err := ia.caller.Call(ctx, prompt,
		WithWorkingDir(ia.projectDir),
		WithTimeout(5*time.Minute),
	)

	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("產生 milestone 失敗: %s", result.Error)
	}

	// Check if file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		// Try to extract content from output and write it
		if err := os.WriteFile(outputPath, []byte(result.Output), 0644); err != nil {
			return "", fmt.Errorf("無法寫入 milestone 檔案: %w", err)
		}
	}

	return outputPath, nil
}
