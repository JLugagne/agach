package converters_test

import (
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToPublicComment(t *testing.T) {
	comment := domain.Comment{
		ID:         domain.CommentID("comment-123"),
		TaskID:     domain.TaskID("task-456"),
		AuthorRole: "architect",
		AuthorName: "Agent1",
		AuthorType: "agent",
		Content:    "This is a comment",
	}

	result := converters.ToPublicComment(comment)

	assert.Equal(t, "comment-123", result.ID)
	assert.Equal(t, "task-456", result.TaskID)
	assert.Equal(t, "architect", result.AuthorRole)
	assert.Equal(t, "Agent1", result.AuthorName)
	assert.Equal(t, "agent", result.AuthorType)
	assert.Equal(t, "This is a comment", result.Content)
}

func TestToPublicComments(t *testing.T) {
	comments := []domain.Comment{
		{ID: domain.CommentID("comment-1"), Content: "Comment 1"},
		{ID: domain.CommentID("comment-2"), Content: "Comment 2"},
	}

	result := converters.ToPublicComments(comments)

	assert.Len(t, result, 2)
	assert.Equal(t, "comment-1", result[0].ID)
	assert.Equal(t, "comment-2", result[1].ID)
}
