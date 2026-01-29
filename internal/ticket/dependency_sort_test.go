package ticket

import (
	"os"
	"testing"
)

// setupTestStoreForDep creates a temporary store for testing
func setupTestStoreForDep(t *testing.T) (*Store, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "dep-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	store := NewStore(tempDir)
	if err := store.Init(); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to init store: %v", err)
	}

	return store, tempDir
}

// cleanupTestStoreForDep removes the temporary test directory
func cleanupTestStoreForDep(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestDependencyResolver_SortByDependency(t *testing.T) {
	tests := []struct {
		name     string
		tickets  []*Ticket
		wantIDs  []string // Expected order of ticket IDs
	}{
		{
			name:     "empty input",
			tickets:  []*Ticket{},
			wantIDs:  []string{},
		},
		{
			name: "single ticket no dependencies",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
			},
			wantIDs: []string{"A"},
		},
		{
			name: "linear dependency chain A -> B -> C",
			tickets: []*Ticket{
				{ID: "C", Dependencies: []string{"B"}},
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
			},
			wantIDs: []string{"A", "B", "C"},
		},
		{
			name: "multiple roots converging",
			tickets: []*Ticket{
				{ID: "D", Dependencies: []string{"B", "C"}},
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"A"}},
			},
			wantIDs: nil, // D must come after B and C, which must come after A
		},
		{
			name: "diamond dependency pattern",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"A"}},
				{ID: "D", Dependencies: []string{"B", "C"}},
			},
			wantIDs: nil, // A first, then B and C (any order), then D
		},
		{
			name: "external dependency ignored",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"EXTERNAL"}},
				{ID: "B", Dependencies: []string{"A"}},
			},
			wantIDs: []string{"A", "B"},
		},
		{
			name: "parallel independent tickets",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{}},
				{ID: "C", Dependencies: []string{}},
			},
			wantIDs: nil, // Any order is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, tempDir := setupTestStoreForDep(t)
			defer cleanupTestStoreForDep(t, tempDir)

			dr := NewDependencyResolver(store)
			got := dr.SortByDependency(tt.tickets)

			// Verify length
			if len(got) != len(tt.tickets) {
				t.Errorf("SortByDependency() returned %d tickets, want %d", len(got), len(tt.tickets))
				return
			}

			// If wantIDs is specified, verify exact order
			if tt.wantIDs != nil {
				gotIDs := make([]string, len(got))
				for i, ticket := range got {
					gotIDs[i] = ticket.ID
				}
				for i, wantID := range tt.wantIDs {
					if gotIDs[i] != wantID {
						t.Errorf("SortByDependency() got %v, want %v", gotIDs, tt.wantIDs)
						return
					}
				}
			} else {
				// Verify topological order is valid
				if !isValidTopologicalOrder(tt.tickets, got) {
					gotIDs := make([]string, len(got))
					for i, ticket := range got {
						gotIDs[i] = ticket.ID
					}
					t.Errorf("SortByDependency() returned invalid topological order: %v", gotIDs)
				}
			}
		})
	}
}

// isValidTopologicalOrder checks if the sorted order respects dependencies
func isValidTopologicalOrder(original, sorted []*Ticket) bool {
	// Build position map
	position := make(map[string]int)
	for i, t := range sorted {
		position[t.ID] = i
	}

	// Build ticket set for checking internal dependencies
	ticketSet := make(map[string]bool)
	for _, t := range original {
		ticketSet[t.ID] = true
	}

	// Check that all dependencies come before their dependents
	for _, t := range sorted {
		for _, depID := range t.Dependencies {
			// Only check internal dependencies
			if !ticketSet[depID] {
				continue
			}
			depPos, ok := position[depID]
			if !ok {
				return false
			}
			if depPos >= position[t.ID] {
				return false // Dependency must come before the ticket
			}
		}
	}
	return true
}

