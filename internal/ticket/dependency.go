package ticket

import (
	"fmt"
)

// ResolverContext holds a cached set of completed ticket IDs for dependency resolution.
// Create once with NewResolverContext(store), then pass to CanProcessWithContext,
// GetProcessableWithContext, GetBlockedTicketsWithContext, and GetMissingDependenciesWithContext
// to avoid repeated LoadByStatus(StatusCompleted) when checking many tickets.
type ResolverContext struct {
	completedIDs map[string]bool
}

// NewResolverContext loads all completed tickets from the store and builds a context
// mapping their IDs to true. Returns an error if LoadByStatus fails.
func NewResolverContext(store *Store) (*ResolverContext, error) {
	completed, err := store.LoadByStatus(StatusCompleted)
	if err != nil {
		return nil, err
	}

	completedIDs := make(map[string]bool)
	for _, t := range completed {
		completedIDs[t.ID] = true
	}

	return &ResolverContext{
		completedIDs: completedIDs,
	}, nil
}

// IsCompleted reports whether the given ticket ID is in the completed set.
func (rc *ResolverContext) IsCompleted(id string) bool {
	return rc.completedIDs[id]
}

// DependencyResolver answers dependency questions for tickets (can process, processable list,
// blocked list, missing dependencies, topological sort). It uses the Store to load completed
// tickets; for batch checks use ResolverContext and the WithContext methods to avoid repeated I/O.
type DependencyResolver struct {
	store *Store
}

// NewDependencyResolver creates a DependencyResolver that uses the given Store.
func NewDependencyResolver(store *Store) *DependencyResolver {
	return &DependencyResolver{
		store: store,
	}
}

// CanProcess reports whether the ticket can be processed (all dependencies are completed).
// It creates a ResolverContext internally; for many tickets use NewResolverContext once
// and CanProcessWithContext to avoid repeated store access.
func (dr *DependencyResolver) CanProcess(ticket *Ticket) (bool, error) {
	ctx, err := NewResolverContext(dr.store)
	if err != nil {
		return false, err
	}
	return dr.CanProcessWithContext(ticket, ctx), nil
}

// CanProcessWithContext reports whether the ticket can be processed using the cached
// completed set in ctx. Use this when checking many tickets: create ctx once with
// NewResolverContext(store), then call CanProcessWithContext for each ticket.
func (dr *DependencyResolver) CanProcessWithContext(ticket *Ticket, ctx *ResolverContext) bool {
	if len(ticket.Dependencies) == 0 {
		return true
	}

	for _, depID := range ticket.Dependencies {
		if !ctx.IsCompleted(depID) {
			return false
		}
	}

	return true
}

// GetProcessable returns all pending tickets whose dependencies are all completed.
// It builds a ResolverContext internally; for repeated use prefer NewResolverContext
// and GetProcessableWithContext.
func (dr *DependencyResolver) GetProcessable() ([]*Ticket, error) {
	ctx, err := NewResolverContext(dr.store)
	if err != nil {
		return nil, err
	}
	return dr.GetProcessableWithContext(ctx)
}

// GetProcessableWithContext returns all pending tickets that can be processed using
// the completed set in ctx. Use with the same ctx when you already have it (e.g. after
// GetBlockedTicketsWithContext) to avoid extra store access.
func (dr *DependencyResolver) GetProcessableWithContext(ctx *ResolverContext) ([]*Ticket, error) {
	pending, err := dr.store.LoadByStatus(StatusPending)
	if err != nil {
		return nil, err
	}

	processable := make([]*Ticket, 0)
	for _, t := range pending {
		if dr.CanProcessWithContext(t, ctx) {
			processable = append(processable, t)
		}
	}

	return processable, nil
}

// GetBlockedTickets returns all pending tickets that are blocked (at least one dependency not completed).
func (dr *DependencyResolver) GetBlockedTickets() ([]*Ticket, error) {
	ctx, err := NewResolverContext(dr.store)
	if err != nil {
		return nil, err
	}
	return dr.GetBlockedTicketsWithContext(ctx)
}

// GetBlockedTicketsWithContext returns all pending tickets that are blocked, using
// the completed set in ctx. Use the same ctx for GetProcessableWithContext if needed.
func (dr *DependencyResolver) GetBlockedTicketsWithContext(ctx *ResolverContext) ([]*Ticket, error) {
	pending, err := dr.store.LoadByStatus(StatusPending)
	if err != nil {
		return nil, err
	}

	blocked := make([]*Ticket, 0)
	for _, t := range pending {
		if !dr.CanProcessWithContext(t, ctx) {
			blocked = append(blocked, t)
		}
	}

	return blocked, nil
}

// GetMissingDependencies returns the list of dependency IDs that are not yet completed for the ticket.
// For many tickets, use NewResolverContext once and GetMissingDependenciesWithContext.
func (dr *DependencyResolver) GetMissingDependencies(ticket *Ticket) ([]string, error) {
	ctx, err := NewResolverContext(dr.store)
	if err != nil {
		return nil, err
	}
	return dr.GetMissingDependenciesWithContext(ticket, ctx), nil
}

// GetMissingDependenciesWithContext returns the dependency IDs that are not in the
// completed set in ctx. Use with the same ResolverContext when batching checks.
func (dr *DependencyResolver) GetMissingDependenciesWithContext(ticket *Ticket, ctx *ResolverContext) []string {
	if len(ticket.Dependencies) == 0 {
		return nil
	}

	missing := make([]string, 0)
	for _, depID := range ticket.Dependencies {
		if !ctx.IsCompleted(depID) {
			missing = append(missing, depID)
		}
	}

	return missing
}

