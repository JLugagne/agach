import { test, expect } from '@playwright/test';
import {
  createProject,
  deleteProject,
  createTask,
  moveTask,
  blockTask,
  requestWontDo,
  createAgent,
  deleteAgent,
  assignAgentToProject,
  createFeature,
  BASE_URL,
} from './helpers';

test.describe('4. Kanban Board', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Kanban Board Test Project');
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } finally {
      // ensure cleanup attempt is made
    }
  });

  test('4.1 — Board displays columns (Todo, In Progress, Done, Blocked)', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const columns = page.locator('[data-qa="column"]');
    await expect(columns).toHaveCount(4);

    const columnTitles = await page.locator('[data-qa="column-title"]').allTextContents();
    expect(columnTitles).toContain('Todo');
    expect(columnTitles).toContain('In Progress');
    expect(columnTitles).toContain('Done');
    expect(columnTitles).toContain('Blocked');
  });

  test('4.2 — Task cards appear in the correct column', async ({ page, request }) => {
    await createTask(request, projectId, 'My First Task', 'Summary of my first task');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const columns = page.locator('[data-qa="column"]');
    const todoColumn = columns.first();

    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'My First Task' })).toBeVisible();
  });

  test('4.3 — Column displays task count badge', async ({ page, request }) => {
    await createTask(request, projectId, 'Count Task 1', 'Summary 1');
    await createTask(request, projectId, 'Count Task 2', 'Summary 2');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // The Todo column header should show a count of 2
    const columns = page.locator('[data-qa="column"]');
    const todoColumn = columns.first();
    // The count badge is a sibling of the column-title within the header
    const todoHeader = todoColumn.locator('[data-qa="column-title"]').locator('..');
    await expect(todoHeader).toContainText('2');
  });

  test('4.4 — WIP limit warning appears when column is at capacity', async ({ page, request }) => {
    // In Progress has WIP limit of 3. Create 3 tasks and move them to in_progress.
    const taskId1 = await createTask(request, projectId, 'WIP Task 1', 'Summary 1');
    const taskId2 = await createTask(request, projectId, 'WIP Task 2', 'Summary 2');
    const taskId3 = await createTask(request, projectId, 'WIP Task 3', 'Summary 3');

    await moveTask(request, projectId, taskId1, 'in_progress');
    await moveTask(request, projectId, taskId2, 'in_progress');
    await moveTask(request, projectId, taskId3, 'in_progress');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // Find the In Progress column
    const inProgressColumn = page.locator('[data-qa="column"]').filter({
      has: page.locator('[data-qa="column-title"]', { hasText: 'In Progress' }),
    });

    // The count should show "3/3"
    await expect(inProgressColumn).toContainText('3/3');
  });

  test('4.6 — Click task card opens the Task Drawer', async ({ page, request }) => {
    await createTask(request, projectId, 'Drawer Test Task', 'Summary for drawer test');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="task-card"]').first().click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });

  test('4.7 — Right-click (context menu) on a task card', async ({ page, request }) => {
    await createTask(request, projectId, 'Context Menu Task', 'Summary for context menu test');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const taskCard = page.locator('[data-qa="task-card"]').first();
    await taskCard.click({ button: 'right' });

    // Context menu should appear with action buttons
    await expect(page.locator('[data-qa="context-menu-edit-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="context-menu-block-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="context-menu-delete-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="context-menu-move_to_project-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="context-menu-duplicate-btn"]')).toBeVisible();
    // Priority options
    await expect(page.locator('[data-qa="context-menu-priority_critical-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="context-menu-priority_high-btn"]')).toBeVisible();
  });

  test('4.8 — Board search filters visible tasks', async ({ page, request }) => {
    await createTask(request, projectId, 'Unique Alpha Task', 'Summary for alpha task');
    await createTask(request, projectId, 'Beta Task', 'Summary for beta task');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.fill('[data-qa="search-input"]', 'Unique Alpha');

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unique Alpha Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Beta Task' })).not.toBeVisible();
  });

  test('4.9 — Clear search button resets filter', async ({ page, request }) => {
    await createTask(request, projectId, 'Unique Alpha Task', 'Summary for alpha task');
    await createTask(request, projectId, 'Beta Task', 'Summary for beta task');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.fill('[data-qa="search-input"]', 'Unique Alpha');
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Beta Task' })).not.toBeVisible();

    await page.click('[data-qa="search-clear-btn"]');

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unique Alpha Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Beta Task' })).toBeVisible();
  });

  test('4.10 — "New Task" button opens the create task modal', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.click('[data-qa="new-task-btn"]');

    await expect(page.locator('[data-qa="new-task-modal"]')).toBeVisible();
  });

  test('4.11 — Keyboard shortcut "/" focuses search', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.keyboard.press('/');

    const searchInput = page.locator('[data-qa="search-input"]');
    await expect(searchInput).toBeFocused();
  });

  test('4.12 — Keyboard shortcut "?" opens shortcuts help overlay', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.keyboard.press('?');

    // The shortcuts overlay should appear
    await expect(page.locator('[data-qa="kanban-shortcuts-close-btn"]')).toBeVisible();

    // Close it
    await page.click('[data-qa="kanban-shortcuts-close-btn"]');
    await expect(page.locator('[data-qa="kanban-shortcuts-close-btn"]')).not.toBeVisible();
  });

  test('4.13 — Blocked banner with unblock button', async ({ page, request }) => {
    const taskId = await createTask(request, projectId, 'Blocked Banner Task', 'Summary for blocked banner test');
    await blockTask(request, projectId, taskId, 'Needs human review', 'test-agent');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // Click on the blocked task to open the drawer
    const blockedColumn = page.locator('[data-qa="column"]').filter({
      has: page.locator('[data-qa="column-title"]', { hasText: 'Blocked' }),
    });
    await blockedColumn.locator('[data-qa="task-card"]').first().click();

    // The blocked banner with unblock button should be visible in the drawer
    await expect(page.locator('[data-qa="blocked-banner-unblock-btn"]')).toBeVisible();
  });

  test('4.14 — Done column filter dropdown', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const doneFilter = page.locator('[data-qa="done-filter-select"]');
    await expect(doneFilter).toBeVisible();

    // Verify it has multiple options
    const options = doneFilter.locator('option');
    const optionCount = await options.count();
    expect(optionCount).toBeGreaterThan(1);

    // Select a specific time range
    await doneFilter.selectOption('24h');
    await expect(doneFilter).toHaveValue('24h');
  });

  test('4.17 — Won\'t-do-requested badge is shown on blocked tasks', async ({ page, request }) => {
    const taskId = await createTask(request, projectId, 'Wont Do Badge Task', 'Summary for wont do test');
    await requestWontDo(request, projectId, taskId, 'Not needed anymore', 'test-agent');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // Find the task card in the Blocked column - it should have the Won't Do Requested badge
    const blockedColumn = page.locator('[data-qa="column"]').filter({
      has: page.locator('[data-qa="column-title"]', { hasText: 'Blocked' }),
    });
    const taskCard = blockedColumn.locator('[data-qa="task-card"]').filter({ hasText: 'Wont Do Badge Task' });
    await expect(taskCard).toBeVisible();
    await expect(taskCard).toContainText("Won't Do Requested");
  });

  test('4.18 — Parent project link', async ({ page, request }) => {
    // Create a sub-project (feature) under the main project
    const featureId = await createFeature(request, projectId, 'Sub-Feature');

    await page.goto(`${BASE_URL}/projects/${featureId}`);
    await page.waitForLoadState('networkidle');

    // The parent project link should be visible
    const parentLink = page.locator('[data-qa="kanban-parent-project-link"]');
    await expect(parentLink).toBeVisible();
    await expect(parentLink).toContainText('Kanban Board Test Project');

    // Clean up feature
    try {
      await deleteProject(request, featureId);
    } catch {
      // ignore - parent cleanup will cascade
    }
  });

  test('4.19 — Bulk select tasks with Ctrl+click', async ({ page, request }) => {
    await createTask(request, projectId, 'Bulk Task A', 'Summary A');
    await createTask(request, projectId, 'Bulk Task B', 'Summary B');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const cards = page.locator('[data-qa="task-card"]');
    await expect(cards).toHaveCount(2);

    // Ctrl+click first task
    await cards.first().click({ modifiers: ['Control'] });
    // Ctrl+click second task
    await cards.nth(1).click({ modifiers: ['Control'] });

    // Bulk actions bar should appear
    await expect(page.locator('[data-qa="bulk-cancel-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="bulk-delete-btn"]')).toBeVisible();
  });

  test('4.20 — Bulk actions bar: context-aware buttons for todo tasks', async ({ page, request }) => {
    await createTask(request, projectId, 'Bulk Todo A', 'Summary A');
    await createTask(request, projectId, 'Bulk Todo B', 'Summary B');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const cards = page.locator('[data-qa="task-card"]');

    // Select both todo tasks with Ctrl+click
    await cards.first().click({ modifiers: ['Control'] });
    await cards.nth(1).click({ modifiers: ['Control'] });

    // For todo tasks, the bar should show: Move to In Progress, Block, Delete
    await expect(page.locator('[data-qa="bulk-move-in-progress-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="bulk-block-btn"]')).toBeVisible();
    await expect(page.locator('[data-qa="bulk-delete-btn"]')).toBeVisible();
  });

  test('4.21 — Bulk actions: cancel', async ({ page, request }) => {
    await createTask(request, projectId, 'Cancel Bulk A', 'Summary A');
    await createTask(request, projectId, 'Cancel Bulk B', 'Summary B');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const cards = page.locator('[data-qa="task-card"]');

    // Select tasks
    await cards.first().click({ modifiers: ['Control'] });
    await cards.nth(1).click({ modifiers: ['Control'] });

    await expect(page.locator('[data-qa="bulk-cancel-btn"]')).toBeVisible();

    // Click cancel
    await page.click('[data-qa="bulk-cancel-btn"]');

    // Bulk bar should disappear
    await expect(page.locator('[data-qa="bulk-cancel-btn"]')).not.toBeVisible();
    await expect(page.locator('[data-qa="bulk-delete-btn"]')).not.toBeVisible();
  });

  test('4.15 — Toggle sub-projects visibility', async ({ page, request }) => {
    // Create a feature (sub-project) with a task
    const featureId = await createFeature(request, projectId, 'Test Feature');
    await createTask(request, featureId, 'Feature Task', 'Task inside feature');
    // Also create a task in the parent project
    await createTask(request, projectId, 'Parent Task', 'Task in parent project');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // Both tasks should be visible initially (include_children is true by default)
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Parent Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Feature Task' })).toBeVisible();

    // Toggle sub-projects off
    const toggleBtn = page.locator('[data-qa="kanban-toggle-subprojects-btn"]');
    await expect(toggleBtn).toBeVisible();
    await toggleBtn.click();

    // Feature task should be hidden, parent task still visible
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Parent Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Feature Task' })).not.toBeVisible();

    // Clean up
    try {
      await deleteProject(request, featureId);
    } catch {
      // parent cleanup will cascade
    }
  });

  test('4.16 — Role filter buttons', async ({ page, request }) => {
    // Create an agent and assign it to the project
    const agentSlug = `test-role-${Date.now()}`;
    await createAgent(request, agentSlug, 'Test Role', { color: '#FF0000' });
    await assignAgentToProject(request, projectId, agentSlug);

    // Create tasks - one assigned to the role, one unassigned
    await createTask(request, projectId, 'Assigned Task', 'Task with role', { assigned_role: agentSlug });
    await createTask(request, projectId, 'Unassigned Task', 'Task without role');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    // Both tasks should be visible initially
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Assigned Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unassigned Task' })).toBeVisible();

    // Click the role filter button
    const roleFilterBtn = page.locator('[data-qa="kanban-role-filter-btn"]').first();
    await expect(roleFilterBtn).toBeVisible();
    await roleFilterBtn.click();

    // Only the assigned task should remain visible
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Assigned Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unassigned Task' })).not.toBeVisible();

    // Clear filters
    await page.click('[data-qa="kanban-clear-filters-btn"]');

    // Both tasks should be visible again
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Assigned Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unassigned Task' })).toBeVisible();

    // Clean up agent
    try {
      await deleteAgent(request, agentSlug);
    } catch {
      // ignore
    }
  });

  test('Closing task drawer via button', async ({ page, request }) => {
    await createTask(request, projectId, 'Drawer Close Task', 'Summary for drawer close test');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.click('[data-qa="drawer-close-btn"]');

    await expect(page.locator('[data-qa="drawer-close-btn"]')).not.toBeVisible();
  });
});
