package e2eapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity"
	identityservice "github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/JLugagne/agach-mcp/internal/pkg/middleware"
	"github.com/JLugagne/agach-mcp/internal/server"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// ---------- singleton test server ------------------------------------------

var (
	setupOnce sync.Once
	serverURL string // set once by startTestServer
	dbPool    *pgxpool.Pool
)

// ensureServer starts the in-process server (postgres via testcontainer) once
// for the whole test binary. Subsequent calls are no-ops.
func ensureServer(t *testing.T) {
	t.Helper()
	setupOnce.Do(func() {
		startTestServer(t)
	})
	require.NotEmpty(t, serverURL, "test server failed to start")
}

func startTestServer(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	jwtSecret := []byte("e2e-test-secret-at-least-32-bytes!")

	// --- Postgres via testcontainers ---
	connStr := os.Getenv("E2E_DATABASE_URL")
	if connStr == "" {
		container, err := tcpostgres.Run(ctx,
			"postgres:17",
			tcpostgres.WithDatabase("agach_e2e"),
			tcpostgres.WithUsername("agach"),
			tcpostgres.WithPassword("agach"),
			tcpostgres.BasicWaitStrategies(),
		)
		require.NoError(t, err)

		cs, err := container.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, err)
		connStr = cs
	}

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	dbPool = pool

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// --- Identity system (auth, teams, nodes) ---
	identitySystem, err := identity.Init(ctx, identity.Config{
		Logger:    logger,
		JWTSecret: jwtSecret,
	}, pool)
	require.NoError(t, err)

	// --- HTTP router ---
	httpRouter := mux.NewRouter()

	// Health check
	httpRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}).Methods("GET")

	ctrl := controller.NewController(logger)
	identitySystem.RegisterRoutes(httpRouter, ctrl)

	// Auth middleware
	requireAuth := middleware.NewRequireAuth(&authAdapter{q: identitySystem.AuthQueries})

	dataDir, dirErr := os.MkdirTemp("", "agach-e2e-data-*")
	require.NoError(t, dirErr)

	serverRouter := httpRouter.PathPrefix("").Subrouter()
	serverRouter.Use(requireAuth)
	_, err = server.InitHTTP(server.Config{
		Pool:        pool,
		Logger:      logger,
		AuthQueries: identitySystem.AuthQueries,
		WSRouter:    httpRouter,
		DataDir:     dataDir,
	}, serverRouter)
	require.NoError(t, err)

	// --- Start httptest server ---
	ts := httptest.NewServer(httpRouter)
	serverURL = ts.URL
}

// authAdapter wraps identity.AuthQueries into middleware.AuthValidator.
type authAdapter struct {
	q identityservice.AuthQueries
}

func (a *authAdapter) ValidateJWT(ctx context.Context, token string) (any, error) {
	actor, err := a.q.ValidateJWT(ctx, token)
	if err == nil {
		return actor, nil
	}
	return a.q.ValidateDaemonJWT(ctx, token)
}