// ValidateDependencies checks that every dependency ID referenced by any ticket in tickets
// is present in the same slice. Returns an error if a ticket references an unknown dependency.
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

// SortByDependency sorts tickets so that dependencies come first using topological sort.
//
// This function implements Kahn's algorithm for topological sorting, which processes
// a directed acyclic graph (DAG) to produce a linear ordering where for every edge (u, v),
// vertex u comes before vertex v in the ordering.
//
// # Algorithm Overview (Kahn's Algorithm)
//
//  1. Build adjacency list: Create a graph where each edge (A -> B) means "B depends on A",
//     so A must be processed before B.
//
//  2. Calculate in-degrees: For each node, count how many edges point to it (i.e., how many
//     dependencies it has within the given ticket set).
//
//  3. Initialize queue: Add all nodes with in-degree 0 (no dependencies) to the processing queue.
//
//  4. Process queue: Repeatedly remove a node from the queue, add it to the result,
//     and decrement the in-degree of all its neighbors. When a neighbor's in-degree
//     becomes 0, add it to the queue.
//
//  5. Return result: The result contains tickets in dependency order.
//
// # Time Complexity
//
// O(V + E) where V is the number of tickets and E is the total number of dependency edges.
// - Building the graph: O(V + E)
// - Processing all nodes and edges: O(V + E)
//
// # Space Complexity
//
// O(V + E) for storing the adjacency list, in-degree map, ticket map, and queue.
//
// # Edge Cases
//
//   - Empty input: Returns an empty slice.
//   - Single ticket with no dependencies: Returns the ticket in a single-element slice.
//   - Tickets with external dependencies (not in the input set): External dependencies are
//     ignored; only dependencies within the provided ticket set are considered.
//   - Circular dependencies: If a cycle exists, the algorithm cannot process all tickets,
//     and the returned slice will contain fewer tickets than the input. Use HasCircularDependency
//     to detect this condition.
//
// # Example
//
// Given tickets A, B, C where B depends on A, and C depends on B:
//
//	Input:  [C, A, B] (any order)
//	Output: [A, B, C] (dependency order)
func (dr *DependencyResolver) SortByDependency(tickets []*Ticket) []*Ticket {
	// Build dependency graph
	// graph[X] contains all tickets that depend on X (X must be completed before them)
	graph := make(map[string][]string)
	// inDegree[X] counts how many dependencies X has within the given ticket set
	inDegree := make(map[string]int)
	// ticketMap provides O(1) lookup from ticket ID to ticket pointer
	ticketMap := make(map[string]*Ticket)

	// Initialize all tickets with in-degree 0 and empty adjacency lists
	for _, t := range tickets {
		ticketMap[t.ID] = t
		graph[t.ID] = make([]string, 0)
		if _, ok := inDegree[t.ID]; !ok {
			inDegree[t.ID] = 0
		}
	}

	// Build edges: for each dependency relationship, add an edge from dependency to dependent
	// and increment the dependent's in-degree
	for _, t := range tickets {
		for _, depID := range t.Dependencies {
			// Only consider dependencies that exist within the provided ticket set
			if _, ok := graph[depID]; ok {
				graph[depID] = append(graph[depID], t.ID)
				inDegree[t.ID]++
			}
		}
	}

	// Kahn's algorithm: start with all nodes that have no dependencies (in-degree = 0)
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Process the queue: each iteration removes one ticket and updates its dependents
	sorted := make([]*Ticket, 0)
	for len(queue) > 0 {
		// Dequeue the first element (FIFO order)
		id := queue[0]
		queue = queue[1:]

		// Add the ticket to the sorted result
		if t, ok := ticketMap[id]; ok {
			sorted = append(sorted, t)
		}

		// For each ticket that depends on the current one, decrement its in-degree
		// If in-degree becomes 0, all its dependencies are satisfied, so add to queue
		for _, neighbor := range graph[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Note: If len(sorted) < len(tickets), a circular dependency exists
	// because some tickets could never have their in-degree reduced to 0
	return sorted
}

// HasCircularDependency checks if there are circular dependencies among the given tickets.
//
// This function leverages the property of Kahn's algorithm: if a directed graph contains
// a cycle, the topological sort cannot include all nodes because nodes in a cycle will
// always have a non-zero in-degree (each node in the cycle depends on another node in the cycle).
//
// # Detection Principle
//
// In Kahn's algorithm, a node is only added to the result when its in-degree becomes 0.
// In a cycle (e.g., A -> B -> C -> A), every node always has at least one incoming edge
// from another node in the cycle, so no node in the cycle can ever reach in-degree 0.
// Therefore, if the sorted result contains fewer tickets than the input, a cycle exists.
//
// # Time Complexity
//
// O(V + E) where V is the number of tickets and E is the total number of dependency edges.
// This is the same as SortByDependency since it internally calls that function.
//
// # Space Complexity
//
// O(V + E) inherited from SortByDependency.
//
// # Edge Cases
//
//   - Empty input: Returns false (no cycle in an empty graph).
//   - Single ticket depending on itself: Returns true (self-loop is a cycle).
//   - Multiple disconnected components: Detects cycles in any component.
//   - External dependencies (not in ticket set): Ignored; only cycles within the
//     provided ticket set are detected.
//
// # Example
//
//	// No cycle: A -> B -> C
//	HasCircularDependency([A, B, C]) // returns false
//
//	// Cycle: A -> B -> C -> A
//	HasCircularDependency([A, B, C]) // returns true
func (dr *DependencyResolver) HasCircularDependency(tickets []*Ticket) bool {
	sorted := dr.SortByDependency(tickets)
	// If topological sort couldn't process all tickets, a cycle exists
	return len(sorted) != len(tickets)
}
