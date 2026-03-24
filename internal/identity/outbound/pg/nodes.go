package pg

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgNodeRepository struct{ *baseRepository }

func newNodeRepository(pool *pgxpool.Pool) *pgNodeRepository {
	return &pgNodeRepository{&baseRepository{pool: pool}}
}

func (r *pgNodeRepository) Create(ctx context.Context, node domain.Node) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO nodes (id, owner_user_id, name, mode, status, refresh_token_hash, last_seen_at, revoked_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		uuid.UUID(node.ID),
		uuid.UUID(node.OwnerUserID),
		node.Name,
		string(node.Mode),
		string(node.Status),
		[]byte(node.RefreshTokenHash),
		node.LastSeenAt,
		node.RevokedAt,
		node.CreatedAt,
		node.UpdatedAt,
	)
	return err
}

func (r *pgNodeRepository) FindByID(ctx context.Context, id domain.NodeID) (domain.Node, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, owner_user_id, name, mode, status, refresh_token_hash, last_seen_at, revoked_at, created_at, updated_at
		FROM nodes WHERE id = $1`,
		uuid.UUID(id),
	)
	return scanNode(row)
}

func (r *pgNodeRepository) ListByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, owner_user_id, name, mode, status, refresh_token_hash, last_seen_at, revoked_at, created_at, updated_at
		FROM nodes WHERE owner_user_id = $1 ORDER BY created_at ASC`,
		uuid.UUID(ownerID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *pgNodeRepository) ListActiveByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, owner_user_id, name, mode, status, refresh_token_hash, last_seen_at, revoked_at, created_at, updated_at
		FROM nodes WHERE owner_user_id = $1 AND status = 'active' ORDER BY created_at ASC`,
		uuid.UUID(ownerID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

func (r *pgNodeRepository) Update(ctx context.Context, node domain.Node) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		UPDATE nodes SET name=$2, mode=$3, status=$4, refresh_token_hash=$5, last_seen_at=$6, revoked_at=$7, updated_at=$8
		WHERE id=$1`,
		uuid.UUID(node.ID),
		node.Name,
		string(node.Mode),
		string(node.Status),
		[]byte(node.RefreshTokenHash),
		node.LastSeenAt,
		node.RevokedAt,
		node.UpdatedAt,
	)
	return err
}

func (r *pgNodeRepository) UpdateLastSeen(ctx context.Context, id domain.NodeID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		UPDATE nodes SET last_seen_at = NOW(), updated_at = NOW() WHERE id = $1`,
		uuid.UUID(id),
	)
	return err
}

func scanNodes(rows pgx.Rows) ([]domain.Node, error) {
	var out []domain.Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNode(row rowScanner) (domain.Node, error) {
	var (
		id               uuid.UUID
		ownerUserID      uuid.UUID
		name             string
		mode             string
		status           string
		refreshTokenHash []byte
		lastSeenAt       *time.Time
		revokedAt        *time.Time
		createdAt        time.Time
		updatedAt        time.Time
	)
	err := row.Scan(&id, &ownerUserID, &name, &mode, &status, &refreshTokenHash, &lastSeenAt, &revokedAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Node{}, domain.ErrNodeNotFound
		}
		return domain.Node{}, err
	}
	return domain.Node{
		ID:               domain.NodeID(id),
		OwnerUserID:      domain.UserID(ownerUserID),
		Name:             name,
		Mode:             domain.NodeMode(mode),
		Status:           domain.NodeStatus(status),
		RefreshTokenHash: string(refreshTokenHash),
		LastSeenAt:       lastSeenAt,
		RevokedAt:        revokedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}, nil
}
