package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type agentRepository struct{ *baseRepository }

func (r *agentRepository) Create(ctx context.Context, agent domain.Agent) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	techStackJSON := jsonMarshal(agent.TechStack)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, model, thinking, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		string(agent.ID), agent.Slug, agent.Name, agent.Icon, agent.Color,
		agent.Description, techStackJSON, agent.PromptHint, agent.PromptTemplate, agent.Content, agent.Model, agent.Thinking, agent.SortOrder, agent.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAgentAlreadyExists
		}
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (r *agentRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, model, thinking, sort_order, created_at
		FROM roles WHERE id = $1`, string(id))
	return scanAgent(row)
}

func (r *agentRepository) FindBySlug(ctx context.Context, slug string) (*domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, model, thinking, sort_order, created_at
		FROM roles WHERE slug = $1`, slug)
	return scanAgent(row)
}

func (r *agentRepository) List(ctx context.Context) ([]domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, model, thinking, sort_order, created_at
		FROM roles ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r *agentRepository) Update(ctx context.Context, agent domain.Agent) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	techStackJSON := jsonMarshal(agent.TechStack)
	tag, err := r.pool.Exec(ctx, `
		UPDATE roles SET slug=$1, name=$2, icon=$3, color=$4, description=$5, tech_stack=$6, prompt_hint=$7, prompt_template=$8, content=$9, model=$10, thinking=$11, sort_order=$12
		WHERE id=$13`,
		agent.Slug, agent.Name, agent.Icon, agent.Color, agent.Description,
		techStackJSON, agent.PromptHint, agent.PromptTemplate, agent.Content, agent.Model, agent.Thinking, agent.SortOrder, string(agent.ID),
	)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAgentNotFound
	}
	return nil
}

func (r *agentRepository) Delete(ctx context.Context, id domain.AgentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAgentNotFound
	}
	return nil
}

func (r *agentRepository) IsInUse(ctx context.Context, slug string) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks WHERE assigned_role = $1 OR created_by_role = $1`, slug).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("is in use: %w", err)
	}
	return count > 0, nil
}

func (r *agentRepository) CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("copy global agents: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `SELECT id FROM roles`)
	if err != nil {
		return fmt.Errorf("list global agents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return err
		}
		prID := newID()
		_, err = tx.Exec(ctx, `
			INSERT INTO project_roles (id, project_id, role_id, sort_order)
			VALUES ($1, $2, $3, 0) ON CONFLICT (project_id, role_id) DO NOTHING`,
			prID, string(projectID), agentID,
		)
		if err != nil {
			return fmt.Errorf("copy agent to project: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *agentRepository) CreateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create agent in project: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	techStackJSON := jsonMarshal(agent.TechStack)
	_, err = tx.Exec(ctx, `
		INSERT INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, model, thinking, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (slug) DO NOTHING`,
		string(agent.ID), agent.Slug, agent.Name, agent.Icon, agent.Color,
		agent.Description, techStackJSON, agent.PromptHint, agent.PromptTemplate, agent.Content, agent.Model, agent.Thinking, agent.SortOrder, agent.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create agent in project: %w", err)
	}
	prID := newID()
	_, err = tx.Exec(ctx, `
		INSERT INTO project_roles (id, project_id, role_id, sort_order)
		VALUES ($1, $2, $3, $4) ON CONFLICT (project_id, role_id) DO NOTHING`,
		prID, string(projectID), string(agent.ID), agent.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("link agent to project: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *agentRepository) FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1 AND ro.slug = $2`, string(projectID), slug)
	agent, err := scanAgent(row)
	if errors.Is(err, domain.ErrAgentNotFound) {
		return nil, domain.ErrAgentNotFound
	}
	return agent, err
}

func (r *agentRepository) FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) (*domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1 AND ro.id = $2`, string(projectID), string(id))
	agent, err := scanAgent(row)
	if errors.Is(err, domain.ErrAgentNotFound) {
		return nil, domain.ErrAgentNotFound
	}
	return agent, err
}

func (r *agentRepository) ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1
		ORDER BY pr.sort_order ASC, ro.sort_order ASC`, string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list agents in project: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r *agentRepository) UpdateInProject(ctx context.Context, projectID domain.ProjectID, agent domain.Agent) error {
	return r.Update(ctx, agent)
}

func (r *agentRepository) DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.AgentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM project_roles WHERE project_id=$1 AND role_id=$2`, string(projectID), string(id))
	if err != nil {
		return fmt.Errorf("delete agent in project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAgentNotFound
	}
	return nil
}

func (r *agentRepository) Clone(ctx context.Context, sourceID domain.AgentID, newSlug, newName string) (domain.Agent, error) {
	source, err := r.FindByID(ctx, sourceID)
	if err != nil {
		return domain.Agent{}, err
	}
	existing, _ := r.FindBySlug(ctx, newSlug)
	if existing != nil {
		return domain.Agent{}, domain.ErrAgentAlreadyExists
	}
	cloned := *source
	cloned.ID = domain.NewAgentID()
	cloned.Slug = newSlug
	if newName != "" {
		cloned.Name = newName
	}
	if err := r.Create(ctx, cloned); err != nil {
		return domain.Agent{}, err
	}
	return cloned, nil
}

func (r *agentRepository) AssignToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	prID := newID()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_agents (id, project_id, role_id, sort_order)
		VALUES ($1, $2, $3, 0)`,
		prID, string(projectID), string(agentID),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAgentAlreadyInProject
		}
		return fmt.Errorf("assign agent to project: %w", err)
	}
	return nil
}

func (r *agentRepository) RemoveFromProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `DELETE FROM project_agents WHERE project_id=$1 AND role_id=$2`, string(projectID), string(agentID))
	if err != nil {
		return fmt.Errorf("remove agent from project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAgentNotInProject
	}
	return nil
}

func (r *agentRepository) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_agents pa ON pa.role_id = ro.id
		WHERE pa.project_id = $1
		ORDER BY pa.sort_order ASC, ro.name ASC`, string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list agents by project: %w", err)
	}
	defer rows.Close()
	return scanAgents(rows)
}

func (r *agentRepository) IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM project_agents WHERE project_id=$1 AND role_id=$2)`,
		string(projectID), string(agentID),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is assigned to project: %w", err)
	}
	return exists, nil
}

func scanAgentInto(s scanner) (domain.Agent, error) {
	var agent domain.Agent
	var techStackJSON []byte
	err := s.Scan(
		(*string)(&agent.ID), &agent.Slug, &agent.Name, &agent.Icon, &agent.Color,
		&agent.Description, &techStackJSON, &agent.PromptHint, &agent.PromptTemplate, &agent.Content, &agent.Model, &agent.Thinking, &agent.SortOrder, &agent.CreatedAt,
	)
	if err != nil {
		return domain.Agent{}, err
	}
	agent.TechStack = jsonUnmarshalStrings(techStackJSON)
	return agent, nil
}

func scanAgent(row pgx.Row) (*domain.Agent, error) {
	agent, err := scanAgentInto(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAgentNotFound
		}
		return nil, err
	}
	return &agent, nil
}

func scanAgents(rows pgx.Rows) ([]domain.Agent, error) {
	var result []domain.Agent
	for rows.Next() {
		agent, err := scanAgentInto(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, agent)
	}
	return result, rows.Err()
}
