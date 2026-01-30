package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

func TestCodingAgent_buildPrompt_outputFormat(t *testing.T) {
	ca := NewCodingAgent(nil, "/test/project")
	tkt := &ticket.Ticket{
		ID:                  "T-001",
		Title:               "標題",
		Description:         "描述",
		Type:                ticket.TypeFeature,
		EstimatedComplexity: "medium",
		FilesToCreate:       []string{"a.go"},
		FilesToModify:       []string{"b.go"},
		AcceptanceCriteria:  []string{"標準1"},
	}

	prompt := ca.buildPrompt(tkt)

	// 必須包含的區塊標題與固定步驟（使用 i18n 常數以與 agent 一致）
	wantSections := []string{
		strings.TrimSuffix(i18n.AgentCodingIntro, "\n\n"),
		"專案根目錄: /test/project",
		strings.TrimSuffix(i18n.AgentCodingSectionTicket, "\n"),
		"- ID: T-001",
		"- 標題: 標題",
		"- 描述: 描述",
		"- 類型: feature",
		"- 複雜度: medium",
		strings.TrimSuffix(i18n.AgentCodingSectionFilesCreate, "\n"),
		"- a.go",
		strings.TrimSuffix(i18n.AgentCodingSectionFilesModify, "\n"),
		"- b.go",
		strings.TrimSuffix(i18n.AgentCodingSectionAcceptance, "\n"),
		"- 標準1",
		"## 請執行以下步驟:",
		"1. 閱讀相關的現有程式碼",
		"完成後，說明你所做的變更。",
	}
	for _, want := range wantSections {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildPrompt() should contain %q", want)
		}
	}
}

func TestAnalyzeAgent_parseIssues(t *testing.T) {
	aa := NewAnalyzeAgent(nil, "/test/project")

	tests := []struct {
		name      string
		data      map[string]interface{}
		wantErr   bool
		wantCount int
		wantID    string // first issue ID if any
	}{
		{
			name: "valid issues",
			data: map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{
						"id": "ISSUE-001", "category": "performance", "severity": "HIGH",
						"title": "N+1", "description": "desc", "location": "a.go:1", "suggestion": "fix",
					},
					map[string]interface{}{
						"id": "ISSUE-002", "category": "refactor", "severity": "MED",
						"title": "Long method", "description": "d", "location": "b.go:2", "suggestion": "split",
					},
				},
			},
			wantErr:   false,
			wantCount: 2,
			wantID:    "ISSUE-001",
		},
		{
			name: "issues missing id or title are skipped",
			data: map[string]interface{}{
				"issues": []interface{}{
					map[string]interface{}{"id": "X", "title": "OK"},
					map[string]interface{}{"id": "", "title": "NoID"},
					map[string]interface{}{"id": "Y", "title": ""},
				},
			},
			wantErr:   false,
			wantCount: 1,
			wantID:    "X",
		},
		{
			name: "invalid issues key - not slice",
			data: map[string]interface{}{
				"issues": "not-a-slice",
			},
			wantErr: true,
		},
		{
			name: "missing issues key",
			data: map[string]interface{}{
				"other": []interface{}{},
			},
			wantErr: true,
		},
		{
			name:      "empty issues slice",
			data:      map[string]interface{}{"issues": []interface{}{}},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "non-map element skipped",
			data: map[string]interface{}{
				"issues": []interface{}{
					"string",
					123,
					map[string]interface{}{"id": "I1", "title": "T1"},
				},
			},
			wantErr:   false,
			wantCount: 1,
			wantID:    "I1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			il, err := aa.parseIssues(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Error("parseIssues() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseIssues() error = %v", err)
				return
			}
			if il == nil {
				t.Fatal("parseIssues() returned nil IssueList")
			}
			count := il.Count()
			if count != tt.wantCount {
				t.Errorf("parseIssues() count = %d, want %d", count, tt.wantCount)
			}
			if tt.wantCount > 0 && tt.wantID != "" {
				issues := il.Issues
				if len(issues) > 0 && issues[0].ID != tt.wantID {
					t.Errorf("parseIssues() first ID = %q, want %q", issues[0].ID, tt.wantID)
				}
			}
		})
	}
}

func TestAnalyzeAgent_createMockIssues_dryRun(t *testing.T) {
	aa := NewAnalyzeAgent(nil, "/test/project")

	tests := []struct {
		name        string
		scope       AnalyzeScope
		wantCount   int
		wantCat     string // category of first issue
		wantContain string // string that should appear in title or description
	}{
		{
			name:        "performance only",
			scope:       AnalyzeScope{Performance: true},
			wantCount:   1,
			wantCat:     "performance",
			wantContain: "N+1",
		},
		{
			name:        "refactor only",
			scope:       AnalyzeScope{Refactor: true},
			wantCount:   1,
			wantCat:     "refactor",
			wantContain: "過長方法",
		},
		{
			name:        "security only",
			scope:       AnalyzeScope{Security: true},
			wantCount:   1,
			wantCat:     "security",
			wantContain: "硬編碼",
		},
		{
			name:      "empty scope returns empty list",
			scope:     AnalyzeScope{},
			wantCount: 0,
		},
		{
			name:        "performance and refactor",
			scope:       AnalyzeScope{Performance: true, Refactor: true},
			wantCount:   2,
			wantCat:     "performance",
			wantContain: "N+1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			il := aa.createMockIssues(tt.scope)
			if il == nil {
				t.Fatal("createMockIssues() returned nil")
			}
			if il.Count() != tt.wantCount {
				t.Errorf("createMockIssues() count = %d, want %d", il.Count(), tt.wantCount)
			}
			if tt.wantCount > 0 && tt.wantCat != "" {
				issues := il.Issues
				if issues[0].Category != tt.wantCat {
					t.Errorf("createMockIssues() first category = %q, want %q", issues[0].Category, tt.wantCat)
				}
				if tt.wantContain != "" && !strings.Contains(issues[0].Title, tt.wantContain) && !strings.Contains(issues[0].Description, tt.wantContain) {
					t.Errorf("createMockIssues() first issue should contain %q", tt.wantContain)
				}
			}
		})
	}
}

func TestAnalyzeAgent_Analyze_dryRunReturnsMockIssues(t *testing.T) {
	dir := t.TempDir()
	caller := NewCaller("cursor", false, "text", "")
	caller.SetDryRun(true)
	aa := NewAnalyzeAgent(caller, dir)
	ctx := context.Background()

	il, err := aa.Analyze(ctx, AnalyzeScope{Performance: true, Refactor: true})
	if err != nil {
		t.Fatalf("Analyze(dry run) error = %v", err)
	}
	if il == nil {
		t.Fatal("Analyze(dry run) returned nil IssueList")
	}
	if il.Count() < 2 {
		t.Errorf("Analyze(dry run) want at least 2 mock issues, got %d", il.Count())
	}
}
