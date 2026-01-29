package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// MockAgentCaller is a mock implementation of agent.Caller for testing
type MockAgentCaller struct {
	// CallFunc allows customizing the Call behavior per test
	CallFunc func(ctx context.Context, prompt string, opts ...agent.CallOption) (*agent.Result, error)
	// CallForJSONFunc allows customizing the CallForJSON behavior per test
	CallForJSONFunc func(ctx context.Context, prompt string, outputFile string, opts ...agent.CallOption) (*agent.Result, map[string]interface{}, error)
	// CallCount tracks how many times Call was invoked
	CallCount int
	// CallForJSONCount tracks how many times CallForJSON was invoked
	CallForJSONCount int
	// Prompts records all prompts received
	Prompts []string
}

// NewMockAgentCaller creates a new mock caller with default success responses
func NewMockAgentCaller() *MockAgentCaller {
	return &MockAgentCaller{
		CallFunc: func(ctx context.Context, prompt string, opts ...agent.CallOption) (*agent.Result, error) {
			return &agent.Result{
				Success:  true,
				Output:   "Mock output",
				Duration: 100 * time.Millisecond,
			}, nil
		},
		CallForJSONFunc: func(ctx context.Context, prompt string, outputFile string, opts ...agent.CallOption) (*agent.Result, map[string]interface{}, error) {
			return &agent.Result{
				Success:  true,
				Output:   "Mock JSON output",
				Duration: 100 * time.Millisecond,
			}, map[string]interface{}{}, nil
		},
		Prompts: make([]string, 0),
	}
}

// Call implements the Call method for MockAgentCaller
func (m *MockAgentCaller) Call(ctx context.Context, prompt string, opts ...agent.CallOption) (*agent.Result, error) {
	m.CallCount++
	m.Prompts = append(m.Prompts, prompt)
	return m.CallFunc(ctx, prompt, opts...)
}

// CallForJSON implements the CallForJSON method for MockAgentCaller
func (m *MockAgentCaller) CallForJSON(ctx context.Context, prompt string, outputFile string, opts ...agent.CallOption) (*agent.Result, map[string]interface{}, error) {
	m.CallForJSONCount++
	m.Prompts = append(m.Prompts, prompt)
	return m.CallForJSONFunc(ctx, prompt, outputFile, opts...)
}

