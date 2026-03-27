package app

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestChatManager creates a ChatManager suitable for unit tests.
// It uses nil for GitService / ProjectClient / UploadClient so that any
// code path that touches them will panic — keeping tests honest about
// exactly what they exercise.
func newTestChatManager(ttl time.Duration, sendFn func(daemonws.Message)) *ChatManager {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	m := &ChatManager{
		sessions:    make(map[string]*ChatSession),
		logger:      logger,
		sendMessage: sendFn,
		stopCh:      make(chan struct{}),
		ttl:         ttl,
	}
	return m
}

// addMockSession injects a pre-built ChatSession into the manager without
// starting a real Claude process.
func addMockSession(m *ChatManager, session *ChatSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
}

// collectMessages drains messages sent to the channel until the deadline.
func collectMessages(ch <-chan daemonws.Message, count int, timeout time.Duration) []daemonws.Message {
	var msgs []daemonws.Message
	deadline := time.After(timeout)
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return msgs
			}
			msgs = append(msgs, msg)
			if len(msgs) >= count {
				return msgs
			}
		case <-deadline:
			return msgs
		}
	}
}

// --------------------------------------------------------------------------
// GetSession
// --------------------------------------------------------------------------

func TestChatManager_GetSession_Exists(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	session := &ChatSession{
		ID:           "sess-001",
		FeatureID:    "feat-1",
		ProjectID:    "proj-1",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	addMockSession(m, session)

	got := m.GetSession("sess-001")
	require.NotNil(t, got)
	assert.Equal(t, "sess-001", got.ID)
}

func TestChatManager_GetSession_NotExists(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	got := m.GetSession("nonexistent")
	assert.Nil(t, got)
}

// --------------------------------------------------------------------------
// SendMessage to non-existent session
// --------------------------------------------------------------------------

func TestChatManager_SendMessage_NotFound(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	err := m.SendMessage("missing-session", "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// --------------------------------------------------------------------------
// RefreshActivity
// --------------------------------------------------------------------------

func TestChatManager_RefreshActivity_Found(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	before := time.Now().Add(-time.Minute)
	session := &ChatSession{
		ID:           "sess-refresh",
		LastActivity: before,
		StartedAt:    before,
	}
	addMockSession(m, session)

	err := m.RefreshActivity("sess-refresh")
	require.NoError(t, err)

	got := m.GetSession("sess-refresh")
	require.NotNil(t, got)
	assert.True(t, got.LastActivity.After(before), "LastActivity should have been updated")
}

func TestChatManager_RefreshActivity_NotFound(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	err := m.RefreshActivity("ghost-session")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// --------------------------------------------------------------------------
// UpdateActivity
// --------------------------------------------------------------------------

func TestChatManager_UpdateActivity(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	before := time.Now().Add(-5 * time.Minute)
	session := &ChatSession{
		ID:           "sess-upd",
		LastActivity: before,
		StartedAt:    before,
	}
	addMockSession(m, session)

	m.UpdateActivity("sess-upd")

	got := m.GetSession("sess-upd")
	require.NotNil(t, got)
	assert.True(t, got.LastActivity.After(before))
}

// --------------------------------------------------------------------------
// TTL expiry
// --------------------------------------------------------------------------

func TestChatManager_TTL_ExpiresIdleSession(t *testing.T) {
	var mu sync.Mutex
	var sent []daemonws.Message

	sendFn := func(msg daemonws.Message) {
		mu.Lock()
		sent = append(sent, msg)
		mu.Unlock()
	}

	// Very short TTL for testing.
	m := newTestChatManager(50*time.Millisecond, sendFn)

	session := &ChatSession{
		ID:           "sess-ttl",
		FeatureID:    "feat-ttl",
		ProjectID:    "proj-ttl",
		StartedAt:    time.Now().Add(-time.Hour),
		LastActivity: time.Now().Add(-time.Hour), // well past TTL
		jsonlPath:    "",                         // no file to upload
	}
	addMockSession(m, session)

	// Manually invoke checkTTL instead of waiting for the ticker goroutine.
	m.checkTTL()

	// The session should have been removed.
	got := m.GetSession("sess-ttl")
	assert.Nil(t, got, "session should have been removed after TTL expiry")

	// A chat.end message should have been sent.
	mu.Lock()
	msgs := append([]daemonws.Message(nil), sent...)
	mu.Unlock()

	require.NotEmpty(t, msgs, "expected at least one message after TTL expiry")
	found := false
	for _, msg := range msgs {
		if msg.Type == daemonws.TypeChatEnd {
			var ev daemonws.ChatEndEvent
			require.NoError(t, json.Unmarshal(msg.Payload, &ev))
			assert.Equal(t, "sess-ttl", ev.SessionID)
			assert.Equal(t, "ttl_expired", ev.Reason)
			found = true
		}
	}
	assert.True(t, found, "expected a chat.end message with reason ttl_expired")
}

// --------------------------------------------------------------------------
// EndSession removes the session and emits chat.end
// --------------------------------------------------------------------------

func TestChatManager_EndSession_RemovesAndBroadcasts(t *testing.T) {
	var mu sync.Mutex
	var sent []daemonws.Message

	sendFn := func(msg daemonws.Message) {
		mu.Lock()
		sent = append(sent, msg)
		mu.Unlock()
	}

	m := newTestChatManager(30*time.Minute, sendFn)

	session := &ChatSession{
		ID:           "sess-end",
		FeatureID:    "feat-end",
		ProjectID:    "proj-end",
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	addMockSession(m, session)

	m.EndSession("sess-end", "user_requested")

	assert.Nil(t, m.GetSession("sess-end"), "session should have been removed")

	mu.Lock()
	msgs := append([]daemonws.Message(nil), sent...)
	mu.Unlock()

	require.NotEmpty(t, msgs)
	var ev daemonws.ChatEndEvent
	require.NoError(t, json.Unmarshal(msgs[len(msgs)-1].Payload, &ev))
	assert.Equal(t, "sess-end", ev.SessionID)
	assert.Equal(t, "user_requested", ev.Reason)
}

// --------------------------------------------------------------------------
// EndSession on unknown session is a no-op
// --------------------------------------------------------------------------

func TestChatManager_EndSession_Unknown_NoOp(t *testing.T) {
	ch := make(chan daemonws.Message, 16)
	m := newTestChatManager(30*time.Minute, func(msg daemonws.Message) { ch <- msg })

	// Must not panic.
	m.EndSession("not-a-real-session", "test")
	assert.Nil(t, m.GetSession("not-a-real-session"))
}
