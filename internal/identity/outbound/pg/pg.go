package pg

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodeaccess"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/onboardingcodes"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_identity.sql
var migrationSQL string

//go:embed migrations/002_nodes.sql
var migration002SQL string

//go:embed migrations/003_team_members.sql
var migration003SQL string

//go:embed migrations/004_user_blocked.sql
var migration004SQL string

const queryTimeout = 30 * time.Second

// Repositories holds all identity PostgreSQL repository implementations.
type Repositories struct {
	Users           users.UserRepository
	Teams           teams.TeamRepository
	Nodes           nodes.NodeRepository
	OnboardingCodes onboardingcodes.OnboardingCodeRepository
	NodeAccess      nodeaccess.NodeAccessRepository
}

// NewRepositories creates identity repositories backed by a pgxpool.Pool and runs migrations.
// encKey is the symmetric encryption key used for sensitive columns (pgp_sym_encrypt).
func NewRepositories(ctx context.Context, pool *pgxpool.Pool, encKey string) (*Repositories, error) {
	mCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	if _, err := pool.Exec(mCtx, migrationSQL); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(mCtx, migration002SQL); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(mCtx, migration003SQL); err != nil {
		return nil, err
	}
	if _, err := pool.Exec(mCtx, migration004SQL); err != nil {
		return nil, err
	}
	base := &baseRepository{pool: pool, encKey: encKey}
	return &Repositories{
		Users:           &userRepository{base},
		Teams:           &teamRepository{base},
		Nodes:           &pgNodeRepository{base},
		OnboardingCodes: &pgOnboardingCodeRepository{base},
		NodeAccess:      &pgNodeAccessRepository{base},
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
	_ users.UserRepository                     = (*userRepository)(nil)
	_ teams.TeamRepository                     = (*teamRepository)(nil)
	_ nodes.NodeRepository                     = (*pgNodeRepository)(nil)
	_ onboardingcodes.OnboardingCodeRepository = (*pgOnboardingCodeRepository)(nil)
	_ nodeaccess.NodeAccessRepository          = (*pgNodeAccessRepository)(nil)
)

// userRepository

type userRepository struct{ *baseRepository }

func (r *userRepository) Create(ctx context.Context, u domain.User) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO users (id, email, display_name, password_hash, sso_provider, sso_subject, role, created_at, updated_at, blocked_at)
		VALUES ($1, $2, $3, pgp_sym_encrypt($4, $8), $5, pgp_sym_encrypt($6, $8), $7, $9, $10, $11)`,
		uuid.UUID(u.ID),
		u.Email,
		u.DisplayName,
		u.PasswordHash,
		u.SSOProvider,
		u.SSOSubject,
		string(u.Role),
		r.encKey,
		u.CreatedAt,
		u.UpdatedAt,
		u.BlockedAt,
	)
	return err
}

func (r *userRepository) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $2)::text, sso_provider, pgp_sym_decrypt(sso_subject, $2)::text, role, created_at, updated_at, blocked_at
		FROM users WHERE id = $1`, uuid.UUID(id), r.encKey)
	return scanUser(row)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $2)::text, sso_provider, pgp_sym_decrypt(sso_subject, $2)::text, role, created_at, updated_at, blocked_at
		FROM users WHERE email = $1`, email, r.encKey)
	return scanUser(row)
}

func (r *userRepository) FindBySSO(ctx context.Context, provider, subject string) (domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, email, display_name, pgp_sym_decrypt(password_hash, $3)::text, sso_provider, pgp_sym_decrypt(sso_subject, $3)::text, role, created_at, updated_at, blocked_at
		FROM users WHERE sso_provider = $1 AND pgp_sym_decrypt(sso_subject, $3)::text = $2`, provider, subject, r.encKey)
	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, u domain.User) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		UPDATE users SET email=$2, display_name=$3, password_hash=pgp_sym_encrypt($4, $8), sso_provider=$5, sso_subject=pgp_sym_encrypt($6, $8), role=$7, updated_at=$9, blocked_at=$10
		WHERE id=$1`,
		uuid.UUID(u.ID),
		u.Email,
		u.DisplayName,
		u.PasswordHash,
		u.SSOProvider,
		u.SSOSubject,
		string(u.Role),
		r.encKey,
		u.UpdatedAt,
		u.BlockedAt,
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
		SELECT id, email, display_name, sso_provider, role, created_at, updated_at, blocked_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users, err := scanUsersWithoutHash(rows)
	if err != nil {
		return nil, err
	}
	// Populate team IDs for all users in one query
	return r.populateTeamIDs(qCtx, users)
}

func (r *userRepository) ListByTeam(ctx context.Context, teamID domain.TeamID) ([]domain.User, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT u.id, u.email, u.display_name, pgp_sym_decrypt(u.password_hash, $2)::text, u.sso_provider, pgp_sym_decrypt(u.sso_subject, $2)::text, u.role, u.created_at, u.updated_at, u.blocked_at
		FROM users u JOIN team_members tm ON tm.user_id = u.id
		WHERE tm.team_id = $1 ORDER BY u.created_at ASC`, uuid.UUID(teamID), r.encKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *userRepository) AddToTeam(ctx context.Context, userID domain.UserID, teamID domain.TeamID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(qCtx, `
		INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		uuid.UUID(teamID), uuid.UUID(userID))
	return err
}

func (r *userRepository) RemoveFromTeam(ctx context.Context, userID domain.UserID, teamID domain.TeamID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(qCtx, `
		DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`,
		uuid.UUID(teamID), uuid.UUID(userID))
	return err
}

func (r *userRepository) ListTeamIDs(ctx context.Context, userID domain.UserID) ([]domain.TeamID, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(qCtx, `SELECT team_id FROM team_members WHERE user_id = $1`, uuid.UUID(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []domain.TeamID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, domain.TeamID(id))
	}
	return ids, rows.Err()
}

func (r *userRepository) populateTeamIDs(ctx context.Context, users []domain.User) ([]domain.User, error) {
	if len(users) == 0 {
		return users, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT user_id, team_id FROM team_members`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[uuid.UUID][]domain.TeamID)
	for rows.Next() {
		var uid, tid uuid.UUID
		if err := rows.Scan(&uid, &tid); err != nil {
			return nil, err
		}
		m[uid] = append(m[uid], domain.TeamID(tid))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range users {
		users[i].TeamIDs = m[uuid.UUID(users[i].ID)]
	}
	return users, nil
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
		createdAt    time.Time
		updatedAt    time.Time
		blockedAt    *time.Time
	)

	err := row.Scan(&id, &email, &displayName, &passwordHash, &ssoProvider, &ssoSubject, &role, &createdAt, &updatedAt, &blockedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}

	return domain.User{
		ID:           domain.UserID(id),
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
		SSOProvider:  ssoProvider,
		SSOSubject:   ssoSubject,
		Role:         domain.MemberRole(role),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		BlockedAt:    blockedAt,
	}, nil
}

func scanUserWithoutHash(row pgx.Row) (domain.User, error) {
	var (
		id          uuid.UUID
		email       string
		displayName string
		ssoProvider string
		role        string
		createdAt   time.Time
		updatedAt   time.Time
		blockedAt   *time.Time
	)

	err := row.Scan(&id, &email, &displayName, &ssoProvider, &role, &createdAt, &updatedAt, &blockedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}

	return domain.User{
		ID:          domain.UserID(id),
		Email:       email,
		DisplayName: displayName,
		SSOProvider: ssoProvider,
		Role:        domain.MemberRole(role),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		BlockedAt:   blockedAt,
	}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
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