// setupTestEnvironment creates a temporary directory structure for testing
func setupTestEnvironment(t *testing.T) (string, func()) {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "pipeline-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories
	dirs := []string{
		filepath.Join(tmpDir, ".tickets", "pending"),
		filepath.Join(tmpDir, ".tickets", "in_progress"),
		filepath.Join(tmpDir, ".tickets", "completed"),
		filepath.Join(tmpDir, ".tickets", "failed"),
		filepath.Join(tmpDir, "docs"),
		filepath.Join(tmpDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createMilestoneFile creates a test milestone file
func createMilestoneFile(t *testing.T, dir string, content string) string {
	t.Helper()

	path := filepath.Join(dir, "docs", "test-milestone.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create milestone file: %v", err)
	}
	return path
}

// createTestConfig creates a test configuration
func createTestConfig(tmpDir string) *config.Config {
	return &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        filepath.Join(tmpDir, ".tickets"),
		DocsDir:           filepath.Join(tmpDir, "docs"),
		LogsDir:           filepath.Join(tmpDir, "logs"),
		AgentCommand:      "mock-agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true, // Important: use dry run mode for testing
		Verbose:           false,
	}
}

// TestPipelineIntegration_FullFlow tests the complete pipeline flow
func TestPipelineIntegration_FullFlow(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create milestone file
	milestoneContent := `# Test Milestone

## Goals
- Implement feature A
- Add tests for feature A
`
	milestonePath := createMilestoneFile(t, tmpDir, milestoneContent)

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create mock tickets (simulating planning phase output)
	mockTickets := []*ticket.Ticket{
		{
			ID:                  "TEST-001",
			Title:               "Implement feature A",
			Description:         "Implement the main feature",
			Type:                ticket.TypeFeature,
			Priority:            1,
			Status:              ticket.StatusPending,
			EstimatedComplexity: "medium",
			Dependencies:        []string{},
			AcceptanceCriteria:  []string{"Feature works"},
			CreatedAt:           time.Now(),
		},
		{
			ID:                  "TEST-002",
			Title:               "Add tests for feature A",
			Description:         "Write unit tests",
			Type:                ticket.TypeTest,
			Priority:            2,
			Status:              ticket.StatusPending,
			EstimatedComplexity: "low",
			Dependencies:        []string{"TEST-001"},
			AcceptanceCriteria:  []string{"Tests pass"},
			CreatedAt:           time.Now(),
		},
	}

	// Save mock tickets
	for _, t := range mockTickets {
		if err := store.Save(t); err != nil {
			// Using different variable name to avoid shadowing
			panic(fmt.Sprintf("Failed to save ticket: %v", err))
		}
	}

	// Verify tickets were saved
	counts, err := store.Count()
	if err != nil {
		t.Fatalf("Failed to count tickets: %v", err)
	}

	if counts[ticket.StatusPending] != 2 {
		t.Errorf("Expected 2 pending tickets, got %d", counts[ticket.StatusPending])
	}

	// Test dependency resolution
	resolver := ticket.NewDependencyResolver(store)
	processable, err := resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable tickets: %v", err)
	}

	// Only TEST-001 should be processable (TEST-002 depends on it)
	if len(processable) != 1 {
		t.Errorf("Expected 1 processable ticket, got %d", len(processable))
	}

	if len(processable) > 0 && processable[0].ID != "TEST-001" {
		t.Errorf("Expected TEST-001 to be processable, got %s", processable[0].ID)
	}

	// Verify milestone file exists
	if _, err := os.Stat(milestonePath); os.IsNotExist(err) {
		t.Errorf("Milestone file should exist at %s", milestonePath)
	}
}

// TestPipelineIntegration_DependencyOrder tests that tickets are processed in correct dependency order
func TestPipelineIntegration_DependencyOrder(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create tickets with chain dependencies: A <- B <- C
	tickets := []*ticket.Ticket{
		{
			ID:           "CHAIN-C",
			Title:        "Task C",
			Description:  "Depends on B",
			Status:       ticket.StatusPending,
			Dependencies: []string{"CHAIN-B"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "CHAIN-A",
			Title:        "Task A",
			Description:  "No dependencies",
			Status:       ticket.StatusPending,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "CHAIN-B",
			Title:        "Task B",
			Description:  "Depends on A",
			Status:       ticket.StatusPending,
			Dependencies: []string{"CHAIN-A"},
			CreatedAt:    time.Now(),
		},
	}

	for _, ticket := range tickets {
		if err := store.Save(ticket); err != nil {
			t.Fatalf("Failed to save ticket: %v", err)
		}
	}

	resolver := ticket.NewDependencyResolver(store)

	// Initially, only CHAIN-A should be processable
	processable, err := resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable: %v", err)
	}

	if len(processable) != 1 || processable[0].ID != "CHAIN-A" {
		t.Errorf("Expected only CHAIN-A to be processable initially")
	}

	// Complete CHAIN-A
	ticketA, _ := store.Load("CHAIN-A")
	ticketA.MarkCompleted("Done")
	store.Save(ticketA)

	// Now CHAIN-B should be processable
	processable, err = resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable: %v", err)
	}

	if len(processable) != 1 || processable[0].ID != "CHAIN-B" {
		t.Errorf("Expected only CHAIN-B to be processable after A completed")
	}

	// Complete CHAIN-B
	ticketB, _ := store.Load("CHAIN-B")
	ticketB.MarkCompleted("Done")
	store.Save(ticketB)

	// Now CHAIN-C should be processable
	processable, err = resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable: %v", err)
	}

	if len(processable) != 1 || processable[0].ID != "CHAIN-C" {
		t.Errorf("Expected only CHAIN-C to be processable after B completed")
	}
}

