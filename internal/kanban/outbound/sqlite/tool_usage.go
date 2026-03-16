package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// ToolUsageRepository implements toolusage.ToolUsageRepository
type ToolUsageRepository struct {
	*baseRepository
}

func (r *ToolUsageRepository) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		_, err := db.ExecContext(ctx, `
			INSERT INTO tool_usage (tool_name, execution_count, last_executed_at)
			VALUES (?, 1, ?)
			ON CONFLICT(tool_name) DO UPDATE SET
				execution_count = execution_count + 1,
				last_executed_at = ?
		`, toolName, time.Now().UTC(), time.Now().UTC())
		return err
	})
}

func (r *ToolUsageRepository) ListToolUsage(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	var stats []domain.ToolUsageStat

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		rows, err := db.QueryContext(ctx, `
			SELECT tool_name, execution_count, last_executed_at
			FROM tool_usage
			ORDER BY execution_count DESC
		`)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.ToolUsageStat
			var lastExec sql.NullTime
			if err := rows.Scan(&s.ToolName, &s.ExecutionCount, &lastExec); err != nil {
				return err
			}
			if lastExec.Valid {
				s.LastExecutedAt = &lastExec.Time
			}
			stats = append(stats, s)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []domain.ToolUsageStat{}
	}
	return stats, nil
}
