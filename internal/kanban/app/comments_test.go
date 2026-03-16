package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Comment Command Tests

func TestApp_CreateComment_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: columnID, Title: "Test Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	comment, err := a.CreateComment(ctx, projectID, taskID, "developer", "Agent 1", domain.AuthorTypeAgent, "This is a comment")

	require.NoError(t, err)
	assert.NotEmpty(t, comment.ID)
	assert.Equal(t, taskID, comment.TaskID)
	assert.Equal(t, "developer", comment.AuthorRole)
	assert.Equal(t, "Agent 1", comment.AuthorName)
	assert.Equal(t, domain.AuthorTypeAgent, comment.AuthorType)
	assert.Equal(t, "This is a comment", comment.Content)
}

func TestApp_CreateComment_EmptyContent_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	_, err := a.CreateComment(ctx, projectID, taskID, "developer", "", domain.AuthorTypeAgent, "")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrCommentContentRequired)
}

func TestApp_CreateComment_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	_, err := a.CreateComment(ctx, projectID, taskID, "developer", "", domain.AuthorTypeAgent, "Some content")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}

func TestApp_CreateComment_HumanAuthor_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: columnID, Title: "Test Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	mockComments.CreateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		return nil
	}

	comment, err := a.CreateComment(ctx, projectID, taskID, "human", "John Doe", domain.AuthorTypeHuman, "Human comment")

	require.NoError(t, err)
	assert.Equal(t, domain.AuthorTypeHuman, comment.AuthorType)
	assert.Equal(t, "John Doe", comment.AuthorName)
}

func TestApp_UpdateComment_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()
	existingComment := &domain.Comment{
		ID:         commentID,
		TaskID:     domain.NewTaskID(),
		AuthorRole: "developer",
		AuthorType: domain.AuthorTypeAgent,
		Content:    "Original content",
	}

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		if pid == projectID && cid == commentID {
			return existingComment, nil
		}
		return nil, errors.New("not found")
	}

	var updatedComment domain.Comment
	mockComments.UpdateFunc = func(ctx context.Context, pid domain.ProjectID, comment domain.Comment) error {
		updatedComment = comment
		return nil
	}

	err := a.UpdateComment(ctx, projectID, commentID, "Updated content")

	require.NoError(t, err)
	assert.Equal(t, "Updated content", updatedComment.Content)
	assert.NotNil(t, updatedComment.EditedAt)
}

func TestApp_UpdateComment_EmptyContent_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()

	err := a.UpdateComment(ctx, projectID, commentID, "")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrCommentContentRequired)
}

func TestApp_UpdateComment_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		return nil, errors.New("not found")
	}

	err := a.UpdateComment(ctx, projectID, commentID, "Updated content")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

func TestApp_DeleteComment_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()
	existingComment := &domain.Comment{
		ID:         commentID,
		TaskID:     domain.NewTaskID(),
		AuthorType: domain.AuthorTypeAgent,
		Content:    "Some content",
	}

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		if pid == projectID && cid == commentID {
			return existingComment, nil
		}
		return nil, errors.New("not found")
	}

	mockComments.DeleteFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) error {
		return nil
	}

	err := a.DeleteComment(ctx, projectID, commentID)

	require.NoError(t, err)
}

func TestApp_DeleteComment_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		return nil, errors.New("not found")
	}

	err := a.DeleteComment(ctx, projectID, commentID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

// Comment Query Tests

func TestApp_GetComment_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()
	expectedComment := &domain.Comment{
		ID:         commentID,
		TaskID:     domain.NewTaskID(),
		AuthorRole: "developer",
		AuthorType: domain.AuthorTypeAgent,
		Content:    "Test comment",
	}

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		if pid == projectID && cid == commentID {
			return expectedComment, nil
		}
		return nil, errors.New("not found")
	}

	comment, err := a.GetComment(ctx, projectID, commentID)

	require.NoError(t, err)
	assert.Equal(t, expectedComment, comment)
}

func TestApp_GetComment_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	commentID := domain.NewCommentID()

	mockComments.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.CommentID) (*domain.Comment, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetComment(ctx, projectID, commentID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrCommentNotFound)
}

func TestApp_ListComments_Success(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		if pid == projectID && tid == taskID {
			return &domain.Task{ID: taskID, ColumnID: columnID, Title: "Test Task", Summary: "Summary"}, nil
		}
		return nil, errors.New("not found")
	}

	expectedComments := []domain.Comment{
		{ID: domain.NewCommentID(), TaskID: taskID, Content: "First comment", AuthorType: domain.AuthorTypeAgent},
		{ID: domain.NewCommentID(), TaskID: taskID, Content: "Second comment", AuthorType: domain.AuthorTypeHuman},
	}

	mockComments.ListFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID, limit, offset int) ([]domain.Comment, error) {
		if pid == projectID && tid == taskID {
			return expectedComments, nil
		}
		return nil, errors.New("not found")
	}

	comments, err := a.ListComments(ctx, projectID, taskID, 0, 0)

	require.NoError(t, err)
	assert.Equal(t, expectedComments, comments)
}

func TestApp_ListComments_TaskNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockTasks.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListComments(ctx, projectID, taskID, 0, 0)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrTaskNotFound)
}
