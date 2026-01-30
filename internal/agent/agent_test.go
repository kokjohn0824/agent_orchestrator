package agent

import (
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/jsonutil"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

func TestCodingAgent_buildPrompt(t *testing.T) {
	ca := NewCodingAgent(nil, "/test/project")

	tests := []struct {
		name           string
		ticket         *ticket.Ticket
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "basic ticket with all fields",
			ticket: &ticket.Ticket{
				ID:                  "TEST-001",
				Title:               "Test Feature",
				Description:         "Implement test feature",
				Type:                ticket.TypeFeature,
				EstimatedComplexity: "medium",
				FilesToCreate:       []string{"new_file.go"},
				FilesToModify:       []string{"existing.go"},
				AcceptanceCriteria:  []string{"Code compiles", "Tests pass"},
			},
			wantContains: []string{
				"TEST-001",
				"Test Feature",
				"Implement test feature",
				"feature",
				"medium",
				"new_file.go",
				"existing.go",
				"Code compiles",
				"Tests pass",
				"/test/project",
			},
			wantNotContain: []string{},
		},
		{
			name: "ticket without files to create",
			ticket: &ticket.Ticket{
				ID:          "TEST-002",
				Title:       "Refactor",
				Description: "Refactor code",
				Type:        ticket.TypeRefactor,
			},
			wantContains: []string{
				"TEST-002",
				"Refactor",
			},
			wantNotContain: []string{
				"需要建立的檔案",
			},
		},
		{
			name: "ticket without acceptance criteria",
			ticket: &ticket.Ticket{
				ID:          "TEST-003",
				Title:       "Quick Fix",
				Description: "Fix bug",
			},
			wantContains: []string{
				"TEST-003",
				"Quick Fix",
			},
			wantNotContain: []string{
				"驗收標準",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := ca.buildPrompt(tt.ticket)

			for _, want := range tt.wantContains {
				if !strings.Contains(prompt, want) {
					t.Errorf("buildPrompt() should contain %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(prompt, notWant) {
					t.Errorf("buildPrompt() should not contain %q", notWant)
				}
			}
		})
	}
}

func TestReviewAgent_buildReviewPrompt(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	files := []string{"file1.go", "file2.go", "file3.go"}
	prompt := ra.buildReviewPrompt(files)

	// Verify prompt contains essential elements
	expectedContents := []string{
		"/test/project",
		"file1.go",
		"file2.go",
		"file3.go",
		"程式碼品質",
		"bugs",
		"效能",
		"安全性",
		"測試",
		"APPROVED",
		"CHANGES_REQUESTED",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("buildReviewPrompt() should contain %q", expected)
		}
	}
}

