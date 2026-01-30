package agent

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ReviewAgent invokes the agent to perform code review on given files.
// It builds a prompt listing the files and asks for status (APPROVED/CHANGES_REQUESTED),
// summary, issues, and suggestions.
type ReviewAgent struct {
	caller     *Caller
	projectDir string
}

// NewReviewAgent creates a ReviewAgent with the given Caller and project directory.
func NewReviewAgent(caller *Caller, projectDir string) *ReviewAgent {
	return &ReviewAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// ReviewResult holds the parsed outcome of a code review: status (APPROVED or CHANGES_REQUESTED),
// summary, list of issues, and list of suggestions.
type ReviewResult struct {
	Status      string   // APPROVED or CHANGES_REQUESTED
	Summary     string
	Issues      []string
	Suggestions []string
}

// Review runs the agent to review the given file paths and returns the raw Result,
// parsed ReviewResult, and any error.
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

// statusPattern matches "狀態: APPROVED" or "Status: CHANGES_REQUESTED" (with optional colon variants)
var statusPattern = regexp.MustCompile(`(?i)(?:狀態|Status)\s*[：:]\s*(APPROVED|CHANGES_REQUESTED)`)

// summaryInlinePattern matches "摘要: xxx" or "Summary: xxx" on same line
var summaryInlinePattern = regexp.MustCompile(`(?i)(?:摘要|Summary)\s*[：:]\s*(.+)`)

// parseReviewResult extracts review result from output with robust parsing.
// It prefers explicit "狀態: APPROVED/CHANGES_REQUESTED", then falls back to keyword presence.
// Summary can be on same line ("摘要: ...") or on the next line after "摘要"/"Summary".
// Issues and Suggestions are parsed from list sections after "問題"/"Issues" and "建議"/"Suggestions".
func (ra *ReviewAgent) parseReviewResult(output string) *ReviewResult {
	result := &ReviewResult{
		Status:      "UNKNOWN",
		Issues:      make([]string, 0),
		Suggestions: make([]string, 0),
	}

	// 1. Status: prefer explicit "狀態: X" / "Status: X"
	if m := statusPattern.FindStringSubmatch(output); len(m) >= 2 {
		result.Status = strings.ToUpper(m[1])
	} else {
		// Fallback: keyword presence (check CHANGES_REQUESTED first to avoid matching inside it)
		upper := strings.ToUpper(output)
		if strings.Contains(upper, "CHANGES_REQUESTED") {
			result.Status = "CHANGES_REQUESTED"
		} else if strings.Contains(upper, "APPROVED") {
			result.Status = "APPROVED"
		}
	}

	// 2. Summary: inline "摘要: xxx" first, then "摘要" or "Summary" next line
	if m := summaryInlinePattern.FindStringSubmatch(output); len(m) >= 2 {
		result.Summary = strings.TrimSpace(m[1])
	}
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		// Header-only line (摘要 or Summary without content on same line)
		if (lower == "摘要" || lower == "summary") && result.Summary == "" {
			if i+1 < len(lines) {
				result.Summary = strings.TrimSpace(lines[i+1])
			}
			break
		}
		if (strings.HasPrefix(lower, "摘要") || strings.HasPrefix(lower, "summary")) && result.Summary == "" {
			// "摘要: xxx" already handled by regex; here handle "摘要 xxx" without colon
			idx := strings.IndexRune(line, ':')
			if idx == -1 {
				idx = strings.IndexRune(line, '：')
			}
			if idx >= 0 {
				result.Summary = strings.TrimSpace(line[idx+1:])
			}
		}
	}

	// 3. Issues: lines after "問題" or "Issues" until next section or empty block
	result.Issues = parseListSection(output, []string{"問題", "issues"}, []string{"建議", "suggestions", "summary", "摘要"})

	// 4. Suggestions: lines after "建議" or "Suggestions"
	result.Suggestions = parseListSection(output, []string{"建議", "suggestions"}, []string{"問題", "issues", "狀態", "status"})

	return result
}

