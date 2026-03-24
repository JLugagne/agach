package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestUser(t *testing.T) domain.User {
	t.Helper()
	return domain.User{
		ID:           domain.NewUserID(),
		Email:        "node-owner-" + domain.NewUserID().String() + "@example.com",
		DisplayName:  "Node Owner",
		PasswordHash: "hash",
		Role:         domain.RoleMember,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}
}

func makeTestNode(ownerID domain.UserID) domain.Node {
	return domain.Node{
		ID:               domain.NewNodeID(),
		OwnerUserID:      ownerID,
		Name:             "test-node",
		Mode:             domain.NodeModeDefault,
		Status:           domain.NodeStatusActive,
		RefreshTokenHash: "tokenhash",
		CreatedAt:        time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:        time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestNodeRepository_CreateAndFindByID(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	found, err := repos.Nodes.FindByID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, node.ID, found.ID)
	assert.Equal(t, node.OwnerUserID, found.OwnerUserID)
	assert.Equal(t, node.Name, found.Name)
	assert.Equal(t, node.Mode, found.Mode)
	assert.Equal(t, node.Status, found.Status)
	assert.Equal(t, node.RefreshTokenHash, found.RefreshTokenHash)
}

func TestNodeRepository_FindByID_NotFound(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	_, err := repos.Nodes.FindByID(ctx, domain.NewNodeID())
	assert.ErrorIs(t, err, domain.ErrNodeNotFound)
}

func TestNodeRepository_ListByOwner(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node1 := makeTestNode(user.ID)
	node1.Name = "node-1"
	node2 := makeTestNode(user.ID)
	node2.Name = "node-2"
	require.NoError(t, repos.Nodes.Create(ctx, node1))
	require.NoError(t, repos.Nodes.Create(ctx, node2))

	list, err := repos.Nodes.ListByOwner(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestNodeRepository_ListActiveByOwner(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	activeNode := makeTestNode(user.ID)
	activeNode.Name = "active"

	revokedNode := makeTestNode(user.ID)
	revokedNode.Name = "revoked"
	revokedNode.Status = domain.NodeStatusRevoked
	now := time.Now().UTC()
	revokedNode.RevokedAt = &now

	require.NoError(t, repos.Nodes.Create(ctx, activeNode))
	require.NoError(t, repos.Nodes.Create(ctx, revokedNode))

	list, err := repos.Nodes.ListActiveByOwner(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, activeNode.ID, list[0].ID)
}

func TestNodeRepository_Update(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	node.Name = "updated-name"
	node.RefreshTokenHash = "newhash"
	node.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, repos.Nodes.Update(ctx, node))

	found, err := repos.Nodes.FindByID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", found.Name)
	assert.Equal(t, "newhash", found.RefreshTokenHash)
}

func TestNodeRepository_UpdateLastSeen(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	require.NoError(t, repos.Nodes.UpdateLastSeen(ctx, node.ID))

	found, err := repos.Nodes.FindByID(ctx, node.ID)
	require.NoError(t, err)
	assert.NotNil(t, found.LastSeenAt)
}
