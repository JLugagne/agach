package pg

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dockerfiles"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
	notificationsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/notifications"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations
var migrationsFS embed.FS

// Repositories holds all PostgreSQL repository implementations.
type Repositories struct {
	Projects     projects.ProjectRepository
	Agents       agentsrepo.AgentRepository
	Tasks        tasks.TaskRepository
	Columns      columns.ColumnRepository
	Comments     comments.CommentRepository
	Dependencies dependencies.DependencyRepository
	ToolUsage    toolusage.ToolUsageRepository
	Skills       skills.SkillRepository
	Dockerfiles  dockerfiles.DockerfileRepository
	Features      featuresrepo.FeatureRepository
	Notifications notificationsrepo.NotificationRepository
}

// NewRepositories creates all repository implementations backed by a pgxpool.Pool and runs migrations.
func NewRepositories(pool *pgxpool.Pool) (*Repositories, error) {
	if pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		entries, err := migrationsFS.ReadDir("migrations")
		if err != nil {
			return nil, fmt.Errorf("reading migrations directory: %w", err)
		}
		for _, entry := range entries {
			sql, err := migrationsFS.ReadFile("migrations/" + entry.Name())
			if err != nil {
				return nil, fmt.Errorf("reading migration %s: %w", entry.Name(), err)
			}
			if _, err := pool.Exec(ctx, string(sql)); err != nil {
				return nil, fmt.Errorf("applying migration %s: %w", entry.Name(), err)
			}
		}
	}
	base := &baseRepository{pool: pool}
	return &Repositories{
		Projects:     &projectRepository{base},
		Agents:       &roleRepository{base},
		Tasks:        &taskRepository{base},
		Columns:      &columnRepository{base},
		Comments:     &commentRepository{base},
		Dependencies: &dependencyRepository{base},
		ToolUsage:    &toolUsageRepository{base},
		Skills:       &skillRepository{base},
		Dockerfiles:  &dockerfileRepository{base},
		Features:      &featureRepository{base},
		Notifications: &notificationRepository{base},
	}, nil
}

type baseRepository struct {
	pool *pgxpool.Pool
}

// isUniqueViolation detects PostgreSQL unique constraint violations (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key")
}

// isCheckViolation detects PostgreSQL check constraint violations (SQLSTATE 23514).
func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23514") || strings.Contains(err.Error(), "check constraint")
}

// jsonMarshal marshals a value to JSON bytes, returning "[]" on nil slices.
func jsonMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}

// jsonUnmarshalStrings unmarshals a JSON byte slice into a string slice.
func jsonUnmarshalStrings(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	if result == nil {
		return []string{}
	}
	return result
}

// newID generates a new UUIDv7 string for tool_usage and project_roles records.
func newID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

// ----------------------------
// projectRepository
// ----------------------------

type projectRepository struct{ *baseRepository }

func (r *projectRepository) Create(ctx context.Context, p domain.Project) error {
	var parentID *string
	if p.ParentID != nil {
		s := string(*p.ParentID)
		parentID = &s
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO projects (id, parent_id, name, description, created_by_role, created_by_agent, default_role, git_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		string(p.ID), parentID, p.Name, p.Description,
		p.CreatedByRole, p.CreatedByAgent, p.DefaultRole, p.GitURL,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	// Create 4 default columns for the project
	colDefs := []struct {
		slug     domain.ColumnSlug
		name     string
		position int
	}{
		{domain.ColumnTodo, "To Do", 0},
		{domain.ColumnInProgress, "In Progress", 1},
		{domain.ColumnDone, "Done", 2},
		{domain.ColumnBlocked, "Blocked", 3},
	}
	for _, cd := range colDefs {
		colID := string(domain.NewColumnID())
		_, err := r.pool.Exec(ctx, `
			INSERT INTO columns (id, project_id, slug, name, position, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())`,
			colID, string(p.ID), string(cd.slug), cd.name, cd.position,
		)
		if err != nil {
			return fmt.Errorf("create default column %s: %w", cd.slug, err)
		}
	}

	return nil
}

func (r *projectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
		FROM projects WHERE id = $1`, string(id))
	return scanProject(row)
}

func (r *projectRepository) List(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
	var rows pgx.Rows
	var err error
	if parentID == nil {
		rows, err = r.pool.Query(ctx, `
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
			FROM projects WHERE parent_id IS NULL ORDER BY created_at ASC`)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,''), dockerfile_id, created_at, updated_at
			FROM projects WHERE parent_id = $1 ORDER BY created_at ASC`, string(*parentID))
	}
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (r *projectRepository) GetTree(ctx context.Context, id domain.ProjectID) ([]domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, COALESCE(git_url,'') AS git_url, dockerfile_id, created_at, updated_at
			FROM projects WHERE id = $1
			UNION ALL
			SELECT p.id, p.parent_id, p.name, p.description, p.created_by_role, p.created_by_agent, p.default_role, COALESCE(p.git_url,''), p.dockerfile_id, p.created_at, p.updated_at
			FROM projects p
			INNER JOIN tree t ON p.parent_id = t.id
		)
		SELECT id, parent_id, name, description, created_by_role, created_by_agent, default_role, git_url, dockerfile_id, created_at, updated_at FROM tree`,
		string(id),
	)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}
	defer rows.Close()
	return scanProjects(rows)
}

func (r *projectRepository) Update(ctx context.Context, p domain.Project) error {
	var parentID *string
	if p.ParentID != nil {
		s := string(*p.ParentID)
		parentID = &s
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects SET parent_id=$1, name=$2, description=$3, created_by_role=$4, created_by_agent=$5, default_role=$6, git_url=$7, updated_at=$8
		WHERE id=$9`,
		parentID, p.Name, p.Description, p.CreatedByRole, p.CreatedByAgent, p.DefaultRole, p.GitURL, p.UpdatedAt, string(p.ID),
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (r *projectRepository) Delete(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error) {
	// Collect all IDs in the tree before deleting
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM projects WHERE id = $1
			UNION ALL
			SELECT p.id FROM projects p INNER JOIN tree t ON p.parent_id = t.id
		)
		SELECT id FROM tree`, string(id))
	if err != nil {
		return nil, fmt.Errorf("collect project tree: %w", err)
	}
	var ids []domain.ProjectID
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, domain.ProjectID(pid))
	}
	rows.Close()

	if len(ids) == 0 {
		return nil, domain.ErrProjectNotFound
	}

	_, err = r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, string(id))
	if err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}
	return ids, nil
}

func (r *projectRepository) GetSummary(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.slug, COUNT(t.id) as cnt
		FROM columns c
		LEFT JOIN tasks t ON t.column_id = c.id AND t.project_id = $1
		WHERE c.project_id = $1
		GROUP BY c.slug`, string(id))
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	defer rows.Close()

	summary := &domain.ProjectSummary{}
	for rows.Next() {
		var slug string
		var cnt int
		if err := rows.Scan(&slug, &cnt); err != nil {
			return nil, err
		}
		switch domain.ColumnSlug(slug) {
		case domain.ColumnBacklog:
			summary.BacklogCount = cnt
		case domain.ColumnTodo:
			summary.TodoCount = cnt
		case domain.ColumnInProgress:
			summary.InProgressCount = cnt
		case domain.ColumnDone:
			summary.DoneCount = cnt
		case domain.ColumnBlocked:
			summary.BlockedCount = cnt
		}
	}
	return summary, nil
}

