package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity"
	service "github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/JLugagne/agach-mcp/internal/server"
	"github.com/JLugagne/agach-mcp/internal/server/ux"
	"github.com/JLugagne/agach-mcp/resources"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", getEnv("AGACH_CONFIG", "agach-server.yml"), "Path to server config file")
	initConfig := flag.Bool("init", false, "Create a default config file and exit")
	flag.Parse()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	if *initConfig {
		if err := writeDefaultConfig(*configPath); err != nil {
			logger.WithError(err).Fatal("Failed to create config file")
		}
		logger.WithField("path", *configPath).Info("Config file created")
		return
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load server config")
	}

	jwtSecret := []byte(getEnv("AGACH_JWT_SECRET", ""))
	if len(jwtSecret) < 64 {
		logger.Fatal("AGACH_JWT_SECRET must be at least 64 bytes")
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

	runHTTP(logger, pool, cfg, jwtSecret)
}

func runHTTP(logger *logrus.Logger, pool *pgxpool.Pool, cfg *serverConfig, jwtSecret []byte) {
	httpHost := getEnv("AGACH_HOST", "127.0.0.1")
	httpPort := getEnv("AGACH_PORT", "8322")

	logger.WithField("httpAddr", httpHost+":"+httpPort).Info("Starting server")

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
		Logger:                 logger,
		JWTSecret:              jwtSecret,
		SSO:                    cfg.SSO,
		DaemonJWTTTL:           cfg.DaemonJWTTTL,
		AuthRateLimitPerSecond: cfg.AuthRateLimitPerSecond,
		AuthRateLimitBurst:     cfg.AuthRateLimitBurst,
	}, pool)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize identity system")
	}
	identitySystem.RegisterRoutes(httpRouter, ctrl)

	// Auth middleware for protected routes
	requireAuth := middleware.NewRequireAuth(&authValidatorAdapter{q: identitySystem.AuthQueries})

	// Compute resource manifest from embedded binaries
	resourceManifest := server.ComputeManifest(resources.FS, logger)

	// Initialize server HTTP system under auth middleware
	serverRouter := httpRouter.PathPrefix("").Subrouter()
	serverRouter.Use(requireAuth)
	if _, err := server.InitHTTP(server.Config{
		Pool:             pool,
		Logger:           logger,
		AuthQueries:      identitySystem.AuthQueries,
		ResourceManifest: resourceManifest,
	}, serverRouter); err != nil {
		logger.WithError(err).Fatal("Failed to initialize HTTP server")
	}

	// Serve embedded frontend SPA
	distFS, err := fs.Sub(ux.DistFS, "dist")
	if err != nil {
		logger.WithError(err).Fatal("Failed to create sub filesystem for frontend")
	}
	spa := &spaHandler{staticFS: http.FileServer(http.FS(distFS)), fs: distFS}
	httpRouter.PathPrefix("/").Handler(spa)

	// Create HTTP server
	httpSrv := &http.Server{
		Addr:         httpHost + ":" + httpPort,
		Handler:      middleware.RequestLogger(logger)(httpRouter),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server
	go func() {
		logger.WithField("addr", httpSrv.Addr).Info("HTTP server listening")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("HTTP server forced to shutdown")
	}

	logger.Info("Server exited gracefully")
}

type spaHandler struct {
	staticFS http.Handler
	fs       fs.FS
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	filePath := strings.TrimPrefix(urlPath, "/")
	if filePath == "" || filePath == "." {
		filePath = "index.html"
	}

	// Check if file exists in embedded FS
	if _, err := fs.Stat(h.fs, filePath); err == nil {
		ext := filepath.Ext(filePath)
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

// authValidatorAdapter wraps identity's AuthQueries (which returns domain.Actor)
// into middleware.AuthValidator (which returns any), so that pkg/middleware
// does not import from internal/.
type authValidatorAdapter struct {
	q service.AuthQueries
}

func (a *authValidatorAdapter) ValidateJWT(ctx context.Context, token string) (any, error) {
	return a.q.ValidateJWT(ctx, token)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
