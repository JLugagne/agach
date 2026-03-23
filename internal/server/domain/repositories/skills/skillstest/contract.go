package skillstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ skills.SkillRepository = (*MockSkill)(nil)

type MockSkill struct {
	CreateFunc          func(ctx context.Context, skill domain.Skill) error
	FindByIDFunc        func(ctx context.Context, id domain.SkillID) (*domain.Skill, error)
	FindBySlugFunc      func(ctx context.Context, slug string) (*domain.Skill, error)
	ListFunc            func(ctx context.Context) ([]domain.Skill, error)
	UpdateFunc          func(ctx context.Context, skill domain.Skill) error
	DeleteFunc          func(ctx context.Context, id domain.SkillID) error
	IsInUseFunc         func(ctx context.Context, id domain.SkillID) (bool, error)
	ListByAgentFunc     func(ctx context.Context, roleID domain.RoleID) ([]domain.Skill, error)
	AssignToAgentFunc   func(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error
	RemoveFromAgentFunc func(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error
}

func (m *MockSkill) Create(ctx context.Context, skill domain.Skill) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, skill)
}

func (m *MockSkill) FindByID(ctx context.Context, id domain.SkillID) (*domain.Skill, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockSkill) FindBySlug(ctx context.Context, slug string) (*domain.Skill, error) {
	if m.FindBySlugFunc == nil {
		panic("called not defined FindBySlugFunc")
	}
	return m.FindBySlugFunc(ctx, slug)
}

func (m *MockSkill) List(ctx context.Context) ([]domain.Skill, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx)
}

func (m *MockSkill) Update(ctx context.Context, skill domain.Skill) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, skill)
}

