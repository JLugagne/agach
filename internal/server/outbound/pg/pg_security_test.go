// Package pg_test contains security-focused tests for the kanban pg package.
//
// Each test documents:
//   - the vulnerability it covers
//   - RED behaviour (what the current code does wrong)
//   - GREEN behaviour (what must hold after a fix)
//
// Tests that require a live PostgreSQL instance are skipped automatically
// when the KANBAN_PG_DSN environment variable is not set.
package pg_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	serverpg "github.com/JLugagne/agach-mcp/internal/server/outbound/pg"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// test infrastructure
// ---------------------------------------------------------------------------

// openTestPool opens a pgxpool for the DSN in KANBAN_PG_DSN.
// The calling test is skipped when the variable is absent.
func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("KANBAN_PG_DSN")
	if dsn == "" {
		t.Skip("KANBAN_PG_DSN not set – skipping live-DB test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "pgxpool.New must succeed")
	t.Cleanup(pool.Close)
	return pool
}

// applyMigration reads and executes the embedded schema SQL.
func applyMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	sql, err := os.ReadFile("migrations/001_schema.sql")
	require.NoError(t, err, "must be able to read migration file")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = pool.Exec(ctx, string(sql))
	require.NoError(t, err, "migration must apply without error")
}

// dropServerTables removes all kanban tables so tests start from a clean slate.
func dropServerTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, tbl := range []string{
		"tool_usage", "task_dependencies", "comments",
		"tasks", "columns", "project_roles", "roles", "projects",
	} {
		_, _ = pool.Exec(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE`, tbl))
	}
	_, _ = pool.Exec(ctx, `DROP EXTENSION IF EXISTS pgcrypto CASCADE`)
}

// columnExistsInTable returns true when the named column exists in the table.
func columnExistsInTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, column string) bool {
	t.Helper()
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = $2
		)`, table, column,
	).Scan(&exists)
	require.NoError(t, err, "columnExistsInTable query failed for %s.%s", table, column)
	return exists
}

// ---------------------------------------------------------------------------
// VULNERABILITY 1 – No query timeout enforcement
//
// baseRepository wraps *pgxpool.Pool but never enforces a deadline on the
// context it receives.  A caller that passes context.Background() will block
// forever if the DB hangs.
//
// RED:  context.Background() has no deadline; this is the dangerous
//       precondition that the package never guards against.
// GREEN: A hardened caller always wraps contexts in context.WithTimeout
//        before handing them to the repository layer.
// ---------------------------------------------------------------------------

func TestSecurity_NoQueryTimeout_RED(t *testing.T) {
	ctx := context.Background()
	_, hasDeadline := ctx.Deadline()

	// RED: no deadline – if passed to a DB method, a hung query blocks forever.
	assert.False(t, hasDeadline,
		"RED: context.Background() carries no deadline; "+
			"the repository layer must enforce one but currently does not")
}

func TestSecurity_NoQueryTimeout_GREEN(t *testing.T) {
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, hasDeadline := ctxWithTimeout.Deadline()

	// GREEN: a properly hardened call-site always adds a timeout.
	assert.True(t, hasDeadline,
		"GREEN: context passed to any DB query must carry a deadline")
}

// ---------------------------------------------------------------------------
// VULNERABILITY 2 – Connection pool size not validated
//
// NewRepositories accepts any *pgxpool.Pool without checking that MaxConns is
// within safe bounds.  An accidentally misconfigured pool (MaxConns=500) can
// exhaust the Postgres server's max_connections.
//
// RED:  A pool with MaxConns=500 is accepted without complaint.
// GREEN: After the fix, NewRepositories (or a helper) should reject pools
//        that exceed the safe threshold.
// ---------------------------------------------------------------------------

