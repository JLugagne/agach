package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills/skillstest"
	"github.com/JLugagne/agach-mcp/internal/server/outbound/pg"
	"github.com/stretchr/testify/require"
)

func setupTestRepositories(t *testing.T) *pg.Repositories {
	t.Helper()
	pool := newTestPool(t)
	repos, err := pg.NewRepositories(pool)
	require.NoError(t, err)
	return repos
}

func TestSkillContract(t *testing.T) {
	ctx := context.Background()
	repo := setupTestRepositories(t)

	roleID := domain.NewRoleID()
	err := repo.Agents.Create(ctx, domain.Role{
		ID:        roleID,
		Slug:      "skill-test-agent",
		Name:      "Skill Test Agent",
		SortOrder: 0,
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)

	skillstest.SkillContractTesting(t, repo.Skills, roleID)
}
