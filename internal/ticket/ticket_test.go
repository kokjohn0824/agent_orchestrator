package ticket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"pending is valid", StatusPending, true},
		{"in_progress is valid", StatusInProgress, true},
		{"completed is valid", StatusCompleted, true},
		{"failed is valid", StatusFailed, true},
		{"empty is invalid", Status(""), false},
		{"unknown is invalid", Status("unknown"), false},
		{"typo is invalid", Status("pendin"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("Status.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPending, "pending"},
		{StatusInProgress, "in_progress"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		ticketType Type
		want       string
	}{
		{TypeFeature, "feature"},
		{TypeTest, "test"},
		{TypeRefactor, "refactor"},
		{TypeDocs, "docs"},
		{TypeBugfix, "bugfix"},
		{TypePerf, "performance"},
		{TypeSecurity, "security"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ticketType), func(t *testing.T) {
			if got := tt.ticketType.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTicket(t *testing.T) {
	ticket := NewTicket("TEST-001", "Test Title", "Test Description")

	if ticket.ID != "TEST-001" {
		t.Errorf("NewTicket().ID = %v, want TEST-001", ticket.ID)
	}
	if ticket.Title != "Test Title" {
		t.Errorf("NewTicket().Title = %v, want Test Title", ticket.Title)
	}
	if ticket.Description != "Test Description" {
		t.Errorf("NewTicket().Description = %v, want Test Description", ticket.Description)
	}
	if ticket.Status != StatusPending {
		t.Errorf("NewTicket().Status = %v, want pending", ticket.Status)
	}
	if ticket.Type != TypeFeature {
		t.Errorf("NewTicket().Type = %v, want feature", ticket.Type)
	}
	if ticket.Priority != 5 {
		t.Errorf("NewTicket().Priority = %v, want 5", ticket.Priority)
	}
	if ticket.EstimatedComplexity != "medium" {
		t.Errorf("NewTicket().EstimatedComplexity = %v, want medium", ticket.EstimatedComplexity)
	}
	if ticket.CreatedAt.IsZero() {
		t.Error("NewTicket().CreatedAt should not be zero")
	}
	if ticket.Dependencies == nil {
		t.Error("NewTicket().Dependencies should not be nil")
	}
	if ticket.AcceptanceCriteria == nil {
		t.Error("NewTicket().AcceptanceCriteria should not be nil")
	}
}

func TestTicket_StatusTransitions(t *testing.T) {
	t.Run("MarkInProgress", func(t *testing.T) {
		ticket := NewTicket("T1", "Test", "desc")
		ticket.MarkInProgress()

		if ticket.Status != StatusInProgress {
			t.Errorf("MarkInProgress() Status = %v, want in_progress", ticket.Status)
		}
	})

	t.Run("MarkCompleted", func(t *testing.T) {
		ticket := NewTicket("T2", "Test", "desc")
		ticket.MarkInProgress()

		beforeComplete := time.Now()
		ticket.MarkCompleted("Agent output message")

		if ticket.Status != StatusCompleted {
			t.Errorf("MarkCompleted() Status = %v, want completed", ticket.Status)
		}
		if ticket.AgentOutput != "Agent output message" {
			t.Errorf("MarkCompleted() AgentOutput = %v, want 'Agent output message'", ticket.AgentOutput)
		}
		if ticket.CompletedAt == nil {
			t.Error("MarkCompleted() CompletedAt should not be nil")
		}
		if ticket.CompletedAt.Before(beforeComplete) {
			t.Error("MarkCompleted() CompletedAt should be after the call")
		}
	})

	t.Run("MarkFailed with error", func(t *testing.T) {
		ticket := NewTicket("T3", "Test", "desc")
		ticket.MarkInProgress()

		testErr := &testError{msg: "test error message"}
		ticket.MarkFailed(testErr)

		if ticket.Status != StatusFailed {
			t.Errorf("MarkFailed() Status = %v, want failed", ticket.Status)
		}
		if ticket.Error != "test error message" {
			t.Errorf("MarkFailed() Error = %v, want 'test error message'", ticket.Error)
		}
		if ticket.CompletedAt == nil {
			t.Error("MarkFailed() CompletedAt should not be nil")
		}
	})

	t.Run("MarkFailed with nil error", func(t *testing.T) {
		ticket := NewTicket("T4", "Test", "desc")
		ticket.MarkFailed(nil)

		if ticket.Status != StatusFailed {
			t.Errorf("MarkFailed(nil) Status = %v, want failed", ticket.Status)
		}
		if ticket.Error != "" {
			t.Errorf("MarkFailed(nil) Error = %v, want empty string", ticket.Error)
		}
	})
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestTicket_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ticket  *Ticket
		wantErr bool
	}{
		{
			name: "valid ticket",
			ticket: &Ticket{
				ID:     "T1",
				Title:  "Test",
				Status: StatusPending,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			ticket: &Ticket{
				Title:  "Test",
				Status: StatusPending,
			},
			wantErr: true,
		},
		{
			name: "missing title",
			ticket: &Ticket{
				ID:     "T1",
				Status: StatusPending,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			ticket: &Ticket{
				ID:     "T1",
				Title:  "Test",
				Status: Status("invalid"),
			},
			wantErr: true,
		},
		{
			name: "empty status is invalid",
			ticket: &Ticket{
				ID:    "T1",
				Title: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ticket.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTicket_Summary(t *testing.T) {
	ticket := &Ticket{
		ID:     "TEST-001",
		Title:  "Test Ticket",
		Status: StatusPending,
	}

	summary := ticket.Summary()
	expected := "[TEST-001] Test Ticket - pending"

	if summary != expected {
		t.Errorf("Summary() = %v, want %v", summary, expected)
	}
}

func TestTicket_JSON_Serialization(t *testing.T) {
	original := &Ticket{
		ID:                  "JSON-001",
		Title:               "JSON Test",
		Description:         "Testing JSON",
		Type:                TypeFeature,
		Priority:            1,
		Status:              StatusInProgress,
		EstimatedComplexity: "high",
		Dependencies:        []string{"DEP-1", "DEP-2"},
		AcceptanceCriteria:  []string{"Criterion 1"},
		FilesToCreate:       []string{"new.go"},
		FilesToModify:       []string{"existing.go"},
		CreatedAt:           time.Now().Truncate(time.Second),
	}

	// Serialize
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Deserialize
	loaded, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	// Verify fields
	if loaded.ID != original.ID {
		t.Errorf("FromJSON().ID = %v, want %v", loaded.ID, original.ID)
	}
	if loaded.Title != original.Title {
		t.Errorf("FromJSON().Title = %v, want %v", loaded.Title, original.Title)
	}
	if loaded.Status != original.Status {
		t.Errorf("FromJSON().Status = %v, want %v", loaded.Status, original.Status)
	}
	if len(loaded.Dependencies) != len(original.Dependencies) {
		t.Errorf("FromJSON().Dependencies = %v, want %v", loaded.Dependencies, original.Dependencies)
	}
}

func TestFromJSON_Invalid(t *testing.T) {
	_, err := FromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("FromJSON() should return error for invalid JSON")
	}
}

func TestTicketList(t *testing.T) {
	tl := NewTicketList()

	if tl.Count() != 0 {
		t.Errorf("NewTicketList().Count() = %d, want 0", tl.Count())
	}

	// Add tickets
	t1 := &Ticket{ID: "T1", Title: "Test 1", Status: StatusPending}
	t2 := &Ticket{ID: "T2", Title: "Test 2", Status: StatusCompleted}
	t3 := &Ticket{ID: "T3", Title: "Test 3", Status: StatusPending}

	tl.Add(t1)
	tl.Add(t2)
	tl.Add(t3)

	if tl.Count() != 3 {
		t.Errorf("After adding 3 tickets, Count() = %d, want 3", tl.Count())
	}

	// Filter by status
	pending := tl.Filter(StatusPending)
	if len(pending) != 2 {
		t.Errorf("Filter(pending) = %d tickets, want 2", len(pending))
	}

	completed := tl.Filter(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("Filter(completed) = %d tickets, want 1", len(completed))
	}

	failed := tl.Filter(StatusFailed)
	if len(failed) != 0 {
		t.Errorf("Filter(failed) = %d tickets, want 0", len(failed))
	}
}

func TestTicketList_JSON_Serialization(t *testing.T) {
	original := &TicketList{
		Tickets: []*Ticket{
			{ID: "T1", Title: "Test 1", Status: StatusPending},
			{ID: "T2", Title: "Test 2", Status: StatusCompleted},
		},
	}

	// Serialize
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Deserialize
	loaded, err := FromJSONList(jsonData)
	if err != nil {
		t.Fatalf("FromJSONList() error = %v", err)
	}

	if len(loaded.Tickets) != len(original.Tickets) {
		t.Errorf("FromJSONList() ticket count = %d, want %d", len(loaded.Tickets), len(original.Tickets))
	}
}

func TestFromJSONList_Invalid(t *testing.T) {
	_, err := FromJSONList([]byte("invalid json"))
	if err == nil {
		t.Error("FromJSONList() should return error for invalid JSON")
	}
}

func TestIssue_ToTickets(t *testing.T) {
	il := NewIssueList()

	il.Add(&Issue{
		ID:          "ISSUE-001",
		Category:    "performance",
		Severity:    "HIGH",
		Title:       "Performance Issue",
		Description: "Description",
		Location:    "file.go:10",
		Suggestion:  "Fix suggestion",
	})

	il.Add(&Issue{
		ID:          "ISSUE-002",
		Category:    "security",
		Severity:    "MED",
		Title:       "Security Issue",
		Description: "Security desc",
		Location:    "auth.go:20",
		Suggestion:  "Security fix",
	})

	il.Add(&Issue{
		ID:          "ISSUE-003",
		Category:    "test",
		Severity:    "LOW",
		Title:       "Missing Test",
		Description: "Need tests",
		Location:    "handler.go",
		Suggestion:  "Add tests",
	})

	il.Add(&Issue{
		ID:          "ISSUE-004",
		Category:    "docs",
		Severity:    "MEDIUM", // Test alternative spelling
		Title:       "Missing Docs",
		Description: "Need docs",
		Location:    "api.go",
		Suggestion:  "Add docs",
	})

	tl := il.ToTickets()

	if tl.Count() != 4 {
		t.Errorf("ToTickets() count = %d, want 4", tl.Count())
	}

	// Verify type mapping
	tests := []struct {
		id           string
		expectedType Type
		priority     int
	}{
		{"ISSUE-001", TypePerf, 1},
		{"ISSUE-002", TypeSecurity, 3},
		{"ISSUE-003", TypeTest, 5},
		{"ISSUE-004", TypeDocs, 3}, // MEDIUM maps to 3
	}

	for _, tt := range tests {
		for _, ticket := range tl.Tickets {
			if ticket.ID == tt.id {
				if ticket.Type != tt.expectedType {
					t.Errorf("Ticket %s type = %v, want %v", tt.id, ticket.Type, tt.expectedType)
				}
				if ticket.Priority != tt.priority {
					t.Errorf("Ticket %s priority = %d, want %d", tt.id, ticket.Priority, tt.priority)
				}
				break
			}
		}
	}
}

func TestIssueList_FilterByCategory(t *testing.T) {
	il := NewIssueList()
	il.Add(&Issue{ID: "I1", Category: "performance"})
	il.Add(&Issue{ID: "I2", Category: "security"})
	il.Add(&Issue{ID: "I3", Category: "performance"})
	il.Add(&Issue{ID: "I4", Category: "test"})

	perf := il.FilterByCategory("performance")
	if len(perf) != 2 {
		t.Errorf("FilterByCategory(performance) = %d, want 2", len(perf))
	}

	security := il.FilterByCategory("security")
	if len(security) != 1 {
		t.Errorf("FilterByCategory(security) = %d, want 1", len(security))
	}

	docs := il.FilterByCategory("docs")
	if len(docs) != 0 {
		t.Errorf("FilterByCategory(docs) = %d, want 0", len(docs))
	}
}

func TestIssueList_Count(t *testing.T) {
	il := NewIssueList()
	if il.Count() != 0 {
		t.Errorf("NewIssueList().Count() = %d, want 0", il.Count())
	}

	il.Add(&Issue{ID: "I1"})
	il.Add(&Issue{ID: "I2"})

	if il.Count() != 2 {
		t.Errorf("After adding 2, Count() = %d, want 2", il.Count())
	}
}

func TestTicket_CompletedAt_JSON(t *testing.T) {
	// Test that CompletedAt is properly serialized when set
	ticket := NewTicket("T1", "Test", "desc")
	ticket.MarkCompleted("output")

	jsonData, err := ticket.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify CompletedAt is in JSON
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, ok := data["completed_at"]; !ok {
		t.Error("completed_at should be present in JSON")
	}

	// Test that CompletedAt is omitted when nil
	ticket2 := NewTicket("T2", "Test 2", "desc")
	jsonData2, _ := ticket2.ToJSON()

	var data2 map[string]interface{}
	json.Unmarshal(jsonData2, &data2)

	if _, ok := data2["completed_at"]; ok {
		t.Error("completed_at should be omitted when nil")
	}
}