func (m *MockSkill) Delete(ctx context.Context, id domain.SkillID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func (m *MockSkill) IsInUse(ctx context.Context, id domain.SkillID) (bool, error) {
	if m.IsInUseFunc == nil {
		panic("called not defined IsInUseFunc")
	}
	return m.IsInUseFunc(ctx, id)
}

func (m *MockSkill) ListByAgent(ctx context.Context, roleID domain.RoleID) ([]domain.Skill, error) {
	if m.ListByAgentFunc == nil {
		panic("called not defined ListByAgentFunc")
	}
	return m.ListByAgentFunc(ctx, roleID)
}

func (m *MockSkill) AssignToAgent(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error {
	if m.AssignToAgentFunc == nil {
		panic("called not defined AssignToAgentFunc")
	}
	return m.AssignToAgentFunc(ctx, roleID, skillID)
}

func (m *MockSkill) RemoveFromAgent(ctx context.Context, roleID domain.RoleID, skillID domain.SkillID) error {
	if m.RemoveFromAgentFunc == nil {
		panic("called not defined RemoveFromAgentFunc")
	}
	return m.RemoveFromAgentFunc(ctx, roleID, skillID)
}

// SkillContractTesting runs all contract tests for a SkillRepository implementation.
// The caller must ensure that at least one role exists in the backing store before
// invoking this function; the roleID parameter identifies that pre-seeded role.
// If no role can be provided, pass a zero-value RoleID and the sub-tests that
// require an agent association will be skipped.
func SkillContractTesting(t *testing.T, repo skills.SkillRepository, roleID domain.RoleID) {
	ctx := context.Background()

	t.Run("Contract: Create and FindByID", func(t *testing.T) {
		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "go-tools",
			Name:      "Go Tools",
			Content:   "## Go Tools",
			SortOrder: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))
		found, err := repo.FindByID(ctx, skill.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, skill.ID, found.ID)
	})

	t.Run("Contract: FindBySlug", func(t *testing.T) {
		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "python-utils",
			Name:      "Python Utils",
			Content:   "## Python Utils",
			SortOrder: 2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))

		found, err := repo.FindBySlug(ctx, "python-utils")
		require.NoError(t, err)
		require.NotNil(t, found)

		missing, err := repo.FindBySlug(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, missing)
	})

	t.Run("Contract: List ordering", func(t *testing.T) {
		skills := []domain.Skill{
			{ID: domain.NewSkillID(), Slug: "order-c", Name: "Order C", SortOrder: 3, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: domain.NewSkillID(), Slug: "order-a", Name: "Order A", SortOrder: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: domain.NewSkillID(), Slug: "order-b", Name: "Order B", SortOrder: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		for _, s := range skills {
			require.NoError(t, repo.Create(ctx, s))
		}

		all, err := repo.List(ctx)
		require.NoError(t, err)

		var orders []int
		for _, s := range all {
			for _, created := range skills {
				if s.ID == created.ID {
					orders = append(orders, s.SortOrder)
				}
			}
		}
		require.Len(t, orders, 3)
		assert.LessOrEqual(t, orders[0], orders[1])
		assert.LessOrEqual(t, orders[1], orders[2])
	})

	t.Run("Contract: Update", func(t *testing.T) {
		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "update-me",
			Name:      "Original Name",
			Content:   "Original content",
			SortOrder: 5,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))

		skill.Name = "Updated Name"
		skill.Content = "Updated content"
		skill.UpdatedAt = time.Now()
		require.NoError(t, repo.Update(ctx, skill))

		found, err := repo.FindByID(ctx, skill.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "Updated Name", found.Name)
		assert.Equal(t, "Updated content", found.Content)
	})

	t.Run("Contract: Delete", func(t *testing.T) {
		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "delete-me",
			Name:      "Delete Me",
			SortOrder: 10,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))

		require.NoError(t, repo.Delete(ctx, skill.ID))

		found, err := repo.FindByID(ctx, skill.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Contract: Delete returns ErrSkillInUse when assigned to agent", func(t *testing.T) {
		if roleID == "" {
			t.Skip("requires seeded role")
		}

		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "inuse-skill",
			Name:      "In-Use Skill",
			SortOrder: 20,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))
		require.NoError(t, repo.AssignToAgent(ctx, roleID, skill.ID))

		err := repo.Delete(ctx, skill.ID)
		assert.ErrorIs(t, err, domain.ErrSkillInUse)
	})

	t.Run("Contract: AssignToAgent and ListByAgent", func(t *testing.T) {
		if roleID == "" {
			t.Skip("requires seeded role")
		}

		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "assign-skill",
			Name:      "Assign Skill",
			SortOrder: 30,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))
		require.NoError(t, repo.AssignToAgent(ctx, roleID, skill.ID))

		list, err := repo.ListByAgent(ctx, roleID)
		require.NoError(t, err)

		found := false
		for _, s := range list {
			if s.ID == skill.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Contract: AssignToAgent idempotency — duplicate returns error", func(t *testing.T) {
		if roleID == "" {
			t.Skip("requires seeded role")
		}

		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "duplicate-assign",
			Name:      "Duplicate Assign",
			SortOrder: 40,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))
		require.NoError(t, repo.AssignToAgent(ctx, roleID, skill.ID))

		err := repo.AssignToAgent(ctx, roleID, skill.ID)
		assert.Error(t, err)
	})

	t.Run("Contract: RemoveFromAgent", func(t *testing.T) {
		if roleID == "" {
			t.Skip("requires seeded role")
		}

		skill := domain.Skill{
			ID:        domain.NewSkillID(),
			Slug:      "remove-skill",
			Name:      "Remove Skill",
			SortOrder: 50,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		require.NoError(t, repo.Create(ctx, skill))
		require.NoError(t, repo.AssignToAgent(ctx, roleID, skill.ID))
		require.NoError(t, repo.RemoveFromAgent(ctx, roleID, skill.ID))

		list, err := repo.ListByAgent(ctx, roleID)
		require.NoError(t, err)

		for _, s := range list {
			assert.NotEqual(t, skill.ID, s.ID)
		}
	})

	t.Run("Contract: RemoveFromAgent on non-existent association", func(t *testing.T) {
		if roleID == "" {
			t.Skip("requires seeded role")
		}

		err := repo.RemoveFromAgent(ctx, roleID, domain.NewSkillID())
		assert.Error(t, err)
	})
}
