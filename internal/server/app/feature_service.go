package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/sirupsen/logrus"
)

type FeatureService struct {
	features featuresrepo.FeatureRepository
	projects projects.ProjectRepository
	logger   *logrus.Logger
}

func newFeatureService(features featuresrepo.FeatureRepository, projects projects.ProjectRepository, logger *logrus.Logger) *FeatureService {
	return &FeatureService{
		features: features,
		projects: projects,
		logger:   logger,
	}
}

func (s *FeatureService) CreateFeature(ctx context.Context, projectID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error) {
	if name == "" {
		return domain.Feature{}, domain.ErrFeatureNameRequired
	}

	if _, err := s.projects.FindByID(ctx, projectID); err != nil {
		return domain.Feature{}, errors.Join(domain.ErrProjectNotFound, err)
	}

	feature := domain.Feature{
		ID:             domain.NewFeatureID(),
		ProjectID:      projectID,
		Name:           name,
		Description:    description,
		Status:         domain.FeatureStatusDraft,
		CreatedByRole:  createdByRole,
		CreatedByAgent: createdByAgent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.features.Create(ctx, feature); err != nil {
		return domain.Feature{}, err
	}

	return feature, nil
}

func (s *FeatureService) UpdateFeature(ctx context.Context, featureID domain.FeatureID, name, description string) error {
	if name == "" {
		return domain.ErrFeatureNameRequired
	}

	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	feature.Name = name
	feature.Description = description
	feature.UpdatedAt = time.Now()

	return s.features.Update(ctx, *feature)
}

// validFeatureTransitions defines allowed transitions between feature statuses.
// Any transition not listed here is forbidden.
var validFeatureTransitions = map[domain.FeatureStatus]map[domain.FeatureStatus]bool{
	domain.FeatureStatusDraft:      {domain.FeatureStatusReady: true, domain.FeatureStatusBlocked: true},
	domain.FeatureStatusReady:      {domain.FeatureStatusDraft: true, domain.FeatureStatusInProgress: true, domain.FeatureStatusBlocked: true},
	domain.FeatureStatusInProgress: {domain.FeatureStatusReady: true, domain.FeatureStatusDone: true, domain.FeatureStatusBlocked: true},
	domain.FeatureStatusDone:       {},
	domain.FeatureStatusBlocked:    {domain.FeatureStatusDraft: true, domain.FeatureStatusReady: true, domain.FeatureStatusInProgress: true},
}

func (s *FeatureService) UpdateFeatureStatus(ctx context.Context, featureID domain.FeatureID, status domain.FeatureStatus, nodeID string) error {
	if !domain.ValidFeatureStatuses[status] {
		return domain.ErrInvalidFeatureStatus
	}

	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	allowed := validFeatureTransitions[feature.Status]
	if !allowed[status] {
		return domain.ErrInvalidFeatureStatus
	}

	return s.features.UpdateStatus(ctx, featureID, status, nodeID)
}

func (s *FeatureService) DeleteFeature(ctx context.Context, featureID domain.FeatureID) error {
	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	return s.features.Delete(ctx, featureID)
}

func (s *FeatureService) GetFeature(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error) {
	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return nil, errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return nil, domain.ErrFeatureNotFound
	}
	return feature, nil
}

func (s *FeatureService) ListFeatures(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	return s.features.List(ctx, projectID, statusFilter)
}

func (s *FeatureService) GetFeatureStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)
	_ = logger
	return s.features.GetStats(ctx, projectID)
}

func (s *FeatureService) UpdateFeatureChangelogs(ctx context.Context, featureID domain.FeatureID, userChangelog, techChangelog *string) error {
	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return domain.ErrFeatureNotFound
	}

	return s.features.UpdateChangelogs(ctx, featureID, userChangelog, techChangelog)
}

func (s *FeatureService) ListFeatureTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
	feature, err := s.features.FindByID(ctx, featureID)
	if err != nil {
		return nil, errors.Join(domain.ErrFeatureNotFound, err)
	}
	if feature == nil {
		return nil, domain.ErrFeatureNotFound
	}

	return s.features.ListTaskSummaries(ctx, featureID)
}
