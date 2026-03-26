package converters

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicFeature converts domain.Feature to pkgserver.FeatureResponse
func ToPublicFeature(f domain.Feature) pkgserver.FeatureResponse {
	return pkgserver.FeatureResponse{
		ID:             f.ID.String(),
		ProjectID:      f.ProjectID.String(),
		Name:           f.Name,
		Description:    f.Description,
		UserChangelog:  f.UserChangelog,
		TechChangelog:  f.TechChangelog,
		Status:         string(f.Status),
		CreatedByRole:  f.CreatedByRole,
		CreatedByAgent: f.CreatedByAgent,
		CreatedAt:      f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      f.UpdatedAt.Format(time.RFC3339),
	}
}

// ToPublicFeatureWithSummary converts domain.FeatureWithTaskSummary to pkgserver.FeatureWithSummaryResponse
func ToPublicFeatureWithSummary(f domain.FeatureWithTaskSummary) pkgserver.FeatureWithSummaryResponse {
	return pkgserver.FeatureWithSummaryResponse{
		FeatureResponse: ToPublicFeature(f.Feature),
		TaskSummary:     ToPublicProjectSummary(f.TaskSummary),
	}
}

// ToPublicFeaturesWithSummary converts []domain.FeatureWithTaskSummary to []pkgserver.FeatureWithSummaryResponse
func ToPublicFeaturesWithSummary(fs []domain.FeatureWithTaskSummary) []pkgserver.FeatureWithSummaryResponse {
	return MapSlice(fs, ToPublicFeatureWithSummary)
}

// ToPublicTaskSummary converts domain.FeatureTaskSummary to pkgserver.TaskSummaryResponse
func ToPublicTaskSummary(ts domain.FeatureTaskSummary) pkgserver.TaskSummaryResponse {
	return pkgserver.TaskSummaryResponse{
		TaskID:            ts.ID.String(),
		Title:             ts.Title,
		CompletionSummary: ts.CompletionSummary,
		CompletedByAgent:  ts.CompletedByAgent,
		CompletedAt:       ts.CompletedAt,
		FilesModified:     ts.FilesModified,
	}
}

// ToPublicTaskSummaries converts []domain.FeatureTaskSummary to []pkgserver.TaskSummaryResponse
func ToPublicTaskSummaries(summaries []domain.FeatureTaskSummary) []pkgserver.TaskSummaryResponse {
	return MapSlice(summaries, ToPublicTaskSummary)
}
