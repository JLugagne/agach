import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiDelete, apiPut, uniqueSlug } from './helpers';

test.describe('Teams', () => {
  test('team CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const slug = uniqueSlug('e2e-team');

    // Create
    const team = await apiPost<{ id: string; name: string; slug: string; description: string }>(
      request, '/api/identity/teams', token, {
        name: 'PW CRUD Team', slug, description: 'Created by Playwright',
      },
    );
    expect(team.id).toBeTruthy();
    expect(team.name).toBe('PW CRUD Team');
    expect(team.slug).toBe(slug);

    // List
    const teams = await apiGet<any[]>(request, '/api/identity/teams', token);
    expect(teams.some((t: any) => t.id === team.id)).toBeTruthy();

    // Delete
    const delResp = await request.delete(`/api/identity/teams/${team.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(delResp.status()).toBe(204);

    const after = await apiGet<any[]>(request, '/api/identity/teams', token);
    expect(after.every((t: any) => t.id !== team.id)).toBeTruthy();
  });

  test('assign and remove user from team via API', async ({ request }) => {
    const token = await apiLogin(request);
    const slug = uniqueSlug('e2e-assign');

    const team = await apiPost<{ id: string }>(
      request, '/api/identity/teams', token, {
        name: 'PW Assign Team', slug, description: 'assignment test',
      },
    );

    const users = await apiGet<any[]>(request, '/api/identity/users', token);
    const admin = users.find((u: any) => u.role === 'admin');
    expect(admin).toBeTruthy();

    // Assign
    await apiPut(request, `/api/identity/users/${admin.id}/team`, token, {
      team_id: team.id,
    });

    const userAfterAssign = (await apiGet<any[]>(request, '/api/identity/users', token))
      .find((u: any) => u.id === admin.id);
    expect(userAfterAssign.team_ids).toContain(team.id);

    // Remove
    const removeResp = await request.delete(`/api/identity/users/${admin.id}/team`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { team_id: team.id },
    });
    expect(removeResp.status()).toBe(204);

    const userAfterRemove = (await apiGet<any[]>(request, '/api/identity/users', token))
      .find((u: any) => u.id === admin.id);
    expect(userAfterRemove.team_ids).not.toContain(team.id);

    // Cleanup
    await request.delete(`/api/identity/teams/${team.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
  });

  test('delete team removes memberships via API', async ({ request }) => {
    const token = await apiLogin(request);

    const team = await apiPost<{ id: string }>(
      request, '/api/identity/teams', token, {
        name: 'PW Ephemeral Team', slug: uniqueSlug('e2e-ephemeral'),
      },
    );

    const users = await apiGet<any[]>(request, '/api/identity/users', token);
    const admin = users.find((u: any) => u.role === 'admin');
    await apiPut(request, `/api/identity/users/${admin.id}/team`, token, {
      team_id: team.id,
    });

    // Delete team
    await request.delete(`/api/identity/teams/${team.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    const userAfter = (await apiGet<any[]>(request, '/api/identity/users', token))
      .find((u: any) => u.id === admin.id);
    expect(userAfter.team_ids).not.toContain(team.id);
  });

  test('set user role via API', async ({ request }) => {
    const token = await apiLogin(request);

    // Register a separate user to change role on
    const email = `pw-role-${Date.now()}@test.local`;
    await request.post('/api/auth/register', {
      data: { email, password: 'RoleTest123!', display_name: 'PW Role User' },
    });

    const users = await apiGet<any[]>(request, '/api/identity/users', token);
    const target = users.find((u: any) => u.email === email);
    expect(target).toBeTruthy();
    expect(target.role).toBe('member');

    // Promote to admin
    let resp = await request.put(`/api/identity/users/${target.id}/role`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { role: 'admin' },
    });
    expect(resp.status()).toBe(204);

    let updated = (await apiGet<any[]>(request, '/api/identity/users', token))
      .find((u: any) => u.id === target.id);
    expect(updated.role).toBe('admin');

    // Demote back to member
    resp = await request.put(`/api/identity/users/${target.id}/role`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { role: 'member' },
    });
    expect(resp.status()).toBe(204);
  });

  test('teams page via UI', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/teams');

    // Create a new team via UI
    await page.locator('[data-qa="create-team-btn"]').click();
    const nameInput = page.locator('[data-qa="new-team-name-input"]');
    await expect(nameInput).toBeVisible();

    const teamName = 'PW Team ' + Date.now();
    await nameInput.fill(teamName);
    await page.locator('[data-qa="new-team-slug-input"]').fill(uniqueSlug('pw-team'));
    await page.locator('[data-qa="confirm-create-team-btn"]').click();

    // Team should appear in list (use first() to avoid strict mode with select options)
    await expect(page.locator(`text=${teamName}`).first()).toBeVisible({ timeout: 10_000 });
  });
});
