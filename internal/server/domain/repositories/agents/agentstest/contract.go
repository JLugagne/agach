package agentstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAgentRepository is a function-based mock implementation of the AgentRepository interface.
type MockAgentRepository struct {
	CreateFunc                   func(ctx context.Context, agent domain.Agent) error
	FindByIDFunc                 func(ctx context.Context, id domain.AgentID) (*domain.Agent, error)
	FindBySlugFunc               func(ctx context.Context, slug string) (*domain.Agent, error)
	ListFunc                     func(ctx context.Context) ([]domain.Agent, error)
	UpdateFunc                   func(ctx context.Context, agent domain.Agent) error
	DeleteFunc                   func(ctx context.Context, id domain.AgentID) error
	IsInUseFunc                  func(ctx context.Context, slug string) (bool, error)
	CloneFunc                    func(ctx context.Context, sourceID domain.AgentID, newSlug, newName string) (domain.Agent, error)
	AssignToProjectFunc          func(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error
	RemoveFromProjectFunc        func(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error
	ListByProjectFunc            func(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	IsAssignedToProjectFunc      func(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) (bool, error)
	CopyGlobalRolesToProjectFunc func(ctx context.Context, projectID domain.ProjectID) error
	CreateInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error
	FindBySlugInProjectFunc      func(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)
	FindByIDInProjectFunc        func(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) (*domain.Agent, error)
	ListInProjectFunc            func(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	UpdateInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error
	DeleteInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) error
}

// MockRoleRepository is an alias for backward compatibility
type MockRoleRepository = MockAgentRepository

func (m *MockAgentRepository) Create(ctx context.Context, agent domain.Agent) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, agent)
}

func (m *MockAgentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockAgentRepository) FindBySlug(ctx context.Context, slug string) (*domain.Agent, error) {
	if m.FindBySlugFunc == nil {
		panic("called not defined FindBySlugFunc")
	}
	return m.FindBySlugFunc(ctx, slug)
}

func (m *MockAgentRepository) List(ctx context.Context) ([]domain.Agent, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx)
}

func (m *MockAgentRepository) Update(ctx context.Context, agent domain.Agent) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, agent)
}

func (m *MockAgentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func (m *MockAgentRepository) IsInUse(ctx context.Context, slug string) (bool, error) {
	if m.IsInUseFunc == nil {
		panic("called not defined IsInUseFunc")
	}
	return m.IsInUseFunc(ctx, slug)
}

func (m *MockAgentRepository) Clone(ctx context.Context, sourceID domain.AgentID, newSlug, newName string) (domain.Agent, error) {
	if m.CloneFunc == nil {
		panic("called not defined CloneFunc")
	}
	return m.CloneFunc(ctx, sourceID, newSlug, newName)
}

func (m *MockAgentRepository) AssignToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error {
	if m.AssignToProjectFunc == nil {
		panic("called not defined AssignToProjectFunc")
	}
	return m.AssignToProjectFunc(ctx, projectID, agentID)
}

func (m *MockAgentRepository) RemoveFromProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error {
	if m.RemoveFromProjectFunc == nil {
		panic("called not defined RemoveFromProjectFunc")
	}
	return m.RemoveFromProjectFunc(ctx, projectID, agentID)
}

func (m *MockAgentRepository) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	if m.ListByProjectFunc == nil {
		panic("called not defined ListByProjectFunc")
	}
	return m.ListByProjectFunc(ctx, projectID)
}

func (m *MockAgentRepository) IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) (bool, error) {
	if m.IsAssignedToProjectFunc == nil {
		panic("called not defined IsAssignedToProjectFunc")
	}
	return m.IsAssignedToProjectFunc(ctx, projectID, agentID)
}

func (m *MockAgentRepository) CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error {
	if m.CopyGlobalRolesToProjectFunc == nil {
		return nil
	}
	return m.CopyGlobalRolesToProjectFunc(ctx, projectID)
}

func (m *MockAgentRepository) CreateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error {
	if m.CreateInProjectFunc == nil {
		return nil
	}
	return m.CreateInProjectFunc(ctx, projectID, agent)
}