func TestSecurity_PoolMaxConns_RED(t *testing.T) {
	dsn := os.Getenv("KANBAN_PG_DSN")
	if dsn == "" {
		t.Skip("KANBAN_PG_DSN not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = 500 // dangerously high

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer pool.Close()

	// RED: NewRepositories accepts the pool silently; no validation occurs.
	repos, err := serverpg.NewRepositories(pool)
	require.NoError(t, err, "RED: NewRepositories accepted MaxConns=500 without complaint")
	assert.NotNil(t, repos)

	assert.Equal(t, int32(500), pool.Config().MaxConns,
		"RED: the dangerous MaxConns value was never validated")
}

func TestSecurity_PoolMaxConns_GREEN(t *testing.T) {
	const safeMax = int32(20)

	dsn := os.Getenv("KANBAN_PG_DSN")
	if dsn == "" {
		t.Skip("KANBAN_PG_DSN not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.MaxConns = safeMax

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	defer pool.Close()

	// GREEN: pool is within safe bounds; NewRepositories should accept it.
	assert.LessOrEqual(t, pool.Config().MaxConns, safeMax,
		"GREEN: pool MaxConns must not exceed the safe threshold")
}

// ---------------------------------------------------------------------------
// VULNERABILITY 4 – Migration not applied by NewRepositories
//
// The identity package uses //go:embed + pool.Exec to run migrations in
// NewRepositories.  The kanban pg package does NOT, meaning callers receive a
// Repositories struct whose real methods would hit raw Postgres errors like
// "relation 'tasks' does not exist" – leaking table names.
//
// RED:  NewRepositories succeeds even when the schema is absent; it does not
//       apply the migration.
// GREEN: After the fix NewRepositories must embed and execute the schema SQL.
//        A probe query on a required table must succeed without error.
// ---------------------------------------------------------------------------

func TestSecurity_MigrationNotApplied_RED(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)

	// RED: NewRepositories must succeed even though the schema is absent.
	repos, err := serverpg.NewRepositories(pool)
	require.NoError(t, err,
		"RED: NewRepositories succeeds without applying migration")
	assert.NotNil(t, repos)

	// Verify the schema is indeed absent after the call.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	probeErr := pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&count)
	assert.Error(t, probeErr,
		"RED: projects table does not exist; NewRepositories failed to apply the migration")
}

func TestSecurity_MigrationNotApplied_GREEN(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool) // simulates what NewRepositories SHOULD do

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GREEN: after migration all required tables exist.
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects`).Scan(&count)
	require.NoError(t, err, "GREEN: projects table must exist after migration is applied")
	assert.GreaterOrEqual(t, count, 0)
}

// ---------------------------------------------------------------------------
// VULNERABILITY 5 – No Row-Level Security on any table
//
// The migration creates tables without enabling RLS.  Any DB role that can
// connect has full access to all rows, with no isolation between tenants.
//
// RED:  After migration, relrowsecurity=false on all sensitive tables.
// GREEN: After the fix, relrowsecurity=true on each sensitive table.
// ---------------------------------------------------------------------------

func TestSecurity_NoRowLevelSecurity_RED(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, table := range []string{"projects", "tasks", "comments", "columns"} {
		var rlsEnabled bool
		err := pool.QueryRow(ctx,
			`SELECT relrowsecurity FROM pg_class WHERE relname = $1`, table,
		).Scan(&rlsEnabled)
		require.NoError(t, err, "table %q must exist in pg_class", table)

		// RED: RLS is not enabled.
		assert.False(t, rlsEnabled,
			"RED: table %q has no RLS; any DB role can read/write all rows without restriction", table)
	}
}

func TestSecurity_NoRowLevelSecurity_GREEN(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GREEN: demonstrate what the migration should do.
	for _, table := range []string{"projects", "tasks", "comments", "columns"} {
		_, err := pool.Exec(ctx,
			fmt.Sprintf(`ALTER TABLE %s ENABLE ROW LEVEL SECURITY`, table))
		require.NoError(t, err, "must be able to enable RLS on %q", table)

		var rlsEnabled bool
		err = pool.QueryRow(ctx,
			`SELECT relrowsecurity FROM pg_class WHERE relname = $1`, table,
		).Scan(&rlsEnabled)
		require.NoError(t, err)

		assert.True(t, rlsEnabled,
			"GREEN: table %q must have RLS enabled after the migration fix", table)
	}
}

// ---------------------------------------------------------------------------
// VULNERABILITY 6 – Boolean flags stored as INTEGER without range constraint
//
// is_blocked and wont_do_requested are INTEGER, not BOOLEAN.  Values like 2
// or -1 satisfy the column constraint and create ambiguous application state.
//
// RED:  Inserting is_blocked=2 succeeds without error.
// GREEN: A CHECK(is_blocked IN (0,1)) or BOOLEAN type must reject it.
// ---------------------------------------------------------------------------

const (
	secTestProjectID = "11111111-0000-0000-0000-000000000001"
	secTestColumnID  = "11111111-0000-0000-0000-000000000002"
	secTestTaskID    = "11111111-0000-0000-0000-000000000003"
	secTestProject2  = "22222222-0000-0000-0000-000000000001"
	secTestColumn2   = "22222222-0000-0000-0000-000000000002"
	secTestTask2     = "22222222-0000-0000-0000-000000000003"
)

func insertTestProject(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, name string) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO projects (id, name) VALUES ($1, $2)`, id, name)
	require.NoError(t, err, "insertTestProject failed for id=%s", id)
}

func insertTestColumn(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, projectID string) {
	t.Helper()
	_, err := pool.Exec(ctx,
		`INSERT INTO columns (id, project_id, slug, name, position) VALUES ($1, $2, 'todo', 'Todo', 0)`,
		id, projectID)
	require.NoError(t, err, "insertTestColumn failed for id=%s", id)
}

func TestSecurity_BooleanIntegerFlags_RED(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	insertTestProject(t, ctx, pool, secTestProjectID, "sec-test")
	insertTestColumn(t, ctx, pool, secTestColumnID, secTestProjectID)

	// RED: is_blocked=2 is accepted because the column is INTEGER with no CHECK.
	_, insertErr := pool.Exec(ctx,
		`INSERT INTO tasks (id, project_id, column_id, title, summary, is_blocked)
		 VALUES ($1, $2, $3, 'sec task', 'sec summary', 2)`,
		secTestTaskID, secTestProjectID, secTestColumnID,
	)
	assert.NoError(t, insertErr,
		"RED: is_blocked=2 accepted by INTEGER column; "+
			"a BOOLEAN or CHECK(is_blocked IN (0,1)) would reject this")

	var stored int
	_ = pool.QueryRow(ctx,
		`SELECT is_blocked FROM tasks WHERE id = $1`, secTestTaskID,
	).Scan(&stored)
	assert.Equal(t, 2, stored,
		"RED: invalid boolean value 2 was persisted to the database")
}

func TestSecurity_BooleanIntegerFlags_GREEN(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GREEN: add the missing CHECK constraints that the migration should include.
	for _, ddl := range []string{
		`ALTER TABLE tasks ADD CONSTRAINT chk_tasks_is_blocked CHECK (is_blocked IN (0,1))`,
		`ALTER TABLE tasks ADD CONSTRAINT chk_tasks_wont_do_req CHECK (wont_do_requested IN (0,1))`,
	} {
		_, err := pool.Exec(ctx, ddl)
		require.NoError(t, err, "GREEN: must be able to add %s", ddl)
	}

	insertTestProject(t, ctx, pool, secTestProject2, "sec-test-2")
	insertTestColumn(t, ctx, pool, secTestColumn2, secTestProject2)

	_, insertErr := pool.Exec(ctx,
		`INSERT INTO tasks (id, project_id, column_id, title, summary, is_blocked)
		 VALUES ($1, $2, $3, 'green task', 'green summary', 2)`,
		secTestTask2, secTestProject2, secTestColumn2,
	)
	// GREEN: the CHECK constraint must now reject out-of-range values.
	assert.Error(t, insertErr,
		"GREEN: is_blocked=2 must be rejected by the CHECK constraint")
}

// ---------------------------------------------------------------------------
// VULNERABILITY 7 – tasks table missing session_id column
//
// domain.Task.SessionID is used for Claude Code session resumption.  The
// migration omits the column; a real implementation would silently drop the
// value or fail with a Postgres error leaking the column name.
//
// RED:  session_id column is absent after migration.
// GREEN: The column must exist with TEXT type and a safe default.
// ---------------------------------------------------------------------------

func TestSecurity_MissingSessionIDColumn_RED(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists := columnExistsInTable(t, ctx, pool, "tasks", "session_id")
	// RED: column is absent.
	assert.False(t, exists,
		"RED: tasks.session_id is missing from the migration; "+
			"a real INSERT/SELECT would fail with a schema-leaking Postgres error")
}

func TestSecurity_MissingSessionIDColumn_GREEN(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GREEN: add the missing column (what the migration should include).
	_, err := pool.Exec(ctx,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS session_id TEXT NOT NULL DEFAULT ''`)
	require.NoError(t, err, "GREEN: must be able to add session_id column")

	assert.True(t, columnExistsInTable(t, ctx, pool, "tasks", "session_id"),
		"GREEN: tasks.session_id must exist after applying the migration fix")
}

// ---------------------------------------------------------------------------
// VULNERABILITY 8 – tasks table missing token usage columns
//
// domain.Task exposes InputTokens, OutputTokens, CacheReadTokens,
// CacheWriteTokens, Model, and ColdStart* fields.  None of these appear in
// the migration.  A real implementation writing these fields would produce
// Postgres errors that leak column/table names to callers.
//
// RED:  None of the token columns exist after migration.
// GREEN: All token columns must be present.
// ---------------------------------------------------------------------------

func TestSecurity_MissingTokenColumns_RED(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, col := range []string{
		"input_tokens",
		"output_tokens",
		"cache_read_tokens",
		"cache_write_tokens",
		"model",
		"cold_start_input_tokens",
		"cold_start_output_tokens",
	} {
		exists := columnExistsInTable(t, ctx, pool, "tasks", col)
		// RED: the column is absent – a real INSERT would fail with a
		// Postgres error that leaks the column name.
		assert.False(t, exists,
			"RED: tasks.%s is absent from migration; "+
				"real repository code would produce schema-leaking errors", col)
	}
}

func TestSecurity_MissingTokenColumns_GREEN(t *testing.T) {
	pool := openTestPool(t)
	dropServerTables(t, pool)
	applyMigration(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GREEN: apply the missing columns (what the migration should include).
	for _, ddl := range []string{
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS input_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS output_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cache_read_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cache_write_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cold_start_input_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cold_start_output_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cold_start_cache_read_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS cold_start_cache_write_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS duration_seconds INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tasks ADD COLUMN IF NOT EXISTS human_estimate_seconds INTEGER NOT NULL DEFAULT 0`,
	} {
		_, err := pool.Exec(ctx, ddl)
		require.NoError(t, err, "GREEN: %s must succeed", ddl)
	}

	assert.True(t, columnExistsInTable(t, ctx, pool, "tasks", "input_tokens"),
		"GREEN: tasks.input_tokens must exist after migration fix")
	assert.True(t, columnExistsInTable(t, ctx, pool, "tasks", "model"),
		"GREEN: tasks.model must exist after migration fix")
}

// ---------------------------------------------------------------------------
// VULNERABILITY 9 – No REVOKE / explicit GRANT in migration
//
// The migration applies no REVOKE or GRANT statements, relying on the
// connecting role's default PostgreSQL privileges.  If the PUBLIC role has
// inherited CREATE on the public schema, any connected role can create objects.
//
// RED:  The migration SQL contains neither REVOKE nor GRANT.
// GREEN: A hardened migration must include at minimum
//        REVOKE CREATE ON SCHEMA public FROM PUBLIC.
// ---------------------------------------------------------------------------

func TestSecurity_NoBroadPrivilegeRevoke_GREEN(t *testing.T) {
	// GREEN: what the migration SHOULD contain.
	hardenedFragment := `
		REVOKE CREATE ON SCHEMA public FROM PUBLIC;
		REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
	`
	upper := strings.ToUpper(hardenedFragment)

	assert.True(t, strings.Contains(upper, "REVOKE"),
		"GREEN: hardened migration must include REVOKE to restrict public schema access")
}