// TestPipelineIntegration_TicketStatusTransitions tests ticket status transitions
func TestPipelineIntegration_TicketStatusTransitions(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create a test ticket
	testTicket := ticket.NewTicket("STATUS-001", "Test Status", "Test status transitions")
	if err := store.Save(testTicket); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	// Verify initial status
	loaded, err := store.Load("STATUS-001")
	if err != nil {
		t.Fatalf("Failed to load ticket: %v", err)
	}
	if loaded.Status != ticket.StatusPending {
		t.Errorf("Expected pending status, got %s", loaded.Status)
	}

	// Transition to in_progress
	loaded.MarkInProgress()
	if err := store.Save(loaded); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	loaded, _ = store.Load("STATUS-001")
	if loaded.Status != ticket.StatusInProgress {
		t.Errorf("Expected in_progress status, got %s", loaded.Status)
	}

	// Transition to completed
	loaded.MarkCompleted("Task completed successfully")
	if err := store.Save(loaded); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	loaded, _ = store.Load("STATUS-001")
	if loaded.Status != ticket.StatusCompleted {
		t.Errorf("Expected completed status, got %s", loaded.Status)
	}
	if loaded.AgentOutput != "Task completed successfully" {
		t.Errorf("Expected agent output to be saved")
	}
	if loaded.CompletedAt == nil {
		t.Errorf("Expected completed_at to be set")
	}
}

// TestPipelineIntegration_FailedTicket tests handling of failed tickets
func TestPipelineIntegration_FailedTicket(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create a test ticket
	testTicket := ticket.NewTicket("FAIL-001", "Test Failure", "Test failure handling")
	testTicket.MarkInProgress()
	if err := store.Save(testTicket); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	// Mark as failed
	testTicket.MarkFailed(fmt.Errorf("simulated error"))
	if err := store.Save(testTicket); err != nil {
		t.Fatalf("Failed to save failed ticket: %v", err)
	}

	// Verify failed status
	loaded, err := store.Load("FAIL-001")
	if err != nil {
		t.Fatalf("Failed to load ticket: %v", err)
	}
	if loaded.Status != ticket.StatusFailed {
		t.Errorf("Expected failed status, got %s", loaded.Status)
	}
	if loaded.Error != "simulated error" {
		t.Errorf("Expected error message to be saved, got %s", loaded.Error)
	}

	// Test retry (move failed back to pending)
	count, err := store.MoveFailed()
	if err != nil {
		t.Fatalf("Failed to move failed tickets: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 ticket moved, got %d", count)
	}

	loaded, _ = store.Load("FAIL-001")
	if loaded.Status != ticket.StatusPending {
		t.Errorf("Expected pending status after retry, got %s", loaded.Status)
	}
}

// TestPipelineIntegration_CircularDependency tests detection of circular dependencies
func TestPipelineIntegration_CircularDependency(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create tickets with circular dependency: A -> B -> C -> A
	circularTickets := []*ticket.Ticket{
		{
			ID:           "CIRC-A",
			Title:        "Circular A",
			Description:  "Depends on C",
			Status:       ticket.StatusPending,
			Dependencies: []string{"CIRC-C"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "CIRC-B",
			Title:        "Circular B",
			Description:  "Depends on A",
			Status:       ticket.StatusPending,
			Dependencies: []string{"CIRC-A"},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "CIRC-C",
			Title:        "Circular C",
			Description:  "Depends on B",
			Status:       ticket.StatusPending,
			Dependencies: []string{"CIRC-B"},
			CreatedAt:    time.Now(),
		},
	}

	for _, ticket := range circularTickets {
		if err := store.Save(ticket); err != nil {
			t.Fatalf("Failed to save ticket: %v", err)
		}
	}

	resolver := ticket.NewDependencyResolver(store)

	// Check for circular dependency
	hasCircular := resolver.HasCircularDependency(circularTickets)
	if !hasCircular {
		t.Error("Expected circular dependency to be detected")
	}

	// No tickets should be processable with circular dependencies
	processable, err := resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable: %v", err)
	}

	if len(processable) != 0 {
		t.Errorf("Expected 0 processable tickets with circular dependencies, got %d", len(processable))
	}
}