func (r *projectRepository) CountChildren(ctx context.Context, id domain.ProjectID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE parent_id = $1`, string(id)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count children: %w", err)
	}
	return count, nil
}

func (r *projectRepository) ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m, updated_at
		FROM model_pricing
		ORDER BY model_id`)
	if err != nil {
		return nil, fmt.Errorf("list model pricing: %w", err)
	}
	defer rows.Close()

	var result []domain.ModelPricing
	for rows.Next() {
		var p domain.ModelPricing
		if err := rows.Scan(&p.ID, &p.ModelID, &p.InputPricePer1M, &p.OutputPricePer1M, &p.CacheReadPricePer1M, &p.CacheWritePricePer1M, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// scanProject scans a single project row.
func scanProject(row pgx.Row) (*domain.Project, error) {
	var p domain.Project
	var parentID *string
	var dockerfileID *string
	err := row.Scan(
		(*string)(&p.ID), &parentID, &p.Name, &p.Description,
		&p.CreatedByRole, &p.CreatedByAgent, &p.DefaultRole, &p.GitURL,
		&dockerfileID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	if parentID != nil {
		pid := domain.ProjectID(*parentID)
		p.ParentID = &pid
	}
	if dockerfileID != nil {
		did := domain.DockerfileID(*dockerfileID)
		p.DockerfileID = &did
	}
	return &p, nil
}

// scanProjects scans multiple project rows.
func scanProjects(rows pgx.Rows) ([]domain.Project, error) {
	var result []domain.Project
	for rows.Next() {
		var p domain.Project
		var parentID *string
		var dockerfileID *string
		err := rows.Scan(
			(*string)(&p.ID), &parentID, &p.Name, &p.Description,
			&p.CreatedByRole, &p.CreatedByAgent, &p.DefaultRole, &p.GitURL,
			&dockerfileID, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if parentID != nil {
			pid := domain.ProjectID(*parentID)
			p.ParentID = &pid
		}
		if dockerfileID != nil {
			did := domain.DockerfileID(*dockerfileID)
			p.DockerfileID = &did
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ----------------------------
// roleRepository
// ----------------------------

type roleRepository struct{ *baseRepository }

func (r *roleRepository) Create(ctx context.Context, role domain.Role) error {
	techStackJSON := jsonMarshal(role.TechStack)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		string(role.ID), role.Slug, role.Name, role.Icon, role.Color,
		role.Description, techStackJSON, role.PromptHint, role.PromptTemplate, role.Content, role.SortOrder, role.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrRoleAlreadyExists
		}
		return fmt.Errorf("create role: %w", err)
	}
	return nil
}

func (r *roleRepository) FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, sort_order, created_at
		FROM roles WHERE id = $1`, string(id))
	return scanRole(row)
}

func (r *roleRepository) FindBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, sort_order, created_at
		FROM roles WHERE slug = $1`, slug)
	return scanRole(row)
}

func (r *roleRepository) List(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, sort_order, created_at
		FROM roles ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *roleRepository) Update(ctx context.Context, role domain.Role) error {
	techStackJSON := jsonMarshal(role.TechStack)
	tag, err := r.pool.Exec(ctx, `
		UPDATE roles SET slug=$1, name=$2, icon=$3, color=$4, description=$5, tech_stack=$6, prompt_hint=$7, prompt_template=$8, content=$9, sort_order=$10
		WHERE id=$11`,
		role.Slug, role.Name, role.Icon, role.Color, role.Description,
		techStackJSON, role.PromptHint, role.PromptTemplate, role.Content, role.SortOrder, string(role.ID),
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id domain.RoleID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE id = $1`, string(id))
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}

func (r *roleRepository) IsInUse(ctx context.Context, slug string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks WHERE assigned_role = $1 OR created_by_role = $1`, slug).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("is in use: %w", err)
	}
	return count > 0, nil
}

func (r *roleRepository) CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error {
	rows, err := r.pool.Query(ctx, `SELECT id FROM roles`)
	if err != nil {
		return fmt.Errorf("list global roles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return err
		}
		prID := newID()
		_, err = r.pool.Exec(ctx, `
			INSERT INTO project_roles (id, project_id, role_id, sort_order)
			VALUES ($1, $2, $3, 0) ON CONFLICT (project_id, role_id) DO NOTHING`,
			prID, string(projectID), roleID,
		)
		if err != nil {
			return fmt.Errorf("copy role to project: %w", err)
		}
	}
	return rows.Err()
}

func (r *roleRepository) CreateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	techStackJSON := jsonMarshal(role.TechStack)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO roles (id, slug, name, icon, color, description, tech_stack, prompt_hint, prompt_template, content, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (slug) DO NOTHING`,
		string(role.ID), role.Slug, role.Name, role.Icon, role.Color,
		role.Description, techStackJSON, role.PromptHint, role.PromptTemplate, role.Content, role.SortOrder, role.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create role in project: %w", err)
	}
	prID := newID()
	_, err = r.pool.Exec(ctx, `
		INSERT INTO project_roles (id, project_id, role_id, sort_order)
		VALUES ($1, $2, $3, $4) ON CONFLICT (project_id, role_id) DO NOTHING`,
		prID, string(projectID), string(role.ID), role.SortOrder,
	)
	if err != nil {
		return fmt.Errorf("link role to project: %w", err)
	}
	return nil
}

func (r *roleRepository) FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1 AND ro.slug = $2`, string(projectID), slug)
	role, err := scanRole(row)
	if errors.Is(err, domain.ErrRoleNotFound) {
		return nil, domain.ErrRoleNotFound
	}
	return role, err
}

func (r *roleRepository) FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) (*domain.Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1 AND ro.id = $2`, string(projectID), string(id))
	role, err := scanRole(row)
	if errors.Is(err, domain.ErrRoleNotFound) {
		return nil, domain.ErrRoleNotFound
	}
	return role, err
}

func (r *roleRepository) ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_roles pr ON pr.role_id = ro.id
		WHERE pr.project_id = $1
		ORDER BY pr.sort_order ASC, ro.sort_order ASC`, string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list roles in project: %w", err)
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *roleRepository) UpdateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	return r.Update(ctx, role)
}

func (r *roleRepository) DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM project_roles WHERE project_id=$1 AND role_id=$2`, string(projectID), string(id))
	if err != nil {
		return fmt.Errorf("delete role in project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrRoleNotFound
	}
	return nil
}

func (r *roleRepository) Clone(ctx context.Context, sourceID domain.RoleID, newSlug, newName string) (domain.Role, error) {
	source, err := r.FindByID(ctx, sourceID)
	if err != nil {
		return domain.Role{}, err
	}
	existing, _ := r.FindBySlug(ctx, newSlug)
	if existing != nil {
		return domain.Role{}, domain.ErrRoleAlreadyExists
	}
	cloned := *source
	cloned.ID = domain.NewRoleID()
	cloned.Slug = newSlug
	if newName != "" {
		cloned.Name = newName
	}
	if err := r.Create(ctx, cloned); err != nil {
		return domain.Role{}, err
	}
	return cloned, nil
}

func (r *roleRepository) AssignToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	prID := newID()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO project_agents (id, project_id, role_id, sort_order)
		VALUES ($1, $2, $3, 0)`,
		prID, string(projectID), string(roleID),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAgentAlreadyInProject
		}
		return fmt.Errorf("assign role to project: %w", err)
	}
	return nil
}

