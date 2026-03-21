package pg

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/apikeys"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_identity.sql
var migrationSQL string

const queryTimeout = 30 * time.Second

// Repositories holds all identity PostgreSQL repository implementations.
type Repositories struct {
	Users   users.UserRepository
	APIKeys apikeys.APIKeyRepository
	Teams   teams.TeamRepository
}

// NewRepositories creates identity repositories backed by a pgxpool.Pool and runs migrations.
// encKey is the symmetric encryption key used for sensitive columns (pgp_sym_encrypt).
func NewRepositories(ctx context.Context, pool *pgxpool.Pool, encKey string) (*Repositories, error) {
	mCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if _, err := pool.Exec(mCtx, migrationSQL); err != nil {
		return nil, err
	}
	base := &baseRepository{pool: pool, encKey: encKey}
	return &Repositories{
		Users:   &userRepository{base},
		APIKeys: &apiKeyRepository{base},
		Teams:   &teamRepository{base},
	}, nil
}

type baseRepository struct {
	pool   *pgxpool.Pool
	encKey string
}

func (b *baseRepository) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, queryTimeout)
}

// compile-time interface checks
var (
	_ users.UserRepository     = (*userRepository)(nil)
	_ apikeys.APIKeyRepository = (*apiKeyRepository)(nil)
	_ teams.TeamRepository     = (*teamRepository)(nil)
)

// userRepository

type userRepository struct{ *baseRepository }

func (r *userRepository) Create(ctx context.Context, u domain.User) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	var teamID *uuid.UUID
	if u.TeamID != nil {
		id := uuid.UUID(*u.TeamID)
		teamID = &id
	}

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO users (id, email, display_name, password_hash, sso_provider, sso_subject, role, team_id, created_at, updated_at)
		VALUES ($1, $2, $3, pgp_sym_encrypt($4, $9), $5, pgp_sym_encrypt($6, $9), $7, $8, $10, $11)`,
		uuid.UUID(u.ID),
		u.Email,
		u.DisplayName,
		u.PasswordHash,
		u.SSOProvider,
		u.SSOSubject,
		string(u.Role),
		teamID,
		r.encKey,
		u.CreatedAt,
		u.UpdatedAt,
	)
	return err
}

func (r *userRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $2)::text, sso_provider, pgp_sym_decrypt(sso_subject, $2)::text, role, team_id, created_at, updated_at
		FROM users WHERE id = $1`, uuid.UUID(id), r.encKey)
	return scanUser(row)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $2)::text, sso_provider, pgp_sym_decrypt(sso_subject, $2)::text, role, team_id, created_at, updated_at
		FROM users WHERE email = $1`, email, r.encKey)
	return scanUser(row)
}

func (r *userRepository) FindBySSO(ctx context.Context, provider, subject string) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $3)::text, sso_provider, pgp_sym_decrypt(sso_subject, $3)::text, role, team_id, created_at, updated_at
		FROM users WHERE sso_provider = $1 AND pgp_sym_decrypt(sso_subject, $3)::text = $2`, provider, subject, r.encKey)
	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, u domain.User) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	var teamID *uuid.UUID
	if u.TeamID != nil {
		id := uuid.UUID(*u.TeamID)
		teamID = &id
	}

	_, err := r.pool.Exec(qCtx, `
		UPDATE users SET email=$2, display_name=$3, password_hash=pgp_sym_encrypt($4, $9), sso_provider=$5, sso_subject=pgp_sym_encrypt($6, $9), role=$7, team_id=$8, updated_at=$10
		WHERE id=$1`,
		uuid.UUID(u.ID),
		u.Email,
		u.DisplayName,
		u.PasswordHash,
		u.SSOProvider,
		u.SSOSubject,
		string(u.Role),
		teamID,
		r.encKey,
		u.UpdatedAt,
	)
	return err
}

