package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/app"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/agents/agentstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/skills/skillstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks/taskstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAgentMgmtApp(mockProjects *projectstest.MockProjectRepository, mockRoles *agentstest.MockRoleRepository, mockTasks *taskstest.MockTaskRepository, mockSkills *skillstest.MockSkill) *app.App {
	return app.NewApp(app.Config{
		Projects: mockProjects,
		Agents:    mockRoles,
		Tasks:    mockTasks,
		Skills:   mockSkills,
	})
}

// TestCloneRole

func TestCloneRole_Success(t *testing.T) {
	ctx := context.Background()
	sourceID := domain.NewRoleID()
	source := &domain.Role{ID: sourceID, Slug: "base", Name: "Base Role"}
	cloned := domain.Role{ID: domain.NewRoleID(), Slug: "copy", Name: "Copy Name"}

	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (*domain.Role, error) {
			if slug == "base" {
				return source, nil
			}
			return nil, errors.New("not found")
		},
		CloneFunc: func(_ context.Context, _ domain.RoleID, newSlug, newName string) (domain.Role, error) {
			cloned.Slug = newSlug
			cloned.Name = newName
			return cloned, nil
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	result, err := a.CloneRole(ctx, "base", "copy", "Copy Name")
	require.NoError(t, err)
	assert.Equal(t, "copy", result.Slug)
}

func TestCloneRole_DefaultName(t *testing.T) {
	ctx := context.Background()
	source := &domain.Role{ID: domain.NewRoleID(), Slug: "base", Name: "Base Role"}

	var capturedName string
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (*domain.Role, error) {
			if slug == "base" {
				return source, nil
			}
			return nil, errors.New("not found")
		},
		CloneFunc: func(_ context.Context, _ domain.RoleID, newSlug, newName string) (domain.Role, error) {
			capturedName = newName
			return domain.Role{Slug: newSlug, Name: newName}, nil
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	_, err := a.CloneRole(ctx, "base", "copy-slug", "")
	require.NoError(t, err)
	assert.Equal(t, "Base Role (copy)", capturedName)
}

func TestCloneRole_SourceNotFound(t *testing.T) {
	ctx := context.Background()
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	_, err := a.CloneRole(ctx, "missing", "new-slug", "")
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestCloneRole_NewSlugTaken(t *testing.T) {
	ctx := context.Background()
	source := &domain.Role{ID: domain.NewRoleID(), Slug: "base", Name: "Base"}
	existing := &domain.Role{ID: domain.NewRoleID(), Slug: "taken", Name: "Taken"}

	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (*domain.Role, error) {
			if slug == "base" {
				return source, nil
			}
			if slug == "taken" {
				return existing, nil
			}
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	_, err := a.CloneRole(ctx, "base", "taken", "")
	assert.ErrorIs(t, err, domain.ErrRoleAlreadyExists)
}

func TestCloneRole_EmptyNewSlug(t *testing.T) {
	ctx := context.Background()
	a := newAgentMgmtApp(nil, &agentstest.MockRoleRepository{}, nil, nil)

	_, err := a.CloneRole(ctx, "base", "", "")
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

// TestAssignAgentToProject

func TestAssignAgentToProject_Success(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		AssignToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
			return nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, nil, nil)

	err := a.AssignAgentToProject(ctx, projectID, "dev")
	require.NoError(t, err)
}

func TestAssignAgentToProject_EmptySlug(t *testing.T) {
	ctx := context.Background()
	a := newAgentMgmtApp(nil, &agentstest.MockRoleRepository{}, nil, nil)

	err := a.AssignAgentToProject(ctx, domain.NewProjectID(), "")
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

func TestAssignAgentToProject_ProjectNotFound(t *testing.T) {
	ctx := context.Background()
	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(mockProjects, &agentstest.MockRoleRepository{}, nil, nil)

	err := a.AssignAgentToProject(ctx, domain.NewProjectID(), "dev")
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestAssignAgentToProject_RoleNotFound(t *testing.T) {
	ctx := context.Background()
	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, nil, nil)

	err := a.AssignAgentToProject(ctx, domain.NewProjectID(), "missing")
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestAssignAgentToProject_AlreadyAssigned(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		AssignToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
			return domain.ErrAgentAlreadyInProject
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, nil, nil)

	err := a.AssignAgentToProject(ctx, projectID, "dev")
	assert.ErrorIs(t, err, domain.ErrAgentAlreadyInProject)
}

// TestRemoveAgentFromProject

func TestRemoveAgentFromProject_SuccessNoTasks(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		IsAssignedToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (bool, error) {
			return true, nil
		},
		RemoveFromProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
			return nil
		},
	}
	mockTasks := &taskstest.MockTaskRepository{
		ListByAssignedRoleFunc: func(_ context.Context, _ domain.ProjectID, _ string) ([]domain.Task, error) {
			return []domain.Task{}, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, mockTasks, nil)

	err := a.RemoveAgentFromProject(ctx, projectID, "dev", nil, false)
	require.NoError(t, err)
}

func TestRemoveAgentFromProject_SuccessWithReassignTo(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()
	targetRoleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (*domain.Role, error) {
			if slug == "dev" {
				return &domain.Role{ID: roleID, Slug: "dev"}, nil
			}
			if slug == "backend" {
				return &domain.Role{ID: targetRoleID, Slug: "backend"}, nil
			}
			return nil, errors.New("not found")
		},
		IsAssignedToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (bool, error) {
			return true, nil
		},
		RemoveFromProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
			return nil
		},
	}

	bulkCalled := false
	mockTasks := &taskstest.MockTaskRepository{
		ListByAssignedRoleFunc: func(_ context.Context, _ domain.ProjectID, _ string) ([]domain.Task, error) {
			return []domain.Task{{ID: domain.NewTaskID()}}, nil
		},
		BulkReassignInProjectFunc: func(_ context.Context, _ domain.ProjectID, _, _ string) (int, error) {
			bulkCalled = true
			return 1, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, mockTasks, nil)

	target := "backend"
	err := a.RemoveAgentFromProject(ctx, projectID, "dev", &target, false)
	require.NoError(t, err)
	assert.True(t, bulkCalled)
}

func TestRemoveAgentFromProject_SuccessWithClearAssignment(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		IsAssignedToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (bool, error) {
			return true, nil
		},
		RemoveFromProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
			return nil
		},
	}

	var capturedNewSlug string
	mockTasks := &taskstest.MockTaskRepository{
		ListByAssignedRoleFunc: func(_ context.Context, _ domain.ProjectID, _ string) ([]domain.Task, error) {
			return []domain.Task{{ID: domain.NewTaskID()}}, nil
		},
		BulkReassignInProjectFunc: func(_ context.Context, _ domain.ProjectID, _, newSlug string) (int, error) {
			capturedNewSlug = newSlug
			return 1, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, mockTasks, nil)

	err := a.RemoveAgentFromProject(ctx, projectID, "dev", nil, true)
	require.NoError(t, err)
	assert.Equal(t, "", capturedNewSlug)
}

func TestRemoveAgentFromProject_ErrAgentHasTasks(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		IsAssignedToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (bool, error) {
			return true, nil
		},
	}
	mockTasks := &taskstest.MockTaskRepository{
		ListByAssignedRoleFunc: func(_ context.Context, _ domain.ProjectID, _ string) ([]domain.Task, error) {
			return []domain.Task{{ID: domain.NewTaskID()}}, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, mockTasks, nil)

	err := a.RemoveAgentFromProject(ctx, projectID, "dev", nil, false)
	assert.ErrorIs(t, err, domain.ErrAgentHasTasks)
}

func TestRemoveAgentFromProject_AgentNotInProject(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: roleID, Slug: "dev"}, nil
		},
		IsAssignedToProjectFunc: func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (bool, error) {
			return false, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, nil, nil)

	err := a.RemoveAgentFromProject(ctx, projectID, "dev", nil, false)
	assert.ErrorIs(t, err, domain.ErrAgentNotInProject)
}

