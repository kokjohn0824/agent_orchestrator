package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestRunStatus_BackgroundWork_WhenPidFileExistsAndAlive_ShowsRunningAndLogPath 驗證當 PID 檔存在且
// 該 process 存活時，status 會顯示「背景工作: 執行中 (PID N)」及日誌路徑。
func TestRunStatus_BackgroundWork_WhenPidFileExistsAndAlive_ShowsRunningAndLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	// 至少一個 ticket 才會進入狀態表與 background 區塊
	if err := store.Save(&ticket.Ticket{ID: "TICKET-001", Title: "Test", Status: ticket.StatusPending}); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	pidPath := filepath.Join(tmpDir, ".work.pid")
	if err := WriteWorkPIDFile(pidPath); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}
	// WriteWorkPIDFile 寫入當前 process PID，所以 IsProcessAlive 為 true
	pid := os.Getpid()

	logDir := filepath.Join(tmpDir, "logs")
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		TicketsDir:       ticketsDir,
		WorkPIDFile:      pidPath,
		WorkDetachLogDir: logDir,
		LogsDir:          filepath.Join(tmpDir, ".agent-logs"),
	}

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

	wantPidLine := fmt.Sprintf(i18n.MsgBackgroundWorkRunningPid, pid)
	if !strings.Contains(output, wantPidLine) {
		t.Errorf("output should contain %q, got:\n%s", wantPidLine, output)
	}
	wantLogLine := fmt.Sprintf(i18n.MsgLogPath, logDir)
	if !strings.Contains(output, logDir) {
		t.Errorf("output should contain log path %q, got:\n%s", wantLogLine, output)
	}
}

// TestRunStatus_BackgroundWork_WhenNoPidFile_DoesNotShowBackgroundLine 驗證當沒有 PID 檔或
// process 已死時，status 不顯示背景工作那一行。
func TestRunStatus_BackgroundWork_WhenNoPidFile_DoesNotShowBackgroundLine(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	if err := store.Save(&ticket.Ticket{ID: "TICKET-001", Title: "Test", Status: ticket.StatusPending}); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	// 不建立 PID 檔；cfg 指向不存在的路徑
	pidPath := filepath.Join(tmpDir, ".work.pid")
	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		TicketsDir:  ticketsDir,
		WorkPIDFile: pidPath,
	}

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

	if strings.Contains(output, i18n.MsgBackgroundWorkRunning) {
		t.Errorf("output should not contain background work line when no pid file, got:\n%s", output)
	}
}

