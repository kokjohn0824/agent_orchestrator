package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

func Test_mergeStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		new      []string
		want     []string
	}{
		{
			name:     "both empty",
			existing: nil,
			new:      nil,
			want:     []string{},
		},
		{
			name:     "existing only",
			existing: []string{"a", "b"},
			new:      nil,
			want:     []string{"a", "b"},
		},
		{
			name:     "new only",
			existing: nil,
			new:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "merge with no overlap",
			existing: []string{"a", "b"},
			new:      []string{"c", "d"},
			want:     []string{"a", "b", "c", "d"},
		},
		{
			name:     "merge with overlap dedupes",
			existing: []string{"a", "b"},
			new:      []string{"b", "c", "a"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "new contains empty string skips",
			existing: []string{"a"},
			new:      []string{"", "b", ""},
			want:     []string{"a", "b"},
		},
		{
			name:     "new contains duplicate skips",
			existing: []string{"a"},
			new:      []string{"b", "b", "c", "b"},
			want:     []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeStringSlices(tt.existing, tt.new)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeStringSlices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnhanceAgent_buildPrompt_outputFormat(t *testing.T) {
	ea := NewEnhanceAgent(nil, "/test/project")
	tkt := &ticket.Ticket{
		ID:                  "T-001",
		Title:               "標題",
		Description:         "描述",
		Type:                ticket.TypeFeature,
		Priority:            1,
		Dependencies:        []string{"D1"},
		AcceptanceCriteria:  []string{"標準1"},
	}

	prompt := ea.buildPrompt(tkt)

	wantContains := []string{
		"你是一個專案分析專家",
		"專案目錄: /test/project",
		strings.TrimSuffix(i18n.AgentEnhanceSection, "\n"),
		"- ID: T-001",
		"- 標題: 標題",
		"- 類型: feature",
		"- 優先級: P1",
		"- 描述: 描述",
		"- 依賴: D1",
		"- 驗收條件:",
		"  - 標準1",
		"請以 JSON 格式輸出分析結果",
		"description",
		"estimated_complexity",
		"acceptance_criteria",
		"files_to_create",
		"files_to_modify",
		"implementation_hints",
		".tickets/enhance-result.json",
	}
	for _, want := range wantContains {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildPrompt() should contain %q", want)
		}
	}
}

func TestEnhanceAgent_applyEnhancements(t *testing.T) {
	ea := NewEnhanceAgent(nil, "/test/project")
	base := &ticket.Ticket{
		ID:           "T-001",
		Title:        "Title",
		Description:  "Original",
		Type:         ticket.TypeFeature,
		Priority:     1,
		AcceptanceCriteria: []string{"C1"},
		FilesToCreate: []string{"a.go"},
		FilesToModify: []string{"b.go"},
	}

	tests := []struct {
		name     string
		data     map[string]interface{}
		wantDesc string // 應包含的 description 片段
		wantCrit int    // 驗收條件數量
		wantMod  int    // files_to_modify 數量
	}{
		{
			name: "merge description and criteria",
			data: map[string]interface{}{
				"description":         "AI 補充",
				"estimated_complexity": "high",
				"acceptance_criteria":  []interface{}{"C2", "C3"},
				"files_to_create":      []interface{}{"c.go"},
				"files_to_modify":      []interface{}{"d.go"},
			},
			wantDesc: "AI 補充說明",
			wantCrit: 3, // C1 + C2, C3 (dedup if C2/C3)
			wantMod:  2, // b.go + d.go
		},
		{
			name: "empty data leaves ticket unchanged",
			data: map[string]interface{}{},
			wantDesc: "Original",
			wantCrit: 1,
			wantMod:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enhanced, err := ea.applyEnhancements(base, tt.data)
			if err != nil {
				t.Fatalf("applyEnhancements() error = %v", err)
			}
			if enhanced == nil {
				t.Fatal("applyEnhancements() returned nil")
			}
			if !strings.Contains(enhanced.Description, tt.wantDesc) && enhanced.Description != tt.wantDesc {
				t.Errorf("applyEnhancements() Description = %q, want to contain or equal %q", enhanced.Description, tt.wantDesc)
			}
			if len(enhanced.AcceptanceCriteria) < tt.wantCrit {
				t.Errorf("applyEnhancements() AcceptanceCriteria len = %d, want at least %d", len(enhanced.AcceptanceCriteria), tt.wantCrit)
			}
			if len(enhanced.FilesToModify) < tt.wantMod {
				t.Errorf("applyEnhancements() FilesToModify len = %d, want at least %d", len(enhanced.FilesToModify), tt.wantMod)
			}
		})
	}
}

func TestEnhanceAgent_createMockEnhanced_dryRun(t *testing.T) {
	ea := NewEnhanceAgent(nil, "/test/project")
	tkt := &ticket.Ticket{
		ID:           "T-001",
		Title:        "Title",
		Description:  "Desc",
		Type:         ticket.TypeFeature,
		Priority:     1,
		AcceptanceCriteria: []string{"C1"},
		FilesToModify: []string{"a.go"},
	}

	enhanced := ea.createMockEnhanced(tkt)

	if enhanced == nil {
		t.Fatal("createMockEnhanced() returned nil")
	}
	if enhanced.ID != tkt.ID || enhanced.Title != tkt.Title {
		t.Errorf("createMockEnhanced() ID/Title = %s %s", enhanced.ID, enhanced.Title)
	}
	if enhanced.EstimatedComplexity != "medium" {
		t.Errorf("createMockEnhanced() EstimatedComplexity = %q, want medium", enhanced.EstimatedComplexity)
	}
	if len(enhanced.AcceptanceCriteria) == 0 {
		t.Error("createMockEnhanced() should add default acceptance criteria")
	}
	if len(enhanced.FilesToModify) == 0 {
		t.Error("createMockEnhanced() should add default FilesToModify when empty")
	}
	// 空描述時應補上 [DRY RUN] 說明
	tktEmptyDesc := &ticket.Ticket{ID: "T-002", Title: "T2", Description: ""}
	enhanced2 := ea.createMockEnhanced(tktEmptyDesc)
	if enhanced2.Description != "[DRY RUN] AI 會根據專案結構分析並補充描述" {
		t.Errorf("createMockEnhanced(empty desc) Description = %q", enhanced2.Description)
	}
}

func TestEnhanceAgent_Enhance_dryRunReturnsMockEnhanced(t *testing.T) {
	dir := t.TempDir()
	caller := NewCaller("cursor", false, "text", "")
	caller.SetDryRun(true)
	ea := NewEnhanceAgent(caller, dir)
	ctx := context.Background()
	tkt := ticket.NewTicket("T-001", "Title", "Desc")

	enhanced, err := ea.Enhance(ctx, tkt)
	if err != nil {
		t.Fatalf("Enhance(dry run) error = %v", err)
	}
	if enhanced == nil {
		t.Fatal("Enhance(dry run) returned nil")
	}
	if enhanced.ID != tkt.ID {
		t.Errorf("Enhance(dry run) ID = %q, want %q", enhanced.ID, tkt.ID)
	}
	if enhanced.EstimatedComplexity != "medium" {
		t.Errorf("Enhance(dry run) EstimatedComplexity = %q, want medium", enhanced.EstimatedComplexity)
	}
}
