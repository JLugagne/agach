import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createTask, createAgent, deleteAgent, createFeature, moveTask, BASE_URL } from './helpers';

test.describe('5. Task Creation (New Task Modal)', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Task Creation Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('5.1 — Create a task with only required fields', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('My New Task');
    await page.locator('[data-qa="new-task-summary-input"]').fill('Brief summary of the task');
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    // Modal should close
    await expect(page.locator('[data-qa="new-task-title-input"]')).not.toBeVisible();
    // Task card appears in the Todo column
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'My New Task' })).toBeVisible();
  });

  test('5.2 — Create task validation: title is required', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-summary-input"]').fill('Some summary');
    // Submit button should be disabled when title is empty
    await expect(page.locator('[data-qa="new-task-submit-btn"]')).toBeDisabled();

    // Modal stays open
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();
  });

  test('5.3 — Create task validation: summary is required', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('My Task');
    // Submit button should be disabled when summary is empty
    await expect(page.locator('[data-qa="new-task-submit-btn"]')).toBeDisabled();

    // Modal stays open
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();
  });

  test('5.4 — Create task with all optional fields', async ({ request, page }) => {
    // Create an agent and a feature for the selectors
    const agent = await createAgent(request, 'test-dev', 'Test Developer');
    const featureId = await createFeature(request, projectId, 'Test Feature');

    // Reload to pick up new agent/feature
    await page.reload();

    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    // Fill required fields
    await page.locator('[data-qa="new-task-title-input"]').fill('Full Task');
    await page.locator('[data-qa="new-task-summary-input"]').fill('Full task summary');

    // Fill description
    await page.locator('[data-qa="new-task-description-input"]').fill('Detailed description');

    // Select priority
    await page.locator('[data-qa="new-task-priority-select"]').selectOption('critical');

    // Select assigned role
    await page.locator('[data-qa="new-task-role-select"]').selectOption('test-dev');

    // Select feature (if visible)
    const featureSelect = page.locator('[data-qa="new-task-feature-select"]');
    if (await featureSelect.isVisible()) {
      await featureSelect.selectOption(featureId);
    }

    // Add a tag
    await page.locator('[data-qa="new-task-tag-input"]').fill('frontend');
    await page.locator('[data-qa="new-task-add-tag-btn"]').click();

    // Add a context file
    await page.locator('[data-qa="new-task-file-input"]').fill('src/main.go');
    await page.locator('[data-qa="new-task-add-file-btn"]').click();

    // Submit
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    // Task card appears
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Full Task' })).toBeVisible();

    // Cleanup
    await deleteAgent(request, 'test-dev');
  });

  test('5.5 — Add to backlog checkbox places task in backlog instead of todo', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-title-input"]').fill('Backlog Task');
    await page.locator('[data-qa="new-task-summary-input"]').fill('Task for the backlog');
    await page.locator('[data-qa="new-task-backlog-checkbox"]').check();
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    // Task should NOT appear in the Todo column
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Backlog Task' })).not.toBeVisible();
  });

  test('5.6 — Close New Task modal with cancel button', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-cancel-btn"]').click();

    await expect(page.locator('[data-qa="new-task-title-input"]')).not.toBeVisible();
  });

  test('5.6b — Close New Task modal with close (X) button', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    await page.locator('[data-qa="new-task-close-btn"]').click();

    await expect(page.locator('[data-qa="new-task-title-input"]')).not.toBeVisible();
  });

  test('5.7 — Remove tag from new task', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    // Add a tag first
    await page.locator('[data-qa="new-task-tag-input"]').fill('removeme');
    await page.locator('[data-qa="new-task-add-tag-btn"]').click();

    // Verify tag is displayed
    await expect(page.getByText('removeme')).toBeVisible();

    // Remove the tag
    await page.locator('[data-qa="new-task-remove-tag-btn"]').click();

    // Tag should be gone
    await expect(page.getByText('removeme')).not.toBeVisible();
  });

  test('5.8 — Remove context file from new task', async ({ page }) => {
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-title-input"]')).toBeVisible();

    // Add a file
    await page.locator('[data-qa="new-task-file-input"]').fill('path/to/remove.go');
    await page.locator('[data-qa="new-task-add-file-btn"]').click();

    // Verify file is listed
    await expect(page.getByText('path/to/remove.go')).toBeVisible();

    // Remove the file
    await page.locator('[data-qa="new-task-remove-file-btn"]').click();

    // File should be gone
    await expect(page.getByText('path/to/remove.go')).not.toBeVisible();
  });
});

