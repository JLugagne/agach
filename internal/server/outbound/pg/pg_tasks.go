package pg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/jackc/pgx/v5"
)

type taskRepository struct{ *baseRepository }

const taskSelectCols = `
	t.id, t.column_id, t.feature_id, t.title, t.summary, t.description, t.priority, t.priority_score,
	t.position, t.created_by_role, t.created_by_agent, t.assigned_role,
	t.is_blocked, t.blocked_reason, t.blocked_at, t.blocked_by_agent,
	t.wont_do_requested, t.wont_do_reason, t.wont_do_requested_by, t.wont_do_requested_at,
	t.completion_summary, t.completed_by_agent, t.completed_at,
	t.files_modified, t.resolution, t.context_files, t.tags, t.estimated_effort,
	t.seen_at, t.session_id, t.node_id,
	t.input_tokens, t.output_tokens, t.cache_read_tokens, t.cache_write_tokens, t.model,
	t.cold_start_input_tokens, t.cold_start_output_tokens, t.cold_start_cache_read_tokens, t.cold_start_cache_write_tokens,
	t.started_at, t.duration_seconds, t.human_estimate_seconds,
	t.created_at, t.updated_at
`

func scanTaskInto(s scanner) (domain.Task, error) {
	var t domain.Task
	var filesModifiedJSON, contextFilesJSON, tagsJSON []byte
	var isBlocked, wontDoRequested int
	var featureIDStr, nodeIDStr *string
	err := s.Scan(
		(*string)(&t.ID), (*string)(&t.ColumnID), &featureIDStr, &t.Title, &t.Summary, &t.Description,
		(*string)(&t.Priority), &t.PriorityScore, &t.Position,
		&t.CreatedByRole, &t.CreatedByAgent, &t.AssignedRole,
		&isBlocked, &t.BlockedReason, &t.BlockedAt, &t.BlockedByAgent,
		&wontDoRequested, &t.WontDoReason, &t.WontDoRequestedBy, &t.WontDoRequestedAt,
		&t.CompletionSummary, &t.CompletedByAgent, &t.CompletedAt,
		&filesModifiedJSON, &t.Resolution, &contextFilesJSON, &tagsJSON, &t.EstimatedEffort,
		&t.SeenAt, &t.SessionID, &nodeIDStr,
		&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.CacheWriteTokens, &t.Model,
		&t.ColdStartInputTokens, &t.ColdStartOutputTokens, &t.ColdStartCacheReadTokens, &t.ColdStartCacheWriteTokens,
		&t.StartedAt, &t.DurationSeconds, &t.HumanEstimateSeconds,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return domain.Task{}, err
	}
	if featureIDStr != nil {
		fid := domain.FeatureID(*featureIDStr)
		t.FeatureID = &fid
	}
	if nodeIDStr != nil {
		t.NodeID = *nodeIDStr
	}
	t.IsBlocked = isBlocked == 1
	t.WontDoRequested = wontDoRequested == 1
	t.FilesModified = jsonUnmarshalStrings(filesModifiedJSON)
	t.ContextFiles = jsonUnmarshalStrings(contextFilesJSON)
	t.Tags = jsonUnmarshalStrings(tagsJSON)
	return t, nil
}

func scanTask(row pgx.Row) (*domain.Task, error) {
	t, err := scanTaskInto(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}
	return &t, nil
}

