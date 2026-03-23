package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/jackc/pgx/v5"
)

type notificationRepository struct{ *baseRepository }

func (r *notificationRepository) Create(ctx context.Context, notification domain.Notification) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, project_id, severity, title, text, link_url, link_text, link_style, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		string(notification.ID), string(notification.ProjectID), string(notification.Severity),
		notification.Title, notification.Text,
		notification.LinkURL, notification.LinkText, notification.LinkStyle,
		notification.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (r *notificationRepository) FindByID(ctx context.Context, id domain.NotificationID) (*domain.Notification, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, severity, title, text, link_url, link_text, link_style, read_at, created_at
		FROM notifications WHERE id = $1`, string(id))
	var n domain.Notification
	err := row.Scan(
		(*string)(&n.ID), (*string)(&n.ProjectID), (*string)(&n.Severity),
		&n.Title, &n.Text, &n.LinkURL, &n.LinkText, &n.LinkStyle,
		&n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("find notification by id: %w", err)
	}
	return &n, nil
}

func (r *notificationRepository) List(ctx context.Context, projectID domain.ProjectID, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	query := `SELECT id, project_id, severity, title, text, link_url, link_text, link_style, read_at, created_at
		FROM notifications WHERE project_id = $1`
	args := []any{string(projectID)}

	if unreadOnly {
		query += ` AND read_at IS NULL`
	}

	query += ` ORDER BY created_at DESC`

	if limit > 0 {
		args = append(args, limit)
		query += fmt.Sprintf(` LIMIT $%d`, len(args))
	}
	if offset > 0 {
		args = append(args, offset)
		query += fmt.Sprintf(` OFFSET $%d`, len(args))
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var result []domain.Notification
	for rows.Next() {
		var n domain.Notification
		err := rows.Scan(
			(*string)(&n.ID), (*string)(&n.ProjectID), (*string)(&n.Severity),
			&n.Title, &n.Text, &n.LinkURL, &n.LinkText, &n.LinkStyle,
			&n.ReadAt, &n.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications rows: %w", err)
	}
	return result, nil
}

func (r *notificationRepository) UnreadCount(ctx context.Context, projectID domain.ProjectID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE project_id = $1 AND read_at IS NULL`,
		string(projectID),
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("unread count: %w", err)
	}
	return count, nil
}

func (r *notificationRepository) MarkRead(ctx context.Context, id domain.NotificationID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND read_at IS NULL`,
		string(id),
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	// Also check if the notification exists at all (might already be read)
	if tag.RowsAffected() == 0 {
		// Check if notification exists
		var exists bool
		err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1)`, string(id)).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check notification exists: %w", err)
		}
		if !exists {
			return domain.ErrNotificationNotFound
		}
		// Already read, that's fine
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(ctx context.Context, projectID domain.ProjectID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE project_id = $1 AND read_at IS NULL`,
		string(projectID),
	)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id domain.NotificationID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}