func (r *roleRepository) RemoveFromProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM project_agents WHERE project_id=$1 AND role_id=$2`, string(projectID), string(roleID))
	if err != nil {
		return fmt.Errorf("remove role from project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAgentNotInProject
	}
	return nil
}

func (r *roleRepository) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ro.id, ro.slug, ro.name, ro.icon, ro.color, ro.description, ro.tech_stack, ro.prompt_hint, ro.prompt_template, ro.content, ro.sort_order, ro.created_at
		FROM roles ro
		JOIN project_agents pa ON pa.role_id = ro.id
		WHERE pa.project_id = $1
		ORDER BY pa.sort_order ASC, ro.name ASC`, string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list roles by project: %w", err)
	}
	defer rows.Close()
	return scanRoles(rows)
}

func (r *roleRepository) IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM project_agents WHERE project_id=$1 AND role_id=$2)`,
		string(projectID), string(roleID),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is assigned to project: %w", err)
	}
	return exists, nil
}

func scanRole(row pgx.Row) (*domain.Role, error) {
	var role domain.Role
	var techStackJSON []byte
	err := row.Scan(
		(*string)(&role.ID), &role.Slug, &role.Name, &role.Icon, &role.Color,
		&role.Description, &techStackJSON, &role.PromptHint, &role.PromptTemplate, &role.Content, &role.SortOrder, &role.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRoleNotFound
		}
		return nil, err
	}
	role.TechStack = jsonUnmarshalStrings(techStackJSON)
	return &role, nil
}

func scanRoles(rows pgx.Rows) ([]domain.Role, error) {
	var result []domain.Role
	for rows.Next() {
		var role domain.Role
		var techStackJSON []byte
		err := rows.Scan(
			(*string)(&role.ID), &role.Slug, &role.Name, &role.Icon, &role.Color,
			&role.Description, &techStackJSON, &role.PromptHint, &role.PromptTemplate, &role.Content, &role.SortOrder, &role.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		role.TechStack = jsonUnmarshalStrings(techStackJSON)
		result = append(result, role)
	}
	return result, rows.Err()
}

// ----------------------------
// taskRepository
// ----------------------------

type taskRepository struct{ *baseRepository }

const taskSelectCols = `
	t.id, t.column_id, t.feature_id, t.title, t.summary, t.description, t.priority, t.priority_score,
	t.position, t.created_by_role, t.created_by_agent, t.assigned_role,
	t.is_blocked, t.blocked_reason, t.blocked_at, t.blocked_by_agent,
	t.wont_do_requested, t.wont_do_reason, t.wont_do_requested_by, t.wont_do_requested_at,
	t.completion_summary, t.completed_by_agent, t.completed_at,
	t.files_modified, t.resolution, t.context_files, t.tags, t.estimated_effort,
	t.seen_at, t.session_id,
	t.input_tokens, t.output_tokens, t.cache_read_tokens, t.cache_write_tokens, t.model,
	t.cold_start_input_tokens, t.cold_start_output_tokens, t.cold_start_cache_read_tokens, t.cold_start_cache_write_tokens,
	t.started_at, t.duration_seconds, t.human_estimate_seconds,
	t.created_at, t.updated_at
`

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	var filesModifiedJSON, contextFilesJSON, tagsJSON []byte
	var isBlocked, wontDoRequested int
	var featureIDStr *string
	err := row.Scan(
		(*string)(&t.ID), (*string)(&t.ColumnID), &featureIDStr, &t.Title, &t.Summary, &t.Description,
		(*string)(&t.Priority), &t.PriorityScore, &t.Position,
		&t.CreatedByRole, &t.CreatedByAgent, &t.AssignedRole,
		&isBlocked, &t.BlockedReason, &t.BlockedAt, &t.BlockedByAgent,
		&wontDoRequested, &t.WontDoReason, &t.WontDoRequestedBy, &t.WontDoRequestedAt,
		&t.CompletionSummary, &t.CompletedByAgent, &t.CompletedAt,
		&filesModifiedJSON, &t.Resolution, &contextFilesJSON, &tagsJSON, &t.EstimatedEffort,
		&t.SeenAt, &t.SessionID,
		&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.CacheWriteTokens, &t.Model,
		&t.ColdStartInputTokens, &t.ColdStartOutputTokens, &t.ColdStartCacheReadTokens, &t.ColdStartCacheWriteTokens,
		&t.StartedAt, &t.DurationSeconds, &t.HumanEstimateSeconds,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, err
	}
	if featureIDStr != nil {
		fid := domain.FeatureID(*featureIDStr)
		t.FeatureID = &fid
	}
	t.IsBlocked = isBlocked == 1
	t.WontDoRequested = wontDoRequested == 1
	t.FilesModified = jsonUnmarshalStrings(filesModifiedJSON)
	t.ContextFiles = jsonUnmarshalStrings(contextFilesJSON)
	t.Tags = jsonUnmarshalStrings(tagsJSON)
	return &t, nil
}

func scanTaskRows(rows pgx.Rows) ([]domain.Task, error) {
	var result []domain.Task
	for rows.Next() {
		var t domain.Task
		var filesModifiedJSON, contextFilesJSON, tagsJSON []byte
		var isBlocked, wontDoRequested int
		var featureIDStr *string
		err := rows.Scan(
			(*string)(&t.ID), (*string)(&t.ColumnID), &featureIDStr, &t.Title, &t.Summary, &t.Description,
			(*string)(&t.Priority), &t.PriorityScore, &t.Position,
			&t.CreatedByRole, &t.CreatedByAgent, &t.AssignedRole,
			&isBlocked, &t.BlockedReason, &t.BlockedAt, &t.BlockedByAgent,
			&wontDoRequested, &t.WontDoReason, &t.WontDoRequestedBy, &t.WontDoRequestedAt,
			&t.CompletionSummary, &t.CompletedByAgent, &t.CompletedAt,
			&filesModifiedJSON, &t.Resolution, &contextFilesJSON, &tagsJSON, &t.EstimatedEffort,
			&t.SeenAt, &t.SessionID,
			&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.CacheWriteTokens, &t.Model,
			&t.ColdStartInputTokens, &t.ColdStartOutputTokens, &t.ColdStartCacheReadTokens, &t.ColdStartCacheWriteTokens,
			&t.StartedAt, &t.DurationSeconds, &t.HumanEstimateSeconds,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if featureIDStr != nil {
			fid := domain.FeatureID(*featureIDStr)
			t.FeatureID = &fid
		}
		t.IsBlocked = isBlocked == 1
		t.WontDoRequested = wontDoRequested == 1
		t.FilesModified = jsonUnmarshalStrings(filesModifiedJSON)
		t.ContextFiles = jsonUnmarshalStrings(contextFilesJSON)
		t.Tags = jsonUnmarshalStrings(tagsJSON)
		result = append(result, t)
	}
	return result, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// featureIDOrNil returns a string representation of a FeatureID pointer, or nil if the pointer is nil.
func featureIDOrNil(id *domain.FeatureID) interface{} {
	if id == nil {
		return nil
	}
	return string(*id)
}

func (r *taskRepository) Create(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	filesJSON := jsonMarshal(task.FilesModified)
	contextJSON := jsonMarshal(task.ContextFiles)
	tagsJSON := jsonMarshal(task.Tags)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO tasks (
			id, project_id, column_id, feature_id, title, summary, description,
			priority, priority_score, position,
			created_by_role, created_by_agent, assigned_role,
			is_blocked, blocked_reason, blocked_at, blocked_by_agent,
			wont_do_requested, wont_do_reason, wont_do_requested_by, wont_do_requested_at,
			completion_summary, completed_by_agent, completed_at,
			files_modified, resolution, context_files, tags, estimated_effort,
			seen_at, session_id,
			input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model,
			cold_start_input_tokens, cold_start_output_tokens, cold_start_cache_read_tokens, cold_start_cache_write_tokens,
			started_at, duration_seconds, human_estimate_seconds,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,
			$11,$12,$13,
			$14,$15,$16,$17,
			$18,$19,$20,$21,
			$22,$23,$24,
			$25,$26,$27,$28,$29,
			$30,$31,
			$32,$33,$34,$35,$36,
			$37,$38,$39,$40,
			$41,$42,$43,
			$44,$45
		)`,
		string(task.ID), string(projectID), string(task.ColumnID), featureIDOrNil(task.FeatureID),
		task.Title, task.Summary, task.Description,
		string(task.Priority), task.PriorityScore, task.Position,
		task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
		boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
		boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
		task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
		filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
		task.SeenAt, task.SessionID,
		task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
		task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
		task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *taskRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+taskSelectCols+` FROM tasks t WHERE t.project_id=$1 AND t.id=$2`,
		string(projectID), string(id))
	return scanTask(row)
}

