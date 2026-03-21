package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicComment converts domain.Comment to pkgkanban.CommentResponse
func ToPublicComment(comment domain.Comment) pkgkanban.CommentResponse {
	authorType := string(comment.AuthorType)
	if comment.AuthorType != domain.AuthorTypeAgent && comment.AuthorType != domain.AuthorTypeHuman {
		authorType = string(domain.AuthorTypeAgent)
	}
	return pkgkanban.CommentResponse{
		ID:         string(comment.ID),
		TaskID:     string(comment.TaskID),
		AuthorRole: comment.AuthorRole,
		AuthorName: comment.AuthorName,
		AuthorType: authorType,
		Content:    comment.Content,
		EditedAt:   comment.EditedAt,
		CreatedAt:  comment.CreatedAt,
	}
}

// ToPublicComments converts []domain.Comment to []pkgkanban.CommentResponse
func ToPublicComments(comments []domain.Comment) []pkgkanban.CommentResponse {
	result := make([]pkgkanban.CommentResponse, len(comments))
	for i, c := range comments {
		result[i] = ToPublicComment(c)
	}
	return result
}
