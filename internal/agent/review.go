package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ReviewAgent handles code review
type ReviewAgent struct {
	caller     *Caller
	projectDir string
}

// NewReviewAgent creates a new review agent
func NewReviewAgent(caller *Caller, projectDir string) *ReviewAgent {
	return &ReviewAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// ReviewResult contains the review outcome
type ReviewResult struct {
	Status   string   // APPROVED or CHANGES_REQUESTED
	Summary  string
	Issues   []string
	Suggestions []string
}

// Review reviews the given files
func (ra *ReviewAgent) Review(ctx context.Context, files []string) (*Result, *ReviewResult, error) {
	if len(files) == 0 {
		return &Result{Success: true, Output: "No files to review"}, nil, nil
	}

	prompt := ra.buildReviewPrompt(files)

	result, err := ra.caller.Call(ctx, prompt,
		WithWorkingDir(ra.projectDir),
		WithContextFiles(files...),
		WithTimeout(10*time.Minute),
	)

	if err != nil {
		return nil, nil, err
	}

	// Parse review result from output
	reviewResult := ra.parseReviewResult(result.Output)

	return result, reviewResult, nil
}

// buildReviewPrompt creates the prompt for code review
func (ra *ReviewAgent) buildReviewPrompt(files []string) string {
	var sb strings.Builder

	sb.WriteString("你是一個程式碼審查 Agent。請審查以下變更的檔案。\n\n")
	sb.WriteString(fmt.Sprintf("專案目錄: %s\n\n", ra.projectDir))
	
	sb.WriteString("變更的檔案:\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s\n", f))
	}

	sb.WriteString(`
請檢查:
1. 程式碼品質與風格一致性
2. 潛在的 bugs 或問題
3. 效能考量
4. 安全性問題
5. 測試覆蓋率

請在輸出中包含:
- 狀態: APPROVED 或 CHANGES_REQUESTED
- 摘要: 簡短的審查摘要
- 問題: 發現的問題列表 (如果有)
- 建議: 改進建議`)

	return sb.String()
}

// parseReviewResult extracts review result from output
func (ra *ReviewAgent) parseReviewResult(output string) *ReviewResult {
	result := &ReviewResult{
		Status:      "UNKNOWN",
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// Simple parsing - look for APPROVED or CHANGES_REQUESTED
	if strings.Contains(strings.ToUpper(output), "APPROVED") {
		result.Status = "APPROVED"
	} else if strings.Contains(strings.ToUpper(output), "CHANGES_REQUESTED") {
		result.Status = "CHANGES_REQUESTED"
	}

	// Extract summary (first paragraph after "摘要" or "Summary")
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "摘要") || strings.Contains(lower, "summary") {
			if i+1 < len(lines) {
				result.Summary = strings.TrimSpace(lines[i+1])
			}
		}
	}

	return result
}

// TestAgent handles test execution
type TestAgent struct {
	caller     *Caller
	projectDir string
}

// NewTestAgent creates a new test agent
func NewTestAgent(caller *Caller, projectDir string) *TestAgent {
	return &TestAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// TestResult contains the test outcome
type TestResult struct {
	Passed  int
	Failed  int
	Skipped int
	Summary string
}

// RunTests executes tests in the project
func (ta *TestAgent) RunTests(ctx context.Context) (*Result, *TestResult, error) {
	prompt := ta.buildTestPrompt()

	result, err := ta.caller.Call(ctx, prompt,
		WithWorkingDir(ta.projectDir),
		WithTimeout(15*time.Minute),
	)

	if err != nil {
		return nil, nil, err
	}

	testResult := ta.parseTestResult(result.Output)

	return result, testResult, nil
}

// buildTestPrompt creates the prompt for test execution
func (ta *TestAgent) buildTestPrompt() string {
	return fmt.Sprintf(`你是一個測試 Agent。請在專案目錄 %s 執行以下任務:

1. 檢查專案類型並找到適合的測試指令
   - Go: go test ./...
   - Java (Maven): mvn test
   - Java (Gradle): gradle test
   - Node.js: npm test
   - Python: pytest
   
2. 執行測試

3. 分析測試結果

4. 如果有測試失敗，分析失敗原因

請在輸出中包含:
- 測試摘要
- 通過/失敗的測試數量
- 失敗測試的詳細資訊 (如果有)
- 修復建議`, ta.projectDir)
}

// parseTestResult extracts test result from output
func (ta *TestAgent) parseTestResult(output string) *TestResult {
	result := &TestResult{}
	
	// Try to extract numbers from common test output formats
	// This is a simple implementation - in production you'd want more robust parsing
	
	return result
}

// CommitAgent handles git commits
type CommitAgent struct {
	caller     *Caller
	projectDir string
}

// NewCommitAgent creates a new commit agent
func NewCommitAgent(caller *Caller, projectDir string) *CommitAgent {
	return &CommitAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Commit creates a commit for a ticket
func (ca *CommitAgent) Commit(ctx context.Context, ticketID, ticketTitle, changes string) (*Result, error) {
	prompt := ca.buildCommitPrompt(ticketID, ticketTitle, changes)

	return ca.caller.Call(ctx, prompt,
		WithWorkingDir(ca.projectDir),
		WithTimeout(5*time.Minute),
	)
}

// buildCommitPrompt creates the prompt for committing
func (ca *CommitAgent) buildCommitPrompt(ticketID, ticketTitle, changes string) string {
	return fmt.Sprintf(`你是一個 Git Commit Agent。請根據以下變更產生適當的 commit 並提交。

專案目錄: %s
Ticket ID: %s
Ticket 標題: %s

目前的變更:
%s

請:
1. 分析變更內容
2. 執行 git add 將相關檔案加入暫存區
3. 產生符合 Conventional Commits 格式的 commit message
4. 執行 git commit

Commit message 格式:
<type>(<scope>): <description>

[optional body]

Refs: %s

Type 應該是: feat, fix, docs, style, refactor, test, chore`, 
		ca.projectDir, ticketID, ticketTitle, changes, ticketID)
}