func (r *taskRepository) List(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
	args := []any{string(projectID)}
	where := []string{"t.project_id = $1"}
	argIdx := 2

	if filters.ColumnSlug != nil {
		where = append(where, fmt.Sprintf(`c.slug = $%d`, argIdx))
		args = append(args, string(*filters.ColumnSlug))
		argIdx++
	}
	if filters.AssignedRole != nil {
		where = append(where, fmt.Sprintf(`t.assigned_role = $%d`, argIdx))
		args = append(args, *filters.AssignedRole)
		argIdx++
	}
	if filters.Priority != nil {
		where = append(where, fmt.Sprintf(`t.priority = $%d`, argIdx))
		args = append(args, string(*filters.Priority))
		argIdx++
	}
	if filters.IsBlocked != nil {
		where = append(where, fmt.Sprintf(`t.is_blocked = $%d`, argIdx))
		args = append(args, boolToInt(*filters.IsBlocked))
		argIdx++
	}
	if filters.WontDoRequested != nil {
		where = append(where, fmt.Sprintf(`t.wont_do_requested = $%d`, argIdx))
		args = append(args, boolToInt(*filters.WontDoRequested))
		argIdx++
	}
	if filters.UpdatedSince != nil {
		where = append(where, fmt.Sprintf(`t.updated_at >= $%d`, argIdx))
		args = append(args, *filters.UpdatedSince)
		argIdx++
	}
	if filters.FeatureID != nil {
		where = append(where, fmt.Sprintf(`t.feature_id = $%d`, argIdx))
		args = append(args, string(*filters.FeatureID))
		argIdx++
	}
	if filters.Search != "" {
		where = append(where, fmt.Sprintf(`t.search_vector @@ plainto_tsquery('english', $%d)`, argIdx))
		args = append(args, filters.Search)
		argIdx++
	}

	query := `
		SELECT ` + taskSelectCols + `,
			(EXISTS (
				SELECT 1 FROM task_dependencies td
				JOIN tasks dep ON dep.id = td.depends_on_task_id
				JOIN columns dc ON dc.id = dep.column_id
				WHERE td.task_id = t.id AND dc.slug != 'done'
			)) AS has_unresolved_deps,
			(SELECT COUNT(*) FROM comments cm WHERE cm.task_id = t.id) AS comment_count
		FROM tasks t
		JOIN columns c ON c.id = t.column_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY t.priority_score DESC, t.created_at ASC`

	if filters.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, filters.Limit)
		argIdx++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var result []domain.TaskWithDetails
	for rows.Next() {
		var t domain.Task
		var filesModifiedJSON, contextFilesJSON, tagsJSON []byte
		var isBlocked, wontDoRequested int
		var hasUnresolvedDeps bool
		var commentCount int
		var featureIDStr *string

		err := rows.Scan(
			(*string)(&t.ID), (*string)(&t.ColumnID), &featureIDStr, &t.Title, &t.Summary, &t.Description,
			(*string)(&t.Priority), &t.PriorityScore, &t.Position,
			&t.CreatedByRole, &t.CreatedByAgent, &t.AssignedRole,
			&isBlocked, &t.BlockedReason, &t.BlockedAt, &t.BlockedByAgent,
			&wontDoRequested, &t.WontDoReason, &t.WontDoRequestedBy, &t.WontDoRequestedAt,
			&t.CompletionSummary, &t.CompletedByAgent, &t.CompletedAt,
			&filesModifiedJSON, &t.Resolution, &contextFilesJSON, &tagsJSON, &t.EstimatedEffort,
			&t.SeenAt, &t.SessionID,
			&t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.CacheWriteTokens, &t.Model,
			&t.ColdStartInputTokens, &t.ColdStartOutputTokens, &t.ColdStartCacheReadTokens, &t.ColdStartCacheWriteTokens,
			&t.StartedAt, &t.DurationSeconds, &t.HumanEstimateSeconds,
			&t.CreatedAt, &t.UpdatedAt,
			&hasUnresolvedDeps, &commentCount,
		)
		if err != nil {
			return nil, err
		}
		if featureIDStr != nil {
			fid := domain.FeatureID(*featureIDStr)
			t.FeatureID = &fid
		}
		t.IsBlocked = isBlocked == 1
		t.WontDoRequested = wontDoRequested == 1
		t.FilesModified = jsonUnmarshalStrings(filesModifiedJSON)
		t.ContextFiles = jsonUnmarshalStrings(contextFilesJSON)
		t.Tags = jsonUnmarshalStrings(tagsJSON)
		result = append(result, domain.TaskWithDetails{
			Task:              t,
			HasUnresolvedDeps: hasUnresolvedDeps,
			CommentCount:      commentCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *taskRepository) Update(ctx context.Context, projectID domain.ProjectID, task domain.Task) error {
	filesJSON := jsonMarshal(task.FilesModified)
	contextJSON := jsonMarshal(task.ContextFiles)
	tagsJSON := jsonMarshal(task.Tags)

	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET
			column_id=$1, feature_id=$2, title=$3, summary=$4, description=$5,
			priority=$6, priority_score=$7, position=$8,
			created_by_role=$9, created_by_agent=$10, assigned_role=$11,
			is_blocked=$12, blocked_reason=$13, blocked_at=$14, blocked_by_agent=$15,
			wont_do_requested=$16, wont_do_reason=$17, wont_do_requested_by=$18, wont_do_requested_at=$19,
			completion_summary=$20, completed_by_agent=$21, completed_at=$22,
			files_modified=$23, resolution=$24, context_files=$25, tags=$26, estimated_effort=$27,
			seen_at=$28, session_id=$29,
			input_tokens=$30, output_tokens=$31, cache_read_tokens=$32, cache_write_tokens=$33, model=$34,
			cold_start_input_tokens=$35, cold_start_output_tokens=$36, cold_start_cache_read_tokens=$37, cold_start_cache_write_tokens=$38,
			started_at=$39, duration_seconds=$40, human_estimate_seconds=$41,
			updated_at=$42
		WHERE project_id=$43 AND id=$44`,
		string(task.ColumnID), featureIDOrNil(task.FeatureID), task.Title, task.Summary, task.Description,
		string(task.Priority), task.PriorityScore, task.Position,
		task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
		boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
		boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
		task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
		filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
		task.SeenAt, task.SessionID,
		task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
		task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
		task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
		task.UpdatedAt,
		string(projectID), string(task.ID),
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.TaskID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(id))
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) Move(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnID domain.ColumnID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET column_id=$1, updated_at=NOW()
		WHERE project_id=$2 AND id=$3`,
		string(targetColumnID), string(projectID), string(taskID),
	)
	if err != nil {
		return fmt.Errorf("move task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *taskRepository) CountByColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks WHERE project_id=$1 AND column_id=$2`,
		string(projectID), string(columnID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by column: %w", err)
	}
	return count, nil
}

func (r *taskRepository) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string) (*domain.Task, error) {
	query := `
		SELECT ` + taskSelectCols + `
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.project_id = t.project_id
		WHERE t.project_id = $1
		  AND c.slug = 'todo'
		  AND t.is_blocked = 0
		  AND t.wont_do_requested = 0
		  AND NOT EXISTS (
			  SELECT 1 FROM task_dependencies td
			  JOIN tasks dep ON dep.id = td.depends_on_task_id
			  JOIN columns dc ON dc.id = dep.column_id
			  WHERE td.task_id = t.id AND dc.slug != 'done'
		  )`

	args := []any{string(projectID)}
	if role != "" {
		query += ` AND (t.assigned_role = $2 OR t.assigned_role = '')`
		args = append(args, role)
	}
	query += ` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT 1`

	row := r.pool.QueryRow(ctx, query, args...)
	task, err := scanTask(row)
	if errors.Is(err, domain.ErrTaskNotFound) {
		return nil, domain.ErrNoTasksAvailable
	}
	return task, err
}

func (r *taskRepository) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int) ([]domain.Task, error) {
	query := `
		SELECT ` + taskSelectCols + `
		FROM tasks t
		JOIN columns c ON c.id = t.column_id AND c.project_id = t.project_id
		WHERE t.project_id = $1
		  AND c.slug = 'todo'
		  AND t.is_blocked = 0
		  AND t.wont_do_requested = 0
		  AND NOT EXISTS (
			  SELECT 1 FROM task_dependencies td
			  JOIN tasks dep ON dep.id = td.depends_on_task_id
			  JOIN columns dc ON dc.id = dep.column_id
			  WHERE td.task_id = t.id AND dc.slug != 'done'
		  )`

	args := []any{string(projectID)}
	if role != "" {
		query += ` AND (t.assigned_role = $2 OR t.assigned_role = '')`
		args = append(args, role)
		query += fmt.Sprintf(` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT $%d`, 3)
	} else {
		query += fmt.Sprintf(` ORDER BY t.priority_score DESC, t.created_at ASC LIMIT $%d`, 2)
	}
	args = append(args, count)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get next tasks: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *taskRepository) HasUnresolvedDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM task_dependencies td
		JOIN tasks dep ON dep.id = td.depends_on_task_id
		JOIN columns dc ON dc.id = dep.column_id
		WHERE td.task_id = $1 AND dc.slug != 'done'`,
		string(taskID)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has unresolved deps: %w", err)
	}
	return count > 0, nil
}

func (r *taskRepository) GetDependentsNotDone(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+taskSelectCols+`
		FROM tasks t
		JOIN task_dependencies td ON td.task_id = t.id
		JOIN columns c ON c.id = t.column_id
		WHERE td.depends_on_task_id = $1 AND c.slug != 'done'`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("get dependents not done: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func (r *taskRepository) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET seen_at = NOW(), seen_by_human = TRUE, updated_at = NOW()
		WHERE project_id = $1 AND id = $2 AND seen_at IS NULL`,
		string(projectID), string(taskID),
	)
	if err != nil {
		return fmt.Errorf("mark task seen: %w", err)
	}
	// If no rows updated, check if task exists
	if tag.RowsAffected() == 0 {
		var count int
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(taskID)).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			return domain.ErrTaskNotFound
		}
		// Already seen — idempotent, no error
	}
	return nil
}

