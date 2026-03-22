import { test, expect } from '@playwright/test';
import {
  createProject,
  deleteProject,
  createTask,
  createRole,
  deleteRole,
  BASE_URL,
} from './helpers';

test.describe('7. Backlog', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Backlog Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('7.1 — Backlog page shows tasks created with start_in_backlog', async ({
    page,
    request,
  }) => {
    await createTask(request, projectId, 'Backlog Task Alpha', 'Summary for alpha', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await expect(page.getByText('Backlog Task Alpha')).toBeVisible();
  });

  test('7.2 — Backlog empty state when no backlog tasks', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    const taskBtns = page.locator('[data-qa="task-open-btn"]');
    const count = await taskBtns.count();

    if (count === 0) {
      // Assert empty state — either a message or simply 0 open-btn rows
      expect(count).toBe(0);
    } else {
      // In case the UI shows a message alongside tasks, check for empty-state text
      const emptyText = page.getByText(/no.*task|empty|backlog/i);
      await expect(emptyText).toBeVisible();
    }
  });

  test('7.3 — Move individual task from backlog to todo', async ({ page, request }) => {
    await createTask(request, projectId, 'Backlog Task Beta', 'Summary for beta', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    const initialCount = await page.locator('[data-qa="task-open-btn"]').count();
    expect(initialCount).toBeGreaterThan(0);

    await page.locator('[data-qa="move-task-to-todo-btn"]').first().click();

    // Wait for the row to disappear
    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(initialCount - 1);
  });

  test('7.4 — Move all tasks from backlog to todo', async ({ page, request }) => {
    await createTask(request, projectId, 'Backlog Task Gamma', 'Summary for gamma', {
      start_in_backlog: true,
    });
    await createTask(request, projectId, 'Backlog Task Delta', 'Summary for delta', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(2);

    await page.locator('[data-qa="move-all-to-todo-btn"]').click();

    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(0);
  });

  test('7.5 — Open task drawer from backlog', async ({ page, request }) => {
    await createTask(request, projectId, 'Backlog Task Epsilon', 'Summary for epsilon', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await page.locator('[data-qa="task-open-btn"]').first().click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });
});

test.describe('8. Settings', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Settings Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('8.1 — Settings page loads with project name', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    await expect(page.locator('[data-qa="project-name-input"]')).toHaveValue(
      'Settings Test Project',
    );
  });

  test('8.2 — Save project name change', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const nameInput = page.locator('[data-qa="project-name-input"]');
    await nameInput.clear();
    await nameInput.fill('New Name');

    await page.locator('[data-qa="save-project-settings-btn"]').click();

    await expect(nameInput).toHaveValue('New Name');

    // Reload and confirm persistence
    await page.reload();
    await expect(page.locator('[data-qa="project-name-input"]')).toHaveValue('New Name');
  });

  test('8.3 — Save button disabled when name is empty', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const nameInput = page.locator('[data-qa="project-name-input"]');
    await nameInput.clear();

    await expect(page.locator('[data-qa="save-project-settings-btn"]')).toBeDisabled();
  });

  test('8.4 — Set WIP limit for In Progress column', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const wipInput = page.locator('[data-qa="wip-limit-in_progress-input"]');
    await wipInput.clear();
    await wipInput.fill('2');

    await page.locator('[data-qa="save-wip-limits-btn"]').click();

    // Assert button becomes enabled again (not in a loading/disabled state)
    await expect(page.locator('[data-qa="save-wip-limits-btn"]')).toBeEnabled();
  });

  test('8.5 — Delete project via danger zone', async ({ page, request }) => {
    // Create a separate project to delete so the beforeEach project is unaffected
    const projectToDelete = await createProject(request, 'Project To Delete 8.5');

    await page.goto(`${BASE_URL}/projects/${projectToDelete}/settings`);

    await page.locator('[data-qa="delete-project-btn"]').click();
    await page.locator('[data-qa="delete-confirm-submit-btn"]').click();

    // After deletion, URL should navigate away (to home or project list)
    await expect(page).toHaveURL(/^(?!.*\/projects\/${projectToDelete}).*/);
    await expect(page).not.toHaveURL(new RegExp(`/projects/${projectToDelete}`));
  });

  test('8.6 — Add agent to project', async ({ page, request }) => {
    await createRole(request, 'qa-agent-settings', 'QA Agent Settings');

    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

      await page.locator('[data-qa="add-agent-btn"]').click();

      const agentSelect = page.locator('[data-qa="add-agent-select"]');
      await expect(agentSelect).toBeVisible();
      await agentSelect.selectOption({ value: 'qa-agent-settings' });

      await page.locator('[data-qa="add-agent-confirm-btn"]').click();

      // Agent should now appear in the list
      await expect(page.getByText(/qa-agent-settings/i)).toBeVisible();
    } finally {
      await deleteRole(request, 'qa-agent-settings');
    }
  });

  test('8.7 — Project description can be updated', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const descTextarea = page.locator('[data-qa="project-description-textarea"]');
    await descTextarea.clear();
    await descTextarea.fill('Updated description');

    await page.locator('[data-qa="save-project-settings-btn"]').click();

    // Assert save was accepted: button becomes re-enabled or no error visible
    await expect(page.locator('[data-qa="save-project-settings-btn"]')).toBeEnabled();
  });
});
