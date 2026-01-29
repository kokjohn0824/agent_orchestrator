package ticket

import (
	"fmt"
)

// DependencyResolver resolves ticket dependencies
type DependencyResolver struct {
	store *Store
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(store *Store) *DependencyResolver {
	return &DependencyResolver{
		store: store,
	}
}

// CanProcess checks if a ticket can be processed (all dependencies completed)
func (dr *DependencyResolver) CanProcess(ticket *Ticket) (bool, error) {
	if len(ticket.Dependencies) == 0 {
		return true, nil
	}

	completed, err := dr.store.LoadByStatus(StatusCompleted)
	if err != nil {
		return false, err
	}

	completedIDs := make(map[string]bool)
	for _, t := range completed {
		completedIDs[t.ID] = true
	}

	for _, depID := range ticket.Dependencies {
		if !completedIDs[depID] {
			return false, nil
		}
	}

	return true, nil
}

// GetProcessable returns all tickets that can be processed
func (dr *DependencyResolver) GetProcessable() ([]*Ticket, error) {
	pending, err := dr.store.LoadByStatus(StatusPending)
	if err != nil {
		return nil, err
	}

	processable := make([]*Ticket, 0)
	for _, t := range pending {
		can, err := dr.CanProcess(t)
		if err != nil {
			return nil, err
		}
		if can {
			processable = append(processable, t)
		}
	}

	return processable, nil
}

// GetBlockedTickets returns all tickets that are blocked by dependencies
func (dr *DependencyResolver) GetBlockedTickets() ([]*Ticket, error) {
	pending, err := dr.store.LoadByStatus(StatusPending)
	if err != nil {
		return nil, err
	}

	blocked := make([]*Ticket, 0)
	for _, t := range pending {
		can, err := dr.CanProcess(t)
		if err != nil {
			return nil, err
		}
		if !can {
			blocked = append(blocked, t)
		}
	}

	return blocked, nil
}

// GetMissingDependencies returns the missing dependencies for a ticket
func (dr *DependencyResolver) GetMissingDependencies(ticket *Ticket) ([]string, error) {
	if len(ticket.Dependencies) == 0 {
		return nil, nil
	}

	completed, err := dr.store.LoadByStatus(StatusCompleted)
	if err != nil {
		return nil, err
	}

	completedIDs := make(map[string]bool)
	for _, t := range completed {
		completedIDs[t.ID] = true
	}

	missing := make([]string, 0)
	for _, depID := range ticket.Dependencies {
		if !completedIDs[depID] {
			missing = append(missing, depID)
		}
	}

	return missing, nil
}

// ValidateDependencies validates that all dependencies exist
func (dr *DependencyResolver) ValidateDependencies(tickets []*Ticket) error {
	ticketIDs := make(map[string]bool)
	for _, t := range tickets {
		ticketIDs[t.ID] = true
	}

	for _, t := range tickets {
		for _, depID := range t.Dependencies {
			if !ticketIDs[depID] {
				return fmt.Errorf("ticket %s has unknown dependency: %s", t.ID, depID)
			}
		}
	}

	return nil
}

// SortByDependency sorts tickets so that dependencies come first
func (dr *DependencyResolver) SortByDependency(tickets []*Ticket) []*Ticket {
	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)
	ticketMap := make(map[string]*Ticket)

	for _, t := range tickets {
		ticketMap[t.ID] = t
		graph[t.ID] = make([]string, 0)
		if _, ok := inDegree[t.ID]; !ok {
			inDegree[t.ID] = 0
		}
	}

	for _, t := range tickets {
		for _, depID := range t.Dependencies {
			if _, ok := graph[depID]; ok {
				graph[depID] = append(graph[depID], t.ID)
				inDegree[t.ID]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	sorted := make([]*Ticket, 0)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		if t, ok := ticketMap[id]; ok {
			sorted = append(sorted, t)
		}

		for _, neighbor := range graph[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return sorted
}

// HasCircularDependency checks if there are circular dependencies
func (dr *DependencyResolver) HasCircularDependency(tickets []*Ticket) bool {
	sorted := dr.SortByDependency(tickets)
	return len(sorted) != len(tickets)
}
