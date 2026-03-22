import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createTask, moveTask, BASE_URL } from './helpers';

test.describe('3. Kanban Board', () => {
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

  test('3.1 — Board displays columns (Todo, In Progress, Done, Blocked)', async ({ page }) => {
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

  test('3.2 — Task card appears in the correct column', async ({ page, request }) => {
    await createTask(request, projectId, 'My First Task', 'Summary of my first task');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const columns = page.locator('[data-qa="column"]');
    const todoColumn = columns.first();

    await expect(todoColumn.locator('[data-qa="task-card"]').filter({ hasText: 'My First Task' })).toBeVisible();
  });

  test('3.3 — "New Task" button opens the new task modal', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.click('[data-qa="new-task-btn"]');

    const modal = page.locator('[data-qa="new-task-modal"], [data-qa="new-task-title-input"]');
    await expect(modal.first()).toBeVisible();
  });

  test('3.4 — Search filters tasks', async ({ page, request }) => {
    await createTask(request, projectId, 'Unique Alpha Task', 'Summary for alpha task');
    await createTask(request, projectId, 'Beta Task', 'Summary for beta task');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.fill('[data-qa="search-input"]', 'Unique Alpha');

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Unique Alpha Task' })).toBeVisible();
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: 'Beta Task' })).not.toBeVisible();
  });

  test('3.5 — Clear search button resets filter', async ({ page, request }) => {
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

  test('3.6 — Done filter select is visible and has options', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    const doneFilter = page.locator('[data-qa="done-filter-select"]');
    await expect(doneFilter).toBeVisible();

    const options = doneFilter.locator('option');
    const optionCount = await options.count();
    expect(optionCount).toBeGreaterThan(1);
  });

  test('3.7 — Task click opens task drawer', async ({ page, request }) => {
    await createTask(request, projectId, 'Drawer Test Task', 'Summary for drawer test');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="task-card"]').first().click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });

  test('3.8 — Closing task drawer via button', async ({ page, request }) => {
    await createTask(request, projectId, 'Drawer Close Task', 'Summary for drawer close test');

    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();

    await page.click('[data-qa="drawer-close-btn"]');

    await expect(page.locator('[data-qa="drawer-close-btn"]')).not.toBeVisible();
  });

  test('3.9 — Keyboard shortcut "/" focuses search', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');

    await page.keyboard.press('/');

    const searchInput = page.locator('[data-qa="search-input"]');
    await expect(searchInput).toBeFocused();
  });
});
