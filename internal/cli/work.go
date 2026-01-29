package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/anthropic/agent-orchestrator/internal/agent"
	"github.com/anthropic/agent-orchestrator/internal/ticket"
	"github.com/anthropic/agent-orchestrator/internal/ui"
	"github.com/spf13/cobra"
)

var (
	workParallel int
)

var workCmd = &cobra.Command{
	Use:   "work [ticket-id]",
	Short: "處理 pending tickets",
	Long: `處理所有 pending 狀態的 tickets，或指定單一 ticket 處理。

範例:
  agent-orchestrator work              # 處理所有 pending tickets
  agent-orchestrator work TICKET-001   # 處理指定 ticket
  agent-orchestrator work -p 5         # 使用 5 個並行 agents`,
	RunE: runWork,
}

func init() {
	workCmd.Flags().IntVarP(&workParallel, "parallel", "p", 0, "最大並行 agents 數量 (預設使用設定值)")
}

func runWork(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		ui.PrintWarning(os.Stdout, "\n收到中斷信號，正在優雅關閉...")
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
		return fmt.Errorf("初始化 ticket store 失敗: %w", err)
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
		ui.PrintError(os.Stdout, "找不到 ticket: "+ticketID)
		return nil
	}

	if t.Status != ticket.StatusPending {
		ui.PrintWarning(os.Stdout, fmt.Sprintf("Ticket %s 狀態為 %s，無法處理", ticketID, t.Status))
		return nil
	}

	ui.PrintHeader(os.Stdout, "處理 Ticket")
	ui.PrintInfo(os.Stdout, fmt.Sprintf("ID: %s", t.ID))
	ui.PrintInfo(os.Stdout, fmt.Sprintf("標題: %s", t.Title))

	return processTicket(ctx, store, t)
}

func workAllTickets(ctx context.Context, store *ticket.Store, parallel int) error {
	w := os.Stdout

	ui.PrintHeader(w, "處理 Tickets")
	ui.PrintInfo(w, fmt.Sprintf("最大並行數: %d", parallel))

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
			ui.PrintWarning(w, "處理已中斷")
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
				ui.PrintWarning(w, fmt.Sprintf("還有 %d 個 tickets 但依賴未滿足", len(pending)))
				results.skipped = len(pending)
			}
			break
		}

		ui.PrintInfo(w, fmt.Sprintf("迭代 %d: 處理 %d 個 tickets", iteration+1, len(processable)))

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
	}

done:
	// Print summary
	ui.PrintInfo(w, "")
	ui.PrintHeader(w, "處理完成")
	ui.PrintSuccess(w, fmt.Sprintf("完成: %d", results.completed))
	if results.failed > 0 {
		ui.PrintError(w, fmt.Sprintf("失敗: %d", results.failed))
	}
	if results.skipped > 0 {
		ui.PrintWarning(w, fmt.Sprintf("跳過: %d", results.skipped))
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
	caller := agent.NewCaller(
		cfg.AgentCommand,
		cfg.AgentForce,
		cfg.AgentOutputFormat,
		cfg.LogsDir,
	)
	caller.SetDryRun(cfg.DryRun)
	caller.SetVerbose(cfg.Verbose)

	if !caller.IsAvailable() && !cfg.DryRun {
		ui.PrintError(w, "找不到 agent 指令")
		t.MarkFailed(fmt.Errorf("agent command not found"))
		store.Save(t)
		return fmt.Errorf("agent not available")
	}

	codingAgent := agent.NewCodingAgent(caller, cfg.ProjectRoot)

	// Execute
	spinner := ui.NewSpinner(fmt.Sprintf("處理 %s: %s", t.ID, t.Title), w)
	spinner.Start()

	result, err := codingAgent.Execute(ctx, t)
	
	if err != nil || !result.Success {
		spinner.Fail(fmt.Sprintf("%s 失敗", t.ID))
		errMsg := "execution failed"
		if err != nil {
			errMsg = err.Error()
		} else if result.Error != "" {
			errMsg = result.Error
		}
		t.MarkFailed(fmt.Errorf(errMsg))
		store.Save(t)
		return fmt.Errorf("ticket %s failed: %s", t.ID, errMsg)
	}

	spinner.Success(fmt.Sprintf("%s 完成", t.ID))
	
	// Truncate output if too long
	output := result.Output
	if len(output) > 1000 {
		output = output[:1000] + "...(truncated)"
	}
	
	t.MarkCompleted(output)
	return store.Save(t)
}
