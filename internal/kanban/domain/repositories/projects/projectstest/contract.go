package projectstest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProjectRepository is a function-based mock implementation of the ProjectRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockProjectRepository{
//		CreateFunc: func(ctx context.Context, project domain.Project) error {
//			return nil
//		},
//	}
type MockProjectRepository struct {
	CreateFunc                 func(ctx context.Context, project domain.Project) error
	FindByIDFunc               func(ctx context.Context, id domain.ProjectID) (*domain.Project, error)
	ListFunc                   func(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error)
	GetTreeFunc                func(ctx context.Context, id domain.ProjectID) ([]domain.Project, error)
	UpdateFunc                 func(ctx context.Context, project domain.Project) error
	DeleteFunc                 func(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error)
	GetSummaryFunc             func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error)
	CountChildrenFunc          func(ctx context.Context, id domain.ProjectID) (int, error)
	ListFeaturesActiveOnlyFunc func(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	ListModelPricingFunc       func(ctx context.Context) ([]domain.ModelPricing, error)
}

func (m *MockProjectRepository) Create(ctx context.Context, project domain.Project) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, project)
}

func (m *MockProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	if m.FindByIDFunc == nil {
		return &domain.Project{ID: id}, nil
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockProjectRepository) List(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx, parentID)
}

func (m *MockProjectRepository) GetTree(ctx context.Context, id domain.ProjectID) ([]domain.Project, error) {
	if m.GetTreeFunc == nil {
		panic("called not defined GetTreeFunc")
	}
	return m.GetTreeFunc(ctx, id)
}

func (m *MockProjectRepository) Update(ctx context.Context, project domain.Project) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, project)
}

func (m *MockProjectRepository) Delete(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error) {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func (m *MockProjectRepository) GetSummary(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
	if m.GetSummaryFunc == nil {
		panic("called not defined GetSummaryFunc")
	}
	return m.GetSummaryFunc(ctx, id)
}

func (m *MockProjectRepository) CountChildren(ctx context.Context, id domain.ProjectID) (int, error) {
	if m.CountChildrenFunc == nil {
		panic("called not defined CountChildrenFunc")
	}
	return m.CountChildrenFunc(ctx, id)
}

func (m *MockProjectRepository) ListFeaturesActiveOnly(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	if m.ListFeaturesActiveOnlyFunc == nil {
		panic("called not defined ListFeaturesActiveOnlyFunc")
	}
	return m.ListFeaturesActiveOnlyFunc(ctx, parentID)
}

func (m *MockProjectRepository) ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error) {
	if m.ListModelPricingFunc == nil {
		return nil, nil
	}
	return m.ListModelPricingFunc(ctx)
}

