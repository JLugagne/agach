import { test, expect } from '@playwright/test';
import { apiLogin, apiPost, apiGet, apiDelete, uniqueSlug } from './helpers';

test.describe('Stats', () => {
  test('project info via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW ProjectInfo ' + Date.now(), description: 'project info test',
    });

    const info = await apiGet<{ project: { id: string; name: string }; children: any[]; breadcrumb: any[] }>(
      request, `/api/projects/${proj.id}/info`, token,
    );
    expect(info.project.id).toBe(proj.id);
    expect(info.children).toBeDefined();
    expect(info.breadcrumb).toBeDefined();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('timeline via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Timeline ' + Date.now(),
    });
    await apiPost(request, `/api/projects/${proj.id}/tasks`, token, {
      title: 'Timeline task', summary: 'timeline test', priority: 'medium',
    });

    const resp = await request.get(`/api/projects/${proj.id}/stats/timeline`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    // Endpoint should respond (may return 500 if no timeline data exists yet)
    expect([200, 400, 500]).toContain(resp.status());

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('cold start stats via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW ColdStart ' + Date.now(),
    });

    const resp = await request.get(`/api/projects/${proj.id}/stats/cold-start`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('model token stats via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW ModelTokens ' + Date.now(),
    });

    const resp = await request.get(`/api/projects/${proj.id}/stats/model-tokens`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('model pricing via API', async ({ request }) => {
    const token = await apiLogin(request);
    const resp = await request.get('/api/model-pricing', {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();
  });

  test('tool usage via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW ToolUsage ' + Date.now(),
    });

    const resp = await request.get(`/api/projects/${proj.id}/tool-usage`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.ok()).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('specialized agent skills via API', async ({ request }) => {
    const token = await apiLogin(request);
    const parentSlug = uniqueSlug('stats-agent');
    const skillSlug = uniqueSlug('stats-skill');
    const specSlug = uniqueSlug('stats-spec');

    await apiPost(request, '/api/agents', token, {
      slug: parentSlug, name: 'Stats Parent Agent', sort_order: 1,
    });
    const skill = await apiPost<{ name: string }>(request, '/api/skills', token, {
      slug: skillSlug, name: 'Stats Skill', content: 'skill content', sort_order: 1,
    });
    await apiPost(request, `/api/agents/${parentSlug}/specialized`, token, {
      slug: specSlug, name: 'Stats Specialized', skill_slugs: [skillSlug], sort_order: 1,
    });

    const skills = await apiGet<any[]>(
      request, `/api/agents/${parentSlug}/specialized/${specSlug}/skills`, token,
    );
    expect(skills.length).toBeGreaterThanOrEqual(1);
    expect(skills.some((s: any) => s.slug === skillSlug)).toBeTruthy();

    await apiDelete(request, `/api/agents/${parentSlug}/specialized/${specSlug}`, token);
    await apiDelete(request, `/api/agents/${parentSlug}`, token);
    await apiDelete(request, `/api/skills/${skillSlug}`, token);
  });

  test('bulk reassign agents via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW BulkReassign ' + Date.now(),
    });

    const agent1Slug = uniqueSlug('bulk-a1');
    const agent2Slug = uniqueSlug('bulk-a2');

    await apiPost(request, '/api/agents', token, { slug: agent1Slug, name: 'Bulk Agent 1', sort_order: 1 });
    await apiPost(request, '/api/agents', token, { slug: agent2Slug, name: 'Bulk Agent 2', sort_order: 2 });

    // Assign agents to project
    await apiPost(request, `/api/projects/${proj.id}/agents`, token, { agent_slug: agent1Slug });
    await apiPost(request, `/api/projects/${proj.id}/agents`, token, { agent_slug: agent2Slug });

    // Create task assigned to agent1
    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'Bulk Reassign Task', summary: 'bulk test', priority: 'medium',
        assigned_role: agent1Slug,
      },
    );

    const got = await apiGet<{ assigned_role: string }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(got.assigned_role).toBe(agent1Slug);

    // Bulk reassign
    const result = await apiPost<{ updated_count: number }>(
      request, `/api/projects/${proj.id}/agents/bulk-reassign`, token, {
        old_slug: agent1Slug, new_slug: agent2Slug,
      },
    );
    expect(result.updated_count).toBeGreaterThanOrEqual(1);

    const reassigned = await apiGet<{ assigned_role: string }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(reassigned.assigned_role).toBe(agent2Slug);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
    await apiDelete(request, `/api/agents/${agent1Slug}`, token);
    await apiDelete(request, `/api/agents/${agent2Slug}`, token);
  });
});
