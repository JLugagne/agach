package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type specializedAgentRepository struct{ *baseRepository }

func (r *specializedAgentRepository) Create(ctx context.Context, agent domain.SpecializedAgent) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO specialized_agents (id, parent_agent_id, slug, name, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		string(agent.ID), string(agent.ParentAgentID), agent.Slug, agent.Name, agent.SortOrder, agent.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrSpecializedAgentAlreadyExists
		}
		return fmt.Errorf("create specialized agent: %w", err)
	}
	return nil
}

func (r *specializedAgentRepository) FindByID(ctx context.Context, id domain.SpecializedAgentID) (*domain.SpecializedAgent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, parent_agent_id, slug, name, sort_order, created_at, updated_at
		FROM specialized_agents WHERE id = $1`, string(id))
	return scanSpecializedAgent(row)
}

func (r *specializedAgentRepository) FindBySlug(ctx context.Context, slug string) (*domain.SpecializedAgent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, parent_agent_id, slug, name, sort_order, created_at, updated_at
		FROM specialized_agents WHERE slug = $1`, slug)
	return scanSpecializedAgent(row)
}

func (r *specializedAgentRepository) ListByParent(ctx context.Context, parentID domain.AgentID) ([]domain.SpecializedAgent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, parent_agent_id, slug, name, sort_order, created_at, updated_at
		FROM specialized_agents WHERE parent_agent_id = $1
		ORDER BY sort_order ASC, name ASC`, string(parentID))
	if err != nil {
		return nil, fmt.Errorf("list specialized agents by parent: %w", err)
	}
	defer rows.Close()
	return scanSpecializedAgents(rows)
}

func (r *specializedAgentRepository) CountByParent(ctx context.Context, parentID domain.AgentID) (int, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM specialized_agents WHERE parent_agent_id = $1`, string(parentID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count specialized agents by parent: %w", err)
	}
	return count, nil
}

func (r *specializedAgentRepository) Update(ctx context.Context, agent domain.SpecializedAgent) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE specialized_agents
		SET name=$2, sort_order=$3, updated_at=NOW()
		WHERE id=$1`,
		string(agent.ID), agent.Name, agent.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("update specialized agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSpecializedAgentNotFound
	}
	return nil
}

func (r *specializedAgentRepository) Delete(ctx context.Context, id domain.SpecializedAgentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM specialized_agents WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete specialized agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSpecializedAgentNotFound
	}
	return nil
}

func (r *specializedAgentRepository) ListSkills(ctx context.Context, specializedAgentID domain.SpecializedAgentID) ([]domain.Skill, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.slug, s.name, s.description, s.content, s.icon, s.color, s.sort_order, s.created_at, s.updated_at
		FROM skills s
		JOIN specialized_agent_skills sas ON sas.skill_id = s.id
		WHERE sas.specialized_agent_id = $1
		ORDER BY sas.sort_order ASC, s.name ASC`, string(specializedAgentID))
	if err != nil {
		return nil, fmt.Errorf("list specialized agent skills: %w", err)
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *specializedAgentRepository) SetSkills(ctx context.Context, specializedAgentID domain.SpecializedAgentID, skillIDs []domain.SkillID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `DELETE FROM specialized_agent_skills WHERE specialized_agent_id = $1`, string(specializedAgentID))
	if err != nil {
		return fmt.Errorf("delete specialized agent skills: %w", err)
	}
	for i, skillID := range skillIDs {
		id := newID()
		_, err := r.pool.Exec(ctx, `
			INSERT INTO specialized_agent_skills (id, specialized_agent_id, skill_id, sort_order)
			VALUES ($1, $2, $3, $4)`,
			id, string(specializedAgentID), string(skillID), i,
		)
		if err != nil {
			return fmt.Errorf("insert specialized agent skill: %w", err)
		}
	}
	return nil
}

func scanSpecializedAgent(row pgx.Row) (*domain.SpecializedAgent, error) {
	var agent domain.SpecializedAgent
	err := row.Scan(
		(*string)(&agent.ID), (*string)(&agent.ParentAgentID),
		&agent.Slug, &agent.Name, &agent.SortOrder,
		&agent.CreatedAt, &agent.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan specialized agent: %w", err)
	}
	return &agent, nil
}

func scanSpecializedAgents(rows pgx.Rows) ([]domain.SpecializedAgent, error) {
	var result []domain.SpecializedAgent
	for rows.Next() {
		var agent domain.SpecializedAgent
		err := rows.Scan(
			(*string)(&agent.ID), (*string)(&agent.ParentAgentID),
			&agent.Slug, &agent.Name, &agent.SortOrder,
			&agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan specialized agent row: %w", err)
		}
		result = append(result, agent)
	}
	if result == nil {
		return []domain.SpecializedAgent{}, nil
	}
	return result, rows.Err()
}
