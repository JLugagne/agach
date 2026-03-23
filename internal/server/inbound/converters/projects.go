package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToDomainProjectID converts a string to domain.ProjectID
func ToDomainProjectID(id *string) *domain.ProjectID {
	if id == nil {
		return nil
	}
	projectID := domain.ProjectID(*id)
	return &projectID
}

// ToPublicProject converts domain.Project to pkgserver.ProjectResponse
func ToPublicProject(project domain.Project) pkgserver.ProjectResponse {
	var parentID *string
	if project.ParentID != nil {
		pid := string(*project.ParentID)
		parentID = &pid
	}

	var dockerfileID *string
	if project.DockerfileID != nil {
		did := string(*project.DockerfileID)
		dockerfileID = &did
	}

	return pkgserver.ProjectResponse{
		ID:             string(project.ID),
		ParentID:       parentID,
		Name:           project.Name,
		Description:    project.Description,
		GitURL:         project.GitURL,
		CreatedByRole:  project.CreatedByRole,
		CreatedByAgent: project.CreatedByAgent,
		DefaultRole:    project.DefaultRole,
		DockerfileID:   dockerfileID,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}
}

// ToPublicProjects converts []domain.Project to []pkgserver.ProjectResponse
func ToPublicProjects(projects []domain.Project) []pkgserver.ProjectResponse {
	result := make([]pkgserver.ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = ToPublicProject(p)
	}
	return result
}

// ToPublicProjectSummary converts domain.ProjectSummary to pkgserver.ProjectSummaryResponse
func ToPublicProjectSummary(summary domain.ProjectSummary) pkgserver.ProjectSummaryResponse {
	return pkgserver.ProjectSummaryResponse{
		TodoCount:       summary.TodoCount,
		InProgressCount: summary.InProgressCount,
		DoneCount:       summary.DoneCount,
		BlockedCount:    summary.BlockedCount,
	}
}
