// Package config provides configuration management
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	// Agent settings
	AgentCommand      string `mapstructure:"agent_command"`
	AgentOutputFormat string `mapstructure:"agent_output_format"`
	AgentForce        bool   `mapstructure:"agent_force"`
	AgentTimeout      int    `mapstructure:"agent_timeout"` // seconds

	// Paths
	ProjectRoot string `mapstructure:"project_root"`
	TicketsDir  string `mapstructure:"tickets_dir"`
	LogsDir     string `mapstructure:"logs_dir"`
	DocsDir     string `mapstructure:"docs_dir"`

	// Execution settings
	MaxParallel int  `mapstructure:"max_parallel"`
	DryRun      bool `mapstructure:"dry_run"`
	Verbose     bool `mapstructure:"verbose"`
	Debug       bool `mapstructure:"debug"`
	Quiet       bool `mapstructure:"quiet"`

	// Analyze settings
	AnalyzeScopes []string `mapstructure:"analyze_scopes"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	return &Config{
		AgentCommand:      "agent",
		AgentOutputFormat: "text",
		AgentForce:        true,
		AgentTimeout:      600,
		ProjectRoot:       cwd,
		TicketsDir:        ".tickets",
		LogsDir:           ".agent-logs",
		DocsDir:           "docs",
		MaxParallel:       3,
		DryRun:            false,
		Verbose:           false,
		Debug:             false,
		Quiet:             false,
		AnalyzeScopes:     []string{"all"},
	}
}

// Load loads configuration from files and environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigName(".agent-orchestrator")
	v.SetConfigType("yaml")

	// Search paths
	v.AddConfigPath(".")                         // Current directory
	v.AddConfigPath("$HOME")                     // Home directory
	v.AddConfigPath("$HOME/.config/agent-orchestrator") // XDG config

	// Environment variables
	v.SetEnvPrefix("AGENT_ORCHESTRATOR")
	v.AutomaticEnv()

	// Bind specific env vars for backward compatibility
	v.BindEnv("agent_command", "AGENT_CMD")
	v.BindEnv("agent_output_format", "AGENT_OUTPUT_FORMAT")
	v.BindEnv("agent_force", "AGENT_FORCE")

	// Set defaults
	v.SetDefault("agent_command", cfg.AgentCommand)
	v.SetDefault("agent_output_format", cfg.AgentOutputFormat)
	v.SetDefault("agent_force", cfg.AgentForce)
	v.SetDefault("agent_timeout", cfg.AgentTimeout)
	v.SetDefault("tickets_dir", cfg.TicketsDir)
	v.SetDefault("logs_dir", cfg.LogsDir)
	v.SetDefault("docs_dir", cfg.DocsDir)
	v.SetDefault("max_parallel", cfg.MaxParallel)
	v.SetDefault("analyze_scopes", cfg.AnalyzeScopes)

	// Try to read config file (don't fail if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal to struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Resolve relative paths
	cfg.resolvePaths()

	return cfg, nil
}

// resolvePaths converts relative paths to absolute paths
func (c *Config) resolvePaths() {
	if c.ProjectRoot == "" {
		c.ProjectRoot, _ = os.Getwd()
	}

	if !filepath.IsAbs(c.TicketsDir) {
		c.TicketsDir = filepath.Join(c.ProjectRoot, c.TicketsDir)
	}

	if !filepath.IsAbs(c.LogsDir) {
		c.LogsDir = filepath.Join(c.ProjectRoot, c.LogsDir)
	}

	if !filepath.IsAbs(c.DocsDir) {
		c.DocsDir = filepath.Join(c.ProjectRoot, c.DocsDir)
	}
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	v := viper.New()

	v.Set("agent_command", c.AgentCommand)
	v.Set("agent_output_format", c.AgentOutputFormat)
	v.Set("agent_force", c.AgentForce)
	v.Set("agent_timeout", c.AgentTimeout)
	v.Set("tickets_dir", c.TicketsDir)
	v.Set("logs_dir", c.LogsDir)
	v.Set("docs_dir", c.DocsDir)
	v.Set("max_parallel", c.MaxParallel)
	v.Set("analyze_scopes", c.AnalyzeScopes)

	return v.WriteConfigAs(path)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.AgentCommand == "" {
		return fmt.Errorf("agent_command is required")
	}

	if c.MaxParallel < 1 {
		return fmt.Errorf("max_parallel must be at least 1")
	}

	if c.AgentTimeout < 1 {
		return fmt.Errorf("agent_timeout must be at least 1 second")
	}

	validFormats := map[string]bool{
		"text":        true,
		"json":        true,
		"stream-json": true,
	}
	if !validFormats[c.AgentOutputFormat] {
		return fmt.Errorf("invalid agent_output_format: %s", c.AgentOutputFormat)
	}

	return nil
}

// EnsureDirs creates necessary directories
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.TicketsDir,
		c.LogsDir,
		c.DocsDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetConfigFilePath returns the path to the config file
func GetConfigFilePath() string {
	// Check current directory first
	if _, err := os.Stat(".agent-orchestrator.yaml"); err == nil {
		return ".agent-orchestrator.yaml"
	}

	// Then home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ".agent-orchestrator.yaml"
	}

	configPath := filepath.Join(home, ".agent-orchestrator.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// XDG config
	xdgConfig := filepath.Join(home, ".config", "agent-orchestrator", "config.yaml")
	if _, err := os.Stat(xdgConfig); err == nil {
		return xdgConfig
	}

	// Default to current directory
	return ".agent-orchestrator.yaml"
}

// GenerateDefaultConfigFile creates a default config file
func GenerateDefaultConfigFile(path string) error {
	cfg := DefaultConfig()

	content := `# Agent Orchestrator Configuration
# 詳細說明請參考: https://github.com/anthropic/agent-orchestrator

# Agent 設定
agent_command: agent           # Cursor Agent CLI 指令
agent_output_format: text      # 輸出格式: text, json, stream-json
agent_force: true              # 是否使用 --force 允許修改檔案
agent_timeout: 600             # Agent 執行超時秒數

# 路徑設定 (相對於專案根目錄)
tickets_dir: .tickets          # Tickets 儲存目錄
logs_dir: .agent-logs          # Agent 執行日誌目錄
docs_dir: docs                 # 文件目錄

# 執行設定
max_parallel: 3                # 最大並行 Agent 數量

# 分析範圍 (用於 analyze 指令)
analyze_scopes:
  - all                        # 可選: performance, refactor, security, test, docs, all
`

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	_ = cfg // silence unused warning

	return nil
}
