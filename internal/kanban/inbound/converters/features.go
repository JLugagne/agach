package converters

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToPublicFeature converts domain.Feature to pkgkanban.FeatureResponse
func ToPublicFeature(f domain.Feature) pkgkanban.FeatureResponse {
	return pkgkanban.FeatureResponse{
		ID:             f.ID.String(),
		ProjectID:      f.ProjectID.String(),
		Name:           f.Name,
		Description:    f.Description,
		Status:         string(f.Status),
		CreatedByRole:  f.CreatedByRole,
		CreatedByAgent: f.CreatedByAgent,
		CreatedAt:      f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      f.UpdatedAt.Format(time.RFC3339),
	}
}

// ToPublicFeatureWithSummary converts domain.FeatureWithTaskSummary to pkgkanban.FeatureWithSummaryResponse
func ToPublicFeatureWithSummary(f domain.FeatureWithTaskSummary) pkgkanban.FeatureWithSummaryResponse {
	return pkgkanban.FeatureWithSummaryResponse{
		FeatureResponse: ToPublicFeature(f.Feature),
		TaskSummary:     ToPublicProjectSummary(f.TaskSummary),
	}
}

// ToPublicFeaturesWithSummary converts []domain.FeatureWithTaskSummary to []pkgkanban.FeatureWithSummaryResponse
func ToPublicFeaturesWithSummary(fs []domain.FeatureWithTaskSummary) []pkgkanban.FeatureWithSummaryResponse {
	result := make([]pkgkanban.FeatureWithSummaryResponse, len(fs))
	for i, f := range fs {
		result[i] = ToPublicFeatureWithSummary(f)
	}
	return result
}
