package featurestest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
)

var _ features.FeatureRepository = (*MockFeature)(nil)

type MockFeature struct {
	CreateFunc           func(ctx context.Context, feature domain.Feature) error
	FindByIDFunc         func(ctx context.Context, id domain.FeatureID) (*domain.Feature, error)
	ListFunc             func(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error)
	UpdateFunc           func(ctx context.Context, feature domain.Feature) error
	UpdateStatusFunc     func(ctx context.Context, id domain.FeatureID, status domain.FeatureStatus, nodeID string) error
	DeleteFunc           func(ctx context.Context, id domain.FeatureID) error
	GetStatsFunc         func(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error)
	UpdateChangelogsFunc func(ctx context.Context, id domain.FeatureID, userChangelog, techChangelog *string) error
	ListTaskSummariesFunc func(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error)
}

func (m *MockFeature) Create(ctx context.Context, feature domain.Feature) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, feature)
}

func (m *MockFeature) FindByID(ctx context.Context, id domain.FeatureID) (*domain.Feature, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockFeature) List(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, projectID, statusFilter)
}

func (m *MockFeature) Update(ctx context.Context, feature domain.Feature) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, feature)
}

func (m *MockFeature) UpdateStatus(ctx context.Context, id domain.FeatureID, status domain.FeatureStatus, nodeID string) error {
	if m.UpdateStatusFunc == nil {
		panic("called not defined UpdateStatusFunc")
	}
	return m.UpdateStatusFunc(ctx, id, status, nodeID)
}

func (m *MockFeature) Delete(ctx context.Context, id domain.FeatureID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func (m *MockFeature) GetStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error) {
	if m.GetStatsFunc == nil {
		panic("called not defined GetStatsFunc")
	}
	return m.GetStatsFunc(ctx, projectID)
}

func (m *MockFeature) UpdateChangelogs(ctx context.Context, id domain.FeatureID, userChangelog, techChangelog *string) error {
	if m.UpdateChangelogsFunc == nil {
		panic("called not defined UpdateChangelogsFunc")
	}
	return m.UpdateChangelogsFunc(ctx, id, userChangelog, techChangelog)
}

func (m *MockFeature) ListTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
	if m.ListTaskSummariesFunc == nil {
		panic("called not defined ListTaskSummariesFunc")
	}
	return m.ListTaskSummariesFunc(ctx, featureID)
}