// parseListSection extracts list items (lines starting with -, *, or digits.) from a section
// that starts with one of startMarkers and ends before any of endMarkers (section headers).
func parseListSection(output string, startMarkers, endMarkers []string) []string {
	lines := strings.Split(output, "\n")
	var inSection bool
	var list []string
	listLinePattern := regexp.MustCompile(`^\s*[-*•]\s+(.+)$`)
	numberedPattern := regexp.MustCompile(`^\s*\d+[.)]\s+(.+)$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if lower == "" {
			continue
		}
		for _, m := range startMarkers {
			if strings.HasPrefix(lower, m) || lower == m {
				inSection = true
				// Inline content after "問題: item1" (use trimmed so offset is correct)
				rest := strings.TrimSpace(trimmed[len(m):])
				if rest != "" {
					rest = strings.TrimPrefix(rest, ":")
					rest = strings.TrimPrefix(rest, "：")
					rest = strings.TrimSpace(rest)
					if rest != "" {
						list = append(list, rest)
					}
				}
				break
			}
		}
		if !inSection {
			continue
		}
		for _, e := range endMarkers {
			if !strings.HasPrefix(lower, e) {
				continue
			}
			if len(lower) == len(e) {
				return list
			}
			rest := lower[len(e):]
			r, _ := utf8.DecodeRuneInString(rest)
			if r == ' ' || r == ':' || r == '\uFF1A' { // \uFF1A = fullwidth colon '：'
				return list
			}
		}
		if sub := listLinePattern.FindStringSubmatch(line); len(sub) >= 2 {
			list = append(list, strings.TrimSpace(sub[1]))
		} else if sub := numberedPattern.FindStringSubmatch(line); len(sub) >= 2 {
			list = append(list, strings.TrimSpace(sub[1]))
		}
	}
	return list
}

// TestAgent invokes the agent to run tests in the project (e.g. go test, pytest).
// It parses the agent output to extract pass/fail/skip counts and a summary.
type TestAgent struct {
	caller     *Caller
	projectDir string
}

// NewTestAgent creates a TestAgent with the given Caller and project directory.
func NewTestAgent(caller *Caller, projectDir string) *TestAgent {
	return &TestAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// TestResult holds the parsed test outcome: passed/failed/skipped counts and a summary string.
type TestResult struct {
	Passed  int
	Failed  int
	Skipped int
	Summary string
}

// RunTests runs the agent to execute tests in the project and returns the raw Result,
// parsed TestResult (go test or pytest format), and any error.
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

// goTestOkPattern matches "ok  \tpath/to/pkg\t0.123s" or "ok  path 0.12s"
var goTestOkPattern = regexp.MustCompile(`(?m)^ok\s+(\S+)\s+([\d.]+s)`)

// goTestFailPattern matches "FAIL\tpath/to/pkg\t0.123s"
var goTestFailPattern = regexp.MustCompile(`(?m)^FAIL\s+(\S+)\s+([\d.]+s)`)

// goTestPassFailPattern matches "--- PASS: TestName (0.00s)" or "--- FAIL: TestName (0.00s)"
var goTestPassFailPattern = regexp.MustCompile(`(?m)^--- (PASS|FAIL): (\S+) \(([\d.]+s)\)`)

// pytestResultPattern matches "3 passed in 0.12s", "2 failed, 1 passed", "1 failed, 2 passed, 1 skipped in 0.30s"
var pytestPassedPattern = regexp.MustCompile(`(\d+)\s+passed`)
var pytestFailedPattern = regexp.MustCompile(`(\d+)\s+failed`)
var pytestSkippedPattern = regexp.MustCompile(`(\d+)\s+skipped`)
var pytestErrorPattern = regexp.MustCompile(`(\d+)\s+error`)

// parseTestResult extracts test result from output.
// It supports common formats: go test (ok/FAIL lines and --- PASS/--- FAIL), pytest (X passed, Y failed).
func (ta *TestAgent) parseTestResult(output string) *TestResult {
	result := &TestResult{}

	// Try go test format first: --- PASS / --- FAIL lines (most precise)
	passCount := 0
	failCount := 0
	for _, m := range goTestPassFailPattern.FindAllStringSubmatch(output, -1) {
		if len(m) >= 2 {
			switch m[1] {
			case "PASS":
				passCount++
			case "FAIL":
				failCount++
			}
		}
	}
	if passCount > 0 || failCount > 0 {
		result.Passed = passCount
		result.Failed = failCount
		result.Summary = summarizeTestResult(result.Passed, result.Failed, result.Skipped)
		return result
	}

	// Go test: ok / FAIL package lines (aggregate)
	okMatches := goTestOkPattern.FindAllStringSubmatch(output, -1)
	failMatches := goTestFailPattern.FindAllStringSubmatch(output, -1)
	if len(okMatches) > 0 || len(failMatches) > 0 {
		// Count packages: each "ok" is at least one passed package; each "FAIL" is one failed package
		result.Passed = len(okMatches)
		result.Failed = len(failMatches)
		result.Summary = summarizeTestResult(result.Passed, result.Failed, result.Skipped)
		return result
	}

	// Pytest format: "X passed", "Y failed", "Z skipped"
	if m := pytestPassedPattern.FindStringSubmatch(output); len(m) >= 2 {
		result.Passed, _ = strconv.Atoi(m[1])
	}
	if m := pytestFailedPattern.FindStringSubmatch(output); len(m) >= 2 {
		result.Failed, _ = strconv.Atoi(m[1])
	}
	if m := pytestSkippedPattern.FindStringSubmatch(output); len(m) >= 2 {
		result.Skipped, _ = strconv.Atoi(m[1])
	}
	if m := pytestErrorPattern.FindStringSubmatch(output); len(m) >= 2 {
		n, _ := strconv.Atoi(m[1])
		result.Failed += n
	}
	if result.Passed > 0 || result.Failed > 0 || result.Skipped > 0 {
		result.Summary = summarizeTestResult(result.Passed, result.Failed, result.Skipped)
		return result
	}

	result.Summary = summarizeTestResult(result.Passed, result.Failed, result.Skipped)
	return result
}

func summarizeTestResult(passed, failed, skipped int) string {
	if passed == 0 && failed == 0 && skipped == 0 {
		return ""
	}
	var parts []string
	if passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", passed))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	return strings.Join(parts, ", ")
}

// CommitAgent invokes the agent to create a git commit for a ticket (git add + commit with
// Conventional Commits style message). It uses the ticket ID, title, and change description.
type CommitAgent struct {
	caller     *Caller
	projectDir string
}

// NewCommitAgent creates a CommitAgent with the given Caller and project directory.
func NewCommitAgent(caller *Caller, projectDir string) *CommitAgent {
	return &CommitAgent{
		caller:     caller,
		projectDir: projectDir,
	}
}

// Commit runs the agent to stage and commit changes with a message referencing the ticket.
// Returns the agent Result and any error.
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
