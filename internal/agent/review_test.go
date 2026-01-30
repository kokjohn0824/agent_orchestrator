package agent

import (
	"strings"
	"testing"
)

func TestReviewAgent_buildReviewPrompt_outputFormat(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")
	files := []string{"file1.go", "file2.go"}

	prompt := ra.buildReviewPrompt(files)

	wantContains := []string{
		"你是一個程式碼審查 Agent",
		"專案目錄: /test/project",
		"變更的檔案:",
		"- file1.go",
		"- file2.go",
		"請檢查:",
		"程式碼品質",
		"潛在的 bugs",
		"效能",
		"安全性",
		"測試覆蓋率",
		"狀態: APPROVED 或 CHANGES_REQUESTED",
		"摘要:",
		"問題:",
		"建議:",
	}
	for _, want := range wantContains {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildReviewPrompt() should contain %q", want)
		}
	}
}

func TestReviewAgent_parseReviewResult_status(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	tests := []struct {
		name       string
		output     string
		wantStatus string
	}{
		{"explicit 狀態 APPROVED", "狀態: APPROVED\n摘要: 通過", "APPROVED"},
		{"explicit 狀態 CHANGES_REQUESTED", "狀態: CHANGES_REQUESTED\n摘要: 需修改", "CHANGES_REQUESTED"},
		{"Status English", "Status: APPROVED\nSummary: OK", "APPROVED"},
		{"keyword fallback approved", "The code is approved and ready", "APPROVED"},
		{"keyword fallback changes", "Result: CHANGES_REQUESTED due to issues", "CHANGES_REQUESTED"},
		{"no status", "Random text without status", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ra.parseReviewResult(tt.output)
			if result == nil {
				t.Fatal("parseReviewResult() returned nil")
			}
			if result.Status != tt.wantStatus {
				t.Errorf("parseReviewResult() Status = %q, want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestReviewAgent_parseReviewResult_summaryAndLists(t *testing.T) {
	ra := NewReviewAgent(nil, "/test/project")

	output := `狀態: CHANGES_REQUESTED
摘要: 需要修改

問題:
- 缺少錯誤處理
- 變數命名不清晰

建議:
- 加上 err 檢查
- 使用更具描述性的名稱`

	result := ra.parseReviewResult(output)

	if result.Summary != "需要修改" {
		t.Errorf("parseReviewResult() Summary = %q, want 需要修改", result.Summary)
	}
	if len(result.Issues) != 2 {
		t.Errorf("parseReviewResult() Issues len = %d, want 2", len(result.Issues))
	}
	if len(result.Suggestions) != 2 {
		t.Errorf("parseReviewResult() Suggestions len = %d, want 2", len(result.Suggestions))
	}
	if result.Issues[0] != "缺少錯誤處理" || result.Issues[1] != "變數命名不清晰" {
		t.Errorf("parseReviewResult() Issues = %v", result.Issues)
	}
	if result.Suggestions[0] != "加上 err 檢查" || result.Suggestions[1] != "使用更具描述性的名稱" {
		t.Errorf("parseReviewResult() Suggestions = %v", result.Suggestions)
	}
}

func TestParseListSection(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		start       []string
		end         []string
		wantCount   int
		wantFirst   string
	}{
		{
			name: "issues section",
			output: `問題:
- Item one
- Item two`,
			start:     []string{"問題", "issues"},
			end:       []string{"建議", "suggestions"},
			wantCount: 2,
			wantFirst: "Item one",
		},
		{
			name: "numbered list",
			output: `Issues:
1. First
2. Second`,
			start:     []string{"issues"},
			end:       []string{"suggestions"},
			wantCount: 2,
			wantFirst: "First",
		},
		{
			name: "inline after marker",
			output: `問題: 單一項目`,
			start:     []string{"問題"},
			end:       []string{"建議"},
			wantCount: 1,
			wantFirst: "單一項目",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseListSection(tt.output, tt.start, tt.end)
			if len(got) != tt.wantCount {
				t.Errorf("parseListSection() len = %d, want %d", len(got), tt.wantCount)
			}
			if tt.wantCount > 0 && len(got) > 0 && got[0] != tt.wantFirst {
				t.Errorf("parseListSection() first = %q, want %q", got[0], tt.wantFirst)
			}
		})
	}
}
