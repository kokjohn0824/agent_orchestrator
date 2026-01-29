package i18n

import (
	"testing"
)

// TestMessagesNotEmpty ensures that all message constants are non-empty.
// This helps catch typos or incomplete message definitions.
func TestMessagesNotEmpty(t *testing.T) {
	messages := map[string]string{
		"MsgSuccess":     MsgSuccess,
		"MsgFailed":      MsgFailed,
		"MsgCompleted":   MsgCompleted,
		"MsgCancelled":   MsgCancelled,
		"CmdRootShort":   CmdRootShort,
		"CmdRootLong":    CmdRootLong,
		"CmdInitShort":   CmdInitShort,
		"CmdAnalyzeShort": CmdAnalyzeShort,
		"CmdPlanShort":   CmdPlanShort,
		"CmdWorkShort":   CmdWorkShort,
		"CmdReviewShort": CmdReviewShort,
		"CmdTestShort":   CmdTestShort,
		"CmdCommitShort": CmdCommitShort,
		"CmdRunShort":    CmdRunShort,
		"CmdStatusShort": CmdStatusShort,
		"CmdRetryShort":  CmdRetryShort,
		"CmdCleanShort":  CmdCleanShort,
		"CmdConfigShort": CmdConfigShort,
	}

	for name, value := range messages {
		if value == "" {
			t.Errorf("Message constant %s is empty", name)
		}
	}
}

// TestErrorMessagesNotEmpty ensures that error message constants are non-empty.
func TestErrorMessagesNotEmpty(t *testing.T) {
	messages := map[string]string{
		"ErrOpAgent":            ErrOpAgent,
		"ErrOpFile":             ErrOpFile,
		"ErrOpStore":            ErrOpStore,
		"ErrMsgAgentNotAvailable": ErrMsgAgentNotAvailable,
		"ErrMsgFileNotFound":    ErrMsgFileNotFound,
		"ErrMsgSaveTicket":      ErrMsgSaveTicket,
		"ErrMsgAnalysisFailed":  ErrMsgAnalysisFailed,
		"ErrMsgTestFailed":      ErrMsgTestFailed,
		"ErrMsgReviewFailed":    ErrMsgReviewFailed,
		"ErrMsgPlanningFailed":  ErrMsgPlanningFailed,
		"ErrMsgStoreInit":       ErrMsgStoreInit,
	}

	for name, value := range messages {
		if value == "" {
			t.Errorf("Error message constant %s is empty", name)
		}
	}
}

// TestUIMessagesConsistency ensures UI messages have consistent format.
func TestUIMessagesConsistency(t *testing.T) {
	// Test that spinner messages follow a consistent pattern
	spinnerMessages := []string{
		SpinnerGeneratingQuestions,
		SpinnerGeneratingMilestone,
		SpinnerAnalyzing,
		SpinnerPlanning,
		SpinnerReviewing,
		SpinnerTesting,
		SpinnerCommitting,
	}

	for _, msg := range spinnerMessages {
		if len(msg) == 0 {
			t.Error("Spinner message should not be empty")
		}
		// Spinner messages should end with "..." in Chinese style
		if msg[len(msg)-3:] != "..." {
			t.Errorf("Spinner message should end with '...': %s", msg)
		}
	}
}
