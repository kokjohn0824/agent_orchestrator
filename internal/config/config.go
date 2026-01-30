// Package config provides configuration management
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
// 預設值以 DefaultConfig() 為準；設定檔與環境變數會覆寫對應欄位。
type Config struct {
	// Agent settings

	// AgentCommand 是呼叫 Cursor Agent 的 CLI 指令名稱或路徑。預設 "agent"。
	// 何時調整：Cursor CLI 安裝在非 PATH 或使用自訂執行檔時，改為完整路徑或別名。
	AgentCommand string `mapstructure:"agent_command"`

	// AgentOutputFormat 為 agent 輸出格式：text、json、stream-json。預設 "text"。
	// 何時調整：需要程式化解析輸出時用 "json" 或 "stream-json"；一般使用 "text" 即可。
	AgentOutputFormat string `mapstructure:"agent_output_format"`

	// AgentForce 為是否在呼叫 agent 時加上 --force，允許寫入/修改檔案。預設 true。
	// 何時調整：僅想預覽不寫入時設為 false（多數情境建議保持 true 以正常執行）。
	AgentForce bool `mapstructure:"agent_force"`

	// AgentTimeout 為單次 agent 呼叫的超時秒數。預設 600（10 分鐘）。
	// 何時調整：任務較大或環境較慢時可提高；想提早中止卡住任務時可降低。
	AgentTimeout int `mapstructure:"agent_timeout"`

	// Paths（皆可為相對路徑，會依 ProjectRoot 解析為絕對路徑）

	// ProjectRoot 為專案根目錄，未設時為當前工作目錄。
	ProjectRoot string `mapstructure:"project_root"`

	// TicketsDir 為 tickets 儲存目錄。預設 ".tickets"。
	TicketsDir string `mapstructure:"tickets_dir"`

	// LogsDir 為 agent 執行日誌目錄；日誌可能含 prompt/輸出內容。預設 ".agent-logs"。
	LogsDir string `mapstructure:"logs_dir"`

	// DocsDir 為文件（如 milestone）輸出目錄。預設 "docs"。
	DocsDir string `mapstructure:"docs_dir"`

	// Execution settings

	// MaxParallel 為 work 指令同時執行的 agent 數量上限。預設 3。
	// 何時調整：機器資源足夠且想加快處理時可提高；資源有限或避免過載時可降低。
	MaxParallel int `mapstructure:"max_parallel"`

	// DryRun 為是否僅模擬不實際呼叫 agent。
	DryRun bool `mapstructure:"dry_run"`

	// Verbose 為是否輸出詳細資訊。
	Verbose bool `mapstructure:"verbose"`

	// Debug 為是否開啟除錯輸出。
	Debug bool `mapstructure:"debug"`

	// Quiet 為是否減少一般輸出。
	Quiet bool `mapstructure:"quiet"`

	// Security settings

	// DisableDetailedLog 為是否停用「詳細日誌」：停用後不會在 LogsDir 寫入含 prompt 與 agent 輸出的日誌檔。
	// 預設 false（會寫入詳細日誌）。副作用：設為 true 時無法從日誌還原對話內容，有利於避免敏感資訊落檔。
	// 何時調整：在含機密或專屬程式碼的環境、或需符合資安/合規要求時，建議設為 true。
	DisableDetailedLog bool `mapstructure:"disable_detailed_log"`

	// Analyze settings

	// AnalyzeScopes 為 analyze 指令的預設分析範圍。預設 ["all"] 表示所有面向。
	// 可選值：performance、refactor、security、test、docs、all。指令列 --scope 會覆寫此預設。
	// 何時調整：若經常只分析部分面向（例如僅 performance,security），可在此設定以省去每次下 --scope。
	AnalyzeScopes []string `mapstructure:"analyze_scopes"`
}

// DefaultConfig 回傳預設設定，為本套件中「預設值」的單一來源；
// Load 會先以此為基底，再以設定檔與環境變數覆寫。
func DefaultConfig() *Config {
	cwd, _ := os.Getwd()
	return &Config{
		AgentCommand:       "agent",
		AgentOutputFormat:  "text",
		AgentForce:         true,
		AgentTimeout:       600,
		ProjectRoot:        cwd,
		TicketsDir:         ".tickets",
		LogsDir:            ".agent-logs",
		DocsDir:            "docs",
		MaxParallel:        3,
		DryRun:             false,
		Verbose:            false,
		Debug:              false,
		Quiet:              false,
		DisableDetailedLog: false,
		AnalyzeScopes:      []string{"all"},
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
	v.SetDefault("disable_detailed_log", cfg.DisableDetailedLog)
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
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	v := viper.New()

	v.Set("agent_command", c.AgentCommand)
	v.Set("agent_output_format", c.AgentOutputFormat)
	v.Set("agent_force", c.AgentForce)
	v.Set("agent_timeout", c.AgentTimeout)
	v.Set("tickets_dir", c.TicketsDir)
	v.Set("logs_dir", c.LogsDir)
	v.Set("docs_dir", c.DocsDir)
	v.Set("max_parallel", c.MaxParallel)
	v.Set("disable_detailed_log", c.DisableDetailedLog)
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
	// Sensitive directories that should be restricted (owner only)
	sensitiveDirs := []string{
		c.TicketsDir,
		c.LogsDir,
	}

	// Non-sensitive directories that can be world-readable
	publicDirs := []string{
		c.DocsDir,
	}

	// Use 0700 for sensitive directories (tickets and logs may contain sensitive data)
	for _, dir := range sensitiveDirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Use 0755 for public directories
	for _, dir := range publicDirs {
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
	content := `# Agent Orchestrator Configuration
# 各欄位說明、預設值與建議情境請見 README「設定說明」章節

# Agent 設定
agent_command: agent           # Cursor Agent CLI 指令 (預設: agent)
agent_output_format: text      # 輸出格式: text, json, stream-json (預設: text)
agent_force: true              # 是否使用 --force 允許修改檔案 (預設: true)
agent_timeout: 600             # Agent 執行超時秒數 (預設: 600)

# 路徑設定 (相對於專案根目錄)
tickets_dir: .tickets          # Tickets 儲存目錄 (預設: .tickets)
logs_dir: .agent-logs          # Agent 執行日誌目錄 (預設: .agent-logs)
docs_dir: docs                 # 文件目錄 (預設: docs)

# 執行設定
max_parallel: 3                # 最大並行 Agent 數量 (預設: 3)

# 安全設定
disable_detailed_log: false    # 設為 true 停用詳細日誌，避免敏感資訊落檔 (預設: false)

# 分析範圍 (用於 analyze 指令，--scope 會覆寫)
analyze_scopes:
  - all                        # 可選: performance, refactor, security, test, docs, all (預設: all)
`

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		// Use 0700 for config directory to protect potential sensitive settings
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
