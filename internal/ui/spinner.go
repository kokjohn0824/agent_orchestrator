package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// TaskStatus represents the status of a task in MultiSpinner
type TaskStatus int

const (
	TaskStatusRunning TaskStatus = iota
	TaskStatusSuccess
	TaskStatusFailed
)

// Task represents a single task in the MultiSpinner
type Task struct {
	ID      string
	Message string
	Status  TaskStatus
}

// MultiSpinner manages multiple spinner tasks with each on its own line
type MultiSpinner struct {
	frames   []string
	interval time.Duration
	writer   io.Writer
	tasks    map[string]*Task
	order    []string // maintain insertion order
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewMultiSpinner creates a new multi-task spinner
func NewMultiSpinner(w io.Writer) *MultiSpinner {
	return &MultiSpinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		writer:   w,
		tasks:    make(map[string]*Task),
		order:    make([]string, 0),
	}
}

// AddTask adds a new task to the spinner
func (m *MultiSpinner) AddTask(id, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[id]; !exists {
		m.order = append(m.order, id)
	}
	m.tasks[id] = &Task{
		ID:      id,
		Message: message,
		Status:  TaskStatusRunning,
	}
}

// UpdateTask updates an existing task's message
func (m *MultiSpinner) UpdateTask(id, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, exists := m.tasks[id]; exists {
		task.Message = message
	}
}

// CompleteTask marks a task as successful
func (m *MultiSpinner) CompleteTask(id, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, exists := m.tasks[id]; exists {
		task.Status = TaskStatusSuccess
		task.Message = message
	}
}

// FailTask marks a task as failed
func (m *MultiSpinner) FailTask(id, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, exists := m.tasks[id]; exists {
		task.Status = TaskStatusFailed
		task.Message = message
	}
}

// Start begins the multi-spinner animation
func (m *MultiSpinner) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stop = make(chan struct{})
	m.done = make(chan struct{})
	m.mu.Unlock()

	go func() {
		defer close(m.done)
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		frameIdx := 0
		m.render(frameIdx)
		frameIdx++

		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				m.render(frameIdx)
				frameIdx++
			}
		}
	}()
}

// render draws all tasks
func (m *MultiSpinner) render(frameIdx int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.order) == 0 {
		return
	}

	// Move cursor up to the first task line (if not first render)
	if frameIdx > 0 {
		fmt.Fprintf(m.writer, "\033[%dA", len(m.order))
	}

	// Render each task on its own line
	for _, id := range m.order {
		task := m.tasks[id]
		var prefix string

		switch task.Status {
		case TaskStatusRunning:
			frame := m.frames[frameIdx%len(m.frames)]
			prefix = StyleInfo.Render(frame)
		case TaskStatusSuccess:
			prefix = StyleSuccess.Render("✓")
		case TaskStatusFailed:
			prefix = StyleError.Render("✗")
		}

		// Clear line and print task
		fmt.Fprintf(m.writer, "\033[K%s %s\n", prefix, task.Message)
	}
}

// Stop stops the multi-spinner
func (m *MultiSpinner) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stop)
	<-m.done
}

// HasRunningTasks checks if there are still running tasks
func (m *MultiSpinner) HasRunningTasks() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range m.tasks {
		if task.Status == TaskStatusRunning {
			return true
		}
	}
	return false
}

// RemoveTask removes a task from the spinner
func (m *MultiSpinner) RemoveTask(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tasks, id)
	for i, taskID := range m.order {
		if taskID == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
}

// Spinner provides a simple terminal spinner
type Spinner struct {
	frames   []string
	interval time.Duration
	message  string
	writer   io.Writer
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string, w io.Writer) *Spinner {
	return &Spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		message:  message,
		writer:   w,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		i := 0
		// Render first frame immediately
		frame := StyleInfo.Render(s.frames[i%len(s.frames)])
		fmt.Fprintf(s.writer, "\r%s %s", frame, s.message)
		i++

		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Fprintf(s.writer, "\r\033[K")
				return
			case <-ticker.C:
				frame := StyleInfo.Render(s.frames[i%len(s.frames)])
				fmt.Fprintf(s.writer, "\r%s %s", frame, s.message)
				i++
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", StyleSuccess.Render("✓"), message)
}

// Fail stops the spinner and shows an error message
func (s *Spinner) Fail(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", StyleError.Render("✗"), message)
}

// Info stops the spinner and shows an info message
func (s *Spinner) Info(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", StyleInfo.Render("ℹ"), message)
}

// ProgressBar represents a simple progress bar
type ProgressBar struct {
	total   int
	current int
	width   int
	writer  io.Writer
	message string
	mu      sync.Mutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string, w io.Writer) *ProgressBar {
	return &ProgressBar{
		total:   total,
		current: 0,
		width:   40,
		writer:  w,
		message: message,
	}
}

// Increment increments the progress bar
func (p *ProgressBar) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	p.render()
}

// SetCurrent sets the current progress
func (p *ProgressBar) SetCurrent(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
	p.render()
}

// render draws the progress bar
func (p *ProgressBar) render() {
	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	bar := StyleSuccess.Render(repeatString("█", filled)) + StyleMuted.Render(repeatString("░", empty))
	fmt.Fprintf(p.writer, "\r%s [%s] %d/%d (%.0f%%)", p.message, bar, p.current, p.total, percent*100)
}

// Done finishes the progress bar
func (p *ProgressBar) Done() {
	fmt.Fprintln(p.writer)
}

func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

// WriteLogProgress writes a plain-text progress line to w (e.g. log file).
// No ANSI codes; for use when TUI (spinner) is disabled (e.g. detach-child).
func WriteLogProgress(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format, args...)
	if len(format) > 0 && format[len(format)-1] != '\n' {
		fmt.Fprint(w, "\n")
	}
}
