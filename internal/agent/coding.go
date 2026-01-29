package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// CodingAgent implements tickets by writing code
type CodingAgent struct {
	caller     *Caller
	projectDir string
}

// NewCodingAgent creates a new coding agent
func NewCodingAgent(caller *Caller, projectDir string) *CodingAgent {
	return &CodingAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Execute implements a ticket
func (ca *CodingAgent) Execute(ctx context.Context, t *ticket.Ticket) (*Result, error) {
	prompt := ca.buildPrompt(t)

	// Collect context files
	contextFiles := make([]string, 0)
	for _, f := range t.FilesToModify {
		fullPath := filepath.Join(ca.projectDir, f)
		if _, err := os.Stat(fullPath); err == nil {
			contextFiles = append(contextFiles, fullPath)
		}
	}

	opts := []CallOption{
		WithWorkingDir(ca.projectDir),
		WithTimeout(10 * time.Minute),
	}

	if len(contextFiles) > 0 {
		opts = append(opts, WithContextFiles(contextFiles...))
	}

	return ca.caller.Call(ctx, prompt, opts...)
}

// buildPrompt creates the prompt for the coding agent
func (ca *CodingAgent) buildPrompt(t *ticket.Ticket) string {
	var sb strings.Builder

	sb.WriteString("你是一個專業的開發 Agent。請根據以下 ticket 實作程式碼。\n\n")
	sb.WriteString(fmt.Sprintf("專案根目錄: %s\n\n", ca.projectDir))

	sb.WriteString("## Ticket 資訊\n")
	sb.WriteString(fmt.Sprintf("- ID: %s\n", t.ID))
	sb.WriteString(fmt.Sprintf("- 標題: %s\n", t.Title))
	sb.WriteString(fmt.Sprintf("- 描述: %s\n", t.Description))
	sb.WriteString(fmt.Sprintf("- 類型: %s\n", t.Type))
	sb.WriteString(fmt.Sprintf("- 複雜度: %s\n\n", t.EstimatedComplexity))

	if len(t.FilesToCreate) > 0 {
		sb.WriteString("## 需要建立的檔案\n")
		for _, f := range t.FilesToCreate {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(t.FilesToModify) > 0 {
		sb.WriteString("## 需要修改的檔案\n")
		for _, f := range t.FilesToModify {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	if len(t.AcceptanceCriteria) > 0 {
		sb.WriteString("## 驗收標準\n")
		for _, c := range t.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## 請執行以下步驟:
1. 閱讀相關的現有程式碼 (如果有)
2. 實作 ticket 所描述的功能
3. 確保程式碼符合最佳實踐
4. 新增必要的 import 語句
5. 確保程式碼可以編譯
6. 如果適當，新增對應的單元測試

完成後，說明你所做的變更。`)

	return sb.String()
}

// AnalyzeAgent analyzes existing code and generates issues
type AnalyzeAgent struct {
	caller     *Caller
	projectDir string
}

// NewAnalyzeAgent creates a new analyze agent
func NewAnalyzeAgent(caller *Caller, projectDir string) *AnalyzeAgent {
	return &AnalyzeAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// AnalyzeScope defines what to analyze
type AnalyzeScope struct {
	Performance bool
	Refactor    bool
	Security    bool
	Test        bool
	Docs        bool
}

// AllScopes returns a scope with all options enabled
func AllScopes() AnalyzeScope {
	return AnalyzeScope{
		Performance: true,
		Refactor:    true,
		Security:    true,
		Test:        true,
		Docs:        true,
	}
}

// ParseScopes parses scope strings into AnalyzeScope
func ParseScopes(scopes []string) AnalyzeScope {
	as := AnalyzeScope{}
	for _, s := range scopes {
		switch strings.ToLower(s) {
		case "all":
			return AllScopes()
		case "performance", "perf":
			as.Performance = true
		case "refactor":
			as.Refactor = true
		case "security", "sec":
			as.Security = true
		case "test":
			as.Test = true
		case "docs":
			as.Docs = true
		}
	}
	return as
}

// Analyze analyzes the project and returns issues
func (aa *AnalyzeAgent) Analyze(ctx context.Context, scope AnalyzeScope) (*ticket.IssueList, error) {
	prompt := aa.buildAnalyzePrompt(scope)

	outputFile := filepath.Join(aa.projectDir, ".tickets", "analysis-result.json")
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return nil, fmt.Errorf("無法建立輸出目錄: %w", err)
	}

	result, jsonData, err := aa.caller.CallForJSON(ctx, prompt, outputFile,
		WithWorkingDir(aa.projectDir),
		WithTimeout(15*time.Minute),
	)

	if err != nil {
		if aa.caller.DryRun {
			return aa.createMockIssues(scope), nil
		}
		return nil, fmt.Errorf("分析失敗: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("分析失敗: %s", result.Error)
	}

	return aa.parseIssues(jsonData)
}

// buildAnalyzePrompt creates the prompt for analysis
func (aa *AnalyzeAgent) buildAnalyzePrompt(scope AnalyzeScope) string {
	var sb strings.Builder

	sb.WriteString("你是一個程式碼分析專家。請分析當前專案的程式碼，找出可改進的地方。\n\n")
	sb.WriteString(fmt.Sprintf("專案目錄: %s\n\n", aa.projectDir))

	sb.WriteString("請分析以下方面：\n")
	if scope.Performance {
		sb.WriteString("- **效能問題**: N+1 查詢、不必要的迴圈、記憶體浪費等\n")
	}
	if scope.Refactor {
		sb.WriteString("- **重構建議**: 過長的方法、重複程式碼、缺少抽象等\n")
	}
	if scope.Security {
		sb.WriteString("- **安全性問題**: 硬編碼密碼、SQL 注入、XSS 等\n")
	}
	if scope.Test {
		sb.WriteString("- **測試覆蓋**: 缺少測試的關鍵功能\n")
	}
	if scope.Docs {
		sb.WriteString("- **文件缺失**: 缺少重要文件或註解\n")
	}

	sb.WriteString(`
請以 JSON 格式輸出分析結果：
{
  "issues": [
    {
      "id": "ISSUE-001",
      "category": "performance|refactor|security|test|docs",
      "severity": "HIGH|MED|LOW",
      "title": "問題標題",
      "description": "詳細描述",
      "location": "檔案路徑:行號",
      "suggestion": "建議修復方式"
    }
  ]
}

請將結果寫入 .tickets/analysis-result.json`)

	return sb.String()
}

// parseIssues parses the JSON output into issues
func (aa *AnalyzeAgent) parseIssues(data map[string]interface{}) (*ticket.IssueList, error) {
	issuesData, ok := data["issues"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("無效的 issues 格式")
	}

	il := ticket.NewIssueList()
	for _, id := range issuesData {
		issueMap, ok := id.(map[string]interface{})
		if !ok {
			continue
		}

		issue := &ticket.Issue{
			ID:          getString(issueMap, "id"),
			Category:    getString(issueMap, "category"),
			Severity:    getString(issueMap, "severity"),
			Title:       getString(issueMap, "title"),
			Description: getString(issueMap, "description"),
			Location:    getString(issueMap, "location"),
			Suggestion:  getString(issueMap, "suggestion"),
		}

		if issue.ID != "" && issue.Title != "" {
			il.Add(issue)
		}
	}

	return il, nil
}

// createMockIssues creates mock issues for dry run
func (aa *AnalyzeAgent) createMockIssues(scope AnalyzeScope) *ticket.IssueList {
	il := ticket.NewIssueList()

	if scope.Performance {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-001",
			Category:    "performance",
			Severity:    "HIGH",
			Title:       "N+1 查詢問題",
			Description: "在迴圈中執行資料庫查詢",
			Location:    "service/user.go:45",
			Suggestion:  "使用批次查詢或 eager loading",
		})
	}

	if scope.Refactor {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-002",
			Category:    "refactor",
			Severity:    "MED",
			Title:       "過長方法需拆分",
			Description: "方法超過 100 行，應該拆分成更小的函數",
			Location:    "handler/api.go:50-180",
			Suggestion:  "將方法拆分成多個小型函數",
		})
	}

	if scope.Security {
		il.Add(&ticket.Issue{
			ID:          "ISSUE-003",
			Category:    "security",
			Severity:    "HIGH",
			Title:       "硬編碼密碼",
			Description: "在程式碼中發現硬編碼的密碼",
			Location:    "config/db.go:15",
			Suggestion:  "使用環境變數或設定檔",
		})
	}

	return il
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
