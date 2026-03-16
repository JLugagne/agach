package converters_test

import (
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToDomainProjectID(t *testing.T) {
	t.Run("Nil input returns nil", func(t *testing.T) {
		result := converters.ToDomainProjectID(nil)
		assert.Nil(t, result)
	})

	t.Run("Valid string ID converts correctly", func(t *testing.T) {
		idStr := "test-project-id-123"
		result := converters.ToDomainProjectID(&idStr)

		assert.NotNil(t, result)
		assert.Equal(t, domain.ProjectID("test-project-id-123"), *result)
	})
}

func TestToPublicProject(t *testing.T) {
	t.Run("Converts project without parent", func(t *testing.T) {
		now := time.Now()
		project := domain.Project{
			ID:             domain.ProjectID("proj-123"),
			ParentID:       nil,
			Name:           "Test Project",
			Description:    "Test Description",
			CreatedByRole:  "architect",
			CreatedByAgent: "agent1",
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		result := converters.ToPublicProject(project)

		assert.Equal(t, "proj-123", result.ID)
		assert.Nil(t, result.ParentID)
		assert.Equal(t, "Test Project", result.Name)
		assert.Equal(t, "Test Description", result.Description)
		assert.Equal(t, "architect", result.CreatedByRole)
		assert.Equal(t, "agent1", result.CreatedByAgent)
		assert.Equal(t, now, result.CreatedAt)
		assert.Equal(t, now, result.UpdatedAt)
	})

	t.Run("Converts project with parent", func(t *testing.T) {
		parentID := domain.ProjectID("parent-123")
		project := domain.Project{
			ID:       domain.ProjectID("child-456"),
			ParentID: &parentID,
			Name:     "Child Project",
		}

		result := converters.ToPublicProject(project)

		assert.Equal(t, "child-456", result.ID)
		assert.NotNil(t, result.ParentID)
		assert.Equal(t, "parent-123", *result.ParentID)
		assert.Equal(t, "Child Project", result.Name)
	})
}

func TestToPublicProjects(t *testing.T) {
	t.Run("Converts empty list", func(t *testing.T) {
		projects := []domain.Project{}

		result := converters.ToPublicProjects(projects)

		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("Converts multiple projects", func(t *testing.T) {
		projects := []domain.Project{
			{
				ID:   domain.ProjectID("proj-1"),
				Name: "Project 1",
			},
			{
				ID:   domain.ProjectID("proj-2"),
				Name: "Project 2",
			},
		}

		result := converters.ToPublicProjects(projects)

		assert.Len(t, result, 2)
		assert.Equal(t, "proj-1", result[0].ID)
		assert.Equal(t, "Project 1", result[0].Name)
		assert.Equal(t, "proj-2", result[1].ID)
		assert.Equal(t, "Project 2", result[1].Name)
	})
}

func TestToPublicProjectSummary(t *testing.T) {
	t.Run("Converts summary correctly", func(t *testing.T) {
		summary := domain.ProjectSummary{
			TodoCount:       5,
			InProgressCount: 3,
			DoneCount:       10,
			BlockedCount:    2,
		}

		result := converters.ToPublicProjectSummary(summary)

		assert.Equal(t, 5, result.TodoCount)
		assert.Equal(t, 3, result.InProgressCount)
		assert.Equal(t, 10, result.DoneCount)
		assert.Equal(t, 2, result.BlockedCount)
	})
}
