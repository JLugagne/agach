package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Create creates a new task dependency in a project database
func (r *DependencyRepository) Create(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		// Check for self-reference
		if dep.TaskID == dep.DependsOnTaskID {
			return domain.ErrCannotDependOnSelf
		}

		query := `
			INSERT INTO task_dependencies (id, task_id, depends_on_task_id, created_at)
			VALUES (?, ?, ?, ?)
		`

		_, err := db.ExecContext(ctx, query,
			string(dep.ID),
			string(dep.TaskID),
			string(dep.DependsOnTaskID),
			dep.CreatedAt,
		)

		if err != nil {
			if isSQLiteConstraintError(err, "UNIQUE") {
				return errors.Join(domain.ErrDependencyAlreadyExists, err)
			}
			return err
		}

		return nil
	})
}

// Delete deletes a task dependency from a project database
func (r *DependencyRepository) Delete(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `DELETE FROM task_dependencies WHERE task_id = ? AND depends_on_task_id = ?`

		result, err := db.ExecContext(ctx, query, string(taskID), string(dependsOnTaskID))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return errors.Join(domain.ErrDependencyNotFound, errors.New("dependency not found"))
		}

		return nil
	})
}

// List retrieves all dependencies for a task
func (r *DependencyRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	var deps []domain.TaskDependency

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, task_id, depends_on_task_id, created_at
			FROM task_dependencies
			WHERE task_id = ?
			ORDER BY created_at ASC
		`

		rows, err := db.QueryContext(ctx, query, string(taskID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var dep domain.TaskDependency
			var createdAt time.Time

			err := rows.Scan(
				&dep.ID,
				&dep.TaskID,
				&dep.DependsOnTaskID,
				&createdAt,
			)

			if err != nil {
				return err
			}

			dep.CreatedAt = createdAt
			deps = append(deps, dep)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return deps, nil
}

// WouldCreateCycle checks if adding a dependency would create a cycle
func (r *DependencyRepository) WouldCreateCycle(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error) {
	// Self-reference is a cycle
	if taskID == dependsOnTaskID {
		return true, nil
	}

	var wouldCycle bool

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		// Use recursive CTE to check for cycles
		// If dependsOnTaskID can reach taskID through its dependencies, adding taskID -> dependsOnTaskID would create a cycle
		query := `
			WITH RECURSIVE dependency_chain(task_id, depends_on_task_id, depth) AS (
				-- Base case: start from dependsOnTaskID
				SELECT task_id, depends_on_task_id, 1
				FROM task_dependencies
				WHERE task_id = ?

				UNION ALL

				-- Recursive case: follow the dependency chain
				SELECT td.task_id, td.depends_on_task_id, dc.depth + 1
				FROM task_dependencies td
				INNER JOIN dependency_chain dc ON td.task_id = dc.depends_on_task_id
				WHERE dc.depth < 100  -- Prevent infinite recursion
			)
			SELECT COUNT(*) FROM dependency_chain WHERE depends_on_task_id = ?
		`

		var count int
		err := db.QueryRowContext(ctx, query, string(dependsOnTaskID), string(taskID)).Scan(&count)
		if err != nil {
			return err
		}

		wouldCycle = count > 0
		return nil
	})

	return wouldCycle, err
}

// ListDependents retrieves all task dependencies where the given task is the dependency target
// (i.e. tasks that have taskID as a dependency)
func (r *DependencyRepository) ListDependents(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	var deps []domain.TaskDependency

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, task_id, depends_on_task_id, created_at
			FROM task_dependencies
			WHERE depends_on_task_id = ?
			ORDER BY created_at ASC
		`

		rows, err := db.QueryContext(ctx, query, string(taskID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var dep domain.TaskDependency
			var createdAt time.Time

			err := rows.Scan(
				&dep.ID,
				&dep.TaskID,
				&dep.DependsOnTaskID,
				&createdAt,
			)

			if err != nil {
				return err
			}

			dep.CreatedAt = createdAt
			deps = append(deps, dep)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return deps, nil
}

// GetDependencyContext retrieves the dependency context for a task
// This includes information about dependencies and their current status
func (r *DependencyRepository) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	var contexts []domain.DependencyContext

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT
				td.depends_on_task_id,
				t.title,
				t.completion_summary,
				t.files_modified
			FROM task_dependencies td
			JOIN tasks t ON td.depends_on_task_id = t.id
			WHERE td.task_id = ?
			ORDER BY td.created_at ASC
		`

		rows, err := db.QueryContext(ctx, query, string(taskID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var depCtx domain.DependencyContext
			var filesModifiedJSON string

			err := rows.Scan(
				&depCtx.TaskID,
				&depCtx.Title,
				&depCtx.CompletionSummary,
				&filesModifiedJSON,
			)

			if err != nil {
				return err
			}

			if err := json.Unmarshal([]byte(filesModifiedJSON), &depCtx.FilesModified); err != nil {
				return err
			}

			contexts = append(contexts, depCtx)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return contexts, nil
}
