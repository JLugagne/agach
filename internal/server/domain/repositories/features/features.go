package features

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// FeatureRepository defines operations for managing features within a project
type FeatureRepository interface {
	// Create creates a new feature
	Create(ctx context.Context, feature domain.Feature) error

	// FindByID retrieves a feature by ID
	FindByID(ctx context.Context, id domain.FeatureID) (*domain.Feature, error)

	// List retrieves features for a project, optionally filtered by status.
	// If statusFilter is empty, all features are returned.
	// Returns FeatureWithTaskSummary so callers can display task counts.
	List(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error)

	// Update updates a feature's name and description
	Update(ctx context.Context, feature domain.Feature) error

	// UpdateStatus updates only the status of a feature
	UpdateStatus(ctx context.Context, id domain.FeatureID, status domain.FeatureStatus) error

	// Delete deletes a feature. Tasks referencing this feature will have feature_id set to NULL.
	Delete(ctx context.Context, id domain.FeatureID) error

	// GetStats returns feature counts grouped by status for a project
	GetStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error)

	// UpdateChangelogs updates the user or tech changelogs for a feature
	UpdateChangelogs(ctx context.Context, id domain.FeatureID, userChangelog, techChangelog *string) error

	// ListTaskSummaries lists completed task summaries for a feature, sorted by completed_at ASC
	ListTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error)
}
