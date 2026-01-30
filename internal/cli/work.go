package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	workParallel  int
	workDetach    bool
	workLogFile   string
	workLogWriter io.Writer // set when running as detach-child; used for log file output
)

var workCmd = &cobra.Command{
	Use:   "work [ticket-id]",
	Short: i18n.CmdWorkShort,
	Long:  i18n.CmdWorkLong,
	RunE:  runWork,
}

func init() {
	workCmd.Flags().IntVarP(&workParallel, "parallel", "p", 0, i18n.FlagParallel)
	workCmd.Flags().BoolVar(&workDetach, "detach", false, i18n.FlagDetach)
	workCmd.Flags().StringVar(&workLogFile, "log-file", "", i18n.FlagLogFile)
}

// WorkDetachParams holds the prepared argv for exec of work in detach (child) mode.
// Used when work --detach (or work [ticket-id] --detach) prepares to exec itself;
// the actual exec is implemented in TICKET-008.
type WorkDetachParams struct {
	Binary  string   // path to agent-orchestrator binary
	Args    []string // e.g. ["work", "TICKET-001", "--detach-child", "--config", "/path"]
	LogPath string   // log file path for child (empty if unknown)
}

// buildWorkDetachParams builds the binary path and args for the detach child process.
// Pass through --config and --log-file so the child loads the same config and writes logs to the given path.
func buildWorkDetachParams(args []string) (WorkDetachParams, error) {
	binary, err := os.Executable()
	if err != nil {
		return WorkDetachParams{}, fmt.Errorf("work --detach: %w", err)
	}
	childArgs := []string{"work"}
	if len(args) > 0 {
		childArgs = append(childArgs, args[0])
	}
	childArgs = append(childArgs, detachChildFlagName)
	if cfgFile != "" {
		childArgs = append(childArgs, "--config", cfgFile)
	}
	var logPath string
	if cfg != nil {
		logPath = cfg.DetachLogPath(workLogFile, time.Now())
		childArgs = append(childArgs, "--log-file", logPath)
	}
	return WorkDetachParams{Binary: binary, Args: childArgs, LogPath: logPath}, nil
}

// WorkLogWriter returns the io.Writer for the work log when running as detach-child (log file).
// Returns nil when not in detach-child mode or when the log file was not set.
func WorkLogWriter() io.Writer {
	if !IsDetachChild() {
		return nil
	}
	return workLogWriter
}

// execWorkDetach starts the child process for detach work.
// The child is detached from the terminal using setsid (Unix) or DETACHED_PROCESS (Windows)
// so that closing the terminal does not kill it.
// Returns the child PID on success; caller should print it and exit 0 without waiting.
func execWorkDetach(params WorkDetachParams) (pid int, err error) {
	if cfg != nil {
		if err := cfg.EnsureDirs(); err != nil {
			return 0, fmt.Errorf("work --detach: ensure dirs: %w", err)
		}
	}
	cmd := exec.Command(params.Binary, params.Args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	setDetachSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("work --detach: start child: %w", err)
	}
	// Do not Wait: parent returns so the user gets the prompt back; child runs in background.
	return cmd.Process.Pid, nil
}

func runWork(cmd *cobra.Command, args []string) error {
	// Refuse to run (or spawn another detach) if background work is already running (TICKET-018).
	if !IsDetachChild() {
		if err := ErrIfBackgroundWorkRunning(); err != nil {
			return err
		}
	}

	// --detach (parent): prepare child argv and exec; print PID and log path then exit 0 (TICKET-009).
	if workDetach && !IsDetachChild() {
		params, err := buildWorkDetachParams(args)
		if err != nil {
			return err
		}
		pid, err := execWorkDetach(params)
		if err != nil {
			return err
		}
		if params.LogPath != "" {
			ui.PrintSuccess(os.Stdout, fmt.Sprintf(i18n.MsgDetachedPidLog, pid, params.LogPath))
		} else {
			ui.PrintSuccess(os.Stdout, fmt.Sprintf(i18n.MsgDetachedPid, pid))
		}
		return nil
	}

	// detach-child: create log file (path from config + --log-file override), redirect stdout/stderr to it.
	// All errors and summary go to the log writer; close log file on process exit (defer or normal path).
	var pidPath string
	if IsDetachChild() {
		// Resolve log path with fallback so we can open log before any check that might fail.
		var logPath string
		if cfg != nil {
			logPath = cfg.DetachLogPath(workLogFile, time.Now())
		} else if workLogFile != "" {
			logPath = workLogFile
		} else {
			logPath = ".tickets/detach.log"
		}
		if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
			return fmt.Errorf("work detach-child: create log dir: %w", err)
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("work detach-child: open log file: %w", err)
		}
		workLogWriter = f
		os.Stdout = f
		os.Stderr = f
		defer f.Close() // close log file on any return path
		if cfg == nil {
			ui.PrintError(f, "work detach-child: config is required")
			f.Close()
			os.Exit(1)
		}
		// Write PID file before entering work logic (TICKET-013).
		pidPath = cfg.WorkPIDFilePath()
		if err := WriteWorkPIDFile(pidPath); err != nil {
			ui.PrintError(f, fmt.Sprintf("work detach-child: %v", err))
			f.Close()
			os.Exit(1)
		}
		// Remove PID file on exit (normal or after signal); TICKET-014.
		defer RemoveWorkPIDFile(pidPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		// Remove PID file on SIGTERM/SIGINT so we don't leave a stale file (TICKET-014).
		RemoveWorkPIDFile(pidPath)
		ui.PrintWarning(os.Stdout, i18n.MsgInterruptSignal)
		cancel()
	}()

	// Determine parallel count
	parallel := cfg.MaxParallel
	if workParallel > 0 {
		parallel = workParallel
	}

	// Initialize store
	store := ticket.NewStore(cfg.TicketsDir)
	if err := store.Init(); err != nil {
		return fmt.Errorf(i18n.ErrInitStoreFailed, err)
	}

	// If specific ticket ID provided
	if len(args) > 0 {
		return workSingleTicket(ctx, store, args[0])
	}

	return workAllTickets(ctx, store, parallel)
}

