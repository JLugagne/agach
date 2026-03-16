package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
)

// Create creates a new task in a project database
func (r *TaskRepository) Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	contextFilesJSON, err := json.Marshal(task.ContextFiles)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return err
	}

	filesModifiedJSON, err := json.Marshal(task.FilesModified)
	if err != nil {
		return err
	}

	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			INSERT INTO tasks (
				id, column_id, title, summary, description, priority, priority_score, position,
				created_by_role, created_by_agent, assigned_role, is_blocked, blocked_reason,
				blocked_at, blocked_by_agent, wont_do_requested, wont_do_reason, wont_do_requested_by,
				wont_do_requested_at, completion_summary, completed_by_agent, completed_at,
				files_modified, resolution, context_files, tags, estimated_effort, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model, created_at, updated_at,
				seen_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err := db.ExecContext(ctx, query,
			string(task.ID),
			string(task.ColumnID),
			task.Title,
			task.Summary,
			task.Description,
			task.Priority,
			task.PriorityScore,
			task.Position,
			task.CreatedByRole,
			task.CreatedByAgent,
			task.AssignedRole,
			boolToInt(task.IsBlocked),
			task.BlockedReason,
			timeToNullTime(task.BlockedAt),
			task.BlockedByAgent,
			boolToInt(task.WontDoRequested),
			task.WontDoReason,
			task.WontDoRequestedBy,
			timeToNullTime(task.WontDoRequestedAt),
			task.CompletionSummary,
			task.CompletedByAgent,
			timeToNullTime(task.CompletedAt),
			string(filesModifiedJSON),
			task.Resolution,
			string(contextFilesJSON),
			string(tagsJSON),
			task.EstimatedEffort,
			task.InputTokens,
			task.OutputTokens,
			task.CacheReadTokens,
			task.CacheWriteTokens,
			task.Model,
			task.CreatedAt,
			task.UpdatedAt,
			timeToNullTime(task.SeenAt),
		)

		if err != nil {
			if isSQLiteConstraintError(err, "PRIMARY KEY") {
				return errors.Join(domain.ErrTaskAlreadyExists, err)
			}
			return err
		}

		return nil
	})
}