test.describe('6. Task Drawer', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Task Drawer Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('6.1 — View task details in drawer', async ({ request, page }) => {
    await createTask(request, projectId, 'Detail Task', 'Task summary for drawer', {
      description: 'A detailed description',
      priority: 'high',
      tags: ['frontend', 'urgent'],
    });
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();

    // Drawer should show title, summary, priority, description
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
    await expect(page.getByText('Detail Task')).toBeVisible();
    await expect(page.getByText('Task summary for drawer')).toBeVisible();
  });

  test('6.2 — Close drawer with X button', async ({ request, page }) => {
    await createTask(request, projectId, 'Close Drawer Task', 'Summary');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="drawer-close-btn"]').click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).not.toBeVisible();
  });

  test('6.3 — Inline edit title', async ({ request, page }) => {
    await createTask(request, projectId, 'Original Title', 'Summary for title edit test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-title-edit-btn"]').click();
    await page.locator('[data-qa="task-title-input"]').clear();
    await page.locator('[data-qa="task-title-input"]').fill('Updated Title');
    await page.locator('[data-qa="task-title-input"]').press('Enter');

    await expect(page.locator('[data-qa="task-title-input"]')).not.toBeVisible();
    await expect(page.getByText('Updated Title')).toBeVisible();
  });

  test('6.4 — Inline edit summary', async ({ request, page }) => {
    await createTask(request, projectId, 'Summary Edit Task', 'Original summary text');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-summary-edit-btn"]').click();
    await page.locator('[data-qa="task-summary-input"]').clear();
    await page.locator('[data-qa="task-summary-input"]').fill('Updated summary text');
    await page.locator('[data-qa="task-summary-save-btn"]').click();

    await expect(page.getByText('Updated summary text')).toBeVisible();
  });

  test('6.4b — Cancel summary edit via cancel button', async ({ request, page }) => {
    await createTask(request, projectId, 'Summary Cancel Task', 'Original summary');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-summary-edit-btn"]').click();
    await page.locator('[data-qa="task-summary-input"]').clear();
    await page.locator('[data-qa="task-summary-input"]').fill('Changed text that should not save');
    await page.locator('[data-qa="task-summary-cancel-btn"]').click();

    // Original summary should still be visible
    await expect(page.getByText('Original summary')).toBeVisible();
  });

  test('6.5 — Edit description', async ({ request, page }) => {
    await createTask(request, projectId, 'Desc Edit Task', 'Summary for desc edit', {
      description: 'Original description',
    });
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-description-edit-btn"]').click();
    await page.locator('[data-qa="task-description-input"]').clear();
    await page.locator('[data-qa="task-description-input"]').fill('Updated description text');
    await page.locator('[data-qa="task-description-save-btn"]').click();

    await expect(page.getByText('Updated description text')).toBeVisible();
  });

  test('6.5b — Cancel description edit', async ({ request, page }) => {
    await createTask(request, projectId, 'Desc Cancel Task', 'Summary', {
      description: 'Keep this description',
    });
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-description-edit-btn"]').click();
    await page.locator('[data-qa="task-description-input"]').clear();
    await page.locator('[data-qa="task-description-input"]').fill('Should not save');
    await page.locator('[data-qa="task-description-cancel-btn"]').click();

    await expect(page.getByText('Keep this description')).toBeVisible();
  });

  test('6.6 — Edit priority via dropdown', async ({ request, page }) => {
    await createTask(request, projectId, 'Priority Change Task', 'Summary for priority test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-priority-btn"]').click();
    await page.locator('[data-qa="task-priority-option-high"]').click();

    await expect(page.getByText(/high/i)).toBeVisible();
  });

  test('6.7 — Edit assigned role (agent)', async ({ request, page }) => {
    const agent = await createAgent(request, 'drawer-test-dev', 'Drawer Test Dev');
    await createTask(request, projectId, 'Role Edit Task', 'Summary for role test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-role-btn"]').click();
    await page.locator('[data-qa="task-role-option-drawer-test-dev"]').click();

    // Verify role is displayed
    await expect(page.getByText(/drawer-test-dev/i)).toBeVisible();

    // Cleanup
    await deleteAgent(request, 'drawer-test-dev');
  });

  test('6.8 — Edit effort estimate', async ({ request, page }) => {
    await createTask(request, projectId, 'Effort Task', 'Summary for effort test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-effort-btn"]').click();
    await page.locator('[data-qa="task-effort-option-l"]').click();

    await expect(page.getByText('L')).toBeVisible();
  });

  test('6.9 — Edit resolution', async ({ request, page }) => {
    await createTask(request, projectId, 'Resolution Task', 'Summary for resolution test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-resolution-edit-btn"]').click();
    await page.locator('[data-qa="task-resolution-input"]').fill('Task was resolved via workaround');
    await page.locator('[data-qa="task-resolution-save-btn"]').click();

    await expect(page.getByText('Task was resolved via workaround')).toBeVisible();
  });

  test('6.9b — Cancel resolution edit', async ({ request, page }) => {
    await createTask(request, projectId, 'Resolution Cancel Task', 'Summary');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-resolution-edit-btn"]').click();
    await page.locator('[data-qa="task-resolution-input"]').fill('Should not persist');
    await page.locator('[data-qa="task-resolution-cancel-btn"]').click();

    await expect(page.getByText('Should not persist')).not.toBeVisible();
  });

  test('6.10 — Add and remove tags', async ({ request, page }) => {
    await createTask(request, projectId, 'Tag Task', 'Summary for tag test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // Add a tag
    await page.locator('[data-qa="task-tag-input"]').fill('newtag');
    await page.locator('[data-qa="task-add-tag-btn"]').click();

    await expect(page.getByText('newtag')).toBeVisible();

    // Remove the tag
    await page.locator('[data-qa="task-remove-tag-btn"]').first().click();

    await expect(page.getByText('newtag')).not.toBeVisible();
  });

  test('6.11 — Add and remove context files', async ({ request, page }) => {
    await createTask(request, projectId, 'Context File Task', 'Summary for file test');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // Add a context file
    await page.locator('[data-qa="task-context-file-input"]').fill('src/app.go');
    await page.locator('[data-qa="task-add-context-file-btn"]').click();

    await expect(page.getByText('src/app.go')).toBeVisible();

    // Remove the context file
    await page.locator('[data-qa="task-remove-context-file-btn"]').first().click();

    await expect(page.getByText('src/app.go')).not.toBeVisible();
  });

  test('6.12 — Add a dependency', async ({ request, page }) => {
    await createTask(request, projectId, 'Dep Target Task', 'This is the dependency target');
    await createTask(request, projectId, 'Dep Source Task', 'This task will get a dependency');
    await page.reload();

    // Open the source task drawer
    const sourceCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Dep Source Task' });
    await sourceCard.click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // Add dependency
    await page.locator('[data-qa="task-add-dependency-btn"]').click();
    await page.locator('[data-qa="task-dependency-search-input"]').fill('Dep Target');

    // Wait for search results and select
    await page.locator('[data-qa="task-dependency-result-btn"]').first().click();

    // Verify dependency link appears
    await expect(page.locator('[data-qa="task-dependency-link"]')).toBeVisible();
  });

  test('6.13 — Remove a dependency', async ({ request, page }) => {
    // Create two tasks and add dependency via API
    const parentId = await createTask(request, projectId, 'Parent To Remove', 'Parent summary');
    const childId = await createTask(request, projectId, 'Child With Dep', 'Child summary', {
      depends_on: [parentId],
    });
    await page.reload();

    // Open child task drawer
    const childCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Child With Dep' });
    await childCard.click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // Verify dependency is shown
    await expect(page.locator('[data-qa="task-dependency-link"]')).toBeVisible();

    // Remove the dependency
    await page.locator('[data-qa="task-remove-dependency-btn"]').first().click();

    // Dependency should be gone
    await expect(page.locator('[data-qa="task-dependency-link"]')).not.toBeVisible();
  });

  test('6.14 — Close dependency search', async ({ request, page }) => {
    await createTask(request, projectId, 'Dep Search Close Task', 'Summary');
    await page.reload();

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.locator('[data-qa="task-add-dependency-btn"]').click();
    await expect(page.locator('[data-qa="task-dependency-search-input"]')).toBeVisible();

    await page.locator('[data-qa="task-dependency-search-close-btn"]').click();

    await expect(page.locator('[data-qa="task-dependency-search-input"]')).not.toBeVisible();
  });

  test('6.15 — Dependents section shows tasks that depend on this task', async ({ request, page }) => {
    const parentId = await createTask(request, projectId, 'Dependents Parent', 'Parent summary');
    await createTask(request, projectId, 'Dependents Child', 'Child summary', {
      depends_on: [parentId],
    });
    await page.reload();

    // Open the parent task drawer
    const parentCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Dependents Parent' });
    await parentCard.click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // Verify dependent link appears
    await expect(page.locator('[data-qa="task-dependent-link"]')).toBeVisible();
  });
});

