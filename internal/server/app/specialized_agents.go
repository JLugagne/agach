package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	agentsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents"
	skillsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	specializedrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/specialized"
	"github.com/sirupsen/logrus"
)

type SpecializedAgentService struct {
	specAgents agentsrepo.AgentRepository
	specSkills skillsrepo.SkillRepository
	specRepo   specializedrepo.SpecializedAgentRepository
	specLogger *logrus.Logger
}

func newSpecializedAgentService(
	agents agentsrepo.AgentRepository,
	skills skillsrepo.SkillRepository,
	specialized specializedrepo.SpecializedAgentRepository,
	logger *logrus.Logger,
) *SpecializedAgentService {
	return &SpecializedAgentService{
		specAgents: agents,
		specSkills: skills,
		specRepo:   specialized,
		specLogger: logger,
	}
}

func (s *SpecializedAgentService) CreateSpecializedAgent(ctx context.Context, parentSlug, slug, name string, skillSlugs []string, sortOrder int) (domain.SpecializedAgent, error) {
	if slug == "" {
		return domain.SpecializedAgent{}, domain.ErrSpecializedAgentSlugRequired
	}
	if name == "" {
		return domain.SpecializedAgent{}, domain.ErrSpecializedAgentNameRequired
	}

	parent, err := s.specAgents.FindBySlug(ctx, parentSlug)
	if err != nil || parent == nil {
		return domain.SpecializedAgent{}, domain.ErrAgentNotFound
	}

	existing, err := s.specRepo.FindBySlug(ctx, slug)
	if err == nil && existing != nil {
		return domain.SpecializedAgent{}, domain.ErrSpecializedAgentAlreadyExists
	}

	now := time.Now()
	agent := domain.SpecializedAgent{
		ID:            domain.NewSpecializedAgentID(),
		ParentAgentID: parent.ID,
		Slug:          slug,
		Name:          name,
		SortOrder:     sortOrder,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.specRepo.Create(ctx, agent); err != nil {
		return domain.SpecializedAgent{}, err
	}

	if len(skillSlugs) > 0 {
		skillIDs, err := s.resolveSkillSlugs(ctx, skillSlugs)
		if err != nil {
			return domain.SpecializedAgent{}, err
		}
		if err := s.specRepo.SetSkills(ctx, agent.ID, skillIDs); err != nil {
			return domain.SpecializedAgent{}, err
		}
	}

	return agent, nil
}

func (s *SpecializedAgentService) UpdateSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID, name string, skillSlugs []string, sortOrder int) error {
	existing, err := s.specRepo.FindByID(ctx, id)
	if err != nil || existing == nil {
		return domain.ErrSpecializedAgentNotFound
	}

	if name != "" {
		existing.Name = name
	}
	if sortOrder != 0 {
		existing.SortOrder = sortOrder
	}

	if err := s.specRepo.Update(ctx, *existing); err != nil {
		return err
	}

	if skillSlugs != nil {
		skillIDs, err := s.resolveSkillSlugs(ctx, skillSlugs)
		if err != nil {
			return err
		}
		if err := s.specRepo.SetSkills(ctx, id, skillIDs); err != nil {
			return err
		}
	}

	return nil
}

func (s *SpecializedAgentService) DeleteSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID) error {
	existing, err := s.specRepo.FindByID(ctx, id)
	if err != nil || existing == nil {
		return domain.ErrSpecializedAgentNotFound
	}
	return s.specRepo.Delete(ctx, id)
}

func (s *SpecializedAgentService) ListSpecializedAgents(ctx context.Context, parentSlug string) ([]domain.SpecializedAgent, error) {
	parent, err := s.specAgents.FindBySlug(ctx, parentSlug)
	if err != nil || parent == nil {
		return nil, domain.ErrAgentNotFound
	}
	return s.specRepo.ListByParent(ctx, parent.ID)
}

func (s *SpecializedAgentService) GetSpecializedAgent(ctx context.Context, slug string) (*domain.SpecializedAgent, error) {
	agent, err := s.specRepo.FindBySlug(ctx, slug)
	if err != nil || agent == nil {
		return nil, domain.ErrSpecializedAgentNotFound
	}
	return agent, nil
}

func (s *SpecializedAgentService) ListSpecializedAgentSkills(ctx context.Context, slug string) ([]domain.Skill, error) {
	agent, err := s.specRepo.FindBySlug(ctx, slug)
	if err != nil || agent == nil {
		return nil, domain.ErrSpecializedAgentNotFound
	}
	return s.specRepo.ListSkills(ctx, agent.ID)
}

func (s *SpecializedAgentService) CountSpecializedByParent(ctx context.Context, parentSlug string) (int, error) {
	parent, err := s.specAgents.FindBySlug(ctx, parentSlug)
	if err != nil || parent == nil {
		return 0, domain.ErrAgentNotFound
	}
	return s.specRepo.CountByParent(ctx, parent.ID)
}

func (s *SpecializedAgentService) resolveSkillSlugs(ctx context.Context, slugs []string) ([]domain.SkillID, error) {
	ids := make([]domain.SkillID, 0, len(slugs))
	for _, slug := range slugs {
		skill, err := s.specSkills.FindBySlug(ctx, slug)
		if err != nil || skill == nil {
			return nil, domain.ErrSkillNotFound
		}
		ids = append(ids, skill.ID)
	}
	return ids, nil
}