// TestBulkReassignTasks

func TestBulkReassignTasks_Success(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: domain.NewRoleID(), Slug: "backend"}, nil
		},
	}
	mockTasks := &taskstest.MockTaskRepository{
		BulkReassignInProjectFunc: func(_ context.Context, _ domain.ProjectID, _, _ string) (int, error) {
			return 3, nil
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, mockTasks, nil)

	count, err := a.BulkReassignTasks(ctx, projectID, "frontend", "backend")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestBulkReassignTasks_EmptyOldSlug(t *testing.T) {
	ctx := context.Background()
	a := newAgentMgmtApp(nil, nil, nil, nil)

	_, err := a.BulkReassignTasks(ctx, domain.NewProjectID(), "", "backend")
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

func TestBulkReassignTasks_NewSlugNotFound(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			return &domain.Project{ID: id}, nil
		},
	}
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(mockProjects, mockRoles, nil, nil)

	_, err := a.BulkReassignTasks(ctx, projectID, "frontend", "missing")
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

// TestAddSkillToAgent

func TestAddSkillToAgent_Success(t *testing.T) {
	ctx := context.Background()
	agentID := domain.NewRoleID()
	skillID := domain.NewSkillID()

	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: agentID, Slug: "dev"}, nil
		},
	}
	mockSkills := &skillstest.MockSkill{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
			return &domain.Skill{ID: skillID, Slug: "go-tools"}, nil
		},
		AssignToAgentFunc: func(_ context.Context, _ domain.RoleID, _ domain.SkillID) error {
			return nil
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, mockSkills)

	err := a.AddSkillToAgent(ctx, "dev", "go-tools")
	require.NoError(t, err)
}

