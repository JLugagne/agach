package pg

import (
	"context"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

type toolUsageRepository struct{ *baseRepository }

func (r *toolUsageRepository) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	id := newID()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tool_usage (id, project_id, tool_name, count, last_used_at)
		VALUES ($1, $2, $3, 1, NOW())
		ON CONFLICT (project_id, tool_name) DO UPDATE
		SET count = tool_usage.count + 1, last_used_at = NOW()`,
		id, string(projectID), toolName,
	)
	if err != nil {
		return fmt.Errorf("increment tool usage: %w", err)
	}
	return nil
}

func (r *toolUsageRepository) ListToolUsage(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT tool_name, count, last_used_at
		FROM tool_usage WHERE project_id=$1
		ORDER BY count DESC, tool_name ASC`,
		string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list tool usage: %w", err)
	}
	defer rows.Close()

	var result []domain.ToolUsageStat
	for rows.Next() {
		var stat domain.ToolUsageStat
		err := rows.Scan(&stat.ToolName, &stat.ExecutionCount, &stat.LastExecutedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

type modelPricingRepository struct{ *baseRepository }

func (r *modelPricingRepository) ListAll(ctx context.Context) ([]domain.ModelPricing, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m, updated_at
		FROM model_pricing
		ORDER BY model_id`)
	if err != nil {
		return nil, fmt.Errorf("list model pricing: %w", err)
	}
	defer rows.Close()

	var result []domain.ModelPricing
	for rows.Next() {
		var p domain.ModelPricing
		if err := rows.Scan(&p.ID, &p.ModelID, &p.InputPricePer1M, &p.OutputPricePer1M, &p.CacheReadPricePer1M, &p.CacheWritePricePer1M, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *modelPricingRepository) Upsert(ctx context.Context, p domain.ModelPricing) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO model_pricing (id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (model_id) DO UPDATE SET
			input_price_per_1m = EXCLUDED.input_price_per_1m,
			output_price_per_1m = EXCLUDED.output_price_per_1m,
			cache_read_price_per_1m = EXCLUDED.cache_read_price_per_1m,
			cache_write_price_per_1m = EXCLUDED.cache_write_price_per_1m,
			updated_at = NOW()`,
		p.ID, p.ModelID, p.InputPricePer1M, p.OutputPricePer1M, p.CacheReadPricePer1M, p.CacheWritePricePer1M,
	)
	return err
}
