package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/spf13/cobra"
)

func TestRunDrop_TicketNotFound_ReturnsError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "drop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		TicketsDir: ticketsDir,
	}

	cmd := &cobra.Command{}
	err = runDrop(cmd, []string{"nonexistent-ticket-id"})
	if err == nil {
		t.Error("runDrop with nonexistent ticket ID should return non-nil error for non-zero exit code")
	}
}
