package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

func TestPlanningAgent_buildPlanningPrompt_outputFormat(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")
	content := "# Milestone"
	milestoneFile := "/path/milestone.md"
	outputFile := "/test/tickets/generated-tickets.json"

	prompt := pa.buildPlanningPrompt(content, milestoneFile, outputFile)

	wantContains := []string{
		"你是一個專案規劃 Agent",
		"請讀取檔案 " + milestoneFile,
		fmt.Sprintf(i18n.AgentWriteJSONToFile, outputFile),
		"id",
		"title",
		"description",
		"type",
		"priority",
		"estimated_complexity",
		"dependencies",
		"acceptance_criteria",
		"files_to_create",
		"files_to_modify",
		`{"tickets": [...]}`,
	}
	for _, want := range wantContains {
		if !strings.Contains(prompt, want) {
			t.Errorf("buildPlanningPrompt() should contain %q", want)
		}
	}
}

func TestPlanningAgent_parseTickets(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")

	tests := []struct {
		name      string
		data      map[string]interface{}
		wantErr   bool
		wantCount int
		wantID    string
	}{
		{
			name: "valid tickets",
			data: map[string]interface{}{
				"tickets": []interface{}{
					map[string]interface{}{"id": "T1", "title": "Title 1", "description": "Desc 1"},
					map[string]interface{}{"id": "T2", "title": "Title 2", "description": "Desc 2"},
				},
			},
			wantErr:   false,
			wantCount: 2,
			wantID:    "T1",
		},
		{
			name: "missing id or title skips ticket",
			data: map[string]interface{}{
				"tickets": []interface{}{
					map[string]interface{}{"id": "T1", "title": "OK"},
					map[string]interface{}{"id": "", "title": "NoID"},
					map[string]interface{}{"id": "T2", "title": ""},
				},
			},
			wantErr:   false,
			wantCount: 1,
			wantID:    "T1",
		},
		{
			name: "invalid tickets key - not slice",
			data: map[string]interface{}{
				"tickets": "not-a-slice",
			},
			wantErr: true,
		},
		{
			name: "missing tickets key",
			data: map[string]interface{}{
				"other": []interface{}{},
			},
			wantErr: true,
		},
		{
			name:      "empty tickets slice",
			data:      map[string]interface{}{"tickets": []interface{}{}},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "non-map element skipped",
			data: map[string]interface{}{
				"tickets": []interface{}{
					"string",
					map[string]interface{}{"id": "T1", "title": "T1"},
				},
			},
			wantErr:   false,
			wantCount: 1,
			wantID:    "T1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, err := pa.parseTickets(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Error("parseTickets() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseTickets() error = %v", err)
				return
			}
			if len(tickets) != tt.wantCount {
				t.Errorf("parseTickets() count = %d, want %d", len(tickets), tt.wantCount)
			}
			if tt.wantCount > 0 && tt.wantID != "" && tickets[0].ID != tt.wantID {
				t.Errorf("parseTickets() first ID = %q, want %q", tickets[0].ID, tt.wantID)
			}
		})
	}
}

func TestPlanningAgent_parseTickets_fullFields(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")
	data := map[string]interface{}{
		"tickets": []interface{}{
			map[string]interface{}{
				"id":                   "T-001",
				"title":                "Feature",
				"description":         "Desc",
				"type":                 "feature",
				"priority":             float64(1),
				"estimated_complexity": "high",
				"dependencies":         []interface{}{"D1"},
				"acceptance_criteria":  []interface{}{"C1", "C2"},
				"files_to_create":      []interface{}{"new.go"},
				"files_to_modify":      []interface{}{"old.go"},
			},
		},
	}

	tickets, err := pa.parseTickets(data)
	if err != nil {
		t.Fatalf("parseTickets() error = %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("parseTickets() count = %d, want 1", len(tickets))
	}
	t0 := tickets[0]
	if t0.ID != "T-001" || t0.Title != "Feature" || t0.Description != "Desc" {
		t.Errorf("parseTickets() ticket = %+v", t0)
	}
	if t0.Type != ticket.TypeFeature {
		t.Errorf("parseTickets() Type = %v, want feature", t0.Type)
	}
	if t0.Priority != 1 {
		t.Errorf("parseTickets() Priority = %d, want 1", t0.Priority)
	}
	if t0.EstimatedComplexity != "high" {
		t.Errorf("parseTickets() EstimatedComplexity = %q, want high", t0.EstimatedComplexity)
	}
	if len(t0.Dependencies) != 1 || t0.Dependencies[0] != "D1" {
		t.Errorf("parseTickets() Dependencies = %v", t0.Dependencies)
	}
	if len(t0.AcceptanceCriteria) != 2 {
		t.Errorf("parseTickets() AcceptanceCriteria len = %d, want 2", len(t0.AcceptanceCriteria))
	}
	if len(t0.FilesToCreate) != 1 || t0.FilesToCreate[0] != "new.go" {
		t.Errorf("parseTickets() FilesToCreate = %v", t0.FilesToCreate)
	}
	if len(t0.FilesToModify) != 1 || t0.FilesToModify[0] != "old.go" {
		t.Errorf("parseTickets() FilesToModify = %v", t0.FilesToModify)
	}
}

func TestPlanningAgent_createMockTickets_dryRun(t *testing.T) {
	pa := NewPlanningAgent(nil, "/test/project", "/test/tickets")

	tickets := pa.createMockTickets()

	if tickets == nil {
		t.Fatal("createMockTickets() returned nil")
	}
	if len(tickets) < 3 {
		t.Errorf("createMockTickets() count = %d, want at least 3", len(tickets))
	}
	// 第一張應為設定專案結構
	if tickets[0].ID != "TICKET-001-setup" || tickets[0].Title != "設定專案結構" {
		t.Errorf("createMockTickets()[0] = %s %s", tickets[0].ID, tickets[0].Title)
	}
	// 依賴關係
	if len(tickets[1].Dependencies) == 0 || tickets[1].Dependencies[0] != "TICKET-001-setup" {
		t.Errorf("createMockTickets()[1].Dependencies = %v", tickets[1].Dependencies)
	}
	// 類型
	if tickets[0].Type != ticket.TypeFeature || tickets[2].Type != ticket.TypeTest {
		t.Errorf("createMockTickets() types: %v %v", tickets[0].Type, tickets[2].Type)
	}
}

func TestPlanningAgent_Plan_dryRunReturnsMockTickets(t *testing.T) {
	// 建立臨時 milestone 檔案
	dir := t.TempDir()
	milestonePath := dir + "/milestone.md"
	if err := writeFile(milestonePath, "# Test milestone"); err != nil {
		t.Fatalf("write milestone: %v", err)
	}

	caller := NewCaller("cursor", false, "text", "")
	caller.SetDryRun(true)
	pa := NewPlanningAgent(caller, "/test/project", dir)
	ctx := context.Background()

	tickets, err := pa.Plan(ctx, milestonePath)
	if err != nil {
		t.Fatalf("Plan(dry run) error = %v", err)
	}
	if tickets == nil {
		t.Fatal("Plan(dry run) returned nil")
	}
	if len(tickets) < 3 {
		t.Errorf("Plan(dry run) want at least 3 mock tickets, got %d", len(tickets))
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
