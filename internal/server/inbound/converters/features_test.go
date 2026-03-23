package converters_test

import (
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/inbound/converters"
	"github.com/stretchr/testify/assert"
)

// TestToPublicFeature tests the ToPublicFeature converter
func TestToPublicFeature(t *testing.T) {
	featureID := domain.NewFeatureID()
	projectID := domain.NewProjectID()
	now := time.Now()

	feature := domain.Feature{
		ID:             featureID,
		ProjectID:      projectID,
		Name:           "Test Feature",
		Description:    "A test feature",
		Status:         domain.FeatureStatusReady,
		CreatedByRole:  "developer",
		CreatedByAgent: "agent1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp := converters.ToPublicFeature(feature)

	assert.Equal(t, feature.ID.String(), resp.ID)
	assert.Equal(t, feature.ProjectID.String(), resp.ProjectID)
	assert.Equal(t, feature.Name, resp.Name)
	assert.Equal(t, feature.Description, resp.Description)
	assert.Equal(t, string(feature.Status), resp.Status)
	assert.Equal(t, feature.CreatedByRole, resp.CreatedByRole)
	assert.Equal(t, feature.CreatedByAgent, resp.CreatedByAgent)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

// TestToPublicFeatureWithSummary tests the ToPublicFeatureWithSummary converter
func TestToPublicFeatureWithSummary(t *testing.T) {
	featureID := domain.NewFeatureID()
	projectID := domain.NewProjectID()
	now := time.Now()

	featureWithSummary := domain.FeatureWithTaskSummary{
		Feature: domain.Feature{
			ID:             featureID,
			ProjectID:      projectID,
			Name:           "Test Feature",
			Description:    "A test feature",
			Status:         domain.FeatureStatusDraft,
			CreatedByRole:  "developer",
			CreatedByAgent: "agent1",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		TaskSummary: domain.ProjectSummary{
			BacklogCount:    1,
			TodoCount:       2,
			InProgressCount: 1,
			DoneCount:       0,
			BlockedCount:    0,
		},
	}

	resp := converters.ToPublicFeatureWithSummary(featureWithSummary)

	assert.Equal(t, featureWithSummary.Feature.ID.String(), resp.ID)
	assert.Equal(t, featureWithSummary.Feature.Name, resp.Name)
	assert.Equal(t, 1, resp.TaskSummary.BacklogCount)
	assert.Equal(t, 2, resp.TaskSummary.TodoCount)
	assert.Equal(t, 1, resp.TaskSummary.InProgressCount)
}

// TestToPublicFeaturesWithSummary tests the ToPublicFeaturesWithSummary converter
func TestToPublicFeaturesWithSummary(t *testing.T) {
	now := time.Now()

	features := []domain.FeatureWithTaskSummary{
		{
			Feature: domain.Feature{
				ID:        domain.NewFeatureID(),
				ProjectID: domain.NewProjectID(),
				Name:      "Feature 1",
				Status:    domain.FeatureStatusDraft,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			Feature: domain.Feature{
				ID:        domain.NewFeatureID(),
				ProjectID: domain.NewProjectID(),
				Name:      "Feature 2",
				Status:    domain.FeatureStatusReady,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	responses := converters.ToPublicFeaturesWithSummary(features)

	assert.Len(t, responses, 2)
	assert.Equal(t, "Feature 1", responses[0].Name)
	assert.Equal(t, "Feature 2", responses[1].Name)
}
