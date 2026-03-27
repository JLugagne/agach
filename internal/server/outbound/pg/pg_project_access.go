package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type projectAccessRepository struct {
	*baseRepository
}

func (r *projectAccessRepository) GrantUser(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	id, _ := uuid.NewV7()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_user_access (id, project_id, user_id, role, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		id.String(), string(projectID), userID, role, time.Now(),
	)
	return err
}

func (r *projectAccessRepository) RevokeUser(ctx context.Context, projectID domain.ProjectID, userID string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(ctx,
		`DELETE FROM project_user_access WHERE project_id = $1 AND user_id = $2`,
		string(projectID), userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user access grant not found for project")
	}
	return nil
}

func (r *projectAccessRepository) UpdateUserRole(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	tag, err := r.pool.Exec(ctx,
		`UPDATE project_user_access SET role = $1 WHERE project_id = $2 AND user_id = $3`,
		role, string(projectID), userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user role grant not found for project")
	}
	return nil
}

func (r *projectAccessRepository) GrantTeam(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	id, _ := uuid.NewV7()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_team_access (id, project_id, team_id, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (project_id, team_id) DO NOTHING`,
		id.String(), string(projectID), teamID, time.Now(),
	)
	return err
}

func (r *projectAccessRepository) RevokeTeam(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`DELETE FROM project_team_access WHERE project_id = $1 AND team_id = $2`,
		string(projectID), teamID,
	)
	return err
}

func (r *projectAccessRepository) ListUserAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, user_id, role, created_at
		 FROM project_user_access WHERE project_id = $1 ORDER BY created_at`,
		string(projectID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProjectUserAccess
	for rows.Next() {
		var a domain.ProjectUserAccess
		var pid string
		if err := rows.Scan(&a.ID, &pid, &a.UserID, &a.Role, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ProjectID = domain.ProjectID(pid)
		result = append(result, a)
	}
	if result == nil {
		result = []domain.ProjectUserAccess{}
	}
	return result, rows.Err()
}

func (r *projectAccessRepository) ListTeamAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, team_id, created_at
		 FROM project_team_access WHERE project_id = $1 ORDER BY created_at`,
		string(projectID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProjectTeamAccess
	for rows.Next() {
		var a domain.ProjectTeamAccess
		var pid string
		if err := rows.Scan(&a.ID, &pid, &a.TeamID, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ProjectID = domain.ProjectID(pid)
		result = append(result, a)
	}
	if result == nil {
		result = []domain.ProjectTeamAccess{}
	}
	return result, rows.Err()
}

func (r *projectAccessRepository) HasAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	// Check direct user access first.
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_user_access WHERE project_id = $1 AND user_id = $2)`,
		string(projectID), userID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// Check team access.
	if len(teamIDs) == 0 {
		return false, nil
	}
	err = r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_team_access WHERE project_id = $1 AND team_id = ANY($2))`,
		string(projectID), teamIDs,
	).Scan(&exists)
	return exists, err
}

func (r *projectAccessRepository) ListAccessibleProjectIDs(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error) {
	ctx, cancel := r.ctx(ctx)
	defer cancel()

	var rows pgx.Rows
	var err error

	if len(teamIDs) > 0 {
		rows, err = r.pool.Query(ctx,
			`SELECT DISTINCT project_id FROM (
				SELECT project_id FROM project_user_access WHERE user_id = $1
				UNION
				SELECT project_id FROM project_team_access WHERE team_id = ANY($2)
			) sub`,
			userID, teamIDs,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT project_id FROM project_user_access WHERE user_id = $1`,
			userID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProjectID
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		result = append(result, domain.ProjectID(pid))
	}
	if result == nil {
		result = []domain.ProjectID{}
	}
	return result, rows.Err()
}