// TestRunStatus_StalePidFile_RemovesFileAndDoesNotShowRunning 驗證當 PID 檔存在但 process 已死時，
// status 會刪除過期 PID 檔，且不顯示為 running（顯示無背景 work）。
func TestRunStatus_StalePidFile_RemovesFileAndDoesNotShowRunning(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	if err := store.Save(&ticket.Ticket{ID: "TICKET-001", Title: "Test", Status: ticket.StatusPending}); err != nil {
		t.Fatalf("Failed to save ticket: %v", err)
	}

	pidPath := filepath.Join(tmpDir, ".work.pid")
	// 寫入一個不存在的 PID（process 已死），通常 999999 在測試環境不會存在
	if err := os.WriteFile(pidPath, []byte("999999\n"), 0600); err != nil {
		t.Fatalf("Write pid file: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		TicketsDir:  ticketsDir,
		WorkPIDFile: pidPath,
	}

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

	// 過期 PID 檔應被刪除
	if _, err := os.Stat(pidPath); err == nil {
		t.Errorf("stale PID file should be removed, but %s still exists", pidPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("Stat pid file: %v", err)
	}

	// 不應顯示為 running
	if strings.Contains(output, "執行中") {
		t.Errorf("output should not show background work as running when pid is dead, got:\n%s", output)
	}
}

// TestRunStatus_QueryOnly_CoexistsWithBackgroundWork 驗證 status 為僅讀指令，不檢查 ErrIfBackgroundWorkRunning，
// 當背景 work 執行中（PID 檔存在且 process 存活）時仍可成功執行、不誤擋（TICKET-019）。
func TestRunStatus_QueryOnly_CoexistsWithBackgroundWork(t *testing.T) {
	tmpDir := t.TempDir()
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}
	if err := store.Save(&ticket.Ticket{ID: "TICKET-001", Title: "Test", Status: ticket.StatusPending}); err != nil {
		t.Fatalf("store.Save(): %v", err)
	}

	pidPath := filepath.Join(tmpDir, ".work.pid")
	if err := WriteWorkPIDFile(pidPath); err != nil {
		t.Fatalf("WriteWorkPIDFile: %v", err)
	}

	originalCfg := cfg
	defer func() { cfg = originalCfg }()
	cfg = &config.Config{
		TicketsDir:  ticketsDir,
		WorkPIDFile: pidPath,
	}

	// status 不應因背景 work 執行中而回傳錯誤（僅寫入 store 的指令才受並行策略限制）
	err := runStatus(nil, nil)
	if err != nil {
		t.Errorf("runStatus() with background work running should succeed (query-only commands coexist); got err: %v", err)
	}
}

// writeIntegrationConfig writes a minimal .agent-orchestrator.yaml in dir for integration tests.
func writeIntegrationConfig(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, ".agent-orchestrator.yaml")
	content := []byte(`dry_run: true
max_parallel: 1
tickets_dir: .tickets
logs_dir: .agent-logs
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// TestIntegration_Status_WhenDetachRunning_ShowsRunningPid 整合測試：執行 work --detach 後
// 立即執行 status，驗證輸出包含「背景工作: 執行中 (PID N)」。
func TestIntegration_Status_WhenDetachRunning_ShowsRunningPid(t *testing.T) {
	tmpDir := t.TempDir()
	writeIntegrationConfig(t, tmpDir)
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

	// 子 process 可能很快結束（dry run、空 store），先盡快跑 status
	statusCmd := exec.Command(binaryPath, "status")
	statusCmd.Dir = tmpDir
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}
	output := string(statusOut)
	wantLine := fmt.Sprintf(i18n.MsgBackgroundWorkRunningPid, pid)
	if !strings.Contains(output, wantLine) {
		// 若子 process 已結束，PID file 已被刪除，則不會顯示 running；重試或視為 skip
		if !IsProcessAlive(pid) {
			t.Logf("child already exited, skipping running check")
			return
		}
		t.Errorf("status output should contain %q when detach is running, got:\n%s", wantLine, output)
	}

	// 等待子 process 結束，避免留下背景 process
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && IsProcessAlive(pid) {
		time.Sleep(50 * time.Millisecond)
	}
}

// TestIntegration_Status_WhenDetachFinished_NoRunningAndPidFileGone 整合測試：執行 work --detach 後
// 等待子 process 結束，再執行 status，驗證不顯示 running 且 PID 檔已刪除。
func TestIntegration_Status_WhenDetachFinished_NoRunningAndPidFileGone(t *testing.T) {
	tmpDir := t.TempDir()
	writeIntegrationConfig(t, tmpDir)
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

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if !IsProcessAlive(pid) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if IsProcessAlive(pid) {
		t.Fatalf("child process %d did not exit within timeout", pid)
	}

	pidPath := filepath.Join(tmpDir, ".tickets", ".work.pid")
	if _, err := os.Stat(pidPath); err == nil {
		t.Errorf("PID file should be removed after child exits, but %s still exists", pidPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("Stat pid file: %v", err)
	}

	statusCmd := exec.Command(binaryPath, "status")
	statusCmd.Dir = tmpDir
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}
	if strings.Contains(string(statusOut), "執行中") {
		t.Errorf("status should not show background work as running after child exits, got:\n%s", statusOut)
	}
}

// TestIntegration_Status_StalePidFile_RemovedAndNotShown 整合測試：寫入過期 PID 檔後執行 status，
// 驗證 status 會刪除過期 PID 檔且不顯示為 running。
func TestIntegration_Status_StalePidFile_RemovedAndNotShown(t *testing.T) {
	tmpDir := t.TempDir()
	writeIntegrationConfig(t, tmpDir)
	ticketsDir := filepath.Join(tmpDir, ".tickets")
	if err := os.MkdirAll(ticketsDir, 0700); err != nil {
		t.Fatalf("mkdir tickets: %v", err)
	}
	store := ticket.NewStore(ticketsDir)
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init(): %v", err)
	}
	if err := store.Save(&ticket.Ticket{ID: "TICKET-001", Title: "Test", Status: ticket.StatusPending}); err != nil {
		t.Fatalf("store.Save: %v", err)
	}

	pidPath := filepath.Join(ticketsDir, ".work.pid")
	if err := os.WriteFile(pidPath, []byte("999999\n"), 0600); err != nil {
		t.Fatalf("write stale pid file: %v", err)
	}

	binaryPath := buildIntegrationBinary(t, tmpDir)
	statusCmd := exec.Command(binaryPath, "status")
	statusCmd.Dir = tmpDir
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}

	if _, err := os.Stat(pidPath); err == nil {
		t.Errorf("stale PID file should be removed by status, but %s still exists", pidPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("Stat pid file: %v", err)
	}
	if strings.Contains(string(statusOut), "執行中") {
		t.Errorf("status should not show background work as running for stale PID, got:\n%s", statusOut)
	}
}