func (r *taskRepository) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	// Get current column
	var columnID string
	err := r.pool.QueryRow(ctx, `SELECT column_id FROM tasks WHERE project_id=$1 AND id=$2`, string(projectID), string(taskID)).Scan(&columnID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaskNotFound
		}
		return err
	}

	// Shift tasks to make room
	_, err = r.pool.Exec(ctx, `
		UPDATE tasks SET position = position + 1
		WHERE project_id = $1 AND column_id = $2 AND position >= $3 AND id != $4`,
		string(projectID), columnID, newPosition, string(taskID),
	)
	if err != nil {
		return fmt.Errorf("reorder shift: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE tasks SET position = $1, updated_at = NOW()
		WHERE project_id = $2 AND id = $3`,
		newPosition, string(projectID), string(taskID),
	)
	return err
}

func (r *taskRepository) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			date_series::date AS date,
			COALESCE(created_count, 0),
			COALESCE(completed_count, 0)
		FROM generate_series(
			(NOW() - ($2 || ' days')::interval)::date,
			NOW()::date,
			'1 day'::interval
		) AS date_series
		LEFT JOIN (
			SELECT created_at::date AS d, COUNT(*) AS created_count
			FROM tasks
			WHERE project_id = $1
			GROUP BY created_at::date
		) c ON c.d = date_series::date
		LEFT JOIN (
			SELECT completed_at::date AS d, COUNT(*) AS completed_count
			FROM tasks
			WHERE project_id = $1 AND completed_at IS NOT NULL
			GROUP BY completed_at::date
		) co ON co.d = date_series::date
		ORDER BY date_series ASC`,
		string(projectID), days,
	)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}
	defer rows.Close()

	var result []domain.TimelineEntry
	for rows.Next() {
		var entry domain.TimelineEntry
		var date time.Time
		if err := rows.Scan(&date, &entry.TasksCreated, &entry.TasksCompleted); err != nil {
			return nil, err
		}
		entry.Date = date.Format("2006-01-02")
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (r *taskRepository) UpdateSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE tasks SET session_id = $1, updated_at = NOW()
		WHERE project_id = $2 AND id = $3`,
		sessionID, string(projectID), string(taskID),
	)
	return err
}

func (r *taskRepository) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.RoleColdStartStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			assigned_role,
			COUNT(*) AS count,
			MIN(cold_start_input_tokens) AS min_input,
			MAX(cold_start_input_tokens) AS max_input,
			AVG(cold_start_input_tokens) AS avg_input,
			MIN(cold_start_output_tokens) AS min_output,
			MAX(cold_start_output_tokens) AS max_output,
			AVG(cold_start_output_tokens) AS avg_output,
			MIN(cold_start_cache_read_tokens) AS min_cache_read,
			MAX(cold_start_cache_read_tokens) AS max_cache_read,
			AVG(cold_start_cache_read_tokens) AS avg_cache_read
		FROM tasks
		WHERE project_id = $1
		  AND cold_start_input_tokens > 0
		GROUP BY assigned_role
		ORDER BY count DESC`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get cold start stats: %w", err)
	}
	defer rows.Close()

	var result []domain.RoleColdStartStat
	for rows.Next() {
		var stat domain.RoleColdStartStat
		err := rows.Scan(
			&stat.AssignedRole, &stat.Count,
			&stat.MinInputTokens, &stat.MaxInputTokens, &stat.AvgInputTokens,
			&stat.MinOutputTokens, &stat.MaxOutputTokens, &stat.AvgOutputTokens,
			&stat.MinCacheReadTokens, &stat.MaxCacheReadTokens, &stat.AvgCacheReadTokens,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

func (r *taskRepository) BulkCreate(ctx context.Context, projectID domain.ProjectID, taskList []domain.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, task := range taskList {
		filesJSON := jsonMarshal(task.FilesModified)
		contextJSON := jsonMarshal(task.ContextFiles)
		tagsJSON := jsonMarshal(task.Tags)

		_, err := tx.Exec(ctx, `
			INSERT INTO tasks (
				id, project_id, column_id, feature_id, title, summary, description,
				priority, priority_score, position,
				created_by_role, created_by_agent, assigned_role,
				is_blocked, blocked_reason, blocked_at, blocked_by_agent,
				wont_do_requested, wont_do_reason, wont_do_requested_by, wont_do_requested_at,
				completion_summary, completed_by_agent, completed_at,
				files_modified, resolution, context_files, tags, estimated_effort,
				seen_at, session_id,
				input_tokens, output_tokens, cache_read_tokens, cache_write_tokens, model,
				cold_start_input_tokens, cold_start_output_tokens, cold_start_cache_read_tokens, cold_start_cache_write_tokens,
				started_at, duration_seconds, human_estimate_seconds,
				created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				$8,$9,$10,
				$11,$12,$13,
				$14,$15,$16,$17,
				$18,$19,$20,$21,
				$22,$23,$24,
				$25,$26,$27,$28,$29,
				$30,$31,
				$32,$33,$34,$35,$36,
				$37,$38,$39,$40,
				$41,$42,$43,
				$44,$45
			)`,
			string(task.ID), string(projectID), string(task.ColumnID), featureIDOrNil(task.FeatureID),
			task.Title, task.Summary, task.Description,
			string(task.Priority), task.PriorityScore, task.Position,
			task.CreatedByRole, task.CreatedByAgent, task.AssignedRole,
			boolToInt(task.IsBlocked), task.BlockedReason, task.BlockedAt, task.BlockedByAgent,
			boolToInt(task.WontDoRequested), task.WontDoReason, task.WontDoRequestedBy, task.WontDoRequestedAt,
			task.CompletionSummary, task.CompletedByAgent, task.CompletedAt,
			filesJSON, task.Resolution, contextJSON, tagsJSON, task.EstimatedEffort,
			task.SeenAt, task.SessionID,
			task.InputTokens, task.OutputTokens, task.CacheReadTokens, task.CacheWriteTokens, task.Model,
			task.ColdStartInputTokens, task.ColdStartOutputTokens, task.ColdStartCacheReadTokens, task.ColdStartCacheWriteTokens,
			task.StartedAt, task.DurationSeconds, task.HumanEstimateSeconds,
			task.CreatedAt, task.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("bulk create task: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *taskRepository) BulkReassignInProject(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET assigned_role=$1 WHERE project_id=$2 AND assigned_role=$3`,
		newSlug, string(projectID), oldSlug,
	)
	if err != nil {
		return 0, fmt.Errorf("bulk reassign tasks: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *taskRepository) ListByAssignedRole(ctx context.Context, projectID domain.ProjectID, slug string) ([]domain.Task, error) {
	filters := tasks.TaskFilters{AssignedRole: &slug}
	withDetails, err := r.List(ctx, projectID, filters)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Task, len(withDetails))
	for i, t := range withDetails {
		result[i] = t.Task
	}
	return result, nil
}

func (r *taskRepository) SearchTasks(ctx context.Context, projectID domain.ProjectID, query string, limit int) ([]domain.TaskWithDetails, error) {
	filters := tasks.TaskFilters{
		Search: query,
		Limit:  limit,
	}
	return r.List(ctx, projectID, filters)
}

// ----------------------------
// columnRepository
// ----------------------------

type columnRepository struct{ *baseRepository }

func (r *columnRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
	// First verify project exists
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1)`, string(projectID)).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrProjectNotFound
	}

	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND id=$2`,
		string(projectID), string(id))
	return scanColumn(row)
}

func (r *columnRepository) FindBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug=$2`,
		string(projectID), string(slug))
	col, err := scanColumn(row)
	if errors.Is(err, domain.ErrColumnNotFound) {
		return nil, domain.ErrColumnNotFound
	}
	return col, err
}

func (r *columnRepository) List(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 ORDER BY position ASC`,
		string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer rows.Close()

	var result []domain.Column
	for rows.Next() {
		col, err := scanColumnRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *col)
	}
	return result, rows.Err()
}

func (r *columnRepository) EnsureBacklog(ctx context.Context, projectID domain.ProjectID) (*domain.Column, error) {
	// Try to get existing backlog
	row := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug='backlog'`,
		string(projectID))
	col, err := scanColumn(row)
	if err == nil {
		return col, nil
	}
	if !errors.Is(err, domain.ErrColumnNotFound) {
		return nil, err
	}

	// Create backlog column
	colID := string(domain.NewColumnID())
	_, err = r.pool.Exec(ctx, `
		INSERT INTO columns (id, project_id, slug, name, position, created_at)
		VALUES ($1, $2, 'backlog', 'Backlog', -1, NOW())
		ON CONFLICT (project_id, slug) DO NOTHING`,
		colID, string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("ensure backlog: %w", err)
	}

	// Fetch the created/existing backlog
	row = r.pool.QueryRow(ctx, `
		SELECT id, slug, name, position, created_at
		FROM columns WHERE project_id=$1 AND slug='backlog'`,
		string(projectID))
	return scanColumn(row)
}

func scanColumn(row pgx.Row) (*domain.Column, error) {
	var col domain.Column
	err := row.Scan((*string)(&col.ID), (*string)(&col.Slug), &col.Name, &col.Position, &col.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrColumnNotFound
		}
		return nil, err
	}
	return &col, nil
}

func scanColumnRow(rows pgx.Rows) (*domain.Column, error) {
	var col domain.Column
	err := rows.Scan((*string)(&col.ID), (*string)(&col.Slug), &col.Name, &col.Position, &col.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &col, nil
}

// ----------------------------
// commentRepository
// ----------------------------

type commentRepository struct{ *baseRepository }

func (r *commentRepository) Create(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO comments (id, task_id, author_role, author_name, author_type, content, edited_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		string(comment.ID), string(comment.TaskID),
		comment.AuthorRole, comment.AuthorName, string(comment.AuthorType),
		comment.Content, comment.EditedAt, comment.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *commentRepository) FindByID(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) (*domain.Comment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
		FROM comments WHERE id = $1`,
		string(id))
	return scanComment(row)
}

