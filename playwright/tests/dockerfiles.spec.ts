import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiDelete, uniqueSlug } from './helpers';

test.describe('Dockerfiles', () => {
  test('dockerfile CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const slug = uniqueSlug('df-crud');

    // Create
    const created = await apiPost<{ id: string; slug: string; name: string; content: string; is_latest: boolean }>(
      request, '/api/dockerfiles', token, {
        slug, name: 'CRUD Dockerfile', description: 'A dockerfile for CRUD testing',
        version: '1.0.0', content: 'FROM golang:1.22\nRUN echo hello',
        is_latest: false, sort_order: 10,
      },
    );
    expect(created.id).toBeTruthy();
    expect(created.slug).toBe(slug);
    expect(created.name).toBe('CRUD Dockerfile');
    expect(created.content).toBe('FROM golang:1.22\nRUN echo hello');

    // Read by ID
    const fetched = await apiGet<{ slug: string }>(request, `/api/dockerfiles/${created.id}`, token);
    expect(fetched.slug).toBe(slug);

    // Read by slug
    const bySlug = await apiGet<{ id: string }>(request, `/api/dockerfiles/by-slug/${slug}`, token);
    expect(bySlug.id).toBe(created.id);

    // List
    const list = await apiGet<any[]>(request, '/api/dockerfiles', token);
    expect(list.some((d: any) => d.id === created.id)).toBeTruthy();

    // Update
    const patchResp = await request.patch(`/api/dockerfiles/${created.id}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name: 'Updated Dockerfile', content: 'FROM golang:1.23\nRUN echo updated' },
    });
    expect(patchResp.ok()).toBeTruthy();

    const refetched = await apiGet<{ name: string; content: string }>(
      request, `/api/dockerfiles/${created.id}`, token,
    );
    expect(refetched.name).toBe('Updated Dockerfile');
    expect(refetched.content).toBe('FROM golang:1.23\nRUN echo updated');

    // Delete
    await apiDelete(request, `/api/dockerfiles/${created.id}`, token);

    const resp = await request.get(`/api/dockerfiles/${created.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(400);
  });

  test('dockerfile is_latest flag', async ({ request }) => {
    const token = await apiLogin(request);
    const slug1 = uniqueSlug('df-latest1');
    const slug2 = uniqueSlug('df-latest2');

    const first = await apiPost<{ id: string; is_latest: boolean }>(
      request, '/api/dockerfiles', token, {
        slug: slug1, name: 'First Latest', version: '1.0.0',
        content: 'FROM alpine:3.19', is_latest: true, sort_order: 1,
      },
    );
    expect(first.is_latest).toBe(true);

    const second = await apiPost<{ id: string; is_latest: boolean }>(
      request, '/api/dockerfiles', token, {
        slug: slug2, name: 'Second Latest', version: '2.0.0',
        content: 'FROM alpine:3.20', is_latest: true, sort_order: 2,
      },
    );
    expect(second.is_latest).toBe(true);

    await apiDelete(request, `/api/dockerfiles/${first.id}`, token);
    await apiDelete(request, `/api/dockerfiles/${second.id}`, token);
  });

  test('dockerfile project assignment via API', async ({ request }) => {
    const token = await apiLogin(request);
    const dfSlug = uniqueSlug('df-assign');

    const df = await apiPost<{ id: string }>(request, '/api/dockerfiles', token, {
      slug: dfSlug, name: 'Assignable Dockerfile', version: '1.0.0',
      content: 'FROM ubuntu:24.04', is_latest: false, sort_order: 1,
    });

    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'DF Assignment Project ' + Date.now(),
    });

    // Assign
    const assignResp = await request.put(`/api/projects/${proj.id}/dockerfile`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { dockerfile_id: df.id },
    });
    expect(assignResp.ok()).toBeTruthy();

    const assigned = await apiGet<{ id: string; slug: string }>(
      request, `/api/projects/${proj.id}/dockerfile`, token,
    );
    expect(assigned.id).toBe(df.id);

    // Remove assignment
    await apiDelete(request, `/api/projects/${proj.id}/dockerfile`, token);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
    await apiDelete(request, `/api/dockerfiles/${df.id}`, token);
  });

  test('create dockerfile via UI', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/dockerfiles');

    await page.locator('[data-qa="new-dockerfile-btn"]').click();
    await expect(page.locator('[data-qa="dockerfile-name-input"]')).toBeVisible();

    const dfName = 'PW Dockerfile ' + Date.now();
    await page.locator('[data-qa="dockerfile-name-input"]').fill(dfName);
    await page.locator('[data-qa="dockerfile-slug-input"]').fill(uniqueSlug('pw-df'));
    await page.locator('[data-qa="dockerfile-version-input"]').fill('1.0.0');

    // Save
    const saveBtn = page.locator('[data-qa="save-dockerfile-btn"]');
    if (await saveBtn.isVisible()) {
      await saveBtn.click();
    }
  });
});