func TestDependencyResolver_HasCircularDependency(t *testing.T) {
	tests := []struct {
		name    string
		tickets []*Ticket
		want    bool
	}{
		{
			name:    "empty input - no cycle",
			tickets: []*Ticket{},
			want:    false,
		},
		{
			name: "single ticket no deps - no cycle",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
			},
			want: false,
		},
		{
			name: "self-referencing - cycle",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"A"}},
			},
			want: true,
		},
		{
			name: "two ticket cycle A <-> B",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"B"}},
				{ID: "B", Dependencies: []string{"A"}},
			},
			want: true,
		},
		{
			name: "three ticket cycle A -> B -> C -> A",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"C"}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"B"}},
			},
			want: true,
		},
		{
			name: "linear chain - no cycle",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"B"}},
			},
			want: false,
		},
		{
			name: "diamond pattern - no cycle",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"A"}},
				{ID: "D", Dependencies: []string{"B", "C"}},
			},
			want: false,
		},
		{
			name: "partial cycle in larger graph",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A", "D"}},
				{ID: "C", Dependencies: []string{"B"}},
				{ID: "D", Dependencies: []string{"C"}}, // Creates cycle: B -> C -> D -> B
			},
			want: true,
		},
		{
			name: "external dependency only - no cycle",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"EXTERNAL"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, tempDir := setupTestStoreForDep(t)
			defer cleanupTestStoreForDep(t, tempDir)

			dr := NewDependencyResolver(store)
			got := dr.HasCircularDependency(tt.tickets)
			if got != tt.want {
				t.Errorf("HasCircularDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDependencyResolver_ValidateDependencies(t *testing.T) {
	tests := []struct {
		name    string
		tickets []*Ticket
		wantErr bool
	}{
		{
			name:    "empty input",
			tickets: []*Ticket{},
			wantErr: false,
		},
		{
			name: "no dependencies",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{}},
			},
			wantErr: false,
		},
		{
			name: "valid internal dependencies",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"A"}},
				{ID: "C", Dependencies: []string{"A", "B"}},
			},
			wantErr: false,
		},
		{
			name: "unknown dependency",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{}},
				{ID: "B", Dependencies: []string{"UNKNOWN"}},
			},
			wantErr: true,
		},
		{
			name: "self-dependency is valid for validation (cycle detection is separate)",
			tickets: []*Ticket{
				{ID: "A", Dependencies: []string{"A"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, tempDir := setupTestStoreForDep(t)
			defer cleanupTestStoreForDep(t, tempDir)

			dr := NewDependencyResolver(store)
			err := dr.ValidateDependencies(tt.tickets)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDependencyResolver_CanProcess(t *testing.T) {
	store, tempDir := setupTestStoreForDep(t)
	defer cleanupTestStoreForDep(t, tempDir)

	// Create some completed tickets
	completedTicket := NewTicket("COMPLETED-1", "Completed", "A completed ticket")
	completedTicket.Status = StatusCompleted
	if err := store.Save(completedTicket); err != nil {
		t.Fatalf("failed to save completed ticket: %v", err)
	}

	dr := NewDependencyResolver(store)

	tests := []struct {
		name    string
		ticket  *Ticket
		want    bool
		wantErr bool
	}{
		{
			name:    "no dependencies - can process",
			ticket:  &Ticket{ID: "A", Dependencies: []string{}},
			want:    true,
			wantErr: false,
		},
		{
			name:    "dependency completed - can process",
			ticket:  &Ticket{ID: "B", Dependencies: []string{"COMPLETED-1"}},
			want:    true,
			wantErr: false,
		},
		{
			name:    "dependency not completed - cannot process",
			ticket:  &Ticket{ID: "C", Dependencies: []string{"PENDING-1"}},
			want:    false,
			wantErr: false,
		},
		{
			name:    "mixed dependencies - cannot process",
			ticket:  &Ticket{ID: "D", Dependencies: []string{"COMPLETED-1", "PENDING-1"}},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dr.CanProcess(tt.ticket)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanProcess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CanProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDependencyResolver_GetMissingDependencies(t *testing.T) {
	store, tempDir := setupTestStoreForDep(t)
	defer cleanupTestStoreForDep(t, tempDir)

	// Create completed tickets
	completed1 := NewTicket("COMPLETED-1", "Completed 1", "desc")
	completed1.Status = StatusCompleted
	completed2 := NewTicket("COMPLETED-2", "Completed 2", "desc")
	completed2.Status = StatusCompleted

	store.Save(completed1)
	store.Save(completed2)

	dr := NewDependencyResolver(store)

	tests := []struct {
		name    string
		ticket  *Ticket
		want    []string
		wantErr bool
	}{
		{
			name:    "no dependencies - no missing",
			ticket:  &Ticket{ID: "A", Dependencies: []string{}},
			want:    nil,
			wantErr: false,
		},
		{
			name:    "all dependencies completed - no missing",
			ticket:  &Ticket{ID: "B", Dependencies: []string{"COMPLETED-1", "COMPLETED-2"}},
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "some dependencies missing",
			ticket:  &Ticket{ID: "C", Dependencies: []string{"COMPLETED-1", "MISSING-1"}},
			want:    []string{"MISSING-1"},
			wantErr: false,
		},
		{
			name:    "all dependencies missing",
			ticket:  &Ticket{ID: "D", Dependencies: []string{"MISSING-1", "MISSING-2"}},
			want:    []string{"MISSING-1", "MISSING-2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dr.GetMissingDependencies(tt.ticket)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMissingDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("GetMissingDependencies() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("GetMissingDependencies() = %v, want %v", got, tt.want)
				return
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("GetMissingDependencies() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

func TestDependencyResolver_GetProcessable(t *testing.T) {
	store, tempDir := setupTestStoreForDep(t)
	defer cleanupTestStoreForDep(t, tempDir)

	// Create completed ticket
	completed := NewTicket("COMPLETED-1", "Completed", "desc")
	completed.Status = StatusCompleted
	store.Save(completed)

	// Create pending tickets
	pending1 := NewTicket("PENDING-1", "Pending 1", "desc") // No deps - processable
	pending1.Status = StatusPending
	store.Save(pending1)

	pending2 := NewTicket("PENDING-2", "Pending 2", "desc") // Depends on completed - processable
	pending2.Dependencies = []string{"COMPLETED-1"}
	pending2.Status = StatusPending
	store.Save(pending2)

	pending3 := NewTicket("PENDING-3", "Pending 3", "desc") // Depends on pending1 - not processable
	pending3.Dependencies = []string{"PENDING-1"}
	pending3.Status = StatusPending
	store.Save(pending3)

	dr := NewDependencyResolver(store)

	processable, err := dr.GetProcessable()
	if err != nil {
		t.Fatalf("GetProcessable() error = %v", err)
	}

	// Should return PENDING-1 and PENDING-2
	if len(processable) != 2 {
		t.Errorf("GetProcessable() returned %d tickets, want 2", len(processable))
	}

	ids := make(map[string]bool)
	for _, p := range processable {
		ids[p.ID] = true
	}

	if !ids["PENDING-1"] {
		t.Errorf("GetProcessable() should include PENDING-1")
	}
	if !ids["PENDING-2"] {
		t.Errorf("GetProcessable() should include PENDING-2")
	}
	if ids["PENDING-3"] {
		t.Errorf("GetProcessable() should not include PENDING-3")
	}
}

func TestDependencyResolver_GetBlockedTickets(t *testing.T) {
	store, tempDir := setupTestStoreForDep(t)
	defer cleanupTestStoreForDep(t, tempDir)

	// Create pending tickets
	pending1 := NewTicket("PENDING-1", "Pending 1", "desc")
	pending1.Status = StatusPending
	store.Save(pending1)

	pending2 := NewTicket("PENDING-2", "Pending 2", "desc")
	pending2.Dependencies = []string{"PENDING-1"}
	pending2.Status = StatusPending
	store.Save(pending2)

	pending3 := NewTicket("PENDING-3", "Pending 3", "desc")
	pending3.Dependencies = []string{"MISSING"}
	pending3.Status = StatusPending
	store.Save(pending3)

	dr := NewDependencyResolver(store)

	blocked, err := dr.GetBlockedTickets()
	if err != nil {
		t.Fatalf("GetBlockedTickets() error = %v", err)
	}

	// Should return PENDING-2 and PENDING-3 (both have unmet dependencies)
	if len(blocked) != 2 {
		ids := make([]string, len(blocked))
		for i, b := range blocked {
			ids[i] = b.ID
		}
		t.Errorf("GetBlockedTickets() returned %d tickets (%v), want 2", len(blocked), ids)
	}
}

func TestDependencyResolver_Integration(t *testing.T) {
	store, tempDir := setupTestStoreForDep(t)
	defer cleanupTestStoreForDep(t, tempDir)

	// Create a realistic ticket dependency graph
	// Setup -> Core -> Tests
	//       -> Docs
	tickets := []*Ticket{
		{ID: "SETUP", Title: "Project Setup", Dependencies: []string{}, Status: StatusPending},
		{ID: "CORE", Title: "Core Implementation", Dependencies: []string{"SETUP"}, Status: StatusPending},
		{ID: "TESTS", Title: "Add Tests", Dependencies: []string{"CORE"}, Status: StatusPending},
		{ID: "DOCS", Title: "Documentation", Dependencies: []string{"SETUP"}, Status: StatusPending},
	}

	for _, t := range tickets {
		store.Save(t)
	}

	dr := NewDependencyResolver(store)

	// Validate dependencies
	if err := dr.ValidateDependencies(tickets); err != nil {
		t.Errorf("ValidateDependencies() unexpected error: %v", err)
	}

	// Check no circular dependency
	if dr.HasCircularDependency(tickets) {
		t.Error("HasCircularDependency() = true, expected false")
	}

	// Sort by dependency
	sorted := dr.SortByDependency(tickets)
	if len(sorted) != len(tickets) {
		t.Errorf("SortByDependency() returned %d tickets, want %d", len(sorted), len(tickets))
	}

	// Verify SETUP comes first
	if sorted[0].ID != "SETUP" {
		t.Errorf("SortByDependency() first ticket = %s, want SETUP", sorted[0].ID)
	}

	// Get processable (only SETUP should be processable initially)
	processable, err := dr.GetProcessable()
	if err != nil {
		t.Fatalf("GetProcessable() error: %v", err)
	}
	if len(processable) != 1 || processable[0].ID != "SETUP" {
		ids := make([]string, len(processable))
		for i, p := range processable {
			ids[i] = p.ID
		}
		t.Errorf("GetProcessable() = %v, want [SETUP]", ids)
	}

	// Mark SETUP as completed
	setupTicket, _ := store.Load("SETUP")
	setupTicket.Status = StatusCompleted
	store.Save(setupTicket)

	// Now CORE and DOCS should be processable
	processable, _ = dr.GetProcessable()
	if len(processable) != 2 {
		t.Errorf("After completing SETUP, GetProcessable() returned %d tickets, want 2", len(processable))
	}
}
