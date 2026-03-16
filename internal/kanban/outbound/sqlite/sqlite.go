package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_global.sql
var globalMigration string

//go:embed migrations/002_project.sql
var projectMigration string

//go:embed migrations/003_seed_roles.sql
var seedRolesMigration string

//go:embed migrations/004_task_seen_at.sql
var taskSeenAtMigration string

//go:embed migrations/005_security_officer_role.sql
var securityOfficerMigration string

//go:embed migrations/006_task_token_usage.sql
var taskTokenUsageMigration string

//go:embed migrations/007_task_fts.sql
var taskFTSMigration string

//go:embed migrations/008_project_manager_role.sql
var projectManagerRoleMigration string

//go:embed migrations/009_global_indexes.sql
var globalIndexesMigration string

//go:embed migrations/010_project_indexes.sql
var projectIndexesMigration string

//go:embed migrations/011_tool_usage.sql
var toolUsageMigration string

// baseRepository contains shared database connections
type baseRepository struct {
	globalDB   *sql.DB
	dataDir    string
	projectDBs map[domain.ProjectID]*sql.DB
	mu         sync.Mutex
}

// Repositories holds all repository implementations
type Repositories struct {
	Projects     *ProjectRepository
	Roles        *RoleRepository
	Tasks        *TaskRepository
	Columns      *ColumnRepository
	Comments     *CommentRepository
	Dependencies *DependencyRepository
	ToolUsage    *ToolUsageRepository
	base         *baseRepository
}

// ProjectRepository implements projects.ProjectRepository
type ProjectRepository struct {
	*baseRepository
}

// RoleRepository implements roles.RoleRepository
type RoleRepository struct {
	*baseRepository
}

// TaskRepository implements tasks.TaskRepository
type TaskRepository struct {
	*baseRepository
}

// ColumnRepository implements columns.ColumnRepository
type ColumnRepository struct {
	*baseRepository
}

// CommentRepository implements comments.CommentRepository
type CommentRepository struct {
	*baseRepository
}

// DependencyRepository implements dependencies.DependencyRepository
type DependencyRepository struct {
	*baseRepository
}

// DefaultDataDir returns the default data directory: os.UserConfigDir()/agach-mcp/
func DefaultDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(configDir, "agach-mcp"), nil
}

// ProjectDBName returns the database filename for a project ID.
func ProjectDBName(projectID domain.ProjectID) string {
	return string(projectID) + ".db"
}

// NewRepositories creates all repository implementations.
// If dataDir is empty, it defaults to os.UserConfigDir()/agach-mcp/.
func NewRepositories(dataDir string) (*Repositories, error) {
	if dataDir == "" {
		var err error
		dataDir, err = DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	globalDBPath := filepath.Join(dataDir, "kanban.db")
	globalDB, err := openDB(globalDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open global database: %w", err)
	}

	// Run global migration
	if err := runMigration(globalDB, globalMigration); err != nil {
		globalDB.Close()
		return nil, fmt.Errorf("failed to run global migration: %w", err)
	}

	// Seed default roles (INSERT OR IGNORE — safe to re-run)
	if err := runMigration(globalDB, seedRolesMigration); err != nil {
		globalDB.Close()
		return nil, fmt.Errorf("failed to seed default roles: %w", err)
	}

	// Add security officer role and update existing roles with security awareness
	if err := runMigration(globalDB, securityOfficerMigration); err != nil {
		globalDB.Close()
		return nil, fmt.Errorf("failed to run security officer migration: %w", err)
	}

	// Add project manager role
	if err := runMigration(globalDB, projectManagerRoleMigration); err != nil {
		globalDB.Close()
		return nil, fmt.Errorf("failed to run project manager role migration: %w", err)
	}

	// Add global composite indexes
	if err := runMigration(globalDB, globalIndexesMigration); err != nil {
		globalDB.Close()
		return nil, fmt.Errorf("failed to run global indexes migration: %w", err)
	}

	base := &baseRepository{
		globalDB:   globalDB,
		dataDir:    dataDir,
		projectDBs: make(map[domain.ProjectID]*sql.DB),
	}

	return &Repositories{
		Projects:     &ProjectRepository{base},
		Roles:        &RoleRepository{base},
		Tasks:        &TaskRepository{base},
		Columns:      &ColumnRepository{base},
		Comments:     &CommentRepository{base},
		Dependencies: &DependencyRepository{base},
		ToolUsage:    &ToolUsageRepository{base},
		base:         base,
	}, nil
}

// Close closes all database connections
func (r *Repositories) Close() error {
	r.base.mu.Lock()
	defer r.base.mu.Unlock()
	for id, db := range r.base.projectDBs {
		db.Close()
		delete(r.base.projectDBs, id)
	}
	return r.base.globalDB.Close()
}

// openDB opens a SQLite database with recommended settings
func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on", path))
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// runMigration runs a SQL migration script
func runMigration(db *sql.DB, migration string) error {
	_, err := db.Exec(migration)
	return err
}

// getProjectDB returns a cached or newly-opened project-specific database.
// Migrations run only on first open; the connection is reused for subsequent calls.
func (r *baseRepository) getProjectDB(projectID domain.ProjectID) (*sql.DB, error) {
	r.mu.Lock()
	if db, ok := r.projectDBs[projectID]; ok {
		r.mu.Unlock()
		return db, nil
	}
	r.mu.Unlock()

	// Verify project exists in global database
	var exists int
	err := r.globalDB.QueryRow("SELECT 1 FROM projects WHERE id = ?", string(projectID)).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}
		return nil, err
	}

	dbPath := filepath.Join(r.dataDir, ProjectDBName(projectID))
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	// Run project migrations in order (only on first open)
	if err := runMigration(db, projectMigration); err != nil {
		db.Close()
		return nil, err
	}
	// ALTER TABLE ADD COLUMN is not idempotent in SQLite — ignore "duplicate column" errors
	if err := runMigration(db, taskSeenAtMigration); err != nil && !isDuplicateColumn(err) {
		db.Close()
		return nil, err
	}
	if err := runMigration(db, taskTokenUsageMigration); err != nil && !isDuplicateColumn(err) {
		db.Close()
		return nil, err
	}
	// FTS5 migration uses IF NOT EXISTS — safe to re-run
	if err := runMigration(db, taskFTSMigration); err != nil {
		db.Close()
		return nil, err
	}
	// Add composite indexes for query optimization
	if err := runMigration(db, projectIndexesMigration); err != nil {
		db.Close()
		return nil, err
	}
	// Tool usage tracking table
	if err := runMigration(db, toolUsageMigration); err != nil {
		db.Close()
		return nil, err
	}

	r.mu.Lock()
	// Double-check: another goroutine may have raced and inserted first
	if existing, ok := r.projectDBs[projectID]; ok {
		r.mu.Unlock()
		db.Close()
		return existing, nil
	}
	r.projectDBs[projectID] = db
	r.mu.Unlock()

	return db, nil
}

// withProjectDB executes a function with a project database connection
func (r *baseRepository) withProjectDB(ctx context.Context, projectID domain.ProjectID, fn func(*sql.DB) error) error {
	db, err := r.getProjectDB(projectID)
	if err != nil {
		return err
	}

	return fn(db)
}

// Helper function to check if error is "not found"
func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// isDuplicateColumn checks if an error is a SQLite "duplicate column name" error
func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}
