package userstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockUserRepository struct {
	CreateFunc      func(ctx context.Context, user domain.User) error
	FindByIDFunc    func(ctx context.Context, id domain.UserID) (domain.User, error)
	FindByEmailFunc func(ctx context.Context, email string) (domain.User, error)
	UpdateFunc      func(ctx context.Context, user domain.User) error
	ListAllFunc     func(ctx context.Context) ([]domain.User, error)
	ListByTeamFunc  func(ctx context.Context, teamID domain.TeamID) ([]domain.User, error)
	FindBySSOFunc   func(ctx context.Context, provider, subject string) (domain.User, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user domain.User) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, user)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	if m.FindByEmailFunc == nil {
		panic("called not defined FindByEmailFunc")
	}
	return m.FindByEmailFunc(ctx, email)
}

func (m *MockUserRepository) Update(ctx context.Context, user domain.User) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, user)
}

func (m *MockUserRepository) ListAll(ctx context.Context) ([]domain.User, error) {
	if m.ListAllFunc == nil {
		panic("called not defined ListAllFunc")
	}
	return m.ListAllFunc(ctx)
}

func (m *MockUserRepository) ListByTeam(ctx context.Context, teamID domain.TeamID) ([]domain.User, error) {
	if m.ListByTeamFunc == nil {
		panic("called not defined ListByTeamFunc")
	}
	return m.ListByTeamFunc(ctx, teamID)
}

func (m *MockUserRepository) FindBySSO(ctx context.Context, provider, subject string) (domain.User, error) {
	if m.FindBySSOFunc == nil {
		panic("called not defined FindBySSOFunc")
	}
	return m.FindBySSOFunc(ctx, provider, subject)
}

// UsersContractTesting runs a standard contract test suite against any UserRepository implementation.
// teamID must refer to an existing team in the underlying store (or be a zero value if teams are not required).
func UsersContractTesting(t *testing.T, repo users.UserRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("Contract: Create stores user and FindByID retrieves it", func(t *testing.T) {
		user := domain.User{
			ID:          domain.NewUserID(),
			Email:       "user-" + domain.NewUserID().String()[:8] + "@example.com",
			DisplayName: "Test User",
			Role:        domain.RoleMember,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.DisplayName, retrieved.DisplayName)
	})

	t.Run("Contract: FindByID returns error for non-existent user", func(t *testing.T) {
		_, err := repo.FindByID(ctx, domain.NewUserID())
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Contract: FindByEmail retrieves user by email", func(t *testing.T) {
		email := "email-" + domain.NewUserID().String()[:8] + "@example.com"
		user := domain.User{
			ID:        domain.NewUserID(),
			Email:     email,
			Role:      domain.RoleMember,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		retrieved, err := repo.FindByEmail(ctx, email)
		require.NoError(t, err, "FindByEmail should succeed")
		assert.Equal(t, user.ID, retrieved.ID)
	})

	t.Run("Contract: FindByEmail returns error for non-existent email", func(t *testing.T) {
		_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Contract: Update modifies user data", func(t *testing.T) {
		user := domain.User{
			ID:          domain.NewUserID(),
			Email:       "update-" + domain.NewUserID().String()[:8] + "@example.com",
			DisplayName: "Before Update",
			Role:        domain.RoleMember,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		user.DisplayName = "After Update"
		user.UpdatedAt = time.Now()
		err = repo.Update(ctx, user)
		require.NoError(t, err, "Update should succeed")

		retrieved, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "After Update", retrieved.DisplayName)
	})

	t.Run("Contract: ListAll returns at least one user", func(t *testing.T) {
		user := domain.User{
			ID:        domain.NewUserID(),
			Email:     "list-" + domain.NewUserID().String()[:8] + "@example.com",
			Role:      domain.RoleMember,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		list, err := repo.ListAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 1)

		found := false
		for _, u := range list {
			if u.ID == user.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created user should appear in ListAll")
	})

	t.Run("Contract: FindBySSO retrieves user by provider and subject", func(t *testing.T) {
		provider := "github"
		subject := "sub-" + domain.NewUserID().String()[:8]
		user := domain.User{
			ID:          domain.NewUserID(),
			Email:       "sso-" + domain.NewUserID().String()[:8] + "@example.com",
			SSOProvider: provider,
			SSOSubject:  subject,
			Role:        domain.RoleMember,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		retrieved, err := repo.FindBySSO(ctx, provider, subject)
		require.NoError(t, err, "FindBySSO should succeed")
		assert.Equal(t, user.ID, retrieved.ID)
	})

	t.Run("Contract: FindBySSO returns error for non-existent provider/subject", func(t *testing.T) {
		_, err := repo.FindBySSO(ctx, "github", "nonexistent-subject-xyz")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
