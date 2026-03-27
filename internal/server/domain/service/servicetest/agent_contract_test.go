package servicetest_test

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service/servicetest"
	"github.com/stretchr/testify/assert"
)

var _ service.Commands = (*servicetest.MockCommands)(nil)

func TestAgentManagementContracts(t *testing.T) {
	ctx := context.Background()

	t.Run("CloneAgent propagates ErrAgentNotFound", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			CloneAgentFunc: func(_ context.Context, _, _, _ string) (domain.Role, error) {
				return domain.Role{}, domain.ErrAgentNotFound
			},
		}
		_, err := mock.CloneAgent(ctx, "missing", "new-slug", "New")
		assert.ErrorIs(t, err, domain.ErrAgentNotFound)
	})

	t.Run("AssignAgentToProject propagates ErrAgentAlreadyInProject", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			AssignAgentToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ string) error {
				return domain.ErrAgentAlreadyInProject
			},
		}
		err := mock.AssignAgentToProject(ctx, domain.NewProjectID(), "some-agent")
		assert.ErrorIs(t, err, domain.ErrAgentAlreadyInProject)
	})

	t.Run("RemoveAgentFromProject with no tasks succeeds", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			RemoveAgentFromProjectFunc: func(_ context.Context, _ domain.ProjectID, _ string, _ *string, _ bool) error {
				return nil
			},
		}
		err := mock.RemoveAgentFromProject(ctx, domain.NewProjectID(), "some-agent", nil, false)
		assert.NoError(t, err)
	})

	t.Run("RemoveAgentFromProject propagates ErrAgentHasTasks when tasks exist and no reassign", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			RemoveAgentFromProjectFunc: func(_ context.Context, _ domain.ProjectID, _ string, _ *string, _ bool) error {
				return domain.ErrAgentHasTasks
			},
		}
		err := mock.RemoveAgentFromProject(ctx, domain.NewProjectID(), "some-agent", nil, false)
		assert.ErrorIs(t, err, domain.ErrAgentHasTasks)
	})

	t.Run("BulkReassignTasks returns count", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			BulkReassignTasksFunc: func(_ context.Context, _ domain.ProjectID, _, _ string) (int, error) {
				return 5, nil
			},
		}
		count, err := mock.BulkReassignTasks(ctx, domain.NewProjectID(), "old-agent", "new-agent")
		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("CreateSkill propagates ErrSkillAlreadyExists", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			CreateSkillFunc: func(_ context.Context, _, _, _, _, _, _ string, _ int) (domain.Skill, error) {
				return domain.Skill{}, domain.ErrSkillAlreadyExists
			},
		}
		_, err := mock.CreateSkill(ctx, "my-skill", "My Skill", "", "", "", "", 0)
		assert.ErrorIs(t, err, domain.ErrSkillAlreadyExists)
	})

	t.Run("AddSkillToAgent propagates ErrSkillNotFound", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			AddSkillToAgentFunc: func(_ context.Context, _, _ string) error {
				return domain.ErrSkillNotFound
			},
		}
		err := mock.AddSkillToAgent(ctx, "some-agent", "missing-skill")
		assert.ErrorIs(t, err, domain.ErrSkillNotFound)
	})

	t.Run("RemoveSkillFromAgent propagates ErrSkillNotFound", func(t *testing.T) {
		mock := &servicetest.MockCommands{
			RemoveSkillFromAgentFunc: func(_ context.Context, _, _ string) error {
				return domain.ErrSkillNotFound
			},
		}
		err := mock.RemoveSkillFromAgent(ctx, "some-agent", "missing-skill")
		assert.ErrorIs(t, err, domain.ErrSkillNotFound)
	})
}
