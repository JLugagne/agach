package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	skillsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	specializedrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/specialized"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/sirupsen/logrus"
)

type AgentService struct {
	agents      agentsrepo.AgentRepository
	specialized specializedrepo.SpecializedAgentRepository
	projects    projects.ProjectRepository
	tasks       tasks.TaskRepository
	skills      skillsrepo.SkillRepository
	logger      *logrus.Logger
}

func newAgentService(
	agents agentsrepo.AgentRepository,
	specialized specializedrepo.SpecializedAgentRepository,
	projects projects.ProjectRepository,
	tasks tasks.TaskRepository,
	skills skillsrepo.SkillRepository,
	logger *logrus.Logger,
) *AgentService {
	return &AgentService{
		agents:      agents,
		specialized: specialized,
		projects:    projects,
		tasks:       tasks,
		skills:      skills,
		logger:      logger,
	}
}

func (s *AgentService) CreateAgent(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error) {
	logger := s.logger.WithContext(ctx)

	if slug == "" {
		return domain.Agent{}, domain.ErrAgentSlugRequired
	}
	if name == "" {
		return domain.Agent{}, domain.ErrAgentNameRequired
	}

	existing, err := s.agents.FindBySlug(ctx, slug)
	if err == nil && existing != nil {
		logger.WithField("slug", slug).Warn("agent with slug already exists")
		return domain.Agent{}, domain.ErrAgentAlreadyExists
	}

	agent := domain.Agent{
		ID:             domain.NewAgentID(),
		Slug:           slug,
		Name:           name,
		Icon:           icon,
		Color:          color,
		Description:    description,
		TechStack:      techStack,
		PromptHint:     promptHint,
		PromptTemplate: promptTemplate,
		Model:          model,
		Thinking:       thinking,
		SortOrder:      sortOrder,
		CreatedAt:      time.Now(),
	}

	if err := s.agents.Create(ctx, agent); err != nil {
		logger.WithError(err).Error("failed to create agent")
		return domain.Agent{}, err
	}

	logger.WithField("agentID", agent.ID).Info("agent created successfully")
	return agent, nil
}

func (s *AgentService) UpdateAgent(ctx context.Context, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error {
	logger := s.logger.WithContext(ctx).WithField("agentID", agentID)

	agent, err := s.agents.FindByID(ctx, agentID)
	if err != nil {
		logger.WithError(err).Error("failed to find agent")
		return errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return domain.ErrAgentNotFound
	}

	if name != "" {
		agent.Name = name
	}
	if icon != "" {
		agent.Icon = icon
	}
	if color != "" {
		agent.Color = color
	}
	if description != "" {
		agent.Description = description
	}
	if promptHint != "" {
		agent.PromptHint = promptHint
	}
	if promptTemplate != "" {
		agent.PromptTemplate = promptTemplate
	}
	if model != "" {
		agent.Model = model
	}
	if thinking != "" {
		agent.Thinking = thinking
	}
	if techStack != nil {
		agent.TechStack = techStack
	}
	if sortOrder != 0 {
		agent.SortOrder = sortOrder
	}

	if err := s.agents.Update(ctx, *agent); err != nil {
		logger.WithError(err).Error("failed to update agent")
		return err
	}

	logger.Info("agent updated successfully")
	return nil
}

func (s *AgentService) DeleteAgent(ctx context.Context, agentID domain.AgentID) error {
	logger := s.logger.WithContext(ctx).WithField("agentID", agentID)

	agent, err := s.agents.FindByID(ctx, agentID)
	if err != nil {
		logger.WithError(err).Error("failed to find agent")
		return errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return domain.ErrAgentNotFound
	}

	if err := s.agents.Delete(ctx, agentID); err != nil {
		logger.WithError(err).Error("failed to delete agent")
		return err
	}

	logger.Info("agent deleted successfully")
	return nil
}

func (s *AgentService) GetAgent(ctx context.Context, agentID domain.AgentID) (*domain.Agent, error) {
	logger := s.logger.WithContext(ctx).WithField("agentID", agentID)

	agent, err := s.agents.FindByID(ctx, agentID)
	if err != nil {
		logger.WithError(err).Error("failed to get agent")
		return nil, errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return nil, domain.ErrAgentNotFound
	}

	return agent, nil
}

func (s *AgentService) GetAgentBySlug(ctx context.Context, slug string) (*domain.Agent, error) {
	logger := s.logger.WithContext(ctx).WithField("slug", slug)

	agent, err := s.agents.FindBySlug(ctx, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get agent by slug")
		return nil, errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return nil, domain.ErrAgentNotFound
	}

	return agent, nil
}

func (s *AgentService) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	logger := s.logger.WithContext(ctx)

	agents, err := s.agents.List(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to list agents")
		return nil, err
	}

	return agents, nil
}

