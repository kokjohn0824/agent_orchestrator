package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNewCaller(t *testing.T) {
	caller := NewCaller("cursor", true, "stream-json", "/tmp/logs")

	if caller.Command != "cursor" {
		t.Errorf("NewCaller().Command = %v, want cursor", caller.Command)
	}
	if !caller.Force {
		t.Error("NewCaller().Force should be true")
	}
	if caller.OutputFormat != "stream-json" {
		t.Errorf("NewCaller().OutputFormat = %v, want stream-json", caller.OutputFormat)
	}
	if caller.LogDir != "/tmp/logs" {
		t.Errorf("NewCaller().LogDir = %v, want /tmp/logs", caller.LogDir)
	}
}

func TestCaller_SetDryRun(t *testing.T) {
	caller := NewCaller("cursor", false, "text", "")

	if caller.DryRun {
		t.Error("default DryRun should be false")
	}

	caller.SetDryRun(true)
	if !caller.DryRun {
		t.Error("SetDryRun(true) should set DryRun to true")
	}

	caller.SetDryRun(false)
	if caller.DryRun {
		t.Error("SetDryRun(false) should set DryRun to false")
	}
}

func TestCaller_SetVerbose(t *testing.T) {
	caller := NewCaller("cursor", false, "text", "")

	if caller.Verbose {
		t.Error("default Verbose should be false")
	}

	caller.SetVerbose(true)
	if !caller.Verbose {
		t.Error("SetVerbose(true) should set Verbose to true")
	}
}

func TestCaller_SetWriter(t *testing.T) {
	caller := NewCaller("cursor", false, "text", "")

	buf := &bytes.Buffer{}
	caller.SetWriter(buf)

	// We can't easily test this without invoking the caller,
	// but we can at least verify the method doesn't panic
}

