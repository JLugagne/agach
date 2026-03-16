package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Create creates a new comment in a project database
func (r *CommentRepository) Create(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			INSERT INTO comments (id, task_id, author_role, author_name, author_type, content, edited_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err := db.ExecContext(ctx, query,
			string(comment.ID),
			string(comment.TaskID),
			comment.AuthorRole,
			comment.AuthorName,
			comment.AuthorType,
			comment.Content,
			timeToNullTime(comment.EditedAt),
			comment.CreatedAt,
		)

		return err
	})
}

// FindByID retrieves a comment by ID from a project database
func (r *CommentRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error) {
	var comment *domain.Comment

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
			FROM comments
			WHERE id = ?
		`

		var c domain.Comment
		var editedAt sql.NullTime
		var createdAt time.Time

		err := db.QueryRowContext(ctx, query, string(id)).Scan(
			&c.ID,
			&c.TaskID,
			&c.AuthorRole,
			&c.AuthorName,
			&c.AuthorType,
			&c.Content,
			&editedAt,
			&createdAt,
		)

		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrCommentNotFound, err)
			}
			return err
		}

		c.CreatedAt = createdAt
		if editedAt.Valid {
			c.EditedAt = &editedAt.Time
		}

		comment = &c
		return nil
	})

	if err != nil {
		return nil, err
	}

	return comment, nil
}

// List retrieves comments for a task, ordered by created_at ASC.
// If limit > 0, returns at most limit comments starting from offset.
func (r *CommentRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	var comments []domain.Comment

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
			FROM comments
			WHERE task_id = ?
			ORDER BY created_at ASC
		`

		var args []interface{}
		args = append(args, string(taskID))

		if limit > 0 {
			query += " LIMIT ?"
			args = append(args, limit)
			if offset > 0 {
				query += " OFFSET ?"
				args = append(args, offset)
			}
		}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var c domain.Comment
			var editedAt sql.NullTime
			var createdAt time.Time

			err := rows.Scan(
				&c.ID,
				&c.TaskID,
				&c.AuthorRole,
				&c.AuthorName,
				&c.AuthorType,
				&c.Content,
				&editedAt,
				&createdAt,
			)

			if err != nil {
				return err
			}

			c.CreatedAt = createdAt
			if editedAt.Valid {
				c.EditedAt = &editedAt.Time
			}

			comments = append(comments, c)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return comments, nil
}

// Update updates an existing comment in a project database
func (r *CommentRepository) Update(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			UPDATE comments
			SET content = ?, edited_at = ?
			WHERE id = ?
		`

		result, err := db.ExecContext(ctx, query,
			comment.Content,
			time.Now(),
			string(comment.ID),
		)

		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return domain.ErrCommentNotFound
		}

		return nil
	})
}

// Delete deletes a comment from a project database
func (r *CommentRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `DELETE FROM comments WHERE id = ?`

		result, err := db.ExecContext(ctx, query, string(id))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return domain.ErrCommentNotFound
		}

		return nil
	})
}

// Count counts the number of comments for a task
func (r *CommentRepository) Count(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	var count int

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `SELECT COUNT(*) FROM comments WHERE task_id = ?`
		return db.QueryRowContext(ctx, query, string(taskID)).Scan(&count)
	})

	return count, err
}

// IsLastComment checks if a comment is the last comment for its task
func (r *CommentRepository) IsLastComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error) {
	var isLast bool

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT CASE
				WHEN c.created_at = (
					SELECT MAX(created_at)
					FROM comments
					WHERE task_id = c.task_id
				) THEN 1
				ELSE 0
			END
			FROM comments c
			WHERE c.id = ?
		`

		var result int
		err := db.QueryRowContext(ctx, query, string(commentID)).Scan(&result)
		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrCommentNotFound, err)
			}
			return err
		}

		isLast = result == 1
		return nil
	})

	return isLast, err
}
