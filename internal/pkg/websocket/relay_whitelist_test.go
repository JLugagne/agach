package websocket

import (
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestRelayWhitelist_ChatMessageTypes verifies that all chat message types defined
// in the daemonws package are present in the relay handler registry so that they are
// forwarded between daemon and browser clients.
func TestRelayWhitelist_ChatMessageTypes(t *testing.T) {
	hub := NewHub(logrus.New())

	// Register the standard relay types (same as init.go does in production)
	relay := hub.NewRelayHandler()
	for _, msgType := range []string{
		daemonws.TypeChatStart,
		daemonws.TypeChatMessage,
		daemonws.TypeChatUserMsg,
		daemonws.TypeChatEnd,
		daemonws.TypeChatError,
		daemonws.TypeChatStats,
	} {
		hub.RegisterHandler(msgType, relay)
	}

	for _, msgType := range []string{
		daemonws.TypeChatStart,
		daemonws.TypeChatMessage,
		daemonws.TypeChatUserMsg,
		daemonws.TypeChatEnd,
		daemonws.TypeChatError,
		daemonws.TypeChatStats,
	} {
		_, ok := hub.handlers[msgType]
		assert.True(t, ok,
			"relay handler registry should contain chat message type %q", msgType)
	}
}
