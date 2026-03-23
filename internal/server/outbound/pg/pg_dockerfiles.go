package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type dockerfileRepository struct{ *baseRepository }

func (r *dockerfileRepository) Create(ctx context.Context, d domain.Dockerfile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dockerfiles (id, slug, name, description, version, content, is_latest, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		string(d.ID), d.Slug, d.Name, d.Description, d.Version,
		d.Content, d.IsLatest, d.SortOrder, d.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDockerfileAlreadyExists
		}
		return fmt.Errorf("create dockerfile: %w", err)
	}
	return nil
}

func (r *dockerfileRepository) FindByID(ctx context.Context, id domain.DockerfileID) (*domain.Dockerfile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, version, content, is_latest, sort_order, created_at, updated_at
		FROM dockerfiles WHERE id = $1`, string(id))
	return scanDockerfile(row)
}

func (r *dockerfileRepository) FindBySlug(ctx context.Context, slug string) (*domain.Dockerfile, error) {
	// Return latest version; if no latest, return most recently created
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, version, content, is_latest, sort_order, created_at, updated_at
		FROM dockerfiles WHERE slug = $1
		ORDER BY is_latest DESC, created_at DESC LIMIT 1`, slug)
	return scanDockerfile(row)
}

func (r *dockerfileRepository) FindBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, version, content, is_latest, sort_order, created_at, updated_at
		FROM dockerfiles WHERE slug = $1 AND version = $2`, slug, version)
	return scanDockerfile(row)
}

func (r *dockerfileRepository) List(ctx context.Context) ([]domain.Dockerfile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, description, version, content, is_latest, sort_order, created_at, updated_at
		FROM dockerfiles ORDER BY slug ASC, sort_order ASC, version ASC`)
	if err != nil {
		return nil, fmt.Errorf("list dockerfiles: %w", err)
	}
	defer rows.Close()
	return scanDockerfiles(rows)
}

func (r *dockerfileRepository) Update(ctx context.Context, d domain.Dockerfile) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE dockerfiles SET name=$1, description=$2, content=$3, is_latest=$4, sort_order=$5, updated_at=$6
		WHERE id=$7`,
		d.Name, d.Description, d.Content, d.IsLatest, d.SortOrder, d.UpdatedAt, string(d.ID),
	)
	if err != nil {
		return fmt.Errorf("update dockerfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDockerfileNotFound
	}
	return nil
}

func (r *dockerfileRepository) Delete(ctx context.Context, id domain.DockerfileID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM dockerfiles WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete dockerfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDockerfileNotFound
	}
	return nil
}

func (r *dockerfileRepository) IsInUse(ctx context.Context, id domain.DockerfileID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects WHERE dockerfile_id = $1`, string(id),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check dockerfile in use: %w", err)
	}
	return count > 0, nil
}

func (r *dockerfileRepository) SetLatest(ctx context.Context, id domain.DockerfileID) error {
	// Get the slug of the target dockerfile
	var slug string
	err := r.pool.QueryRow(ctx, `SELECT slug FROM dockerfiles WHERE id = $1`, string(id)).Scan(&slug)
	if err != nil {
		return fmt.Errorf("set latest dockerfile: %w", err)
	}

	// Clear is_latest on all versions with same slug, then set on this one
	_, err = r.pool.Exec(ctx, `
		UPDATE dockerfiles SET is_latest = (id = $1) WHERE slug = $2`,
		string(id), slug,
	)
	if err != nil {
		return fmt.Errorf("set latest dockerfile: %w", err)
	}
	return nil
}

func (r *dockerfileRepository) GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT d.id, d.slug, d.name, d.description, d.version, d.content, d.is_latest, d.sort_order, d.created_at, d.updated_at
		FROM dockerfiles d
		JOIN projects p ON p.dockerfile_id = d.id
		WHERE p.id = $1`, string(projectID))
	d, err := scanDockerfile(row)
	if err != nil {
		if errors.Is(err, domain.ErrDockerfileNotFound) {
			return nil, nil // no dockerfile assigned
		}
		return nil, err
	}
	return d, nil
}

func (r *dockerfileRepository) SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET dockerfile_id = $1, updated_at = NOW() WHERE id = $2`,
		string(dockerfileID), string(projectID),
	)
	if err != nil {
		return fmt.Errorf("set project dockerfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (r *dockerfileRepository) ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE projects SET dockerfile_id = NULL, updated_at = NOW() WHERE id = $1`,
		string(projectID),
	)
	if err != nil {
		return fmt.Errorf("clear project dockerfile: %w", err)
	}
	return nil
}

func scanDockerfile(row pgx.Row) (*domain.Dockerfile, error) {
	var d domain.Dockerfile
	var id string
	err := row.Scan(
		&id, &d.Slug, &d.Name, &d.Description, &d.Version,
		&d.Content, &d.IsLatest, &d.SortOrder, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDockerfileNotFound
		}
		return nil, fmt.Errorf("scan dockerfile: %w", err)
	}
	d.ID = domain.DockerfileID(id)
	return &d, nil
}

func scanDockerfiles(rows pgx.Rows) ([]domain.Dockerfile, error) {
	var result []domain.Dockerfile
	for rows.Next() {
		var d domain.Dockerfile
		var id string
		err := rows.Scan(
			&id, &d.Slug, &d.Name, &d.Description, &d.Version,
			&d.Content, &d.IsLatest, &d.SortOrder, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan dockerfiles: %w", err)
		}
		d.ID = domain.DockerfileID(id)
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dockerfiles: %w", err)
	}
	return result, nil
}