func (r *userRepository) Delete(ctx context.Context, id domain.UserID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(qCtx, `DELETE FROM users WHERE id = $1`, uuid.UUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) ListAll(ctx context.Context) ([]domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, email, display_name, sso_provider, role, team_id, created_at, updated_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsersWithoutHash(rows)
}

func (r *userRepository) ListByTeam(ctx context.Context, teamID domain.TeamID) ([]domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $2)::text, sso_provider, pgp_sym_decrypt(sso_subject, $2)::text, role, team_id, created_at, updated_at
		FROM users WHERE team_id = $1 ORDER BY created_at ASC`, uuid.UUID(teamID), r.encKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func scanUsers(rows pgx.Rows) ([]domain.User, error) {
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func scanUsersWithoutHash(rows pgx.Rows) ([]domain.User, error) {
	var out []domain.User
	for rows.Next() {
		u, err := scanUserWithoutHash(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func scanUser(row pgx.Row) (domain.User, error) {
	var (
		id           uuid.UUID
		email        string
		displayName  string
		passwordHash string
		ssoProvider  string
		ssoSubject   string
		role         string
		teamID       *uuid.UUID
		createdAt    time.Time
		updatedAt    time.Time
	)

	err := row.Scan(&id, &email, &displayName, &passwordHash, &ssoProvider, &ssoSubject, &role, &teamID, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}

	u := domain.User{
		ID:           domain.UserID(id),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		SSOProvider:  ssoProvider,
		SSOSubject:   ssoSubject,
		Role:         domain.MemberRole(role),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
	if teamID != nil {
		tid := domain.TeamID(*teamID)
		u.TeamID = &tid
	}
	return u, nil
}

func scanUserWithoutHash(row pgx.Row) (domain.User, error) {
	var (
		id          uuid.UUID
		email       string
		displayName string
		ssoProvider string
		role        string
		teamID      *uuid.UUID
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := row.Scan(&id, &email, &displayName, &ssoProvider, &role, &teamID, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}

	u := domain.User{
		ID:          domain.UserID(id),
		Email:       email,
		DisplayName: displayName,
		SSOProvider: ssoProvider,
		Role:        domain.MemberRole(role),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if teamID != nil {
		tid := domain.TeamID(*teamID)
		u.TeamID = &tid
	}
	return u, nil
}

// apiKeyRepository

type apiKeyRepository struct{ *baseRepository }

func (r *apiKeyRepository) Create(ctx context.Context, k domain.APIKey) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	scopesJSON, err := json.Marshal(k.Scopes)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(qCtx, `
		INSERT INTO api_keys (id, user_id, name, key_hash, scopes, expires_at, last_used_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.UUID(k.ID),
		uuid.UUID(k.UserID),
		k.Name,
		k.KeyHash,
		scopesJSON,
		k.ExpiresAt,
		k.LastUsedAt,
		k.RevokedAt,
		k.CreatedAt,
	)
	return err
}

func (r *apiKeyRepository) FindByHash(ctx context.Context, keyHash string) (domain.APIKey, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used_at, revoked_at, created_at
		FROM api_keys WHERE key_hash = $1`, keyHash)
	return scanAPIKey(row)
}

func (r *apiKeyRepository) FindByID(ctx context.Context, id domain.APIKeyID) (domain.APIKey, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used_at, revoked_at, created_at
		FROM api_keys WHERE id = $1`, uuid.UUID(id))
	return scanAPIKey(row)
}

func (r *apiKeyRepository) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.APIKey, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, user_id, name, key_hash, scopes, expires_at, last_used_at, revoked_at, created_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, uuid.UUID(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, id domain.APIKeyID, _ time.Time) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, uuid.UUID(id))
	return err
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id domain.APIKeyID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(qCtx, `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1`, uuid.UUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(row rowScanner) (domain.APIKey, error) {
	var (
		id         uuid.UUID
		userID     uuid.UUID
		name       string
		keyHash    string
		scopesJSON []byte
		expiresAt  *time.Time
		lastUsedAt *time.Time
		revokedAt  *time.Time
		createdAt  time.Time
	)

	err := row.Scan(&id, &userID, &name, &keyHash, &scopesJSON, &expiresAt, &lastUsedAt, &revokedAt, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.APIKey{}, domain.ErrAPIKeyNotFound
		}
		return domain.APIKey{}, err
	}

	var scopes []string
	if err := json.Unmarshal(scopesJSON, &scopes); err != nil {
		scopes = []string{}
	}

	return domain.APIKey{
		ID:         domain.APIKeyID(id),
		UserID:     domain.UserID(userID),
		Name:       name,
		KeyHash:    keyHash,
		Scopes:     scopes,
		ExpiresAt:  expiresAt,
		LastUsedAt: lastUsedAt,
		RevokedAt:  revokedAt,
		CreatedAt:  createdAt,
	}, nil
}

// teamRepository

type teamRepository struct{ *baseRepository }

func (r *teamRepository) Create(ctx context.Context, t domain.Team) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO teams (id, name, slug, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.UUID(t.ID), t.Name, t.Slug, t.Description, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *teamRepository) FindByID(ctx context.Context, id domain.TeamID) (domain.Team, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, name, slug, description, created_at, updated_at FROM teams WHERE id = $1`,
		uuid.UUID(id))
	return scanTeam(row)
}

func (r *teamRepository) FindBySlug(ctx context.Context, slug string) (domain.Team, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, name, slug, description, created_at, updated_at FROM teams WHERE slug = $1`, slug)
	return scanTeam(row)
}

func (r *teamRepository) List(ctx context.Context) ([]domain.Team, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, name, slug, description, created_at, updated_at FROM teams ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Team
	for rows.Next() {
		t, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *teamRepository) Update(ctx context.Context, t domain.Team) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		UPDATE teams SET name=$2, slug=$3, description=$4, updated_at=$5 WHERE id=$1`,
		uuid.UUID(t.ID), t.Name, t.Slug, t.Description, t.UpdatedAt,
	)
	return err
}

func (r *teamRepository) Delete(ctx context.Context, id domain.TeamID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(qCtx, `DELETE FROM teams WHERE id = $1`, uuid.UUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTeamNotFound
	}
	return nil
}

func scanTeam(row rowScanner) (domain.Team, error) {
	var (
		id          uuid.UUID
		name        string
		slug        string
		description string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := row.Scan(&id, &name, &slug, &description, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Team{}, domain.ErrTeamNotFound
		}
		return domain.Team{}, err
	}
	return domain.Team{
		ID:          domain.TeamID(id),
		Name:        name,
		Slug:        slug,
		Description: description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
