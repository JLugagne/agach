import { APIRequestContext, Page } from '@playwright/test';
import * as fs from 'fs';

export const BASE_URL = process.env.BASE_URL ?? 'http://localhost:8322';

// JSend response wrapper
interface JSendResponse<T> {
  status: 'success' | 'fail' | 'error';
  data?: T;
  error?: { code: string; message: string };
}

interface ProjectResponse {
  id: string;
  parent_id: string | null;
  name: string;
  description: string;
  created_by_role: string;
  created_by_agent: string;
  default_role: string;
  created_at: string;
  updated_at: string;
}

interface AgentResponse {
  id: string;
  slug: string;
  name: string;
  icon: string;
  color: string;
  description: string;
  tech_stack: string[];
  prompt_hint: string;
  prompt_template: string;
  content: string;
  skill_count: number;
  sort_order: number;
  created_at: string;
}

// Backward compatibility alias
type RoleResponse = AgentResponse;

interface DockerfileResponse {
  id: string;
  slug: string;
  name: string;
  description: string;
  version: string;
  content: string;
  is_latest: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

interface SkillResponse {
  id: string;
  slug: string;
  name: string;
  description: string;
  content: string;
  icon: string;
  color: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

interface TaskResponse {
  id: string;
  column_id: string;
  feature_id: string | null;
  title: string;
  summary: string;
  description: string;
  priority: string;
  priority_score: number;
  position: number;
  created_by_role: string;
  created_by_agent: string;
  assigned_role: string;
  is_blocked: boolean;
  blocked_reason: string;
  wont_do_requested: boolean;
  completion_summary: string;
  completed_by_agent: string;
  completed_at: string | null;
  files_modified: string[];
  resolution: string;
  context_files: string[];
  tags: string[];
  estimated_effort: string;
  created_at: string;
  updated_at: string;
}

interface ColumnWithTasksResponse {
  id: string;
  slug: string;
  name: string;
  position: number;
  wip_limit: number;
  created_at: string;
  tasks: TaskResponse[];
}

interface BoardResponse {
  columns: ColumnWithTasksResponse[];
}

interface AuthResponse {
  access_token: string;
  user: Record<string, unknown>;
}

// Token cache: use in-memory first, then fall back to reading from the auth-state file
// written by global-setup. This avoids re-logging in on every spec file.
let cachedToken: string | null = null;

function readTokenFromAuthState(): string | null {
  try {
    const authStatePath = '/tmp/auth-state.json';
    if (!fs.existsSync(authStatePath)) return null;
    const raw = fs.readFileSync(authStatePath, 'utf8');
    const state = JSON.parse(raw) as { origins?: { origin: string; localStorage: { name: string; value: string }[] }[] };
    for (const origin of state.origins ?? []) {
      for (const item of origin.localStorage ?? []) {
        if (item.name === 'agach_access_token') return item.value;
      }
    }
  } catch {
    // ignore
  }
  return null;
}

async function getAuthToken(request: APIRequestContext): Promise<string> {
  if (cachedToken) return cachedToken;

  // Try reading from the global-setup auth state file first (avoids rate limits)
  const fromFile = readTokenFromAuthState();
  if (fromFile) {
    cachedToken = fromFile;
    return cachedToken;
  }

  // Fall back to logging in
  const url = `${BASE_URL}/api/auth/login`;
  const res = await request.fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    data: { email: 'admin@agach.local', password: 'admin' },
  });
  const text = await res.text();
  let json: JSendResponse<AuthResponse>;
  try {
    json = JSON.parse(text) as JSendResponse<AuthResponse>;
  } catch {
    throw new Error(`Authentication failed: non-JSON response (status ${res.status()}): ${text.slice(0, 200)}`);
  }
  if (json.status !== 'success' || !json.data) {
    throw new Error(`Authentication failed: status=${json.status} error=${JSON.stringify(json.error)} body=${text.slice(0, 200)}`);
  }
  cachedToken = json.data.access_token;
  return cachedToken;
}

async function apiRequest<T>(
  request: APIRequestContext,
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const token = await getAuthToken(request);
  const url = `${BASE_URL}${path}`;
  const options: Parameters<APIRequestContext['fetch']>[1] = {
    method,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };
  if (body !== undefined) {
    options.data = body;
  }

  const res = await request.fetch(url, options);

  if (res.status() === 204) return undefined as T;

  const json: JSendResponse<T> = await res.json();
  if (json.status !== 'success') {
    throw new Error(json.error?.message || `Request failed: ${method} ${path}`);
  }
  return json.data as T;
}

// Projects

