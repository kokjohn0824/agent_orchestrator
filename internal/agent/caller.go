// Package agent provides Cursor Agent integration
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/ui"
)

// Result represents the result of an agent call
type Result struct {
	Success      bool
	Output       string
	Error        string
	Duration     time.Duration
	ExitCode     int
	StreamEvents []StreamEvent
	LogPath      string // Path to log file when detailed logging is enabled
}

// StreamEvent represents a streaming event from the agent
type StreamEvent struct {
	Type    string                 `json:"type"`
	Subtype string                 `json:"subtype,omitempty"`
	Data    map[string]interface{} `json:"-"`
	Raw     string                 `json:"-"`
}

// CallOption configures an agent call
type CallOption func(*callOptions)

type callOptions struct {
	contextFiles []string
	workingDir   string
	timeout      time.Duration
	onStream     func(StreamEvent)
}

// WithContextFiles adds context files to the agent call
func WithContextFiles(files ...string) CallOption {
	return func(o *callOptions) {
		o.contextFiles = append(o.contextFiles, files...)
	}
}

// WithWorkingDir sets the working directory for the agent call
func WithWorkingDir(dir string) CallOption {
	return func(o *callOptions) {
		o.workingDir = dir
	}
}

// WithTimeout sets the timeout for the agent call
func WithTimeout(d time.Duration) CallOption {
	return func(o *callOptions) {
		o.timeout = d
	}
}

// WithStreamHandler sets a callback for stream events
func WithStreamHandler(fn func(StreamEvent)) CallOption {
	return func(o *callOptions) {
		o.onStream = fn
	}
}

// Caller handles Cursor Agent CLI invocations
type Caller struct {
	Command            string
	Force              bool
	OutputFormat       string
	DryRun             bool
	LogDir             string
	Verbose            bool
	DisableDetailedLog bool // When true, disables logging of prompts and outputs
	writer             io.Writer
}

// NewCaller creates a new agent caller
func NewCaller(command string, force bool, outputFormat string, logDir string) *Caller {
	return &Caller{
		Command:      command,
		Force:        force,
		OutputFormat: outputFormat,
		LogDir:       logDir,
		writer:       os.Stdout,
	}
}

// SetWriter sets the output writer
func (c *Caller) SetWriter(w io.Writer) {
	c.writer = w
}

// SetDryRun enables/disables dry run mode
func (c *Caller) SetDryRun(dryRun bool) {
	c.DryRun = dryRun
}

// SetVerbose enables/disables verbose output
func (c *Caller) SetVerbose(verbose bool) {
	c.Verbose = verbose
}

// IsAvailable checks if the agent command is available
func (c *Caller) IsAvailable() bool {
	_, err := exec.LookPath(c.Command)
	return err == nil
}

// Call invokes the Cursor Agent with the given prompt
func (c *Caller) Call(ctx context.Context, prompt string, opts ...CallOption) (*Result, error) {
	options := &callOptions{
		timeout: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(options)
	}

	startTime := time.Now()

	// Create log file
	logFile := c.createLogFile()

	if c.DryRun {
		c.logDryRun(prompt, options)
		return &Result{
			Success:  true,
			Output:   "[DRY RUN] Agent call skipped",
			Duration: time.Since(startTime),
		}, nil
	}

	// Build command arguments
	args := c.buildArgs(prompt, options)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, c.Command, args...)
	if options.workingDir != "" {
		cmd.Dir = options.workingDir
	}

	// Log the command
	c.logCommand(logFile, prompt, args, options)

	// Execute based on output format
	var result *Result
	var err error

	if c.OutputFormat == "stream-json" {
		result, err = c.executeStream(ctx, cmd, logFile, options.onStream)
	} else {
		result, err = c.executeNormal(ctx, cmd, logFile)
	}

	if result != nil {
		result.Duration = time.Since(startTime)
		if logFile != nil {
			result.LogPath = logFile.Name()
		}
	}

	// Log result
	c.logResult(logFile, result, err)

	return result, err
}

// buildArgs constructs the command line arguments
func (c *Caller) buildArgs(prompt string, opts *callOptions) []string {
	args := []string{"-p"}

	if c.Force {
		args = append(args, "--force")
	}

	args = append(args, "--output-format", c.OutputFormat)

	// Build full prompt with context files
	fullPrompt := prompt
	if len(opts.contextFiles) > 0 {
		fullPrompt = fmt.Sprintf("%s\n\n相關檔案: %s", prompt, strings.Join(opts.contextFiles, " "))
	}

	args = append(args, fullPrompt)

	return args
}

