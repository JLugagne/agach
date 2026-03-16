package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Create creates a new role in the global database
func (r *RoleRepository) Create(ctx context.Context, role domain.Role) error {
	techStackJSON, err := json.Marshal(role.TechStack)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.globalDB.ExecContext(ctx, query,
		string(role.ID),
		role.Slug,
		role.Name,
		role.Icon,
		role.Color,
		role.Description,
		string(techStackJSON),
		role.PromptHint,
		role.SortOrder,
		role.CreatedAt,
	)

	if err != nil {
		if isSQLiteConstraintError(err, "UNIQUE") {
			return errors.Join(domain.ErrRoleAlreadyExists, err)
		}
		return err
	}

	return nil
}

// FindByID retrieves a role by ID from the global database
func (r *RoleRepository) FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
	query := `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
		FROM roles
		WHERE id = ?
	`

	var role domain.Role
	var techStackJSON string
	var createdAt time.Time

	err := r.globalDB.QueryRowContext(ctx, query, string(id)).Scan(
		&role.ID,
		&role.Slug,
		&role.Name,
		&role.Icon,
		&role.Color,
		&role.Description,
		&techStackJSON,
		&role.PromptHint,
		&role.SortOrder,
		&createdAt,
	)

	if err != nil {
		if isNotFound(err) {
			return nil, errors.Join(domain.ErrRoleNotFound, err)
		}
		return nil, err
	}

	role.CreatedAt = createdAt

	if err := json.Unmarshal([]byte(techStackJSON), &role.TechStack); err != nil {
		return nil, err
	}

	return &role, nil
}

// FindBySlug retrieves a role by slug from the global database
func (r *RoleRepository) FindBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	query := `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
		FROM roles
		WHERE slug = ?
	`

	var role domain.Role
	var techStackJSON string
	var createdAt time.Time

	err := r.globalDB.QueryRowContext(ctx, query, slug).Scan(
		&role.ID,
		&role.Slug,
		&role.Name,
		&role.Icon,
		&role.Color,
		&role.Description,
		&techStackJSON,
		&role.PromptHint,
		&role.SortOrder,
		&createdAt,
	)

	if err != nil {
		if isNotFound(err) {
			return nil, errors.Join(domain.ErrRoleNotFound, err)
		}
		return nil, err
	}

	role.CreatedAt = createdAt

	if err := json.Unmarshal([]byte(techStackJSON), &role.TechStack); err != nil {
		return nil, err
	}

	return &role, nil
}

// List retrieves all roles from the global database, ordered by sort_order
func (r *RoleRepository) List(ctx context.Context) ([]domain.Role, error) {
	query := `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
		FROM roles
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := r.globalDB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role

	for rows.Next() {
		var role domain.Role
		var techStackJSON string
		var createdAt time.Time

		err := rows.Scan(
			&role.ID,
			&role.Slug,
			&role.Name,
			&role.Icon,
			&role.Color,
			&role.Description,
			&techStackJSON,
			&role.PromptHint,
			&role.SortOrder,
			&createdAt,
		)

		if err != nil {
			return nil, err
		}

		role.CreatedAt = createdAt

		if err := json.Unmarshal([]byte(techStackJSON), &role.TechStack); err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

// Update updates an existing role in the global database
func (r *RoleRepository) Update(ctx context.Context, role domain.Role) error {
	techStackJSON, err := json.Marshal(role.TechStack)
	if err != nil {
		return err
	}

	query := `
		UPDATE roles
		SET name = ?, icon = ?, color = ?, description = ?, tech_stack = ?, prompt_hint = ?, sort_order = ?
		WHERE id = ?
	`

	result, err := r.globalDB.ExecContext(ctx, query,
		role.Name,
		role.Icon,
		role.Color,
		role.Description,
		string(techStackJSON),
		role.PromptHint,
		role.SortOrder,
		string(role.ID),
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrRoleNotFound
	}

	return nil
}

// Delete deletes a role from the global database
func (r *RoleRepository) Delete(ctx context.Context, id domain.RoleID) error {
	query := `DELETE FROM roles WHERE id = ?`

	result, err := r.globalDB.ExecContext(ctx, query, string(id))
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrRoleNotFound
	}

	return nil
}

// IsInUse checks if a role is currently assigned to any tasks across all projects
func (r *RoleRepository) IsInUse(ctx context.Context, slug string) (bool, error) {
	// Since each project has its own database, we need to check all project databases
	// For now, we'll return false to satisfy the interface
	// This will be properly implemented when we add project tracking
	// TODO: Implement proper cross-project role usage checking
	return false, nil
}
