package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type commentRepository struct{ *baseRepository }

func (r *commentRepository) Create(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO comments (id, task_id, author_role, author_name, author_type, content, edited_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(comment.ID), string(comment.TaskID),
		comment.AuthorRole, comment.AuthorName, string(comment.AuthorType),
		comment.Content, comment.EditedAt, comment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *commentRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
		FROM comments WHERE id = $1`,
		string(id))
	return scanComment(row)
}

func (r *commentRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	query := `
		SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
		FROM comments WHERE task_id = $1 ORDER BY created_at ASC`
	args := []any{string(taskID)}
	argIdx := 2
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argIdx)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var result []domain.Comment
	for rows.Next() {
		var c domain.Comment
		err := rows.Scan(
			(*string)(&c.ID), (*string)(&c.TaskID),
			&c.AuthorRole, &c.AuthorName, (*string)(&c.AuthorType),
			&c.Content, &c.EditedAt, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *commentRepository) Update(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE comments SET content=$1, edited_at=$2 WHERE id=$3`,
		comment.Content, comment.EditedAt, string(comment.ID),
	)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *commentRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id=$1`, string(id))
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *commentRepository) Count(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM comments WHERE task_id=$1`, string(taskID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return count, nil
}

func (r *commentRepository) IsLastComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var taskID string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT task_id, created_at FROM comments WHERE id=$1`, string(commentID)).Scan(&taskID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, domain.ErrCommentNotFound
		}
		return false, err
	}

	var lastID string
	err = r.pool.QueryRow(ctx, `
		SELECT id FROM comments WHERE task_id=$1 ORDER BY created_at DESC LIMIT 1`, taskID).Scan(&lastID)
	if err != nil {
		return false, err
	}
	return lastID == string(commentID), nil
}

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(
		(*string)(&c.ID), (*string)(&c.TaskID),
		&c.AuthorRole, &c.AuthorName, (*string)(&c.AuthorType),
		&c.Content, &c.EditedAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, err
	}
	return &c, nil
}
