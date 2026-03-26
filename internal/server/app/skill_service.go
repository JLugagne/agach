package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	skillsrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/skills"
	"github.com/sirupsen/logrus"
)

type SkillService struct {
	skills skillsrepo.SkillRepository
	logger *logrus.Logger
}

func newSkillService(skills skillsrepo.SkillRepository, logger *logrus.Logger) *SkillService {
	return &SkillService{
		skills: skills,
		logger: logger,
	}
}

func (s *SkillService) CreateSkill(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error) {
	logger := s.logger.WithContext(ctx)

	if slug == "" {
		return domain.Skill{}, domain.ErrSkillSlugRequired
	}
	if name == "" {
		return domain.Skill{}, domain.ErrSkillNameRequired
	}

	existing, err := s.skills.FindBySlug(ctx, slug)
	if err == nil && existing != nil {
		return domain.Skill{}, domain.ErrSkillAlreadyExists
	}

	skill := domain.Skill{
		ID:          domain.NewSkillID(),
		Slug:        slug,
		Name:        name,
		Description: description,
		Content:     content,
		Icon:        icon,
		Color:       color,
		SortOrder:   sortOrder,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.skills.Create(ctx, skill); err != nil {
		logger.WithError(err).Error("failed to create skill")
		return domain.Skill{}, err
	}

	logger.WithField("skillID", skill.ID).Info("skill created")
	return skill, nil
}

func (s *SkillService) UpdateSkill(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error {
	logger := s.logger.WithContext(ctx).WithField("skillID", skillID)

	skill, err := s.skills.FindByID(ctx, skillID)
	if err != nil {
		logger.WithError(err).Error("failed to find skill")
		return domain.ErrSkillNotFound
	}
	if skill == nil {
		return domain.ErrSkillNotFound
	}

	if name != "" {
		skill.Name = name
	}
	if description != "" {
		skill.Description = description
	}
	if content != "" {
		skill.Content = content
	}
	if icon != "" {
		skill.Icon = icon
	}
	if color != "" {
		skill.Color = color
	}
	if sortOrder != 0 {
		skill.SortOrder = sortOrder
	}
	skill.UpdatedAt = time.Now()

	if err := s.skills.Update(ctx, *skill); err != nil {
		logger.WithError(err).Error("failed to update skill")
		return err
	}

	return nil
}

func (s *SkillService) DeleteSkill(ctx context.Context, skillID domain.SkillID) error {
	logger := s.logger.WithContext(ctx).WithField("skillID", skillID)

	skill, err := s.skills.FindByID(ctx, skillID)
	if err != nil {
		logger.WithError(err).Error("failed to find skill")
		return domain.ErrSkillNotFound
	}
	if skill == nil {
		return domain.ErrSkillNotFound
	}

	if err := s.skills.Delete(ctx, skillID); err != nil {
		logger.WithError(err).Error("failed to delete skill")
		return err
	}

	return nil
}

func (s *SkillService) GetSkill(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error) {
	logger := s.logger.WithContext(ctx).WithField("skillID", skillID)

	skill, err := s.skills.FindByID(ctx, skillID)
	if err != nil {
		logger.WithError(err).Error("failed to get skill")
		return nil, domain.ErrSkillNotFound
	}
	if skill == nil {
		return nil, domain.ErrSkillNotFound
	}

	return skill, nil
}

func (s *SkillService) GetSkillBySlug(ctx context.Context, slug string) (*domain.Skill, error) {
	logger := s.logger.WithContext(ctx).WithField("slug", slug)

	skill, err := s.skills.FindBySlug(ctx, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get skill by slug")
		return nil, domain.ErrSkillNotFound
	}
	if skill == nil {
		return nil, domain.ErrSkillNotFound
	}

	return skill, nil
}

func (s *SkillService) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	logger := s.logger.WithContext(ctx)

	list, err := s.skills.List(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to list skills")
		return nil, err
	}

	return list, nil
}
