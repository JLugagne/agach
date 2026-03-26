package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type projectRepository struct{ *baseRepository }

func (r *projectRepository) Create(ctx context.Context, p domain.Project) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var parentID *string
	if p.ParentID != nil {
		s := string(*p.ParentID)
		parentID = &s
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create project: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, parent_id, name, description, created_by_role, created_by_agent, default_role, git_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		string(p.ID), parentID, p.Name, p.Description,
		p.CreatedByRole, p.CreatedByAgent, p.DefaultRole, p.GitURL,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	colDefs := []struct {
		slug     domain.ColumnSlug
		name     string
		position int
	}{
		{domain.ColumnTodo, "To Do", 0},
		{domain.ColumnInProgress, "In Progress", 1},
		{domain.ColumnDone, "Done", 2},
		{domain.ColumnBlocked, "Blocked", 3},
	}
	for _, cd := range colDefs {
		colID := string(domain.NewColumnID())
		_, err := tx.Exec(ctx, `
			INSERT INTO columns (id, project_id, slug, name, position, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())`,
			colID, string(p.ID), string(cd.slug), cd.name, cd.position,
		)
		if err != nil {
			return fmt.Errorf("create default column %s: %w", cd.slug, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *projectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
		FROM projects WHERE id = $1`, string(id))
	return scanProject(row)
}

func (r *projectRepository) List(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var rows pgx.Rows
	var err error
	if parentID == nil {
		rows, err = r.pool.Query(ctx, `
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
			FROM projects WHERE parent_id IS NULL ORDER BY created_at ASC`)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
			FROM projects WHERE parent_id = $1 ORDER BY created_at ASC`, string(*parentID))
	}
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (r *projectRepository) GetTree(ctx context.Context, id domain.ProjectID) ([]domain.Project, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,'') AS git_url, dockerfile_id, created_at, updated_at
			FROM projects WHERE id = $1
			UNION ALL
			SELECT p.id, p.parent_id, p.name, p.description, p.created_by_role, p.created_by_agent, p.default_role, COALESCE(p.git_url,''), p.dockerfile_id, p.created_at, p.updated_at
			FROM projects p
			INNER JOIN tree t ON p.parent_id = t.id
		)
		SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, git_url, dockerfile_id, created_at, updated_at FROM tree`,
		string(id),
	)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (r *projectRepository) Update(ctx context.Context, p domain.Project) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var parentID *string
	if p.ParentID != nil {
		s := string(*p.ParentID)
		parentID = &s
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects SET parent_id=$1, name=$2, description=$3, created_by_role=$4, created_by_agent=$5, default_role=$6, git_url=$7, updated_at=$8
		WHERE id=$9`,
		parentID, p.Name, p.Description, p.CreatedByRole, p.CreatedByAgent, p.DefaultRole, p.GitURL, p.UpdatedAt, string(p.ID),
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (r *projectRepository) Delete(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM projects WHERE id = $1
			UNION ALL
			SELECT p.id FROM projects p INNER JOIN tree t ON p.parent_id = t.id
		)
		SELECT id FROM tree`, string(id))
	if err != nil {
		return nil, fmt.Errorf("collect project tree: %w", err)
	}
	var ids []domain.ProjectID
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, domain.ProjectID(pid))
	}
	rows.Close()

	if len(ids) == 0 {
		return nil, domain.ErrProjectNotFound
	}

	_, err = r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, string(id))
	if err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}
	return ids, nil
}

func (r *projectRepository) GetSummary(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT c.slug, COUNT(t.id) as cnt
		FROM columns c
		LEFT JOIN tasks t ON t.column_id = c.id AND t.project_id = $1
		WHERE c.project_id = $1
		GROUP BY c.slug`, string(id))
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	defer rows.Close()

	summary := &domain.ProjectSummary{}
	for rows.Next() {
		var slug string
		var cnt int
		if err := rows.Scan(&slug, &cnt); err != nil {
			return nil, err
		}
		switch domain.ColumnSlug(slug) {
		case domain.ColumnBacklog:
			summary.BacklogCount = cnt
		case domain.ColumnTodo:
			summary.TodoCount = cnt
		case domain.ColumnInProgress:
			summary.InProgressCount = cnt
		case domain.ColumnDone:
			summary.DoneCount = cnt
		case domain.ColumnBlocked:
			summary.BlockedCount = cnt
		}
	}
	return summary, nil
}

func (r *projectRepository) CountChildren(ctx context.Context, id domain.ProjectID) (int, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE parent_id = $1`, string(id)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count children: %w", err)
	}
	return count, nil
}

func (r *projectRepository) ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error) {
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

func scanProjectInto(s scanner) (domain.Project, error) {
	var p domain.Project
	var parentID *string
	var dockerfileID *string
	err := s.Scan(
		(*string)(&p.ID), &parentID, &p.Name, &p.Description,
		&p.CreatedByRole, &p.CreatedByAgent, &p.DefaultRole, &p.GitURL,
		&dockerfileID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return domain.Project{}, err
	}
	if parentID != nil {
		pid := domain.ProjectID(*parentID)
		p.ParentID = &pid
	}
	if dockerfileID != nil {
		did := domain.DockerfileID(*dockerfileID)
		p.DockerfileID = &did
	}
	return p, nil
}

func scanProject(row pgx.Row) (*domain.Project, error) {
	p, err := scanProjectInto(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return &p, nil
}

func scanProjects(rows pgx.Rows) ([]domain.Project, error) {
	var result []domain.Project
	for rows.Next() {
		p, err := scanProjectInto(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