func scanTaskRows(rows pgx.Rows) ([]domain.Task, error) {
	var result []domain.Task
	for rows.Next() {
		t, err := scanTaskInto(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func featureIDOrNil(id *domain.FeatureID) interface{} {
	if id == nil {
		return nil
	}
	return string(*id)
}

func (r *taskRepository) Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	filesJSON := jsonMarshal(task.FilesModified)
	contextJSON := jsonMarshal(task.ContextFiles)
	tagsJSON := jsonMarshal(task.Tags)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO tasks (
			id, project_id, column_id, feature_id, title, summary, description,
			priority, priority_score, position,
			created_by_role, created_by_agent, assigned_role,
			is_blocked, blocked_reason, blocked_at, blocked_by_agent,
			wont_do_requested, wont_do_reason, wont_do_requested_by, wont_do_requested_at,
			completion_summary, completed_by_agent, completed_at,
			files_modified, resolution, context_files, tags, estimated_effort,
			seen_at, session_id, node_id,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model,
			cold_start_input_tokens, cold_start_output_tokens, cold_start_cache_read_tokens, cold_start_cache_write_tokens,
			started_at, duration_seconds, human_estimate_seconds,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,
			$11,$12,$13,
			$14,$15,$16,$17,
			$18,$19,$20,$21,
			$22,$23,$24,
			$25,$26,$27,$28,$29,
			$30,$31,$32,
			$33,$34,$35,$36,$37,
			$38,$39,$40,$41,
			$42,$43,$44,
			$45,$46
		)`,
		string(task.ID), string(projectID), string(task.ColumnID), featureIDOrNil(task.FeatureID),
		task.Title, task.Summary, task.Description,
		string(task.Priority), task.PriorityScore, task.Position,
		task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
		boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
		boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
		task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
		filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
		task.SeenAt, task.SessionID, nullableString(task.NodeID),
		task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
		task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
		task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *taskRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `SELECT `+taskSelectCols+` FROM tasks t WHERE t.project_id=$1 AND t.id=$2`,
		string(projectID), string(id))
	return scanTask(row)
}

func (r *taskRepository) List(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	args := []any{string(projectID)}
	where := []string{"t.project_id = $1"}
	argIdx := 2

	if filters.ColumnSlug != nil {
		where = append(where, fmt.Sprintf(`c.slug = $%d`, argIdx))
		args = append(args, string(*filters.ColumnSlug))
		argIdx++
	}
	if filters.AssignedRole != nil {
		where = append(where, fmt.Sprintf(`t.assigned_role = $%d`, argIdx))
		args = append(args, *filters.AssignedRole)
		argIdx++
	}
	if filters.Priority != nil {
		where = append(where, fmt.Sprintf(`t.priority = $%d`, argIdx))
		args = append(args, string(*filters.Priority))
		argIdx++
	}
	if filters.IsBlocked != nil {
		where = append(where, fmt.Sprintf(`t.is_blocked = $%d`, argIdx))
		args = append(args, boolToInt(*filters.IsBlocked))
		argIdx++
	}
	if filters.WontDoRequested != nil {
		where = append(where, fmt.Sprintf(`t.wont_do_requested = $%d`, argIdx))
		args = append(args, boolToInt(*filters.WontDoRequested))
		argIdx++
	}
	if filters.UpdatedSince != nil {
		where = append(where, fmt.Sprintf(`t.updated_at >= $%d`, argIdx))
		args = append(args, *filters.UpdatedSince)
		argIdx++
	}
	if filters.FeatureID != nil {
		where = append(where, fmt.Sprintf(`t.feature_id = $%d`, argIdx))
		args = append(args, string(*filters.FeatureID))
		argIdx++
	}
	if filters.Search != "" {
		where = append(where, fmt.Sprintf(`t.search_vector @@ plainto_tsquery('english', $%d)`, argIdx))
		args = append(args, filters.Search)
		argIdx++
	}

	query := `
		SELECT ` + taskSelectCols + `,
			(EXISTS (
				SELECT 1 FROM task_dependencies td
				JOIN tasks dep ON dep.id = td.depends_on_task_id
				JOIN columns dc ON dc.id = dep.column_id
				WHERE td.task_id = t.id AND dc.slug != 'done'
			)) AS has_unresolved_deps,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id) AS comment_count
		FROM tasks t
		JOIN columns c ON c.id = t.column_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY t.priority_score DESC, t.created_at ASC`

	if filters.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, filters.Limit)
		argIdx++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var result []domain.TaskWithDetails
	for rows.Next() {
		var t domain.Task
		var filesModifiedJSON, contextFilesJSON, tagsJSON []byte
		var isBlocked, wontDoRequested int
		var hasUnresolvedDeps bool
		var commentCount int
		var featureIDStr, nodeIDStr *string

		err := rows.Scan(
			(*string)(&t.ID), (*string)(&t.ColumnID), &featureIDStr, &t.Title, &t.Summary, &t.Description,
			(*string)(&t.Priority), &t.PriorityScore, &t.Position,
			&t.CreatedByRole, &t.CreatedByAgent, &t.AssignedRole,
			&isBlocked, &t.BlockedReason, &t.BlockedAt, &t.BlockedByAgent,
			&wontDoRequested, &t.WontDoReason, &t.WontDoRequestedBy, &t.WontDoRequestedAt,
			&t.CompletionSummary, &t.CompletedByAgent, &t.CompletedAt,
			&filesModifiedJSON, &t.Resolution, &contextFilesJSON, &tagsJSON, &t.EstimatedEffort,
			&t.SeenAt, &t.SessionID, &nodeIDStr,
			&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.CacheWriteTokens, &t.Model,
			&t.ColdStartInputTokens, &t.ColdStartOutputTokens, &t.ColdStartCacheReadTokens, &t.ColdStartCacheWriteTokens,
			&t.StartedAt, &t.DurationSeconds, &t.HumanEstimateSeconds,
			&t.CreatedAt, &t.UpdatedAt,
			&hasUnresolvedDeps, &commentCount,
		)
		if err != nil {
			return nil, err
		}
		if featureIDStr != nil {
			fid := domain.FeatureID(*featureIDStr)
			t.FeatureID = &fid
		}
		if nodeIDStr != nil {
			t.NodeID = *nodeIDStr
		}
		t.IsBlocked = isBlocked == 1
		t.WontDoRequested = wontDoRequested == 1
		t.FilesModified = jsonUnmarshalStrings(filesModifiedJSON)
		t.ContextFiles = jsonUnmarshalStrings(contextFilesJSON)
		t.Tags = jsonUnmarshalStrings(tagsJSON)
		result = append(result, domain.TaskWithDetails{
			Task:              t,
			HasUnresolvedDeps: hasUnresolvedDeps,
			CommentCount:      commentCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *taskRepository) Update(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	filesJSON := jsonMarshal(task.FilesModified)
	contextJSON := jsonMarshal(task.ContextFiles)
	tagsJSON := jsonMarshal(task.Tags)

	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET
			column_id=$1, feature_id=$2, title=$3, summary=$4, description=$5,
			priority=$6, priority_score=$7, position=$8,
			created_by_role=$9, created_by_agent=$10, assigned_role=$11,
			is_blocked=$12, blocked_reason=$13, blocked_at=$14, blocked_by_agent=$15,
			wont_do_requested=$16, wont_do_reason=$17, wont_do_requested_by=$18, wont_do_requested_at=$19,
			completion_summary=$20, completed_by_agent=$21, completed_at=$22,
			files_modified=$23, resolution=$24, context_files=$25, tags=$26, estimated_effort=$27,
			seen_at=$28, session_id=$29, node_id=$30,
			input_tokens=$31, output_tokens=$32, cache_read_tokens=$33, cache_write_tokens=$34, model=$35,
			cold_start_input_tokens=$36, cold_start_output_tokens=$37, cold_start_cache_read_tokens=$38, cold_start_cache_write_tokens=$39,
			started_at=$40, duration_seconds=$41, human_estimate_seconds=$42,
			updated_at=$43
		WHERE project_id=$44 AND id=$45`,
		string(task.ColumnID), featureIDOrNil(task.FeatureID), task.Title, task.Summary, task.Description,
		string(task.Priority), task.PriorityScore, task.Position,
		task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
		boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
		boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
		task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
		filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
		task.SeenAt, task.SessionID, nullableString(task.NodeID),
		task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
		task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
		task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
		task.UpdatedAt,
		string(projectID), string(task.ID),
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(id))
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) Move(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET column_id=$1, updated_at=NOW()
		WHERE project_id=$2 AND id=$3`,
		string(targetColumnID), string(projectID), string(taskID),
	)
	if err != nil {
		return fmt.Errorf("move task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) CountByColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks WHERE project_id=$1 AND column_id=$2`,
		string(projectID), string(columnID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by column: %w", err)
	}
	return count, nil
}

