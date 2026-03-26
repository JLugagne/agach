package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/chats"
	"github.com/jackc/pgx/v5"
)

var _ chats.ChatSessionRepository = (*chatSessionRepository)(nil)

type chatSessionRepository struct {
	*baseRepository
}

func (r *chatSessionRepository) Create(ctx context.Context, session domain.ChatSession) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO chat_sessions (
			id, feature_id, project_id, node_id, state, claude_session_id, jsonl_path,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
			model, created_at, ended_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		string(session.ID),
		string(session.FeatureID),
		string(session.ProjectID),
		nullableString(session.NodeID),
		string(session.State),
		nullableString(session.ClaudeSessionID),
		nullableString(session.JSONLPath),
		session.InputTokens,
		session.OutputTokens,
		session.CacheReadTokens,
		session.CacheWriteTokens,
		nullableString(session.Model),
		session.CreatedAt,
		session.EndedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create chat session: %w", err)
	}
	return nil
}

func (r *chatSessionRepository) FindByID(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, feature_id, project_id, node_id, state, claude_session_id, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
		       model, created_at, ended_at, updated_at
		FROM chat_sessions WHERE id = $1`, string(id))
	session, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrChatSessionNotFound
		}
		return nil, fmt.Errorf("find chat session by id: %w", err)
	}
	return session, nil
}

func (r *chatSessionRepository) FindByFeature(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, feature_id, project_id, node_id, state, claude_session_id, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_write_tokens,
		       model, created_at, ended_at, updated_at
		FROM chat_sessions WHERE feature_id = $1 ORDER BY created_at DESC`, string(featureID))
	if err != nil {
		return nil, fmt.Errorf("find chat sessions by feature: %w", err)
	}
	defer rows.Close()

	var result []domain.ChatSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan chat session row: %w", err)
		}
		result = append(result, *session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find chat sessions by feature rows: %w", err)
	}
	return result, nil
}

func (r *chatSessionRepository) Update(ctx context.Context, session domain.ChatSession) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_sessions SET
			feature_id = $2, project_id = $3, node_id = $4, state = $5, claude_session_id = $6,
			jsonl_path = $7, input_tokens = $8, output_tokens = $9,
			cache_read_tokens = $10, cache_write_tokens = $11, model = $12,
			ended_at = $13, updated_at = NOW()
		WHERE id = $1`,
		string(session.ID),
		string(session.FeatureID),
		string(session.ProjectID),
		nullableString(session.NodeID),
		string(session.State),
		nullableString(session.ClaudeSessionID),
		nullableString(session.JSONLPath),
		session.InputTokens,
		session.OutputTokens,
		session.CacheReadTokens,
		session.CacheWriteTokens,
		nullableString(session.Model),
		session.EndedAt,
	)
	if err != nil {
		return fmt.Errorf("update chat session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrChatSessionNotFound
	}
	return nil
}

func (r *chatSessionRepository) UpdateState(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState, endedAt *time.Time) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_sessions SET state = $2, ended_at = $3, updated_at = NOW()
		WHERE id = $1`,
		string(id), string(state), endedAt,
	)
	if err != nil {
		return fmt.Errorf("update chat session state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrChatSessionNotFound
	}
	return nil
}

func (r *chatSessionRepository) UpdateJSONLPath(ctx context.Context, id domain.ChatSessionID, path string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_sessions SET jsonl_path = $2, updated_at = NOW()
		WHERE id = $1`,
		string(id), path,
	)
	if err != nil {
		return fmt.Errorf("update chat session jsonl path: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrChatSessionNotFound
	}
	return nil
}

func (r *chatSessionRepository) UpdateTokenUsage(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_sessions SET
			input_tokens = $2, output_tokens = $3,
			cache_read_tokens = $4, cache_write_tokens = $5,
			model = $6, updated_at = NOW()
		WHERE id = $1`,
		string(id),
		usage.InputTokens,
		usage.OutputTokens,
		usage.CacheReadTokens,
		usage.CacheWriteTokens,
		nullableString(usage.Model),
	)
	if err != nil {
		return fmt.Errorf("update chat session token usage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrChatSessionNotFound
	}
	return nil
}

func scanSession(s scanner) (*domain.ChatSession, error) {
	var sess domain.ChatSession
	var nodeID *string
	var claudeSessionID *string
	var jsonlPath *string
	var model *string
	err := s.Scan(
		(*string)(&sess.ID),
		(*string)(&sess.FeatureID),
		(*string)(&sess.ProjectID),
		&nodeID,
		(*string)(&sess.State),
		&claudeSessionID,
		&jsonlPath,
		&sess.InputTokens,
		&sess.OutputTokens,
		&sess.CacheReadTokens,
		&sess.CacheWriteTokens,
		&model,
		&sess.CreatedAt,
		&sess.EndedAt,
		&sess.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if nodeID != nil {
		sess.NodeID = *nodeID
	}
	if claudeSessionID != nil {
		sess.ClaudeSessionID = *claudeSessionID
	}
	if jsonlPath != nil {
		sess.JSONLPath = *jsonlPath
	}
	if model != nil {
		sess.Model = *model
	}
	return &sess, nil
}

// nullableString returns nil if s is empty, otherwise returns a pointer to s.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