export async function createProject(
  request: APIRequestContext,
  name: string,
  description?: string,
): Promise<string> {
  const body: Record<string, unknown> = { name };
  if (description !== undefined) body.description = description;
  const project = await apiRequest<ProjectResponse>(request, 'POST', '/api/projects', body);
  return project.id;
}

export async function deleteProject(
  request: APIRequestContext,
  projectId: string,
): Promise<void> {
  await apiRequest<void>(request, 'DELETE', `/api/projects/${projectId}`);
}

// Tasks

export async function createTask(
  request: APIRequestContext,
  projectId: string,
  title: string,
  summary: string,
  opts?: {
    description?: string;
    priority?: string;
    assigned_role?: string;
    created_by_role?: string;
    created_by_agent?: string;
    context_files?: string[];
    tags?: string[];
    estimated_effort?: string;
    depends_on?: string[];
    start_in_backlog?: boolean;
    feature_id?: string | null;
  },
): Promise<string> {
  const body: Record<string, unknown> = { title, summary, ...opts };
  const task = await apiRequest<TaskResponse>(
    request,
    'POST',
    `/api/projects/${projectId}/tasks`,
    body,
  );
  return task.id;
}

// Agents (formerly Roles)

export async function createAgent(
  request: APIRequestContext,
  slug: string,
  name: string,
  opts?: {
    icon?: string;
    color?: string;
    description?: string;
    tech_stack?: string[];
    prompt_hint?: string;
    prompt_template?: string;
    sort_order?: number;
  },
): Promise<AgentResponse> {
  const body: Record<string, unknown> = { slug, name, ...opts };
  return apiRequest<AgentResponse>(request, 'POST', '/api/agents', body);
}

export async function deleteAgent(
  request: APIRequestContext,
  slug: string,
): Promise<void> {
  await apiRequest<void>(request, 'DELETE', `/api/agents/${slug}`);
}

// Backward compatibility aliases
export const createRole = createAgent;
export const deleteRole = deleteAgent;

// Features

export async function createFeature(
  request: APIRequestContext,
  parentProjectId: string,
  name: string,
  description?: string,
): Promise<string> {
  const body: Record<string, unknown> = { name, parent_id: parentProjectId };
  if (description !== undefined) body.description = description;
  const feature = await apiRequest<ProjectResponse>(request, 'POST', '/api/projects', body);
  return feature.id;
}

// Skills

export async function createSkill(
  request: APIRequestContext,
  slug: string,
  name: string,
  opts?: {
    description?: string;
    content?: string;
    icon?: string;
    color?: string;
    sort_order?: number;
  },
): Promise<SkillResponse> {
  const body: Record<string, unknown> = { slug, name, ...opts };
  return apiRequest<SkillResponse>(request, 'POST', '/api/skills', body);
}

export async function deleteSkill(
  request: APIRequestContext,
  slug: string,
): Promise<void> {
  await apiRequest<void>(request, 'DELETE', `/api/skills/${slug}`);
}

// Dockerfiles

export async function createDockerfile(
  request: APIRequestContext,
  slug: string,
  name: string,
  version: string,
  opts?: {
    description?: string;
    content?: string;
    is_latest?: boolean;
    sort_order?: number;
  },
): Promise<DockerfileResponse> {
  const body: Record<string, unknown> = { slug, name, version, ...opts };
  return apiRequest<DockerfileResponse>(request, 'POST', '/api/dockerfiles', body);
}

export async function deleteDockerfile(
  request: APIRequestContext,
  id: string,
): Promise<void> {
  await apiRequest<void>(request, 'DELETE', `/api/dockerfiles/${id}`);
}

// Task operations

export async function moveTask(
  request: APIRequestContext,
  projectId: string,
  taskId: string,
  targetColumn: string,
  reason?: string,
): Promise<TaskResponse> {
  const body: Record<string, unknown> = { target_column: targetColumn };
  if (reason !== undefined) body.reason = reason;
  return apiRequest<TaskResponse>(
    request,
    'POST',
    `/api/projects/${projectId}/tasks/${taskId}/move`,
    body,
  );
}

export async function getBoard(
  request: APIRequestContext,
  projectId: string,
): Promise<BoardResponse> {
  return apiRequest<BoardResponse>(request, 'GET', `/api/projects/${projectId}/board`);
}

// Auth

export async function loginAdmin(page: Page): Promise<void> {
  // If the login page is visible, log in; otherwise do nothing.
  const loginInput = page.locator('input[type="email"], input[name="email"]');
  const isLoginPage = await loginInput.isVisible({ timeout: 2000 }).catch(() => false);
  if (!isLoginPage) return;

  await loginInput.fill('admin@agach.local');
  const passwordInput = page.locator('input[type="password"]');
  await passwordInput.fill('admin');
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('login'), { timeout: 5000 });
}