// executeNormal executes the command and captures output
func (c *Caller) executeNormal(ctx context.Context, cmd *exec.Cmd, logFile *os.File) (*Result, error) {
	output, err := cmd.CombinedOutput()

	result := &Result{
		Output:   string(output),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		result.Success = false
	} else {
		result.Success = true
	}

	if logFile != nil {
		// Sanitize output before writing to log
		logFile.WriteString(sanitizeSensitiveData(string(output)))
	}

	return result, nil
}

// executeStream executes the command with streaming output
func (c *Caller) executeStream(ctx context.Context, cmd *exec.Cmd, logFile *os.File, onStream func(StreamEvent)) (*Result, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	result := &Result{
		StreamEvents: make([]StreamEvent, 0),
	}

	var outputBuilder strings.Builder

	// Process stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		outputBuilder.WriteString(line + "\n")

		if logFile != nil {
			// Sanitize each line before writing to log
			logFile.WriteString(sanitizeSensitiveData(line) + "\n")
		}

		// Try to parse as JSON event
		event := c.parseStreamEvent(line)
		if event != nil {
			result.StreamEvents = append(result.StreamEvents, *event)
			if onStream != nil {
				onStream(*event)
			}
			c.handleStreamEvent(*event)
		}
	}

	// Read stderr
	stderrBytes, _ := io.ReadAll(stderr)
	if len(stderrBytes) > 0 {
		result.Error = string(stderrBytes)
		outputBuilder.WriteString(string(stderrBytes))
	}

	err = cmd.Wait()
	result.Output = outputBuilder.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Success = false
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result, nil
}

// parseStreamEvent parses a JSON stream event
func (c *Caller) parseStreamEvent(line string) *StreamEvent {
	if !strings.HasPrefix(line, "{") {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		return nil
	}

	event := &StreamEvent{
		Data: data,
		Raw:  line,
	}

	if t, ok := data["type"].(string); ok {
		event.Type = t
	}
	if st, ok := data["subtype"].(string); ok {
		event.Subtype = st
	}

	return event
}

// handleStreamEvent processes a stream event and outputs to terminal
func (c *Caller) handleStreamEvent(event StreamEvent) {
	switch event.Type {
	case "system":
		if event.Subtype == "init" {
			if model, ok := event.Data["model"].(string); ok && c.Verbose {
				ui.PrintInfo(c.writer, fmt.Sprintf("使用模型: %s", model))
			}
		}
	case "tool_call":
		if event.Subtype == "started" {
			if toolCall, ok := event.Data["tool_call"].(map[string]interface{}); ok {
				c.logToolCall(toolCall)
			}
		}
	case "result":
		if duration, ok := event.Data["duration_ms"].(float64); ok && c.Verbose {
			ui.PrintSuccess(c.writer, fmt.Sprintf("完成，耗時 %.0fms", duration))
		}
	}
}

// logToolCall logs a tool call event
func (c *Caller) logToolCall(toolCall map[string]interface{}) {
	if !c.Verbose {
		return
	}

	if writeCall, ok := toolCall["writeToolCall"].(map[string]interface{}); ok {
		if args, ok := writeCall["args"].(map[string]interface{}); ok {
			if path, ok := args["path"].(string); ok {
				ui.PrintInfo(c.writer, fmt.Sprintf("寫入檔案: %s", path))
			}
		}
	} else if readCall, ok := toolCall["readToolCall"].(map[string]interface{}); ok {
		if args, ok := readCall["args"].(map[string]interface{}); ok {
			if path, ok := args["path"].(string); ok {
				ui.PrintInfo(c.writer, fmt.Sprintf("讀取檔案: %s", path))
			}
		}
	}
}

// createLogFile creates a log file for the agent call
// Security: Uses 0700 for directory and 0600 for file to protect sensitive data
func (c *Caller) createLogFile() *os.File {
	if c.LogDir == "" {
		return nil
	}

	// Disable detailed logging if configured
	if c.DisableDetailedLog {
		return nil
	}

	// Use 0700 for log directory - only owner can access
	if err := os.MkdirAll(c.LogDir, 0700); err != nil {
		return nil
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("agent-%s.log", timestamp)
	path := filepath.Join(c.LogDir, filename)

	// Use 0600 for log file - only owner can read/write
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil
	}

	return file
}

