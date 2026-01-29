package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestRecoverableError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		underlying := fmt.Errorf("underlying error")
		err := NewRecoverable("test_op", "test message", underlying)

		if err.Severity() != SeverityRecoverable {
			t.Errorf("expected SeverityRecoverable, got %v", err.Severity())
		}

		expected := "test_op: test message: underlying error"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}

		if err.Unwrap() != underlying {
			t.Error("Unwrap should return the underlying error")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := NewRecoverable("test_op", "test message", nil)

		expected := "test_op: test message"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}

		if err.Unwrap() != nil {
			t.Error("Unwrap should return nil when no underlying error")
		}
	})
}

func TestFatalError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		underlying := fmt.Errorf("underlying error")
		err := NewFatal("test_op", "test message", underlying)

		if err.Severity() != SeverityFatal {
			t.Errorf("expected SeverityFatal, got %v", err.Severity())
		}

		expected := "test_op: test message: underlying error"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}

		if err.Unwrap() != underlying {
			t.Error("Unwrap should return the underlying error")
		}
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := NewFatal("test_op", "test message", nil)

		expected := "test_op: test message"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}

		if err.Unwrap() != nil {
			t.Error("Unwrap should return nil when no underlying error")
		}
	})
}

func TestIsRecoverable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "recoverable error",
			err:      NewRecoverable("op", "msg", nil),
			expected: true,
		},
		{
			name:     "fatal error",
			err:      NewFatal("op", "msg", nil),
			expected: false,
		},
		{
			name:     "wrapped recoverable error",
			err:      fmt.Errorf("wrapped: %w", NewRecoverable("op", "msg", nil)),
			expected: true,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoverable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRecoverable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsFatal(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "fatal error",
			err:      NewFatal("op", "msg", nil),
			expected: true,
		},
		{
			name:     "recoverable error",
			err:      NewRecoverable("op", "msg", nil),
			expected: false,
		},
		{
			name:     "wrapped fatal error",
			err:      fmt.Errorf("wrapped: %w", NewFatal("op", "msg", nil)),
			expected: true,
		},
		{
			name:     "standard error (treated as fatal)",
			err:      fmt.Errorf("standard error"),
			expected: true,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFatal(tt.err)
			if result != tt.expected {
				t.Errorf("IsFatal(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCommonErrorConstructors(t *testing.T) {
	t.Run("ErrAgentNotAvailable", func(t *testing.T) {
		err := ErrAgentNotAvailable()
		if !IsFatal(err) {
			t.Error("ErrAgentNotAvailable should be fatal")
		}
		if err.Op != "agent" {
			t.Errorf("expected Op to be 'agent', got %q", err.Op)
		}
	})

	t.Run("ErrFileNotFound", func(t *testing.T) {
		err := ErrFileNotFound("/path/to/file")
		if !IsFatal(err) {
			t.Error("ErrFileNotFound should be fatal")
		}
		if err.Op != "file" {
			t.Errorf("expected Op to be 'file', got %q", err.Op)
		}
	})

	t.Run("ErrSaveTicket", func(t *testing.T) {
		underlying := fmt.Errorf("disk full")
		err := ErrSaveTicket("TICKET-001", underlying)
		if !IsRecoverable(err) {
			t.Error("ErrSaveTicket should be recoverable")
		}
		if err.Op != "store" {
			t.Errorf("expected Op to be 'store', got %q", err.Op)
		}
		if !errors.Is(err, underlying) {
			t.Error("should wrap the underlying error")
		}
	})

	t.Run("ErrAnalysis", func(t *testing.T) {
		underlying := fmt.Errorf("analysis error")
		err := ErrAnalysis(underlying)
		if !IsRecoverable(err) {
			t.Error("ErrAnalysis should be recoverable")
		}
	})

	t.Run("ErrTest", func(t *testing.T) {
		underlying := fmt.Errorf("test failed")
		err := ErrTest(underlying)
		if !IsRecoverable(err) {
			t.Error("ErrTest should be recoverable")
		}
	})

	t.Run("ErrReview", func(t *testing.T) {
		underlying := fmt.Errorf("review error")
		err := ErrReview(underlying)
		if !IsRecoverable(err) {
			t.Error("ErrReview should be recoverable")
		}
	})

	t.Run("ErrPlanning", func(t *testing.T) {
		underlying := fmt.Errorf("planning error")
		err := ErrPlanning(underlying)
		if !IsFatal(err) {
			t.Error("ErrPlanning should be fatal")
		}
	})

	t.Run("ErrStoreInit", func(t *testing.T) {
		underlying := fmt.Errorf("init error")
		err := ErrStoreInit(underlying)
		if !IsFatal(err) {
			t.Error("ErrStoreInit should be fatal")
		}
	})
}

func TestErrorWrapping(t *testing.T) {
	t.Run("errors.Is works with wrapped errors", func(t *testing.T) {
		underlying := fmt.Errorf("base error")
		recErr := NewRecoverable("op", "msg", underlying)

		if !errors.Is(recErr, underlying) {
			t.Error("errors.Is should find underlying error")
		}
	})

	t.Run("errors.As works with custom types", func(t *testing.T) {
		err := NewRecoverable("op", "msg", nil)
		wrapped := fmt.Errorf("wrapped: %w", err)

		var recErr *RecoverableError
		if !errors.As(wrapped, &recErr) {
			t.Error("errors.As should find RecoverableError")
		}
	})
}