func (r *commentRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	query := `
		SELECT id, task_id, author_role, author_name, author_type, content, edited_at, created_at
		FROM comments WHERE task_id = $1 ORDER BY created_at ASC`
	args := []any{string(taskID)}
	argIdx := 2
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, argIdx)
		args = append(args, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var result []domain.Comment
	for rows.Next() {
		var c domain.Comment
		err := rows.Scan(
			(*string)(&c.ID), (*string)(&c.TaskID),
			&c.AuthorRole, &c.AuthorName, (*string)(&c.AuthorType),
			&c.Content, &c.EditedAt, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *commentRepository) Update(ctx context.Context, projectID domain.ProjectID, comment domain.Comment) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE comments SET content=$1, edited_at=$2 WHERE id=$3`,
		comment.Content, comment.EditedAt, string(comment.ID),
	)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *commentRepository) Delete(ctx context.Context, projectID domain.ProjectID, id domain.CommentID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id=$1`, string(id))
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}

func (r *commentRepository) Count(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM comments WHERE task_id=$1`, string(taskID)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return count, nil
}

func (r *commentRepository) IsLastComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (bool, error) {
	// First verify the comment exists
	var taskID string
	var createdAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT task_id, created_at FROM comments WHERE id=$1`, string(commentID)).Scan(&taskID, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, domain.ErrCommentNotFound
		}
		return false, err
	}

	// Check if it's the latest comment for its task
	var lastID string
	err = r.pool.QueryRow(ctx, `
		SELECT id FROM comments WHERE task_id=$1 ORDER BY created_at DESC LIMIT 1`, taskID).Scan(&lastID)
	if err != nil {
		return false, err
	}
	return lastID == string(commentID), nil
}

