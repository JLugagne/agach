package apikeystest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAPIKeyRepository struct {
	CreateFunc         func(ctx context.Context, key domain.APIKey) error
	FindByIDFunc       func(ctx context.Context, id domain.APIKeyID) (domain.APIKey, error)
	FindByHashFunc     func(ctx context.Context, hash string) (domain.APIKey, error)
	RevokeFunc         func(ctx context.Context, id domain.APIKeyID) error
	UpdateLastUsedFunc func(ctx context.Context, id domain.APIKeyID, at time.Time) error
	ListByUserFunc     func(ctx context.Context, userID domain.UserID) ([]domain.APIKey, error)
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, key domain.APIKey) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, key)
}

func (m *MockAPIKeyRepository) FindByID(ctx context.Context, id domain.APIKeyID) (domain.APIKey, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockAPIKeyRepository) FindByHash(ctx context.Context, hash string) (domain.APIKey, error) {
	if m.FindByHashFunc == nil {
		panic("called not defined FindByHashFunc")
	}
	return m.FindByHashFunc(ctx, hash)
}

func (m *MockAPIKeyRepository) Revoke(ctx context.Context, id domain.APIKeyID) error {
	if m.RevokeFunc == nil {
		panic("called not defined RevokeFunc")
	}
	return m.RevokeFunc(ctx, id)
}

func (m *MockAPIKeyRepository) UpdateLastUsed(ctx context.Context, id domain.APIKeyID, at time.Time) error {
	if m.UpdateLastUsedFunc == nil {
		return nil
	}
	return m.UpdateLastUsedFunc(ctx, id, at)
}

func (m *MockAPIKeyRepository) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.APIKey, error) {
	if m.ListByUserFunc == nil {
		panic("called not defined ListByUserFunc")
	}
	return m.ListByUserFunc(ctx, userID)
}

func keyHash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}

// APIKeysContractTesting runs a standard contract test suite against any APIKeyRepository implementation.
// userID must refer to an existing user in the underlying store.
func APIKeysContractTesting(t *testing.T, repo apikeys.APIKeyRepository, userID domain.UserID) {
	t.Helper()
	ctx := context.Background()

	t.Run("Contract: Create stores key and FindByID retrieves it", func(t *testing.T) {
		raw := "rawkey-" + domain.NewAPIKeyID().String()
		key := domain.APIKey{
			ID:        domain.NewAPIKeyID(),
			UserID:    userID,
			Name:      "test-key",
			KeyHash:   keyHash(raw),
			Scopes:    []string{"read", "write"},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, key)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, key.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, key.ID, retrieved.ID)
		assert.Equal(t, key.Name, retrieved.Name)
		assert.Equal(t, key.KeyHash, retrieved.KeyHash)
	})

	t.Run("Contract: FindByID returns error for non-existent key", func(t *testing.T) {
		_, err := repo.FindByID(ctx, domain.NewAPIKeyID())
		assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
	})

	t.Run("Contract: FindByHash retrieves key by hash", func(t *testing.T) {
		raw := "hashkey-" + domain.NewAPIKeyID().String()
		hash := keyHash(raw)
		key := domain.APIKey{
			ID:        domain.NewAPIKeyID(),
			UserID:    userID,
			Name:      "hash-key",
			KeyHash:   hash,
			Scopes:    []string{},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, key)
		require.NoError(t, err)

		retrieved, err := repo.FindByHash(ctx, hash)
		require.NoError(t, err, "FindByHash should succeed")
		assert.Equal(t, key.ID, retrieved.ID)
	})

	t.Run("Contract: FindByHash returns error for non-existent hash", func(t *testing.T) {
		_, err := repo.FindByHash(ctx, "nonexistent-hash-xyz")
		assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
	})

	t.Run("Contract: ListByUser returns keys for user", func(t *testing.T) {
		raw := "listkey-" + domain.NewAPIKeyID().String()
		key := domain.APIKey{
			ID:        domain.NewAPIKeyID(),
			UserID:    userID,
			Name:      "list-key",
			KeyHash:   keyHash(raw),
			Scopes:    []string{},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, key)
		require.NoError(t, err)

		list, err := repo.ListByUser(ctx, userID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 1)

		found := false
		for _, k := range list {
			if k.ID == key.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created key should appear in ListByUser")
	})

	t.Run("Contract: Revoke marks key as revoked", func(t *testing.T) {
		raw := "revokekey-" + domain.NewAPIKeyID().String()
		key := domain.APIKey{
			ID:        domain.NewAPIKeyID(),
			UserID:    userID,
			Name:      "revoke-key",
			KeyHash:   keyHash(raw),
			Scopes:    []string{},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, key)
		require.NoError(t, err)

		err = repo.Revoke(ctx, key.ID)
		require.NoError(t, err, "Revoke should succeed")

		retrieved, err := repo.FindByID(ctx, key.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.RevokedAt, "RevokedAt should be set after revoke")
	})

	t.Run("Contract: Revoke returns error for non-existent key", func(t *testing.T) {
		err := repo.Revoke(ctx, domain.NewAPIKeyID())
		assert.ErrorIs(t, err, domain.ErrAPIKeyNotFound)
	})

	t.Run("Contract: UpdateLastUsed sets last_used_at", func(t *testing.T) {
		raw := "lastused-" + domain.NewAPIKeyID().String()
		key := domain.APIKey{
			ID:        domain.NewAPIKeyID(),
			UserID:    userID,
			Name:      "lastused-key",
			KeyHash:   keyHash(raw),
			Scopes:    []string{},
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, key)
		require.NoError(t, err)

		now := time.Now()
		err = repo.UpdateLastUsed(ctx, key.ID, now)
		require.NoError(t, err, "UpdateLastUsed should succeed")

		retrieved, err := repo.FindByID(ctx, key.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.LastUsedAt, "LastUsedAt should be set")
	})
}
