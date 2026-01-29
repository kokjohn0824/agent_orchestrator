package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/i18n"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	workParallel int
)

var workCmd = &cobra.Command{
	Use:   "work [ticket-id]",
	Short: i18n.CmdWorkShort,
	Long:  i18n.CmdWorkLong,
	RunE:  runWork,
}

func init() {
	workCmd.Flags().IntVarP(&workParallel, "parallel", "p", 0, i18n.FlagParallel)
}

func runWork(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
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

		// Create multi-spinner for this batch
		multiSpinner := ui.NewMultiSpinner(w)

		// Add all tasks to the multi-spinner first
		for _, t := range processable {
			multiSpinner.AddTask(t.ID, fmt.Sprintf(i18n.SpinnerProcessing, t.ID, t.Title))
		}

		// Start the multi-spinner
		multiSpinner.Start()

		// Process tickets in parallel
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, parallel)

		for _, t := range processable {
			// Check for cancellation
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

		// Stop the multi-spinner after all tasks in this batch are done
		multiSpinner.Stop()
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

	// Mark as in progress
	t.MarkInProgress()
	if err := store.Save(t); err != nil {
		return err
	}

	// Create coding agent
	caller, err := CreateAgentCaller()
	if err != nil {
		ui.PrintError(w, i18n.ErrAgentCommand)
		t.MarkFailed(fmt.Errorf("agent command not found"))
		store.Save(t)
		return fmt.Errorf("agent not available")
	}

	codingAgent := agent.NewCodingAgent(caller, cfg.ProjectRoot)

	// Execute
	spinner := ui.NewSpinner(fmt.Sprintf(i18n.SpinnerProcessing, t.ID, t.Title), w)
	spinner.Start()

	result, err := codingAgent.Execute(ctx, t)

	if err != nil || !result.Success {
		spinner.Fail(fmt.Sprintf(i18n.SpinnerFailTicket, t.ID))
		errMsg := "execution failed"
		if err != nil {
			errMsg = err.Error()
		} else if result.Error != "" {
			errMsg = result.Error
		}
		t.MarkFailed(fmt.Errorf("%s", errMsg))
		store.Save(t)
		return fmt.Errorf("ticket %s failed: %s", t.ID, errMsg)
	}

	spinner.Success(fmt.Sprintf(i18n.MsgProcessingComplete, t.ID))

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
		} else if result.Error != "" {
			errMsg = result.Error
		}
		t.MarkFailed(fmt.Errorf("%s", errMsg))
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