func TestAddSkillToAgent_AgentNotFound(t *testing.T) {
	ctx := context.Background()
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	err := a.AddSkillToAgent(ctx, "missing", "go-tools")
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestAddSkillToAgent_SkillNotFound(t *testing.T) {
	ctx := context.Background()
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: domain.NewRoleID(), Slug: "dev"}, nil
		},
	}
	mockSkills := &skillstest.MockSkill{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, mockSkills)

	err := a.AddSkillToAgent(ctx, "dev", "missing-skill")
	assert.ErrorIs(t, err, domain.ErrSkillNotFound)
}

// TestRemoveSkillFromAgent

func TestRemoveSkillFromAgent_Success(t *testing.T) {
	ctx := context.Background()
	agentID := domain.NewRoleID()
	skillID := domain.NewSkillID()

	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: agentID, Slug: "dev"}, nil
		},
	}
	mockSkills := &skillstest.MockSkill{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
			return &domain.Skill{ID: skillID, Slug: "go-tools"}, nil
		},
		RemoveFromAgentFunc: func(_ context.Context, _ domain.RoleID, _ domain.SkillID) error {
			return nil
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, mockSkills)

	err := a.RemoveSkillFromAgent(ctx, "dev", "go-tools")
	require.NoError(t, err)
}

func TestRemoveSkillFromAgent_AgentNotFound(t *testing.T) {
	ctx := context.Background()
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, nil)

	err := a.RemoveSkillFromAgent(ctx, "missing", "go-tools")
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestRemoveSkillFromAgent_SkillNotFound(t *testing.T) {
	ctx := context.Background()
	mockRoles := &agentstest.MockRoleRepository{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Role, error) {
			return &domain.Role{ID: domain.NewRoleID(), Slug: "dev"}, nil
		},
	}
	mockSkills := &skillstest.MockSkill{
		FindBySlugFunc: func(_ context.Context, _ string) (*domain.Skill, error) {
			return nil, errors.New("not found")
		},
	}
	a := newAgentMgmtApp(nil, mockRoles, nil, mockSkills)

	err := a.RemoveSkillFromAgent(ctx, "dev", "missing-skill")
	assert.ErrorIs(t, err, domain.ErrSkillNotFound)
}
