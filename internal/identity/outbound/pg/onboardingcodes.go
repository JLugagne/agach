package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgOnboardingCodeRepository struct{ *baseRepository }

func newOnboardingCodeRepository(pool *pgxpool.Pool) *pgOnboardingCodeRepository {
	return &pgOnboardingCodeRepository{&baseRepository{pool: pool}}
}

func (r *pgOnboardingCodeRepository) Create(ctx context.Context, code domain.OnboardingCode) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(qCtx, `
		INSERT INTO onboarding_codes (id, code, created_by_user_id, node_mode, node_name, expires_at, used_at, used_by_node_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.UUID(code.ID),
		code.Code,
		uuid.UUID(code.CreatedByUserID),
		string(code.NodeMode),
		code.NodeName,
		code.ExpiresAt,
		code.UsedAt,
		nodeIDPtr(code.UsedByNodeID),
		code.CreatedAt,
	)
	return err
}

func (r *pgOnboardingCodeRepository) FindByCode(ctx context.Context, code string) (domain.OnboardingCode, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	row := r.pool.QueryRow(qCtx, `
		SELECT id, code, created_by_user_id, node_mode, node_name, expires_at, used_at, used_by_node_id, created_at
		FROM onboarding_codes WHERE code = $1 AND used_at IS NULL`,
		code,
	)
	return scanOnboardingCode(row)
}

func (r *pgOnboardingCodeRepository) MarkUsed(ctx context.Context, codeID domain.OnboardingCodeID, nodeID domain.NodeID) error {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	tx, err := r.pool.Begin(qCtx)
	if err != nil {
		return fmt.Errorf("mark used: begin tx: %w", err)
	}
	defer tx.Rollback(qCtx) //nolint:errcheck

	row := tx.QueryRow(qCtx, `
		SELECT id, used_at, expires_at FROM onboarding_codes WHERE id = $1 FOR UPDATE`,
		uuid.UUID(codeID),
	)
	var (
		id        uuid.UUID
		usedAt    *time.Time
		expiresAt time.Time
	)
	if err := row.Scan(&id, &usedAt, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrOnboardingCodeNotFound
		}
		return err
	}
	if usedAt != nil {
		return domain.ErrOnboardingCodeUsed
	}
	if time.Now().After(expiresAt) {
		return domain.ErrOnboardingCodeExpired
	}

	now := time.Now()
	nid := uuid.UUID(nodeID)
	_, err = tx.Exec(qCtx, `
		UPDATE onboarding_codes SET used_at = $2, used_by_node_id = $3 WHERE id = $1`,
		uuid.UUID(codeID),
		now,
		nid,
	)
	if err != nil {
		return err
	}
	return tx.Commit(qCtx)
}

func (r *pgOnboardingCodeRepository) DeleteExpired(ctx context.Context) (int64, error) {
	qCtx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(qCtx, `
		DELETE FROM onboarding_codes WHERE expires_at < NOW() AND used_at IS NULL`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func scanOnboardingCode(row rowScanner) (domain.OnboardingCode, error) {
	var (
		id              uuid.UUID
		code            string
		createdByUserID uuid.UUID
		nodeMode        string
		nodeName        string
		expiresAt       time.Time
		usedAt          *time.Time
		usedByNodeID    *uuid.UUID
		createdAt       time.Time
	)
	err := row.Scan(&id, &code, &createdByUserID, &nodeMode, &nodeName, &expiresAt, &usedAt, &usedByNodeID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.OnboardingCode{}, domain.ErrOnboardingCodeNotFound
		}
		return domain.OnboardingCode{}, err
	}

	oc := domain.OnboardingCode{
		ID:              domain.OnboardingCodeID(id),
		Code:            code,
		CreatedByUserID: domain.UserID(createdByUserID),
		NodeMode:        domain.NodeMode(nodeMode),
		NodeName:        nodeName,
		ExpiresAt:       expiresAt,
		UsedAt:          usedAt,
		CreatedAt:       createdAt,
	}
	if usedByNodeID != nil {
		nid := domain.NodeID(*usedByNodeID)
		oc.UsedByNodeID = &nid
	}
	return oc, nil
}

func nodeIDPtr(id *domain.NodeID) *uuid.UUID {
	if id == nil {
		return nil
	}
	uid := uuid.UUID(*id)
	return &uid
}
