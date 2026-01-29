package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

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
		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Fprintf(s.writer, "\r\033[K")
				return
			default:
				frame := StyleInfo.Render(s.frames[i%len(s.frames)])
				fmt.Fprintf(s.writer, "\r%s %s", frame, s.message)
				i++
				time.Sleep(s.interval)
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
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
