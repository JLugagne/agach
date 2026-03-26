package mcp

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/sidecar/app"
	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server with tool registrations.
type Server struct {
	server *mcp.Server
}

// NewServer creates an MCP server and registers tools based on the mode.
func NewServer(application *app.App, mode domain.Mode) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "agach-sidecar",
			Version: "1.0.0",
		},
		nil,
	)

	// Always available in all modes
	registerBulkCreateTasks(s, application)
	registerBulkAddDependencies(s, application)

	// Only available in default mode (not PM)
	if mode != domain.ModePM {
		registerCompleteTask(s, application)
		registerRunTask(s, application)
		registerBlockTask(s, application)
		registerWontDoTask(s, application)
		registerFeatureChangelogs(s, application)
	}

	return &Server{server: s}
}

// Run starts the MCP server on stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
