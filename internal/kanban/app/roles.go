package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Role Commands

func (a *App) CreateRole(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Role, error) {
	logger := a.logger.WithContext(ctx)

	if slug == "" {
		return domain.Role{}, domain.ErrRoleSlugRequired
	}
	if name == "" {
		return domain.Role{}, domain.ErrRoleNameRequired
	}

	// Check if role with same slug already exists
	existing, err := a.roles.FindBySlug(ctx, slug)
	if err == nil && existing != nil {
		logger.WithField("slug", slug).Warn("role with slug already exists")
		return domain.Role{}, domain.ErrRoleAlreadyExists
	}

	role := domain.Role{
		ID:             domain.NewRoleID(),
		Slug:           slug,
		Name:           name,
		Icon:           icon,
		Color:          color,
		Description:    description,
		TechStack:      techStack,
		PromptHint:     promptHint,
		PromptTemplate: promptTemplate,
		SortOrder:      sortOrder,
		CreatedAt:      time.Now(),
	}

	if err := a.roles.Create(ctx, role); err != nil {
		logger.WithError(err).Error("failed to create role")
		return domain.Role{}, err
	}

	logger.WithField("roleID", role.ID).Info("role created successfully")
	return role, nil
}

func (a *App) UpdateRole(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error {
	logger := a.logger.WithContext(ctx).WithField("roleID", roleID)

	role, err := a.roles.FindByID(ctx, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to find role")
		return errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return domain.ErrRoleNotFound
	}

	if name != "" {
		role.Name = name
	}
	if icon != "" {
		role.Icon = icon
	}
	if color != "" {
		role.Color = color
	}
	if description != "" {
		role.Description = description
	}
	if promptHint != "" {
		role.PromptHint = promptHint
	}
	if promptTemplate != "" {
		role.PromptTemplate = promptTemplate
	}
	if techStack != nil {
		role.TechStack = techStack
	}
	if sortOrder != 0 {
		role.SortOrder = sortOrder
	}

	if err := a.roles.Update(ctx, *role); err != nil {
		logger.WithError(err).Error("failed to update role")
		return err
	}

	logger.Info("role updated successfully")
	return nil
}

func (a *App) DeleteRole(ctx context.Context, roleID domain.RoleID) error {
	logger := a.logger.WithContext(ctx).WithField("roleID", roleID)

	// Verify role exists
	role, err := a.roles.FindByID(ctx, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to find role")
		return errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return domain.ErrRoleNotFound
	}

	if err := a.roles.Delete(ctx, roleID); err != nil {
		logger.WithError(err).Error("failed to delete role")
		return err
	}

	logger.Info("role deleted successfully")
	return nil
}

// Role Queries

func (a *App) GetRole(ctx context.Context, roleID domain.RoleID) (*domain.Role, error) {
	logger := a.logger.WithContext(ctx).WithField("roleID", roleID)

	role, err := a.roles.FindByID(ctx, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return nil, errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return nil, domain.ErrRoleNotFound
	}

	return role, nil
}

func (a *App) GetRoleBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	logger := a.logger.WithContext(ctx).WithField("slug", slug)

	role, err := a.roles.FindBySlug(ctx, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get role by slug")
		return nil, errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return nil, domain.ErrRoleNotFound
	}

	return role, nil
}

func (a *App) ListRoles(ctx context.Context) ([]domain.Role, error) {
	logger := a.logger.WithContext(ctx)

	roles, err := a.roles.List(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to list roles")
		return nil, err
	}

	return roles, nil
}

// Per-project role commands

func (a *App) CreateProjectRole(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Role, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	if slug == "" {
		return domain.Role{}, domain.ErrRoleSlugRequired
	}
	if name == "" {
		return domain.Role{}, domain.ErrRoleNameRequired
	}

	existing, err := a.roles.FindBySlugInProject(ctx, projectID, slug)
	if err == nil && existing != nil {
		return domain.Role{}, domain.ErrRoleAlreadyExists
	}

	role := domain.Role{
		ID:             domain.NewRoleID(),
		Slug:           slug,
		Name:           name,
		Icon:           icon,
		Color:          color,
		Description:    description,
		TechStack:      techStack,
		PromptHint:     promptHint,
		PromptTemplate: promptTemplate,
		SortOrder:      sortOrder,
		CreatedAt:      time.Now(),
	}

	if err := a.roles.CreateInProject(ctx, projectID, role); err != nil {
		logger.WithError(err).Error("failed to create project role")
		return domain.Role{}, err
	}

	return role, nil
}

func (a *App) UpdateProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID).WithField("roleID", roleID)

	role, err := a.roles.FindByIDInProject(ctx, projectID, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to find project role")
		return errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return domain.ErrRoleNotFound
	}

	if name != "" {
		role.Name = name
	}
	if icon != "" {
		role.Icon = icon
	}
	if color != "" {
		role.Color = color
	}
	if description != "" {
		role.Description = description
	}
	if promptHint != "" {
		role.PromptHint = promptHint
	}
	if promptTemplate != "" {
		role.PromptTemplate = promptTemplate
	}
	if techStack != nil {
		role.TechStack = techStack
	}
	if sortOrder != 0 {
		role.SortOrder = sortOrder
	}

	if err := a.roles.UpdateInProject(ctx, projectID, *role); err != nil {
		logger.WithError(err).Error("failed to update project role")
		return err
	}

	return nil
}

func (a *App) DeleteProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID).WithField("roleID", roleID)

	role, err := a.roles.FindByIDInProject(ctx, projectID, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to find project role")
		return errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return domain.ErrRoleNotFound
	}

	if err := a.roles.DeleteInProject(ctx, projectID, roleID); err != nil {
		logger.WithError(err).Error("failed to delete project role")
		return err
	}

	return nil
}

// Per-project role queries

func (a *App) ListProjectRoles(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	roles, err := a.roles.ListInProject(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list project roles")
		return nil, err
	}

	return roles, nil
}

func (a *App) GetProjectRoleBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID).WithField("slug", slug)

	role, err := a.roles.FindBySlugInProject(ctx, projectID, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get project role by slug")
		return nil, errors.Join(domain.ErrRoleNotFound, err)
	}
	if role == nil {
		return nil, domain.ErrRoleNotFound
	}

	return role, nil
}
