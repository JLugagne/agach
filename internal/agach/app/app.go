package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JLugagne/agach-mcp/internal/agach/domain"
	"github.com/JLugagne/agach-mcp/pkg/server/client"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

var (
	// uuidRe validates UUID format (used for project/task IDs).
	uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// slugRe validates agent role slugs.
	slugRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	// sessionIDRe validates Claude session IDs (hex strings).
	sessionIDRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func isValidUUID(s string) bool   { return uuidRe.MatchString(s) }
func isValidSlug(s string) bool   { return s != "" && len(s) <= 128 && slugRe.MatchString(s) }
func isValidSession(s string) bool { return s != "" && len(s) <= 256 && sessionIDRe.MatchString(s) }

// WorkerUpdate is sent to the TUI whenever a worker state changes
type WorkerUpdate struct {
	WorkerID    int
	State       domain.WorkerState
	NewMessages []domain.LiveMessage
}

// App orchestrates workers that run Claude Code against kanban tasks
type App struct {
	client       *client.Client
	runningTasks sync.Map // task ID → struct{}, prevents double-execution
	pauseMu      sync.Mutex
	pauseCh      chan struct{} // closed when paused; nil when running
	liveConfig   atomic.Pointer[domain.RunConfig] // updated via UpdateConfig while running
}

func New(serverURL string) *App {
	return &App{
		client: client.New(serverURL),
	}
}

func (a *App) Client() *client.Client {
	return a.client
}

// Pause signals workers to stop picking up new tasks after finishing the current one.
// Returns immediately; workers drain naturally.
func (a *App) Pause() {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()
	if a.pauseCh == nil {
		// pauseCh is an open channel that blocks readers while paused.
		// Resume closes it, unblocking all waiters.
		a.pauseCh = make(chan struct{})
	}
}

// Resume re-enables workers to pick up new tasks.
func (a *App) Resume() {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()
	if a.pauseCh != nil {
		close(a.pauseCh) // unblock all waiters
		a.pauseCh = nil
	}
}

// IsPaused reports whether the app is currently paused.
func (a *App) IsPaused() bool {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()
	return a.pauseCh != nil
}

// waitForResume blocks until the app is resumed or the context is cancelled.
func (a *App) waitForResume(ctx context.Context) error {
	a.pauseMu.Lock()
	ch := a.pauseCh
	a.pauseMu.Unlock()
	if ch == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

// UpdateConfig replaces the live run configuration read by workers on the next task fetch.
func (a *App) UpdateConfig(cfg domain.RunConfig) {
	a.liveConfig.Store(&cfg)
}

// Run starts N workers and returns a channel of updates.
// Callers should read updates until the channel is closed.
func (a *App) Run(ctx context.Context, cfg domain.RunConfig, updates chan<- WorkerUpdate) {
	// Store initial config so workers can read live updates via liveConfig.
	a.liveConfig.Store(&cfg)

	var wg sync.WaitGroup

	// Pre-fetch MaxWorkers tasks so all workers start immediately without racing
	// on the first GetNextTasks call.
	prefetch := make(chan client.NextTaskResult, cfg.MaxWorkers)
	var subProjectID *string
	if cfg.Scope == domain.RunScopeSpecific {
		subProjectID = &cfg.SubProjectID
	}
	if tasks, err := a.client.GetNextTasks(cfg.ProjectID, cfg.MaxWorkers, cfg.RoleSlug, subProjectID, cfg.Scope == domain.RunScopeAll); err == nil {
		for _, t := range tasks {
			prefetch <- t
		}
	}
	close(prefetch)

	for i := range cfg.MaxWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			a.runWorker(ctx, workerID, prefetch, updates)
		}(i)
	}

	go func() {
		wg.Wait()
		close(updates)
	}()
}

