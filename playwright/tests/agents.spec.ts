import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiDelete, apiPatch, uniqueSlug } from './helpers';

test.describe('Agents', () => {
  test('agent CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const slug = uniqueSlug('agent-crud');

    // Create
    const created = await apiPost<{ id: string; slug: string; name: string }>(
      request, '/api/agents', token, {
        slug, name: 'CRUD Agent', icon: '⚙️', color: '#FF0000',
        description: 'An agent for CRUD testing', tech_stack: ['Go'], sort_order: 10,
      },
    );
    expect(created.id).toBeTruthy();
    expect(created.slug).toBe(slug);
    expect(created.name).toBe('CRUD Agent');

    // Read
    const fetched = await apiGet<{ id: string; slug: string; name: string }>(
      request, `/api/agents/${slug}`, token,
    );
    expect(fetched.id).toBe(created.id);

    // List
    const agents = await apiGet<any[]>(request, '/api/agents', token);
    expect(agents.some((a: any) => a.slug === slug)).toBeTruthy();

    // Update
    const patchResp = await request.patch(`/api/agents/${slug}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name: 'Updated CRUD Agent' },
    });
    expect(patchResp.ok()).toBeTruthy();

    const refetched = await apiGet<{ name: string }>(request, `/api/agents/${slug}`, token);
    expect(refetched.name).toBe('Updated CRUD Agent');

    // Delete
    await apiDelete(request, `/api/agents/${slug}`, token);

    const resp = await request.get(`/api/agents/${slug}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(400);
  });

  test('clone agent via API', async ({ request }) => {
    const token = await apiLogin(request);
    const srcSlug = uniqueSlug('agent-clone-src');
    const dstSlug = uniqueSlug('agent-clone-dst');

    await apiPost(request, '/api/agents', token, {
      slug: srcSlug, name: 'Source Agent', icon: '📋', color: '#00FF00', sort_order: 1,
    });

    const cloned = await apiPost<{ id: string; slug: string; name: string }>(
      request, `/api/agents/${srcSlug}/clone`, token, {
        new_slug: dstSlug, new_name: 'Cloned Agent',
      },
    );
    expect(cloned.slug).toBe(dstSlug);
    expect(cloned.name).toBe('Cloned Agent');

    await apiDelete(request, `/api/agents/${srcSlug}`, token);
    await apiDelete(request, `/api/agents/${dstSlug}`, token);
  });

  test('create agent via UI', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/roles');

    await page.locator('[data-qa="new-agent-btn"]').click();
    const nameInput = page.locator('[data-qa="agent-name-input"]');
    await expect(nameInput).toBeVisible();

    const agentName = 'PW Agent ' + Date.now();
    await nameInput.fill(agentName);
    await page.locator('[data-qa="agent-slug-input"]').fill(uniqueSlug('pw-agent'));
    await page.locator('[data-qa="agent-save-btn"]').click();

    // Agent should appear in list
    await expect(page.locator('[data-qa="agent-card"]').filter({ hasText: agentName })).toBeVisible({ timeout: 10_000 });
  });
});

