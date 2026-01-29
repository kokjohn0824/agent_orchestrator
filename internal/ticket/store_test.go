package ticket

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestStoreForStore creates a temporary store for testing
func setupTestStoreForStore(t *testing.T) (*Store, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "store-test-*")
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

// cleanupTestStoreForStore removes the temporary test directory
func cleanupTestStoreForStore(t *testing.T, tempDir string) {
	t.Helper()
	os.RemoveAll(tempDir)
}

func TestStore_Init(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		filepath.Join(tempDir, "pending"),
		filepath.Join(tempDir, "in_progress"),
		filepath.Join(tempDir, "completed"),
		filepath.Join(tempDir, "failed"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Init() did not create directory: %s", dir)
		}
	}
}

func TestStore_Save_Load(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	tests := []struct {
		name    string
		ticket  *Ticket
		wantErr bool
	}{
		{
			name: "save valid ticket",
			ticket: &Ticket{
				ID:     "TEST-001",
				Title:  "Test Ticket",
				Status: StatusPending,
			},
			wantErr: false,
		},
		{
			name: "save ticket with all fields",
			ticket: &Ticket{
				ID:                  "TEST-002",
				Title:               "Full Ticket",
				Description:         "A complete ticket",
				Type:                TypeFeature,
				Priority:            1,
				Status:              StatusInProgress,
				EstimatedComplexity: "high",
				Dependencies:        []string{"TEST-001"},
				AcceptanceCriteria:  []string{"Criterion 1", "Criterion 2"},
				FilesToCreate:       []string{"new.go"},
				FilesToModify:       []string{"existing.go"},
			},
			wantErr: false,
		},
		{
			name: "save ticket without ID - should fail",
			ticket: &Ticket{
				Title:  "No ID Ticket",
				Status: StatusPending,
			},
			wantErr: true,
		},
		{
			name: "save ticket without title - should fail",
			ticket: &Ticket{
				ID:     "TEST-003",
				Status: StatusPending,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(tt.ticket)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify ticket can be loaded
				loaded, err := store.Load(tt.ticket.ID)
				if err != nil {
					t.Errorf("Load() error = %v", err)
					return
				}

				if loaded.ID != tt.ticket.ID {
					t.Errorf("Load() ID = %v, want %v", loaded.ID, tt.ticket.ID)
				}
				if loaded.Title != tt.ticket.Title {
					t.Errorf("Load() Title = %v, want %v", loaded.Title, tt.ticket.Title)
				}
				if loaded.Status != tt.ticket.Status {
					t.Errorf("Load() Status = %v, want %v", loaded.Status, tt.ticket.Status)
				}
			}
		})
	}
}

func TestStore_Load_NotFound(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	_, err := store.Load("NONEXISTENT")
	if err == nil {
		t.Error("Load() expected error for non-existent ticket, got nil")
	}
}

func TestStore_LoadByStatus(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create tickets with different statuses
	pending1 := &Ticket{ID: "P1", Title: "Pending 1", Status: StatusPending, Priority: 2}
	pending2 := &Ticket{ID: "P2", Title: "Pending 2", Status: StatusPending, Priority: 1}
	completed := &Ticket{ID: "C1", Title: "Completed 1", Status: StatusCompleted, Priority: 1}

	store.Save(pending1)
	store.Save(pending2)
	store.Save(completed)

	tests := []struct {
		name      string
		status    Status
		wantCount int
	}{
		{"pending tickets", StatusPending, 2},
		{"completed tickets", StatusCompleted, 1},
		{"in_progress tickets", StatusInProgress, 0},
		{"failed tickets", StatusFailed, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tickets, err := store.LoadByStatus(tt.status)
			if err != nil {
				t.Errorf("LoadByStatus() error = %v", err)
				return
			}

			if len(tickets) != tt.wantCount {
				t.Errorf("LoadByStatus() count = %d, want %d", len(tickets), tt.wantCount)
			}
		})
	}

	// Verify sorting by priority
	pendingTickets, _ := store.LoadByStatus(StatusPending)
	if len(pendingTickets) == 2 {
		if pendingTickets[0].Priority > pendingTickets[1].Priority {
			t.Error("LoadByStatus() did not sort by priority")
		}
	}
}