func workSingleTicket(ctx context.Context, store *ticket.Store, ticketID string) error {
	t, err := store.Load(ticketID)
	if err != nil {
		ui.PrintError(os.Stdout, fmt.Sprintf(i18n.ErrTicketNotFound, ticketID))
		return nil
	}

	if t.Status != ticket.StatusPending {
		ui.PrintWarning(os.Stdout, fmt.Sprintf(i18n.MsgTicketCannotProcess, ticketID, t.Status))
		return nil
	}

	ui.PrintHeader(os.Stdout, i18n.UIProcessTicket)
	ui.PrintInfo(os.Stdout, fmt.Sprintf(i18n.MsgTicketInfo, t.ID))
	ui.PrintInfo(os.Stdout, fmt.Sprintf(i18n.MsgTicketTitle, t.Title))

	return processTicket(ctx, store, t)
}

func workAllTickets(ctx context.Context, store *ticket.Store, parallel int) error {
	w := os.Stdout

	ui.PrintHeader(w, i18n.UIProcessTickets)
	ui.PrintInfo(w, fmt.Sprintf(i18n.MsgMaxParallel, parallel))

	resolver := ticket.NewDependencyResolver(store)

	results := struct {
		completed int
		failed    int
		skipped   int
		mu        sync.Mutex
	}{}

	maxIterations := 20
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			ui.PrintWarning(w, i18n.MsgProcessInterrupted)
			goto done
		default:
		}

		// Get processable tickets
		processable, err := resolver.GetProcessable()
		if err != nil {
			return err
		}

		if len(processable) == 0 {
			// Check if there are still pending tickets (blocked by dependencies)
			pending, _ := store.LoadByStatus(ticket.StatusPending)
			if len(pending) > 0 {
				ui.PrintWarning(w, fmt.Sprintf(i18n.MsgPendingBlocked, len(pending)))
				results.skipped = len(pending)
			}
			break
		}

		ui.PrintInfo(w, fmt.Sprintf(i18n.MsgIteration, iteration+1, len(processable)))

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, parallel)

		if IsDetachChild() {
			// detach-child: no TUI; processTicket writes plain text progress to log
			for _, t := range processable {
				select {
				case <-ctx.Done():
					break
				default:
				}

				wg.Add(1)
				go func(t *ticket.Ticket) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					err := processTicket(ctx, store, t)

					results.mu.Lock()
					if err != nil {
						results.failed++
					} else {
						results.completed++
					}
					results.mu.Unlock()
				}(t)
			}
			wg.Wait()
		} else {
			// Interactive: use MultiSpinner for this batch
			multiSpinner := ui.NewMultiSpinner(w)
			for _, t := range processable {
				multiSpinner.AddTask(t.ID, fmt.Sprintf(i18n.SpinnerProcessing, t.ID, t.Title))
			}
			multiSpinner.Start()

			for _, t := range processable {
				select {
				case <-ctx.Done():
					break
				default:
				}

				wg.Add(1)
				go func(t *ticket.Ticket) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					err := processTicketWithMultiSpinner(ctx, store, t, multiSpinner)

					results.mu.Lock()
					if err != nil {
						results.failed++
					} else {
						results.completed++
					}
					results.mu.Unlock()
				}(t)
			}
			wg.Wait()
			multiSpinner.Stop()
		}
	}

