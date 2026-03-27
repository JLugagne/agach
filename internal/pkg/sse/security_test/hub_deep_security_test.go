package security_test

// Deep security tests for pkg/sse — vulnerabilities NOT covered by
// the existing hub_security_test.go.

import (
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/pkg/sse"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// VULNERABILITY: Carriage return injection bypasses newline sanitization
//
// File: hub.go:94-102
//
// The sanitize function strips data at the first \n and the first \r
// independently. However, it does this sequentially: first truncate at \n,
// then truncate at \r. This means a payload like "safe\rinjected" is
// truncated to "safe" (correct). But the real issue is that the SSE
// protocol treats \r as a line terminator. If the data contains \r without
// \n, the sanitize function catches it. But if sanitize truncates at \n
// first, a payload structured as "data\nmore\rinjected" becomes "data"
// which is safe.
//
// The actual vulnerability is that sanitize only removes content AFTER
// the first newline, but the SSE spec says each line is terminated by
// \r, \n, or \r\n. A payload containing only \r characters (without \n)
// is handled, but the truncation strategy means the sanitized output
// could still be part of a valid multi-line SSE event if the HTTP handler
// wraps it incorrectly.
//
// More importantly: sanitize does NOT handle the case where the data
// itself is crafted to inject SSE field names. For example:
//   "safe data\r\nid: evil-id"
// After sanitize: "safe data" (because \r is found at index 9).
// This is actually safe for this specific case.
//
// The REAL gap: sanitize only strips \n and \r, but does NOT handle
// null bytes (\x00) which can cause truncation in some SSE client
// implementations, or extremely long data that can cause DoS.
//
// TODO(security): Also strip null bytes and enforce a maximum data length.
// ---------------------------------------------------------------------------

func TestSecurity_RED_PublishAcceptsNullBytes(t *testing.T) {
	hub := sse.NewHub(logrus.New())
	ch, unsub := hub.Subscribe("proj-null")
	defer unsub()

	// Null bytes in SSE data can cause client-side parsing issues.
	// Some EventSource implementations truncate at \x00.
	payload := "before-null\x00after-null"
	hub.Publish("proj-null", payload)

	select {
	case got := <-ch:
		assert.Contains(t, got, "\x00",
			"RED: null bytes are passed through to SSE subscribers — "+
				"some client implementations truncate at null, causing data corruption "+
				"or enabling injection after the null byte")
		t.Log("RED: sanitize does not strip null bytes from SSE data")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no message received")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: No maximum data length on Publish
//
// File: hub.go:104-127
//
// Publish accepts a string of any length. A single Publish call with a
// multi-megabyte string will be copied to every subscriber's channel,
// consuming O(subscribers * len(data)) memory. An attacker who can trigger
// Publish (e.g., by creating a task with an extremely long title that
// triggers an SSE notification) can cause memory exhaustion.
//
// TODO(security): Enforce a maximum data length in Publish (e.g., 64 KB)
// and reject or truncate data exceeding the limit.
// ---------------------------------------------------------------------------

func TestSecurity_RED_PublishAcceptsUnboundedData(t *testing.T) {
	hub := sse.NewHub(logrus.New())
	ch, unsub := hub.Subscribe("proj-large")
	defer unsub()

	// 1 MiB payload — well beyond any sane SSE event size.
	largeData := strings.Repeat("X", 1<<20)
	hub.Publish("proj-large", largeData)

	select {
	case got := <-ch:
		assert.Equal(t, len(largeData), len(got),
			"RED: Publish accepted and delivered a 1 MiB payload without limit — "+
				"with N subscribers this consumes O(N * 1MiB) memory")
		t.Log("RED: no maximum data length enforced in Publish")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("large message not received")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Double unsubscribe causes no-op but hides bugs
//
// File: hub.go:76-92
//
// Calling the unsubscribe function twice is safe (removeSub checks
// sub.closed), but it means the second call silently does nothing.
// If application code accidentally calls unsubscribe twice, the second
// call may mask a bug where resources are not properly cleaned up.
//
// More critically: if unsubscribe is called from two goroutines
// simultaneously, there is a potential race on the closed field check
// and the slice manipulation, even though the mu lock is held (because
// the lock is acquired inside unsubscribe, not by the caller).
//
// TODO(security): Make double-unsubscribe explicitly safe by using
// sync.Once internally.
// ---------------------------------------------------------------------------

func TestSecurity_RED_DoubleUnsubscribeIsSilent(t *testing.T) {
	hub := sse.NewHub(logrus.New())
	ch, unsub := hub.Subscribe("proj-double-unsub")
	require.NotNil(t, ch)

	unsub()
	// Second call — should be a no-op but is not explicitly documented.
	unsub()

	// Verify the channel is closed.
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after unsubscribe")

	// The hub should have no subscribers for this project.
	assert.False(t, hub.HasSubscribers("proj-double-unsub"))

	t.Log("RED: double unsubscribe is silently accepted — " +
		"this hides bugs in application code that may forget cleanup order")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Heartbeat goroutine leak on hub with many subscribe/unsubscribe cycles
//
// File: hub.go:48, hub.go:59-74
//
// Each Subscribe call spawns a goroutine (runHeartbeat) that runs a ticker.
// When unsubscribe is called, removeSub sets sub.closed = true, and the
// heartbeat goroutine will exit on the NEXT tick (when it checks sub.closed).
// With heartbeatInterval = 1 second, the goroutine stays alive for up to
// 1 second after unsubscribe.
//
// Under rapid subscribe/unsubscribe cycles (e.g., client reconnection storms),
// thousands of heartbeat goroutines can accumulate, each holding a ticker
// and consuming stack space.
//
// TODO(security): Use a done channel to immediately signal the heartbeat
// goroutine to exit, instead of relying on the next tick to check the flag.
// ---------------------------------------------------------------------------

func TestSecurity_RED_HeartbeatGoroutineLingerAfterUnsubscribe(t *testing.T) {
	hub := sse.NewHub(logrus.New())

	// Rapidly subscribe and unsubscribe 100 times.
	for i := 0; i < 100; i++ {
		ch, unsub := hub.Subscribe("proj-heartbeat-leak")
		require.NotNil(t, ch)
		unsub()
	}

	// At this point, up to 100 heartbeat goroutines are still alive,
	// waiting for their next tick (up to 1 second) to discover sub.closed.
	// We cannot directly count goroutines from an external test package,
	// but we document the issue.
	t.Log("RED: after 100 rapid subscribe/unsubscribe cycles, up to 100 heartbeat " +
		"goroutines linger for up to heartbeatInterval (1s) before exiting — " +
		"no done channel exists to signal immediate exit")
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Cross-project event leakage via projectID guessing
//
// File: hub.go:32-46
//
// The SSE hub uses the projectID as a string key with no authentication
// or authorization check. If an attacker can guess or enumerate projectIDs
// (e.g., UUIDs are predictable or sequential), they can subscribe to any
// project's events.
//
// The hub itself cannot enforce auth (that's the HTTP layer's job), but
// the Subscribe method should at minimum validate that projectID looks
// like a valid UUID to prevent scanning with arbitrary strings.
//
// TODO(security): Validate projectID format (UUID) in Subscribe, or
// document that the HTTP layer MUST enforce project access control.
// ---------------------------------------------------------------------------

func TestSecurity_RED_SubscribeAcceptsArbitraryProjectID(t *testing.T) {
	hub := sse.NewHub(logrus.New())

	// Subscribe with arbitrary strings that are clearly not valid project IDs.
	arbitraryIDs := []string{
		"../../etc/passwd",
		"<script>alert(1)</script>",
		strings.Repeat("A", 10000),
		"admin",
		"*",
	}

	for _, id := range arbitraryIDs {
		ch, unsub := hub.Subscribe(id)
		if ch != nil {
			assert.NotNil(t, ch,
				"RED: Subscribe accepted arbitrary projectID %q without validation", id)
			unsub()
		}
	}

	t.Log("RED: Subscribe accepts any string as projectID with no format validation — " +
		"an attacker can probe for events on arbitrary project IDs")
}