func TestStore_LoadAll(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create tickets with different statuses
	tickets := []*Ticket{
		{ID: "P1", Title: "Pending", Status: StatusPending},
		{ID: "IP1", Title: "In Progress", Status: StatusInProgress},
		{ID: "C1", Title: "Completed", Status: StatusCompleted},
		{ID: "F1", Title: "Failed", Status: StatusFailed},
	}

	for _, t := range tickets {
		store.Save(t)
	}

	tl, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if tl.Count() != len(tickets) {
		t.Errorf("LoadAll() count = %d, want %d", tl.Count(), len(tickets))
	}
}

func TestStore_Delete(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "DELETE-ME", Title: "To Delete", Status: StatusPending}
	if err := store.Save(ticket); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify it exists
	if _, err := store.Load("DELETE-ME"); err != nil {
		t.Fatalf("ticket should exist before delete")
	}

	// Delete it
	if err := store.Delete("DELETE-ME"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify it's gone
	if _, err := store.Load("DELETE-ME"); err == nil {
		t.Error("ticket should not exist after delete")
	}

	// Delete non-existent ticket
	if err := store.Delete("NONEXISTENT"); err == nil {
		t.Error("Delete() should return error for non-existent ticket")
	}
}

func TestStore_MoveToStatus(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a pending ticket
	ticket := &Ticket{ID: "MOVE-ME", Title: "To Move", Status: StatusPending}
	store.Save(ticket)

	tests := []struct {
		name      string
		ticketID  string
		newStatus Status
		wantErr   bool
	}{
		{"move to in_progress", "MOVE-ME", StatusInProgress, false},
		{"move to completed", "MOVE-ME", StatusCompleted, false},
		{"move non-existent", "NONEXISTENT", StatusCompleted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.MoveToStatus(tt.ticketID, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("MoveToStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				loaded, _ := store.Load(tt.ticketID)
				if loaded.Status != tt.newStatus {
					t.Errorf("MoveToStatus() status = %v, want %v", loaded.Status, tt.newStatus)
				}

				// Verify old file is removed
				for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
					if status == tt.newStatus {
						continue
					}
					oldPath := filepath.Join(tempDir, string(status), tt.ticketID+".json")
					if _, err := os.Stat(oldPath); err == nil {
						t.Errorf("old file should not exist at %s", oldPath)
					}
				}
			}
		})
	}
}

func TestStore_Count(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create tickets
	tickets := []*Ticket{
		{ID: "P1", Title: "Pending 1", Status: StatusPending},
		{ID: "P2", Title: "Pending 2", Status: StatusPending},
		{ID: "IP1", Title: "In Progress", Status: StatusInProgress},
		{ID: "C1", Title: "Completed", Status: StatusCompleted},
	}

	for _, t := range tickets {
		store.Save(t)
	}

	counts, err := store.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	expectedCounts := map[Status]int{
		StatusPending:    2,
		StatusInProgress: 1,
		StatusCompleted:  1,
		StatusFailed:     0,
	}

	for status, expected := range expectedCounts {
		if counts[status] != expected {
			t.Errorf("Count()[%s] = %d, want %d", status, counts[status], expected)
		}
	}
}

