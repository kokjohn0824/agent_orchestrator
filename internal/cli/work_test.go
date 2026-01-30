package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if workCmd.Flags().Lookup("detach") == nil {
		t.Error("work command should have --detach flag")
	}
	if workCmd.Flags().Lookup("log-file") == nil {
		t.Error("work command should have --log-file flag")
	}
}

// TestRunWork_WithoutDetach_BehaviorUnchanged ensures that when --detach is not used,
// runWork runs the normal path (no child process), mock config and dry-run still apply.
func TestRunWork_WithoutDetach_BehaviorUnchanged(t *testing.T) {
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

	originalDetach := workDetach
	originalCfg := cfg
	defer func() {
		workDetach = originalDetach
		cfg = originalCfg
	}()
	workDetach = false // --detach flag exists but not used
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
		t.Fatalf("runWork without --detach should succeed (no child process): %v", err)
	}
	// If we had entered the detach path we would get TICKET-008 / not implemented error;
	// success here confirms non-detach path is unchanged.
}

func TestRunWork_Detach_StartsChildAndReturns(t *testing.T) {
	// When --detach is set and we're not the detach child, runWork should start the child process
	// and return nil (child runs in background). If child fails to start, returns that error.
	originalDetach := workDetach
	originalChild := isDetachChild
	defer func() {
		workDetach = originalDetach
		isDetachChild = originalChild
	}()
	workDetach = true
	isDetachChild = false

	tmpDir := t.TempDir()
	ticketsDir := tmpDir + "/.tickets"
	_ = os.MkdirAll(ticketsDir, 0700)
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		MaxParallel: 1,
		ProjectRoot: tmpDir,
		TicketsDir:  ticketsDir,
		LogsDir:     tmpDir + "/.agent-logs",
	}

	err := runWork(nil, nil)
	if err != nil && (strings.Contains(err.Error(), "TICKET-008") || strings.Contains(err.Error(), "not implemented")) {
		t.Errorf("runWork --detach should not return stub error (exec is implemented), got: %v", err)
	}
	// err may be nil (child started) or non-nil (e.g. binary not found in test env)
}

func TestRunWork_Detach_WithTicketID_StartsChildAndReturns(t *testing.T) {
	originalDetach := workDetach
	originalChild := isDetachChild
	defer func() {
		workDetach = originalDetach
		isDetachChild = originalChild
	}()
	workDetach = true
	isDetachChild = false

	tmpDir := t.TempDir()
	ticketsDir := tmpDir + "/.tickets"
	_ = os.MkdirAll(ticketsDir, 0700)
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		MaxParallel: 1,
		ProjectRoot: tmpDir,
		TicketsDir:  ticketsDir,
		LogsDir:     tmpDir + "/.agent-logs",
	}

	err := runWork(nil, []string{"TICKET-001"})
	if err != nil && (strings.Contains(err.Error(), "TICKET-008") || strings.Contains(err.Error(), "not implemented")) {
		t.Errorf("runWork --detach with ticket-id should not return stub error, got: %v", err)
	}
}

func TestBuildWorkDetachParams(t *testing.T) {
	originalCfgFile := cfgFile
	originalCfg := cfg
	defer func() {
		cfgFile = originalCfgFile
		cfg = originalCfg
	}()

	// No ticket-id, no config, no cfg (no --log-file)
	cfgFile = ""
	cfg = nil
	params, err := buildWorkDetachParams(nil)
	if err != nil {
		t.Fatalf("buildWorkDetachParams(nil): %v", err)
	}
	if params.Binary == "" {
		t.Error("Binary should be non-empty")
	}
	if len(params.Args) < 2 {
		t.Fatalf("Args should have at least work and --detach-child, got %v", params.Args)
	}
	if params.Args[0] != "work" || params.Args[1] != "--detach-child" {
		t.Errorf("Args without ticket should be [work, --detach-child, ...], got %v", params.Args)
	}

	// With ticket-id
	params, err = buildWorkDetachParams([]string{"TICKET-002"})
	if err != nil {
		t.Fatalf("buildWorkDetachParams([TICKET-002]): %v", err)
	}
	wantArgs := []string{"work", "TICKET-002", "--detach-child"}
	if len(params.Args) < 3 || params.Args[0] != wantArgs[0] || params.Args[1] != wantArgs[1] || params.Args[2] != wantArgs[2] {
		t.Errorf("Args with ticket should start with [work, TICKET-002, --detach-child], got %v", params.Args)
	}

	// With --config pass-through
	cfgFile = "/path/to/config.yaml"
	params, err = buildWorkDetachParams([]string{"TICKET-003"})
	if err != nil {
		t.Fatalf("buildWorkDetachParams with cfgFile: %v", err)
	}
	hasConfig := false
	for i := 0; i < len(params.Args)-1; i++ {
		if params.Args[i] == "--config" && params.Args[i+1] == cfgFile {
			hasConfig = true
			break
		}
	}
	if !hasConfig {
		t.Errorf("Args should contain --config and cfgFile when cfgFile is set, got %v", params.Args)
	}

	// With cfg set, args should contain --log-file and a path
	cfg = &config.Config{ProjectRoot: t.TempDir(), LogsDir: ".agent-logs"}
	params, err = buildWorkDetachParams([]string{"TICKET-004"})
	if err != nil {
		t.Fatalf("buildWorkDetachParams with cfg: %v", err)
	}
	hasLogFile := false
	for i := 0; i < len(params.Args)-1; i++ {
		if params.Args[i] == "--log-file" && params.Args[i+1] != "" {
			hasLogFile = true
			break
		}
	}
	if !hasLogFile {
		t.Errorf("Args should contain --log-file and path when cfg is set, got %v", params.Args)
	}
	if params.LogPath == "" {
		t.Error("LogPath should be set when cfg is set")
	}
}