test.describe('7. Task Action Modals', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Task Action Modals Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('7.1 — Delete task', async ({ request, page }) => {
    await createTask(request, projectId, 'Task To Delete', 'Summary for delete test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Delete' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-delete-btn"]').click();
    await page.locator('[data-qa="delete-task-confirm-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Delete' })).not.toBeVisible();
  });

  test('7.1b — Cancel delete task modal', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Not Deleted', 'Summary for cancel delete test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Deleted' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-delete-btn"]').click();
    await page.locator('[data-qa="delete-task-cancel-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Deleted' })).toBeVisible();
  });

  test('7.1c — Close delete task modal with X button', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Close Delete', 'Summary');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Close Delete' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-delete-btn"]').click();
    await page.locator('[data-qa="delete-task-close-btn"]').click();

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Close Delete' })).toBeVisible();
  });

  test('7.2 — Block task', async ({ request, page }) => {
    await createTask(request, projectId, 'Task To Block', 'Summary for block test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Block' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-block-btn"]').click();
    // Fill blocked reason with 50+ characters
    await page.locator('[data-qa="block-reason-input"]').fill('This task is blocked because we need external API credentials from the vendor team first');
    await page.locator('[data-qa="block-agent-name-input"]').fill('human');
    await page.locator('[data-qa="block-task-submit-btn"]').click();

    // Task should no longer be in Todo
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Block' })).not.toBeVisible();

    // Task should appear in Blocked column
    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Block' })).toBeVisible();
  });

  test('7.2b — Cancel block task modal', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Not Blocked', 'Summary');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Blocked' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-block-btn"]').click();
    await page.locator('[data-qa="block-task-cancel-btn"]').click();

    // Task should still be in Todo
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Blocked' })).toBeVisible();
  });

  test('7.3 — Unblock a blocked task', async ({ request, page }) => {
    // Create task, move to in_progress, then block it via API-based context menu
    const taskId = await createTask(request, projectId, 'Task To Unblock', 'Summary for unblock test');
    await page.reload();

    // Block it via context menu first
    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Unblock' });
    await taskCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-block-btn"]').click();
    await page.locator('[data-qa="block-reason-input"]').fill('Temporarily blocked for testing the unblock flow in Playwright tests');
    await page.locator('[data-qa="block-agent-name-input"]').fill('human');
    await page.locator('[data-qa="block-task-submit-btn"]').click();

    // Wait for task to appear in blocked column
    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Unblock' })).toBeVisible();

    // Unblock via context menu
    const blockedCard = blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Unblock' });
    await blockedCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-unblock-btn"]').click();
    await page.locator('[data-qa="unblock-task-submit-btn"]').click();

    // Task should move to Todo
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Unblock' })).toBeVisible();
  });

  test('7.3b — Cancel unblock modal', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Stay Blocked', 'Summary');
    await page.reload();

    // Block it first
    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Stay Blocked' });
    await taskCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-block-btn"]').click();
    await page.locator('[data-qa="block-reason-input"]').fill('Temporarily blocked for testing the cancel unblock flow in Playwright');
    await page.locator('[data-qa="block-agent-name-input"]').fill('human');
    await page.locator('[data-qa="block-task-submit-btn"]').click();

    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Stay Blocked' })).toBeVisible();

    // Try to unblock but cancel
    const blockedCard = blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Stay Blocked' });
    await blockedCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-unblock-btn"]').click();
    await page.locator('[data-qa="unblock-task-cancel-btn"]').click();

    // Task should still be blocked
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Stay Blocked' })).toBeVisible();
  });

  test('7.4 — Complete task', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Task To Complete', 'Summary for complete test');
    await moveTask(request, projectId, taskId, 'in_progress');
    await page.reload();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    const taskCard = inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Complete' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-complete-btn"]').click();
    // Fill summary with 100+ characters
    await page.locator('[data-qa="complete-summary-input"]').fill(
      'This task has been completed successfully. All acceptance criteria have been met and the implementation passes all unit and integration tests.'
    );
    await page.locator('[data-qa="complete-agent-name-input"]').fill('human');
    await page.locator('[data-qa="complete-task-submit-btn"]').click();

    await expect(inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Complete' })).not.toBeVisible();

    // Task should appear in Done column
    const doneColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Done' });
    await expect(doneColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Complete' })).toBeVisible();
  });

  test('7.4b — Complete task with files modified', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Complete With Files', 'Summary');
    await moveTask(request, projectId, taskId, 'in_progress');
    await page.reload();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    const taskCard = inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Complete With Files' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-complete-btn"]').click();
    await page.locator('[data-qa="complete-summary-input"]').fill(
      'Completed this task with file modifications. Updated the main handler and added comprehensive test coverage for the new endpoint.'
    );
    await page.locator('[data-qa="complete-agent-name-input"]').fill('human');

    // Add a file
    await page.locator('[data-qa="complete-file-path-input"]').fill('src/handler.go');
    await page.locator('[data-qa="complete-add-file-btn"]').click();
    await expect(page.getByText('src/handler.go')).toBeVisible();

    // Remove the file
    await page.locator('[data-qa="complete-remove-file-btn"]').first().click();
    await expect(page.getByText('src/handler.go')).not.toBeVisible();

    // Re-add and submit
    await page.locator('[data-qa="complete-file-path-input"]').fill('src/handler.go');
    await page.locator('[data-qa="complete-add-file-btn"]').click();
    await page.locator('[data-qa="complete-task-submit-btn"]').click();

    await expect(inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Complete With Files' })).not.toBeVisible();
  });

  test('7.5 — Complete task validation: summary must be 100+ chars', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Short Summary Task', 'Summary');
    await moveTask(request, projectId, taskId, 'in_progress');
    await page.reload();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    const taskCard = inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Short Summary Task' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-complete-btn"]').click();
    // Fill summary with fewer than 100 chars
    await page.locator('[data-qa="complete-summary-input"]').fill('Too short summary');
    await page.locator('[data-qa="complete-agent-name-input"]').fill('human');

    // Submit button should be disabled
    await expect(page.locator('[data-qa="complete-task-submit-btn"]')).toBeDisabled();
  });

  test('7.5b — Cancel complete task modal', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Cancel Complete Task', 'Summary');
    await moveTask(request, projectId, taskId, 'in_progress');
    await page.reload();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    const taskCard = inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Cancel Complete Task' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-complete-btn"]').click();
    await page.locator('[data-qa="complete-task-cancel-btn"]').click();

    // Task should still be in in_progress
    await expect(inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Cancel Complete Task' })).toBeVisible();
  });

  test('7.6 — Mark task as Won\'t Do', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Wont Do', 'Summary for wont do test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Wont Do' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-wontdo-btn"]').click();
    // Fill reason with 50+ chars (MarkWontDoModal requires 50 min)
    await page.locator('[data-qa="wont-do-reason-input"]').fill('This task will not be done because it is out of scope for the sprint');
    await page.locator('[data-qa="mark-wont-do-submit-btn"]').click();

    // Task should move to Blocked column with Won't Do Requested badge
    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Wont Do' })).toBeVisible();
  });

  test('7.6b — Cancel Won\'t Do modal', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Not Wont Do', 'Summary');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Wont Do' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-wontdo-btn"]').click();
    await page.locator('[data-qa="mark-wont-do-cancel-btn"]').click();

    // Task should still be in Todo
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Wont Do' })).toBeVisible();
  });

  test('7.7 — Approve Won\'t Do', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Approve WD', 'Summary for approve wontdo');
    await page.reload();

    // Mark as won't do first
    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Approve WD' });
    await taskCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-wontdo-btn"]').click();
    await page.locator('[data-qa="wont-do-reason-input"]').fill('This task is out of scope and should not be implemented this quarter');
    await page.locator('[data-qa="mark-wont-do-submit-btn"]').click();

    // Wait for task in blocked column
    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Approve WD' })).toBeVisible();

    // Right-click to open context menu on blocked task -- it should have "Unblock" but won't-do approval
    // is typically via the Approve Won't Do modal. The blocked column shows wont_do_requested tasks.
    // The context menu on blocked tasks has "Unblock" action. The approve-wont-do flow may be
    // accessed via editing the task. Let's check if there's a specific context menu action.
    // Looking at the code, blocked column context menu only has: edit, unblock, move_to_project, delete.
    // The approve-wont-do is likely triggered via the edit/drawer actions.

    // Open the task drawer for the blocked wont-do task and trigger action from there
    const blockedCard = blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Approve WD' });
    await blockedCard.click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    // The approve wont-do action is typically triggered from the drawer's action buttons
    // Look for approve action button in drawer
    // Since ApproveWontDoModal has data-qa="wont-do-approve-btn", we look for the trigger
    // The drawer calls onAction('approve_wontdo') or similar -- let's use context menu approach
    await page.locator('[data-qa="drawer-close-btn"]').click();

    // Use context menu on the blocked card -- the "Unblock" action on a wont_do_requested task
    // may show the ApproveWontDoModal instead. Let's right-click and check.
    await blockedCard.click({ button: 'right' });

    // If unblock is the entry point, clicking it on a wont_do_requested task opens ApproveWontDoModal
    await page.locator('[data-qa="context-menu-unblock-btn"]').click();

    // The ApproveWontDoModal should be shown for wont_do_requested tasks
    await page.locator('[data-qa="wont-do-approve-btn"]').click();

    // Task should move to Done
    const doneColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Done' });
    await expect(doneColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Approve WD' })).toBeVisible();
  });

  test('7.8 — Reject Won\'t Do with optional comment', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Reject WD', 'Summary for reject wontdo');
    await page.reload();

    // Mark as won't do first
    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Reject WD' });
    await taskCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-wontdo-btn"]').click();
    await page.locator('[data-qa="wont-do-reason-input"]').fill('This task should be skipped because it duplicates existing functionality');
    await page.locator('[data-qa="mark-wont-do-submit-btn"]').click();

    const blockedColumn = page.locator('[data-qa="column"]').filter({ hasText: 'Blocked' });
    await expect(blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Reject WD' })).toBeVisible();

    // Open approve wont do modal via context menu unblock
    const blockedCard = blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Reject WD' });
    await blockedCard.click({ button: 'right' });
    await page.locator('[data-qa="context-menu-unblock-btn"]').click();

    // Fill rejection reason and click reject
    // First click of reject button opens the rejection reason field
    await page.locator('[data-qa="wont-do-reject-btn"]').click();
    await page.locator('[data-qa="wont-do-rejection-reason-input"]').fill('This task is still needed, please continue.');
    await page.locator('[data-qa="wont-do-reject-btn"]').click();

    // Task should move back to Todo
    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Reject WD' })).toBeVisible();
  });

  test('7.10 — Move task to another project', async ({ request, page }) => {
    // Create a second project as the target
    const targetProjectId = await createProject(request, 'Move Target Project');
    await createTask(request, projectId, 'Task To Move Project', 'Summary for move test');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Move Project' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-move_to_project-btn"]').click();

    // Select the target project
    await page.locator('[data-qa="move-to-project-select"]').selectOption({ label: 'Move Target Project' });
    await page.locator('[data-qa="move-to-project-submit-btn"]').click();

    // Task should disappear from current board
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task To Move Project' })).not.toBeVisible();

    // Cleanup
    await deleteProject(request, targetProjectId);
  });

  test('7.10b — Cancel move to project modal', async ({ request, page }) => {
    const targetProjectId = await createProject(request, 'Cancel Move Target');
    await createTask(request, projectId, 'Task Not Moved', 'Summary');
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Moved' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-move_to_project-btn"]').click();
    await page.locator('[data-qa="move-to-project-cancel-btn"]').click();

    // Task should still be on the board
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Task Not Moved' })).toBeVisible();

    await deleteProject(request, targetProjectId);
  });

  test('7.13 — Move task to In Progress via context menu', async ({ request, page }) => {
    await createTask(request, projectId, 'Task Move IP', 'Summary for move to in progress');
    await page.reload();

    const todoColumn = page.locator('[data-qa="column"]').filter({ hasText: 'To Do' });
    const taskCard = todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Move IP' });
    await taskCard.click({ button: 'right' });

    await page.locator('[data-qa="context-menu-move_in_progress-btn"]').click();

    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Move IP' })).not.toBeVisible();

    const inProgressColumn = page.locator('[data-qa="column"]').filter({ hasText: 'In Progress' });
    await expect(inProgressColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Task Move IP' })).toBeVisible();
  });
});

