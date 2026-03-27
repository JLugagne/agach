import { test, expect } from '@playwright/test';
import { apiLogin, apiPost, apiGet, apiDelete, apiPut, uniqueSlug } from './helpers';

test.describe('Notifications', () => {
  test('project-scoped notification CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Notif CRUD ' + Date.now(),
    });

    const base = `/api/projects/${proj.id}/notifications`;

    // Create
    const created = await apiPost<{ id: string; severity: string; title: string; read_at: string | null }>(
      request, base, token, {
        severity: 'info', title: 'Build completed',
        text: 'The build finished successfully.',
      },
    );
    expect(created.id).toBeTruthy();
    expect(created.severity).toBe('info');
    expect(created.title).toBe('Build completed');
    expect(created.read_at).toBeNull();

    // List
    const notifs = await apiGet<any[]>(request, base, token);
    expect(notifs.some((n: any) => n.id === created.id)).toBeTruthy();

    // Mark read
    await apiPut(request, `/api/notifications/${created.id}/read`, token);

    // Verify read
    const afterRead = await apiGet<any[]>(request, base, token);
    const readNotif = afterRead.find((n: any) => n.id === created.id);
    expect(readNotif?.read_at).toBeTruthy();

    // Delete
    await apiDelete(request, `/api/notifications/${created.id}`, token);
    const afterDel = await apiGet<any[]>(request, base, token);
    expect(afterDel.every((n: any) => n.id !== created.id)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('global notifications via API', async ({ request }) => {
    const token = await apiLogin(request);

    const created = await apiPost<{ id: string; scope: string }>(
      request, '/api/notifications', token, {
        severity: 'warning', title: 'System maintenance',
        text: 'Scheduled downtime at midnight.',
      },
    );
    expect(created.scope).toBe('global');

    const notifs = await apiGet<any[]>(request, '/api/notifications', token);
    expect(notifs.some((n: any) => n.id === created.id)).toBeTruthy();

    await apiDelete(request, `/api/notifications/${created.id}`, token);
  });

  test('unread count via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Notif Unread ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/notifications`;

    const n1 = await apiPost<{ id: string }>(request, base, token, {
      severity: 'info', title: 'Unread 1', text: 'First unread.',
    });
    const n2 = await apiPost<{ id: string }>(request, base, token, {
      severity: 'error', title: 'Unread 2', text: 'Second unread.',
    });

    const uc = await apiGet<{ unread_count: number }>(request, `${base}/unread-count`, token);
    expect(uc.unread_count).toBe(2);

    // Mark one as read
    await apiPut(request, `/api/notifications/${n1.id}/read`, token);

    const uc2 = await apiGet<{ unread_count: number }>(request, `${base}/unread-count`, token);
    expect(uc2.unread_count).toBe(1);

    await apiDelete(request, `/api/notifications/${n1.id}`, token);
    await apiDelete(request, `/api/notifications/${n2.id}`, token);
    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('read-all notifications via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Notif ReadAll ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/notifications`;

    const ids: string[] = [];
    for (let i = 1; i <= 3; i++) {
      const n = await apiPost<{ id: string }>(request, base, token, {
        severity: 'info', title: `ReadAll ${i}`, text: `Notification ${i}.`,
      });
      ids.push(n.id);
    }

    const uc = await apiGet<{ unread_count: number }>(request, `${base}/unread-count`, token);
    expect(uc.unread_count).toBe(3);

    // Read all
    await apiPut(request, `${base}/read-all`, token);

    const uc2 = await apiGet<{ unread_count: number }>(request, `${base}/unread-count`, token);
    expect(uc2.unread_count).toBe(0);

    for (const id of ids) {
      await apiDelete(request, `/api/notifications/${id}`, token);
    }
    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('all severity levels via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Notif Sev ' + Date.now(),
    });
    const base = `/api/projects/${proj.id}/notifications`;

    const severities = ['info', 'success', 'warning', 'error'];
    const createdIds: Record<string, string> = {};

    for (const sev of severities) {
      const n = await apiPost<{ id: string; severity: string }>(request, base, token, {
        severity: sev, title: `Severity ${sev}`, text: `Testing ${sev}`,
      });
      expect(n.severity).toBe(sev);
      createdIds[sev] = n.id;
    }

    const notifs = await apiGet<any[]>(request, base, token);
    for (const sev of severities) {
      expect(notifs.some((n: any) => n.id === createdIds[sev])).toBeTruthy();
    }

    for (const id of Object.values(createdIds)) {
      await apiDelete(request, `/api/notifications/${id}`, token);
    }
    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });
});
