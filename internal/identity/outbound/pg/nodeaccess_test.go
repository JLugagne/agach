package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeAccessRepository_GrantUser_AndListByNode(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	owner := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, owner))

	grantee := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, grantee))

	node := makeTestNode(owner.ID)
	node.Mode = domain.NodeModeShared
	require.NoError(t, repos.Nodes.Create(ctx, node))

	require.NoError(t, repos.NodeAccess.GrantUser(ctx, node.ID, grantee.ID))

	list, err := repos.NodeAccess.ListByNode(ctx, node.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].UserID)
	assert.Equal(t, grantee.ID, *list[0].UserID)
	assert.Nil(t, list[0].TeamID)
}

func TestNodeAccessRepository_GrantTeam(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	owner := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, owner))

	team := domain.Team{
		ID:          domain.NewTeamID(),
		Name:        "test-team",
		Slug:        "test-team-" + domain.NewTeamID().String(),
		Description: "",
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, repos.Teams.Create(ctx, team))

	node := makeTestNode(owner.ID)
	node.Mode = domain.NodeModeShared
	require.NoError(t, repos.Nodes.Create(ctx, node))

	require.NoError(t, repos.NodeAccess.GrantTeam(ctx, node.ID, team.ID))

	list, err := repos.NodeAccess.ListByNode(ctx, node.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].TeamID)
	assert.Equal(t, team.ID, *list[0].TeamID)
	assert.Nil(t, list[0].UserID)
}

func TestNodeAccessRepository_RevokeUser(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	owner := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, owner))

	grantee := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, grantee))

	node := makeTestNode(owner.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	require.NoError(t, repos.NodeAccess.GrantUser(ctx, node.ID, grantee.ID))
	require.NoError(t, repos.NodeAccess.RevokeUser(ctx, node.ID, grantee.ID))

	list, err := repos.NodeAccess.ListByNode(ctx, node.ID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestNodeAccessRepository_HasAccess_Direct(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	owner := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, owner))

	grantee := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, grantee))

	node := makeTestNode(owner.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	// No access yet
	has, err := repos.NodeAccess.HasAccess(ctx, node.ID, grantee.ID)
	require.NoError(t, err)
	assert.False(t, has)

	require.NoError(t, repos.NodeAccess.GrantUser(ctx, node.ID, grantee.ID))

	has, err = repos.NodeAccess.HasAccess(ctx, node.ID, grantee.ID)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestNodeAccessRepository_HasAccess_ViaTeam(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	// Create a team
	team := domain.Team{
		ID:          domain.NewTeamID(),
		Name:        "access-team",
		Slug:        "access-team-" + domain.NewTeamID().String(),
		Description: "",
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, repos.Teams.Create(ctx, team))

	// Create owner (no team)
	owner := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, owner))

	// Create a user belonging to the team
	teamUser := makeTestUser(t)
	teamUser.TeamID = &team.ID
	require.NoError(t, repos.Users.Create(ctx, teamUser))

	node := makeTestNode(owner.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	// Grant access to the team
	require.NoError(t, repos.NodeAccess.GrantTeam(ctx, node.ID, team.ID))

	// Team member should have access
	has, err := repos.NodeAccess.HasAccess(ctx, node.ID, teamUser.ID)
	require.NoError(t, err)
	assert.True(t, has)
}
