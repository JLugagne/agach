package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/daemon/outbound/sqlite"
	"github.com/stretchr/testify/require"
)

func TestRunMigrations_CreatesBuildTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := sqlite.NewDB(dbPath)
	require.NoError(t, err)
	require.NotNil(t, db, "NewDB must return a non-nil database")
	t.Cleanup(func() { db.Close() })

	err = sqlite.RunMigrations(db)
	require.NoError(t, err)

	// Verify builds table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='builds'").Scan(&tableName)
	require.NoError(t, err, "builds table must exist after migration")
	require.Equal(t, "builds", tableName)

	// Verify expected columns exist
	rows, err := db.Query("PRAGMA table_info(builds)")
	require.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue *string
		err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		require.NoError(t, err)
		columns[name] = true
	}
	require.NoError(t, rows.Err())

	expectedColumns := []string{"id", "dockerfile_slug", "version", "image_hash", "image_size", "status", "build_log", "created_at", "completed_at"}
	for _, col := range expectedColumns {
		require.True(t, columns[col], "builds table must have column %q", col)
	}
}
