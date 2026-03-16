package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/sirupsen/logrus"
)

// Create creates a new project in the global database
func (r *ProjectRepository) Create(ctx context.Context, project domain.Project) error {
	query := `
		INSERT INTO projects (id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var parentID *string
	if project.ParentID != nil {
		id := string(*project.ParentID)
		parentID = &id
	}

	_, err := r.globalDB.ExecContext(ctx, query,
		string(project.ID),
		parentID,
		project.Name,
		project.Description,
		project.WorkDir,
		project.CreatedByRole,
		project.CreatedByAgent,
		project.CreatedAt,
		project.UpdatedAt,
	)

	if err != nil {
		if isSQLiteConstraintError(err, "UNIQUE") || isSQLiteConstraintError(err, "PRIMARY KEY") {
			return errors.Join(domain.ErrProjectAlreadyExists, err)
		}
		return err
	}

	return nil
}

// FindByID retrieves a project by ID from the global database
func (r *ProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	query := `
		SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
		FROM projects
		WHERE id = ?
	`

	var project domain.Project
	var parentID sql.NullString
	var createdAt, updatedAt time.Time

	err := r.globalDB.QueryRowContext(ctx, query, string(id)).Scan(
		&project.ID,
		&parentID,
		&project.Name,
		&project.Description,
		&project.WorkDir,
		&project.CreatedByRole,
		&project.CreatedByAgent,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if isNotFound(err) {
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}
		return nil, err
	}

	project.CreatedAt = createdAt
	project.UpdatedAt = updatedAt

	if parentID.Valid {
		pid := domain.ProjectID(parentID.String)
		project.ParentID = &pid
	}

	return &project, nil
}

// List retrieves all root projects (parent_id IS NULL) or children of a parent
func (r *ProjectRepository) List(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
	var query string
	var args []interface{}

	if parentID == nil {
		query = `
			SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
			FROM projects
			WHERE parent_id IS NULL
			ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
			FROM projects
			WHERE parent_id = ?
			ORDER BY created_at DESC
		`
		args = append(args, string(*parentID))
	}

	rows, err := r.globalDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanProjects(rows)
}

// Update updates an existing project in the global database
func (r *ProjectRepository) Update(ctx context.Context, project domain.Project) error {
	query := `
		UPDATE projects
		SET name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.globalDB.ExecContext(ctx, query,
		project.Name,
		project.Description,
		time.Now(),
		string(project.ID),
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrProjectNotFound
	}

	return nil
}

// GetTree retrieves a project and all its sub-projects recursively
func (r *ProjectRepository) GetTree(ctx context.Context, id domain.ProjectID) ([]domain.Project, error) {
	query := `
		WITH RECURSIVE project_tree AS (
			-- Base case: the requested project
			SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
			FROM projects
			WHERE id = ?

			UNION ALL

			-- Recursive case: all children
			SELECT p.id, p.parent_id, p.name, p.description, p.work_dir, p.created_by_role, p.created_by_agent, p.created_at, p.updated_at
			FROM projects p
			INNER JOIN project_tree pt ON p.parent_id = pt.id
		)
		SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
		FROM project_tree
		ORDER BY created_at ASC
	`

	rows, err := r.globalDB.QueryContext(ctx, query, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects, err := r.scanProjects(rows)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 {
		return nil, domain.ErrProjectNotFound
	}

	return projects, nil
}

// Delete deletes a project and all sub-projects in cascade
// Returns the list of deleted project IDs (including descendants)
func (r *ProjectRepository) Delete(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error) {
	// First get all projects in the tree
	projects, err := r.GetTree(ctx, id)
	if err != nil {
		return nil, err
	}

	// Extract IDs
	deletedIDs := make([]domain.ProjectID, len(projects))
	for i, p := range projects {
		deletedIDs[i] = p.ID
	}

	// Delete the root project (CASCADE will handle children)
	query := `DELETE FROM projects WHERE id = ?`
	_, err = r.globalDB.ExecContext(ctx, query, string(id))
	if err != nil {
		return nil, err
	}

	// Close cached DB connections and remove SQLite database files for each deleted project
	r.mu.Lock()
	for _, p := range projects {
		if db, ok := r.projectDBs[p.ID]; ok {
			db.Close()
			delete(r.projectDBs, p.ID)
		}
	}
	r.mu.Unlock()

	for _, p := range projects {
		dbPath := filepath.Join(r.dataDir, ProjectDBName(p.ID))
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			logrus.WithError(err).WithField("projectID", string(p.ID)).Warn("failed to remove project database file")
		}
		// Also remove WAL and SHM files if they exist
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	}

	return deletedIDs, nil
}

// GetSummary returns task counts per column for a project.
// Uses a single JOIN query to avoid two round-trips (was: separate column lookup + count query).
func (r *ProjectRepository) GetSummary(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
	var summary domain.ProjectSummary

	err := r.withProjectDB(ctx, id, func(db *sql.DB) error {
		// Single query: join columns with tasks and pivot by slug in Go.
		query := `
			SELECT c.slug, COUNT(t.id)
			FROM columns c
			LEFT JOIN tasks t ON t.column_id = c.id
			GROUP BY c.id, c.slug
		`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var slug string
			var count int
			if err := rows.Scan(&slug, &count); err != nil {
				return err
			}
			switch slug {
			case "todo":
				summary.TodoCount = count
			case "in_progress":
				summary.InProgressCount = count
			case "done":
				summary.DoneCount = count
			case "blocked":
				summary.BlockedCount = count
			}
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// ListByWorkDir retrieves all projects matching the given work_dir
func (r *ProjectRepository) ListByWorkDir(ctx context.Context, workDir string) ([]domain.Project, error) {
	query := `
		SELECT id, parent_id, name, description, work_dir, created_by_role, created_by_agent, created_at, updated_at
		FROM projects
		WHERE work_dir = ?
		ORDER BY created_at DESC
	`

	rows, err := r.globalDB.QueryContext(ctx, query, workDir)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanProjects(rows)
}

// CountChildren returns the number of direct children
func (r *ProjectRepository) CountChildren(ctx context.Context, id domain.ProjectID) (int, error) {
	query := `SELECT COUNT(*) FROM projects WHERE parent_id = ?`

	var count int
	err := r.globalDB.QueryRowContext(ctx, query, string(id)).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// scanProjects is a helper to scan multiple project rows
func (r *ProjectRepository) scanProjects(rows *sql.Rows) ([]domain.Project, error) {
	var projects []domain.Project

	for rows.Next() {
		var project domain.Project
		var parentID sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&project.ID,
			&parentID,
			&project.Name,
			&project.Description,
			&project.WorkDir,
			&project.CreatedByRole,
			&project.CreatedByAgent,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, err
		}

		project.CreatedAt = createdAt
		project.UpdatedAt = updatedAt

		if parentID.Valid {
			pid := domain.ProjectID(parentID.String)
			project.ParentID = &pid
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}

// isSQLiteConstraintError checks if error is a SQLite constraint error
func isSQLiteConstraintError(err error, constraintType string) bool {
	if err == nil {
		return false
	}
	// SQLite constraint errors contain the constraint type in the message
	errMsg := err.Error()
	return contains(errMsg, "UNIQUE") && constraintType == "UNIQUE" ||
		contains(errMsg, "PRIMARY KEY") && constraintType == "PRIMARY KEY"
}

// contains checks if string contains substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