// FindByID retrieves a task by ID from a project database
func (r *TaskRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
	var task *domain.Task

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, column_id, title, summary, description, priority, priority_score, position,
				created_by_role, created_by_agent, assigned_role, is_blocked, blocked_reason,
				blocked_at, blocked_by_agent, wont_do_requested, wont_do_reason, wont_do_requested_by,
				wont_do_requested_at, completion_summary, completed_by_agent, completed_at,
				files_modified, resolution, context_files, tags, estimated_effort, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model, created_at, updated_at,
				seen_at
			FROM tasks
			WHERE id = ?
		`

		t, err := r.scanTask(db.QueryRowContext(ctx, query, string(id)))
		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrTaskNotFound, err)
			}
			return err
		}

		task = t
		return nil
	})

	if err != nil {
		return nil, err
	}

	return task, nil
}

// List retrieves tasks from a project database with optional filters
func (r *TaskRepository) List(ctx context.Context, projectID domain.ProjectID, filters tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
	var tasks []domain.TaskWithDetails

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		// When a search term is provided, JOIN with the FTS5 table
		var ftsJoin string
		var args []interface{}
		var whereClauses []string

		if filters.Search != "" {
			ftsJoin = " JOIN tasks_fts ON tasks.rowid = tasks_fts.rowid"
			whereClauses = append(whereClauses, "tasks_fts MATCH ?")
			args = append(args, filters.Search)
		}

		query := `
			SELECT tasks.id, tasks.column_id, tasks.title, tasks.summary, tasks.description, tasks.priority, tasks.priority_score, tasks.position,
				tasks.created_by_role, tasks.created_by_agent, tasks.assigned_role, tasks.is_blocked, tasks.blocked_reason,
				tasks.blocked_at, tasks.blocked_by_agent, tasks.wont_do_requested, tasks.wont_do_reason, tasks.wont_do_requested_by,
				tasks.wont_do_requested_at, tasks.completion_summary, tasks.completed_by_agent, tasks.completed_at,
				tasks.files_modified, tasks.resolution, tasks.context_files, tasks.tags, tasks.estimated_effort, tasks.input_tokens, tasks.output_tokens, tasks.cache_read_tokens, tasks.cache_write_tokens, tasks.model, tasks.created_at, tasks.updated_at,
				tasks.seen_at
			FROM tasks` + ftsJoin + `
			WHERE 1=1
		`

		if filters.ColumnSlug != nil {
			// Use a subquery instead of a separate round-trip to look up the column ID.
			whereClauses = append(whereClauses, "column_id = (SELECT id FROM columns WHERE slug = ?)")
			args = append(args, string(*filters.ColumnSlug))
		}

		if filters.AssignedRole != nil {
			whereClauses = append(whereClauses, "assigned_role = ?")
			args = append(args, *filters.AssignedRole)
		}

		if filters.Priority != nil {
			whereClauses = append(whereClauses, "priority = ?")
			args = append(args, string(*filters.Priority))
		}

		if filters.Tag != nil {
			// Escape SQLite LIKE wildcards in the user-supplied tag value so that
			// a tag containing '%' or '_' matches literally rather than as a pattern.
			escapedTag := escapeLike(*filters.Tag)
			whereClauses = append(whereClauses, `tags LIKE ? ESCAPE '\'`)
			args = append(args, `%"`+escapedTag+`"%`)
		}

		if filters.IsBlocked != nil {
			whereClauses = append(whereClauses, "is_blocked = ?")
			args = append(args, boolToInt(*filters.IsBlocked))
		}

		if filters.WontDoRequested != nil {
			whereClauses = append(whereClauses, "wont_do_requested = ?")
			args = append(args, boolToInt(*filters.WontDoRequested))
		}

		if filters.UpdatedSince != nil {
			whereClauses = append(whereClauses, "updated_at >= ?")
			args = append(args, *filters.UpdatedSince)
		}

		if len(whereClauses) > 0 {
			query += " AND " + strings.Join(whereClauses, " AND ")
		}

		query += " ORDER BY priority_score DESC, created_at ASC"

		if filters.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filters.Limit)
			if filters.Offset > 0 {
				query += " OFFSET ?"
				args = append(args, filters.Offset)
			}
		}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			t, err := r.scanTask(rows)
			if err != nil {
				return err
			}

			// For now, TaskWithDetails just wraps Task
			// HasUnresolvedDeps and CommentCount will be fetched separately if needed
			tasks = append(tasks, domain.TaskWithDetails{
				Task:              *t,
				HasUnresolvedDeps: false,
				CommentCount:      0,
			})
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// Update updates an existing task in a project database
func (r *TaskRepository) Update(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	contextFilesJSON, err := json.Marshal(task.ContextFiles)
	if err != nil {
		return err
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return err
	}

	filesModifiedJSON, err := json.Marshal(task.FilesModified)
	if err != nil {
		return err
	}

	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			UPDATE tasks
			SET column_id = ?, title = ?, summary = ?, description = ?, priority = ?, priority_score = ?,
				position = ?, assigned_role = ?, is_blocked = ?, blocked_reason = ?, blocked_at = ?,
				blocked_by_agent = ?, wont_do_requested = ?, wont_do_reason = ?, wont_do_requested_by = ?,
				wont_do_requested_at = ?, completion_summary = ?, completed_by_agent = ?, completed_at = ?,
				files_modified = ?, resolution = ?, context_files = ?, tags = ?, estimated_effort = ?,
				input_tokens = ?, output_tokens = ?, cache_read_tokens = ?, cache_write_tokens = ?, model = ?,
				updated_at = ?
			WHERE id = ?
		`

		result, err := db.ExecContext(ctx, query,
			string(task.ColumnID),
			task.Title,
			task.Summary,
			task.Description,
			task.Priority,
			task.PriorityScore,
			task.Position,
			task.AssignedRole,
			boolToInt(task.IsBlocked),
			task.BlockedReason,
			timeToNullTime(task.BlockedAt),
			task.BlockedByAgent,
			boolToInt(task.WontDoRequested),
			task.WontDoReason,
			task.WontDoRequestedBy,
			timeToNullTime(task.WontDoRequestedAt),
			task.CompletionSummary,
			task.CompletedByAgent,
			timeToNullTime(task.CompletedAt),
			string(filesModifiedJSON),
			task.Resolution,
			string(contextFilesJSON),
			string(tagsJSON),
			task.EstimatedEffort,
			task.InputTokens,
			task.OutputTokens,
			task.CacheReadTokens,
			task.CacheWriteTokens,
			task.Model,
			time.Now(),
			string(task.ID),
		)

		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return domain.ErrTaskNotFound
		}

		return nil
	})
}

// Delete deletes a task from a project database
func (r *TaskRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `DELETE FROM tasks WHERE id = ?`

		result, err := db.ExecContext(ctx, query, string(id))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return domain.ErrTaskNotFound
		}

		return nil
	})
}

// Move moves a task to a different column
func (r *TaskRepository) Move(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `UPDATE tasks SET column_id = ?, updated_at = ? WHERE id = ?`

		result, err := db.ExecContext(ctx, query, string(targetColumnID), time.Now(), string(taskID))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return domain.ErrTaskNotFound
		}

		return nil
	})
}

// CountByColumn counts tasks in a specific column
func (r *TaskRepository) CountByColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error) {
	var count int

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `SELECT COUNT(*) FROM tasks WHERE column_id = ?`
		return db.QueryRowContext(ctx, query, string(columnID)).Scan(&count)
	})

	return count, err
}

// GetNextTask retrieves the highest priority task for a role that is not blocked or won't-do-requested.
// The "todo" column ID is resolved inline via a subquery to avoid a separate round-trip.
func (r *TaskRepository) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error) {
	var task *domain.Task

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, column_id, title, summary, description, priority, priority_score, position,
				created_by_role, created_by_agent, assigned_role, is_blocked, blocked_reason,
				blocked_at, blocked_by_agent, wont_do_requested, wont_do_reason, wont_do_requested_by,
				wont_do_requested_at, completion_summary, completed_by_agent, completed_at,
				files_modified, resolution, context_files, tags, estimated_effort, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model, created_at, updated_at,
				seen_at
			FROM tasks
			WHERE column_id = 'col_todo'
				AND (assigned_role = '' OR assigned_role = ?)
				AND is_blocked = 0
				AND wont_do_requested = 0
			ORDER BY priority_score DESC, created_at ASC
			LIMIT 1
		`

		t, err := r.scanTask(db.QueryRowContext(ctx, query, role))
		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrNoTasksAvailable, err)
			}
			return err
		}

		task = t
		return nil
	})

	if err != nil {
		return nil, err
	}

	return task, nil
}