// TestPipelineIntegration_ParallelTickets tests handling of tickets that can be processed in parallel
func TestPipelineIntegration_ParallelTickets(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create multiple tickets without dependencies (can be parallel)
	parallelTickets := []*ticket.Ticket{
		{
			ID:           "PAR-A",
			Title:        "Parallel A",
			Description:  "No dependencies",
			Status:       ticket.StatusPending,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "PAR-B",
			Title:        "Parallel B",
			Description:  "No dependencies",
			Status:       ticket.StatusPending,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
		{
			ID:           "PAR-C",
			Title:        "Parallel C",
			Description:  "No dependencies",
			Status:       ticket.StatusPending,
			Dependencies: []string{},
			CreatedAt:    time.Now(),
		},
	}

	for _, ticket := range parallelTickets {
		if err := store.Save(ticket); err != nil {
			t.Fatalf("Failed to save ticket: %v", err)
		}
	}

	resolver := ticket.NewDependencyResolver(store)

	// All tickets should be processable
	processable, err := resolver.GetProcessable()
	if err != nil {
		t.Fatalf("Failed to get processable: %v", err)
	}

	if len(processable) != 3 {
		t.Errorf("Expected 3 processable tickets, got %d", len(processable))
	}
}

// TestPipelineIntegration_StoreCleanup tests store cleanup functionality
func TestPipelineIntegration_StoreCleanup(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test config
	originalCfg := cfg
	cfg = createTestConfig(tmpDir)
	defer func() { cfg = originalCfg }()

	// Initialize ticket store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	// Create some tickets
	for i := 1; i <= 5; i++ {
		ticket := ticket.NewTicket(
			fmt.Sprintf("CLEAN-%03d", i),
			fmt.Sprintf("Cleanup Test %d", i),
			"Test cleanup",
		)
		if err := store.Save(ticket); err != nil {
			t.Fatalf("Failed to save ticket: %v", err)
		}
	}

	// Verify tickets exist
	counts, _ := store.Count()
	if counts[ticket.StatusPending] != 5 {
		t.Errorf("Expected 5 pending tickets, got %d", counts[ticket.StatusPending])
	}

	// Clean the store
	if err := store.Clean(); err != nil {
		t.Fatalf("Failed to clean store: %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(cfg.TicketsDir); !os.IsNotExist(err) {
		t.Error("Expected tickets directory to be removed after clean")
	}
}

// TestPipelineIntegration_TicketValidation tests ticket validation
func TestPipelineIntegration_TicketValidation(t *testing.T) {
	tests := []struct {
		name        string
		ticket      *ticket.Ticket
		expectError bool
	}{
		{
			name: "valid ticket",
			ticket: &ticket.Ticket{
				ID:     "VALID-001",
				Title:  "Valid Ticket",
				Status: ticket.StatusPending,
			},
			expectError: false,
		},
		{
			name: "missing ID",
			ticket: &ticket.Ticket{
				ID:     "",
				Title:  "No ID",
				Status: ticket.StatusPending,
			},
			expectError: true,
		},
		{
			name: "missing title",
			ticket: &ticket.Ticket{
				ID:     "NOTITLE-001",
				Title:  "",
				Status: ticket.StatusPending,
			},
			expectError: true,
		},
		{
			name: "invalid status",
			ticket: &ticket.Ticket{
				ID:     "BADSTATUS-001",
				Title:  "Bad Status",
				Status: ticket.Status("invalid"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ticket.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

// TestPipelineIntegration_JSONSerialization tests ticket JSON serialization
func TestPipelineIntegration_JSONSerialization(t *testing.T) {
	original := &ticket.Ticket{
		ID:                  "JSON-001",
		Title:               "JSON Test",
		Description:         "Test JSON serialization",
		Type:                ticket.TypeFeature,
		Priority:            1,
		Status:              ticket.StatusPending,
		EstimatedComplexity: "medium",
		Dependencies:        []string{"DEP-001", "DEP-002"},
		AcceptanceCriteria:  []string{"Criterion 1", "Criterion 2"},
		FilesToCreate:       []string{"new.go"},
		FilesToModify:       []string{"existing.go"},
		CreatedAt:           time.Now(),
	}

	// Serialize to JSON
	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize ticket: %v", err)
	}

	// Deserialize from JSON
	restored, err := ticket.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize ticket: %v", err)
	}

	// Verify fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", restored.ID, original.ID)
	}
	if restored.Title != original.Title {
		t.Errorf("Title mismatch: %s != %s", restored.Title, original.Title)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: %s != %s", restored.Type, original.Type)
	}
	if len(restored.Dependencies) != len(original.Dependencies) {
		t.Errorf("Dependencies count mismatch: %d != %d", len(restored.Dependencies), len(original.Dependencies))
	}
}

// TestPipelineIntegration_AgentPromptGeneration tests that agent prompts are generated correctly
func TestPipelineIntegration_AgentPromptGeneration(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a coding agent and test prompt generation
	caller := agent.NewCaller("mock", true, "text", "")
	caller.SetDryRun(true)

	codingAgent := agent.NewCodingAgent(caller, tmpDir)

	testTicket := &ticket.Ticket{
		ID:                  "PROMPT-001",
		Title:               "Test Prompt",
		Description:         "Test prompt generation",
		Type:                ticket.TypeFeature,
		EstimatedComplexity: "high",
		FilesToModify:       []string{"main.go", "util.go"},
		AcceptanceCriteria:  []string{"Tests pass", "Code compiles"},
	}

	// Execute will generate a prompt - in dry run mode it won't actually call agent
	ctx := context.Background()
	result, err := codingAgent.Execute(ctx, testTicket)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// In dry run mode, it should return success
	if !result.Success {
		t.Error("Expected success in dry run mode")
	}
}

// TestPipelineIntegration_TopologicalSort tests dependency-based sorting
func TestPipelineIntegration_TopologicalSort(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	store.Init()

	// Create tickets: D depends on B and C, B depends on A, C depends on A
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	tickets := []*ticket.Ticket{
		{ID: "D", Title: "D", Status: ticket.StatusPending, Dependencies: []string{"B", "C"}, CreatedAt: time.Now()},
		{ID: "A", Title: "A", Status: ticket.StatusPending, Dependencies: []string{}, CreatedAt: time.Now()},
		{ID: "C", Title: "C", Status: ticket.StatusPending, Dependencies: []string{"A"}, CreatedAt: time.Now()},
		{ID: "B", Title: "B", Status: ticket.StatusPending, Dependencies: []string{"A"}, CreatedAt: time.Now()},
	}

	resolver := ticket.NewDependencyResolver(store)
	sorted := resolver.SortByDependency(tickets)

	// A must come first
	if sorted[0].ID != "A" {
		t.Errorf("Expected A first, got %s", sorted[0].ID)
	}

	// D must come last
	if sorted[len(sorted)-1].ID != "D" {
		t.Errorf("Expected D last, got %s", sorted[len(sorted)-1].ID)
	}

	// B and C must come before D
	dIndex := -1
	for i, ticket := range sorted {
		if ticket.ID == "D" {
			dIndex = i
			break
		}
	}

	for _, dep := range []string{"B", "C"} {
		depIndex := -1
		for i, ticket := range sorted {
			if ticket.ID == dep {
				depIndex = i
				break
			}
		}
		if depIndex >= dIndex {
			t.Errorf("Expected %s to come before D", dep)
		}
	}
}

// TestPipelineIntegration_LoadByStatus tests loading tickets by status
func TestPipelineIntegration_LoadByStatus(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	store.Init()

	// Create tickets with different statuses
	statuses := []ticket.Status{
		ticket.StatusPending,
		ticket.StatusPending,
		ticket.StatusInProgress,
		ticket.StatusCompleted,
		ticket.StatusFailed,
	}

	for i, status := range statuses {
		t := ticket.NewTicket(fmt.Sprintf("STATUS-%d", i), fmt.Sprintf("Test %d", i), "Description")
		t.Status = status
		if status == ticket.StatusCompleted {
			now := time.Now()
			t.CompletedAt = &now
		}
		store.Save(t)
	}

	// Test loading by status
	pending, _ := store.LoadByStatus(ticket.StatusPending)
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending tickets, got %d", len(pending))
	}

	inProgress, _ := store.LoadByStatus(ticket.StatusInProgress)
	if len(inProgress) != 1 {
		t.Errorf("Expected 1 in_progress ticket, got %d", len(inProgress))
	}

	completed, _ := store.LoadByStatus(ticket.StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed ticket, got %d", len(completed))
	}

	failed, _ := store.LoadByStatus(ticket.StatusFailed)
	if len(failed) != 1 {
		t.Errorf("Expected 1 failed ticket, got %d", len(failed))
	}
}

// TestPipelineIntegration_TicketListOperations tests TicketList operations
func TestPipelineIntegration_TicketListOperations(t *testing.T) {
	tl := ticket.NewTicketList()

	// Add tickets
	tl.Add(ticket.NewTicket("TL-001", "Test 1", "Description"))
	tl.Add(ticket.NewTicket("TL-002", "Test 2", "Description"))
	tl.Add(ticket.NewTicket("TL-003", "Test 3", "Description"))

	// Mark one as completed
	tl.Tickets[1].MarkCompleted("Done")

	// Test Count
	if tl.Count() != 3 {
		t.Errorf("Expected count 3, got %d", tl.Count())
	}

	// Test Filter
	pending := tl.Filter(ticket.StatusPending)
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending, got %d", len(pending))
	}

	completed := tl.Filter(ticket.StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed, got %d", len(completed))
	}

	// Test JSON serialization
	jsonData, err := tl.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize TicketList: %v", err)
	}

	restored, err := ticket.FromJSONList(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize TicketList: %v", err)
	}

	if restored.Count() != tl.Count() {
		t.Errorf("Restored count mismatch: %d != %d", restored.Count(), tl.Count())
	}
}

// TestPipelineIntegration_IssueToTicketConversion tests Issue to Ticket conversion
func TestPipelineIntegration_IssueToTicketConversion(t *testing.T) {
	issues := ticket.NewIssueList()

	issues.Add(&ticket.Issue{
		ID:          "ISSUE-001",
		Category:    "performance",
		Severity:    "HIGH",
		Title:       "Performance Issue",
		Description: "Slow query",
		Location:    "db/query.go:42",
		Suggestion:  "Add index",
	})

	issues.Add(&ticket.Issue{
		ID:          "ISSUE-002",
		Category:    "security",
		Severity:    "MED",
		Title:       "Security Issue",
		Description: "Missing validation",
		Location:    "api/handler.go:15",
		Suggestion:  "Add input validation",
	})

	// Convert to tickets
	tickets := issues.ToTickets()

	if tickets.Count() != 2 {
		t.Errorf("Expected 2 tickets, got %d", tickets.Count())
	}

	// Check first ticket (HIGH severity should be priority 1)
	if tickets.Tickets[0].Priority != 1 {
		t.Errorf("Expected priority 1 for HIGH severity, got %d", tickets.Tickets[0].Priority)
	}
	if tickets.Tickets[0].Type != ticket.TypePerf {
		t.Errorf("Expected performance type, got %s", tickets.Tickets[0].Type)
	}

	// Check second ticket (MED severity should be priority 3)
	if tickets.Tickets[1].Priority != 3 {
		t.Errorf("Expected priority 3 for MED severity, got %d", tickets.Tickets[1].Priority)
	}
	if tickets.Tickets[1].Type != ticket.TypeSecurity {
		t.Errorf("Expected security type, got %s", tickets.Tickets[1].Type)
	}
}

// TestRunPipelineFlags tests the run command flags
func TestRunPipelineFlags(t *testing.T) {
	// Test that flags are registered correctly
	cmd := runCmd

	// Check flags exist
	flags := []string{"analyze-first", "skip-test", "skip-review", "skip-commit"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Flag %s should be registered", flag)
		}
	}
}

