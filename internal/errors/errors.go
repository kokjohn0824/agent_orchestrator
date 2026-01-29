// Package errors provides custom error types for the agent orchestrator.
// It distinguishes between recoverable errors (can be logged and continue)
// and fatal errors (must halt execution and return).
package errors

import (
	"errors"
	"fmt"

	"github.com/anthropic/agent-orchestrator/internal/i18n"
)

// Severity represents the severity level of an error
type Severity int

const (
	// SeverityRecoverable indicates an error that can be logged and execution can continue
	SeverityRecoverable Severity = iota
	// SeverityFatal indicates an error that must halt execution
	SeverityFatal
)

// OrchestratorError is the base interface for all orchestrator errors
type OrchestratorError interface {
	error
	Severity() Severity
	Unwrap() error
}

// RecoverableError represents an error that can be logged and execution can continue
type RecoverableError struct {
	Op      string // Operation that failed
	Message string // Human-readable message
	Err     error  // Underlying error
}

func (e *RecoverableError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *RecoverableError) Severity() Severity {
	return SeverityRecoverable
}

func (e *RecoverableError) Unwrap() error {
	return e.Err
}

// FatalError represents an error that must halt execution
type FatalError struct {
	Op      string // Operation that failed
	Message string // Human-readable message
	Err     error  // Underlying error
}

func (e *FatalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *FatalError) Severity() Severity {
	return SeverityFatal
}

func (e *FatalError) Unwrap() error {
	return e.Err
}

// NewRecoverable creates a new recoverable error
func NewRecoverable(op, message string, err error) *RecoverableError {
	return &RecoverableError{
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// NewFatal creates a new fatal error
func NewFatal(op, message string, err error) *FatalError {
	return &FatalError{
		Op:      op,
		Message: message,
		Err:     err,
	}
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	var recErr *RecoverableError
	if errors.As(err, &recErr) {
		return true
	}

	// Check if it implements OrchestratorError interface
	var orchErr OrchestratorError
	if errors.As(err, &orchErr) {
		return orchErr.Severity() == SeverityRecoverable
	}

	return false
}

// IsFatal checks if an error is fatal
func IsFatal(err error) bool {
	var fatalErr *FatalError
	if errors.As(err, &fatalErr) {
		return true
	}

	// Check if it implements OrchestratorError interface
	var orchErr OrchestratorError
	if errors.As(err, &orchErr) {
		return orchErr.Severity() == SeverityFatal
	}

	// By default, unknown errors are treated as fatal
	return err != nil
}

// Common error constructors for specific scenarios

// ErrAgentNotAvailable creates an error for when the agent is not available
func ErrAgentNotAvailable() *FatalError {
	return NewFatal(i18n.ErrOpAgent, i18n.ErrMsgAgentNotAvailable, nil)
}

// ErrFileNotFound creates an error for when a file is not found
func ErrFileNotFound(path string) *FatalError {
	return NewFatal(i18n.ErrOpFile, fmt.Sprintf(i18n.ErrMsgFileNotFound, path), nil)
}

// ErrSaveTicket creates a recoverable error for ticket save failures
func ErrSaveTicket(ticketID string, err error) *RecoverableError {
	return NewRecoverable(i18n.ErrOpStore, fmt.Sprintf(i18n.ErrMsgSaveTicket, ticketID), err)
}

// ErrAnalysis creates a recoverable error for analysis failures
func ErrAnalysis(err error) *RecoverableError {
	return NewRecoverable(i18n.ErrOpAnalyze, i18n.ErrMsgAnalysisFailed, err)
}

// ErrTest creates a recoverable error for test failures
func ErrTest(err error) *RecoverableError {
	return NewRecoverable(i18n.ErrOpTest, i18n.ErrMsgTestFailed, err)
}

// ErrReview creates a recoverable error for review failures
func ErrReview(err error) *RecoverableError {
	return NewRecoverable(i18n.ErrOpReview, i18n.ErrMsgReviewFailed, err)
}

// ErrPlanning creates a fatal error for planning failures
func ErrPlanning(err error) *FatalError {
	return NewFatal(i18n.ErrOpPlanning, i18n.ErrMsgPlanningFailed, err)
}

// ErrStoreInit creates a fatal error for store initialization failures
func ErrStoreInit(err error) *FatalError {
	return NewFatal(i18n.ErrOpStore, i18n.ErrMsgStoreInit, err)
}
