package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys/apikeystest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams/teamstest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/JLugagne/agach-mcp/internal/identity/outbound/pg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const testEncKey = "test-encryption-key-32-bytes-ok!"

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:17",
		tcpostgres.WithDatabase("identity_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func newTestRepos(t *testing.T) *pg.Repositories {
	t.Helper()
	pool := newTestPool(t)
	repos, err := pg.NewRepositories(context.Background(), pool, testEncKey)
	require.NoError(t, err)
	return repos
}

func TestTeamRepository_Contract(t *testing.T) {
	repos := newTestRepos(t)
	teamstest.TeamsContractTesting(t, repos.Teams)
}

func TestUserRepository_Contract(t *testing.T) {
	repos := newTestRepos(t)
	userstest.UsersContractTesting(t, repos.Users)
}

func TestAPIKeyRepository_Contract(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	// APIKeys require an existing user
	user := domain.User{
		ID:        domain.NewUserID(),
		Email:     "apikey-test@example.com",
		Role:      domain.RoleMember,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repos.Users.Create(ctx, user)
	require.NoError(t, err)

	apikeystest.APIKeysContractTesting(t, repos.APIKeys, user.ID)
}