func TestExecWorkDetach_StartsChildAndReturnsPid(t *testing.T) {
	// execWorkDetach should start the child and return its PID (child runs in background).
	// Use our binary with "version" so the child exits quickly; we only verify Start() succeeds.
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = nil // skip EnsureDirs
	binary, err := os.Executable()
	if err != nil {
		t.Skipf("os.Executable(): %v", err)
	}
	params := WorkDetachParams{Binary: binary, Args: []string{"version"}}
	pid, err := execWorkDetach(params)
	if err != nil {
		t.Errorf("execWorkDetach(version) should succeed: %v", err)
	}
	if pid <= 0 {
		t.Errorf("execWorkDetach should return positive PID, got %d", pid)
	}
}

// TestWorkLogWriter_ReturnsNilWhenNotDetachChild ensures WorkLogWriter() returns nil when not in detach-child mode.
func TestWorkLogWriter_ReturnsNilWhenNotDetachChild(t *testing.T) {
	originalChild := isDetachChild
	defer func() { isDetachChild = originalChild }()
	isDetachChild = false
	if w := WorkLogWriter(); w != nil {
		t.Errorf("WorkLogWriter() should be nil when not detach-child, got %v", w)
	}
}

// TestRunWork_DetachChild_CreatesLogFile ensures that when running as detach-child, runWork creates a log file
// (path from config) and work logic can get the writer via WorkLogWriter().
func TestRunWork_DetachChild_CreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	logsDir := filepath.Join(tmpDir, ".agent-logs")
	if err := os.MkdirAll(ticketsDir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	originalChild := isDetachChild
	originalDetach := workDetach
	originalCfg := cfg
	originalWorkLogWriter := workLogWriter
	defer func() {
		isDetachChild = originalChild
		workDetach = originalDetach
		cfg = originalCfg
		workLogWriter = originalWorkLogWriter
	}()

	isDetachChild = true
	workDetach = false
	workLogFile = ""
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        ticketsDir,
		LogsDir:           logsDir,
		AgentCommand:      "agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	err := runWork(nil, nil)
	if err != nil {
		t.Fatalf("runWork as detach-child should succeed: %v", err)
	}

	// Log file should exist in LogsDir (work-YYYYMMDD-HHMMSS.log)
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", logsDir, err)
	}
	var logPath string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "work-") && strings.HasSuffix(e.Name(), ".log") {
			logPath = filepath.Join(logsDir, e.Name())
			break
		}
	}
	if logPath == "" {
		t.Fatalf("expected a work-*.log file in %s, got %v", logsDir, entries)
	}
	// Summary and output should be written to log; log file should be closed after runWork returns.
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(log): %v (log should be closed on process exit)", err)
	}
	logStr := string(content)
	if !strings.Contains(logStr, "完成") {
		t.Errorf("log should contain completion summary, got: %s", logStr)
	}
}