func (a *App) runWorker(ctx context.Context, id int, prefetch <-chan client.NextTaskResult, updates chan<- WorkerUpdate) {
	sendUpdate := func(ws domain.WorkerState) {
		select {
		case updates <- WorkerUpdate{WorkerID: id, State: ws}:
		case <-ctx.Done():
		}
	}

	state := domain.WorkerState{ID: id, Status: domain.WorkerIdle}
	sendUpdate(state)

	var past []domain.TaskRun

	for {
		if ctx.Err() != nil {
			return
		}

		// Read the latest config on every iteration so Apply & Close takes effect immediately.
		cfg := *a.liveConfig.Load()

		// Block here if paused, until resumed or context cancelled
		if a.IsPaused() {
			state = domain.WorkerState{ID: id, Status: domain.WorkerIdle, Past: past}
			sendUpdate(state)
			if err := a.waitForResume(ctx); err != nil {
				return
			}
		}

		// Determine subproject scope
		var subProjectID *string
		switch cfg.Scope {
		case domain.RunScopeSpecific:
			subProjectID = &cfg.SubProjectID
		case domain.RunScopeAll:
			// include_subprojects handled by client
		}

		// Try to consume a pre-fetched task first, then fall back to individual fetch.
		var tasks []client.NextTaskResult
		var err error
		if t, ok := <-prefetch; ok {
			tasks = []client.NextTaskResult{t}
		} else {
			tasks, err = a.client.GetNextTasks(cfg.ProjectID, 1, cfg.RoleSlug, subProjectID, cfg.Scope == domain.RunScopeAll)
		}
		if err != nil || len(tasks) == 0 {
			state = domain.WorkerState{ID: id, Status: domain.WorkerIdle, Past: past}
			sendUpdate(state)
			if cfg.AutoStart {
				// Wait for SSE notification then retry immediately
				if err := a.client.WaitForNextTask(ctx, cfg.ProjectID); err != nil {
					// context cancelled or SSE error — fall back to polling
					select {
					case <-ctx.Done():
						return
					case <-time.After(5 * time.Second):
					}
				}
			} else {
				// Poll every 5 seconds
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
			continue
		}

		task := tasks[0]

		// Prevent two workers from executing the same task concurrently.
		if _, loaded := a.runningTasks.LoadOrStore(task.ID, struct{}{}); loaded {
			// Another worker already claimed this task; skip and fetch a new one.
			continue
		}

		// Move task to in_progress before starting the agent.
		// This will fail if the task has unresolved dependencies or WIP limit is exceeded.
		if err := a.client.MoveTask(task.ProjectID, task.ID, "in_progress"); err != nil {
			a.runningTasks.Delete(task.ID)
				continue
		}

		run := domain.TaskRun{
			TaskID:    task.ID,
			TaskTitle: task.Title,
			ProjectID: task.ProjectID,
			AgentRole: task.Role,
			StartedAt: time.Now(),
			Status:    domain.WorkerRunning,
		}
		if task.SessionID != "" {
			run.SessionID = task.SessionID
		}

		state = domain.WorkerState{ID: id, Status: domain.WorkerRunning, Current: &run, Past: past}
		sendUpdate(state)

		// Validate all external inputs before passing to exec
		if err := validateTaskInput(task, cfg); err != nil {
			run.Status = domain.WorkerError
			run.Error = fmt.Sprintf("input validation failed: %v", err)
			a.runningTasks.Delete(task.ID)
			past = append([]domain.TaskRun{run}, past...)
			state = domain.WorkerState{ID: id, Status: domain.WorkerIdle, Past: past}
			sendUpdate(state)
			continue
		}

		// Build prompt
		prompt := buildPrompt(task, cfg)

		// Execute claude
		if err := a.executeTask(ctx, &run, prompt, updates, id, past); err != nil {
			run.Status = domain.WorkerError
			run.Error = err.Error()
		} else {
			run.Status = domain.WorkerDone
		}

		a.runningTasks.Delete(task.ID)

		// Persist cumulative token usage and model to the API
		if run.InputTokens > 0 || run.OutputTokens > 0 {
			inputTokens := run.InputTokens
			outputTokens := run.OutputTokens
			cacheReadTokens := run.CacheReadInputTokens
			cacheWriteTokens := run.CacheCreationInputTokens
			model := run.Model
			_ = a.client.UpdateTask(run.ProjectID, run.TaskID, pkgserver.UpdateTaskRequest{
				InputTokens:      &inputTokens,
				OutputTokens:     &outputTokens,
				CacheReadTokens:  &cacheReadTokens,
				CacheWriteTokens: &cacheWriteTokens,
				Model:            &model,
			})
		}

		now := time.Now()
		run.CompletedAt = &now
		past = append([]domain.TaskRun{run}, past...)

		state = domain.WorkerState{ID: id, Status: domain.WorkerIdle, Past: past}
		sendUpdate(state)
	}
}

func validateTaskInput(task client.NextTaskResult, cfg domain.RunConfig) error {
	if !isValidUUID(task.ID) {
		return fmt.Errorf("invalid task ID: %q", task.ID)
	}
	if !isValidUUID(task.ProjectID) {
		return fmt.Errorf("invalid task project ID: %q", task.ProjectID)
	}
	if !isValidUUID(cfg.ProjectID) {
		return fmt.Errorf("invalid config project ID: %q", cfg.ProjectID)
	}
	if task.Role != "" && !isValidSlug(task.Role) {
		return fmt.Errorf("invalid task role: %q", task.Role)
	}
	if task.SessionID != "" && !isValidSession(task.SessionID) {
		return fmt.Errorf("invalid session ID: %q", task.SessionID)
	}
	return nil
}

func buildPrompt(task client.NextTaskResult, cfg domain.RunConfig) string {
	if !isValidUUID(task.ID) || !isValidUUID(task.ProjectID) || !isValidUUID(cfg.ProjectID) {
		return ""
	}
	if task.SessionID != "" {
		if task.ProjectID != cfg.ProjectID {
			return fmt.Sprintf("check status of task %s from sub project %s from agach project %s - if it was done move it to complete, otherwise check for comments and continue working",
				task.ID, task.ProjectID, cfg.ProjectID)
		}
		return fmt.Sprintf("check status of task %s from agach project %s - if it was done move it to complete, otherwise check for comments and continue working",
			task.ID, cfg.ProjectID)
	}
	if task.ProjectID != cfg.ProjectID {
		return fmt.Sprintf("execute task %s from sub project %s from agach project %s",
			task.ID, task.ProjectID, cfg.ProjectID)
	}
	return fmt.Sprintf("execute task %s from agach project %s", task.ID, cfg.ProjectID)
}

func (a *App) executeTask(
	ctx context.Context,
	run *domain.TaskRun,
	prompt string,
	updates chan<- WorkerUpdate,
	workerID int,
	past []domain.TaskRun,
) error {
	args := []string{
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"--verbose",
		"--append-system-prompt", "Be concise. No explanations unless asked.",
		"-p", prompt,
	}
	if run.AgentRole != "" {
		if !isValidSlug(run.AgentRole) {
			return fmt.Errorf("invalid agent role for exec: %q", run.AgentRole)
		}
		args = append(args, "--agent", run.AgentRole)
	}
	if run.SessionID != "" {
		if !isValidSession(run.SessionID) {
			return fmt.Errorf("invalid session ID for exec: %q", run.SessionID)
		}
		args = append(args, "--resume", run.SessionID)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var coldStartPersisted bool

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ev, ok := parseStreamLine(line)
		if !ok {
			continue
		}

		switch ev.Type {
		case "system":
			if ev.SessionID != "" && run.SessionID == "" && isValidSession(ev.SessionID) {
				run.SessionID = ev.SessionID
				// Persist session_id to the task via API (best-effort)
				_ = a.client.UpdateTaskSessionID(run.ProjectID, run.TaskID, ev.SessionID)
			}
		case "assistant":
			applyAssistantEvent(ev, run)

			// Persist cold start tokens immediately (fire-and-forget)
			if !coldStartPersisted && run.ColdStartCaptured {
				coldStartPersisted = true
				coldStartInput := run.ColdStartInputTokens
				coldStartOutput := run.ColdStartOutputTokens
				coldStartCacheRead := run.ColdStartCacheReadInputTokens
				coldStartCacheCreation := run.ColdStartCacheCreationInputTokens
				projectID := run.ProjectID
				taskID := run.TaskID
				go func() {
					_ = a.client.UpdateTask(projectID, taskID, pkgserver.UpdateTaskRequest{
						ColdStartInputTokens:      &coldStartInput,
						ColdStartOutputTokens:     &coldStartOutput,
						ColdStartCacheReadTokens:  &coldStartCacheRead,
						ColdStartCacheWriteTokens: &coldStartCacheCreation,
					})
				}()
			}
		}

		// Stamp worker ID and timestamp on each message
		now := time.Now()
		for i := range ev.Messages {
			ev.Messages[i].WorkerID = workerID
			if ev.Messages[i].At.IsZero() {
				ev.Messages[i].At = now
			}
		}

		// Send live update
		current := *run
		select {
		case updates <- WorkerUpdate{
			WorkerID: workerID,
			State:    domain.WorkerState{ID: workerID, Status: domain.WorkerRunning, Current: &current, Past: past},
		}:
		default:
		}

		// Send message update if there are new messages
		if len(ev.Messages) > 0 {
			current := *run
			select {
			case updates <- WorkerUpdate{
				WorkerID:    workerID,
				State:       domain.WorkerState{ID: workerID, Status: domain.WorkerRunning, Current: &current, Past: past},
				NewMessages: ev.Messages,
			}:
			default:
			}
		}
	}

	return cmd.Wait()
}

// parseStreamLine parses a single stream-json line into a StreamEvent.
func parseStreamLine(line string) (domain.StreamEvent, bool) {
	var ev domain.StreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return ev, false
	}

	now := time.Now()

	switch ev.Type {
	case "system":
		ev.Messages = []domain.LiveMessage{{
			Kind:    domain.MessageKindSystem,
			Content: ev.Subtype,
			At:      now,
		}}
	case "assistant":
		if ev.Message != nil {
			for _, block := range ev.Message.Content {
				switch block.Type {
				case "text":
					ev.Messages = append(ev.Messages, domain.LiveMessage{
						Kind:    domain.MessageKindAssistant,
						Content: block.Text,
						At:      now,
					})
				case "tool_use":
					content := block.Name
					if len(block.Input) > 0 {
						content += " " + string(block.Input)
					}
					ev.Messages = append(ev.Messages, domain.LiveMessage{
						Kind:    domain.MessageKindToolUse,
						Content: content,
						At:      now,
					})
				}
			}
		}
	case "user":
		if ev.Message != nil {
			for _, block := range ev.Message.Content {
				if block.Type == "tool_result" {
					ev.Messages = append(ev.Messages, domain.LiveMessage{
						Kind:    domain.MessageKindToolResult,
						Content: block.Content,
						At:      now,
					})
				}
			}
		}
	case "result":
		ev.Messages = []domain.LiveMessage{{
			Kind:    domain.MessageKindResult,
			Content: ev.Result,
			At:      now,
		}}
	}

	return ev, true
}

// applyAssistantEvent updates a TaskRun with token usage from an assistant event.
func applyAssistantEvent(ev domain.StreamEvent, run *domain.TaskRun) {
	if ev.Message != nil && ev.Message.Usage != nil {
		u := ev.Message.Usage

		// Capture cold start cost from the very first exchange
		if run.Exchanges == 0 && !run.ColdStartCaptured {
			run.ColdStartCaptured = true
			run.ColdStartInputTokens = u.InputTokens
			run.ColdStartOutputTokens = u.OutputTokens
			run.ColdStartCacheReadInputTokens = u.CacheReadInputTokens
			run.ColdStartCacheCreationInputTokens = u.CacheCreationInputTokens
		}

		run.InputTokens += u.InputTokens
		run.OutputTokens += u.OutputTokens
		run.CacheReadInputTokens += u.CacheReadInputTokens
		run.CacheCreationInputTokens += u.CacheCreationInputTokens
		run.TotalTokens = run.InputTokens + run.OutputTokens
		run.Exchanges++
		if ev.Message.Model != "" {
			run.Model = ev.Message.Model
		}
	}
}
