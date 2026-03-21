package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Comment Commands

func (a *App) CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	if content == "" {
		return domain.Comment{}, domain.ErrCommentContentRequired
	}

	if authorType != domain.AuthorTypeAgent && authorType != domain.AuthorTypeHuman {
		return domain.Comment{}, domain.ErrInvalidCommentData
	}

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return domain.Comment{}, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.Comment{}, domain.ErrTaskNotFound
	}

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     taskID,
		AuthorRole: authorRole,
		AuthorName: authorName,
		AuthorType: authorType,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	if err := a.comments.Create(ctx, projectID, comment); err != nil {
		logger.WithError(err).Error("failed to create comment")
		return domain.Comment{}, err
	}

	logger.WithField("commentID", comment.ID).Info("comment created successfully")
	return comment, nil
}

func (a *App) UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"commentID": commentID,
	})

	if content == "" {
		return domain.ErrCommentContentRequired
	}

	comment, err := a.comments.FindByID(ctx, projectID, commentID)
	if err != nil {
		logger.WithError(err).Error("failed to find comment")
		return errors.Join(domain.ErrCommentNotFound, err)
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	comment.Content = content
	now := time.Now()
	comment.EditedAt = &now

	if err := a.comments.Update(ctx, projectID, *comment); err != nil {
		logger.WithError(err).Error("failed to update comment")
		return err
	}

	logger.Info("comment updated successfully")
	return nil
}

func (a *App) DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"commentID": commentID,
	})

	// Verify comment exists
	comment, err := a.comments.FindByID(ctx, projectID, commentID)
	if err != nil {
		logger.WithError(err).Error("failed to find comment")
		return errors.Join(domain.ErrCommentNotFound, err)
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}

	if err := a.comments.Delete(ctx, projectID, commentID); err != nil {
		logger.WithError(err).Error("failed to delete comment")
		return err
	}

	logger.Info("comment deleted successfully")
	return nil
}

// Comment Queries

func (a *App) GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"commentID": commentID,
	})

	comment, err := a.comments.FindByID(ctx, projectID, commentID)
	if err != nil {
		logger.WithError(err).Error("failed to get comment")
		return nil, errors.Join(domain.ErrCommentNotFound, err)
	}
	if comment == nil {
		return nil, domain.ErrCommentNotFound
	}

	return comment, nil
}

func (a *App) CountComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	return a.comments.Count(ctx, projectID, taskID)
}

func (a *App) ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	comments, err := a.comments.List(ctx, projectID, taskID, limit, offset)
	if err != nil {
		logger.WithError(err).Error("failed to list comments")
		return nil, err
	}

	return comments, nil
}
