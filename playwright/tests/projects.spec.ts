import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiDelete, uniqueSlug } from './helpers';

test.describe('Projects', () => {
  test('create project via UI and see it on home page', async ({ page, request }) => {
    await uiLogin(page);

    // Click "New Project"
    await page.locator('[data-qa="create-project-btn"]').click();
    await expect(page.locator('[data-qa="create-project-name-input"]')).toBeVisible();

    const projectName = 'PW Test Project ' + Date.now();
    await page.locator('[data-qa="create-project-name-input"]').fill(projectName);
    await page.locator('[data-qa="create-project-description-input"]').fill('Created by Playwright');
    await page.locator('[data-qa="create-project-submit-btn"]').click();

    // Should navigate to the new project's board
    await page.waitForURL(/\/projects\/.*\/board/, { timeout: 10_000 });

    // Go back to home and verify project is listed
    await page.goto('/');
    await expect(page.locator('[data-qa="project-card"]').filter({ hasText: projectName })).toBeVisible();
  });

  test('project card opens kanban board', async ({ page }) => {
    await uiLogin(page);
    // Click the first project card's Open button
    const firstOpen = page.locator('[data-qa="project-open-btn"]').first();
    await expect(firstOpen).toBeVisible();
    await firstOpen.click();
    await page.waitForURL(/\/projects\/.+\/board/);
    // Board should have columns
    await expect(page.locator('[data-qa="column"]').first()).toBeVisible();
  });

  test('kanban board shows columns with correct slugs', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="project-open-btn"]').first().click();
    await page.waitForURL(/\/projects\/.+\/board/);

    const columns = page.locator('[data-qa="column"]');
    await expect(columns).toHaveCount(4, { timeout: 10_000 }); // todo, in_progress, blocked, done (backlog hidden when empty)

    const titles = await page.locator('[data-qa="column-title"]').allInnerTexts();
    const normalized = titles.map(t => t.toLowerCase());
    expect(normalized).toContain('to do');
    expect(normalized).toContain('in progress');
    expect(normalized).toContain('done');
    expect(normalized).toContain('blocked');
  });

  test('project settings page loads and shows project name', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="project-open-btn"]').first().click();
    await page.waitForURL(/\/projects\/.+\/board/);

    // Navigate to settings
    await page.locator('[data-qa="nav-settings-btn"]').click();
    await page.waitForURL(/\/projects\/.+\/settings/);
    await expect(page.locator('[data-qa="project-name-input"]')).toBeVisible();
  });

  test('update project settings', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="project-open-btn"]').first().click();
    await page.waitForURL(/\/projects\/.+\/board/);

    await page.locator('[data-qa="nav-settings-btn"]').click();
    await page.waitForURL(/\/projects\/.+\/settings/);

    const nameInput = page.locator('[data-qa="project-name-input"]');
    const original = await nameInput.inputValue();
    await nameInput.fill('PW Updated Project');
    await page.locator('[data-qa="save-project-settings-btn"]').click();

    // Reload and verify
    await page.reload();
    await expect(nameInput).toHaveValue('PW Updated Project');

    // Restore
    await nameInput.fill(original);
    await page.locator('[data-qa="save-project-settings-btn"]').click();
  });

  test('sub-projects via API', async ({ request }) => {
    const token = await apiLogin(request);
    const parent = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Parent ' + Date.now(),
      description: 'parent',
    });
    const child = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Child ' + Date.now(),
      description: 'child',
      parent_id: parent.id,
    });

    const children = await apiGet<any[]>(request, `/api/projects/${parent.id}/children`, token);
    expect(children.some((c: any) => c.id === child.id)).toBeTruthy();

    // Cleanup
    await apiDelete(request, `/api/projects/${child.id}`, token);
    await apiDelete(request, `/api/projects/${parent.id}`, token);
  });

  test('project board via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Board ' + Date.now(),
    });

    // Create a task
    await apiPost(request, `/api/projects/${proj.id}/tasks`, token, {
      title: 'Board task',
      summary: 'Board task summary',
      priority: 'high',
    });

    const board = await apiGet<{ columns: any[] }>(request, `/api/projects/${proj.id}/board`, token);
    expect(board.columns.length).toBeGreaterThanOrEqual(4);

    const slugs = board.columns.map((c: any) => c.slug);
    expect(slugs).toContain('todo');
    expect(slugs).toContain('in_progress');
    expect(slugs).toContain('done');
    expect(slugs).toContain('blocked');

    const totalTasks = board.columns.reduce((sum: number, c: any) => sum + (c.tasks?.length || 0), 0);
    expect(totalTasks).toBeGreaterThanOrEqual(1);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('project summary via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Summary ' + Date.now(),
    });

    for (let i = 0; i < 3; i++) {
      await apiPost(request, `/api/projects/${proj.id}/tasks`, token, {
        title: `Summary task ${i}`,
        summary: `Summary for task ${i}`,
        priority: 'medium',
      });
    }

    const summary = await apiGet<any>(request, `/api/projects/${proj.id}/summary`, token);
    const total = summary.backlog_count + summary.todo_count + summary.in_progress_count + summary.done_count + summary.blocked_count;
    expect(total).toBeGreaterThanOrEqual(3);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('delete project via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Delete ' + Date.now(),
    });

    await apiDelete(request, `/api/projects/${proj.id}`, token);

    const list = await apiGet<any[]>(request, '/api/projects', token);
    expect(list.every((p: any) => p.id !== proj.id)).toBeTruthy();
  });
});
