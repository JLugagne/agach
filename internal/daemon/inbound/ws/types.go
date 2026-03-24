package ws

import (
	"context"

	"github.com/JLugagne/agach-mcp/pkg/daemonws"
)

// HandlerFunc is the signature for WebSocket message handlers.
type HandlerFunc func(ctx context.Context, msg daemonws.Message) (daemonws.Message, error)