func TestReviewAgent_parseReviewResult(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	tests := []struct {
		name       string
		output     string
		wantStatus string
	}{
		{
			name:       "approved result",
			output:     "Code looks good. Status: APPROVED\n摘要\nAll checks passed",
			wantStatus: "APPROVED",
		},
		{
			name:       "changes requested",
			output:     "Found issues. Status: CHANGES_REQUESTED\nSummary\nNeeds fixes",
			wantStatus: "CHANGES_REQUESTED",
		},
		{
			name:       "unknown status",
			output:     "Some random output without clear status",
			wantStatus: "UNKNOWN",
		},
		{
			name:       "approved case insensitive",
			output:     "The code is approved and ready to merge",
			wantStatus: "APPROVED",
		},
		{
			name:       "explicit status line 狀態",
			output:     "狀態: APPROVED\n摘要: 通過審查",
			wantStatus: "APPROVED",
		},
		{
			name:       "explicit status line CHANGES_REQUESTED wins over APPROVED in text",
			output:     "狀態: CHANGES_REQUESTED\nCode is not approved yet.",
			wantStatus: "CHANGES_REQUESTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ra.parseReviewResult(tt.output)

			if result.Status != tt.wantStatus {
				t.Errorf("parseReviewResult() Status = %v, want %v", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestReviewAgent_parseReviewResult_Summary(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	output := `Review completed.

摘要
All tests passing and code is clean.

Issues: None found`

	result := ra.parseReviewResult(output)

	if result.Summary != "All tests passing and code is clean." {
		t.Errorf("parseReviewResult() Summary = %v, want 'All tests passing and code is clean.'", result.Summary)
	}
}

func TestReviewAgent_parseReviewResult_SummaryAndIssuesSuggestions(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	tests := []struct {
		name            string
		output          string
		wantSummary     string
		wantIssues      []string
		wantSuggestions []string
	}{
		{
			name: "inline summary 摘要:",
			output: `狀態: APPROVED
摘要: 程式碼品質良好，可合併。`,
			wantSummary: "程式碼品質良好，可合併。",
		},
		{
			name: "issues and suggestions list",
			output: `狀態: CHANGES_REQUESTED
摘要: 需要修改

問題:
- 缺少錯誤處理
- 變數命名不清晰

建議:
- 加上 err 檢查
- 使用更具描述性的名稱`,
			wantSummary: "需要修改",
			wantIssues: []string{"缺少錯誤處理", "變數命名不清晰"},
			wantSuggestions: []string{"加上 err 檢查", "使用更具描述性的名稱"},
		},
		{
			name: "numbered list and English headers",
			output: `Status: CHANGES_REQUESTED
Summary: Fix required

Issues:
1. Missing test
2. Naming

Suggestions:
* Add unit test
* Rename variable`,
			wantSummary:     "Fix required",
			wantIssues:      []string{"Missing test", "Naming"},
			wantSuggestions: []string{"Add unit test", "Rename variable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ra.parseReviewResult(tt.output)
			if tt.wantSummary != "" && result.Summary != tt.wantSummary {
				t.Errorf("parseReviewResult() Summary = %q, want %q", result.Summary, tt.wantSummary)
			}
			if len(tt.wantIssues) > 0 {
				if len(result.Issues) != len(tt.wantIssues) {
					t.Errorf("parseReviewResult() Issues len = %d, want %d", len(result.Issues), len(tt.wantIssues))
				}
				for i, w := range tt.wantIssues {
					if i < len(result.Issues) && result.Issues[i] != w {
						t.Errorf("parseReviewResult() Issues[%d] = %q, want %q", i, result.Issues[i], w)
					}
				}
			}
			if len(tt.wantSuggestions) > 0 {
				if len(result.Suggestions) != len(tt.wantSuggestions) {
					t.Errorf("parseReviewResult() Suggestions len = %d, want %d", len(result.Suggestions), len(tt.wantSuggestions))
				}
				for i, w := range tt.wantSuggestions {
					if i < len(result.Suggestions) && result.Suggestions[i] != w {
						t.Errorf("parseReviewResult() Suggestions[%d] = %q, want %q", i, result.Suggestions[i], w)
					}
				}
			}
		})
	}
}

func TestAnalyzeAgent_buildAnalyzePrompt(t *testing.T) {
	aa := NewAnalyzeAgent(nil, "/test/project")

	tests := []struct {
		name         string
		scope        AnalyzeScope
		wantContains []string
	}{
		{
			name: "all scopes",
			scope: AnalyzeScope{
				Performance: true,
				Refactor:    true,
				Security:    true,
				Test:        true,
				Docs:        true,
			},
			wantContains: []string{
				"效能問題",
				"重構建議",
				"安全性問題",
				"測試覆蓋",
				"文件缺失",
			},
		},
		{
			name: "performance only",
			scope: AnalyzeScope{
				Performance: true,
			},
			wantContains: []string{
				"效能問題",
			},
		},
		{
			name: "security only",
			scope: AnalyzeScope{
				Security: true,
			},
			wantContains: []string{
				"安全性問題",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := aa.buildAnalyzePrompt(tt.scope)

			for _, want := range tt.wantContains {
				if !strings.Contains(prompt, want) {
					t.Errorf("buildAnalyzePrompt() should contain %q", want)
				}
			}
		})
	}
}

func TestAllScopes(t *testing.T) {
	scope := AllScopes()

	if !scope.Performance {
		t.Error("AllScopes().Performance should be true")
	}
	if !scope.Refactor {
		t.Error("AllScopes().Refactor should be true")
	}
	if !scope.Security {
		t.Error("AllScopes().Security should be true")
	}
	if !scope.Test {
		t.Error("AllScopes().Test should be true")
	}
	if !scope.Docs {
		t.Error("AllScopes().Docs should be true")
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		want   AnalyzeScope
	}{
		{
			name:   "all scope",
			scopes: []string{"all"},
			want:   AllScopes(),
		},
		{
			name:   "performance variants",
			scopes: []string{"performance"},
			want:   AnalyzeScope{Performance: true},
		},
		{
			name:   "perf shorthand",
			scopes: []string{"perf"},
			want:   AnalyzeScope{Performance: true},
		},
		{
			name:   "security variants",
			scopes: []string{"sec"},
			want:   AnalyzeScope{Security: true},
		},
		{
			name:   "multiple scopes",
			scopes: []string{"performance", "security", "test"},
			want:   AnalyzeScope{Performance: true, Security: true, Test: true},
		},
		{
			name:   "case insensitive",
			scopes: []string{"PERFORMANCE", "Security", "TEST"},
			want:   AnalyzeScope{Performance: true, Security: true, Test: true},
		},
		{
			name:   "empty scopes",
			scopes: []string{},
			want:   AnalyzeScope{},
		},
		{
			name:   "unknown scope ignored",
			scopes: []string{"unknown", "test"},
			want:   AnalyzeScope{Test: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseScopes(tt.scopes)

			if got.Performance != tt.want.Performance {
				t.Errorf("ParseScopes().Performance = %v, want %v", got.Performance, tt.want.Performance)
			}
			if got.Refactor != tt.want.Refactor {
				t.Errorf("ParseScopes().Refactor = %v, want %v", got.Refactor, tt.want.Refactor)
			}
			if got.Security != tt.want.Security {
				t.Errorf("ParseScopes().Security = %v, want %v", got.Security, tt.want.Security)
			}
			if got.Test != tt.want.Test {
				t.Errorf("ParseScopes().Test = %v, want %v", got.Test, tt.want.Test)
			}
			if got.Docs != tt.want.Docs {
				t.Errorf("ParseScopes().Docs = %v, want %v", got.Docs, tt.want.Docs)
			}
		})
	}
}

func TestCommitAgent_buildCommitPrompt(t *testing.T) {
	ca := NewCommitAgent(nil, "/test/project")

	prompt := ca.buildCommitPrompt("TICKET-001", "Add feature", "M file.go\nA new.go")

	expectedContents := []string{
		"/test/project",
		"TICKET-001",
		"Add feature",
		"M file.go",
		"A new.go",
		"Conventional Commits",
		"git add",
		"git commit",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("buildCommitPrompt() should contain %q", expected)
		}
	}
}

func TestTestAgent_buildTestPrompt(t *testing.T) {
	ta := NewTestAgent(nil, "/test/project")

	prompt := ta.buildTestPrompt()

	expectedContents := []string{
		"/test/project",
		"go test",
		"npm test",
		"pytest",
		"mvn test",
		"gradle test",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("buildTestPrompt() should contain %q", expected)
		}
	}
}

func TestTestAgent_parseTestResult(t *testing.T) {
	ta := NewTestAgent(nil, "/test/project")

	tests := []struct {
		name        string
		output      string
		wantPassed  int
		wantFailed  int
		wantSkipped int
		wantSummary string
	}{
		{
			name:        "empty output",
			output:      "",
			wantPassed:  0,
			wantFailed:  0,
			wantSkipped: 0,
			wantSummary: "",
		},
		{
			name: "go test ok/FAIL package lines",
			output: `ok  	github.com/foo/bar	0.123s
ok  	github.com/foo/baz	0.056s
FAIL	github.com/foo/qux	0.200s`,
			wantPassed:  2,
			wantFailed:  1,
			wantSkipped: 0,
			wantSummary: "2 passed, 1 failed",
		},
		{
			name: "go test --- PASS/--- FAIL lines",
			output: `--- PASS: TestFoo (0.00s)
--- PASS: TestBar (0.01s)
--- FAIL: TestBaz (0.00s)
--- PASS: TestQux (0.00s)`,
			wantPassed:  3,
			wantFailed:  1,
			wantSkipped: 0,
			wantSummary: "3 passed, 1 failed",
		},
		{
			name: "pytest passed only",
			output: `======================== 3 passed in 0.12s ========================`,
			wantPassed:  3,
			wantFailed:  0,
			wantSkipped: 0,
			wantSummary: "3 passed",
		},
		{
			name: "pytest failed and passed",
			output: `2 failed, 5 passed in 0.45s`,
			wantPassed:  5,
			wantFailed:  2,
			wantSkipped: 0,
			wantSummary: "5 passed, 2 failed",
		},
		{
			name: "pytest with skipped",
			output: `1 failed, 2 passed, 1 skipped in 0.30s`,
			wantPassed:  2,
			wantFailed:  1,
			wantSkipped: 1,
			wantSummary: "2 passed, 1 failed, 1 skipped",
		},
		{
			name: "pytest with error count",
			output: `1 error, 2 passed, 1 failed in 0.20s`,
			wantPassed:  2,
			wantFailed:  2, // failed + error
			wantSkipped: 0,
			wantSummary: "2 passed, 2 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ta.parseTestResult(tt.output)
			if result.Passed != tt.wantPassed {
				t.Errorf("parseTestResult() Passed = %d, want %d", result.Passed, tt.wantPassed)
			}
			if result.Failed != tt.wantFailed {
				t.Errorf("parseTestResult() Failed = %d, want %d", result.Failed, tt.wantFailed)
			}
			if result.Skipped != tt.wantSkipped {
				t.Errorf("parseTestResult() Skipped = %d, want %d", result.Skipped, tt.wantSkipped)
			}
			if tt.wantSummary != "" && result.Summary != tt.wantSummary {
				t.Errorf("parseTestResult() Summary = %q, want %q", result.Summary, tt.wantSummary)
			}
		})
	}
}

func TestPlanningAgent_buildPlanningPrompt(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")

	prompt := pa.buildPlanningPrompt("# Milestone content", "/test/milestone.md", "/test/output.json")

	expectedContents := []string{
		"/test/milestone.md",
		"/test/output.json",
		"feature",
		"test",
		"refactor",
		"docs",
		"bugfix",
		"dependencies",
		"priority",
		"acceptance_criteria",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(prompt, expected) {
			t.Errorf("buildPlanningPrompt() should contain %q", expected)
		}
	}
}

func TestPlanningAgent_mapToTicket(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")

	tests := []struct {
		name     string
		data     map[string]interface{}
		wantNil  bool
		wantID   string
		wantType ticket.Type
	}{
		{
			name: "valid complete data",
			data: map[string]interface{}{
				"id":                   "T1",
				"title":                "Test",
				"description":          "Description",
				"type":                 "feature",
				"priority":             float64(1),
				"estimated_complexity": "high",
				"dependencies":         []interface{}{"D1", "D2"},
				"acceptance_criteria":  []interface{}{"C1"},
				"files_to_create":      []interface{}{"new.go"},
				"files_to_modify":      []interface{}{"old.go"},
			},
			wantNil:  false,
			wantID:   "T1",
			wantType: ticket.TypeFeature,
		},
		{
			name: "missing id",
			data: map[string]interface{}{
				"title": "Test",
			},
			wantNil: true,
		},
		{
			name: "missing title",
			data: map[string]interface{}{
				"id": "T1",
			},
			wantNil: true,
		},
		{
			name: "minimal valid data",
			data: map[string]interface{}{
				"id":    "T2",
				"title": "Minimal",
			},
			wantNil: false,
			wantID:  "T2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pa.mapToTicket(tt.data)

			if tt.wantNil {
				if result != nil {
					t.Errorf("mapToTicket() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("mapToTicket() = nil, want non-nil")
			}

			if result.ID != tt.wantID {
				t.Errorf("mapToTicket().ID = %v, want %v", result.ID, tt.wantID)
			}

			if tt.wantType != "" && result.Type != tt.wantType {
				t.Errorf("mapToTicket().Type = %v, want %v", result.Type, tt.wantType)
			}
		})
	}
}

func TestInitAgent_parseQuestions(t *testing.T) {
	ia := NewInitAgent(nil, "/test/project", "/test/docs")

	tests := []struct {
		name      string
		output    string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "valid JSON",
			output:    `Some text {"questions": ["Q1", "Q2", "Q3"]} more text`,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "JSON at start",
			output:    `{"questions": ["Q1", "Q2"]}`,
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no JSON",
			output:    `Some text without JSON`,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "invalid JSON",
			output:    `{invalid json}`,
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questions, err := ia.parseQuestions(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Error("parseQuestions() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parseQuestions() error = %v", err)
				return
			}

			if len(questions) != tt.wantCount {
				t.Errorf("parseQuestions() returned %d questions, want %d", len(questions), tt.wantCount)
			}
		})
	}
}

func TestInitAgent_defaultQuestions(t *testing.T) {
	ia := NewInitAgent(nil, "/test/project", "/test/docs")

	questions := ia.defaultQuestions()

	if len(questions) == 0 {
		t.Error("defaultQuestions() should return non-empty slice")
	}

	// Verify questions are about expected topics
	topics := []string{"語言", "使用者", "功能", "效能", "格式"}
	foundTopics := 0

	for _, q := range questions {
		for _, topic := range topics {
			if strings.Contains(q, topic) {
				foundTopics++
				break
			}
		}
	}

	if foundTopics < 3 {
		t.Errorf("defaultQuestions() should cover at least 3 topics, found %d", foundTopics)
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []interface{}
		want  []string
	}{
		{
			name:  "string elements",
			input: []interface{}{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "mixed types - only strings extracted",
			input: []interface{}{"a", 123, "b", true},
			want:  []string{"a", "b"},
		},
		{
			name:  "empty slice",
			input: []interface{}{},
			want:  []string{},
		},
		{
			name:  "no strings",
			input: []interface{}{123, 456},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonutil.ToStringSlice(tt.input)

			if len(got) != len(tt.want) {
				t.Errorf("ToStringSlice() = %v, want %v", got, tt.want)
				return
			}

			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("ToStringSlice()[%d] = %v, want %v", i, got[i], v)
				}
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "existing key",
			m:    map[string]interface{}{"key": "value"},
			key:  "key",
			want: "value",
		},
		{
			name: "missing key",
			m:    map[string]interface{}{"other": "value"},
			key:  "key",
			want: "",
		},
		{
			name: "non-string value",
			m:    map[string]interface{}{"key": 123},
			key:  "key",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonutil.GetString(tt.m, tt.key)

			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}
