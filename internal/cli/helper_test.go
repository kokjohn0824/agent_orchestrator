package cli

import (
	"testing"

	"github.com/anthropic/agent-orchestrator/internal/config"
)

func TestCreateAgentCaller(t *testing.T) {
	// Store original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	tests := []struct {
		name      string
		setupCfg  func() *config.Config
		wantErr   bool
		wantNil   bool
		checkFunc func(t *testing.T, err error)
	}{
		{
			name: "with dry run mode enabled - agent not available is ok",
			setupCfg: func() *config.Config {
				return &config.Config{
					AgentCommand:      "nonexistent-agent-command-12345",
					AgentForce:        true,
					AgentOutputFormat: "text",
					LogsDir:           "/tmp/test-logs",
					DryRun:            true,
					Verbose:           false,
				}
			},
			wantErr: false,
			wantNil: false,
		},
		{
			name: "without dry run mode - agent not available returns error",
			setupCfg: func() *config.Config {
				return &config.Config{
					AgentCommand:      "nonexistent-agent-command-12345",
					AgentForce:        true,
					AgentOutputFormat: "text",
					LogsDir:           "/tmp/test-logs",
					DryRun:            false,
					Verbose:           false,
				}
			},
			wantErr: true,
			wantNil: true,
			checkFunc: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error when agent is not available and dry run is disabled")
				}
			},
		},
		{
			name: "verbose mode is set correctly",
			setupCfg: func() *config.Config {
				return &config.Config{
					AgentCommand:      "nonexistent-agent-command-12345",
					AgentForce:        false,
					AgentOutputFormat: "json",
					LogsDir:           "/tmp/test-logs",
					DryRun:            true,
					Verbose:           true,
				}
			},
			wantErr: false,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = tt.setupCfg()

			caller, err := CreateAgentCaller()

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAgentCaller() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (caller == nil) != tt.wantNil {
				t.Errorf("CreateAgentCaller() caller = %v, wantNil %v", caller, tt.wantNil)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, err)
			}

			// Verify caller properties when not nil
			if caller != nil {
				if caller.Command != cfg.AgentCommand {
					t.Errorf("caller.Command = %v, want %v", caller.Command, cfg.AgentCommand)
				}
				if caller.Force != cfg.AgentForce {
					t.Errorf("caller.Force = %v, want %v", caller.Force, cfg.AgentForce)
				}
				if caller.OutputFormat != cfg.AgentOutputFormat {
					t.Errorf("caller.OutputFormat = %v, want %v", caller.OutputFormat, cfg.AgentOutputFormat)
				}
				if caller.LogDir != cfg.LogsDir {
					t.Errorf("caller.LogDir = %v, want %v", caller.LogDir, cfg.LogsDir)
				}
				if caller.DryRun != cfg.DryRun {
					t.Errorf("caller.DryRun = %v, want %v", caller.DryRun, cfg.DryRun)
				}
				if caller.Verbose != cfg.Verbose {
					t.Errorf("caller.Verbose = %v, want %v", caller.Verbose, cfg.Verbose)
				}
			}
		})
	}
}

func TestCreateAgentCaller_ConfigProperties(t *testing.T) {
	// Store original config and restore after test
	originalCfg := cfg
	defer func() { cfg = originalCfg }()

	cfg = &config.Config{
		AgentCommand:      "test-agent",
		AgentForce:        true,
		AgentOutputFormat: "stream-json",
		LogsDir:           "/custom/logs/dir",
		DryRun:            true,
		Verbose:           true,
	}

	caller, err := CreateAgentCaller()
	if err != nil {
		t.Fatalf("CreateAgentCaller() unexpected error: %v", err)
	}

	if caller == nil {
		t.Fatal("CreateAgentCaller() returned nil caller")
	}

	// Verify all properties are correctly transferred
	if caller.Command != "test-agent" {
		t.Errorf("Command = %q, want %q", caller.Command, "test-agent")
	}
	if caller.Force != true {
		t.Errorf("Force = %v, want %v", caller.Force, true)
	}
	if caller.OutputFormat != "stream-json" {
		t.Errorf("OutputFormat = %q, want %q", caller.OutputFormat, "stream-json")
	}
	if caller.LogDir != "/custom/logs/dir" {
		t.Errorf("LogDir = %q, want %q", caller.LogDir, "/custom/logs/dir")
	}
	if caller.DryRun != true {
		t.Errorf("DryRun = %v, want %v", caller.DryRun, true)
	}
	if caller.Verbose != true {
		t.Errorf("Verbose = %v, want %v", caller.Verbose, true)
	}
}
