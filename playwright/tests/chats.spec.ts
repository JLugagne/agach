import { test, expect } from '@playwright/test';
import { apiLogin, apiPost, apiGet, apiDelete, uniqueSlug } from './helpers';

function chatBasePath(projectID: string, featureID: string): string {
  return `/api/projects/${projectID}/features/${featureID}/chats`;
}

async function createChatFixtures(request: any, token: string): Promise<{ projectID: string; featureID: string }> {
  const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
    name: 'PW Chat Project ' + Date.now(),
  });
  const feat = await apiPost<{ id: string }>(
    request, `/api/projects/${proj.id}/features`, token, {
      name: 'PW Chat Feature ' + Date.now(),
    },
  );
  return { projectID: proj.id, featureID: feat.id };
}

async function isChatsEnabled(request: any, token: string, projectID: string, featureID: string): Promise<boolean> {
  const resp = await request.get(chatBasePath(projectID, featureID), {
    headers: { Authorization: `Bearer ${token}` },
  });
  return resp.status() !== 404;
}

test.describe('Chats', () => {
  test('start and end chat session via API', async ({ request }) => {
    const token = await apiLogin(request);
    const { projectID, featureID } = await createChatFixtures(request, token);
    const base = chatBasePath(projectID, featureID);

    if (!(await isChatsEnabled(request, token, projectID, featureID))) {
      test.skip();
      return;
    }

    // Start session
    const session = await apiPost<{ id: string; feature_id: string; state: string }>(
      request, base, token, {},
    );
    expect(session.id).toBeTruthy();
    expect(session.feature_id).toBe(featureID);
    expect(session.state).toBe('active');

    // End session
    const endResp = await apiPost<{ message: string }>(
      request, `${base}/${session.id}/end`, token, {},
    );
    expect(endResp.message).toBe('chat session ended');

    // Verify ended
    const got = await apiGet<{ state: string; ended_at: string | null }>(
      request, `${base}/${session.id}`, token,
    );
    expect(got.state).toBe('ended');
    expect(got.ended_at).toBeTruthy();

    await apiDelete(request, `/api/projects/${projectID}`, token);
  });

  test('update chat stats via API', async ({ request }) => {
    const token = await apiLogin(request);
    const { projectID, featureID } = await createChatFixtures(request, token);
    const base = chatBasePath(projectID, featureID);

    if (!(await isChatsEnabled(request, token, projectID, featureID))) {
      test.skip();
      return;
    }

    const session = await apiPost<{ id: string }>(request, base, token, {});

    const statsResp = await request.put(`${base}/${session.id}/stats`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        input_tokens: 1500, output_tokens: 3000,
        cache_read_tokens: 200, cache_write_tokens: 100,
        model: 'claude-sonnet-4-20250514',
      },
    });
    expect(statsResp.ok()).toBeTruthy();

    const got = await apiGet<{
      input_tokens: number; output_tokens: number;
      cache_read_tokens: number; cache_write_tokens: number; model: string;
    }>(request, `${base}/${session.id}`, token);

    expect(got.input_tokens).toBe(1500);
    expect(got.output_tokens).toBe(3000);
    expect(got.cache_read_tokens).toBe(200);
    expect(got.cache_write_tokens).toBe(100);
    expect(got.model).toBe('claude-sonnet-4-20250514');

    await apiDelete(request, `/api/projects/${projectID}`, token);
  });

  test('list and get chat sessions via API', async ({ request }) => {
    const token = await apiLogin(request);
    const { projectID, featureID } = await createChatFixtures(request, token);
    const base = chatBasePath(projectID, featureID);

    if (!(await isChatsEnabled(request, token, projectID, featureID))) {
      test.skip();
      return;
    }

    const s1 = await apiPost<{ id: string }>(request, base, token, {});
    const s2 = await apiPost<{ id: string }>(request, base, token, {});
    expect(s1.id).not.toBe(s2.id);

    const list = await apiGet<any[]>(request, base, token);
    expect(list.length).toBeGreaterThanOrEqual(2);

    const ids = new Set(list.map((s: any) => s.id));
    expect(ids.has(s1.id)).toBeTruthy();
    expect(ids.has(s2.id)).toBeTruthy();

    const got = await apiGet<{ id: string; state: string }>(request, `${base}/${s1.id}`, token);
    expect(got.id).toBe(s1.id);
    expect(got.state).toBe('active');

    await apiDelete(request, `/api/projects/${projectID}`, token);
  });

  test('download without upload fails via API', async ({ request }) => {
    const token = await apiLogin(request);
    const { projectID, featureID } = await createChatFixtures(request, token);
    const base = chatBasePath(projectID, featureID);

    if (!(await isChatsEnabled(request, token, projectID, featureID))) {
      test.skip();
      return;
    }

    const session = await apiPost<{ id: string }>(request, base, token, {});

    const dlResp = await request.get(`${base}/${session.id}/download`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(dlResp.ok()).toBeFalsy();

    await apiDelete(request, `/api/projects/${projectID}`, token);
  });
});