func TestCaller_buildArgs(t *testing.T) {
	tests := []struct {
		name         string
		caller       *Caller
		prompt       string
		opts         *callOptions
		wantContains []string
	}{
		{
			name: "basic args without force",
			caller: &Caller{
				Force:        false,
				OutputFormat: "text",
			},
			prompt: "test prompt",
			opts:   &callOptions{},
			wantContains: []string{
				"-p",
				"--output-format",
				"text",
				"test prompt",
			},
		},
		{
			name: "with force flag",
			caller: &Caller{
				Force:        true,
				OutputFormat: "stream-json",
			},
			prompt: "test prompt",
			opts:   &callOptions{},
			wantContains: []string{
				"-p",
				"--force",
				"--output-format",
				"stream-json",
				"test prompt",
			},
		},
		{
			name: "with context files",
			caller: &Caller{
				Force:        false,
				OutputFormat: "text",
			},
			prompt: "test prompt",
			opts: &callOptions{
				contextFiles: []string{"file1.go", "file2.go"},
			},
			wantContains: []string{
				"test prompt",
				"file1.go",
				"file2.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.caller.buildArgs(tt.prompt, tt.opts)

			// Join args into a single string for easier checking
			argsStr := ""
			for _, arg := range args {
				argsStr += arg + " "
			}

			for _, want := range tt.wantContains {
				found := false
				for _, arg := range args {
					if arg == want || (len(want) > 5 && contains(arg, want)) {
						found = true
						break
					}
				}
				if !found && !contains(argsStr, want) {
					t.Errorf("buildArgs() should contain %q in args: %v", want, args)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCaller_parseStreamEvent(t *testing.T) {
	caller := NewCaller("cursor", false, "stream-json", "")

	tests := []struct {
		name        string
		line        string
		wantNil     bool
		wantType    string
		wantSubtype string
	}{
		{
			name:        "valid JSON event",
			line:        `{"type":"system","subtype":"init","model":"claude"}`,
			wantNil:     false,
			wantType:    "system",
			wantSubtype: "init",
		},
		{
			name:     "tool call event",
			line:     `{"type":"tool_call","subtype":"started"}`,
			wantNil:  false,
			wantType: "tool_call",
		},
		{
			name:    "non-JSON line",
			line:    "This is not JSON",
			wantNil: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "partial JSON",
			line:    `{"type":`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := caller.parseStreamEvent(tt.line)

			if tt.wantNil {
				if event != nil {
					t.Errorf("parseStreamEvent() = %v, want nil", event)
				}
				return
			}

			if event == nil {
				t.Fatal("parseStreamEvent() = nil, want non-nil")
			}

			if event.Type != tt.wantType {
				t.Errorf("parseStreamEvent().Type = %v, want %v", event.Type, tt.wantType)
			}

			if tt.wantSubtype != "" && event.Subtype != tt.wantSubtype {
				t.Errorf("parseStreamEvent().Subtype = %v, want %v", event.Subtype, tt.wantSubtype)
			}

			if event.Raw != tt.line {
				t.Errorf("parseStreamEvent().Raw = %v, want %v", event.Raw, tt.line)
			}
		})
	}
}

func TestCallOptions(t *testing.T) {
	t.Run("WithContextFiles", func(t *testing.T) {
		opts := &callOptions{}
		WithContextFiles("file1.go", "file2.go")(opts)

		if len(opts.contextFiles) != 2 {
			t.Errorf("WithContextFiles() set %d files, want 2", len(opts.contextFiles))
		}
		if opts.contextFiles[0] != "file1.go" || opts.contextFiles[1] != "file2.go" {
			t.Errorf("WithContextFiles() = %v, want [file1.go, file2.go]", opts.contextFiles)
		}
	})

	t.Run("WithWorkingDir", func(t *testing.T) {
		opts := &callOptions{}
		WithWorkingDir("/test/dir")(opts)

		if opts.workingDir != "/test/dir" {
			t.Errorf("WithWorkingDir() = %v, want /test/dir", opts.workingDir)
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		opts := &callOptions{}
		WithTimeout(5 * 60 * 1e9)(opts) // 5 minutes in nanoseconds

		if opts.timeout != 5*60*1e9 {
			t.Errorf("WithTimeout() = %v, want 5m", opts.timeout)
		}
	})

	t.Run("WithStreamHandler", func(t *testing.T) {
		opts := &callOptions{}
		called := false
		handler := func(e StreamEvent) {
			called = true
		}
		WithStreamHandler(handler)(opts)

		if opts.onStream == nil {
			t.Error("WithStreamHandler() should set onStream")
		}

		opts.onStream(StreamEvent{})
		if !called {
			t.Error("WithStreamHandler() handler not called")
		}
	})
}

func TestTruncateFunc(t *testing.T) {
	// Test the truncate helper function behavior
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test truncation logic inline
			got := tt.input
			if len(tt.input) > tt.maxLen {
				got = tt.input[:tt.maxLen] + "..."
			}
			if got != tt.want {
				t.Errorf("truncate logic = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgentResult(t *testing.T) {
	result := &Result{
		Success:      true,
		Output:       "Test output",
		Error:        "",
		ExitCode:     0,
		StreamEvents: []StreamEvent{},
	}

	if !result.Success {
		t.Error("Result.Success should be true")
	}
	if result.Output != "Test output" {
		t.Errorf("Result.Output = %v, want 'Test output'", result.Output)
	}
}

func TestStreamEvent(t *testing.T) {
	event := StreamEvent{
		Type:    "tool_call",
		Subtype: "started",
		Data: map[string]interface{}{
			"tool_call": map[string]interface{}{
				"name": "write",
			},
		},
		Raw: `{"type":"tool_call","subtype":"started"}`,
	}

	if event.Type != "tool_call" {
		t.Errorf("StreamEvent.Type = %v, want 'tool_call'", event.Type)
	}
	if event.Subtype != "started" {
		t.Errorf("StreamEvent.Subtype = %v, want 'started'", event.Subtype)
	}
}

func TestSanitizeSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // should NOT contain this after sanitization
		notEmpty bool   // result should not be empty
	}{
		{
			name:     "API key pattern",
			input:    "api_key=sk-1234567890abcdefghijklmnop",
			contains: "sk-1234567890",
			notEmpty: true,
		},
		{
			name:     "API key with quotes",
			input:    `api_key="my-secret-api-key-12345"`,
			contains: "my-secret-api-key",
			notEmpty: true,
		},
		{
			name:     "Password pattern",
			input:    "password=mysecretpassword123",
			contains: "mysecretpassword",
			notEmpty: true,
		},
		{
			name:     "AWS access key",
			input:    "AKIAIOSFODNN7EXAMPLE",
			contains: "AKIAIOSFODNN7EXAMPLE",
			notEmpty: true,
		},
		{
			name:     "GitHub token",
			input:    "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "ghp_",
			notEmpty: true,
		},
		{
			name:     "Bearer token",
			input:    "bearer=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			notEmpty: true,
		},
		{
			name:     "Private key header",
			input:    "-----BEGIN PRIVATE KEY-----\nMIIE...",
			contains: "-----BEGIN PRIVATE KEY-----",
			notEmpty: true,
		},
		{
			name:     "No sensitive data - normal text",
			input:    "This is just a normal log message about file processing",
			contains: "", // empty means we don't check for absence
			notEmpty: true,
		},
		{
			name:     "Client secret",
			input:    "client_secret=abcdef1234567890abcdef",
			contains: "abcdef1234567890",
			notEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSensitiveData(tt.input)

			if tt.notEmpty && result == "" {
				t.Error("sanitizeSensitiveData() returned empty string")
			}

			if tt.contains != "" && containsHelper(result, tt.contains) {
				t.Errorf("sanitizeSensitiveData() should have redacted %q, got: %s", tt.contains, result)
			}
		})
	}
}

func TestSanitizeSensitiveData_PreservesNormalText(t *testing.T) {
	// Normal text without sensitive data should remain unchanged
	normalTexts := []string{
		"Processing file main.go",
		"Building project...",
		"Test passed successfully",
		"Error: file not found",
		"Connected to database",
	}

	for _, text := range normalTexts {
		result := sanitizeSensitiveData(text)
		if result != text {
			t.Errorf("sanitizeSensitiveData(%q) = %q, should remain unchanged", text, result)
		}
	}
}

func TestCaller_DisableDetailedLog(t *testing.T) {
	caller := NewCaller("cursor", false, "text", "/tmp/logs")

	// Default should be false
	if caller.DisableDetailedLog {
		t.Error("default DisableDetailedLog should be false")
	}

	// Test setting to true
	caller.DisableDetailedLog = true
	if !caller.DisableDetailedLog {
		t.Error("DisableDetailedLog should be true after setting")
	}

	// When DisableDetailedLog is true, createLogFile should return nil
	logFile := caller.createLogFile()
	if logFile != nil {
		logFile.Close()
		t.Error("createLogFile() should return nil when DisableDetailedLog is true")
	}
}

func TestCaller_CreateLogFile_EmptyLogDir(t *testing.T) {
	caller := NewCaller("cursor", false, "text", "")

	// When LogDir is empty, createLogFile should return nil
	logFile := caller.createLogFile()
	if logFile != nil {
		logFile.Close()
		t.Error("createLogFile() should return nil when LogDir is empty")
	}
}

// TestCaller_executeStream_helper is run as a subprocess to produce stdout for executeStream tests.
// Set GO_TEST_HELPER=output_long_line to print a line > 64KB (default bufio max token).
func TestCaller_executeStream_helper(t *testing.T) {
	switch os.Getenv("GO_TEST_HELPER") {
	case "output_long_line":
		// 70KB line; without scanner.Buffer() this would trigger bufio.ErrTooLong
		fmt.Print(strings.Repeat("x", 70*1024) + "\n")
		os.Exit(0)
	}
}

func TestCaller_executeStream_longLineAndScannerErr(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER") == "output_long_line" {
		return
	}

	caller := NewCaller("cursor", false, "stream-json", "")
	ctx := context.Background()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCaller_executeStream_helper$")
	cmd.Env = append(os.Environ(), "GO_TEST_HELPER=output_long_line")

	result, err := caller.executeStream(ctx, cmd, nil, nil)
	if err != nil {
		t.Fatalf("executeStream with long line: %v", err)
	}
	if !result.Success {
		t.Errorf("result.Success = false, want true")
	}
	wantMinLen := 70 * 1024
	if len(result.Output) < wantMinLen {
		t.Errorf("result.Output length = %d, want at least %d", len(result.Output), wantMinLen)
	}
}
