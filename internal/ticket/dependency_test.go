package ticket

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ticket-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	store := NewStore(tmpDir)
	if err := store.Init(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init store: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestResolverContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed tickets
	t1 := NewTicket("T1", "Task 1", "Description 1")
	t1.Status = StatusCompleted
	if err := store.Save(t1); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	t2 := NewTicket("T2", "Task 2", "Description 2")
	t2.Status = StatusCompleted
	if err := store.Save(t2); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	// Create pending ticket
	t3 := NewTicket("T3", "Task 3", "Description 3")
	t3.Status = StatusPending
	if err := store.Save(t3); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	// Test ResolverContext
	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	if !ctx.IsCompleted("T1") {
		t.Error("expected T1 to be completed")
	}
	if !ctx.IsCompleted("T2") {
		t.Error("expected T2 to be completed")
	}
	if ctx.IsCompleted("T3") {
		t.Error("expected T3 to not be completed")
	}
	if ctx.IsCompleted("T4") {
		t.Error("expected non-existent T4 to not be completed")
	}
}

func TestCanProcessWithContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed tickets
	t1 := NewTicket("T1", "Task 1", "Description 1")
	t1.Status = StatusCompleted
	if err := store.Save(t1); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	dr := NewDependencyResolver(store)

	// Test ticket with no dependencies
	t2 := NewTicket("T2", "Task 2", "Description 2")
	t2.Dependencies = []string{}
	if !dr.CanProcessWithContext(t2, ctx) {
		t.Error("expected ticket with no dependencies to be processable")
	}

	// Test ticket with satisfied dependency
	t3 := NewTicket("T3", "Task 3", "Description 3")
	t3.Dependencies = []string{"T1"}
	if !dr.CanProcessWithContext(t3, ctx) {
		t.Error("expected ticket with satisfied dependency to be processable")
	}

	// Test ticket with unsatisfied dependency
	t4 := NewTicket("T4", "Task 4", "Description 4")
	t4.Dependencies = []string{"T999"}
	if dr.CanProcessWithContext(t4, ctx) {
		t.Error("expected ticket with unsatisfied dependency to not be processable")
	}

	// Test ticket with mixed dependencies
	t5 := NewTicket("T5", "Task 5", "Description 5")
	t5.Dependencies = []string{"T1", "T999"}
	if dr.CanProcessWithContext(t5, ctx) {
		t.Error("expected ticket with mixed dependencies to not be processable")
	}
}

func TestGetMissingDependenciesWithContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed ticket
	t1 := NewTicket("T1", "Task 1", "Description 1")
	t1.Status = StatusCompleted
	if err := store.Save(t1); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	dr := NewDependencyResolver(store)

	// Test ticket with no dependencies
	t2 := NewTicket("T2", "Task 2", "Description 2")
	t2.Dependencies = []string{}
	missing := dr.GetMissingDependenciesWithContext(t2, ctx)
	if len(missing) != 0 {
		t.Errorf("expected no missing dependencies, got %v", missing)
	}

	// Test ticket with all satisfied dependencies
	t3 := NewTicket("T3", "Task 3", "Description 3")
	t3.Dependencies = []string{"T1"}
	missing = dr.GetMissingDependenciesWithContext(t3, ctx)
	if len(missing) != 0 {
		t.Errorf("expected no missing dependencies, got %v", missing)
	}

	// Test ticket with unsatisfied dependencies
	t4 := NewTicket("T4", "Task 4", "Description 4")
	t4.Dependencies = []string{"T1", "T999", "T888"}
	missing = dr.GetMissingDependenciesWithContext(t4, ctx)
	if len(missing) != 2 {
		t.Errorf("expected 2 missing dependencies, got %v", missing)
	}
}

func TestGetProcessableWithContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed ticket
	t1 := NewTicket("T1", "Task 1", "Description 1")
	t1.Status = StatusCompleted
	if err := store.Save(t1); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	// Create pending tickets
	t2 := NewTicket("T2", "Task 2", "Description 2")
	t2.Status = StatusPending
	t2.Dependencies = []string{"T1"} // satisfied
	if err := store.Save(t2); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	t3 := NewTicket("T3", "Task 3", "Description 3")
	t3.Status = StatusPending
	t3.Dependencies = []string{"T999"} // not satisfied
	if err := store.Save(t3); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	t4 := NewTicket("T4", "Task 4", "Description 4")
	t4.Status = StatusPending
	t4.Dependencies = []string{} // no dependencies
	if err := store.Save(t4); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	dr := NewDependencyResolver(store)

	processable, err := dr.GetProcessableWithContext(ctx)
	if err != nil {
		t.Fatalf("failed to get processable tickets: %v", err)
	}

	if len(processable) != 2 {
		t.Errorf("expected 2 processable tickets, got %d", len(processable))
	}

	// Verify T2 and T4 are processable, T3 is not
	processableIDs := make(map[string]bool)
	for _, ticket := range processable {
		processableIDs[ticket.ID] = true
	}

	if !processableIDs["T2"] {
		t.Error("expected T2 to be processable")
	}
	if !processableIDs["T4"] {
		t.Error("expected T4 to be processable")
	}
	if processableIDs["T3"] {
		t.Error("expected T3 to not be processable")
	}
}

func TestGetBlockedTicketsWithContext(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed ticket
	t1 := NewTicket("T1", "Task 1", "Description 1")
	t1.Status = StatusCompleted
	if err := store.Save(t1); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	// Create pending tickets
	t2 := NewTicket("T2", "Task 2", "Description 2")
	t2.Status = StatusPending
	t2.Dependencies = []string{"T1"} // satisfied
	if err := store.Save(t2); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	t3 := NewTicket("T3", "Task 3", "Description 3")
	t3.Status = StatusPending
	t3.Dependencies = []string{"T999"} // not satisfied
	if err := store.Save(t3); err != nil {
		t.Fatalf("failed to save ticket: %v", err)
	}

	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	dr := NewDependencyResolver(store)

	blocked, err := dr.GetBlockedTicketsWithContext(ctx)
	if err != nil {
		t.Fatalf("failed to get blocked tickets: %v", err)
	}

	if len(blocked) != 1 {
		t.Errorf("expected 1 blocked ticket, got %d", len(blocked))
	}

	if blocked[0].ID != "T3" {
		t.Errorf("expected blocked ticket to be T3, got %s", blocked[0].ID)
	}
}

// TestContextReuse verifies that the same context can be reused for multiple operations
func TestContextReuse(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create completed tickets
	for i := 1; i <= 10; i++ {
		ticket := NewTicket(filepath.Join("T", string(rune('0'+i))), "Task", "Description")
		ticket.ID = "T" + string(rune('0'+i))
		ticket.Status = StatusCompleted
		if err := store.Save(ticket); err != nil {
			t.Fatalf("failed to save ticket: %v", err)
		}
	}

	// Create a single context
	ctx, err := NewResolverContext(store)
	if err != nil {
		t.Fatalf("failed to create resolver context: %v", err)
	}

	dr := NewDependencyResolver(store)

	// Reuse context for multiple operations
	for i := 1; i <= 10; i++ {
		id := "T" + string(rune('0'+i))
		if !ctx.IsCompleted(id) {
			t.Errorf("expected %s to be completed", id)
		}

		ticket := NewTicket("NEW", "New Task", "Description")
		ticket.Dependencies = []string{id}
		if !dr.CanProcessWithContext(ticket, ctx) {
			t.Errorf("expected ticket with dependency on %s to be processable", id)
		}

		missing := dr.GetMissingDependenciesWithContext(ticket, ctx)
		if len(missing) != 0 {
			t.Errorf("expected no missing dependencies for ticket depending on %s", id)
		}
	}
}
