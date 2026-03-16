package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToDomainProjectID converts a string to domain.ProjectID
func ToDomainProjectID(id *string) *domain.ProjectID {
	if id == nil {
		return nil
	}
	projectID := domain.ProjectID(*id)
	return &projectID
}

// ToPublicProject converts domain.Project to pkgkanban.ProjectResponse
func ToPublicProject(project domain.Project) pkgkanban.ProjectResponse {
	var parentID *string
	if project.ParentID != nil {
		pid := string(*project.ParentID)
		parentID = &pid
	}

	return pkgkanban.ProjectResponse{
		ID:             string(project.ID),
		ParentID:       parentID,
		Name:           project.Name,
		Description:    project.Description,
		WorkDir:        project.WorkDir,
		CreatedByRole:  project.CreatedByRole,
		CreatedByAgent: project.CreatedByAgent,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}
}

// ToPublicProjects converts []domain.Project to []pkgkanban.ProjectResponse
func ToPublicProjects(projects []domain.Project) []pkgkanban.ProjectResponse {
	result := make([]pkgkanban.ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = ToPublicProject(p)
	}
	return result
}

// ToPublicProjectSummary converts domain.ProjectSummary to pkgkanban.ProjectSummaryResponse
func ToPublicProjectSummary(summary domain.ProjectSummary) pkgkanban.ProjectSummaryResponse {
	return pkgkanban.ProjectSummaryResponse{
		TodoCount:       summary.TodoCount,
		InProgressCount: summary.InProgressCount,
		DoneCount:       summary.DoneCount,
		BlockedCount:    summary.BlockedCount,
	}
}
