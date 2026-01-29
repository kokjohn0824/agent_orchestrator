package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Store handles ticket persistence
type Store struct {
	baseDir string
}

// NewStore creates a new ticket store
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: baseDir,
	}
}

// Init initializes the store directories
func (s *Store) Init() error {
	dirs := []string{
		filepath.Join(s.baseDir, string(StatusPending)),
		filepath.Join(s.baseDir, string(StatusInProgress)),
		filepath.Join(s.baseDir, string(StatusCompleted)),
		filepath.Join(s.baseDir, string(StatusFailed)),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Save saves a ticket to the store
func (s *Store) Save(t *Ticket) error {
	if err := t.Validate(); err != nil {
		return err
	}

	// Remove from other status directories
	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		if status != t.Status {
			oldPath := filepath.Join(s.baseDir, string(status), t.ID+".json")
			if _, err := os.Stat(oldPath); err == nil {
				if err := os.Remove(oldPath); err != nil {
					return fmt.Errorf("failed to remove old ticket file: %w", err)
				}
			}
		}
	}

	// Save to new location
	dir := filepath.Join(s.baseDir, string(t.Status))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	data, err := t.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %w", err)
	}

	path := filepath.Join(dir, t.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write ticket file: %w", err)
	}

	return nil
}

// Load loads a ticket by ID
func (s *Store) Load(id string) (*Ticket, error) {
	// Search in all status directories
	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		path := filepath.Join(s.baseDir, string(status), id+".json")
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read ticket file: %w", err)
			}
			return FromJSON(data)
		}
	}
	return nil, fmt.Errorf("ticket not found: %s", id)
}

// LoadByStatus loads all tickets with the given status
func (s *Store) LoadByStatus(status Status) ([]*Ticket, error) {
	dir := filepath.Join(s.baseDir, string(status))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []*Ticket{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	tickets := make([]*Ticket, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		ticket, err := FromJSON(data)
		if err != nil {
			continue
		}
		tickets = append(tickets, ticket)
	}

	// Sort by priority
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].Priority < tickets[j].Priority
	})

	return tickets, nil
}

// LoadAll loads all tickets
func (s *Store) LoadAll() (*TicketList, error) {
	tl := NewTicketList()

	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		tickets, err := s.LoadByStatus(status)
		if err != nil {
			return nil, err
		}
		for _, t := range tickets {
			tl.Add(t)
		}
	}

	return tl, nil
}

// Delete removes a ticket from the store
func (s *Store) Delete(id string) error {
	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		path := filepath.Join(s.baseDir, string(status), id+".json")
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}
	return fmt.Errorf("ticket not found: %s", id)
}

// Count returns the count of tickets by status
func (s *Store) Count() (map[Status]int, error) {
	counts := make(map[Status]int)

	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		tickets, err := s.LoadByStatus(status)
		if err != nil {
			return nil, err
		}
		counts[status] = len(tickets)
	}

	return counts, nil
}

// MoveToStatus moves a ticket to a new status
func (s *Store) MoveToStatus(id string, newStatus Status) error {
	ticket, err := s.Load(id)
	if err != nil {
		return err
	}

	ticket.Status = newStatus
	return s.Save(ticket)
}

// MoveFailed moves all failed tickets back to pending
func (s *Store) MoveFailed() (int, error) {
	failed, err := s.LoadByStatus(StatusFailed)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, t := range failed {
		t.Status = StatusPending
		t.Error = ""
		t.CompletedAt = nil
		if err := s.Save(t); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// Clean removes all tickets and the base directory
func (s *Store) Clean() error {
	return os.RemoveAll(s.baseDir)
}

// SaveGeneratedTickets saves tickets from planning output
func (s *Store) SaveGeneratedTickets(path string, tickets []*Ticket) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tl := &TicketList{Tickets: tickets}
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tickets: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadGeneratedTickets loads tickets from planning output
func (s *Store) LoadGeneratedTickets(path string) ([]*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	tl, err := FromJSONList(data)
	if err != nil {
		return nil, err
	}

	return tl.Tickets, nil
}
