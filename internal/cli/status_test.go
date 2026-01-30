package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

// TestRunStatus_NoTickets_ShowsAddHint 驗證當沒有任何 tickets 時，
// status 命令會顯示可透過 agent-orchestrator add 新增 ticket 的提示。
func TestRunStatus_NoTickets_ShowsAddHint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "status-test-*")
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

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	err = runStatus(nil, nil)
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// 應包含「直接新增」與 add 指令提示
	if !strings.Contains(output, "agent-orchestrator add") {
		t.Errorf("output should contain 'agent-orchestrator add', got:\n%s", output)
	}
	if !strings.Contains(output, i18n.MsgGettingStartedAdd) {
		t.Errorf("output should contain MsgGettingStartedAdd (%q), got:\n%s", i18n.MsgGettingStartedAdd, output)
	}
}