func TestStore_MoveFailed(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create failed tickets
	failed1 := &Ticket{ID: "F1", Title: "Failed 1", Status: StatusFailed, Error: "error 1"}
	failed2 := &Ticket{ID: "F2", Title: "Failed 2", Status: StatusFailed, Error: "error 2"}
	store.Save(failed1)
	store.Save(failed2)

	count, err := store.MoveFailed()
	if err != nil {
		t.Fatalf("MoveFailed() error = %v", err)
	}

	if count != 2 {
		t.Errorf("MoveFailed() count = %d, want 2", count)
	}

	// Verify tickets are now pending
	pending, _ := store.LoadByStatus(StatusPending)
	if len(pending) != 2 {
		t.Errorf("After MoveFailed(), pending count = %d, want 2", len(pending))
	}

	// Verify error is cleared
	for _, p := range pending {
		if p.Error != "" {
			t.Errorf("After MoveFailed(), error should be cleared, got %s", p.Error)
		}
	}
}

func TestStore_Clean(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	// Note: don't defer cleanup since Clean will remove it

	// Create some tickets
	store.Save(&Ticket{ID: "T1", Title: "Test", Status: StatusPending})

	if err := store.Clean(); err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Error("Clean() should remove base directory")
		os.RemoveAll(tempDir) // cleanup anyway
	}
}

func TestStore_SaveGeneratedTickets_LoadGeneratedTickets(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	tickets := []*Ticket{
		{ID: "G1", Title: "Generated 1", Status: StatusPending},
		{ID: "G2", Title: "Generated 2", Status: StatusPending},
	}

	outputPath := filepath.Join(tempDir, "generated", "tickets.json")

	// Save
	if err := store.SaveGeneratedTickets(outputPath, tickets); err != nil {
		t.Fatalf("SaveGeneratedTickets() error = %v", err)
	}

	// Load
	loaded, err := store.LoadGeneratedTickets(outputPath)
	if err != nil {
		t.Fatalf("LoadGeneratedTickets() error = %v", err)
	}

	if len(loaded) != len(tickets) {
		t.Errorf("LoadGeneratedTickets() count = %d, want %d", len(loaded), len(tickets))
	}

	for i, l := range loaded {
		if l.ID != tickets[i].ID {
			t.Errorf("LoadGeneratedTickets()[%d].ID = %s, want %s", i, l.ID, tickets[i].ID)
		}
	}
}

func TestStore_LoadGeneratedTickets_NotFound(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	_, err := store.LoadGeneratedTickets(filepath.Join(tempDir, "nonexistent.json"))
	if err == nil {
		t.Error("LoadGeneratedTickets() should return error for non-existent file")
	}
}

func TestStore_Init_DirectoryPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "store-perm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewStore(tempDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify directories have restricted permissions (0700)
	expectedDirs := []string{
		filepath.Join(tempDir, "pending"),
		filepath.Join(tempDir, "in_progress"),
		filepath.Join(tempDir, "completed"),
		filepath.Join(tempDir, "failed"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("failed to stat directory %s: %v", dir, err)
			continue
		}

		perm := info.Mode().Perm()
		// On Unix, check that group and others don't have read/write/execute
		// 0700 means owner has rwx, group and others have nothing
		if perm&0077 != 0 {
			t.Errorf("directory %s has permissions %o, expected 0700 (no group/other access)", dir, perm)
		}
	}
}

func TestStore_SaveGeneratedTickets_DirectoryPermissions(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	tickets := []*Ticket{
		{ID: "G1", Title: "Generated 1", Status: StatusPending},
	}

	generatedDir := filepath.Join(tempDir, "generated_subdir")
	outputPath := filepath.Join(generatedDir, "tickets.json")

	if err := store.SaveGeneratedTickets(outputPath, tickets); err != nil {
		t.Fatalf("SaveGeneratedTickets() error = %v", err)
	}

	// Verify the created directory has restricted permissions
	info, err := os.Stat(generatedDir)
	if err != nil {
		t.Fatalf("failed to stat generated directory: %v", err)
	}

	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		t.Errorf("generated directory has permissions %o, expected 0700 (no group/other access)", perm)
	}
}

