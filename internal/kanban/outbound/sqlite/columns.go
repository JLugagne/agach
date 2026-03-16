package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// FindByID retrieves a column by ID from a project database
func (r *ColumnRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
	var column *domain.Column
	var err error

	dbErr := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, position, wip_limit, created_at
			FROM columns
			WHERE id = ?
		`

		var col domain.Column
		var createdAt time.Time

		err = db.QueryRowContext(ctx, query, string(id)).Scan(
			&col.ID,
			&col.Slug,
			&col.Name,
			&col.Position,
			&col.WIPLimit,
			&createdAt,
		)

		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrColumnNotFound, err)
			}
			return err
		}

		col.CreatedAt = createdAt
		column = &col
		return nil
	})

	if dbErr != nil {
		return nil, dbErr
	}

	return column, nil
}

// FindBySlug retrieves a column by slug from a project database
func (r *ColumnRepository) FindBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	var column *domain.Column
	var err error

	dbErr := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, position, wip_limit, created_at
			FROM columns
			WHERE slug = ?
		`

		var col domain.Column
		var createdAt time.Time

		err = db.QueryRowContext(ctx, query, string(slug)).Scan(
			&col.ID,
			&col.Slug,
			&col.Name,
			&col.Position,
			&col.WIPLimit,
			&createdAt,
		)

		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrColumnNotFound, err)
			}
			return err
		}

		col.CreatedAt = createdAt
		column = &col
		return nil
	})

	if dbErr != nil {
		return nil, dbErr
	}

	return column, nil
}

// List retrieves all columns from a project database, ordered by position
func (r *ColumnRepository) List(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	var columns []domain.Column

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, position, wip_limit, created_at
			FROM columns
			ORDER BY position ASC
		`

		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var col domain.Column
			var createdAt time.Time

			err := rows.Scan(
				&col.ID,
				&col.Slug,
				&col.Name,
				&col.Position,
				&col.WIPLimit,
				&createdAt,
			)

			if err != nil {
				return err
			}

			col.CreatedAt = createdAt
			columns = append(columns, col)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return columns, nil
}

// UpdateWIPLimit updates the WIP limit for a column in a project database
func (r *ColumnRepository) UpdateWIPLimit(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID, wipLimit int) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		result, err := db.ExecContext(ctx, `UPDATE columns SET wip_limit = ? WHERE id = ?`, wipLimit, string(columnID))
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return domain.ErrColumnNotFound
		}
		return nil
	})
}
