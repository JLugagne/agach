package pg

import (
	"context"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type dependencyRepository struct{ *baseRepository }

func (r *dependencyRepository) Create(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	if dep.TaskID == dep.DependsOnTaskID {
		return domain.ErrCannotDependOnSelf
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_dependencies (id, task_id, depends_on_task_id, created_at)
		VALUES ($1, $2, $3, $4)`,
		string(dep.ID), string(dep.TaskID), string(dep.DependsOnTaskID), dep.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDependencyAlreadyExists
		}
		if isCheckViolation(err) {
			return domain.ErrCannotDependOnSelf
		}
		return fmt.Errorf("create dependency: %w", err)
	}
	return nil
}

func (r *dependencyRepository) Delete(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM task_dependencies WHERE task_id=$1 AND depends_on_task_id=$2`,
		string(taskID), string(dependsOnTaskID),
	)
	if err != nil {
		return fmt.Errorf("delete dependency: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDependencyNotFound
	}
	return nil
}

func (r *dependencyRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE task_id=$1`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	defer rows.Close()
	return scanDependencies(rows)
}

func (r *dependencyRepository) WouldCreateCycle(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error) {
	if taskID == dependsOnTaskID {
		return true, nil
	}

	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE reachable AS (
			SELECT depends_on_task_id AS tid FROM task_dependencies WHERE task_id = $1
			UNION
			SELECT td.depends_on_task_id FROM task_dependencies td
			INNER JOIN reachable r ON td.task_id = r.tid
		)
		SELECT COUNT(*) FROM reachable WHERE tid = $2`,
		string(dependsOnTaskID), string(taskID),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("would create cycle: %w", err)
	}
	return count > 0, nil
}

func (r *dependencyRepository) ListDependents(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE depends_on_task_id=$1`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("list dependents: %w", err)
	}
	defer rows.Close()
	return scanDependencies(rows)
}

func (r *dependencyRepository) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title, t.completion_summary, t.files_modified
		FROM task_dependencies td
		JOIN tasks t ON t.id = td.depends_on_task_id
		JOIN columns c ON c.id = t.column_id
		WHERE td.task_id = $1 AND c.slug = 'done'`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("get dependency context: %w", err)
	}
	defer rows.Close()

	var result []domain.DependencyContext
	for rows.Next() {
		var dc domain.DependencyContext
		var filesJSON []byte
		err := rows.Scan((*string)(&dc.TaskID), &dc.Title, &dc.CompletionSummary, &filesJSON)
		if err != nil {
			return nil, err
		}
		dc.FilesModified = jsonUnmarshalStrings(filesJSON)
		result = append(result, dc)
	}
	return result, rows.Err()
}

func scanDependencies(rows pgx.Rows) ([]domain.TaskDependency, error) {
	var result []domain.TaskDependency
	for rows.Next() {
		var dep domain.TaskDependency
		err := rows.Scan(
			(*string)(&dep.ID), (*string)(&dep.TaskID), (*string)(&dep.DependsOnTaskID), &dep.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dep)
	}
	return result, rows.Err()
}