func (s *AgentService) CreateProjectAgent(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	if slug == "" {
		return domain.Agent{}, domain.ErrAgentSlugRequired
	}
	if name == "" {
		return domain.Agent{}, domain.ErrAgentNameRequired
	}

	existing, err := s.agents.FindBySlugInProject(ctx, projectID, slug)
	if err == nil && existing != nil {
		return domain.Agent{}, domain.ErrAgentAlreadyExists
	}

	agent := domain.Agent{
		ID:             domain.NewAgentID(),
		Slug:           slug,
		Name:           name,
		Icon:           icon,
		Color:          color,
		Description:    description,
		TechStack:      techStack,
		PromptHint:     promptHint,
		PromptTemplate: promptTemplate,
		Model:          model,
		Thinking:       thinking,
		SortOrder:      sortOrder,
		CreatedAt:      time.Now(),
	}

	if err := s.agents.CreateInProject(ctx, projectID, agent); err != nil {
		logger.WithError(err).Error("failed to create project agent")
		return domain.Agent{}, err
	}

	return agent, nil
}

func (s *AgentService) UpdateProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID).WithField("agentID", agentID)

	agent, err := s.agents.FindByIDInProject(ctx, projectID, agentID)
	if err != nil {
		logger.WithError(err).Error("failed to find project agent")
		return errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return domain.ErrAgentNotFound
	}

	if name != "" {
		agent.Name = name
	}
	if icon != "" {
		agent.Icon = icon
	}
	if color != "" {
		agent.Color = color
	}
	if description != "" {
		agent.Description = description
	}
	if promptHint != "" {
		agent.PromptHint = promptHint
	}
	if promptTemplate != "" {
		agent.PromptTemplate = promptTemplate
	}
	if model != "" {
		agent.Model = model
	}
	if thinking != "" {
		agent.Thinking = thinking
	}
	if techStack != nil {
		agent.TechStack = techStack
	}
	if sortOrder != 0 {
		agent.SortOrder = sortOrder
	}

	if err := s.agents.UpdateInProject(ctx, projectID, *agent); err != nil {
		logger.WithError(err).Error("failed to update project agent")
		return err
	}

	return nil
}

func (s *AgentService) DeleteProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID).WithField("agentID", agentID)

	agent, err := s.agents.FindByIDInProject(ctx, projectID, agentID)
	if err != nil {
		logger.WithError(err).Error("failed to find project agent")
		return errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return domain.ErrAgentNotFound
	}

	if err := s.agents.DeleteInProject(ctx, projectID, agentID); err != nil {
		logger.WithError(err).Error("failed to delete project agent")
		return err
	}

	return nil
}

func (s *AgentService) ListProjectAgents(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	agents, err := s.agents.ListInProject(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list project agents")
		return nil, err
	}

	return agents, nil
}

func (s *AgentService) GetProjectAgentBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID).WithField("slug", slug)

	agent, err := s.agents.FindBySlugInProject(ctx, projectID, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get project agent by slug")
		return nil, errors.Join(domain.ErrAgentNotFound, err)
	}
	if agent == nil {
		return nil, domain.ErrAgentNotFound
	}

	return agent, nil
}

func (s *AgentService) CloneAgent(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Agent, error) {
	logger := s.logger.WithContext(ctx)

	if newSlug == "" {
		return domain.Agent{}, domain.ErrAgentSlugRequired
	}

	source, err := s.agents.FindBySlug(ctx, sourceSlug)
	if err != nil || source == nil {
		return domain.Agent{}, domain.ErrAgentNotFound
	}

	existing, err := s.agents.FindBySlug(ctx, newSlug)
	if err == nil && existing != nil {
		return domain.Agent{}, domain.ErrAgentAlreadyExists
	}

	if newName == "" {
		newName = source.Name + " (copy)"
	}

	cloned, err := s.agents.Clone(ctx, source.ID, newSlug, newName)
	if err != nil {
		return domain.Agent{}, err
	}

	logger.WithField("sourceSlug", sourceSlug).WithField("newSlug", newSlug).Info("agent cloned")
	return cloned, nil
}

