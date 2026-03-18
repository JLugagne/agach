package app

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/agach/domain"
)

func TestPauseResume(t *testing.T) {
	a := New("http://fake:9999")

	// Initially not paused
	if a.IsPaused() {
		t.Fatal("expected not paused initially")
	}

	// After Pause, IsPaused returns true
	a.Pause()
	if !a.IsPaused() {
		t.Fatal("expected paused after Pause()")
	}

	// Pause is idempotent
	a.Pause()
	if !a.IsPaused() {
		t.Fatal("expected still paused after double Pause()")
	}

	// After Resume, IsPaused returns false
	a.Resume()
	if a.IsPaused() {
		t.Fatal("expected not paused after Resume()")
	}

	// Resume when not paused is a no-op
	a.Resume()
	if a.IsPaused() {
		t.Fatal("expected still not paused after Resume() when not paused")
	}
}

func TestWaitForResume(t *testing.T) {
	a := New("http://fake:9999")
	a.Pause()

	// waitForResume should unblock when Resume is called
	done := make(chan error, 1)
	go func() {
		done <- a.waitForResume(context.Background())
	}()

	// Should not complete yet
	select {
	case <-done:
		t.Fatal("waitForResume returned before Resume()")
	case <-time.After(50 * time.Millisecond):
	}

	a.Resume()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForResume returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForResume did not return after Resume()")
	}
}

func TestWaitForResumeContextCancel(t *testing.T) {
	a := New("http://fake:9999")
	a.Pause()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- a.waitForResume(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForResume did not return after context cancel")
	}
}

func TestParseStreamLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOK  bool
		wantTyp string
	}{
		{
			name:    "valid assistant event",
			line:    `{"type":"assistant","message":{"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":2,"cache_creation_input_tokens":1},"model":"claude-sonnet-4-20250514"}}`,
			wantOK:  true,
			wantTyp: "assistant",
		},
		{
			name:    "valid system event",
			line:    `{"type":"system","session_id":"sess-123"}`,
			wantOK:  true,
			wantTyp: "system",
		},
		{
			name:   "invalid json",
			line:   `not json at all`,
			wantOK: false,
		},
		{
			name:   "empty string",
			line:   ``,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, ok := parseStreamLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseStreamLine ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && ev.Type != tt.wantTyp {
				t.Fatalf("event type = %q, want %q", ev.Type, tt.wantTyp)
			}
		})
	}
}

func TestApplyAssistantEvent(t *testing.T) {
	run := &domain.TaskRun{}

	lines := []string{
		`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":20,"cache_creation_input_tokens":10},"model":"claude-sonnet-4-20250514"}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":200,"output_tokens":80,"cache_read_input_tokens":30,"cache_creation_input_tokens":5},"model":"claude-sonnet-4-20250514"}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":150,"output_tokens":60,"cache_read_input_tokens":10,"cache_creation_input_tokens":0},"model":"claude-opus-4-20250514"}}`,
	}

	for _, line := range lines {
		ev, ok := parseStreamLine(line)
		if !ok {
			t.Fatalf("failed to parse: %s", line)
		}
		applyAssistantEvent(ev, run)
	}

	if run.Exchanges != 3 {
		t.Fatalf("Exchanges = %d, want 3", run.Exchanges)
	}
	if run.InputTokens != 450 {
		t.Fatalf("InputTokens = %d, want 450", run.InputTokens)
	}
	if run.OutputTokens != 190 {
		t.Fatalf("OutputTokens = %d, want 190", run.OutputTokens)
	}
	if run.CacheReadInputTokens != 60 {
		t.Fatalf("CacheReadTokens = %d, want 60", run.CacheReadInputTokens)
	}
	if run.CacheCreationInputTokens != 15 {
		t.Fatalf("CacheWriteTokens = %d, want 15", run.CacheCreationInputTokens)
	}
	if run.TotalTokens != 640 {
		t.Fatalf("TotalTokens = %d, want 640", run.TotalTokens)
	}
	if run.Model != "claude-opus-4-20250514" {
		t.Fatalf("Model = %q, want claude-opus-4-20250514", run.Model)
	}
}

func TestApplyAssistantEvent_ColdStart(t *testing.T) {
	run := &domain.TaskRun{}

	ev, ok := parseStreamLine(`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":20,"cache_creation_input_tokens":10},"model":"claude-sonnet-4-20250514"}}`)
	if !ok {
		t.Fatal("failed to parse")
	}
	applyAssistantEvent(ev, run)

	if !run.ColdStartCaptured {
		t.Fatal("expected ColdStartCaptured to be true")
	}
	if run.ColdStartInputTokens != 100 {
		t.Fatalf("ColdStartInputTokens = %d, want 100", run.ColdStartInputTokens)
	}
	if run.ColdStartOutputTokens != 50 {
		t.Fatalf("ColdStartOutputTokens = %d, want 50", run.ColdStartOutputTokens)
	}
	if run.ColdStartCacheReadInputTokens != 20 {
		t.Fatalf("ColdStartCacheReadTokens = %d, want 20", run.ColdStartCacheReadInputTokens)
	}
	if run.ColdStartCacheCreationInputTokens != 10 {
		t.Fatalf("ColdStartCacheWriteTokens = %d, want 10", run.ColdStartCacheCreationInputTokens)
	}
}

func TestApplyAssistantEvent_ColdStartNotOverwritten(t *testing.T) {
	run := &domain.TaskRun{}

	// First exchange — sets cold start
	ev1, ok := parseStreamLine(`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":20,"cache_creation_input_tokens":10},"model":"claude-sonnet-4-20250514"}}`)
	if !ok {
		t.Fatal("failed to parse first event")
	}
	applyAssistantEvent(ev1, run)

	// Second exchange — should NOT overwrite cold start
	ev2, ok := parseStreamLine(`{"type":"assistant","message":{"usage":{"input_tokens":200,"output_tokens":80,"cache_read_input_tokens":30,"cache_creation_input_tokens":5},"model":"claude-sonnet-4-20250514"}}`)
	if !ok {
		t.Fatal("failed to parse second event")
	}
	applyAssistantEvent(ev2, run)

	if !run.ColdStartCaptured {
		t.Fatal("expected ColdStartCaptured to remain true")
	}
	if run.ColdStartInputTokens != 100 {
		t.Fatalf("ColdStartInputTokens = %d, want 100 (should not be overwritten)", run.ColdStartInputTokens)
	}
	if run.InputTokens != 300 {
		t.Fatalf("InputTokens = %d, want 300 (sum of both exchanges)", run.InputTokens)
	}
}

func TestApplyAssistantEvent_NoUsage(t *testing.T) {
	run := &domain.TaskRun{}

	// Event with no usage should not increment exchanges or set cold start
	ev, ok := parseStreamLine(`{"type":"assistant","message":{}}`)
	if !ok {
		t.Fatal("failed to parse")
	}
	applyAssistantEvent(ev, run)

	if run.Exchanges != 0 {
		t.Fatalf("Exchanges = %d, want 0 (no usage)", run.Exchanges)
	}
	if run.ColdStartCaptured {
		t.Fatal("expected ColdStartCaptured to be false when no usage")
	}
	if run.ColdStartInputTokens != 0 {
		t.Fatalf("ColdStartInputTokens = %d, want 0", run.ColdStartInputTokens)
	}
}
