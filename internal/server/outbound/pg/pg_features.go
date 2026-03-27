package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type featureRepository struct{ *baseRepository }

func (r *featureRepository) Create(ctx context.Context, feature domain.Feature) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO features (id, project_id, name, description, user_changelog, tech_changelog, status, created_by_role, created_by_agent, node_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		string(feature.ID), string(feature.ProjectID), feature.Name, feature.Description,
		feature.UserChangelog, feature.TechChangelog,
		string(feature.Status), feature.CreatedByRole, feature.CreatedByAgent,
		nullableString(feature.NodeID),
		feature.CreatedAt, feature.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create feature: %w", err)
	}
	return nil
}

func (r *featureRepository) FindByID(ctx context.Context, id domain.FeatureID) (*domain.Feature, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, user_changelog, tech_changelog, status, created_by_role, created_by_agent, node_id, created_at, updated_at
		FROM features WHERE id = $1
		-- security: project_id = $2 filter must be added once interface exposes projectID`, string(id))
	var f domain.Feature
	var nodeID *string
	err := row.Scan(
		(*string)(&f.ID), (*string)(&f.ProjectID), &f.Name, &f.Description,
		&f.UserChangelog, &f.TechChangelog,
		(*string)(&f.Status), &f.CreatedByRole, &f.CreatedByAgent, &nodeID, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFeatureNotFound
		}
		return nil, fmt.Errorf("find feature by id: %w", err)
	}
	if nodeID != nil {
		f.NodeID = *nodeID
	}
	return &f, nil
}

func (r *featureRepository) List(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	args := []any{string(projectID)}
	query := `
		SELECT f.id, f.project_id, f.name, f.description, f.user_changelog, f.tech_changelog, f.status, f.created_by_role, f.created_by_agent, f.node_id, f.created_at, f.updated_at,
			COALESCE(SUM(CASE WHEN c.slug = 'backlog'     THEN 1 ELSE 0 END), 0) AS backlog_count,
			COALESCE(SUM(CASE WHEN c.slug = 'todo'        THEN 1 ELSE 0 END), 0) AS todo_count,
			COALESCE(SUM(CASE WHEN c.slug = 'in_progress' THEN 1 ELSE 0 END), 0) AS in_progress_count,
			COALESCE(SUM(CASE WHEN c.slug = 'done'        THEN 1 ELSE 0 END), 0) AS done_count,
			COALESCE(SUM(CASE WHEN c.slug = 'blocked'     THEN 1 ELSE 0 END), 0) AS blocked_count
		FROM features f
		LEFT JOIN tasks t ON t.feature_id = f.id
		LEFT JOIN columns c ON c.id = t.column_id
		WHERE f.project_id = $1`

	if len(statusFilter) > 0 {
		statuses := make([]string, len(statusFilter))
		for i, s := range statusFilter {
			statuses[i] = string(s)
		}
		args = append(args, statuses)
		query += fmt.Sprintf(` AND f.status = ANY($%d)`, len(args))
	}
	query += ` GROUP BY f.id ORDER BY f.created_at ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	defer rows.Close()

	var result []domain.FeatureWithTaskSummary
	for rows.Next() {
		var fw domain.FeatureWithTaskSummary
		var nodeID *string
		err := rows.Scan(
			(*string)(&fw.ID), (*string)(&fw.ProjectID), &fw.Name, &fw.Description,
			&fw.UserChangelog, &fw.TechChangelog,
			(*string)(&fw.Status), &fw.CreatedByRole, &fw.CreatedByAgent, &nodeID, &fw.CreatedAt, &fw.UpdatedAt,
			&fw.TaskSummary.BacklogCount,
			&fw.TaskSummary.TodoCount,
			&fw.TaskSummary.InProgressCount,
			&fw.TaskSummary.DoneCount,
			&fw.TaskSummary.BlockedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feature row: %w", err)
		}
		if nodeID != nil {
			fw.NodeID = *nodeID
		}
		result = append(result, fw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list features rows: %w", err)
	}
	return result, nil
}

func (r *featureRepository) Update(ctx context.Context, feature domain.Feature) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE features SET name=$1, description=$2, updated_at=$3
		WHERE id=$4 AND project_id=$5`,
		feature.Name, feature.Description, feature.UpdatedAt, string(feature.ID), string(feature.ProjectID),
	)
	if err != nil {
		return fmt.Errorf("update feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) UpdateStatus(ctx context.Context, id domain.FeatureID, status domain.FeatureStatus, nodeID string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	// security: project_id filter must be added once interface exposes projectID
	tag, err := r.pool.Exec(ctx, `
		UPDATE features SET status=$1, node_id=COALESCE($3, node_id), updated_at=NOW()
		WHERE id=$2`,
		string(status), string(id), nullableString(nodeID),
	)
	if err != nil {
		return fmt.Errorf("update feature status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) Delete(ctx context.Context, id domain.FeatureID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	// security: project_id filter must be added once interface exposes projectID
	tag, err := r.pool.Exec(ctx, `DELETE FROM features WHERE id=$1`, string(id))
	if err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) GetStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM features WHERE project_id=$1 GROUP BY status`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get feature stats: %w", err)
	}
	defer rows.Close()

	stats := &domain.FeatureStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan feature stats: %w", err)
		}
		stats.TotalCount += count
		switch domain.FeatureStatus(status) {
		case domain.FeatureStatusDraft:
			stats.NotReadyCount += count
		case domain.FeatureStatusReady:
			stats.ReadyCount += count
		case domain.FeatureStatusInProgress:
			stats.InProgressCount += count
		case domain.FeatureStatusDone:
			stats.DoneCount += count
		case domain.FeatureStatusBlocked:
			stats.BlockedCount += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("feature stats rows: %w", err)
	}
	return stats, nil
}

func (r *featureRepository) UpdateChangelogs(ctx context.Context, id domain.FeatureID, userChangelog, techChangelog *string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	// security: project_id filter must be added once interface exposes projectID
	setClauses := []string{}
	args := []any{}

	if userChangelog != nil {
		args = append(args, *userChangelog)
		setClauses = append(setClauses, fmt.Sprintf("user_changelog=$%d", len(args)))
	}
	if techChangelog != nil {
		args = append(args, *techChangelog)
		setClauses = append(setClauses, fmt.Sprintf("tech_changelog=$%d", len(args)))
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, string(id))
	query := fmt.Sprintf("UPDATE features SET %s, updated_at=NOW() WHERE id=$%d",
		joinStrings(setClauses, ", "), len(args))

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update feature changelogs: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) ListTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	// security: project_id filter must be added once interface exposes projectID
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, completion_summary, completed_by_agent, completed_at, files_modified,
		       duration_seconds, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model
		FROM tasks
		WHERE feature_id = $1 AND completed_at IS NOT NULL
		ORDER BY completed_at ASC`,
		string(featureID),
	)
	if err != nil {
		return nil, fmt.Errorf("list task summaries: %w", err)
	}
	defer rows.Close()

	var result []domain.FeatureTaskSummary
	for rows.Next() {
		var s domain.FeatureTaskSummary
		var completedAt time.Time
		err := rows.Scan(
			(*string)(&s.ID), &s.Title, &s.CompletionSummary, &s.CompletedByAgent,
			&completedAt, &s.FilesModified,
			&s.DurationSeconds, &s.InputTokens, &s.OutputTokens, &s.CacheReadTokens, &s.CacheWriteTokens, &s.Model,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task summary row: %w", err)
		}
		s.CompletedAt = completedAt
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list task summaries rows: %w", err)
	}
	return result, nil
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
