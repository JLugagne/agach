import { test, expect } from '@playwright/test';
import { uiLogin, apiLogin, apiPost, apiGet, apiDelete, apiPatch, uniqueSlug } from './helpers';

test.describe('Tasks', () => {
  test('create task via UI new task modal', async ({ page }) => {
    await uiLogin(page);
    await page.locator('[data-qa="project-open-btn"]').first().click();
    await page.waitForURL(/\/projects\/.+\/board/);

    // Open new task modal
    await page.locator('[data-qa="new-task-btn"]').click();
    await expect(page.locator('[data-qa="new-task-modal"]')).toBeVisible();

    const title = 'PW Task ' + Date.now();
    await page.locator('[data-qa="new-task-title-input"]').fill(title);
    await page.locator('[data-qa="new-task-summary-input"]').fill('Created by Playwright');
    await page.locator('[data-qa="new-task-submit-btn"]').click();

    // Modal should close and task should appear on board
    await expect(page.locator('[data-qa="new-task-modal"]')).not.toBeVisible({ timeout: 5_000 });
    await expect(page.locator('[data-qa="task-card"]').filter({ hasText: title })).toBeVisible({ timeout: 10_000 });
  });

  test('task CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Tasks CRUD ' + Date.now(),
    });

    // Create
    const task = await apiPost<{ id: string; title: string; priority: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW CRUD Task',
        summary: 'A test task summary for Playwright',
        description: 'Detailed description',
        priority: 'medium',
        assigned_role: 'backend',
      },
    );
    expect(task.id).toBeTruthy();
    expect(task.title).toBe('PW CRUD Task');

    // Read
    const got = await apiGet<{ id: string; title: string }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(got.id).toBe(task.id);

    // List
    const list = await apiGet<any[]>(request, `/api/projects/${proj.id}/tasks`, token);
    expect(list.some((t: any) => t.id === task.id)).toBeTruthy();

    // Update
    await apiPatch(request, `/api/projects/${proj.id}/tasks/${task.id}`, token, {
      title: 'PW Updated Task',
      priority: 'high',
    });
    const updated = await apiGet<{ title: string; priority: string }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(updated.title).toBe('PW Updated Task');
    expect(updated.priority).toBe('high');

    // Delete
    const delResp = await request.delete(`/api/projects/${proj.id}/tasks/${task.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(delResp.ok()).toBeTruthy();

    // Verify deleted
    const getResp = await request.get(`/api/projects/${proj.id}/tasks/${task.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(getResp.ok()).toBeFalsy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('move task between columns via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Move Task ' + Date.now(),
    });

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW Move Task', summary: 'move test', priority: 'medium',
      },
    );

    // Move to in_progress
    const moveResp = await request.post(`/api/projects/${proj.id}/tasks/${task.id}/move`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { target_column: 'in_progress' },
    });
    expect(moveResp.ok()).toBeTruthy();

    // Verify via board
    const board = await apiGet<{ columns: any[] }>(request, `/api/projects/${proj.id}/board`, token);
    const ipCol = board.columns.find((c: any) => c.slug === 'in_progress');
    expect(ipCol?.tasks?.some((t: any) => t.id === task.id)).toBeTruthy();

    // Verify started_at is set
    const got = await apiGet<{ started_at: string | null }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(got.started_at).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('complete task via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Complete Task ' + Date.now(),
    });

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW Complete Task', summary: 'complete test', priority: 'medium',
      },
    );

    // Move to in_progress first
    await request.post(`/api/projects/${proj.id}/tasks/${task.id}/move`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { target_column: 'in_progress' },
    });

    // Complete
    const summary = 'Task completed successfully. '.repeat(5);
    const completeResp = await request.post(`/api/projects/${proj.id}/tasks/${task.id}/complete`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        completion_summary: summary,
        files_modified: ['main.go', 'handler.go'],
        completed_by_agent: 'pw-test-agent',
      },
    });
    expect(completeResp.ok()).toBeTruthy();

    const got = await apiGet<{ completed_at: string | null; completed_by_agent: string; completion_summary: string }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(got.completed_at).toBeTruthy();
    expect(got.completed_by_agent).toBe('pw-test-agent');
    expect(got.completion_summary).toBe(summary);

    // Verify in done column
    const board = await apiGet<{ columns: any[] }>(request, `/api/projects/${proj.id}/board`, token);
    const doneCol = board.columns.find((c: any) => c.slug === 'done');
    expect(doneCol?.tasks?.some((t: any) => t.id === task.id)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('wont-do request and approve via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW WontDo ' + Date.now(),
    });

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW WontDo Task', summary: 'wontdo test', priority: 'medium',
      },
    );

    // Move to in_progress
    await request.post(`/api/projects/${proj.id}/tasks/${task.id}/move`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { target_column: 'in_progress' },
    });

    // Request won't do
    const wontDoResp = await request.post(`/api/projects/${proj.id}/tasks/${task.id}/wont-do`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        wont_do_reason: 'This task is no longer needed due to scope change. '.repeat(2),
        wont_do_requested_by: 'pw-human',
      },
    });
    expect(wontDoResp.ok()).toBeTruthy();

    const got = await apiGet<{ wont_do_requested: boolean }>(
      request, `/api/projects/${proj.id}/tasks/${task.id}`, token,
    );
    expect(got.wont_do_requested).toBe(true);

    // Should be in done column
    const board = await apiGet<{ columns: any[] }>(request, `/api/projects/${proj.id}/board`, token);
    const doneCol = board.columns.find((c: any) => c.slug === 'done');
    expect(doneCol?.tasks?.some((t: any) => t.id === task.id)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('task comments CRUD via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Comments ' + Date.now(),
    });

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW Comment Task', summary: 'comment test', priority: 'medium',
      },
    );

    const taskURL = `/api/projects/${proj.id}/tasks/${task.id}`;

    // Create comment
    const comment = await apiPost<{ id: string; content: string }>(
      request, `${taskURL}/comments`, token, {
        author_role: 'architect',
        author_name: 'PW Tester',
        content: 'This is a test comment.',
      },
    );
    expect(comment.id).toBeTruthy();
    expect(comment.content).toBe('This is a test comment.');

    // List comments
    const comments = await apiGet<any[]>(request, `${taskURL}/comments`, token);
    expect(comments.some((c: any) => c.id === comment.id)).toBeTruthy();

    // Update comment
    await apiPatch(request, `${taskURL}/comments/${comment.id}`, token, {
      content: 'Updated comment content.',
    });
    const updated = await apiGet<any[]>(request, `${taskURL}/comments`, token);
    const updatedComment = updated.find((c: any) => c.id === comment.id);
    expect(updatedComment?.content).toBe('Updated comment content.');
    expect(updatedComment?.edited_at).toBeTruthy();

    // Delete comment
    await apiDelete(request, `${taskURL}/comments/${comment.id}`, token);
    const after = await apiGet<any[]>(request, `${taskURL}/comments`, token);
    expect(after.every((c: any) => c.id !== comment.id)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('task backlog via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Backlog ' + Date.now(),
    });

    const task = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW Backlog Task', summary: 'backlog test', priority: 'low',
        start_in_backlog: true,
      },
    );

    const board = await apiGet<{ columns: any[] }>(request, `/api/projects/${proj.id}/board`, token);
    const backlogCol = board.columns.find((c: any) => c.slug === 'backlog');
    expect(backlogCol?.tasks?.some((t: any) => t.id === task.id)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('search tasks via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Search ' + Date.now(),
    });

    const keyword = uniqueSlug('searchterm');
    await apiPost(request, `/api/projects/${proj.id}/tasks`, token, {
      title: 'Alpha ' + keyword + ' Task',
      summary: 'Task with keyword in title',
      priority: 'medium',
    });
    await apiPost(request, `/api/projects/${proj.id}/tasks`, token, {
      title: 'Unrelated Beta Task',
      summary: 'should not match',
      priority: 'medium',
    });

    const results = await apiGet<any[]>(
      request, `/api/projects/${proj.id}/tasks/search?q=${keyword}`, token,
    );
    expect(results.length).toBeGreaterThanOrEqual(1);
    for (const r of results) {
      expect(r.title).toContain(keyword);
    }

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });

  test('move task to another project via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj1 = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW MoveProj Source ' + Date.now(),
    });
    const proj2 = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW MoveProj Target ' + Date.now(),
      parent_id: proj1.id,
    });

    const task = await apiPost<{ id: string; title: string }>(
      request, `/api/projects/${proj1.id}/tasks`, token, {
        title: 'PW Move To Project Task',
        summary: 'move to project test',
        priority: 'medium',
      },
    );

    const moveResp = await request.post(`/api/projects/${proj1.id}/tasks/${task.id}/move-to-project`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { target_project_id: proj2.id },
    });
    expect(moveResp.ok()).toBeTruthy();

    // Verify task is no longer in source
    const srcResp = await request.get(`/api/projects/${proj1.id}/tasks/${task.id}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(srcResp.ok()).toBeFalsy();

    // Verify task is in target
    const targetTasks = await apiGet<any[]>(request, `/api/projects/${proj2.id}/tasks`, token);
    expect(targetTasks.some((t: any) => t.title === task.title)).toBeTruthy();

    await apiDelete(request, `/api/projects/${proj2.id}`, token);
    await apiDelete(request, `/api/projects/${proj1.id}`, token);
  });

  test('reorder tasks via API', async ({ request }) => {
    const token = await apiLogin(request);
    const proj = await apiPost<{ id: string }>(request, '/api/projects', token, {
      name: 'PW Reorder ' + Date.now(),
    });

    const task1 = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW First Task', summary: 'first', priority: 'medium',
      },
    );
    const task2 = await apiPost<{ id: string }>(
      request, `/api/projects/${proj.id}/tasks`, token, {
        title: 'PW Second Task', summary: 'second', priority: 'medium',
      },
    );

    const got1 = await apiGet<{ position: number }>(request, `/api/projects/${proj.id}/tasks/${task1.id}`, token);
    const got2 = await apiGet<{ position: number }>(request, `/api/projects/${proj.id}/tasks/${task2.id}`, token);
    expect(got1.position).toBeLessThan(got2.position);

    // Reorder task2 to position 0
    const reorderResp = await request.post(`/api/projects/${proj.id}/tasks/${task2.id}/reorder`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { position: 0 },
    });
    expect(reorderResp.ok()).toBeTruthy();

    const after1 = await apiGet<{ position: number }>(request, `/api/projects/${proj.id}/tasks/${task1.id}`, token);
    const after2 = await apiGet<{ position: number }>(request, `/api/projects/${proj.id}/tasks/${task2.id}`, token);
    expect(after2.position).toBeLessThan(after1.position);

    await apiDelete(request, `/api/projects/${proj.id}`, token);
  });
});