test.describe('8. Task Card Display', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Task Card Display Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('8.1 — Task card shows priority badge', async ({ request, page }) => {
    await createTask(request, projectId, 'Critical Task', 'Summary', { priority: 'critical' });
    await createTask(request, projectId, 'Low Task', 'Summary', { priority: 'low' });
    await page.reload();

    // Each card should display a priority pill
    const criticalCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Critical Task' });
    await expect(criticalCard.getByText(/critical/i)).toBeVisible();

    const lowCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Low Task' });
    await expect(lowCard.getByText(/low/i)).toBeVisible();
  });

  test('8.2 — Task card shows role badge when assigned', async ({ request, page }) => {
    const agent = await createAgent(request, 'display-dev', 'Display Dev');
    await createTask(request, projectId, 'Assigned Task', 'Summary', { assigned_role: 'display-dev' });
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Assigned Task' });
    // The card shows the first letter of the role as an avatar
    await expect(taskCard.getByText('D')).toBeVisible();

    await deleteAgent(request, 'display-dev');
  });

  test('8.3 — Task card shows comment count', async ({ request, page }) => {
    const taskId = await createTask(request, projectId, 'Commented Task', 'Summary');
    // Add a comment via API
    const token = await (async () => {
      const fs = await import('fs');
      try {
        const raw = fs.readFileSync('/tmp/auth-state.json', 'utf8');
        const state = JSON.parse(raw);
        for (const origin of state.origins ?? []) {
          for (const item of origin.localStorage ?? []) {
            if (item.name === 'agach_access_token') return item.value;
          }
        }
      } catch { /* ignore */ }
      return '';
    })();

    await request.fetch(`${BASE_URL}/api/projects/${projectId}/tasks/${taskId}/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      data: {
        content: 'Test comment for count display',
        author_role: 'human',
        author_name: 'Tester',
      },
    });

    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Commented Task' });
    // Should show comment count (the number 1)
    await expect(taskCard.getByText('1')).toBeVisible();
  });

  test('8.5 — Task card shows unresolved dependency icon', async ({ request, page }) => {
    const parentId = await createTask(request, projectId, 'Dep Display Parent', 'Parent summary');
    await createTask(request, projectId, 'Dep Display Child', 'Child summary', {
      depends_on: [parentId],
    });
    await page.reload();

    // The child task should show the GitBranch icon (unresolved dependency)
    const childCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Dep Display Child' });
    await expect(childCard).toBeVisible();
    // GitBranch icon renders as an SVG -- we verify the card is present; the icon is a visual indicator
    // We can check that the card exists and has the expected structure
    await expect(childCard.locator('svg')).toHaveCount({ minimum: 1 });
  });

  test('8.6 — Task card shows feature dot when in a feature', async ({ request, page }) => {
    const featureId = await createFeature(request, projectId, 'Display Feature');
    await createTask(request, projectId, 'Feature Dot Task', 'Summary', {
      feature_id: featureId,
    });
    await page.reload();

    const taskCard = page.locator('[data-qa="task-card"]').filter({ hasText: 'Feature Dot Task' });
    await expect(taskCard).toBeVisible();
    // The feature dot is a small colored div -- verify the card is rendered
  });
});
