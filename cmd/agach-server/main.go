package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity"
	"github.com/JLugagne/agach-mcp/internal/kanban"
	"github.com/JLugagne/agach-mcp/internal/svrconfig"
	"github.com/JLugagne/agach-mcp/pkg/controller"
	"github.com/JLugagne/agach-mcp/pkg/middleware"
	"github.com/JLugagne/agach-mcp/ux"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	mcpMode := flag.Bool("mcp", false, "Run as MCP server over stdio (for Claude Code integration)")
	configPath := flag.String("config", getEnv("AGACH_CONFIG", "agach-server.yml"), "Path to server config file")
	flag.Parse()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	cfg, err := svrconfig.Load(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load server config")
	}

	jwtSecret := []byte(getEnv("AGACH_JWT_SECRET", ""))
	if len(jwtSecret) < 32 {
		logger.Fatal("AGACH_JWT_SECRET must be at least 32 bytes")
	}

	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer pool.Close()

	if *mcpMode {
		runMCP(logger, pool)
		return
	}

	runHTTP(logger, pool, cfg, jwtSecret)
}

func runMCP(logger *logrus.Logger, pool *pgxpool.Pool) {
	// In MCP stdio mode, redirect logs to stderr so they don't interfere with protocol
	logger.SetOutput(os.Stderr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	if err := kanban.RunMCPStdio(ctx, kanban.Config{
		Pool:   pool,
		Logger: logger,
	}); err != nil {
		logger.WithError(err).Error("MCP server exited with error")
		os.Exit(1)
	}
}

func runHTTP(logger *logrus.Logger, pool *pgxpool.Pool, cfg *svrconfig.Config, jwtSecret []byte) {
	httpHost := getEnv("AGACH_HOST", "127.0.0.1")
	httpPort := getEnv("AGACH_PORT", "8322")
	mcpHost := getEnv("AGACH_MCP_HOST", "127.0.0.1")
	mcpPort := getEnv("AGACH_MCP_PORT", "8323")

	logger.WithFields(logrus.Fields{
		"httpAddr": httpHost + ":" + httpPort,
		"mcpAddr":  mcpHost + ":" + mcpPort,
	}).Info("Starting Kanban Server")

	// Shared controller and router
	ctrl := controller.NewController(logger)
	httpRouter := mux.NewRouter()

	// Health check (unauthenticated)
	httpRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}).Methods("GET")

	// Initialize identity system (auth + SSO)
	identitySystem, err := identity.Init(context.Background(), identity.Config{
		Logger:    logger,
		JWTSecret: jwtSecret,
		SSO:       cfg.SSO,
	}, pool)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize identity system")
	}
	identitySystem.RegisterRoutes(httpRouter, ctrl)

	// Auth middleware for protected routes
	requireAuth := middleware.NewRequireAuth(identitySystem.AuthQueries)

	// Initialize Kanban HTTP system under auth middleware
	kanbanRouter := httpRouter.PathPrefix("").Subrouter()
	kanbanRouter.Use(requireAuth)
	hub, err := kanban.InitKanbanHTTP(kanban.Config{
		Pool:   pool,
		Logger: logger,
	}, kanbanRouter)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kanban HTTP system")
	}

	// Serve embedded frontend SPA
	distFS, err := fs.Sub(ux.DistFS, "dist")
	if err != nil {
		logger.WithError(err).Fatal("Failed to create sub filesystem for frontend")
	}
	spa := &spaHandler{staticFS: http.FileServer(http.FS(distFS)), fs: distFS}
	httpRouter.PathPrefix("/").Handler(spa)

	// Create MCP router and initialize MCP SSE server (shares the HTTP hub)
	mcpRouter := mux.NewRouter()
	if err := kanban.InitKanbanMCP(kanban.Config{
		Pool:   pool,
		Logger: logger,
	}, mcpRouter, hub); err != nil {
		logger.WithError(err).Fatal("Failed to initialize Kanban MCP system")
	}

	// Create HTTP server
	httpSrv := &http.Server{
		Addr:         httpHost + ":" + httpPort,
		Handler:      httpRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create MCP server
	// No WriteTimeout: SSE/streamable transports use long-lived connections
	// that must stay open for the entire session.
	mcpSrv := &http.Server{
		Addr:        mcpHost + ":" + mcpPort,
		Handler:     mcpRouter,
		ReadTimeout: 30 * time.Second,
		IdleTimeout: 120 * time.Second,
	}

	// Start HTTP server
	go func() {
		logger.WithField("addr", httpSrv.Addr).Info("HTTP server listening")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Start MCP server
	go func() {
		logger.WithField("addr", mcpSrv.Addr).Info("MCP SSE server listening")
		if err := mcpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("MCP server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("HTTP server forced to shutdown")
	}

	if err := mcpSrv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("MCP server forced to shutdown")
	}

	logger.Info("Servers exited gracefully")
}

type spaHandler struct {
	staticFS http.Handler
	fs       fs.FS
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if file exists in embedded FS
	if _, err := fs.Stat(h.fs, path); err == nil {
		ext := filepath.Ext(path)
		switch ext {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".woff":
			w.Header().Set("Content-Type", "font/woff")
		}
		h.staticFS.ServeHTTP(w, r)
		return
	}

	// SPA fallback: serve index.html for all non-file routes
	r.URL.Path = "/"
	h.staticFS.ServeHTTP(w, r)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
