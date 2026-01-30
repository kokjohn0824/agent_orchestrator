package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
)

func TestRunWork_StoreInitFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "work-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsPath := filepath.Join(tmpDir, ".tickets")
	if err := os.WriteFile(ticketsPath, []byte("x"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        ticketsPath,
		AgentCommand:      "agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	err = runWork(nil, nil)
	if err == nil {
		t.Error("runWork expected error when store init fails")
	}
	if err != nil && !strings.Contains(err.Error(), "store") && !strings.Contains(err.Error(), "初始化") {
		t.Errorf("error should mention store/init, got: %v", err)
	}
}

func TestRunWork_NoArgs_EmptyStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "work-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:       ticketsDir,
		AgentCommand:     "agent",
		AgentForce:       true,
		AgentOutputFormat: "text",
		DryRun:           true,
		MaxParallel:      3,
	}

	err = runWork(nil, nil)
	if err != nil {
		t.Fatalf("runWork with no args and empty store should succeed: %v", err)
	}
}

func TestWorkSingleTicket_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "work-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:       ticketsDir,
		AgentCommand:     "agent",
		AgentForce:       true,
		AgentOutputFormat: "text",
		DryRun:           true,
		MaxParallel:      3,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err = workSingleTicket(context.Background(), store, "NONEXISTENT-001")
	w.Close()
	if err != nil {
		t.Fatalf("workSingleTicket with nonexistent ID should return nil (prints error): %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "NONEXISTENT") && !strings.Contains(out, "找不到") {
		t.Errorf("output should mention ticket not found, got: %s", out)
	}
}

func TestWorkSingleTicket_NotPending(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "work-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	tkt := ticket.NewTicket("DONE-001", "Done ticket", "Already completed")
	tkt.MarkCompleted("done")
	if err := store.Save(tkt); err != nil {
		t.Fatalf("store.Save(): %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:       ticketsDir,
		AgentCommand:     "agent",
		AgentForce:       true,
		AgentOutputFormat: "text",
		DryRun:           true,
		MaxParallel:      3,
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err = workSingleTicket(context.Background(), store, "DONE-001")
	w.Close()
	if err != nil {
		t.Fatalf("workSingleTicket with non-pending ticket should return nil: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "DONE-001") && !strings.Contains(out, "completed") && !strings.Contains(out, "無法") {
		t.Errorf("output should mention ticket cannot be processed, got: %s", out)
	}
}

func TestWorkCmd_Flags(t *testing.T) {
	if workCmd.Flags().Lookup("parallel") == nil {
		t.Error("work command should have --parallel flag")
	}
}