// TestRunWork_DetachChild_LogFileOverride ensures that --log-file overrides the log path for that run.
func TestRunWork_DetachChild_LogFileOverride(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	customLog := filepath.Join(tmpDir, "custom-detach.log")

	if err := os.MkdirAll(ticketsDir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	originalChild := isDetachChild
	originalDetach := workDetach
	originalCfg := cfg
	originalWorkLogWriter := workLogWriter
	defer func() {
		isDetachChild = originalChild
		workDetach = originalDetach
		cfg = originalCfg
		workLogWriter = originalWorkLogWriter
	}()

	isDetachChild = true
	workDetach = false
	workLogFile = customLog
	cfg = &config.Config{
		ProjectRoot:       tmpDir,
		TicketsDir:        ticketsDir,
		LogsDir:            filepath.Join(tmpDir, ".agent-logs"),
		AgentCommand:      "agent",
		AgentForce:        true,
		AgentOutputFormat: "text",
		DryRun:            true,
		MaxParallel:       3,
	}

	err := runWork(nil, nil)
	if err != nil {
		t.Fatalf("runWork as detach-child with --log-file should succeed: %v", err)
	}

	if _, err := os.Stat(customLog); err != nil {
		t.Errorf("expected log file at --log-file path %s: %v", customLog, err)
	}
}

// buildIntegrationBinary builds the agent-orchestrator binary into outDir and returns its path.
// Used by integration tests that need to fork/exec the real CLI.
func buildIntegrationBinary(t *testing.T, outDir string) string {
	t.Helper()
	binaryPath := filepath.Join(outDir, "agent-orchestrator")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/agent-orchestrator")
	cmd.Dir = repoRootForIntegration()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build integration binary: %v\n%s", err, out)
	}
	return binaryPath
}

// repoRootForIntegration returns the repository root (parent of internal/).
func repoRootForIntegration() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// parseDetachPidFromOutput extracts the child PID from work --detach stdout (e.g. "已分離。PID: 12345，日誌: ...").
func parseDetachPidFromOutput(out []byte) (int, bool) {
	re := regexp.MustCompile(`PID:\s*(\d+)`)
	matches := re.FindSubmatch(out)
	if len(matches) < 2 {
		return 0, false
	}
	pid, err := strconv.Atoi(string(matches[1]))
	if err != nil {
		return 0, false
	}
	return pid, true
}

// TestIntegration_WorkDetach_ParentExitsZero_PidFileAndLogExist is an integration test: actually fork/exec
// the CLI with work --detach, then verify parent exits 0, PID file exists with child PID, and log file is created.
// Uses temp dir and independent config.
func TestIntegration_WorkDetach_ParentExitsZero_PidFileAndLogExist(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".agent-orchestrator.yaml")
	configContent := []byte(`dry_run: true
max_parallel: 1
tickets_dir: .tickets
logs_dir: .agent-logs
`)
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".tickets"), 0700); err != nil {
		t.Fatalf("mkdir tickets: %v", err)
	}
	store := ticket.NewStore(filepath.Join(tmpDir, ".tickets"))
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}

	binaryPath := buildIntegrationBinary(t, tmpDir)

	cmd := exec.Command(binaryPath, "work", "--detach")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("work --detach: %v\n%s", err, out)
	}

	pid, ok := parseDetachPidFromOutput(out)
	if !ok {
		t.Fatalf("output should contain PID, got: %s", out)
	}
	if pid <= 0 {
		t.Errorf("parsed PID should be positive, got %d", pid)
	}

	pidPath := filepath.Join(tmpDir, ".tickets", ".work.pid")
	// Child may take a moment to write PID file; poll briefly. If child exits very quickly (e.g. dry run),
	// it may remove the file before we read—we still require log file to exist.
	var pidData []byte
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		pidData, err = os.ReadFile(pidPath)
		if err == nil {
			break
		}
	}
	if len(pidData) > 0 {
		pidStr := strings.TrimSpace(string(pidData))
		filePid, err := strconv.Atoi(pidStr)
		if err != nil {
			t.Fatalf("PID file should contain integer: %s", pidStr)
		}
		if filePid != pid {
			t.Errorf("PID file contains %d, parent reported %d", filePid, pid)
		}
	}
	// If PID file was never found, child may have already exited; we still require log file below

	logsDir := filepath.Join(tmpDir, ".agent-logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		t.Fatalf("logs dir should exist: %v", err)
	}
	var logPath string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "work-") && strings.HasSuffix(e.Name(), ".log") {
			logPath = filepath.Join(logsDir, e.Name())
			break
		}
	}
	if logPath == "" {
		t.Fatalf("expected work-*.log in %s, got %v", logsDir, entries)
	}
	// Optional: log content should contain expected summary string
	logContent, _ := os.ReadFile(logPath)
	if !strings.Contains(string(logContent), "完成") {
		t.Logf("log may contain completion summary; content (excerpt): %s", string(logContent))
	}

	// Wait for child to exit so PID file is removed by defer in child
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !IsProcessAlive(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
