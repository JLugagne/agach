package app

import (
	"context"
	"errors"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

func (a *App) CloneAgent(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Role, error) {
	logger := a.logger.WithContext(ctx)

	if newSlug == "" {
		return domain.Role{}, domain.ErrRoleSlugRequired
	}

	source, err := a.agents.FindBySlug(ctx, sourceSlug)
	if err != nil || source == nil {
		return domain.Role{}, domain.ErrRoleNotFound
	}

	existing, err := a.agents.FindBySlug(ctx, newSlug)
	if err == nil && existing != nil {
		return domain.Role{}, domain.ErrRoleAlreadyExists
	}

	if newName == "" {
		newName = source.Name + " (copy)"
	}

	cloned, err := a.agents.Clone(ctx, source.ID, newSlug, newName)
	if err != nil {
		return domain.Role{}, err
	}

	logger.WithField("sourceSlug", sourceSlug).WithField("newSlug", newSlug).Info("role cloned")
	return cloned, nil
}

func (a *App) AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error {
	logger := a.logger.WithContext(ctx)

	if agentSlug == "" {
		return domain.ErrRoleSlugRequired
	}

	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	role, err := a.agents.FindBySlug(ctx, agentSlug)
	if err != nil || role == nil {
		return domain.ErrRoleNotFound
	}

	if err := a.agents.AssignToProject(ctx, projectID, role.ID); err != nil {
		return err
	}

	logger.WithField("projectID", projectID).WithField("agentSlug", agentSlug).Info("agent assigned to project")
	return nil
}

func (a *App) RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error {
	logger := a.logger.WithContext(ctx)

	if agentSlug == "" {
		return domain.ErrRoleSlugRequired
	}

	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	role, err := a.agents.FindBySlug(ctx, agentSlug)
	if err != nil || role == nil {
		return domain.ErrRoleNotFound
	}

	assigned, err := a.agents.IsAssignedToProject(ctx, projectID, role.ID)
	if err != nil {
		return err
	}
	if !assigned {
		return domain.ErrAgentNotInProject
	}

	taskList, err := a.tasks.ListByAssignedRole(ctx, projectID, agentSlug)
	if err != nil {
		return err
	}

	if len(taskList) > 0 {
		if reassignTo != nil {
			if _, err := a.BulkReassignTasks(ctx, projectID, agentSlug, *reassignTo); err != nil {
				return err
			}
		} else if clearAssignment {
			if _, err := a.BulkReassignTasks(ctx, projectID, agentSlug, ""); err != nil {
				return err
			}
		} else {
			return domain.ErrAgentHasTasks
		}
	}

	if err := a.agents.RemoveFromProject(ctx, projectID, role.ID); err != nil {
		return err
	}

	logger.WithField("projectID", projectID).WithField("agentSlug", agentSlug).Info("agent removed from project")
	return nil
}

func (a *App) BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	logger := a.logger.WithContext(ctx)

	if oldSlug == "" {
		return 0, domain.ErrRoleSlugRequired
	}

	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		return 0, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return 0, domain.ErrProjectNotFound
	}

	if newSlug != "" {
		target, err := a.agents.FindBySlug(ctx, newSlug)
		if err != nil || target == nil {
			return 0, domain.ErrRoleNotFound
		}
	}

	count, err := a.tasks.BulkReassignInProject(ctx, projectID, oldSlug, newSlug)
	if err != nil {
		return 0, err
	}

	logger.WithField("projectID", projectID).WithField("oldSlug", oldSlug).WithField("newSlug", newSlug).WithField("count", count).Info("bulk reassigned tasks")
	return count, nil
}

func (a *App) AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error {
	agent, err := a.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return domain.ErrRoleNotFound
	}

	skill, err := a.skills.FindBySlug(ctx, skillSlug)
	if err != nil || skill == nil {
		return domain.ErrSkillNotFound
	}

	return a.skills.AssignToAgent(ctx, agent.ID, skill.ID)
}

func (a *App) RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error {
	agent, err := a.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return domain.ErrRoleNotFound
	}

	skill, err := a.skills.FindBySlug(ctx, skillSlug)
	if err != nil || skill == nil {
		return domain.ErrSkillNotFound
	}

	return a.skills.RemoveFromAgent(ctx, agent.ID, skill.ID)
}

func (a *App) ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error) {
	agent, err := a.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return nil, domain.ErrRoleNotFound
	}

	return a.skills.ListByAgent(ctx, agent.ID)
}

func (a *App) GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error) {
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	return a.tasks.ListByAssignedRole(ctx, projectID, agentSlug)
}
