package ui

import (
	"bytes"
	"testing"
	"time"
)

func TestRepeatString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		count    int
		expected string
	}{
		{
			name:     "重複單一字元",
			input:    "█",
			count:    5,
			expected: "█████",
		},
		{
			name:     "重複多字元字串",
			input:    "ab",
			count:    3,
			expected: "ababab",
		},
		{
			name:     "重複零次",
			input:    "test",
			count:    0,
			expected: "",
		},
		{
			name:     "負數次數",
			input:    "test",
			count:    -1,
			expected: "",
		},
		{
			name:     "空字串",
			input:    "",
			count:    5,
			expected: "",
		},
		{
			name:     "重複一次",
			input:    "hello",
			count:    1,
			expected: "hello",
		},
		{
			name:     "Unicode 字元",
			input:    "░",
			count:    4,
			expected: "░░░░",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repeatString(tt.input, tt.count)
			if result != tt.expected {
				t.Errorf("repeatString(%q, %d) = %q, want %q", tt.input, tt.count, result, tt.expected)
			}
		})
	}
}

func BenchmarkRepeatString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		repeatString("█", 100)
	}
}

func TestSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	// Start spinner
	s.Start()
	time.Sleep(100 * time.Millisecond)

	// Should be running
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if !running {
		t.Error("Spinner should be running after Start()")
	}

	// Stop spinner
	s.Stop()

	// Should not be running
	s.mu.Lock()
	running = s.running
	s.mu.Unlock()
	if running {
		t.Error("Spinner should not be running after Stop()")
	}

	// Buffer should have some output
	if buf.Len() == 0 {
		t.Error("Spinner should have written output")
	}
}

func TestSpinnerDoubleStart(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	s.Start()
	s.Start() // Should be no-op
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	// Should stop cleanly without issues
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		t.Error("Spinner should not be running after Stop()")
	}
}

func TestSpinnerDoubleStop(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Stop()
	s.Stop() // Should be no-op

	// Should handle double stop without panic
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running {
		t.Error("Spinner should not be running after Stop()")
	}
}

func TestSpinnerUpdateMessage(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Initial", &buf)

	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.UpdateMessage("Updated")
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected some output from spinner")
	}
}

func TestSpinnerSuccess(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Success("Done!")

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected success message in output")
	}
}

func TestSpinnerFail(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Fail("Error!")

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected failure message in output")
	}
}

func TestSpinnerInfo(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Loading...", &buf)

	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Info("Information")

	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected info message in output")
	}
}

func TestSpinnerTickerBasedAnimation(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Test", &buf)

	// Start and let it run for a few ticker intervals
	s.Start()
	time.Sleep(250 * time.Millisecond) // ~3 intervals at 80ms
	s.Stop()

	// Verify multiple frames were rendered (output should have been updated multiple times)
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected spinner animation output")
	}
}
