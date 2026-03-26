package pg

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgNodeAccessRepository struct{ *baseRepository }

func newNodeAccessRepository(pool *pgxpool.Pool) *pgNodeAccessRepository {
	return &pgNodeAccessRepository{&baseRepository{pool: pool}}
}

func (r *pgNodeAccessRepository) GrantUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO node_access (id, node_id, user_id, team_id, created_at)
		VALUES ($1, $2, $3, NULL, NOW())
		ON CONFLICT DO NOTHING`,
		uuid.New(),
		uuid.UUID(nodeID),
		uuid.UUID(userID),
	)
	return err
}

func (r *pgNodeAccessRepository) GrantTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO node_access (id, node_id, user_id, team_id, created_at)
		VALUES ($1, $2, NULL, $3, NOW())
		ON CONFLICT DO NOTHING`,
		uuid.New(),
		uuid.UUID(nodeID),
		uuid.UUID(teamID),
	)
	return err
}

func (r *pgNodeAccessRepository) RevokeUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		DELETE FROM node_access WHERE node_id = $1 AND user_id = $2`,
		uuid.UUID(nodeID),
		uuid.UUID(userID),
	)
	return err
}

func (r *pgNodeAccessRepository) RevokeTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		DELETE FROM node_access WHERE node_id = $1 AND team_id = $2`,
		uuid.UUID(nodeID),
		uuid.UUID(teamID),
	)
	return err
}

func (r *pgNodeAccessRepository) ListByNode(ctx context.Context, nodeID domain.NodeID) ([]domain.NodeAccess, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(qCtx, `
		SELECT id, node_id, user_id, team_id, created_at
		FROM node_access WHERE node_id = $1 ORDER BY created_at ASC`,
		uuid.UUID(nodeID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.NodeAccess
	for rows.Next() {
		var (
			id        uuid.UUID
			nid       uuid.UUID
			userID    *uuid.UUID
			teamID    *uuid.UUID
			createdAt time.Time
		)
		if err := rows.Scan(&id, &nid, &userID, &teamID, &createdAt); err != nil {
			return nil, err
		}
		na := domain.NodeAccess{
			ID:        id,
			NodeID:    domain.NodeID(nid),
			CreatedAt: createdAt,
		}
		if userID != nil {
			uid := domain.UserID(*userID)
			na.UserID = &uid
		}
		if teamID != nil {
			tid := domain.TeamID(*teamID)
			na.TeamID = &tid
		}
		out = append(out, na)
	}
	return out, rows.Err()
}

func (r *pgNodeAccessRepository) HasAccess(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) (bool, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	var exists bool
	err := r.pool.QueryRow(qCtx, `
		SELECT EXISTS(
			SELECT 1 FROM node_access
			WHERE node_id = $1 AND (
				user_id = $2
				OR team_id IN (SELECT team_id FROM team_members WHERE user_id = $2)
			)
		)`,
		uuid.UUID(nodeID),
		uuid.UUID(userID),
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