func TestStore_StatusTransition_FileMovement(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "TRANSITION", Title: "Test", Status: StatusPending}
	store.Save(ticket)

	// Verify file is in pending directory
	pendingPath := filepath.Join(tempDir, "pending", "TRANSITION.json")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Error("ticket should be in pending directory initially")
	}

	// Change status to in_progress and save
	ticket.Status = StatusInProgress
	store.Save(ticket)

	// Verify file moved
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("ticket should not be in pending directory after status change")
	}

	inProgressPath := filepath.Join(tempDir, "in_progress", "TRANSITION.json")
	if _, err := os.Stat(inProgressPath); os.IsNotExist(err) {
		t.Error("ticket should be in in_progress directory after status change")
	}
}

func TestStore_PathCache_SaveWithoutStatusChange(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "CACHE-TEST", Title: "Cache Test", Status: StatusPending}
	store.Save(ticket)

	pendingPath := filepath.Join(tempDir, "pending", "CACHE-TEST.json")
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Error("ticket should be in pending directory")
	}

	// Update ticket without changing status
	ticket.Title = "Updated Cache Test"
	store.Save(ticket)

	// Verify file is still in pending directory
	if _, err := os.Stat(pendingPath); os.IsNotExist(err) {
		t.Error("ticket should still be in pending directory after save without status change")
	}

	// Load and verify content was updated
	loaded, err := store.Load("CACHE-TEST")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Title != "Updated Cache Test" {
		t.Errorf("Load() Title = %v, want 'Updated Cache Test'", loaded.Title)
	}
}

func TestStore_PathCache_LoadUpdatesCache(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "LOAD-CACHE", Title: "Load Cache Test", Status: StatusPending}
	store.Save(ticket)

	// Create a new store instance (simulating fresh start with empty cache)
	store2 := NewStore(tempDir)

	// Load ticket - this should populate the cache
	_, err := store2.Load("LOAD-CACHE")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Now change status and save
	ticket.Status = StatusInProgress
	store2.Save(ticket)

	// Verify old file is removed
	pendingPath := filepath.Join(tempDir, "pending", "LOAD-CACHE.json")
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Error("old file should be removed after status change")
	}

	// Verify new file exists
	inProgressPath := filepath.Join(tempDir, "in_progress", "LOAD-CACHE.json")
	if _, err := os.Stat(inProgressPath); os.IsNotExist(err) {
		t.Error("ticket should be in in_progress directory")
	}
}

func TestStore_PathCache_DeleteClearsCache(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "DELETE-CACHE", Title: "Delete Cache Test", Status: StatusPending}
	store.Save(ticket)

	// Load to ensure cache is populated
	_, err := store.Load("DELETE-CACHE")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Delete the ticket
	if err := store.Delete("DELETE-CACHE"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify ticket is gone
	_, err = store.Load("DELETE-CACHE")
	if err == nil {
		t.Error("Load() should return error for deleted ticket")
	}
}

func TestStore_PathCache_MultipleStatusTransitions(t *testing.T) {
	store, tempDir := setupTestStoreForStore(t)
	defer cleanupTestStoreForStore(t, tempDir)

	// Create a ticket
	ticket := &Ticket{ID: "MULTI-TRANS", Title: "Multi Transition", Status: StatusPending}
	store.Save(ticket)

	// Transition through multiple statuses
	transitions := []Status{StatusInProgress, StatusCompleted, StatusFailed, StatusPending}

	for _, newStatus := range transitions {
		oldStatus := ticket.Status
		ticket.Status = newStatus
		store.Save(ticket)

		// Verify old location is cleaned up
		if oldStatus != newStatus {
			oldPath := filepath.Join(tempDir, string(oldStatus), "MULTI-TRANS.json")
			if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
				t.Errorf("old file should be removed after transition from %s to %s", oldStatus, newStatus)
			}
		}

		// Verify new location exists
		newPath := filepath.Join(tempDir, string(newStatus), "MULTI-TRANS.json")
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Errorf("ticket should be in %s directory", newStatus)
		}
	}
}
