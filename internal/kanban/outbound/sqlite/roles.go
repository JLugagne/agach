package sqlite

import (
	"context"
	"database/sql"
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
	return false, nil
}

// CopyGlobalRolesToProject reads all global roles and inserts them into the project DB
func (r *RoleRepository) CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error {
	roles, err := r.List(ctx)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if err := r.CreateInProject(ctx, projectID, role); err != nil {
			return err
		}
	}

	return nil
}

func (r *RoleRepository) CreateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	techStackJSON, err := json.Marshal(role.TechStack)
	if err != nil {
		return err
	}

	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			INSERT OR REPLACE INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := db.ExecContext(ctx, query,
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
		return err
	})
}

func (r *RoleRepository) FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error) {
	var role *domain.Role

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
			FROM roles WHERE slug = ?
		`
		var row domain.Role
		var techStackJSON string
		var createdAt time.Time

		err := db.QueryRowContext(ctx, query, slug).Scan(
			&row.ID, &row.Slug, &row.Name, &row.Icon, &row.Color, &row.Description,
			&techStackJSON, &row.PromptHint, &row.SortOrder, &createdAt,
		)
		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrRoleNotFound, err)
			}
			return err
		}

		row.CreatedAt = createdAt
		if err := json.Unmarshal([]byte(techStackJSON), &row.TechStack); err != nil {
			return err
		}
		role = &row
		return nil
	})

	return role, err
}

func (r *RoleRepository) FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) (*domain.Role, error) {
	var role *domain.Role

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
			FROM roles WHERE id = ?
		`
		var row domain.Role
		var techStackJSON string
		var createdAt time.Time

		err := db.QueryRowContext(ctx, query, string(id)).Scan(
			&row.ID, &row.Slug, &row.Name, &row.Icon, &row.Color, &row.Description,
			&techStackJSON, &row.PromptHint, &row.SortOrder, &createdAt,
		)
		if err != nil {
			if isNotFound(err) {
				return errors.Join(domain.ErrRoleNotFound, err)
			}
			return err
		}

		row.CreatedAt = createdAt
		if err := json.Unmarshal([]byte(techStackJSON), &row.TechStack); err != nil {
			return err
		}
		role = &row
		return nil
	})

	return role, err
}

func (r *RoleRepository) ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	var roles []domain.Role

	err := r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, sort_order, created_at
			FROM roles ORDER BY sort_order ASC, name ASC
		`
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.Role
			var techStackJSON string
			var createdAt time.Time

			if err := rows.Scan(
				&row.ID, &row.Slug, &row.Name, &row.Icon, &row.Color, &row.Description,
				&techStackJSON, &row.PromptHint, &row.SortOrder, &createdAt,
			); err != nil {
				return err
			}

			row.CreatedAt = createdAt
			if err := json.Unmarshal([]byte(techStackJSON), &row.TechStack); err != nil {
				return err
			}
			roles = append(roles, row)
		}

		return rows.Err()
	})

	return roles, err
}

func (r *RoleRepository) UpdateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	techStackJSON, err := json.Marshal(role.TechStack)
	if err != nil {
		return err
	}

	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		query := `
			UPDATE roles
			SET name = ?, icon = ?, color = ?, description = ?, tech_stack = ?, prompt_hint = ?, sort_order = ?
			WHERE id = ?
		`
		result, err := db.ExecContext(ctx, query,
			role.Name, role.Icon, role.Color, role.Description,
			string(techStackJSON), role.PromptHint, role.SortOrder, string(role.ID),
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
	})
}

func (r *RoleRepository) DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) error {
	return r.withProjectDB(ctx, projectID, func(db *sql.DB) error {
		result, err := db.ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, string(id))
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
	})
}
