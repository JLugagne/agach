package e2eapi

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// testPool returns the shared pgxpool (started by the test server).
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ensureServer(t)
	require.NotNil(t, dbPool, "database pool not initialized")
	return dbPool
}

// countRows returns the number of rows in a table matching an optional WHERE clause.
func countRows(t *testing.T, pool *pgxpool.Pool, table, where string, args ...any) int {
	t.Helper()
	q := "SELECT COUNT(*) FROM " + table
	if where != "" {
		q += " WHERE " + where
	}
	var n int
	err := pool.QueryRow(context.Background(), q, args...).Scan(&n)
	require.NoError(t, err)
	return n
}

// rowExists checks whether at least one row matches.
func rowExists(t *testing.T, pool *pgxpool.Pool, table, where string, args ...any) bool {
	t.Helper()
	return countRows(t, pool, table, where, args...) > 0
}

// queryString returns a single string column value.
func queryString(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var v string
	err := pool.QueryRow(context.Background(), query, args...).Scan(&v)
	require.NoError(t, err)
	return v
}
