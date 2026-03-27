package teamstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockTeamRepository struct {
	CreateFunc     func(ctx context.Context, team domain.Team) error
	FindByIDFunc   func(ctx context.Context, id domain.TeamID) (domain.Team, error)
	FindBySlugFunc func(ctx context.Context, slug string) (domain.Team, error)
	ListFunc       func(ctx context.Context) ([]domain.Team, error)
	UpdateFunc     func(ctx context.Context, team domain.Team) error
	DeleteFunc     func(ctx context.Context, id domain.TeamID) error
}

func (m *MockTeamRepository) Create(ctx context.Context, team domain.Team) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, team)
}

func (m *MockTeamRepository) FindByID(ctx context.Context, id domain.TeamID) (domain.Team, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockTeamRepository) FindBySlug(ctx context.Context, slug string) (domain.Team, error) {
	if m.FindBySlugFunc == nil {
		panic("called not defined FindBySlugFunc")
	}
	return m.FindBySlugFunc(ctx, slug)
}

func (m *MockTeamRepository) List(ctx context.Context) ([]domain.Team, error) {
	if m.ListFunc == nil {
		return nil, nil
	}
	return m.ListFunc(ctx)
}

func (m *MockTeamRepository) Update(ctx context.Context, team domain.Team) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, team)
}

func (m *MockTeamRepository) Delete(ctx context.Context, id domain.TeamID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func TeamsContractTesting(t *testing.T, repo teams.TeamRepository) {
	ctx := context.Background()

	t.Run("Contract: Create stores team and FindByID retrieves it", func(t *testing.T) {
		team := domain.Team{
			ID:          domain.NewTeamID(),
			Name:        "Engineering",
			Slug:        "engineering-" + domain.NewTeamID().String()[:8],
			Description: "Engineering team",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := repo.Create(ctx, team)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, team.ID)
		require.NoError(t, err, "FindByID should succeed for created team")
		assert.Equal(t, team.ID, retrieved.ID, "ID must match")
		assert.Equal(t, team.Name, retrieved.Name, "Name must match")
		assert.Equal(t, team.Slug, retrieved.Slug, "Slug must match")
	})

	t.Run("Contract: FindByID returns error for non-existent team", func(t *testing.T) {
		_, err := repo.FindByID(ctx, domain.NewTeamID())
		assert.Error(t, err, "FindByID should return error for non-existent team")
		assert.ErrorIs(t, err, domain.ErrTeamNotFound, "Error should be ErrTeamNotFound")
	})

	t.Run("Contract: FindBySlug retrieves team by slug", func(t *testing.T) {
		slug := "slug-" + domain.NewTeamID().String()[:8]
		team := domain.Team{
			ID:        domain.NewTeamID(),
			Name:      "Slug Team",
			Slug:      slug,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, team)
		require.NoError(t, err)

		retrieved, err := repo.FindBySlug(ctx, slug)
		require.NoError(t, err, "FindBySlug should succeed")
		assert.Equal(t, team.ID, retrieved.ID, "ID must match")
		assert.Equal(t, slug, retrieved.Slug, "Slug must match")
	})

	t.Run("Contract: FindBySlug returns error for non-existent slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, "nonexistent-slug-xyz")
		assert.Error(t, err, "FindBySlug should return error for non-existent slug")
		assert.ErrorIs(t, err, domain.ErrTeamNotFound, "Error should be ErrTeamNotFound")
	})

	t.Run("Contract: List returns all teams", func(t *testing.T) {
		team := domain.Team{
			ID:        domain.NewTeamID(),
			Name:      "List Team",
			Slug:      "list-" + domain.NewTeamID().String()[:8],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, team)
		require.NoError(t, err)

		list, err := repo.List(ctx)
		require.NoError(t, err, "List should succeed")
		assert.GreaterOrEqual(t, len(list), 1, "Should have at least 1 team")

		found := false
		for _, t2 := range list {
			if t2.ID == team.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created team should appear in list")
	})

	t.Run("Contract: Update modifies team data", func(t *testing.T) {
		team := domain.Team{
			ID:        domain.NewTeamID(),
			Name:      "Original Name",
			Slug:      "update-" + domain.NewTeamID().String()[:8],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, team)
		require.NoError(t, err)

		team.Name = "Updated Name"
		team.UpdatedAt = time.Now()
		err = repo.Update(ctx, team)
		require.NoError(t, err, "Update should succeed")

		updated, err := repo.FindByID(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name, "Name should be updated")
	})

	t.Run("Contract: Delete removes team", func(t *testing.T) {
		team := domain.Team{
			ID:        domain.NewTeamID(),
			Name:      "To Delete",
			Slug:      "delete-" + domain.NewTeamID().String()[:8],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.Delete(ctx, team.ID)
		require.NoError(t, err, "Delete should succeed")

		_, err = repo.FindByID(ctx, team.ID)
		assert.Error(t, err, "Team should not exist after deletion")
		assert.ErrorIs(t, err, domain.ErrTeamNotFound)
	})
}