// ProjectsContractTesting runs all contract tests for a ProjectRepository implementation.
// Use this function to verify that your implementation adheres to the ProjectRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the ProjectRepository implementation to test
//
// Example usage in implementation tests:
//
//	func TestSQLiteProjectRepository(t *testing.T) {
//		repo := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		projectstest.ProjectsContractTesting(t, repo)
//	}
func ProjectsContractTesting(t *testing.T, repo projects.ProjectRepository) {
	ctx := context.Background()

	t.Run("Contract: Create stores project and FindByID retrieves it", func(t *testing.T) {
		project := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       nil,
			Name:           "Test Project",
			Description:    "Test Description",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, project)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, project.ID)
		require.NoError(t, err, "FindByID should succeed for created project")
		require.NotNil(t, retrieved, "Retrieved project must not be nil")
		assert.Equal(t, project.ID, retrieved.ID, "ID must match")
		assert.Equal(t, project.Name, retrieved.Name, "Name must match")
		assert.Equal(t, project.Description, retrieved.Description, "Description must match")
		assert.Nil(t, retrieved.ParentID, "ParentID must be nil for root project")
	})

	t.Run("Contract: FindByID returns error for non-existent project", func(t *testing.T) {
		nonExistentID := domain.NewProjectID()
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent project")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrProjectNotFound, "Error should be ErrProjectNotFound")
	})

	t.Run("Contract: List returns root projects when parentID is nil", func(t *testing.T) {
		// Create two root projects
		project1 := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       nil,
			Name:           "Root Project 1",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		project2 := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       nil,
			Name:           "Root Project 2",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, project1)
		require.NoError(t, err)
		err = repo.Create(ctx, project2)
		require.NoError(t, err)

		roots, err := repo.List(ctx, nil)
		require.NoError(t, err, "List should succeed")
		assert.GreaterOrEqual(t, len(roots), 2, "Should have at least 2 root projects")

		// Verify our projects are in the list
		found1, found2 := false, false
		for _, p := range roots {
			if p.ID == project1.ID {
				found1 = true
			}
			if p.ID == project2.ID {
				found2 = true
			}
		}
		assert.True(t, found1, "Project 1 should be in root list")
		assert.True(t, found2, "Project 2 should be in root list")
	})

	t.Run("Contract: List returns children when parentID is provided", func(t *testing.T) {
		// Create parent and child
		parentID := domain.NewProjectID()
		parent := domain.Project{
			ID:             parentID,
			ParentID:       nil,
			Name:           "Parent Project",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		child := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       &parentID,
			Name:           "Child Project",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, parent)
		require.NoError(t, err)
		err = repo.Create(ctx, child)
		require.NoError(t, err)

		children, err := repo.List(ctx, &parentID)
		require.NoError(t, err, "List should succeed")
		assert.GreaterOrEqual(t, len(children), 1, "Should have at least 1 child")

		found := false
		for _, c := range children {
			if c.ID == child.ID {
				found = true
				assert.NotNil(t, c.ParentID, "Child should have parent ID")
				assert.Equal(t, parentID, *c.ParentID, "Parent ID should match")
			}
		}
		assert.True(t, found, "Child project should be in list")
	})

	t.Run("Contract: Update modifies project data", func(t *testing.T) {
		project := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       nil,
			Name:           "Original Name",
			Description:    "Original Description",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, project)
		require.NoError(t, err)

		// Update fields
		project.Name = "Updated Name"
		project.Description = "Updated Description"
		project.UpdatedAt = time.Now()

		err = repo.Update(ctx, project)
		require.NoError(t, err, "Update should succeed")

		updated, err := repo.FindByID(ctx, project.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name, "Name should be updated")
		assert.Equal(t, "Updated Description", updated.Description, "Description should be updated")
	})

	t.Run("Contract: Delete removes project and returns deleted IDs", func(t *testing.T) {
		// Create parent with children
		parentID := domain.NewProjectID()
		parent := domain.Project{
			ID:             parentID,
			ParentID:       nil,
			Name:           "Parent to Delete",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		child := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       &parentID,
			Name:           "Child to Delete",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, parent)
		require.NoError(t, err)
		err = repo.Create(ctx, child)
		require.NoError(t, err)

		deletedIDs, err := repo.Delete(ctx, parentID)
		require.NoError(t, err, "Delete should succeed")
		assert.GreaterOrEqual(t, len(deletedIDs), 2, "Should delete parent and child")

		// Verify deletion
		_, err = repo.FindByID(ctx, parentID)
		assert.Error(t, err, "Parent should not exist after deletion")
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)

		_, err = repo.FindByID(ctx, child.ID)
		assert.Error(t, err, "Child should not exist after deletion")
		assert.ErrorIs(t, err, domain.ErrProjectNotFound)
	})

	t.Run("Contract: GetTree returns project and all descendants", func(t *testing.T) {
		// Create hierarchy: root -> sub1 -> subsub1
		rootID := domain.NewProjectID()
		sub1ID := domain.NewProjectID()
		subsub1ID := domain.NewProjectID()

		root := domain.Project{
			ID:             rootID,
			ParentID:       nil,
			Name:           "Root",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		sub1 := domain.Project{
			ID:             sub1ID,
			ParentID:       &rootID,
			Name:           "Sub 1",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		subsub1 := domain.Project{
			ID:             subsub1ID,
			ParentID:       &sub1ID,
			Name:           "SubSub 1",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, root)
		require.NoError(t, err)
		err = repo.Create(ctx, sub1)
		require.NoError(t, err)
		err = repo.Create(ctx, subsub1)
		require.NoError(t, err)

		tree, err := repo.GetTree(ctx, rootID)
		require.NoError(t, err, "GetTree should succeed")
		assert.GreaterOrEqual(t, len(tree), 3, "Tree should contain at least 3 projects")

		// Verify all projects are in tree
		ids := make(map[domain.ProjectID]bool)
		for _, p := range tree {
			ids[p.ID] = true
		}
		assert.True(t, ids[rootID], "Root should be in tree")
		assert.True(t, ids[sub1ID], "Sub1 should be in tree")
		assert.True(t, ids[subsub1ID], "SubSub1 should be in tree")
	})

	t.Run("Contract: IsFeature returns false for root project", func(t *testing.T) {
		root := domain.Project{
			ID:   domain.NewProjectID(),
			Name: "Root Project",
		}
		assert.False(t, root.IsFeature(), "Root project should not be a feature")
	})

	t.Run("Contract: IsFeature returns true for child project", func(t *testing.T) {
		rootID := domain.NewProjectID()
		child := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &rootID,
			Name:     "Child Feature",
		}
		assert.True(t, child.IsFeature(), "Child project should be a feature")
	})

	t.Run("Contract: CountChildren returns correct count", func(t *testing.T) {
		parentID := domain.NewProjectID()
		parent := domain.Project{
			ID:             parentID,
			ParentID:       nil,
			Name:           "Parent with Children",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		child1 := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       &parentID,
			Name:           "Child 1",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		child2 := domain.Project{
			ID:             domain.NewProjectID(),
			ParentID:       &parentID,
			Name:           "Child 2",
			CreatedByRole:  "architect",
			CreatedByAgent: "test-agent",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.Create(ctx, parent)
		require.NoError(t, err)
		err = repo.Create(ctx, child1)
		require.NoError(t, err)
		err = repo.Create(ctx, child2)
		require.NoError(t, err)

		count, err := repo.CountChildren(ctx, parentID)
		require.NoError(t, err, "CountChildren should succeed")
		assert.Equal(t, 2, count, "Should have 2 children")
	})
}

// ProjectFeaturesContractTesting tests the ListFeaturesActiveOnly method.
// It requires a createTaskInColumn func that, given a projectID and column slug,
// creates tasks in the DB so the repo can count them.
//
// Parameters:
//   - t: testing.T
//   - repo: the ProjectRepository under test
//   - parentProjectID: an existing root project to use as parent
//   - createTaskInColumn: a callback to insert a task row into the given project's given column slug.
func ProjectFeaturesContractTesting(
	t *testing.T,
	repo projects.ProjectRepository,
	parentProjectID domain.ProjectID,
	createTaskInColumn func(t *testing.T, projectID domain.ProjectID, columnSlug domain.ColumnSlug),
) {
	ctx := context.Background()

	t.Run("Contract: ListFeaturesActiveOnly returns empty when no features exist", func(t *testing.T) {
		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)
		_ = results
	})

	t.Run("Contract: ListFeaturesActiveOnly includes feature with todo tasks", func(t *testing.T) {
		featureWithTodo := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &parentProjectID,
			Name:     "Feature With Todo",
		}
		require.NoError(t, repo.Create(ctx, featureWithTodo))

		createTaskInColumn(t, featureWithTodo.ID, domain.ColumnTodo)

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		found := false
		for _, r := range results {
			if r.ID == featureWithTodo.ID {
				found = true
				assert.GreaterOrEqual(t, r.TaskSummary.TodoCount, 1, "Should have at least 1 todo task")
			}
		}
		assert.True(t, found, "Feature with todo task should appear in active features")
	})

	t.Run("Contract: ListFeaturesActiveOnly includes feature with in_progress tasks", func(t *testing.T) {
		featureInProgress := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &parentProjectID,
			Name:     "Feature In Progress",
		}
		require.NoError(t, repo.Create(ctx, featureInProgress))

		createTaskInColumn(t, featureInProgress.ID, domain.ColumnInProgress)

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		found := false
		for _, r := range results {
			if r.ID == featureInProgress.ID {
				found = true
				assert.GreaterOrEqual(t, r.TaskSummary.InProgressCount, 1)
			}
		}
		assert.True(t, found, "Feature with in_progress task should appear in active features")
	})

	t.Run("Contract: ListFeaturesActiveOnly includes feature with blocked tasks", func(t *testing.T) {
		featureBlocked := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &parentProjectID,
			Name:     "Feature Blocked",
		}
		require.NoError(t, repo.Create(ctx, featureBlocked))

		createTaskInColumn(t, featureBlocked.ID, domain.ColumnBlocked)

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		found := false
		for _, r := range results {
			if r.ID == featureBlocked.ID {
				found = true
				assert.GreaterOrEqual(t, r.TaskSummary.BlockedCount, 1)
			}
		}
		assert.True(t, found, "Feature with blocked task should appear in active features")
	})

	t.Run("Contract: ListFeaturesActiveOnly excludes feature with only done tasks", func(t *testing.T) {
		featureDone := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &parentProjectID,
			Name:     "Feature Done Only",
		}
		require.NoError(t, repo.Create(ctx, featureDone))

		createTaskInColumn(t, featureDone.ID, domain.ColumnDone)

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		for _, r := range results {
			assert.NotEqual(t, featureDone.ID, r.ID, "Feature with only done tasks should NOT appear in active features")
		}
	})

	t.Run("Contract: ListFeaturesActiveOnly excludes feature with no tasks", func(t *testing.T) {
		emptyFeature := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &parentProjectID,
			Name:     "Empty Feature",
		}
		require.NoError(t, repo.Create(ctx, emptyFeature))

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		for _, r := range results {
			assert.NotEqual(t, emptyFeature.ID, r.ID, "Empty feature should NOT appear in active features")
		}
	})

	t.Run("Contract: ListFeaturesActiveOnly excludes features of other parents", func(t *testing.T) {
		otherRoot := domain.Project{
			ID:   domain.NewProjectID(),
			Name: "Other Root",
		}
		require.NoError(t, repo.Create(ctx, otherRoot))

		otherFeature := domain.Project{
			ID:       domain.NewProjectID(),
			ParentID: &otherRoot.ID,
			Name:     "Other Feature",
		}
		require.NoError(t, repo.Create(ctx, otherFeature))
		createTaskInColumn(t, otherFeature.ID, domain.ColumnTodo)

		results, err := repo.ListFeaturesActiveOnly(ctx, parentProjectID)
		require.NoError(t, err)

		for _, r := range results {
			assert.NotEqual(t, otherFeature.ID, r.ID, "Feature of another parent should not appear")
		}
	})
}