func (m *MockAgentRepository) FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error) {
	if m.FindBySlugInProjectFunc == nil {
		panic("called not defined FindBySlugInProjectFunc")
	}
	return m.FindBySlugInProjectFunc(ctx, projectID, slug)
}

func (m *MockAgentRepository) FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) (*domain.Agent, error) {
	if m.FindByIDInProjectFunc == nil {
		panic("called not defined FindByIDInProjectFunc")
	}
	return m.FindByIDInProjectFunc(ctx, projectID, id)
}

func (m *MockAgentRepository) ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	if m.ListInProjectFunc == nil {
		panic("called not defined ListInProjectFunc")
	}
	return m.ListInProjectFunc(ctx, projectID)
}

func (m *MockAgentRepository) UpdateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error {
	if m.UpdateInProjectFunc == nil {
		panic("called not defined UpdateInProjectFunc")
	}
	return m.UpdateInProjectFunc(ctx, projectID, agent)
}

func (m *MockAgentRepository) DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) error {
	if m.DeleteInProjectFunc == nil {
		panic("called not defined DeleteInProjectFunc")
	}
	return m.DeleteInProjectFunc(ctx, projectID, id)
}

// AgentsContractTesting runs all contract tests for an AgentRepository implementation.
func AgentsContractTesting(t *testing.T, repo agents.AgentRepository) {
	ctx := context.Background()

	t.Run("Contract: Create stores agent and FindByID retrieves it", func(t *testing.T) {
		agent := domain.Agent{
			ID:          domain.NewAgentID(),
			Slug:        "test-developer",
			Name:        "Test Developer",
			Icon:        "💻",
			Color:       "#3B82F6",
			Description: "A test developer agent",
			TechStack:   []string{"Go", "React"},
			PromptHint:  "Focus on clean code",
			SortOrder:   1,
			CreatedAt:   time.Now(),
		}

		err := repo.Create(ctx, agent)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, agent.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved agent must not be nil")
		assert.Equal(t, agent.ID, retrieved.ID)
		assert.Equal(t, agent.Slug, retrieved.Slug)
		assert.Equal(t, agent.Name, retrieved.Name)
		assert.Equal(t, agent.Icon, retrieved.Icon)
		assert.Equal(t, agent.Color, retrieved.Color)
		assert.Equal(t, agent.Description, retrieved.Description)
		assert.Equal(t, agent.TechStack, retrieved.TechStack)
		assert.Equal(t, agent.PromptHint, retrieved.PromptHint)
		assert.Equal(t, agent.SortOrder, retrieved.SortOrder)
	})

	t.Run("Contract: FindByID returns error for non-existent agent", func(t *testing.T) {
		nonExistentID := domain.NewAgentID()
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent agent")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("Contract: Create stores agent and FindBySlug retrieves it", func(t *testing.T) {
		agent := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "test-designer",
			Name:      "Test Designer",
			SortOrder: 2,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, agent)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindBySlug(ctx, agent.Slug)
		require.NoError(t, err, "FindBySlug should succeed")
		require.NotNil(t, retrieved, "Retrieved agent must not be nil")
		assert.Equal(t, agent.ID, retrieved.ID)
		assert.Equal(t, agent.Slug, retrieved.Slug)
		assert.Equal(t, agent.Name, retrieved.Name)
	})

	t.Run("Contract: FindBySlug returns error for non-existent slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, "non-existent-slug")
		assert.Error(t, err, "FindBySlug should return error for non-existent slug")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("Contract: Create returns error for duplicate slug", func(t *testing.T) {
		agent1 := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "duplicate-slug",
			Name:      "First Agent",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		agent2 := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "duplicate-slug",
			Name:      "Second Agent",
			SortOrder: 2,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, agent1)
		require.NoError(t, err, "First Create should succeed")

		err = repo.Create(ctx, agent2)
		assert.Error(t, err, "Second Create with duplicate slug should fail")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrAgentAlreadyExists)
	})

	t.Run("Contract: List returns all agents ordered by sort_order", func(t *testing.T) {
		agentList := []domain.Agent{
			{
				ID:        domain.NewAgentID(),
				Slug:      "list-agent-3",
				Name:      "List Agent 3",
				SortOrder: 3,
				CreatedAt: time.Now(),
			},
			{
				ID:        domain.NewAgentID(),
				Slug:      "list-agent-1",
				Name:      "List Agent 1",
				SortOrder: 1,
				CreatedAt: time.Now(),
			},
			{
				ID:        domain.NewAgentID(),
				Slug:      "list-agent-2",
				Name:      "List Agent 2",
				SortOrder: 2,
				CreatedAt: time.Now(),
			},
		}

		for _, agent := range agentList {
			err := repo.Create(ctx, agent)
			require.NoError(t, err, "Create should succeed")
		}

		retrieved, err := repo.List(ctx)
		require.NoError(t, err, "List should succeed")
		require.NotEmpty(t, retrieved, "List should return agents")

		var testAgents []domain.Agent
		for _, r := range retrieved {
			if r.Slug == "list-agent-1" || r.Slug == "list-agent-2" || r.Slug == "list-agent-3" {
				testAgents = append(testAgents, r)
			}
		}

		require.Len(t, testAgents, 3, "Should find all test agents")

		assert.Equal(t, "list-agent-1", testAgents[0].Slug, "First agent should be list-agent-1")
		assert.Equal(t, "list-agent-2", testAgents[1].Slug, "Second agent should be list-agent-2")
		assert.Equal(t, "list-agent-3", testAgents[2].Slug, "Third agent should be list-agent-3")
	})

	t.Run("Contract: Update modifies agent data", func(t *testing.T) {
		agent := domain.Agent{
			ID:          domain.NewAgentID(),
			Slug:        "update-agent",
			Name:        "Update Agent",
			Icon:        "📝",
			Color:       "#EF4444",
			Description: "Original description",
			SortOrder:   1,
			CreatedAt:   time.Now(),
		}

		err := repo.Create(ctx, agent)
		require.NoError(t, err, "Create should succeed")

		agent.Name = "Updated Agent"
		agent.Description = "Updated description"
		agent.Icon = "✏️"
		agent.Color = "#10B981"

		err = repo.Update(ctx, agent)
		require.NoError(t, err, "Update should succeed")

		retrieved, err := repo.FindByID(ctx, agent.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, "Updated Agent", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, "✏️", retrieved.Icon)
		assert.Equal(t, "#10B981", retrieved.Color)
		assert.Equal(t, "update-agent", retrieved.Slug, "Slug should remain unchanged")
	})

	t.Run("Contract: Update returns error for non-existent agent", func(t *testing.T) {
		nonExistent := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "non-existent",
			Name:      "Non-existent",
			CreatedAt: time.Now(),
		}

		err := repo.Update(ctx, nonExistent)
		assert.Error(t, err, "Update should return error for non-existent agent")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("Contract: Delete removes agent", func(t *testing.T) {
		agent := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "delete-agent",
			Name:      "Delete Agent",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, agent)
		require.NoError(t, err, "Create should succeed")

		err = repo.Delete(ctx, agent.ID)
		require.NoError(t, err, "Delete should succeed")

		_, err = repo.FindByID(ctx, agent.ID)
		assert.Error(t, err, "FindByID should return error for deleted agent")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("Contract: Delete returns error for non-existent agent", func(t *testing.T) {
		nonExistentID := domain.NewAgentID()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err, "Delete should return error for non-existent agent")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("Contract: IsInUse returns false for unused agent", func(t *testing.T) {
		agent := domain.Agent{
			ID:        domain.NewAgentID(),
			Slug:      "unused-agent",
			Name:      "Unused Agent",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, agent)
		require.NoError(t, err, "Create should succeed")

		inUse, err := repo.IsInUse(ctx, agent.Slug)
		require.NoError(t, err, "IsInUse should succeed")
		assert.False(t, inUse, "Unused agent should return false")
	})
}

// RolesContractTesting is an alias for backward compatibility
var RolesContractTesting = AgentsContractTesting

// AgentExtendedContractTesting runs contract tests for Clone/AssignToProject/
// RemoveFromProject/ListByProject/IsAssignedToProject methods.
func AgentExtendedContractTesting(t *testing.T, repo agents.AgentRepository) {
	ctx := context.Background()

	seedAgent := func(t *testing.T, slug, name string) domain.Agent {
		t.Helper()
		agent := domain.Agent{
			ID:          domain.NewAgentID(),
			Slug:        slug,
			Name:        name,
			Description: "Contract test agent",
			TechStack:   []string{"Go"},
			PromptHint:  "Be helpful",
			SortOrder:   0,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, repo.Create(ctx, agent))
		return agent
	}

	t.Run("Contract: Clone creates copy with new slug", func(t *testing.T) {
		original := seedAgent(t, "original-"+domain.NewAgentID().String(), "Original")

		cloned, err := repo.Clone(ctx, original.ID, "clone-1-"+domain.NewAgentID().String(), "Clone 1")
		require.NoError(t, err)
		assert.NotEqual(t, original.ID, cloned.ID)
		assert.Equal(t, "Clone 1", cloned.Name)
		assert.Equal(t, original.Description, cloned.Description)
		assert.Equal(t, original.TechStack, cloned.TechStack)

		retrieved, err := repo.FindByID(ctx, original.ID)
		require.NoError(t, err)
		assert.Equal(t, original.ID, retrieved.ID, "original must still exist")
	})

	t.Run("Contract: Clone returns ErrAgentAlreadyExists for duplicate slug", func(t *testing.T) {
		original := seedAgent(t, "orig-dup-"+domain.NewAgentID().String(), "Original Dup")

		_, err := repo.Clone(ctx, original.ID, original.Slug, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgentAlreadyExists)
	})

	t.Run("Contract: AssignToProject and ListByProject", func(t *testing.T) {
		agent := seedAgent(t, "assign-agent-"+domain.NewAgentID().String(), "Assign Agent")
		projectID := domain.NewProjectID()

		err := repo.AssignToProject(ctx, projectID, agent.ID)
		require.NoError(t, err)

		list, err := repo.ListByProject(ctx, projectID)
		require.NoError(t, err)
		found := false
		for _, r := range list {
			if r.ID == agent.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "assigned agent must appear in ListByProject")

		assigned, err := repo.IsAssignedToProject(ctx, projectID, agent.ID)
		require.NoError(t, err)
		assert.True(t, assigned)
	})

	t.Run("Contract: AssignToProject idempotency", func(t *testing.T) {
		agent := seedAgent(t, "idempotent-agent-"+domain.NewAgentID().String(), "Idempotent Agent")
		projectID := domain.NewProjectID()

		err := repo.AssignToProject(ctx, projectID, agent.ID)
		require.NoError(t, err)

		err = repo.AssignToProject(ctx, projectID, agent.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgentAlreadyInProject)
	})

	t.Run("Contract: RemoveFromProject", func(t *testing.T) {
		agent := seedAgent(t, "remove-agent-"+domain.NewAgentID().String(), "Remove Agent")
		projectID := domain.NewProjectID()

		require.NoError(t, repo.AssignToProject(ctx, projectID, agent.ID))

		err := repo.RemoveFromProject(ctx, projectID, agent.ID)
		require.NoError(t, err)

		list, err := repo.ListByProject(ctx, projectID)
		require.NoError(t, err)
		for _, r := range list {
			assert.NotEqual(t, agent.ID, r.ID, "removed agent must not appear in ListByProject")
		}

		assigned, err := repo.IsAssignedToProject(ctx, projectID, agent.ID)
		require.NoError(t, err)
		assert.False(t, assigned)
	})

	t.Run("Contract: RemoveFromProject not-assigned", func(t *testing.T) {
		agent := seedAgent(t, "never-assigned-"+domain.NewAgentID().String(), "Never Assigned")
		projectID := domain.NewProjectID()

		err := repo.RemoveFromProject(ctx, projectID, agent.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgentNotInProject)
	})
}

// RoleExtendedContractTesting is an alias for backward compatibility
var RoleExtendedContractTesting = AgentExtendedContractTesting