// TestPipelineIntegration_MilestoneFileValidation tests milestone file validation
func TestPipelineIntegration_MilestoneFileValidation(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "non-existent.md")
	if _, err := os.Stat(nonExistent); !os.IsNotExist(err) {
		t.Error("Expected file to not exist")
	}

	// Test valid file
	validPath := createMilestoneFile(t, tmpDir, "# Valid Milestone")
	if _, err := os.Stat(validPath); err != nil {
		t.Errorf("Expected file to exist: %v", err)
	}
}

// BenchmarkTicketSave benchmarks ticket save operations
func BenchmarkTicketSave(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	store.Init()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := ticket.NewTicket(fmt.Sprintf("BENCH-%d", i), "Benchmark", "Description")
		store.Save(t)
	}
}

// BenchmarkDependencyResolution benchmarks dependency resolution
func BenchmarkDependencyResolution(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	store.Init()

	// Create 100 tickets with varying dependencies
	for i := 0; i < 100; i++ {
		t := ticket.NewTicket(fmt.Sprintf("BENCH-%03d", i), fmt.Sprintf("Ticket %d", i), "Description")
		if i > 0 {
			t.Dependencies = []string{fmt.Sprintf("BENCH-%03d", i-1)}
		}
		store.Save(t)
	}

	resolver := ticket.NewDependencyResolver(store)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.GetProcessable()
	}
}

