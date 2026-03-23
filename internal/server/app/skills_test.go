package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills/skillstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAppWithSkills(mock *skillstest.MockSkill) *app.App {
	return app.NewApp(app.Config{
		Skills: mock,
	})
}

func TestCreateSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock := &skillstest.MockSkill{
			FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
				return nil, nil
			},
			CreateFunc: func(_ context.Context, _ domain.Skill) error {
				return nil
			},
		}
		a := newAppWithSkills(mock)

		skill, err := a.CreateSkill(ctx, "go-tools", "Go Tools", "desc", "content", "icon", "#fff", 1)
		require.NoError(t, err)
		assert.Equal(t, "go-tools", skill.Slug)
		assert.Equal(t, "Go Tools", skill.Name)
		assert.NotEmpty(t, skill.ID)
	})

	t.Run("slug required", func(t *testing.T) {
		a := newAppWithSkills(&skillstest.MockSkill{})

		_, err := a.CreateSkill(ctx, "", "Go Tools", "", "", "", "", 0)
		assert.ErrorIs(t, err, domain.ErrSkillSlugRequired)
	})

	t.Run("name required", func(t *testing.T) {
		a := newAppWithSkills(&skillstest.MockSkill{})

		_, err := a.CreateSkill(ctx, "go-tools", "", "", "", "", "", 0)
		assert.ErrorIs(t, err, domain.ErrSkillNameRequired)
	})

	t.Run("already exists", func(t *testing.T) {
		existing := &domain.Skill{ID: domain.NewSkillID(), Slug: "go-tools", Name: "Go Tools"}
		mock := &skillstest.MockSkill{
			FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
				return existing, nil
			},
		}
		a := newAppWithSkills(mock)

		_, err := a.CreateSkill(ctx, "go-tools", "Go Tools", "", "", "", "", 0)
		assert.ErrorIs(t, err, domain.ErrSkillAlreadyExists)
	})
}

func TestUpdateSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		skillID := domain.NewSkillID()
		original := &domain.Skill{
			ID:   skillID,
			Slug: "go-tools",
			Name: "Go Tools",
		}
		var updated domain.Skill
		mock := &skillstest.MockSkill{
			FindByIDFunc: func(_ context.Context, _ domain.SkillID) (*domain.Skill, error) {
				return original, nil
			},
			UpdateFunc: func(_ context.Context, s domain.Skill) error {
				updated = s
				return nil
			},
		}
		a := newAppWithSkills(mock)

		err := a.UpdateSkill(ctx, skillID, "New Name", "new desc", "new content", "new-icon", "#000", 5)
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
		assert.Equal(t, "new desc", updated.Description)
		assert.Equal(t, "new content", updated.Content)
		assert.Equal(t, "new-icon", updated.Icon)
		assert.Equal(t, "#000", updated.Color)
		assert.Equal(t, 5, updated.SortOrder)
	})

	t.Run("not found", func(t *testing.T) {
		mock := &skillstest.MockSkill{
			FindByIDFunc: func(_ context.Context, _ domain.SkillID) (*domain.Skill, error) {
				return nil, nil
			},
		}
		a := newAppWithSkills(mock)

		err := a.UpdateSkill(ctx, domain.NewSkillID(), "Name", "", "", "", "", 0)
		assert.ErrorIs(t, err, domain.ErrSkillNotFound)
	})
}

func TestDeleteSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		skillID := domain.NewSkillID()
		skill := &domain.Skill{ID: skillID, Slug: "go-tools", Name: "Go Tools"}
		mock := &skillstest.MockSkill{
			FindByIDFunc: func(_ context.Context, _ domain.SkillID) (*domain.Skill, error) {
				return skill, nil
			},
			DeleteFunc: func(_ context.Context, _ domain.SkillID) error {
				return nil
			},
		}
		a := newAppWithSkills(mock)

		err := a.DeleteSkill(ctx, skillID)
		require.NoError(t, err)
	})

	t.Run("in use", func(t *testing.T) {
		skillID := domain.NewSkillID()
		skill := &domain.Skill{ID: skillID, Slug: "go-tools", Name: "Go Tools"}
		mock := &skillstest.MockSkill{
			FindByIDFunc: func(_ context.Context, _ domain.SkillID) (*domain.Skill, error) {
				return skill, nil
			},
			DeleteFunc: func(_ context.Context, _ domain.SkillID) error {
				return domain.ErrSkillInUse
			},
		}
		a := newAppWithSkills(mock)

		err := a.DeleteSkill(ctx, skillID)
		assert.ErrorIs(t, err, domain.ErrSkillInUse)
	})
}

func TestListSkills(t *testing.T) {
	ctx := context.Background()

	t.Run("returns skills from repo", func(t *testing.T) {
		expected := []domain.Skill{
			{ID: domain.NewSkillID(), Slug: "skill-a", Name: "Skill A"},
			{ID: domain.NewSkillID(), Slug: "skill-b", Name: "Skill B"},
		}
		mock := &skillstest.MockSkill{
			ListFunc: func(_ context.Context) ([]domain.Skill, error) {
				return expected, nil
			},
		}
		a := newAppWithSkills(mock)

		list, err := a.ListSkills(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 2)
		assert.Equal(t, expected[0].Slug, list[0].Slug)
		assert.Equal(t, expected[1].Slug, list[1].Slug)
	})

	t.Run("propagates repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		mock := &skillstest.MockSkill{
			ListFunc: func(_ context.Context) ([]domain.Skill, error) {
				return nil, repoErr
			},
		}
		a := newAppWithSkills(mock)

		_, err := a.ListSkills(ctx)
		assert.ErrorIs(t, err, repoErr)
	})
}
