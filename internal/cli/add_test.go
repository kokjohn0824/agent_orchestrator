package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// resetAddFlags restores add command flags to default for test isolation
func resetAddFlags() {
	addTitle = ""
	addType = "feature"
	addPriority = 3
	addDescription = ""
	addDeps = ""
	addCriteria = ""
	addEnhance = false
}

func TestCreateTicketFromFlags_Feature(t *testing.T) {
	resetAddFlags()
	addTitle = "Test feature"
	addDescription = "Description"
	addType = "feature"
	addPriority = 1

	tkt, err := createTicketFromFlags()
	if err != nil {
		t.Fatalf("createTicketFromFlags() err = %v", err)
	}
	if tkt.Title != "Test feature" {
		t.Errorf("Title = %q, want Test feature", tkt.Title)
	}
	if tkt.Type != ticket.TypeFeature {
		t.Errorf("Type = %s, want feature", tkt.Type)
	}
	if tkt.Priority != 1 {
		t.Errorf("Priority = %d, want 1", tkt.Priority)
	}
	if !strings.HasPrefix(tkt.ID, "TICKET-") {
		t.Errorf("ID should start with TICKET-, got %q", tkt.ID)
	}
}

func TestCreateTicketFromFlags_AllTypes(t *testing.T) {
	types := []struct {
		flag string
		want ticket.Type
	}{
		{"feature", ticket.TypeFeature},
		{"bugfix", ticket.TypeBugfix},
		{"refactor", ticket.TypeRefactor},
		{"test", ticket.TypeTest},
		{"docs", ticket.TypeDocs},
		{"performance", ticket.TypePerf},
		{"perf", ticket.TypePerf},
		{"security", ticket.TypeSecurity},
		{"unknown", ticket.TypeFeature}, // default
	}

	for _, tc := range types {
		t.Run(tc.flag, func(t *testing.T) {
			resetAddFlags()
			addTitle = "Title"
			addType = tc.flag

			tkt, err := createTicketFromFlags()
			if err != nil {
				t.Fatalf("createTicketFromFlags() err = %v", err)
			}
			if tkt.Type != tc.want {
				t.Errorf("Type = %s, want %s", tkt.Type, tc.want)
			}
		})
	}
}

func TestCreateTicketFromFlags_PriorityBounds(t *testing.T) {
	resetAddFlags()
	addTitle = "Title"
	addPriority = 5

	tkt, err := createTicketFromFlags()
	if err != nil {
		t.Fatalf("createTicketFromFlags() err = %v", err)
	}
	if tkt.Priority != 5 {
		t.Errorf("Priority = %d, want 5", tkt.Priority)
	}

	addPriority = 0 // outside 1-5, should not override default from NewTicket
	tkt, _ = createTicketFromFlags()
	// Priority 0 is not in [1,5] so it stays at default 5 from NewTicket
	if tkt.Priority != 5 {
		t.Errorf("Priority with 0 = %d, expected 5 (default)", tkt.Priority)
	}
}

func TestCreateTicketFromFlags_DepsAndCriteria(t *testing.T) {
	resetAddFlags()
	addTitle = "Title"
	addDeps = "TICKET-001, TICKET-002"
	addCriteria = "AC1, AC2"

	tkt, err := createTicketFromFlags()
	if err != nil {
		t.Fatalf("createTicketFromFlags() err = %v", err)
	}
	if len(tkt.Dependencies) != 2 {
		t.Errorf("Dependencies len = %d, want 2", len(tkt.Dependencies))
	}
	if len(tkt.AcceptanceCriteria) != 2 {
		t.Errorf("AcceptanceCriteria len = %d, want 2", len(tkt.AcceptanceCriteria))
	}
}

func TestGenerateTicketID(t *testing.T) {
	id := generateTicketID()
	if id == "" {
		t.Error("generateTicketID() returned empty")
	}
	if !strings.HasPrefix(id, "TICKET-") {
		t.Errorf("ID should start with TICKET-, got %q", id)
	}
}

func TestRunAdd_WithFlags_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "add-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsDir := filepath.Join(tmpDir, ".tickets")
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        ticketsDir,
		AgentCommand:      "nonexistent-agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	resetAddFlags()
	addTitle = "Test ticket"
	addDescription = "Description"
	addType = "feature"
	addPriority = 2

	err = runAdd(nil, nil)
	if err != nil {
		t.Fatalf("runAdd() err = %v", err)
	}

	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init() err = %v", err)
	}
	counts, err := store.Count()
	if err != nil {
		t.Fatalf("store.Count() err = %v", err)
	}
	if counts[ticket.StatusPending] != 1 {
		t.Errorf("Expected 1 pending ticket, got %d", counts[ticket.StatusPending])
	}
}

func TestRunAdd_StoreInitFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "add-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Make TicketsDir a file so MkdirAll in store.Init() fails
	ticketsPath := filepath.Join(tmpDir, ".tickets")
	if err := os.WriteFile(ticketsPath, []byte("x"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        ticketsPath,
		AgentCommand:      "nonexistent-agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	resetAddFlags()
	addTitle = "Test"
	addType = "feature"

	err = runAdd(nil, nil)
	if err == nil {
		t.Error("runAdd() expected error when store init fails")
	}
	if err != nil && !strings.Contains(err.Error(), "store") && !strings.Contains(err.Error(), "初始化") {
		t.Errorf("error should mention store/init, got: %v", err)
	}
}

func TestAddCmd_FlagsRegistered(t *testing.T) {
	cmd := addCmd
	flags := []string{"title", "type", "priority", "description", "deps", "criteria", "enhance"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("add command should have flag %q", name)
		}
	}
}

func TestDisplayTicketDetails(t *testing.T) {
	tkt := ticket.NewTicket("T-001", "Test Title", "Test description")
	tkt.Type = ticket.TypeFeature
	tkt.Priority = 2
	tkt.Dependencies = []string{"T-000"}
	tkt.AcceptanceCriteria = []string{"AC1"}

	r, w, _ := os.Pipe()
	displayTicketDetails(w, tkt)
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if !strings.Contains(out, "T-001") {
		t.Error("output should contain ticket ID")
	}
	if !strings.Contains(out, "Test Title") {
		t.Error("output should contain title")
	}
	if !strings.Contains(out, "feature") {
		t.Error("output should contain type")
	}
	if !strings.Contains(out, "P2") {
		t.Error("output should contain priority")
	}
	if !strings.Contains(out, "T-000") {
		t.Error("output should contain dependency")
	}
	if !strings.Contains(out, "AC1") {
		t.Error("output should contain acceptance criteria")
	}
}
