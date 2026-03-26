package sidecar

import (
	"context"
	"fmt"
	"os"

	"github.com/JLugagne/agach-mcp/internal/sidecar/app"
	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
	mcpserver "github.com/JLugagne/agach-mcp/internal/sidecar/inbound/mcp"
	"github.com/JLugagne/agach-mcp/internal/sidecar/outbound/proxy"
)

// Run initializes and runs the sidecar MCP server.
func Run(mode string) error {
	socketPath := os.Getenv("AGACH_PROXY")
	if socketPath == "" {
		return fmt.Errorf("AGACH_PROXY environment variable is required")
	}

	apiKey := os.Getenv("AGACH_PROXY_KEY")
	if apiKey == "" {
		return fmt.Errorf("AGACH_PROXY_KEY environment variable is required")
	}

	featureID := os.Getenv("AGACH_FEATURE_ID")

	// Outbound: HTTP client over Unix socket
	client := proxy.New(socketPath, apiKey)

	// App layer
	application := app.New(client, featureID)

	// Inbound: MCP server
	server := mcpserver.NewServer(application, domain.Mode(mode))

	return server.Run(context.Background())
}
