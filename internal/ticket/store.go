package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Store handles ticket persistence. Tickets are stored as JSON files under baseDir,
// organized by status (pending, in_progress, completed, failed). A path cache
// speeds up Load/Save/Delete by avoiding directory scans.
type Store struct {
	baseDir   string
	pathCache map[string]string // ticket ID -> file path cache
	cacheMu   sync.RWMutex      // protects pathCache
}

// NewStore creates a Store with the given base directory (e.g. .tickets).
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir:   baseDir,
		pathCache: make(map[string]string),
	}
}

// Init creates the status subdirectories under baseDir (pending, in_progress, completed, failed).
// Call before Save or LoadByStatus. Directory permissions are 0700 to protect sensitive data.
func (s *Store) Init() error {
	dirs := []string{
		filepath.Join(s.baseDir, string(StatusPending)),
		filepath.Join(s.baseDir, string(StatusInProgress)),
		filepath.Join(s.baseDir, string(StatusCompleted)),
		filepath.Join(s.baseDir, string(StatusFailed)),
	}

	for _, dir := range dirs {
		// Use 0700 for ticket directories to protect sensitive data
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Save writes a ticket to the store under baseDir/<status>/<id>.json.
// If the ticket's status changed, the old file in the previous status directory is removed.
// Validates the ticket before saving. Updates the path cache.
func (s *Store) Save(t *Ticket) error {
	if err := t.Validate(); err != nil {
		return err
	}

	newPath := filepath.Join(s.baseDir, string(t.Status), t.ID+".json")

	// Check if we have a cached path for this ticket
	s.cacheMu.RLock()
	cachedPath, hasCached := s.pathCache[t.ID]
	s.cacheMu.RUnlock()

	// Only remove old file if status changed (path is different)
	if hasCached && cachedPath != newPath {
		if _, err := os.Stat(cachedPath); err == nil {
			if err := os.Remove(cachedPath); err != nil {
				return fmt.Errorf("failed to remove old ticket file: %w", err)
			}
		}
	} else if !hasCached {
		// No cache entry - this might be a new ticket or cache was cleared
		// Search other directories only if ticket might exist elsewhere
		for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
			if status != t.Status {
				oldPath := filepath.Join(s.baseDir, string(status), t.ID+".json")
				if _, err := os.Stat(oldPath); err == nil {
					if err := os.Remove(oldPath); err != nil {
						return fmt.Errorf("failed to remove old ticket file: %w", err)
					}
					break // Found and removed, no need to check other directories
				}
			}
		}
	}

	// Save to new location
	dir := filepath.Join(s.baseDir, string(t.Status))
	// Use 0700 for ticket directories to protect sensitive data
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create status directory: %w", err)
	}

	data, err := t.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal ticket: %w", err)
	}

	if err := os.WriteFile(newPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write ticket file: %w", err)
	}

	// Update cache with new path
	s.cacheMu.Lock()
	s.pathCache[t.ID] = newPath
	s.cacheMu.Unlock()

	return nil
}

// Load reads a ticket by ID. Uses the path cache when available; otherwise searches
// all status directories. Returns an error if the ticket is not found.
func (s *Store) Load(id string) (*Ticket, error) {
	// First check cache for known path
	s.cacheMu.RLock()
	cachedPath, hasCached := s.pathCache[id]
	s.cacheMu.RUnlock()

	if hasCached {
		if _, err := os.Stat(cachedPath); err == nil {
			data, err := os.ReadFile(cachedPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read ticket file: %w", err)
			}
			return FromJSON(data)
		}
		// Cache entry is stale, remove it and search
		s.cacheMu.Lock()
		delete(s.pathCache, id)
		s.cacheMu.Unlock()
	}

	// Search in all status directories
	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		path := filepath.Join(s.baseDir, string(status), id+".json")
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read ticket file: %w", err)
			}
			ticket, err := FromJSON(data)
			if err != nil {
				return nil, err
			}
			// Update cache
			s.cacheMu.Lock()
			s.pathCache[id] = path
			s.cacheMu.Unlock()
			return ticket, nil
		}
	}
	return nil, fmt.Errorf("ticket not found: %s", id)
}

// LoadByStatus loads all tickets in the given status directory, sorted by priority.
// Returns an empty slice if the directory does not exist.
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

// LoadAll loads tickets from all status directories and returns a TicketList.
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

// Delete removes the ticket file for the given ID. Uses the path cache when available.
// Returns an error if the ticket is not found.
func (s *Store) Delete(id string) error {
	// First check cache for known path
	s.cacheMu.RLock()
	cachedPath, hasCached := s.pathCache[id]
	s.cacheMu.RUnlock()

	if hasCached {
		if _, err := os.Stat(cachedPath); err == nil {
			if err := os.Remove(cachedPath); err != nil {
				return err
			}
			s.cacheMu.Lock()
			delete(s.pathCache, id)
			s.cacheMu.Unlock()
			return nil
		}
		// Cache entry is stale, remove it and search
		s.cacheMu.Lock()
		delete(s.pathCache, id)
		s.cacheMu.Unlock()
	}

	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		path := filepath.Join(s.baseDir, string(status), id+".json")
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}
	return fmt.Errorf("ticket not found: %s", id)
}

// CountByStatus returns the number of tickets with the given status by counting
// .json files in the status directory. It does not read or parse ticket JSON.
func (s *Store) CountByStatus(status Status) (int, error) {
	dir := filepath.Join(s.baseDir, string(status))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		count++
	}
	return count, nil
}

// Count returns the count of tickets per status using ReadDir only (no JSON parsing).
func (s *Store) Count() (map[Status]int, error) {
	counts := make(map[Status]int)

	for _, status := range []Status{StatusPending, StatusInProgress, StatusCompleted, StatusFailed} {
		n, err := s.CountByStatus(status)
		if err != nil {
			return nil, err
		}
		counts[status] = n
	}

	return counts, nil
}

// MoveToStatus loads the ticket by ID, sets its status to newStatus, and saves it.
// The file is moved from the old status directory to the new one.
func (s *Store) MoveToStatus(id string, newStatus Status) error {
	ticket, err := s.Load(id)
	if err != nil {
		return err
	}

	ticket.Status = newStatus
	return s.Save(ticket)
}

// MoveFailed loads all failed tickets, sets their status to pending and clears Error/CompletedAt, then saves.
// Returns the number of tickets moved.
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

// Clean removes the base directory and all ticket files.
func (s *Store) Clean() error {
	return os.RemoveAll(s.baseDir)
}

// SaveGeneratedTickets writes a ticket list (e.g. from planning) to the given path as JSON.
// Creates parent directories with 0700 if needed.
func (s *Store) SaveGeneratedTickets(path string, tickets []*Ticket) error {
	dir := filepath.Dir(path)
	// Use 0700 for ticket directories to protect sensitive data
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tl := &TicketList{Tickets: tickets}
	data, err := json.MarshalIndent(tl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tickets: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadGeneratedTickets reads a JSON file at path (e.g. generated-tickets.json) and returns the ticket list.
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
