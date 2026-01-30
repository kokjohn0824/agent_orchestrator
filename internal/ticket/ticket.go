// Package ticket provides ticket data structures and operations
package ticket

import (
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the status of a ticket
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}

// Type represents the type of a ticket
type Type string

const (
	TypeFeature  Type = "feature"
	TypeTest     Type = "test"
	TypeRefactor Type = "refactor"
	TypeDocs     Type = "docs"
	TypeBugfix   Type = "bugfix"
	TypePerf     Type = "performance"
	TypeSecurity Type = "security"
)

// String returns the string representation of the type
func (t Type) String() string {
	return string(t)
}

// Ticket represents a work ticket
type Ticket struct {
	ID                  string     `json:"id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Type                Type       `json:"type"`
	Priority            int        `json:"priority"`
	Status              Status     `json:"status"`
	EstimatedComplexity string     `json:"estimated_complexity"`
	Dependencies        []string   `json:"dependencies"`
	AcceptanceCriteria  []string   `json:"acceptance_criteria"`
	FilesToCreate       []string   `json:"files_to_create"`
	FilesToModify       []string   `json:"files_to_modify"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	AgentOutput         string     `json:"agent_output,omitempty"`
	Error               string     `json:"error,omitempty"`
	ErrorLog            string     `json:"error_log,omitempty"` // Path to agent log file when failed
}

// NewTicket creates a new ticket with default values
func NewTicket(id, title, description string) *Ticket {
	return &Ticket{
		ID:                  id,
		Title:               title,
		Description:         description,
		Type:                TypeFeature,
		Priority:            5,
		Status:              StatusPending,
		EstimatedComplexity: "medium",
		Dependencies:        make([]string, 0),
		AcceptanceCriteria:  make([]string, 0),
		FilesToCreate:       make([]string, 0),
		FilesToModify:       make([]string, 0),
		CreatedAt:           time.Now(),
	}
}

// MarkInProgress marks the ticket as in progress
func (t *Ticket) MarkInProgress() {
	t.Status = StatusInProgress
}

// MarkCompleted marks the ticket as completed
func (t *Ticket) MarkCompleted(output string) {
	t.Status = StatusCompleted
	now := time.Now()
	t.CompletedAt = &now
	t.AgentOutput = output
}

// MarkFailed marks the ticket as failed
func (t *Ticket) MarkFailed(err error) {
	t.Status = StatusFailed
	now := time.Now()
	t.CompletedAt = &now
	if err != nil {
		t.Error = err.Error()
	}
}

// ToJSON converts the ticket to JSON
func (t *Ticket) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// FromJSON creates a ticket from JSON
func FromJSON(data []byte) (*Ticket, error) {
	var ticket Ticket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse ticket JSON: %w", err)
	}
	return &ticket, nil
}

// Validate validates the ticket
func (t *Ticket) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("ticket ID is required")
	}
	if t.Title == "" {
		return fmt.Errorf("ticket title is required")
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid ticket status: %s", t.Status)
	}
	return nil
}

// Summary returns a short summary of the ticket
func (t *Ticket) Summary() string {
	return fmt.Sprintf("[%s] %s - %s", t.ID, t.Title, t.Status)
}

// TicketList represents a list of tickets
type TicketList struct {
	Tickets []*Ticket `json:"tickets"`
}

// NewTicketList creates a new empty ticket list
func NewTicketList() *TicketList {
	return &TicketList{
		Tickets: make([]*Ticket, 0),
	}
}

// Add adds a ticket to the list
func (tl *TicketList) Add(t *Ticket) {
	tl.Tickets = append(tl.Tickets, t)
}

// Count returns the number of tickets
func (tl *TicketList) Count() int {
	return len(tl.Tickets)
}

// Filter returns tickets matching the given status
func (tl *TicketList) Filter(status Status) []*Ticket {
	result := make([]*Ticket, 0)
	for _, t := range tl.Tickets {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

// ToJSON converts the ticket list to JSON
func (tl *TicketList) ToJSON() ([]byte, error) {
	return json.MarshalIndent(tl, "", "  ")
}

// FromJSONList creates a ticket list from JSON
func FromJSONList(data []byte) (*TicketList, error) {
	var tl TicketList
	if err := json.Unmarshal(data, &tl); err != nil {
		return nil, fmt.Errorf("failed to parse ticket list JSON: %w", err)
	}
	return &tl, nil
}

// Issue represents an issue found by analyze command
type Issue struct {
	ID          string `json:"id"`
	Category    string `json:"category"` // performance, refactor, security, test, docs
	Severity    string `json:"severity"` // HIGH, MED, LOW
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Suggestion  string `json:"suggestion"`
}

// IssueList represents a list of issues
type IssueList struct {
	Issues []*Issue `json:"issues"`
}

// NewIssueList creates a new empty issue list
func NewIssueList() *IssueList {
	return &IssueList{
		Issues: make([]*Issue, 0),
	}
}

// Add adds an issue to the list
func (il *IssueList) Add(i *Issue) {
	il.Issues = append(il.Issues, i)
}

// Count returns the number of issues
func (il *IssueList) Count() int {
	return len(il.Issues)
}

// FilterByCategory returns issues matching the given category
func (il *IssueList) FilterByCategory(category string) []*Issue {
	result := make([]*Issue, 0)
	for _, i := range il.Issues {
		if i.Category == category {
			result = append(result, i)
		}
	}
	return result
}

// ToTickets converts issues to tickets
func (il *IssueList) ToTickets() *TicketList {
	tl := NewTicketList()
	for _, issue := range il.Issues {
		ticketType := TypeRefactor
		switch issue.Category {
		case "performance":
			ticketType = TypePerf
		case "security":
			ticketType = TypeSecurity
		case "test":
			ticketType = TypeTest
		case "docs":
			ticketType = TypeDocs
		}

		priority := 5
		switch issue.Severity {
		case "HIGH":
			priority = 1
		case "MED", "MEDIUM":
			priority = 3
		case "LOW":
			priority = 5
		}

		t := NewTicket(issue.ID, issue.Title, issue.Description)
		t.Type = ticketType
		t.Priority = priority
		t.AcceptanceCriteria = []string{issue.Suggestion}
		t.FilesToModify = []string{issue.Location}

		tl.Add(t)
	}
	return tl
}
