package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table represents a simple table renderer
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table with the given headers
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		widths:  widths,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	// Pad or truncate cells to match header count
	row := make([]string, len(t.headers))
	for i := range row {
		if i < len(cells) {
			row[i] = cells[i]
			if len(cells[i]) > t.widths[i] {
				t.widths[i] = len(cells[i])
			}
		}
	}
	t.rows = append(t.rows, row)
}

// Render renders the table to the writer
func (t *Table) Render(w io.Writer) {
	if len(t.headers) == 0 {
		return
	}

	// Build the header
	headerCells := make([]string, len(t.headers))
	for i, h := range t.headers {
		headerCells[i] = StyleBold.Render(padRight(h, t.widths[i]))
	}
	headerLine := strings.Join(headerCells, "  ")

	// Build separator
	sepParts := make([]string, len(t.widths))
	for i, w := range t.widths {
		sepParts[i] = strings.Repeat("─", w)
	}
	separator := StyleMuted.Render(strings.Join(sepParts, "──"))

	// Print header
	fmt.Fprintln(w, headerLine)
	fmt.Fprintln(w, separator)

	// Print rows
	for _, row := range t.rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = padRight(cell, t.widths[i])
		}
		fmt.Fprintln(w, strings.Join(cells, "  "))
	}
}

// String returns the table as a string
func (t *Table) String() string {
	var sb strings.Builder
	t.Render(&sb)
	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// StatusTable renders a summary table of ticket statuses
type StatusTable struct {
	pending    int
	inProgress int
	completed  int
	failed     int
}

// NewStatusTable creates a new status table
func NewStatusTable() *StatusTable {
	return &StatusTable{}
}

// SetCounts sets the counts for each status
func (st *StatusTable) SetCounts(pending, inProgress, completed, failed int) {
	st.pending = pending
	st.inProgress = inProgress
	st.completed = completed
	st.failed = failed
}

// Render renders the status table
func (st *StatusTable) Render(w io.Writer) {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	total := st.pending + st.inProgress + st.completed + st.failed

	content := fmt.Sprintf(
		"%s Pending:     %d\n%s In Progress: %d\n%s Completed:   %d\n%s Failed:      %d\n%s\n%s Total:       %d",
		StatusPending, st.pending,
		StatusInProgress, st.inProgress,
		StatusCompleted, st.completed,
		StatusFailed, st.failed,
		StyleMuted.Render("─────────────────"),
		StyleBold.Render(""), total,
	)

	fmt.Fprintln(w, box.Render(content))
}

// IssueTable renders a table of issues from analyze command
type IssueTable struct {
	title  string
	issues []Issue
}

// Issue represents an issue found by analyze
type Issue struct {
	Severity    string
	Description string
	Location    string
}

// NewIssueTable creates a new issue table
func NewIssueTable(title string) *IssueTable {
	return &IssueTable{
		title:  title,
		issues: make([]Issue, 0),
	}
}

// AddIssue adds an issue to the table
func (it *IssueTable) AddIssue(severity, description, location string) {
	it.issues = append(it.issues, Issue{
		Severity:    severity,
		Description: description,
		Location:    location,
	})
}

// Count returns the number of issues
func (it *IssueTable) Count() int {
	return len(it.issues)
}

// Render renders the issue table
func (it *IssueTable) Render(w io.Writer) {
	if len(it.issues) == 0 {
		return
	}

	header := HeaderStyle.Render(fmt.Sprintf("%s (%d)", it.title, len(it.issues)))
	fmt.Fprintln(w, header)

	for _, issue := range it.issues {
		severity := SeverityStyle(issue.Severity).Render(fmt.Sprintf("[%s]", issue.Severity))
		fmt.Fprintf(w, "  • %s %s - %s\n", severity, issue.Description, StyleMuted.Render(issue.Location))
	}
	fmt.Fprintln(w)
}
