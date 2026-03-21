package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/agach/domain"
)

// RunDiagnostic launches a cold-start probe for a baseline (no agent) and each
// agent in agents. Results are sent via the updates channel as they complete.
// The channel is closed when all probes are done.
func RunDiagnostic(ctx context.Context, workDir string, agents []AgentDef, updates chan<- domain.DiagnosticUpdate) {
	defer close(updates)

	// Build results slice: baseline first, then one per agent
	results := make([]domain.DiagnosticResult, 0, 1+len(agents))
	results = append(results, domain.DiagnosticResult{Status: domain.DiagnosticPending})
	for _, ag := range agents {
		results = append(results, domain.DiagnosticResult{
			AgentSlug: ag.Slug,
			Status:    domain.DiagnosticPending,
		})
	}

	send := func(done bool) {
		snap := make([]domain.DiagnosticResult, len(results))
		copy(snap, results)
		select {
		case updates <- domain.DiagnosticUpdate{Results: snap, Done: done}:
		case <-ctx.Done():
		}
	}

	send(false)

	for i := range results {
		if ctx.Err() != nil {
			return
		}

		results[i].Status = domain.DiagnosticRunning
		send(false)

		r, err := runDiagnosticProbe(ctx, workDir, results[i].AgentSlug)
		if err != nil {
			results[i].Status = domain.DiagnosticError
			results[i].Error = err.Error()
		} else {
			results[i] = r
			results[i].Status = domain.DiagnosticDone
		}

		done := i == len(results)-1
		send(done)
	}
}

// diagInitEvent captures fields from the system init event
type diagInitEvent struct {
	Type   string   `json:"type"`
	Tools  []string `json:"tools"`
	Agents []string `json:"agents"`
	Skills []string `json:"skills"`
}

// diagResultEvent captures fields from the result event
type diagResultEvent struct {
	Type         string  `json:"type"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	ModelUsage   map[string]struct {
		InputTokens              int     `json:"inputTokens"`
		OutputTokens             int     `json:"outputTokens"`
		CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
		CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
		CostUSD                  float64 `json:"costUSD"`
	} `json:"modelUsage"`
}

// runDiagnosticProbe runs a single claude invocation and returns token metrics.
func runDiagnosticProbe(ctx context.Context, workDir, agentSlug string) (domain.DiagnosticResult, error) {
	args := []string{
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", "1",
		"-p", "Reply with exactly one word: DONE",
	}
	if agentSlug != "" {
		args = append(args, "--agent", agentSlug)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return domain.DiagnosticResult{AgentSlug: agentSlug}, fmt.Errorf("stdout pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return domain.DiagnosticResult{AgentSlug: agentSlug}, fmt.Errorf("start: %w", err)
	}

	var run domain.TaskRun
	result := domain.DiagnosticResult{AgentSlug: agentSlug}
	var sessionID string

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse init event for metadata
		var rawType struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
		}
		if json.Unmarshal([]byte(line), &rawType) != nil {
			continue
		}

		if rawType.SessionID != "" && sessionID == "" && isValidSession(rawType.SessionID) {
			sessionID = rawType.SessionID
		}

		switch rawType.Type {
		case "system":
			var init diagInitEvent
			if json.Unmarshal([]byte(line), &init) == nil {
				// Separate system tools from MCP tools
				for _, t := range init.Tools {
					if strings.HasPrefix(t, "mcp__") {
						result.MCPToolCount++
						result.MCPTools = append(result.MCPTools, t)
					} else {
						result.SystemToolCount++
					}
				}
				result.Agents = init.Agents
				result.AgentCount = len(init.Agents)
				result.Skills = init.Skills
				result.SkillCount = len(init.Skills)
			}

		case "assistant":
			ev, ok := parseStreamLine(line)
			if ok {
				applyAssistantEvent(ev, &run)
			}

		case "result":
			var res diagResultEvent
			if json.Unmarshal([]byte(line), &res) == nil {
				result.CostUSD = res.TotalCostUSD
				for model := range res.ModelUsage {
					result.Model = model
					break
				}
			}
		}
	}

	_ = cmd.Wait()
	result.Duration = time.Since(start)
	result.InputTokens = run.ColdStartInputTokens + run.ColdStartCacheReadInputTokens + run.ColdStartCacheCreationInputTokens
	result.OutputTokens = run.ColdStartOutputTokens
	result.CacheReadInputTokens = run.ColdStartCacheReadInputTokens
	result.CacheCreationInputTokens = run.ColdStartCacheCreationInputTokens

	// Fetch /context from the session
	if sessionID != "" && ctx.Err() == nil {
		if contextRaw, err := fetchSessionContext(ctx, workDir, sessionID); err == nil {
			result.ContextRaw = contextRaw
		}
	}

	return result, nil
}

// fetchSessionContext resumes a claude session with /context and returns the raw output.
func fetchSessionContext(ctx context.Context, workDir, sessionID string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude",
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"--verbose",
		"--max-turns", "2",
		"--resume", sessionID,
		"-p", "/context",
	)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}

	var contextResult string
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev struct {
			Type   string `json:"type"`
			Result string `json:"result"`
		}
		if json.Unmarshal([]byte(line), &ev) == nil && ev.Type == "result" && ev.Result != "" {
			contextResult = ev.Result
		}
	}

	_ = cmd.Wait()
	return contextResult, nil
}