// HasUnresolvedDependencies checks if a task has any unresolved dependencies.
// The "done" column ID is resolved inline via a subquery to avoid a separate round-trip.
func (r *TaskRepository) HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error) {
	var hasUnresolved bool

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT EXISTS(
				SELECT 1
				FROM task_dependencies td
				JOIN tasks t ON td.depends_on_task_id = t.id
				WHERE td.task_id = ?
					AND t.column_id != 'col_done'
			)
		`

		err := db.QueryRowContext(ctx, query, string(taskID)).Scan(&hasUnresolved)
		if err != nil {
			return err
		}

		return nil
	})

	return hasUnresolved, err
}

// GetDependentsNotDone retrieves all tasks that depend on this task and are not done.
// The "done" column ID is resolved inline via a subquery to avoid a separate round-trip.
func (r *TaskRepository) GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	var tasks []domain.Task

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT t.id, t.column_id, t.title, t.summary, t.description, t.priority, t.priority_score, t.position,
				t.created_by_role, t.created_by_agent, t.assigned_role, t.is_blocked, t.blocked_reason,
				t.blocked_at, t.blocked_by_agent, t.wont_do_requested, t.wont_do_reason, t.wont_do_requested_by,
				t.wont_do_requested_at, t.completion_summary, t.completed_by_agent, t.completed_at,
				t.files_modified, t.resolution, t.context_files, t.tags, t.estimated_effort, t.input_tokens, t.output_tokens, t.cache_read_tokens, t.cache_write_tokens, t.model, t.created_at, t.updated_at,
				t.seen_at
			FROM task_dependencies td
			JOIN tasks t ON td.task_id = t.id
			WHERE td.depends_on_task_id = ?
				AND t.column_id != 'col_done'
		`

		rows, err := db.QueryContext(ctx, query, string(taskID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			t, err := r.scanTask(rows)
			if err != nil {
				return err
			}
			tasks = append(tasks, *t)
		}

		return rows.Err()
	})

	return tasks, err
}