func scanComment(row pgx.Row) (*domain.Comment, error) {
	var c domain.Comment
	err := row.Scan(
		(*string)(&c.ID), (*string)(&c.TaskID),
		&c.AuthorRole, &c.AuthorName, (*string)(&c.AuthorType),
		&c.Content, &c.EditedAt, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCommentNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ----------------------------
// dependencyRepository
// ----------------------------

type dependencyRepository struct{ *baseRepository }

func (r *dependencyRepository) Create(ctx context.Context, projectID domain.ProjectID, dep domain.TaskDependency) error {
	// Check self-reference
	if dep.TaskID == dep.DependsOnTaskID {
		return domain.ErrCannotDependOnSelf
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO task_dependencies (id, task_id, depends_on_task_id, created_at)
		VALUES ($1, $2, $3, $4)`,
		string(dep.ID), string(dep.TaskID), string(dep.DependsOnTaskID), dep.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDependencyAlreadyExists
		}
		if isCheckViolation(err) {
			return domain.ErrCannotDependOnSelf
		}
		return fmt.Errorf("create dependency: %w", err)
	}
	return nil
}

func (r *dependencyRepository) Delete(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM task_dependencies WHERE task_id=$1 AND depends_on_task_id=$2`,
		string(taskID), string(dependsOnTaskID),
	)
	if err != nil {
		return fmt.Errorf("delete dependency: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDependencyNotFound
	}
	return nil
}

func (r *dependencyRepository) List(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE task_id=$1`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	defer rows.Close()
	return scanDependencies(rows)
}

func (r *dependencyRepository) WouldCreateCycle(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, dependsOnTaskID domain.TaskID) (bool, error) {
	// Self-reference always creates a cycle
	if taskID == dependsOnTaskID {
		return true, nil
	}

	// Check if adding taskID -> dependsOnTaskID would create a cycle.
	// A cycle exists if dependsOnTaskID can already reach taskID via existing dependencies.
	// i.e., is taskID reachable from dependsOnTaskID?
	var count int
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE reachable AS (
			SELECT depends_on_task_id AS tid FROM task_dependencies WHERE task_id = $1
			UNION
			SELECT td.depends_on_task_id FROM task_dependencies td
			INNER JOIN reachable r ON td.task_id = r.tid
		)
		SELECT COUNT(*) FROM reachable WHERE tid = $2`,
		string(dependsOnTaskID), string(taskID),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("would create cycle: %w", err)
	}
	return count > 0, nil
}

func (r *dependencyRepository) ListDependents(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, task_id, depends_on_task_id, created_at
		FROM task_dependencies WHERE depends_on_task_id=$1`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("list dependents: %w", err)
	}
	defer rows.Close()
	return scanDependencies(rows)
}

func (r *dependencyRepository) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.title, t.completion_summary, t.files_modified
		FROM task_dependencies td
		JOIN tasks t ON t.id = td.depends_on_task_id
		JOIN columns c ON c.id = t.column_id
		WHERE td.task_id = $1 AND c.slug = 'done'`,
		string(taskID))
	if err != nil {
		return nil, fmt.Errorf("get dependency context: %w", err)
	}
	defer rows.Close()

	var result []domain.DependencyContext
	for rows.Next() {
		var dc domain.DependencyContext
		var filesJSON []byte
		err := rows.Scan((*string)(&dc.TaskID), &dc.Title, &dc.CompletionSummary, &filesJSON)
		if err != nil {
			return nil, err
		}
		dc.FilesModified = jsonUnmarshalStrings(filesJSON)
		result = append(result, dc)
	}
	return result, rows.Err()
}

func scanDependencies(rows pgx.Rows) ([]domain.TaskDependency, error) {
	var result []domain.TaskDependency
	for rows.Next() {
		var dep domain.TaskDependency
		err := rows.Scan(
			(*string)(&dep.ID), (*string)(&dep.TaskID), (*string)(&dep.DependsOnTaskID), &dep.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, dep)
	}
	return result, rows.Err()
}

// ----------------------------
// toolUsageRepository
// ----------------------------

type toolUsageRepository struct{ *baseRepository }

func (r *toolUsageRepository) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	id := newID()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tool_usage (id, project_id, tool_name, count, last_used_at)
		VALUES ($1, $2, $3, 1, NOW())
		ON CONFLICT (project_id, tool_name) DO UPDATE
		SET count = tool_usage.count + 1, last_used_at = NOW()`,
		id, string(projectID), toolName,
	)
	if err != nil {
		return fmt.Errorf("increment tool usage: %w", err)
	}
	return nil
}

func (r *toolUsageRepository) ListToolUsage(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tool_name, count, last_used_at
		FROM tool_usage WHERE project_id=$1
		ORDER BY count DESC, tool_name ASC`,
		string(projectID))
	if err != nil {
		return nil, fmt.Errorf("list tool usage: %w", err)
	}
	defer rows.Close()

	var result []domain.ToolUsageStat
	for rows.Next() {
		var stat domain.ToolUsageStat
		err := rows.Scan(&stat.ToolName, &stat.ExecutionCount, &stat.LastExecutedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

func (r *taskRepository) GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			model,
			COUNT(*) AS task_count,
			SUM(input_tokens) AS input_tokens,
			SUM(output_tokens) AS output_tokens,
			SUM(cache_read_tokens) AS cache_read_tokens,
			SUM(cache_write_tokens) AS cache_write_tokens
		FROM tasks
		WHERE project_id = $1
		  AND model != ''
		  AND (input_tokens > 0 OR output_tokens > 0)
		GROUP BY model
		ORDER BY (SUM(input_tokens) + SUM(output_tokens)) DESC`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get model token stats: %w", err)
	}
	defer rows.Close()

	var result []domain.ModelTokenStat
	for rows.Next() {
		var stat domain.ModelTokenStat
		if err := rows.Scan(&stat.Model, &stat.TaskCount, &stat.InputTokens, &stat.OutputTokens, &stat.CacheReadTokens, &stat.CacheWriteTokens); err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

// --- Model Pricing Repository ---

type modelPricingRepository struct{ *baseRepository }

func (r *modelPricingRepository) ListAll(ctx context.Context) ([]domain.ModelPricing, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m, updated_at
		FROM model_pricing
		ORDER BY model_id`)
	if err != nil {
		return nil, fmt.Errorf("list model pricing: %w", err)
	}
	defer rows.Close()

	var result []domain.ModelPricing
	for rows.Next() {
		var p domain.ModelPricing
		if err := rows.Scan(&p.ID, &p.ModelID, &p.InputPricePer1M, &p.OutputPricePer1M, &p.CacheReadPricePer1M, &p.CacheWritePricePer1M, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *modelPricingRepository) Upsert(ctx context.Context, p domain.ModelPricing) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO model_pricing (id, model_id, input_price_per_1m, output_price_per_1m, cache_read_price_per_1m, cache_write_price_per_1m, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (model_id) DO UPDATE SET
			input_price_per_1m = EXCLUDED.input_price_per_1m,
			output_price_per_1m = EXCLUDED.output_price_per_1m,
			cache_read_price_per_1m = EXCLUDED.cache_read_price_per_1m,
			cache_write_price_per_1m = EXCLUDED.cache_write_price_per_1m,
			updated_at = NOW()`,
		p.ID, p.ModelID, p.InputPricePer1M, p.OutputPricePer1M, p.CacheReadPricePer1M, p.CacheWritePricePer1M,
	)
	return err
}

// ----------------------------
// featureRepository
// ----------------------------

type featureRepository struct{ *baseRepository }

func (r *featureRepository) Create(ctx context.Context, feature domain.Feature) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO features (id, project_id, name, description, status, created_by_role, created_by_agent, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		string(feature.ID), string(feature.ProjectID), feature.Name, feature.Description,
		string(feature.Status), feature.CreatedByRole, feature.CreatedByAgent,
		feature.CreatedAt, feature.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create feature: %w", err)
	}
	return nil
}

func (r *featureRepository) FindByID(ctx context.Context, id domain.FeatureID) (*domain.Feature, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, status, created_by_role, created_by_agent, created_at, updated_at
		FROM features WHERE id = $1`, string(id))
	var f domain.Feature
	err := row.Scan(
		(*string)(&f.ID), (*string)(&f.ProjectID), &f.Name, &f.Description,
		(*string)(&f.Status), &f.CreatedByRole, &f.CreatedByAgent, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrFeatureNotFound
		}
		return nil, fmt.Errorf("find feature by id: %w", err)
	}
	return &f, nil
}