// sensitivePatterns contains regex patterns for sensitive information
var sensitivePatterns = []string{
	// API keys and tokens
	`(?i)(api[_-]?key|apikey|api_secret|secret[_-]?key)\s*[:=]\s*['"]?[a-zA-Z0-9_\-]{16,}['"]?`,
	`(?i)(access[_-]?token|auth[_-]?token|bearer)\s*[:=]\s*['"]?[a-zA-Z0-9_\-\.]{20,}['"]?`,
	// AWS credentials
	`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret)\s*[:=]\s*['"]?[A-Z0-9]{16,}['"]?`,
	`AKIA[0-9A-Z]{16}`,
	// Password patterns
	`(?i)(password|passwd|pwd)\s*[:=]\s*['"]?[^\s'"]{4,}['"]?`,
	// Private keys
	`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`,
	// GitHub tokens
	`gh[pousr]_[A-Za-z0-9_]{36,}`,
	// Generic secrets
	`(?i)(client[_-]?secret|secret)\s*[:=]\s*['"]?[a-zA-Z0-9_\-]{16,}['"]?`,
}

// compiledPatterns holds compiled regex patterns (initialized lazily)
var compiledPatterns []*regexp.Regexp
var patternsOnce sync.Once

// getCompiledPatterns returns compiled regex patterns, initializing them once
func getCompiledPatterns() []*regexp.Regexp {
	patternsOnce.Do(func() {
		compiledPatterns = make([]*regexp.Regexp, 0, len(sensitivePatterns))
		for _, pattern := range sensitivePatterns {
			if re, err := regexp.Compile(pattern); err == nil {
				compiledPatterns = append(compiledPatterns, re)
			}
		}
	})
	return compiledPatterns
}

// sanitizeSensitiveData removes or masks sensitive information from text
func sanitizeSensitiveData(text string) string {
	result := text
	for _, re := range getCompiledPatterns() {
		result = re.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// logCommand logs the command being executed
func (c *Caller) logCommand(file *os.File, prompt string, args []string, opts *callOptions) {
	if file == nil {
		return
	}

	file.WriteString("=== Agent Call ===\n")
	file.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
	file.WriteString(fmt.Sprintf("Command: %s %s\n", c.Command, strings.Join(args[:min(len(args), 5)], " ")))
	file.WriteString(fmt.Sprintf("Prompt length: %d\n", len(prompt)))
	file.WriteString(fmt.Sprintf("Context files: %v\n", opts.contextFiles))
	file.WriteString(fmt.Sprintf("Working dir: %s\n", opts.workingDir))
	file.WriteString("=== Output ===\n")
}

// logResult logs the result of the agent call
func (c *Caller) logResult(file *os.File, result *Result, err error) {
	if file == nil {
		return
	}
	defer file.Close()

	file.WriteString("\n=== End ===\n")
	if result != nil {
		file.WriteString(fmt.Sprintf("Success: %v\n", result.Success))
		file.WriteString(fmt.Sprintf("Exit code: %d\n", result.ExitCode))
		file.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
	}
	if err != nil {
		file.WriteString(fmt.Sprintf("Error: %v\n", err))
	}
}

// logDryRun logs a dry run
func (c *Caller) logDryRun(prompt string, opts *callOptions) {
	ui.PrintWarning(c.writer, "[DRY RUN] 跳過實際 agent 呼叫")
	if c.Verbose {
		ui.PrintInfo(c.writer, fmt.Sprintf("Prompt: %s", ui.Truncate(prompt, 200)))
		if len(opts.contextFiles) > 0 {
			ui.PrintInfo(c.writer, fmt.Sprintf("Context: %v", opts.contextFiles))
		}
	}
}

// CallForJSON calls the agent and expects JSON output to a file
func (c *Caller) CallForJSON(ctx context.Context, prompt string, outputFile string, opts ...CallOption) (*Result, map[string]interface{}, error) {
	// Add instruction to write JSON to file
	fullPrompt := fmt.Sprintf("%s\n\n請將結果以 JSON 格式寫入檔案: %s", prompt, outputFile)

	result, err := c.Call(ctx, fullPrompt, opts...)
	if err != nil {
		return result, nil, err
	}

	if !result.Success {
		return result, nil, fmt.Errorf("agent call failed: %s", result.Error)
	}

	// Read the output file
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return result, nil, fmt.Errorf("failed to read output file: %w", err)
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return result, nil, fmt.Errorf("failed to parse JSON output: %w", err)
	}

	return result, jsonData, nil
}

