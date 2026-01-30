package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
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

// ProjectSummary contains analyzed information about an existing project
type ProjectSummary struct {
	Language    string   // Primary programming language
	Framework   string   // Framework if detected
	Structure   string   // Project structure description
	MainFiles   []string // Key files in the project
	HasTests    bool     // Whether project has tests
	HasDocs     bool     // Whether project has documentation
	Description string   // AI-generated project description
}

// String returns a formatted string representation of the summary
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

// ScanProject analyzes the existing project and returns a summary
func (ia *InitAgent) ScanProject(ctx context.Context) (*ProjectSummary, error) {
	prompt := fmt.Sprintf(`你是一個專案分析專家。請分析當前目錄的專案結構。

專案目錄: %s

請掃描專案並回答：
1. 主要使用的程式語言
2. 使用的框架或工具（如果有）
3. 專案結構（主要資料夾）
4. 是否有測試檔案
5. 是否有文件（README, docs/）
6. 簡短描述這個專案的功能

請以 JSON 格式輸出：
{
  "language": "主要語言",
  "framework": "框架名稱（沒有則空字串）",
  "structure": "主要資料夾，如 cmd/, internal/, pkg/",
  "main_files": ["重要檔案1", "重要檔案2"],
  "has_tests": true/false,
  "has_docs": true/false,
  "description": "專案功能簡述"
}`, ia.projectDir)

	result, err := ia.caller.Call(ctx, prompt,
		WithWorkingDir(ia.projectDir),
		WithTimeout(3*time.Minute),
	)

	if err != nil {
		if ia.caller.DryRun {
			return ia.createMockSummary(), nil
		}
		return nil, fmt.Errorf("掃描專案失敗: %w", err)
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

// GenerateQuestions generates questions based on the initial goal and project summary
func (ia *InitAgent) GenerateQuestions(ctx context.Context, goal string, summary *ProjectSummary) ([]string, error) {
	var prompt string

	if summary != nil {
		// Existing project - generate targeted questions
		prompt = fmt.Sprintf(`你是一個專案規劃助手。使用者想要在現有專案上進行以下開發：

## 開發目標
"%s"

## 現有專案資訊
- 語言: %s
- 框架: %s
- 結構: %s
- 專案描述: %s
- 已有測試: %v
- 已有文件: %v

請產生 5-7 個針對性問題，幫助我了解更多細節以便產生完整的 milestone。
因為這是現有專案，問題應該聚焦在：
1. 新功能如何與現有架構整合
2. 是否需要修改現有模組
3. 與現有功能的互動方式
4. 相容性考量
5. 測試策略
6. 部署/遷移考量

請以 JSON 格式輸出：{"questions": ["問題1", "問題2", ...]}`,
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
		prompt = fmt.Sprintf(`你是一個專案規劃助手。使用者想要建立以下專案：

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

// GenerateMilestone generates a milestone document based on goal and answers
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
		return "", fmt.Errorf("無法建立文件目錄: %w", err)
	}

	var prompt string

	if summary != nil {
		// Existing project - generate enhancement milestone
		prompt = fmt.Sprintf(`你是一個專案規劃專家。請根據以下資訊產生詳細的 milestone 文件。

## 開發目標
%s

## 現有專案資訊
- 語言: %s
- 框架: %s
- 專案結構: %s
- 專案描述: %s
- 已有測試: %v
- 已有文件: %v

## 需求細節
%s

請產生一個 Markdown 格式的 milestone 文件，包含：
1. 開發目標概述
2. 現有架構分析（與新功能的關聯）
3. 功能需求清單
4. 實作階段規劃（分成多個 phase）
   - 考慮與現有程式碼的整合順序
   - 標註需要修改的現有模組
5. 每個階段的具體任務
6. 測試計畫（包含整合測試）
7. 驗收標準

請將結果寫入檔案: %s`,
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
		prompt = fmt.Sprintf(`你是一個專案規劃專家。請根據以下資訊產生詳細的 milestone 文件。

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
	}

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
