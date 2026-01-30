package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestRunPlan_ExactArgs(t *testing.T) {
	// plan command requires exactly 1 argument (cobra.ExactArgs(1))
	if planCmd.Args == nil {
		t.Error("planCmd.Args should be set")
	}
	// With wrong number of args, cobra validation returns error
	if err := planCmd.Args(planCmd, nil); err == nil {
		t.Error("plan with no args should fail validation")
	}
}

func TestRunPlanWithFile_MilestoneNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	nonExistent := filepath.Join(tmpDir, "nonexistent-milestone.md")
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        filepath.Join(tmpDir, ".tickets"),
		AgentCommand:      "agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	err = runPlanWithFile(context.Background(), nonExistent)
	if err == nil {
		t.Error("runPlanWithFile with missing file should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), nonExistent) {
		t.Errorf("error should mention file not found, got: %v", err)
	}
}

func TestRunPlanWithFile_AgentNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	milestonePath := filepath.Join(tmpDir, "milestone.md")
	if err := os.WriteFile(milestonePath, []byte("# Milestone\n## Goals\n- Goal 1"), 0644); err != nil {
		t.Fatalf("Failed to create milestone file: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        filepath.Join(tmpDir, ".tickets"),
		AgentCommand:      "nonexistent-agent-plan-xyz",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            false, // so CreateAgentCaller fails
		MaxParallel:       3,
	}

	err = runPlanWithFile(context.Background(), milestonePath)
	if err == nil {
		t.Error("runPlanWithFile when agent not found should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "agent") {
		t.Errorf("error should mention agent, got: %v", err)
	}
}

func TestPlanCmd_Args(t *testing.T) {
	// plan expects ExactArgs(1)
	if planCmd.Args == nil {
		t.Error("planCmd.Args should be set")
	}
}

func TestRunPlan_WithFile(t *testing.T) {
	// runPlan delegates to runPlanWithFile with args[0]
	tmpDir, err := os.MkdirTemp("", "plan-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	missing := filepath.Join(tmpDir, "missing.md")
	cmd := planCmd
	err = runPlan(cmd, []string{missing})
	if err == nil {
		t.Error("runPlan with missing file should return error")
	}
}