test.describe('Skills', () => {
  test('skill CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const slug = uniqueSlug('skill-crud');

    // Create
    const created = await apiPost<{ id: string; slug: string; name: string }>(
      request, '/api/skills', token, {
        slug, name: 'CRUD Skill', description: 'A skill for CRUD testing',
        content: 'Some content here', icon: 'wrench', color: '#0000FF', sort_order: 5,
      },
    );
    expect(created.id).toBeTruthy();
    expect(created.slug).toBe(slug);

    // Read
    const fetched = await apiGet<{ slug: string }>(request, `/api/skills/${slug}`, token);
    expect(fetched.slug).toBe(slug);

    // List
    const skills = await apiGet<any[]>(request, '/api/skills', token);
    expect(skills.some((s: any) => s.slug === slug)).toBeTruthy();

    // Update
    const patchResp = await request.patch(`/api/skills/${slug}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name: 'Updated Skill' },
    });
    expect(patchResp.ok()).toBeTruthy();

    const updatedSkill = await apiGet<{ name: string }>(request, `/api/skills/${slug}`, token);
    expect(updatedSkill.name).toBe('Updated Skill');

    // Delete
    await apiDelete(request, `/api/skills/${slug}`, token);

    const resp = await request.get(`/api/skills/${slug}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(400);
  });

  test('create skill via UI', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/skills');

    await page.locator('[data-qa="new-skill-btn"]').click();
    await expect(page.locator('[data-qa="skill-name-input"]')).toBeVisible();

    const skillName = 'PW Skill ' + Date.now();
    await page.locator('[data-qa="skill-name-input"]').fill(skillName);
    await page.locator('[data-qa="skill-slug-input"]').fill(uniqueSlug('pw-skill'));
    await page.locator('[data-qa="skill-content-textarea"]').fill('Test content');
    await page.locator('[data-qa="save-skill-btn"]').click();

    await expect(page.locator('[data-qa="skill-card"]').filter({ hasText: skillName })).toBeVisible({ timeout: 10_000 });
  });

  test('assign skill to agent via API', async ({ request }) => {
    const token = await apiLogin(request);
    const agentSlug = uniqueSlug('agent-sk');
    const skillSlug = uniqueSlug('skill-sk');

    await apiPost(request, '/api/agents', token, {
      slug: agentSlug, name: 'Agent With Skills', sort_order: 1,
    });
    await apiPost(request, '/api/skills', token, {
      slug: skillSlug, name: 'Assignable Skill', content: 'skill content', sort_order: 1,
    });

    // Assign
    const assignResp = await request.post(`/api/agents/${agentSlug}/skills`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { skill_slug: skillSlug },
    });
    expect(assignResp.ok()).toBeTruthy();

    // Verify
    const agentSkills = await apiGet<any[]>(request, `/api/agents/${agentSlug}/skills`, token);
    expect(agentSkills.some((s: any) => s.slug === skillSlug)).toBeTruthy();

    const agent = await apiGet<{ skill_count: number }>(request, `/api/agents/${agentSlug}`, token);
    expect(agent.skill_count).toBeGreaterThanOrEqual(1);

    // Remove
    await apiDelete(request, `/api/agents/${agentSlug}/skills/${skillSlug}`, token);
    const after = await apiGet<any[]>(request, `/api/agents/${agentSlug}/skills`, token);
    expect(after.every((s: any) => s.slug !== skillSlug)).toBeTruthy();

    await apiDelete(request, `/api/agents/${agentSlug}`, token);
    await apiDelete(request, `/api/skills/${skillSlug}`, token);
  });
});

test.describe('Specialized Agents', () => {
  test('specialized agent CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const parentSlug = uniqueSlug('agent-spec');
    const skillSlug = uniqueSlug('skill-spec');
    const specSlug = uniqueSlug('spec');

    const parent = await apiPost<{ id: string }>(request, '/api/agents', token, {
      slug: parentSlug, name: 'Parent Agent', sort_order: 1,
    });
    await apiPost(request, '/api/skills', token, {
      slug: skillSlug, name: 'Spec Skill', content: 'specialized content', sort_order: 1,
    });

    // Create specialized
    const spec = await apiPost<{ id: string; slug: string; parent_agent_id: string }>(
      request, `/api/agents/${parentSlug}/specialized`, token, {
        slug: specSlug, name: 'Specialized Agent', skill_slugs: [skillSlug], sort_order: 1,
      },
    );
    expect(spec.id).toBeTruthy();
    expect(spec.slug).toBe(specSlug);
    expect(spec.parent_agent_id).toBe(parent.id);

    // Read
    const fetched = await apiGet<{ slug: string }>(
      request, `/api/agents/${parentSlug}/specialized/${specSlug}`, token,
    );
    expect(fetched.slug).toBe(specSlug);

    // List
    const specList = await apiGet<any[]>(request, `/api/agents/${parentSlug}/specialized`, token);
    expect(specList.some((s: any) => s.slug === specSlug)).toBeTruthy();

    // Update
    const patchResp = await request.patch(`/api/agents/${parentSlug}/specialized/${specSlug}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name: 'Updated Specialized' },
    });
    expect(patchResp.ok()).toBeTruthy();

    // Delete
    await apiDelete(request, `/api/agents/${parentSlug}/specialized/${specSlug}`, token);

    const resp = await request.get(`/api/agents/${parentSlug}/specialized/${specSlug}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(400);

    await apiDelete(request, `/api/agents/${parentSlug}`, token);
    await apiDelete(request, `/api/skills/${skillSlug}`, token);
  });
});
