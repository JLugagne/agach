package pg_test

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestOnboardingCode(createdBy domain.UserID) domain.OnboardingCode {
	return domain.OnboardingCode{
		ID:              domain.NewOnboardingCodeID(),
		Code:            "code-" + domain.NewOnboardingCodeID().String(),
		CreatedByUserID: createdBy,
		NodeMode:        domain.NodeModeDefault,
		NodeName:        "test-node",
		ExpiresAt:       time.Now().Add(1 * time.Hour).UTC().Truncate(time.Millisecond),
		CreatedAt:       time.Now().UTC().Truncate(time.Millisecond),
	}
}

func TestOnboardingCodeRepository_CreateAndFindByCode(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	code := makeTestOnboardingCode(user.ID)
	require.NoError(t, repos.OnboardingCodes.Create(ctx, code))

	found, err := repos.OnboardingCodes.FindByCode(ctx, code.Code)
	require.NoError(t, err)
	assert.Equal(t, code.ID, found.ID)
	assert.Equal(t, code.Code, found.Code)
	assert.Equal(t, code.NodeMode, found.NodeMode)
	assert.Equal(t, code.NodeName, found.NodeName)
}

func TestOnboardingCodeRepository_FindByCode_NotFound(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	_, err := repos.OnboardingCodes.FindByCode(ctx, "nonexistent-code")
	assert.ErrorIs(t, err, domain.ErrOnboardingCodeNotFound)
}

func TestOnboardingCodeRepository_FindByCode_UnusedOnly(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	code := makeTestOnboardingCode(user.ID)
	require.NoError(t, repos.OnboardingCodes.Create(ctx, code))
	require.NoError(t, repos.OnboardingCodes.MarkUsed(ctx, code.ID, node.ID))

	// FindByCode should not return used codes
	_, err := repos.OnboardingCodes.FindByCode(ctx, code.Code)
	assert.ErrorIs(t, err, domain.ErrOnboardingCodeNotFound)
}

func TestOnboardingCodeRepository_MarkUsed_Success(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	code := makeTestOnboardingCode(user.ID)
	require.NoError(t, repos.OnboardingCodes.Create(ctx, code))

	err := repos.OnboardingCodes.MarkUsed(ctx, code.ID, node.ID)
	require.NoError(t, err)
}

func TestOnboardingCodeRepository_MarkUsed_AlreadyUsed(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	code := makeTestOnboardingCode(user.ID)
	require.NoError(t, repos.OnboardingCodes.Create(ctx, code))

	require.NoError(t, repos.OnboardingCodes.MarkUsed(ctx, code.ID, node.ID))
	err := repos.OnboardingCodes.MarkUsed(ctx, code.ID, node.ID)
	assert.ErrorIs(t, err, domain.ErrOnboardingCodeUsed)
}

func TestOnboardingCodeRepository_MarkUsed_Expired(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	node := makeTestNode(user.ID)
	require.NoError(t, repos.Nodes.Create(ctx, node))

	code := makeTestOnboardingCode(user.ID)
	code.ExpiresAt = time.Now().Add(-1 * time.Hour).UTC() // already expired
	require.NoError(t, repos.OnboardingCodes.Create(ctx, code))

	err := repos.OnboardingCodes.MarkUsed(ctx, code.ID, node.ID)
	assert.ErrorIs(t, err, domain.ErrOnboardingCodeExpired)
}

func TestOnboardingCodeRepository_DeleteExpired(t *testing.T) {
	repos := newTestRepos(t)
	ctx := context.Background()

	user := makeTestUser(t)
	require.NoError(t, repos.Users.Create(ctx, user))

	expiredCode := makeTestOnboardingCode(user.ID)
	expiredCode.ExpiresAt = time.Now().Add(-1 * time.Hour).UTC()

	validCode := makeTestOnboardingCode(user.ID)

	require.NoError(t, repos.OnboardingCodes.Create(ctx, expiredCode))
	require.NoError(t, repos.OnboardingCodes.Create(ctx, validCode))

	deleted, err := repos.OnboardingCodes.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Valid code should still be findable
	_, err = repos.OnboardingCodes.FindByCode(ctx, validCode.Code)
	require.NoError(t, err)

	// Expired code should be gone (not found)
	_, err = repos.OnboardingCodes.FindByCode(ctx, expiredCode.Code)
	assert.ErrorIs(t, err, domain.ErrOnboardingCodeNotFound)
}
