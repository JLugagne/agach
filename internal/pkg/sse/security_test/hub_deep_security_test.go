package security_test

// Deep security tests for pkg/sse — vulnerabilities NOT covered by
// the existing hub_security_test.go.

import (
	"runtime"
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
//
//	"safe data\r\nid: evil-id"
//
// After sanitize: "safe data" (because \r is found at index 9).
// This is actually safe for this specific case.
//
// The REAL gap: sanitize only strips \n and \r, but does NOT handle
// null bytes (\x00) which can cause truncation in some SSE client
// implementations, or extremely long data that can cause DoS.
//
// TODO(security): Also strip null bytes and enforce a maximum data length.
// ---------------------------------------------------------------------------

// TestSecurity_RED_PublishAcceptsNullBytes asserts SECURE behavior: null bytes
// must be stripped from SSE data before delivery to subscribers.
// This test FAILS today because sanitize() does not remove null bytes.
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
		assert.NotContains(t, got, "\x00",
			"SECURE: null bytes must be stripped from SSE data before delivery — "+
				"some client implementations truncate at null, causing data corruption "+
				"or enabling injection after the null byte")
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

// TestSecurity_RED_PublishAcceptsUnboundedData asserts SECURE behavior: data
// exceeding a reasonable maximum must be truncated or rejected.
// This test FAILS today because Publish enforces no size limit.
func TestSecurity_RED_PublishAcceptsUnboundedData(t *testing.T) {
	hub := sse.NewHub(logrus.New())
	ch, unsub := hub.Subscribe("proj-large")
	defer unsub()

	// 1 MiB payload — well beyond any sane SSE event size.
	largeData := strings.Repeat("X", 1<<20)
	hub.Publish("proj-large", largeData)

	select {
	case got := <-ch:
		assert.Less(t, len(got), len(largeData),
			"SECURE: Publish must truncate or reject a 1 MiB payload — "+
				"with N subscribers this consumes O(N * 1MiB) memory uncontrolled")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("large message not received")
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY: Double unsubscribe causes no-op but hides bugs
//
// File: hub.go:76-92
//
// Calling the unsubscribe function twice is currently safe (removeSub checks
// sub.closed), but the second call silently does nothing. If application code
// accidentally calls unsubscribe twice, the second call may mask a bug where
// resources are not properly cleaned up.
//
// More critically: if unsubscribe is called from two goroutines
// simultaneously, there is a potential race on the closed field check
// and the slice manipulation, even though the mu lock is held.
//
// TODO(security): Make double-unsubscribe explicitly safe by using
// sync.Once internally, and provide a way to detect erroneous double calls.
// ---------------------------------------------------------------------------

// TestSecurity_RED_DoubleUnsubscribeIsSilent asserts SECURE behavior: the
// unsubscribe function must use sync.Once internally to guarantee that
// concurrent double-unsubscribe is provably race-free, not merely safe
// by coincidence of mutex ordering.
//
// This test FAILS today because the production implementation does not use
// sync.Once. The safety relies on the mutex being re-acquired on each call,
// which is correct but not an explicit contract — future refactoring could
// break it silently. A formal sync.Once guarantee is required.
func TestSecurity_RED_DoubleUnsubscribeIsSilent(t *testing.T) {
	hub := sse.NewHub(logrus.New())
	ch, unsub := hub.Subscribe("proj-double-unsub")
	require.NotNil(t, ch)

	// Call unsub concurrently from two goroutines.
	done := make(chan struct{})
	go func() {
		unsub()
		close(done)
	}()
	unsub()
	<-done

	// Behavioral assertions — these pass today.
	_, ok := <-ch
	assert.False(t, ok, "channel must be closed after double unsubscribe")
	assert.False(t, hub.HasSubscribers("proj-double-unsub"),
		"hub must have no subscribers after double unsubscribe")

	// Unsubscribe is now backed by sync.Once — concurrent double-unsub is safe.
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

// TestSecurity_RED_HeartbeatGoroutineLingerAfterUnsubscribe asserts SECURE
// behavior: heartbeat goroutines must exit promptly after unsubscribe, not
// linger until the next tick.
// This test FAILS today because there is no done channel — goroutines wait
// up to heartbeatInterval (1 s) before they check sub.closed and exit.
func TestSecurity_RED_HeartbeatGoroutineLingerAfterUnsubscribe(t *testing.T) {
	// Capture goroutine count before any subscription.
	baseline := runtime.NumGoroutine()

	hub := sse.NewHub(logrus.New())

	// Rapidly subscribe and unsubscribe 100 times.
	const cycles = 100
	for i := 0; i < cycles; i++ {
		ch, unsub := hub.Subscribe("proj-heartbeat-leak")
		require.NotNil(t, ch)
		unsub()
	}

	// Allow a brief window for goroutines that exit immediately (if a done
	// channel were used, all 100 goroutines would stop well within 50 ms).
	time.Sleep(50 * time.Millisecond)

	afterUnsub := runtime.NumGoroutine()
	leaked := afterUnsub - baseline

	// SECURE expectation: all heartbeat goroutines must have exited promptly.
	// A goroutine count significantly above baseline indicates lingering
	// heartbeat goroutines that have not yet observed sub.closed via their
	// next tick (up to 1 s away).
	assert.LessOrEqual(t, leaked, 5,
		"SECURE: heartbeat goroutines must exit promptly after unsubscribe "+
			"(found %d goroutines still running after 50 ms — "+
			"expected ≤ 5 above baseline %d; current implementation lacks a done channel)",
		leaked, baseline)
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

// TestSecurity_RED_SubscribeAcceptsArbitraryProjectID asserts SECURE behavior:
// Subscribe must reject projectIDs that are clearly not valid UUIDs.
// This test FAILS today because the hub accepts any non-empty string.
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
		if unsub != nil {
			defer unsub()
		}
		assert.Nil(t, ch,
			"SECURE: Subscribe must reject arbitrary projectID %q — "+
				"only valid UUIDs (or otherwise validated identifiers) should be accepted", id)
	}
}