done:
	// Print summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, i18n.UIProcessComplete)
	ui.PrintSuccess(w, fmt.Sprintf(i18n.MsgCountCompleted, results.completed))
	if results.failed > 0 {
		ui.PrintError(w, fmt.Sprintf(i18n.MsgCountFailed, results.failed))
	}
	if results.skipped > 0 {
		ui.PrintWarning(w, fmt.Sprintf(i18n.MsgCountSkipped, results.skipped))
	}

	return nil
}

func processTicket(ctx context.Context, store *ticket.Store, t *ticket.Ticket) error {
	w := os.Stdout
	logW := WorkLogWriter()
	useLogOnly := IsDetachChild() && logW != nil

	// Mark as in progress
	t.MarkInProgress()
	if err := store.Save(t); err != nil {
		return err
	}

	// Create coding agent
	caller, err := CreateAgentCaller()
	if err != nil {
		if useLogOnly {
			ui.WriteLogProgress(logW, i18n.SpinnerFailTicket, t.ID)
		} else {
			ui.PrintError(w, i18n.ErrAgentCommand)
		}
		t.MarkFailed(fmt.Errorf("agent command not found"))
		store.Save(t)
		return fmt.Errorf("agent not available")
	}

	codingAgent := agent.NewCodingAgent(caller, cfg.ProjectRoot)

	// Execute: detach-child uses plain text to log; otherwise use TUI spinner
	var spinner *ui.Spinner
	if !useLogOnly {
		spinner = ui.NewSpinner(fmt.Sprintf(i18n.SpinnerProcessing, t.ID, t.Title), w)
		spinner.Start()
	} else {
		ui.WriteLogProgress(logW, i18n.SpinnerProcessing, t.ID, t.Title)
	}

	result, err := codingAgent.Execute(ctx, t)

	if err != nil || !result.Success {
		if useLogOnly {
			ui.WriteLogProgress(logW, i18n.SpinnerFailTicket, t.ID)
		} else {
			spinner.Fail(fmt.Sprintf(i18n.SpinnerFailTicket, t.ID))
		}
		errMsg := "execution failed"
		if err != nil {
			errMsg = err.Error()
		} else if result != nil && result.Error != "" {
			errMsg = result.Error
		}
		t.MarkFailed(fmt.Errorf("%s", errMsg))
		if result != nil && result.LogPath != "" {
			t.ErrorLog = result.LogPath
		}
		store.Save(t)
		return fmt.Errorf("ticket %s failed: %s", t.ID, errMsg)
	}

	if useLogOnly {
		ui.WriteLogProgress(logW, i18n.MsgProcessingComplete, t.ID)
	} else {
		spinner.Success(fmt.Sprintf(i18n.MsgProcessingComplete, t.ID))
	}

	// Truncate output if too long
	output := result.Output
	if len(output) > 1000 {
		output = output[:1000] + "...(truncated)"
	}

	t.MarkCompleted(output)
	return store.Save(t)
}

func processTicketWithMultiSpinner(ctx context.Context, store *ticket.Store, t *ticket.Ticket, multiSpinner *ui.MultiSpinner) error {
	// Mark as in progress
	t.MarkInProgress()
	if err := store.Save(t); err != nil {
		return err
	}

	// Create coding agent
	caller, err := CreateAgentCaller()
	if err != nil {
		multiSpinner.FailTask(t.ID, fmt.Sprintf(i18n.SpinnerFailTicket, t.ID))
		t.MarkFailed(fmt.Errorf("agent command not found"))
		store.Save(t)
		return fmt.Errorf("agent not available")
	}

	codingAgent := agent.NewCodingAgent(caller, cfg.ProjectRoot)

	// Execute
	result, err := codingAgent.Execute(ctx, t)

	if err != nil || !result.Success {
		multiSpinner.FailTask(t.ID, fmt.Sprintf(i18n.SpinnerFailTicket, t.ID))
		errMsg := "execution failed"
		if err != nil {
			errMsg = err.Error()
		} else if result != nil && result.Error != "" {
			errMsg = result.Error
		}
		t.MarkFailed(fmt.Errorf("%s", errMsg))
		if result != nil && result.LogPath != "" {
			t.ErrorLog = result.LogPath
		}
		store.Save(t)
		return fmt.Errorf("ticket %s failed: %s", t.ID, errMsg)
	}

	multiSpinner.CompleteTask(t.ID, fmt.Sprintf(i18n.MsgProcessingComplete, t.ID))

	// Truncate output if too long
	output := result.Output
	if len(output) > 1000 {
		output = output[:1000] + "...(truncated)"
	}

	t.MarkCompleted(output)
	return store.Save(t)
}
