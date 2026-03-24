package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/internal/daemon/domain/repositories/builds"
)

var _ builds.DockerBuildRepository = (*buildRepository)(nil)

const timeFormat = time.RFC3339

type buildRepository struct {
	db *sql.DB
}

// NewBuildRepository creates a new SQLite-backed DockerBuildRepository.
func NewBuildRepository(db *sql.DB) builds.DockerBuildRepository {
	return &buildRepository{db: db}
}

func (r *buildRepository) Create(ctx context.Context, build domain.DockerBuild) error {
	var completedAt *string
	if build.CompletedAt != nil {
		s := build.CompletedAt.Format(timeFormat)
		completedAt = &s
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO builds (id, dockerfile_slug, version, image_hash, image_size, status, build_log, created_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(build.ID), build.DockerfileSlug, build.Version, build.ImageHash, build.ImageSize,
		string(build.Status), build.BuildLog, build.CreatedAt.Format(timeFormat), completedAt,
	)
	return err
}

func (r *buildRepository) FindByID(ctx context.Context, id domain.BuildID) (*domain.DockerBuild, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, dockerfile_slug, version, image_hash, image_size, status, build_log, created_at, completed_at
		 FROM builds WHERE id = ?`, string(id))
	b, err := scanBuild(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *buildRepository) ListByDockerfile(ctx context.Context, slug string) ([]domain.DockerBuild, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, dockerfile_slug, version, image_hash, image_size, status, build_log, created_at, completed_at
		 FROM builds WHERE dockerfile_slug = ? ORDER BY created_at DESC`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBuilds(rows)
}

func (r *buildRepository) ListAll(ctx context.Context) ([]domain.DockerBuild, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, dockerfile_slug, version, image_hash, image_size, status, build_log, created_at, completed_at
		 FROM builds ORDER BY dockerfile_slug, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBuilds(rows)
}

func (r *buildRepository) UpdateStatus(ctx context.Context, id domain.BuildID, status domain.BuildStatus, log string) error {
	now := time.Now().UTC().Format(timeFormat)
	_, err := r.db.ExecContext(ctx,
		`UPDATE builds SET status = ?, build_log = ?, completed_at = ? WHERE id = ?`,
		string(status), log, now, string(id))
	return err
}

func (r *buildRepository) Delete(ctx context.Context, id domain.BuildID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM builds WHERE id = ?`, string(id))
	return err
}

func (r *buildRepository) DeleteNonLatest(ctx context.Context, slug string) (int, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM builds WHERE dockerfile_slug = ? AND id != (
			SELECT id FROM builds WHERE dockerfile_slug = ? ORDER BY created_at DESC LIMIT 1
		)`, slug, slug)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	return int(n), err
}

func scanBuild(row *sql.Row) (*domain.DockerBuild, error) {
	var b domain.DockerBuild
	var id, slug, version, status, createdAtStr string
	var imageHash, buildLog sql.NullString
	var imageSize sql.NullInt64
	var completedAtStr sql.NullString

	if err := row.Scan(&id, &slug, &version, &imageHash, &imageSize, &status, &buildLog, &createdAtStr, &completedAtStr); err != nil {
		return nil, err
	}

	b.ID = domain.BuildID(id)
	b.DockerfileSlug = slug
	b.Version = version
	b.Status = domain.BuildStatus(status)
	if imageHash.Valid {
		b.ImageHash = imageHash.String
	}
	if imageSize.Valid {
		b.ImageSize = imageSize.Int64
	}
	if buildLog.Valid {
		b.BuildLog = buildLog.String
	}
	createdAt, err := time.Parse(timeFormat, createdAtStr)
	if err != nil {
		return nil, err
	}
	b.CreatedAt = createdAt
	if completedAtStr.Valid {
		t, err := time.Parse(timeFormat, completedAtStr.String)
		if err != nil {
			return nil, err
		}
		b.CompletedAt = &t
	}
	return &b, nil
}

func scanBuilds(rows *sql.Rows) ([]domain.DockerBuild, error) {
	var result []domain.DockerBuild
	for rows.Next() {
		var b domain.DockerBuild
		var id, slug, version, status, createdAtStr string
		var imageHash, buildLog sql.NullString
		var imageSize sql.NullInt64
		var completedAtStr sql.NullString

		if err := rows.Scan(&id, &slug, &version, &imageHash, &imageSize, &status, &buildLog, &createdAtStr, &completedAtStr); err != nil {
			return nil, err
		}

		b.ID = domain.BuildID(id)
		b.DockerfileSlug = slug
		b.Version = version
		b.Status = domain.BuildStatus(status)
		if imageHash.Valid {
			b.ImageHash = imageHash.String
		}
		if imageSize.Valid {
			b.ImageSize = imageSize.Int64
		}
		if buildLog.Valid {
			b.BuildLog = buildLog.String
		}
		createdAt, err := time.Parse(timeFormat, createdAtStr)
		if err != nil {
			return nil, err
		}
		b.CreatedAt = createdAt
		if completedAtStr.Valid {
			t, err := time.Parse(timeFormat, completedAtStr.String)
			if err != nil {
				return nil, err
			}
			b.CompletedAt = &t
		}
		result = append(result, b)
	}
	return result, rows.Err()
}
