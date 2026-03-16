package comments

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// CommentRepository defines operations for managing comments within a project
type CommentRepository interface {
	// Create creates a new comment in the specified project's DB
	Create(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error

	// FindByID retrieves a comment by ID from the specified project's DB
	FindByID(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error)

	// List retrieves comments for a task ordered by created_at ASC.
	// If limit > 0, returns at most limit comments starting from offset.
	List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)

	// Update updates a comment (content and edited_at)
	Update(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error

	// Delete deletes a comment
	Delete(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error

	// Count returns the number of comments for a task
	Count(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error)

	// IsLastComment checks if a comment is the last one on its task
	IsLastComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error)
}
