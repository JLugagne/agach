package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicComment converts domain.Comment to pkgserver.CommentResponse
func ToPublicComment(comment domain.Comment) pkgserver.CommentResponse {
	authorType := string(comment.AuthorType)
	if comment.AuthorType != domain.AuthorTypeAgent && comment.AuthorType != domain.AuthorTypeHuman {
		authorType = string(domain.AuthorTypeAgent)
	}
	return pkgserver.CommentResponse{
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

// ToPublicComments converts []domain.Comment to []pkgserver.CommentResponse
func ToPublicComments(comments []domain.Comment) []pkgserver.CommentResponse {
	result := make([]pkgserver.CommentResponse, len(comments))
	for i, c := range comments {
		result[i] = ToPublicComment(c)
	}
	return result
}
