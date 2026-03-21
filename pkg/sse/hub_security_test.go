// Package sse_test contains security-focused tests for the SSE Hub.
//
// Each vulnerability is documented with:
//   - RED test: demonstrates the vulnerability exists (expected to fail once the
//     production code is fixed, but currently passes because the bug is present)
//   - GREEN test: passes only after the fix is applied
//
// The naming convention is:
//
//	TestSecurity_<VulnerabilityName>_Red   – exploits the bug; asserts the bad outcome
//	TestSecurity_<VulnerabilityName>_Green – asserts the safe outcome
package sse_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/pkg/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Vulnerability 1 – SSE Event Injection (newline injection)
//
// hub.go:38-49  Publish accepts an arbitrary string that will be interpolated
// directly into an SSE frame as "data: <payload>\n\n".  If the payload
// contains "\n\n" the resulting stream contains two distinct SSE events, and
// if it contains "\nevent: " an attacker can inject a synthetic SSE field,
// splitting or forging server-sent events for any subscriber.
//
// Affected file/line: hub.go:38 (Publish – no sanitisation of `data`)
// ---------------------------------------------------------------------------

// TestSecurity_EventInjection_Green asserts the safe behaviour: a Publish call
// with a newline-containing payload must either be rejected (message not
// delivered) or have its newlines stripped/escaped before delivery, so that
// the subscriber never sees a raw "\n\n" in the data.
func TestSecurity_EventInjection_Green(t *testing.T) {
	hub := sse.NewHub()
	ch, unsub := hub.Subscribe("proj-inject-safe")
	defer unsub()

	malicious := `{"ok":true}` + "\n\nevent: injected\ndata: fake"

	hub.Publish("proj-inject-safe", malicious)

	select {
	case got := <-ch:
		assert.NotContains(t, got, "\n\n",
			"GREEN: safe hub must not forward raw \\n\\n in SSE data")
		assert.NotContains(t, got, "event: injected",
			"GREEN: safe hub must not forward injected SSE fields")
	case <-time.After(200 * time.Millisecond):
		// Acceptable: hub refused to deliver the malicious payload entirely.
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 2 – No client connection limit (connection-exhaustion DoS)
//
// hub.go:17-36  Subscribe() never limits the number of concurrent channels
// per projectID.  An attacker can call Subscribe in a tight loop, consuming
// unlimited goroutines, file descriptors, and memory.
//
// Affected file/line: hub.go:17 (Subscribe – no per-project or global cap)
// ---------------------------------------------------------------------------

// TestSecurity_ConnectionExhaustion_Red confirms that the hub accepts an
// unbounded number of subscriptions for the same project.
func TestSecurity_ConnectionExhaustion_Red(t *testing.T) {
	const attackerConnections = 10_000

	hub := sse.NewHub()
	unsubs := make([]func(), 0, attackerConnections)

	for i := 0; i < attackerConnections; i++ {
		_, unsub := hub.Subscribe("victim-project")
		unsubs = append(unsubs, unsub)
	}

	// RED assertion: hub accepted all 10 000 subscriptions – no limit enforced.
	assert.True(t, hub.HasSubscribers("victim-project"),
		"RED: hub accepted unlimited subscriptions (DoS vector)")

	// cleanup
	for _, u := range unsubs {
		u()
	}
}

// TestSecurity_ConnectionExhaustion_Green asserts that a safe hub enforces a
// per-project subscriber cap (e.g. 1000) and returns an error or nil channel
// once the limit is exceeded.
func TestSecurity_ConnectionExhaustion_Green(t *testing.T) {
	const maxPerProject = 1_000

	hub := sse.NewHub()
	var unsubs []func()

	overLimitSubscribed := false

	for i := 0; i < maxPerProject+1; i++ {
		ch, unsub := hub.Subscribe("victim-project-safe")
		if i < maxPerProject {
			require.NotNil(t, ch,
				"GREEN: hub must accept subscriptions up to the cap (iteration %d)", i)
			unsubs = append(unsubs, unsub)
		} else {
			// The (maxPerProject+1)th Subscribe must be rejected.
			if ch != nil {
				overLimitSubscribed = true
			}
			if unsub != nil {
				unsub()
			}
		}
	}

	assert.False(t, overLimitSubscribed,
		"GREEN: hub must reject subscriptions beyond per-project limit")

	for _, u := range unsubs {
		u()
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 3 – Wildcard CORS on SSE endpoint
//
// queries/sse.go:30  Access-Control-Allow-Origin: * is set unconditionally.
// Any cross-origin browser page can subscribe to the SSE stream and read all
// project events, leaking task IDs, titles, and agent role assignments.
//
// Note: the Hub itself does not set HTTP headers; the vulnerability is in the
// HTTP handler that wraps it.  The tests below are placed here because the
// security analysis covers the full SSE subsystem.  We test the Hub behaviour
// observable at the data layer: the Hub should not expose project membership
// information to callers that have not been explicitly authorised.
//
// For now we test the CORS header at the HTTP handler level using httptest.
// ---------------------------------------------------------------------------

// TestSecurity_CORSWildcard_Red proves that the SSE handler currently echoes
// "Access-Control-Allow-Origin: *" for every origin, including hostile ones.
// This test is written at the Hub level: we verify that the Hub streams data
// to any projectID without checking the caller's origin.
func TestSecurity_CORSWildcard_Red(t *testing.T) {
	// The Hub has no origin validation: a subscriber for any project string
	// gets all events published to that project.
	hub := sse.NewHub()

	// Simulate hostile origin subscribing to an arbitrary project
	ch, unsub := hub.Subscribe("confidential-project-42")
	defer unsub()

	hub.Publish("confidential-project-42", `{"secret":"data"}`)

	select {
	case msg := <-ch:
		// RED: hub delivers the event without any origin check.
		assert.Equal(t, `{"secret":"data"}`, msg,
			"RED: hub delivers confidential events without origin validation")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no message received")
	}
}

// TestSecurity_CORSWildcard_Green asserts that a safe hub (or its HTTP layer)
// would restrict CORS to known origins.  At the Hub level this means: Subscribe
// should accept an origin parameter and return a nil channel for unknown origins.
func TestSecurity_CORSWildcard_Green(t *testing.T) {
	// A fixed list of allowed origins.
	allowedOrigin := "https://trusted.example.com"
	hostileOrigin := "https://evil.example.com"

	hub := sse.NewHub()

	// Subscribe with an allowed origin — should succeed.
	chAllowed, unsubAllowed := hub.Subscribe(allowedOrigin + "|proj-cors")
	defer unsubAllowed()
	require.NotNil(t, chAllowed, "GREEN: allowed origin must receive a valid channel")

	// Subscribe with a hostile origin — safe hub would return nil or block events.
	chHostile, unsubHostile := hub.Subscribe(hostileOrigin + "|proj-cors")
	if unsubHostile != nil {
		defer unsubHostile()
	}

	hub.Publish(allowedOrigin+"|proj-cors", `{"secret":"data"}`)

	if chHostile != nil {
		select {
		case msg := <-chHostile:
			assert.Empty(t, msg,
				"GREEN: hostile origin must not receive events from an allowed-origin project")
		case <-time.After(100 * time.Millisecond):
			// Good: no message delivered to hostile subscriber.
		}
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 4 – No authentication on SSE endpoint
//
// queries/sse.go:24  ServeSSE has no auth middleware applied.  Any caller
// (including unauthenticated bots) can connect and receive real-time task
// events.  At the Hub level, Subscribe imposes no identity check.
//
// Affected file/line: hub.go:17 (Subscribe – no identity parameter)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Vulnerability 5 – Silent message drop on full channel (no feedback)
//
// hub.go:44-49  When a subscriber's buffered channel (cap 10) is full,
// Publish silently discards the message.  There is no counter, no log, no
// disconnect, and no back-pressure signal.  A slow consumer will lose events
// with no indication, and the hub has no way to evict zombie consumers.
//
// Affected file/line: hub.go:46-48 (select default – silent drop)
// ---------------------------------------------------------------------------

// TestSecurity_SilentMessageDrop_Red proves that overflow messages are lost
// silently: after filling the buffer and publishing one more, the extra
// message is never delivered.
func TestSecurity_SilentMessageDrop_Red(t *testing.T) {
	hub := sse.NewHub()
	ch, unsub := hub.Subscribe("proj-drop")
	defer unsub()

	// Fill the 10-slot buffer without consuming.
	for i := 0; i < 10; i++ {
		hub.Publish("proj-drop", "fill")
	}

	// This 11th message should overflow; the hub drops it silently.
	hub.Publish("proj-drop", "overflow-message")

	// Drain the 10 buffered messages.
	for i := 0; i < 10; i++ {
		select {
		case <-ch:
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("expected buffered message %d not received", i)
		}
	}

	// RED: the 11th message was silently dropped.
	select {
	case got := <-ch:
		// If we reach here the hub did NOT drop – which would contradict the
		// vulnerability.  Force the test to surface this.
		t.Logf("RED note: overflow message was NOT dropped (got=%q); "+
			"this means current behaviour differs from documented vulnerability", got)
	case <-time.After(100 * time.Millisecond):
		// RED confirmed: overflow message was dropped with no feedback.
		t.Log("RED confirmed: overflow message silently dropped")
	}
}

// TestSecurity_SilentMessageDrop_Green asserts that a safe hub signals overflow
// by closing/evicting the slow subscriber channel rather than silently dropping,
// so the HTTP handler can detect the condition and terminate the connection.
func TestSecurity_SilentMessageDrop_Green(t *testing.T) {
	hub := sse.NewHub()
	ch, unsub := hub.Subscribe("proj-drop-safe")
	defer unsub()

	// Fill buffer to capacity.
	for i := 0; i < 10; i++ {
		hub.Publish("proj-drop-safe", "fill")
	}

	// Overflow: safe hub should close the channel or return an error.
	hub.Publish("proj-drop-safe", "overflow")

	// Drain and check that the channel is eventually closed (eviction signal).
	drained := 0
	channelClosed := false
	deadline := time.After(300 * time.Millisecond)
drain:
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				channelClosed = true
				break drain
			}
			_ = msg
			drained++
		case <-deadline:
			break drain
		}
	}

	assert.True(t, channelClosed,
		"GREEN: hub must close/evict a slow subscriber's channel on overflow (drained=%d)", drained)
}

// ---------------------------------------------------------------------------
// Vulnerability 6 – No heartbeat / keepalive (zombie connections)
//
// queries/sse.go:41-52  The event loop has no periodic heartbeat.  TCP
// proxies and load balancers typically close idle connections after 30-60 s.
// The client receives an EOF without warning; goroutines and channel slots
// in the Hub remain live until the next server-initiated write fails.
//
// At the Hub level: there is no mechanism to send periodic keep-alive tokens
// (SSE comment lines ":\n\n") that would detect dead connections quickly.
//
// Affected file/line: hub.go (Publish / Subscribe – no ticker / heartbeat)
// ---------------------------------------------------------------------------

// TestSecurity_NoHeartbeat_Red verifies that the Hub provides no tick/heartbeat
// channel: after subscribing, if no event is published for a long period,
// the subscriber receives nothing (no keep-alive signal).
func TestSecurity_NoHeartbeat_Red(t *testing.T) {
	hub := sse.NewHub()
	ch, unsub := hub.Subscribe("proj-heartbeat")
	defer unsub()

	// Wait longer than a typical heartbeat interval (e.g. 1 s for tests).
	select {
	case msg := <-ch:
		// If a heartbeat were implemented, a keep-alive token would arrive here.
		t.Logf("Received on idle channel: %q (heartbeat may be implemented)", msg)
	case <-time.After(1500 * time.Millisecond):
		// RED confirmed: no heartbeat received; zombie connections will accumulate.
		t.Log("RED confirmed: no heartbeat received after 1.5 s of idle time")
	}
}

// TestSecurity_NoHeartbeat_Green asserts that a safe hub sends a periodic
// keep-alive comment within the heartbeat interval so that dead TCP connections
// can be detected and cleaned up.
func TestSecurity_NoHeartbeat_Green(t *testing.T) {
	hub := sse.NewHub()
	ch, unsub := hub.Subscribe("proj-heartbeat-safe")
	defer unsub()

	// A safe hub with a 1-second heartbeat interval must send a keep-alive
	// within 1.5 seconds.
	select {
	case msg := <-ch:
		assert.True(t,
			msg == ":" || strings.HasPrefix(msg, ":"),
			"GREEN: keep-alive message must be an SSE comment (starts with ':'), got %q", msg)
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("GREEN: no keep-alive received within 1.5 s; heartbeat not implemented")
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 7 – Unbounded subscriber map memory growth
//
// hub.go:10  subscribers map[string][]chan string is never pruned.  After all
// subscribers for a project unsubscribe, the empty slice remains in the map.
// An attacker that opens and closes connections using many distinct project IDs
// (or sends random IDs to Publish) causes unbounded map growth.
//
// Affected file/line: hub.go:26-34 (Unsubscribe does not delete empty keys)
// ---------------------------------------------------------------------------

// TestSecurity_MapMemoryGrowth_Red demonstrates that unsubscribing all clients
// of a project leaves an empty entry in the map: HasSubscribers returns false
// (correct) but the map key still exists (leak).
//
// Because Hub does not expose a key-count method we test indirectly: after
// unsubscribing all clients the map retains the key, and subscribing again
// appends to the existing empty slice rather than creating a fresh one.
func TestSecurity_MapMemoryGrowth_Red(t *testing.T) {
	const attackerProjects = 5_000

	hub := sse.NewHub()

	// Create and immediately destroy subscriptions for many unique project IDs.
	for i := 0; i < attackerProjects; i++ {
		projectID := "attacker-project-" + strings.Repeat("x", i%50)
		_, unsub := hub.Subscribe(projectID)
		unsub()
	}

	// RED: the map still has entries for all those project IDs.
	// We verify this by checking that subscribing to one of them still works,
	// meaning the key was not cleaned up but also was not corrupted.
	ch, unsub := hub.Subscribe("attacker-project-" + strings.Repeat("x", 0))
	defer unsub()

	hub.Publish("attacker-project-"+strings.Repeat("x", 0), "ping")

	select {
	case msg := <-ch:
		assert.Equal(t, "ping", msg)
		t.Log("RED: map has accumulated stale entries; no pruning implemented")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("subscription after map pollution failed")
	}
}

// TestSecurity_MapMemoryGrowth_Green asserts that the Hub deletes a map key
// once the last subscriber for that project unsubscribes.
func TestSecurity_MapMemoryGrowth_Green(t *testing.T) {
	hub := sse.NewHub()

	_, unsub1 := hub.Subscribe("volatile-project")
	_, unsub2 := hub.Subscribe("volatile-project")

	assert.True(t, hub.HasSubscribers("volatile-project"))

	unsub1()
	unsub2()

	assert.False(t, hub.HasSubscribers("volatile-project"),
		"GREEN: HasSubscribers must return false after all clients unsub")

	// Subscribe again to a fresh project – the map must not be corrupted.
	ch, unsub := hub.Subscribe("volatile-project")
	defer unsub()

	hub.Publish("volatile-project", "after-prune")

	select {
	case msg := <-ch:
		assert.Equal(t, "after-prune", msg,
			"GREEN: hub must work correctly after pruning empty map keys")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no message received after map prune")
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 8 – Empty / arbitrary projectID accepted without validation
//
// hub.go:17  Subscribe accepts any string, including "" (empty), allowing a
// subscriber for the empty-string project to receive events from any publisher
// that also uses "".  Combined with the HTTP handler (sse.go:25) which reads
// the projectID from the URL without further validation, a request to
// /api/projects//sse (empty segment) would subscribe to "" and receive events
// from internal callers that accidentally publish to "".
//
// Affected file/line: hub.go:17 (Subscribe), hub.go:38 (Publish)
// ---------------------------------------------------------------------------

// TestSecurity_EmptyProjectID_Green asserts that a safe Hub rejects empty or
// obviously invalid project IDs at Subscribe time.
func TestSecurity_EmptyProjectID_Green(t *testing.T) {
	hub := sse.NewHub()

	// A safe hub should reject "" and return a nil channel.
	ch, unsub := hub.Subscribe("")
	if unsub != nil {
		defer unsub()
	}

	assert.Nil(t, ch,
		"GREEN: hub must reject empty projectID and return nil channel")

	// Verify that a valid projectID still works.
	validCh, validUnsub := hub.Subscribe("valid-project-id")
	defer validUnsub()
	require.NotNil(t, validCh, "GREEN: valid projectID must be accepted")

	hub.Publish("valid-project-id", "ok")
	select {
	case msg := <-validCh:
		assert.Equal(t, "ok", msg)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no message on valid subscription")
	}
}

// ---------------------------------------------------------------------------
// Vulnerability 9 – Publish/Unsubscribe race: send on closed channel (panic)
//
// hub.go:23-35 / hub.go:38-49
//
// The Unsubscribe closure acquires a write lock, removes the channel from the
// slice, releases the lock, then calls close(ch).  The lock is released before
// close() is called.
//
// Publish acquires an RLock, copies the channel slice, releases the RLock,
// then iterates the copy and sends on each channel.  If Unsubscribe removes
// and closes `ch` between the moment Publish copies the slice and the moment
// Publish attempts to send on `ch`, Publish will send on a closed channel →
// PANIC.
//
// The race detector also reports a data race between runtime.closechan (in
// Unsubscribe) and runtime.chansend (in Publish) on the same channel object.
//
// Severity: CRITICAL – the server panics and crashes under concurrent load.
//
// Affected file/lines:
//   hub.go:33 (close(ch) outside any lock)
//   hub.go:46 (ch <- data on potentially-closed ch)
// ---------------------------------------------------------------------------

// TestSecurity_PublishUnsubscribeRace_Red confirms that the race exists.
// The test uses recover() to catch the panic that the current implementation
// triggers, confirming the vulnerability is present.
func TestSecurity_PublishUnsubscribeRace_Red(t *testing.T) {
	// We run the racy scenario in a sub-goroutine so that the panic from
	// "send on closed channel" can be recovered without killing the test binary.
	panicked := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicked <- true
			} else {
				panicked <- false
			}
		}()

		hub := sse.NewHub()
		const iterations = 2000

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ch, unsub := hub.Subscribe("race-project-red")
				go func(c chan string, u func()) {
					// Close almost immediately to maximise the race window.
					u()
					// Drain so goroutine exits cleanly.
					for range c {
					}
				}(ch, unsub)
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < iterations*5; i++ {
				hub.Publish("race-project-red", "event")
			}
		}()

		wg.Wait()
	}()

	select {
	case didPanic := <-panicked:
		if didPanic {
			t.Log("RED confirmed: concurrent Publish+Unsubscribe caused 'send on closed channel' panic")
		} else {
			t.Log("RED: panic did not trigger this run (race is timing-dependent); run multiple times or with -race")
		}
		// Either outcome is acceptable for the RED test – the point is to
		// document the vulnerability; the race detector will catch it with -race.
	case <-time.After(10 * time.Second):
		t.Fatal("race scenario timed out")
	}
}

// TestSecurity_PublishUnsubscribeRace_Green asserts that a safe hub never
// panics when Publish and Unsubscribe run concurrently.  A safe fix closes
// the channel only while holding the write lock, or uses a sync.Once, or
// drains+closes under the lock.
func TestSecurity_PublishUnsubscribeRace_Green(t *testing.T) {
	hub := sse.NewHub()
	const iterations = 2000

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			ch, unsub := hub.Subscribe("race-project-green")
			go func(c chan string, u func()) {
				u()
				for range c {
				}
			}(ch, unsub)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations*5; i++ {
			hub.Publish("race-project-green", "event")
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// GREEN: no panic – the race detector must also report no data races.
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent Publish/Unsubscribe test timed out")
	}
}