func (r *taskRepository) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	query := `
		SELECT ` + taskSelectCols + `
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.project_id = t.project_id
		WHERE t.project_id = $1
		  AND c.slug = 'todo'
		  AND t.is_blocked = 0
		  AND t.wont_do_requested = 0
		  AND NOT EXISTS (
			  SELECT 1 FROM task_dependencies td
			  JOIN tasks dep ON dep.id = td.depends_on_task_id
			  JOIN columns dc ON dc.id = dep.column_id
			  WHERE td.task_id = t.id AND dc.slug != 'done'
		  )`

	args := []any{string(projectID)}
	if role != "" {
		query += ` AND (t.assigned_role = $2 OR t.assigned_role = '')`
		args = append(args, role)
	}
	query += ` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT 1`

	row := r.pool.QueryRow(ctx, query, args...)
	task, err := scanTask(row)
	if errors.Is(err, domain.ErrTaskNotFound) {
		return nil, domain.ErrNoTasksAvailable
	}
	return task, err
}

func (r *taskRepository) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int) ([]domain.Task, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	query := `
		SELECT ` + taskSelectCols + `
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.project_id = t.project_id
		WHERE t.project_id = $1
		  AND c.slug = 'todo'
		  AND t.is_blocked = 0
		  AND t.wont_do_requested = 0
		  AND NOT EXISTS (
			  SELECT 1 FROM task_dependencies td
			  JOIN tasks dep ON dep.id = td.depends_on_task_id
			  JOIN columns dc ON dc.id = dep.column_id
			  WHERE td.task_id = t.id AND dc.slug != 'done'
		  )`

	args := []any{string(projectID)}
	if role != "" {
		query += ` AND (t.assigned_role = $2 OR t.assigned_role = '')`
		args = append(args, role)
		query += fmt.Sprintf(` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT $%d`, 3)
	} else {
		query += fmt.Sprintf(` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT $%d`, 2)
	}
	args = append(args, count)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get next tasks: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *taskRepository) HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM task_dependencies td
		JOIN tasks dep ON dep.id = td.depends_on_task_id
		JOIN columns dc ON dc.id = dep.column_id
		JOIN tasks t ON t.id = td.task_id
		WHERE td.task_id = $1 AND dc.slug != 'done' AND t.project_id = $2`,
		string(taskID), string(projectID)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has unresolved deps: %w", err)
	}
	return count > 0, nil
}

func (r *taskRepository) GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT `+taskSelectCols+`
		FROM tasks t
		JOIN task_dependencies td ON td.task_id = t.id
		JOIN columns c ON c.id = t.column_id
		WHERE td.depends_on_task_id = $1 AND c.slug != 'done' AND t.project_id = $2`,
		string(taskID), string(projectID))
	if err != nil {
		return nil, fmt.Errorf("get dependents not done: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *taskRepository) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET seen_at = NOW(), seen_by_human = TRUE, updated_at = NOW()
		WHERE project_id = $1 AND id = $2 AND seen_at IS NULL`,
		string(projectID), string(taskID),
	)
	if err != nil {
		return fmt.Errorf("mark task seen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var count int
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(taskID)).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			return domain.ErrTaskNotFound
		}
	}
	return nil
}

func (r *taskRepository) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("reorder task: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var columnID string
	err = tx.QueryRow(ctx, `SELECT column_id FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(taskID)).Scan(&columnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaskNotFound
		}
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks SET position = position + 1
		WHERE project_id = $1 AND column_id = $2 AND position >= $3 AND id != $4`,
		string(projectID), columnID, newPosition, string(taskID),
	)
	if err != nil {
		return fmt.Errorf("reorder shift: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks SET position = $1, updated_at = NOW()
		WHERE project_id = $2 AND id = $3`,
		newPosition, string(projectID), string(taskID),
	)
	if err != nil {
		return fmt.Errorf("reorder update: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *taskRepository) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT
			date_series::date AS date,
			COALESCE(created_count, 0),
			COALESCE(completed_count, 0)
		FROM generate_series(
			(NOW() - ($2 || ' days')::interval)::date,
			NOW()::date,
			'1 day'::interval
		) AS date_series
		LEFT JOIN (
			SELECT created_at::date AS d, COUNT(*) AS created_count
			FROM tasks
			WHERE project_id = $1
			GROUP BY created_at::date
		) c ON c.d = date_series::date
		LEFT JOIN (
			SELECT completed_at::date AS d, COUNT(*) AS completed_count
			FROM tasks
			WHERE project_id = $1 AND completed_at IS NOT NULL
			GROUP BY completed_at::date
		) co ON co.d = date_series::date
		ORDER BY date_series ASC`,
		string(projectID), days,
	)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}
	defer rows.Close()

	var result []domain.TimelineEntry
	for rows.Next() {
		var entry domain.TimelineEntry
		var date time.Time
		if err := rows.Scan(&date, &entry.TasksCreated, &entry.TasksCompleted); err != nil {
			return nil, err
		}
		entry.Date = date.Format("2006-01-02")
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (r *taskRepository) UpdateSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		UPDATE tasks SET session_id = $1, updated_at = NOW()
		WHERE project_id = $2 AND id = $3`,
		sessionID, string(projectID), string(taskID),
	)
	return err
}

