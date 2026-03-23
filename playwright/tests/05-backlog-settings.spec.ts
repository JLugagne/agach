import { test, expect } from '@playwright/test';
import {
  createProject,
  deleteProject,
  createTask,
  createFeature,
  createAgent,
  deleteAgent,
  assignAgentToProject,
  BASE_URL,
} from './helpers';

test.describe('11. Backlog', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Backlog Test Project');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('11.1 — Backlog page shows tasks in the backlog column', async ({
    page,
    request,
  }) => {
    await createTask(request, projectId, 'Backlog Task Alpha', 'Summary for alpha', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    const row = page.locator('[data-qa="task-open-btn"]').first();
    await expect(row).toBeVisible();
    await expect(page.getByText('Backlog Task Alpha')).toBeVisible();
    await expect(page.getByText('Summary for alpha')).toBeVisible();
  });

  test('11.2 — Empty backlog shows empty state', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await expect(page.getByText(/no tasks in backlog/i)).toBeVisible();
    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(0);
  });

  test('11.3 — Move individual task from backlog to Todo', async ({ page, request }) => {
    await createTask(request, projectId, 'Backlog Task Beta', 'Summary for beta', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(1);

    await page.locator('[data-qa="move-task-to-todo-btn"]').first().click();

    // Task should disappear from backlog
    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(0);
  });

  test('11.4 — Move all tasks from backlog to Todo', async ({ page, request }) => {
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

  test('11.5 — Filter backlog by feature', async ({ page, request }) => {
    // Create two features
    const featureA = await createFeature(request, projectId, 'Feature A');
    const featureB = await createFeature(request, projectId, 'Feature B');

    // Create backlog tasks in each feature
    await createTask(request, featureA, 'Task in Feature A', 'Summary A', {
      start_in_backlog: true,
    });
    await createTask(request, featureB, 'Task in Feature B', 'Summary B', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    // Initially both tasks are visible
    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(2);

    // Select Feature A from filter
    const filterSelect = page.locator('[data-qa="feature-filter-select"]');
    await expect(filterSelect).toBeVisible();
    await filterSelect.selectOption({ value: featureA });

    // Only Feature A task should be visible
    await expect(page.locator('[data-qa="task-open-btn"]')).toHaveCount(1);
    await expect(page.getByText('Task in Feature A')).toBeVisible();
    await expect(page.getByText('Task in Feature B')).not.toBeVisible();
  });

  test('11.6 — Open task drawer from backlog', async ({ page, request }) => {
    await createTask(request, projectId, 'Backlog Task Epsilon', 'Summary for epsilon', {
      start_in_backlog: true,
    });

    await page.goto(`${BASE_URL}/projects/${projectId}/backlog`);

    await page.locator('[data-qa="task-open-btn"]').first().click();

    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });
});

test.describe('12. Project Settings', () => {
  let projectId: string;

  test.beforeEach(async ({ request }) => {
    projectId = await createProject(request, 'Settings Test Project', 'Initial description');
  });

  test.afterEach(async ({ request }) => {
    await deleteProject(request, projectId);
  });

  test('12.1 — Settings page displays project fields', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    await expect(page.locator('[data-qa="project-name-input"]')).toHaveValue(
      'Settings Test Project',
    );
    await expect(page.locator('[data-qa="project-description-textarea"]')).toHaveValue(
      'Initial description',
    );
    await expect(page.locator('[data-qa="project-git-url-input"]')).toBeVisible();
  });

  test('12.2 — Update project name', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const nameInput = page.locator('[data-qa="project-name-input"]');
    await nameInput.clear();
    await nameInput.fill('Renamed Project');

    await page.locator('[data-qa="save-project-settings-btn"]').click();

    // Button shows "Saved" momentarily
    await expect(page.locator('[data-qa="save-project-settings-btn"]')).toContainText('Saved');

    // Reload and confirm persistence
    await page.reload();
    await expect(page.locator('[data-qa="project-name-input"]')).toHaveValue('Renamed Project');
  });

  test('12.3 — Update project description', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const descTextarea = page.locator('[data-qa="project-description-textarea"]');
    await descTextarea.clear();
    await descTextarea.fill('Updated description');

    await page.locator('[data-qa="save-project-settings-btn"]').click();

    // Wait for save to complete
    await expect(page.locator('[data-qa="save-project-settings-btn"]')).toBeEnabled();

    // Reload and confirm persistence
    await page.reload();
    await expect(page.locator('[data-qa="project-description-textarea"]')).toHaveValue(
      'Updated description',
    );
  });

  test('12.4 — Save button is disabled when name is empty', async ({ page }) => {
    await page.goto(`${BASE_URL}/projects/${projectId}/settings`);

    const nameInput = page.locator('[data-qa="project-name-input"]');
    await nameInput.clear();

    await expect(page.locator('[data-qa="save-project-settings-btn"]')).toBeDisabled();
  });

  test('12.6 — Project Agents section: list assigned agents', async ({ page, request }) => {
    // Create and assign an agent
    await createAgent(request, 'qa-agent-list', 'QA Agent List');
    try {
      await assignAgentToProject(request, projectId, 'qa-agent-list');

      await page.goto(`${BASE_URL}/projects/${projectId}/settings/agents`);

      // Agent should be listed with name and slug
      await expect(page.getByText('QA Agent List')).toBeVisible();
      await expect(page.getByText('qa-agent-list')).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-agent-list');
    }
  });

  test('12.7 — Project Agents section: add an agent', async ({ page, request }) => {
    await createAgent(request, 'qa-agent-add', 'QA Agent Add');

    try {
      await page.goto(`${BASE_URL}/projects/${projectId}/settings/agents`);

      await page.locator('[data-qa="add-agent-btn"]').click();

      const agentSelect = page.locator('[data-qa="add-agent-select"]');
      await expect(agentSelect).toBeVisible();
      await agentSelect.selectOption({ value: 'qa-agent-add' });

      await page.locator('[data-qa="add-agent-confirm-btn"]').click();

      // Agent should now appear in the list
      await expect(page.getByText('QA Agent Add')).toBeVisible();
      await expect(page.getByText('qa-agent-add')).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-agent-add');
    }
  });

  test('12.8 — Project Agents section: set default agent', async ({ page, request }) => {
    await createAgent(request, 'qa-agent-default', 'QA Agent Default');
    try {
      await assignAgentToProject(request, projectId, 'qa-agent-default');

      await page.goto(`${BASE_URL}/projects/${projectId}/settings/agents`);

      // Click set-default button
      await page.locator('[data-qa="set-default-agent-btn"]').first().click();

      // Agent should be marked as default
      await expect(page.getByText('default')).toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-agent-default');
    }
  });

  test('12.9 — Project Agents section: remove an agent', async ({ page, request }) => {
    await createAgent(request, 'qa-agent-remove', 'QA Agent Remove');
    try {
      await assignAgentToProject(request, projectId, 'qa-agent-remove');

      await page.goto(`${BASE_URL}/projects/${projectId}/settings/agents`);

      await expect(page.getByText('QA Agent Remove')).toBeVisible();

      // Click remove button
      await page.locator('[data-qa="remove-agent-btn"]').first().click();

      // Choose "clear assignments" option
      await page.locator('[data-qa="remove-agent-clear-radio"]').click();

      // Confirm removal
      await page.locator('[data-qa="remove-agent-confirm-btn"]').click();

      // Agent should no longer appear
      await expect(page.getByText('QA Agent Remove')).not.toBeVisible();
    } finally {
      await deleteAgent(request, 'qa-agent-remove').catch(() => {});
    }
  });

  test('12.10 — Danger Zone: delete project', async ({ page, request }) => {
    // Create a separate project to delete
    const projectToDelete = await createProject(request, 'Project To Delete 12.10');

    await page.goto(`${BASE_URL}/projects/${projectToDelete}/settings`);

    await page.locator('[data-qa="delete-project-btn"]').click();
    await page.locator('[data-qa="delete-confirm-submit-btn"]').click();

    // After deletion, user is redirected to home
    await expect(page).toHaveURL(/\/$/);
  });
});
