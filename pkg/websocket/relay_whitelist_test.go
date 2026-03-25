package websocket

import (
	"testing"

	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/stretchr/testify/assert"
)

// TestRelayWhitelist_ChatMessageTypes verifies that all chat message types defined
// in the daemonws package are present in the relay whitelist so that they are
// forwarded between daemon and browser clients.
func TestRelayWhitelist_ChatMessageTypes(t *testing.T) {
	requiredChatTypes := []string{
		daemonws.TypeChatStart,
		daemonws.TypeChatMessage,
		daemonws.TypeChatUserMsg,
		daemonws.TypeChatEnd,
		daemonws.TypeChatError,
		daemonws.TypeChatStats,
	}

	for _, msgType := range requiredChatTypes {
		assert.True(t, relayMessageTypes[msgType],
			"relay whitelist should contain chat message type %q", msgType)
	}
}