// Helper function to check if output contains expected strings
func outputContains(output string, expected []string) []string {
	var missing []string
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			missing = append(missing, exp)
		}
	}
	return missing
}

// Helper for capturing stdout during test
func captureOutput(f func()) string {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	return buf.String()
}

// TestPipelineIntegration_GeneratedTicketsFile tests saving/loading generated tickets
func TestPipelineIntegration_GeneratedTicketsFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	store.Init()

	tickets := []*ticket.Ticket{
		ticket.NewTicket("GEN-001", "Generated 1", "Description"),
		ticket.NewTicket("GEN-002", "Generated 2", "Description"),
	}

	// Save generated tickets
	outputPath := filepath.Join(tmpDir, ".tickets", "generated-tickets.json")
	if err := store.SaveGeneratedTickets(outputPath, tickets); err != nil {
		t.Fatalf("Failed to save generated tickets: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Generated tickets file not created: %v", err)
	}

	// Load generated tickets
	loaded, err := store.LoadGeneratedTickets(outputPath)
	if err != nil {
		t.Fatalf("Failed to load generated tickets: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("Expected 2 tickets, got %d", len(loaded))
	}

	// Verify content
	var data map[string]interface{}
	content, _ := os.ReadFile(outputPath)
	json.Unmarshal(content, &data)

	if _, ok := data["tickets"]; !ok {
		t.Error("Expected 'tickets' key in JSON")
	}
}