func (r *taskRepository) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT
			assigned_role,
			COUNT(*) AS count,
			MIN(cold_start_input_tokens) AS min_input,
			MAX(cold_start_input_tokens) AS max_input,
			AVG(cold_start_input_tokens) AS avg_input,
			MIN(cold_start_output_tokens) AS min_output,
			MAX(cold_start_output_tokens) AS max_output,
			AVG(cold_start_output_tokens) AS avg_output,
			MIN(cold_start_cache_read_tokens) AS min_cache_read,
			MAX(cold_start_cache_read_tokens) AS max_cache_read,
			AVG(cold_start_cache_read_tokens) AS avg_cache_read
		FROM tasks
		WHERE project_id = $1
		  AND cold_start_input_tokens > 0
		GROUP BY assigned_role
		ORDER BY count DESC`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get cold start stats: %w", err)
	}
	defer rows.Close()

	var result []domain.AgentColdStartStat
	for rows.Next() {
		var stat domain.AgentColdStartStat
		err := rows.Scan(
			&stat.AssignedRole, &stat.Count,
			&stat.MinInputTokens, &stat.MaxInputTokens, &stat.AvgInputTokens,
			&stat.MinOutputTokens, &stat.MaxOutputTokens, &stat.AvgOutputTokens,
			&stat.MinCacheReadTokens, &stat.MaxCacheReadTokens, &stat.AvgCacheReadTokens,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

func (r *taskRepository) BulkCreate(ctx context.Context, projectID domain.ProjectID, taskList []domain.Task) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, task := range taskList {
		filesJSON := jsonMarshal(task.FilesModified)
		contextJSON := jsonMarshal(task.ContextFiles)
		tagsJSON := jsonMarshal(task.Tags)

		_, err := tx.Exec(ctx, `
			INSERT INTO tasks (
				id, project_id, column_id, feature_id, title, summary, description,
				priority, priority_score, position,
				created_by_role, created_by_agent, assigned_role,
				is_blocked, blocked_reason, blocked_at, blocked_by_agent,
				wont_do_requested, wont_do_reason, wont_do_requested_by, wont_do_requested_at,
				completion_summary, completed_by_agent, completed_at,
				files_modified, resolution, context_files, tags, estimated_effort,
				seen_at, session_id,
				input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model,
				cold_start_input_tokens, cold_start_output_tokens, cold_start_cache_read_tokens, cold_start_cache_write_tokens,
				started_at, duration_seconds, human_estimate_seconds,
				created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				$8,$9,$10,
				$11,$12,$13,
				$14,$15,$16,$17,
				$18,$19,$20,$21,
				$22,$23,$24,
				$25,$26,$27,$28,$29,
				$30,$31,
				$32,$33,$34,$35,$36,
				$37,$38,$39,$40,
				$41,$42,$43,
				$44,$45
			)`,
			string(task.ID), string(projectID), string(task.ColumnID), featureIDOrNil(task.FeatureID),
			task.Title, task.Summary, task.Description,
			string(task.Priority), task.PriorityScore, task.Position,
			task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
			boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
			boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
			task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
			filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
			task.SeenAt, task.SessionID,
			task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
			task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
			task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
			task.CreatedAt, task.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("bulk create task: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *taskRepository) BulkReassignInProject(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET assigned_role=$1 WHERE project_id=$2 AND assigned_role=$3`,
		newSlug, string(projectID), oldSlug,
	)
	if err != nil {
		return 0, fmt.Errorf("bulk reassign tasks: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *taskRepository) ListByAssignedRole(ctx context.Context, projectID domain.ProjectID, slug string) ([]domain.Task, error) {
	filters := tasks.TaskFilters{AssignedRole: &slug}
	withDetails, err := r.List(ctx, projectID, filters)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, len(withDetails))
	for i, t := range withDetails {
		result[i] = t.Task
	}
	return result, nil
}

func (r *taskRepository) SearchTasks(ctx context.Context, projectID domain.ProjectID, query string, limit int) ([]domain.TaskWithDetails, error) {
	filters := tasks.TaskFilters{
		Search: query,
		Limit:  limit,
	}
	return r.List(ctx, projectID, filters)
}

func (r *taskRepository) GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT
			model,
			COUNT(*) AS task_count,
			SUM(input_tokens) AS input_tokens,
			SUM(output_tokens) AS output_tokens,
			SUM(cache_read_tokens) AS cache_read_tokens,
			SUM(cache_write_tokens) AS cache_write_tokens
		FROM tasks
		WHERE project_id = $1
		  AND model != ''
		  AND (input_tokens > 0 OR output_tokens > 0)
		GROUP BY model
		ORDER BY (SUM(input_tokens) + SUM(output_tokens)) DESC`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get model token stats: %w", err)
	}
	defer rows.Close()

	var result []domain.ModelTokenStat
	for rows.Next() {
		var stat domain.ModelTokenStat
		if err := rows.Scan(&stat.Model, &stat.TaskCount, &stat.InputTokens, &stat.OutputTokens, &stat.CacheReadTokens, &stat.CacheWriteTokens); err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}