func (r *featureRepository) List(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	args := []any{string(projectID)}
	query := `
		SELECT f.id, f.project_id, f.name, f.description, f.status, f.created_by_role, f.created_by_agent, f.created_at, f.updated_at,
			COALESCE(SUM(CASE WHEN c.slug = 'backlog'     THEN 1 ELSE 0 END), 0) AS backlog_count,
			COALESCE(SUM(CASE WHEN c.slug = 'todo'        THEN 1 ELSE 0 END), 0) AS todo_count,
			COALESCE(SUM(CASE WHEN c.slug = 'in_progress' THEN 1 ELSE 0 END), 0) AS in_progress_count,
			COALESCE(SUM(CASE WHEN c.slug = 'done'        THEN 1 ELSE 0 END), 0) AS done_count,
			COALESCE(SUM(CASE WHEN c.slug = 'blocked'     THEN 1 ELSE 0 END), 0) AS blocked_count
		FROM features f
		LEFT JOIN tasks t ON t.feature_id = f.id
		LEFT JOIN columns c ON c.id = t.column_id
		WHERE f.project_id = $1`

	if len(statusFilter) > 0 {
		statuses := make([]string, len(statusFilter))
		for i, s := range statusFilter {
			statuses[i] = string(s)
		}
		args = append(args, statuses)
		query += fmt.Sprintf(` AND f.status = ANY($%d)`, len(args))
	}
	query += ` GROUP BY f.id ORDER BY f.created_at ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	defer rows.Close()

	var result []domain.FeatureWithTaskSummary
	for rows.Next() {
		var fw domain.FeatureWithTaskSummary
		err := rows.Scan(
			(*string)(&fw.ID), (*string)(&fw.ProjectID), &fw.Name, &fw.Description,
			(*string)(&fw.Status), &fw.CreatedByRole, &fw.CreatedByAgent, &fw.CreatedAt, &fw.UpdatedAt,
			&fw.TaskSummary.BacklogCount,
			&fw.TaskSummary.TodoCount,
			&fw.TaskSummary.InProgressCount,
			&fw.TaskSummary.DoneCount,
			&fw.TaskSummary.BlockedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feature row: %w", err)
		}
		result = append(result, fw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list features rows: %w", err)
	}
	return result, nil
}

func (r *featureRepository) Update(ctx context.Context, feature domain.Feature) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE features SET name=$1, description=$2, updated_at=$3
		WHERE id=$4`,
		feature.Name, feature.Description, feature.UpdatedAt, string(feature.ID),
	)
	if err != nil {
		return fmt.Errorf("update feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) UpdateStatus(ctx context.Context, id domain.FeatureID, status domain.FeatureStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE features SET status=$1, updated_at=NOW()
		WHERE id=$2`,
		string(status), string(id),
	)
	if err != nil {
		return fmt.Errorf("update feature status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) Delete(ctx context.Context, id domain.FeatureID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM features WHERE id=$1`, string(id))
	if err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFeatureNotFound
	}
	return nil
}

func (r *featureRepository) GetStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM features WHERE project_id=$1 GROUP BY status`,
		string(projectID),
	)
	if err != nil {
		return nil, fmt.Errorf("get feature stats: %w", err)
	}
	defer rows.Close()

	stats := &domain.FeatureStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan feature stats: %w", err)
		}
		stats.TotalCount += count
		switch domain.FeatureStatus(status) {
		case domain.FeatureStatusDraft:
			stats.NotReadyCount += count
		case domain.FeatureStatusReady:
			stats.ReadyCount += count
		case domain.FeatureStatusInProgress:
			stats.InProgressCount += count
		case domain.FeatureStatusDone:
			stats.DoneCount += count
		case domain.FeatureStatusBlocked:
			stats.BlockedCount += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("feature stats rows: %w", err)
	}
	return stats, nil
}

// compile-time interface checks
var (
	_ projects.ProjectRepository        = (*projectRepository)(nil)
	_ agentsrepo.AgentRepository        = (*roleRepository)(nil)
	_ tasks.TaskRepository              = (*taskRepository)(nil)
	_ columns.ColumnRepository          = (*columnRepository)(nil)
	_ comments.CommentRepository        = (*commentRepository)(nil)
	_ dependencies.DependencyRepository = (*dependencyRepository)(nil)
	_ toolusage.ToolUsageRepository     = (*toolUsageRepository)(nil)
	_ skills.SkillRepository            = (*skillRepository)(nil)
	_ featuresrepo.FeatureRepository    = (*featureRepository)(nil)
)