func (s *AgentService) AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error {
	logger := s.logger.WithContext(ctx)

	if agentSlug == "" {
		return domain.ErrAgentSlugRequired
	}

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	// Try specialized agent first, then fall back to parent agent
	var spec *domain.SpecializedAgent
	if s.specialized != nil {
		spec, _ = s.specialized.FindBySlug(ctx, agentSlug)
	}
	var parentID domain.AgentID
	if spec != nil {
		parentID = spec.ParentAgentID
	} else {
		agent, err := s.agents.FindBySlug(ctx, agentSlug)
		if err != nil || agent == nil {
			return domain.ErrAgentNotFound
		}
		parentID = agent.ID
	}

	if err := s.agents.AssignToProject(ctx, projectID, parentID); err != nil {
		return err
	}

	logger.WithField("projectID", projectID).WithField("agentSlug", agentSlug).Info("agent assigned to project")
	return nil
}

func (s *AgentService) RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error {
	logger := s.logger.WithContext(ctx)

	if agentSlug == "" {
		return domain.ErrAgentSlugRequired
	}

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.ErrProjectNotFound
	}

	agent, err := s.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return domain.ErrAgentNotFound
	}

	assigned, err := s.agents.IsAssignedToProject(ctx, projectID, agent.ID)
	if err != nil {
		return err
	}
	if !assigned {
		return domain.ErrAgentNotInProject
	}

	taskList, err := s.tasks.ListByAssignedRole(ctx, projectID, agentSlug)
	if err != nil {
		return err
	}

	if len(taskList) > 0 {
		if reassignTo != nil {
			if _, err := s.BulkReassignTasks(ctx, projectID, agentSlug, *reassignTo); err != nil {
				return err
			}
		} else if clearAssignment {
			if _, err := s.BulkReassignTasks(ctx, projectID, agentSlug, ""); err != nil {
				return err
			}
		} else {
			return domain.ErrAgentHasTasks
		}
	}

	if err := s.agents.RemoveFromProject(ctx, projectID, agent.ID); err != nil {
		return err
	}

	logger.WithField("projectID", projectID).WithField("agentSlug", agentSlug).Info("agent removed from project")
	return nil
}

func (s *AgentService) BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	logger := s.logger.WithContext(ctx)

	if oldSlug == "" {
		return 0, domain.ErrAgentSlugRequired
	}

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		return 0, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return 0, domain.ErrProjectNotFound
	}

	if newSlug != "" {
		target, err := s.agents.FindBySlug(ctx, newSlug)
		if err != nil || target == nil {
			return 0, domain.ErrAgentNotFound
		}
	}

	count, err := s.tasks.BulkReassignInProject(ctx, projectID, oldSlug, newSlug)
	if err != nil {
		return 0, err
	}

	logger.WithField("projectID", projectID).WithField("oldSlug", oldSlug).WithField("newSlug", newSlug).WithField("count", count).Info("bulk reassigned tasks")
	return count, nil
}

func (s *AgentService) AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error {
	agent, err := s.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return domain.ErrAgentNotFound
	}

	skill, err := s.skills.FindBySlug(ctx, skillSlug)
	if err != nil || skill == nil {
		return domain.ErrSkillNotFound
	}

	return s.skills.AssignToAgent(ctx, agent.ID, skill.ID)
}

func (s *AgentService) RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error {
	agent, err := s.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return domain.ErrAgentNotFound
	}

	skill, err := s.skills.FindBySlug(ctx, skillSlug)
	if err != nil || skill == nil {
		return domain.ErrSkillNotFound
	}

	return s.skills.RemoveFromAgent(ctx, agent.ID, skill.ID)
}

func (s *AgentService) ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error) {
	agent, err := s.agents.FindBySlug(ctx, agentSlug)
	if err != nil || agent == nil {
		return nil, domain.ErrAgentNotFound
	}

	return s.skills.ListByAgent(ctx, agent.ID)
}

func (s *AgentService) GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error) {
	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	return s.tasks.ListByAssignedRole(ctx, projectID, agentSlug)
}
