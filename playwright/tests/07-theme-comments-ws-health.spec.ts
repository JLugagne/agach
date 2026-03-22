import { test, expect } from '@playwright/test';
import { createProject, deleteProject, createTask, moveTask, BASE_URL } from './helpers';

// ─────────────────────────────────────────────
// Suite 12: Theme Toggle
// ─────────────────────────────────────────────

test.describe('12. Theme Toggle', () => {
  test('12.1 — Theme toggle button is visible', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    await expect(page.locator('[data-qa="theme-toggle-btn"]')).toBeVisible();
  });

  test('12.2 — Theme toggles from dark to light', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    const html = page.locator('html');
    const initialTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    await page.click('[data-qa="theme-toggle-btn"]');

    const newTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    expect(newTheme).not.toEqual(initialTheme);

    // Restore
    await page.click('[data-qa="theme-toggle-btn"]');

    const restoredTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));
    expect(restoredTheme).toEqual(initialTheme);
  });

  test('12.3 — Theme persists across navigation', async ({ page }) => {
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');

    const html = page.locator('html');
    const initialTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));

    await page.click('[data-qa="theme-toggle-btn"]');
    const switchedTheme =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));
    expect(switchedTheme).not.toEqual(initialTheme);

    await page.goto(`${BASE_URL}/roles`);
    await page.waitForLoadState('networkidle');

    const themeAfterNav =
      (await html.getAttribute('data-theme')) ?? (await html.getAttribute('class'));
    expect(themeAfterNav).toEqual(switchedTheme);

    // Restore
    await page.goto(`${BASE_URL}/`);
    await page.waitForLoadState('networkidle');
    await page.click('[data-qa="theme-toggle-btn"]');
  });
});

// ─────────────────────────────────────────────
// Suite 13: Comments
// ─────────────────────────────────────────────

test.describe('13. Comments', () => {
  let projectId: string;
  let taskId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'Comments Test Project');
    taskId = await createTask(request, projectId, 'Comment Test Task', 'Task for comment tests');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
    await page.locator('[data-qa="task-card"]').first().click();
    await expect(page.locator('[data-qa="drawer-close-btn"]')).toBeVisible();
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } finally {
      // ensure cleanup attempt is made
    }
  });

  test('13.1 — Comment input is visible in task drawer', async ({ page }) => {
    await expect(page.locator('[data-qa="comment-content-input"]')).toBeVisible();
  });

  test('13.2 — Submit a comment', async ({ page }) => {
    await page.locator('[data-qa="comment-content-input"]').fill('This is a test comment');
    await page.click('[data-qa="comment-submit-btn"]');
    await expect(page.getByText('This is a test comment')).toBeVisible();
  });

  test('13.3 — Submit button is disabled when comment is empty', async ({ page }) => {
    const input = page.locator('[data-qa="comment-content-input"]');
    await expect(input).toBeEmpty();
    await expect(page.locator('[data-qa="comment-submit-btn"]')).toBeDisabled();
  });

  test('13.4 — Comment author select is visible', async ({ page }) => {
    await expect(page.locator('[data-qa="comment-author-select"]')).toBeVisible();
  });

  test('13.5 — Ctrl+Enter submits comment', async ({ page }) => {
    await page.locator('[data-qa="comment-content-input"]').fill('Ctrl+Enter comment');
    await page.locator('[data-qa="comment-content-input"]').press('Control+Enter');
    await expect(page.getByText('Ctrl+Enter comment')).toBeVisible();
  });
});

// ─────────────────────────────────────────────
// Suite 14: Real-time WebSocket Updates
// ─────────────────────────────────────────────

test.describe('14. Real-time WebSocket Updates', () => {
  let projectId: string;

  test.beforeEach(async ({ request, page }) => {
    projectId = await createProject(request, 'WebSocket Test Project');
    await page.goto(`${BASE_URL}/projects/${projectId}`);
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ request }) => {
    try {
      await deleteProject(request, projectId);
    } finally {
      // ensure cleanup attempt is made
    }
  });

  test('14.1 — New task created via API appears on board without reload', async ({
    page,
    request,
  }) => {
    const title = 'WS Live Task ' + Date.now();
    await createTask(request, projectId, title, 'Real-time task summary');

    await page.waitForSelector('[data-qa="task-card"]', { timeout: 5000 });

    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: title })).toBeVisible();
  });

  test('14.2 — Task moved via API updates board column in real-time', async ({
    page,
    request,
  }) => {
    const title = 'WS Move Task ' + Date.now();
    const taskId = await createTask(request, projectId, title, 'Task to be moved');

    // Wait for initial render
    await page.waitForSelector('[data-qa="task-card"]', { timeout: 5000 });

    // Move via API
    await moveTask(request, projectId, taskId, 'in_progress');

    // Assert it shows up in In Progress column
    await page
      .locator('[data-qa="column"]')
      .filter({ hasText: 'In Progress' })
      .locator('[data-qa="task-card"]')
      .waitFor({ timeout: 5000 });

    await expect(
      page
        .locator('[data-qa="column"]')
        .filter({ hasText: 'In Progress' })
        .locator('[data-qa="task-card"]')
        .filter({ hasText: title }),
    ).toBeVisible();
  });
});

// ─────────────────────────────────────────────
// Suite 15: API Health Check
// ─────────────────────────────────────────────

test.describe('15. API Health Check', () => {
  test('15.1 — Health endpoint returns 200', async ({ request }) => {
    const response = await request.get(BASE_URL + '/health');
    expect(response.status()).toBe(200);
  });

  test('15.2 — Health endpoint returns non-empty body', async ({ request }) => {
    const response = await request.get(BASE_URL + '/health');
    const body = await response.text();
    expect(body.length).toBeGreaterThan(0);
  });

  test('15.3 — Projects API endpoint is reachable (authenticated)', async ({ request }) => {
    // All kanban API endpoints require authentication
    const projectId = await createProject(request, 'Health Check Test Project');
    expect(projectId.length).toBeGreaterThan(0);
    // Cleanup
    try {
      await deleteProject(request, projectId);
    } catch {
      // best-effort cleanup
    }
  });
});