// scanTask scans a task row into a domain.Task
func (r *TaskRepository) scanTask(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Task, error) {
	var task domain.Task
	var blockedAt, wontDoRequestedAt, completedAt, seenAt sql.NullTime
	var createdAt, updatedAt time.Time
	var contextFilesJSON, tagsJSON, filesModifiedJSON string
	var isBlocked, wontDoRequested int

	err := scanner.Scan(
		&task.ID,
		&task.ColumnID,
		&task.Title,
		&task.Summary,
		&task.Description,
		&task.Priority,
		&task.PriorityScore,
		&task.Position,
		&task.CreatedByRole,
		&task.CreatedByAgent,
		&task.AssignedRole,
		&isBlocked,
		&task.BlockedReason,
		&blockedAt,
		&task.BlockedByAgent,
		&wontDoRequested,
		&task.WontDoReason,
		&task.WontDoRequestedBy,
		&wontDoRequestedAt,
		&task.CompletionSummary,
		&task.CompletedByAgent,
		&completedAt,
		&filesModifiedJSON,
		&task.Resolution,
		&contextFilesJSON,
		&tagsJSON,
		&task.EstimatedEffort,
		&task.InputTokens,
		&task.OutputTokens,
		&task.CacheReadTokens,
		&task.CacheWriteTokens,
		&task.Model,
		&createdAt,
		&updatedAt,
		&seenAt,
	)

	if err != nil {
		return nil, err
	}

	task.IsBlocked = isBlocked == 1
	task.WontDoRequested = wontDoRequested == 1
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt

	if blockedAt.Valid {
		task.BlockedAt = &blockedAt.Time
	}
	if wontDoRequestedAt.Valid {
		task.WontDoRequestedAt = &wontDoRequestedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if seenAt.Valid {
		task.SeenAt = &seenAt.Time
	}

	if err := json.Unmarshal([]byte(contextFilesJSON), &task.ContextFiles); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(tagsJSON), &task.Tags); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(filesModifiedJSON), &task.FilesModified); err != nil {
		return nil, err
	}

	return &task, nil
}

// MarkTaskSeen sets seen_at = CURRENT_TIMESTAMP for the given task if it has not been seen yet.
// The operation is idempotent: if seen_at is already set it is left unchanged.
func (r *TaskRepository) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		// Try the conditional update first; check RowsAffected to distinguish
		// "already seen" from "task not found".
		query := `UPDATE tasks SET seen_at = CURRENT_TIMESTAMP WHERE id = ? AND seen_at IS NULL`
		result, err := db.ExecContext(ctx, query, string(taskID))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			// Either already seen or task doesn't exist — check which
			var exists int
			err := db.QueryRowContext(ctx, "SELECT 1 FROM tasks WHERE id = ?", string(taskID)).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return errors.Join(domain.ErrTaskNotFound, err)
				}
				return err
			}
			// Task exists but was already seen — idempotent success
		}

		return nil
	})
}

// Helper functions for SQLite conversions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timeToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// escapeLike escapes SQLite LIKE special characters ('%', '_', '\') in a user-supplied
// string so the value is matched literally when used with LIKE ... ESCAPE '\'.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
