import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiPatch, apiDelete, uniqueSlug } from './helpers';

test.describe('Features', () => {
  test('feature CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Feature CRUD ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/features`;

    // Create
    const created = await apiPost<{ id: string; name: string; status: string; created_by_role: string }>(
      request, base, token, {
        name: 'Login Page', description: 'OAuth2 login page',
        created_by_role: 'architect', created_by_agent: 'claude',
      },
    );
    expect(created.id).toBeTruthy();
    expect(created.name).toBe('Login Page');
    expect(created.status).toBe('draft');
    expect(created.created_by_role).toBe('architect');

    // Read
    const got = await apiGet<{ id: string; name: string }>(request, `${base}/${created.id}`, token);
    expect(got.name).toBe('Login Page');

    // List
    const list = await apiGet<any[]>(request, base, token);
    expect(list.some((f: any) => f.id === created.id)).toBeTruthy();

    // Update
    await apiPatch(request, `${base}/${created.id}`, token, { name: 'Updated Login Page' });
    const updated = await apiGet<{ name: string }>(request, `${base}/${created.id}`, token);
    expect(updated.name).toBe('Updated Login Page');

    // Delete
    await apiDelete(request, `${base}/${created.id}`, token);
    const resp = await request.get(`${base}/${created.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect([400, 404]).toContain(resp.status());

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('feature status workflow via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Feature Status ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/features`;

    const created = await apiPost<{ id: string; status: string }>(
      request, base, token, { name: 'Status WF Feature', description: 'wf test' },
    );
    expect(created.status).toBe('draft');

    const statusPath = `${base}/${created.id}/status`;
    const featurePath = `${base}/${created.id}`;

    // draft -> ready
    await apiPatch(request, statusPath, token, { status: 'ready' });
    let f = await apiGet<{ status: string }>(request, featurePath, token);
    expect(f.status).toBe('ready');

    // ready -> in_progress
    await apiPatch(request, statusPath, token, { status: 'in_progress' });
    f = await apiGet<{ status: string }>(request, featurePath, token);
    expect(f.status).toBe('in_progress');

    // in_progress -> done
    await apiPatch(request, statusPath, token, { status: 'done' });
    f = await apiGet<{ status: string }>(request, featurePath, token);
    expect(f.status).toBe('done');

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('feature changelogs via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Feature Changelogs ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/features`;

    const created = await apiPost<{ id: string }>(
      request, base, token, { name: 'Changelog Feature', description: 'cl test' },
    );

    const changelogPath = `${base}/${created.id}/changelogs`;
    const featurePath = `${base}/${created.id}`;

    await apiPatch(request, changelogPath, token, {
      user_changelog: 'Added dark mode support',
      tech_changelog: 'Refactored CSS variables',
    });

    const got = await apiGet<{ user_changelog: string; tech_changelog: string }>(request, featurePath, token);
    expect(got.user_changelog).toBe('Added dark mode support');
    expect(got.tech_changelog).toBe('Refactored CSS variables');

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('feature with tasks summary via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Feature Tasks ' + Date.now(),
    });

    const feat = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/features`, token, {
        name: 'Tasks Feature', description: 'with tasks',
      },
    );

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'Implement login', summary: 'Implement the login page UI',
      },
    );

    // Assign task to feature
    await apiPatch(request, `/api/projects/${proj.id}/tasks/${task.id}`, token, {
      feature_id: feat.id,
    });

    // Check feature list includes task summary
    const list = await apiGet<any[]>(request, `/api/projects/${proj.id}/features`, token);
    const found = list.find((f: any) => f.id === feat.id);
    expect(found).toBeTruthy();
    const total = found.task_summary.backlog_count + found.task_summary.todo_count +
      found.task_summary.in_progress_count + found.task_summary.done_count +
      found.task_summary.blocked_count;
    expect(total).toBeGreaterThanOrEqual(1);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('feature stats via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Feature Stats ' + Date.now(),
    });

    await apiPost(request, `/api/projects/${proj.id}/features`, token, {
      name: 'Stats Feature', description: 'stats test',
    });

    const stats = await apiGet<{ total_count: number; not_ready_count: number }>(
      request, `/api/projects/${proj.id}/stats/features`, token,
    );
    expect(stats.total_count).toBeGreaterThanOrEqual(1);
    expect(stats.not_ready_count).toBeGreaterThanOrEqual(1);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('create feature via UI', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="project-open-btn"]').first().click();
    await page.waitForURL(/\/projects\/.+\/board/);

    await page.locator('[data-qa="nav-features-btn"]').click();
    await page.waitForURL(/\/projects\/.+\/features/);

    await page.locator('[data-qa="add-feature-btn"]').click();
    await expect(page.locator('[data-qa="new-feature-name-input"]')).toBeVisible();

    const featureName = 'PW Feature ' + Date.now();
    await page.locator('[data-qa="new-feature-name-input"]').fill(featureName);
    await page.locator('[data-qa="new-feature-description-textarea"]').fill('Created by Playwright');
    await page.locator('[data-qa="confirm-create-feature-btn"]').click();

    await expect(page.locator('[data-qa="feature-row"]').filter({ hasText: featureName })).toBeVisible({ timeout: 10_000 });
  });
});
