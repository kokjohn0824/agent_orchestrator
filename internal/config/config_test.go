package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_EnsureDirs_Permissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-perm-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &Config{
		ProjectRoot: tempDir,
		TicketsDir:  filepath.Join(tempDir, ".tickets"),
		LogsDir:     filepath.Join(tempDir, ".agent-logs"),
		DocsDir:     filepath.Join(tempDir, "docs"),
	}

	if err := cfg.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	// Verify sensitive directories have restricted permissions (0700)
	sensitiveDirs := []string{cfg.TicketsDir, cfg.LogsDir}
	for _, dir := range sensitiveDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("failed to stat directory %s: %v", dir, err)
			continue
		}

		perm := info.Mode().Perm()
		// 0700 means owner has rwx, group and others have nothing
		if perm&0077 != 0 {
			t.Errorf("sensitive directory %s has permissions %o, expected 0700 (no group/other access)", dir, perm)
		}
	}

	// Verify public directories have standard permissions (0755)
	publicDirs := []string{cfg.DocsDir}
	for _, dir := range publicDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("failed to stat directory %s: %v", dir, err)
			continue
		}

		perm := info.Mode().Perm()
		// 0755 means owner has rwx, group and others have rx
		if perm != 0755 {
			t.Errorf("public directory %s has permissions %o, expected 0755", dir, perm)
		}
	}
}

func TestGenerateDefaultConfigFile_DirectoryPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-gen-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, "config_subdir")
	configPath := filepath.Join(configDir, "config.yaml")

	if err := GenerateDefaultConfigFile(configPath); err != nil {
		t.Fatalf("GenerateDefaultConfigFile() error = %v", err)
	}

	// Verify the config directory has restricted permissions
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}

	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		t.Errorf("config directory has permissions %o, expected 0700 (no group/other access)", perm)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.TicketsDir != ".tickets" {
		t.Errorf("TicketsDir = %s, want .tickets", cfg.TicketsDir)
	}

	if cfg.LogsDir != ".agent-logs" {
		t.Errorf("LogsDir = %s, want .agent-logs", cfg.LogsDir)
	}

	if cfg.MaxParallel != 3 {
		t.Errorf("MaxParallel = %d, want 3", cfg.MaxParallel)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				AgentCommand:      "agent",
				AgentOutputFormat: "text",
				AgentTimeout:      600,
				MaxParallel:       3,
			},
			wantErr: false,
		},
		{
			name: "missing agent command",
			cfg: &Config{
				AgentCommand:      "",
				AgentOutputFormat: "text",
				AgentTimeout:      600,
				MaxParallel:       3,
			},
			wantErr: true,
		},
		{
			name: "invalid max parallel",
			cfg: &Config{
				AgentCommand:      "agent",
				AgentOutputFormat: "text",
				AgentTimeout:      600,
				MaxParallel:       0,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			cfg: &Config{
				AgentCommand:      "agent",
				AgentOutputFormat: "text",
				AgentTimeout:      0,
				MaxParallel:       3,
			},
			wantErr: true,
		},
		{
			name: "invalid output format",
			cfg: &Config{
				AgentCommand:      "agent",
				AgentOutputFormat: "invalid",
				AgentTimeout:      600,
				MaxParallel:       3,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
