package pg

import (
	"context"
	"errors"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/jackc/pgx/v5"
)

type skillRepository struct{ *baseRepository }

func (r *skillRepository) Create(ctx context.Context, skill domain.Skill) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO skills (id, slug, name, description, content, icon, color, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		string(skill.ID), skill.Slug, skill.Name, skill.Description,
		skill.Content, skill.Icon, skill.Color, skill.SortOrder, skill.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrSkillAlreadyExists
		}
		return fmt.Errorf("create skill: %w", err)
	}
	return nil
}

func (r *skillRepository) FindByID(ctx context.Context, id domain.SkillID) (*domain.Skill, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, content, icon, color, sort_order, created_at, updated_at
		FROM skills WHERE id = $1`, string(id))
	return scanSkill(row)
}

func (r *skillRepository) FindBySlug(ctx context.Context, slug string) (*domain.Skill, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, content, icon, color, sort_order, created_at, updated_at
		FROM skills WHERE slug = $1`, slug)
	skill, err := scanSkill(row)
	if err != nil {
		return nil, err
	}
	return skill, nil
}

func (r *skillRepository) List(ctx context.Context) ([]domain.Skill, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, description, content, icon, color, sort_order, created_at, updated_at
		FROM skills ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *skillRepository) Update(ctx context.Context, skill domain.Skill) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		UPDATE skills
		SET slug=$2, name=$3, description=$4, content=$5, icon=$6, color=$7, sort_order=$8, updated_at=NOW()
		WHERE id=$1`,
		string(skill.ID), skill.Slug, skill.Name, skill.Description,
		skill.Content, skill.Icon, skill.Color, skill.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSkillNotFound
	}
	return nil
}

func (r *skillRepository) Delete(ctx context.Context, id domain.SkillID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	inUse, err := r.IsInUse(ctx, id)
	if err != nil {
		return err
	}
	if inUse {
		return domain.ErrSkillInUse
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM skills WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

func (r *skillRepository) IsInUse(ctx context.Context, id domain.SkillID) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM agent_skills WHERE skill_id = $1)`, string(id)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is skill in use: %w", err)
	}
	return exists, nil
}

func (r *skillRepository) ListByAgent(ctx context.Context, agentID domain.AgentID) ([]domain.Skill, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.slug, s.name, s.description, s.content, s.icon, s.color, s.sort_order, s.created_at, s.updated_at
		FROM skills s
		JOIN agent_skills a ON a.skill_id = s.id
		WHERE a.role_id = $1
		ORDER BY a.sort_order ASC, s.name ASC`, string(agentID))
	if err != nil {
		return nil, fmt.Errorf("list skills by agent: %w", err)
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *skillRepository) AssignToAgent(ctx context.Context, agentID domain.AgentID, skillID domain.SkillID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	id := newID()
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO agent_skills (id, role_id, skill_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		id, string(agentID), string(skillID),
	)
	if err != nil {
		return fmt.Errorf("assign skill to agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSkillAlreadyExists
	}
	return nil
}

func (r *skillRepository) RemoveFromAgent(ctx context.Context, agentID domain.AgentID, skillID domain.SkillID) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM agent_skills WHERE role_id = $1 AND skill_id = $2`,
		string(agentID), string(skillID),
	)
	if err != nil {
		return fmt.Errorf("remove skill from agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSkillNotFound
	}
	return nil
}

func scanSkill(row pgx.Row) (*domain.Skill, error) {
	var skill domain.Skill
	err := row.Scan(
		(*string)(&skill.ID), &skill.Slug, &skill.Name, &skill.Description,
		&skill.Content, &skill.Icon, &skill.Color, &skill.SortOrder,
		&skill.CreatedAt, &skill.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan skill: %w", err)
	}
	return &skill, nil
}

func scanSkills(rows pgx.Rows) ([]domain.Skill, error) {
	var result []domain.Skill
	for rows.Next() {
		var skill domain.Skill
		err := rows.Scan(
			(*string)(&skill.ID), &skill.Slug, &skill.Name, &skill.Description,
			&skill.Content, &skill.Icon, &skill.Color, &skill.SortOrder,
			&skill.CreatedAt, &skill.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skill row: %w", err)
		}
		result = append(result, skill)
	}
	if result == nil {
		return []domain.Skill{}, nil
	}
	return result, rows.Err()
}
