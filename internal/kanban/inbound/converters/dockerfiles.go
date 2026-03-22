package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicDockerfile converts a domain Dockerfile to a public API response.
func ToPublicDockerfile(d domain.Dockerfile) pkgkanban.DockerfileResponse {
	return pkgkanban.DockerfileResponse{
		ID:          string(d.ID),
		Slug:        d.Slug,
		Name:        d.Name,
		Description: d.Description,
		Version:     d.Version,
		Content:     d.Content,
		IsLatest:    d.IsLatest,
		SortOrder:   d.SortOrder,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// ToPublicDockerfiles converts a slice of domain Dockerfiles to public API responses.
func ToPublicDockerfiles(ds []domain.Dockerfile) []pkgkanban.DockerfileResponse {
	result := make([]pkgkanban.DockerfileResponse, len(ds))
	for i, d := range ds {
		result[i] = ToPublicDockerfile(d)
	}
	return result
}
