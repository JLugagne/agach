package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type columnRepository struct{ *baseRepository }

func (r *columnRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1)`, string(projectID)).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrProjectNotFound
	}

	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND id=$2`,
		string(projectID), string(id))
	return scanColumn(row)
}

func (r *columnRepository) FindBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug=$2`,
		string(projectID), string(slug))
	col, err := scanColumn(row)
	if errors.Is(err, domain.ErrColumnNotFound) {
		return nil, domain.ErrColumnNotFound
	}
	return col, err
}

func (r *columnRepository) List(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 ORDER BY position ASC`,
		string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var result []domain.Column
	for rows.Next() {
		col, err := scanColumnRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *col)
	}
	return result, rows.Err()
}

func (r *columnRepository) EnsureBacklog(ctx context.Context, projectID domain.ProjectID) (*domain.Column, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug='backlog'`,
		string(projectID))
	col, err := scanColumn(row)
	if err == nil {
		return col, nil
	}
	if !errors.Is(err, domain.ErrColumnNotFound) {
		return nil, err
	}

	colID := string(domain.NewColumnID())
	_, err = r.pool.Exec(ctx, `
		INSERT INTO columns (id, project_id, slug, name, position, created_at)
		VALUES ($1, $2, 'backlog', 'Backlog', -1, NOW())
		ON CONFLICT (project_id, slug) DO NOTHING`,
		colID, string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("ensure backlog: %w", err)
	}

	row = r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug='backlog'`,
		string(projectID))
	return scanColumn(row)
}

func scanColumn(row pgx.Row) (*domain.Column, error) {
	var col domain.Column
	err := row.Scan((*string)(&col.ID), (*string)(&col.Slug), &col.Name, &col.Position, &col.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrColumnNotFound
		}
		return nil, err
	}
	return &col, nil
}

func scanColumnRow(rows pgx.Rows) (*domain.Column, error) {
	var col domain.Column
	err := rows.Scan((*string)(&col.ID), (*string)(&col.Slug), &col.Name, &col.Position, &col.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &col, nil
}
