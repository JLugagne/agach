import type {
  JSendResponse, ProjectResponse, ProjectWithSummary, ProjectSummaryResponse,
  CreateProjectRequest, RoleResponse, CreateRoleRequest, UpdateRoleRequest, CloneRoleRequest,
  TaskResponse, TaskWithDetailsResponse, CreateTaskRequest, UpdateTaskRequest,
  MoveTaskRequest, CompleteTaskRequest, BlockTaskRequest, RequestWontDoRequest,
  RejectWontDoRequest, CommentResponse, CreateCommentRequest, UpdateCommentRequest,
  BoardResponse, ColumnResponse, AddDependencyRequest,
  UpdateProjectRequest, ToolUsageStatResponse, TimelineEntryResponse,
  SkillResponse, CreateSkillRequest, UpdateSkillRequest, AddSkillToAgentRequest,
  AssignAgentToProjectRequest, RemoveAgentFromProjectRequest,
  BulkReassignTasksRequest, BulkReassignTasksResponse, TasksByAgentResponse,
} from './types';

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const opts: RequestInit = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (!res.ok) {
    const err: JSendResponse<unknown> = await res.json().catch(() => ({
      status: 'error' as const, error: { code: 'UNKNOWN', message: res.statusText },
    }));
    throw new Error(err.error?.message || res.statusText);
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') return undefined as T;
  const json: JSendResponse<T> = await res.json();
  if (json.status !== 'success') throw new Error(json.error?.message || 'Request failed');
  return json.data as T;
}

// Projects
export const listProjects = () => request<ProjectWithSummary[]>('GET', '/api/projects');
export const getProject = (id: string) => request<ProjectResponse>('GET', `/api/projects/${id}`);
export const getProjectInfo = (id: string) => request<ProjectResponse>('GET', `/api/projects/${id}/info`);
export const getProjectSummary = (id: string) => request<ProjectSummaryResponse>('GET', `/api/projects/${id}/summary`);
export const listSubProjects = (id: string) => request<ProjectWithSummary[]>('GET', `/api/projects/${id}/children`);
export const createProject = (data: CreateProjectRequest) => request<ProjectResponse>('POST', '/api/projects', data);
export const updateProject = (id: string, data: UpdateProjectRequest) => request<ProjectResponse>('PATCH', `/api/projects/${id}`, data);
export const deleteProject = (id: string) => request<void>('DELETE', `/api/projects/${id}`);

// Board
export const getBoard = (projectId: string, doneSince?: string, includeChildren?: boolean, search?: string) => {
  const params = new URLSearchParams();
  if (doneSince) params.set('done_since', doneSince);
  if (includeChildren) params.set('include_children', 'true');
  if (search) params.set('search', search);
  const qs = params.toString() ? `?${params.toString()}` : '';
  return request<BoardResponse>('GET', `/api/projects/${projectId}/board${qs}`);
};
export const listColumns = (projectId: string) => request<ColumnResponse[]>('GET', `/api/projects/${projectId}/columns`);
export const updateColumnWIPLimit = (projectId: string, slug: string, wipLimit: number) => request<void>('PATCH', `/api/projects/${projectId}/columns/${slug}/wip-limit`, { wip_limit: wipLimit });

// Tasks
export const listTasks = (projectId: string, params?: Record<string, string>) => {
  const qs = params ? '?' + new URLSearchParams(params).toString() : '';
  return request<TaskWithDetailsResponse[]>('GET', `/api/projects/${projectId}/tasks${qs}`);
};
export const getTask = (projectId: string, taskId: string) => request<TaskWithDetailsResponse>('GET', `/api/projects/${projectId}/tasks/${taskId}`);
export const createTask = (projectId: string, data: CreateTaskRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks`, data);
export const updateTask = (projectId: string, taskId: string, data: UpdateTaskRequest) => request<TaskResponse>('PATCH', `/api/projects/${projectId}/tasks/${taskId}`, data);
export const deleteTask = (projectId: string, taskId: string) => request<void>('DELETE', `/api/projects/${projectId}/tasks/${taskId}`);
export const moveTask = (projectId: string, taskId: string, data: MoveTaskRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/move`, data);
export const completeTask = (projectId: string, taskId: string, data: CompleteTaskRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/complete`, data);
export const blockTask = (projectId: string, taskId: string, data: BlockTaskRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/block`, data);
export const unblockTask = (projectId: string, taskId: string) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/unblock`, {});
export const markWontDo = (projectId: string, taskId: string, data: RequestWontDoRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/wont-do`, data);
export const approveWontDo = (projectId: string, taskId: string) => request<void>('POST', `/api/projects/${projectId}/tasks/${taskId}/approve-wont-do`, {});
export const rejectWontDo = (projectId: string, taskId: string, data: RejectWontDoRequest) => request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/reject-wont-do`, data);
export async function markTaskSeen(projectId: string, taskId: string): Promise<void> {
  await fetch(`/api/projects/${projectId}/tasks/${taskId}/seen`, { method: 'POST' });
}
export const reorderTask = (projectId: string, taskId: string, position: number) =>
  request<void>('POST', `/api/projects/${projectId}/tasks/${taskId}/reorder`, { position });
export const moveTaskToProject = (projectId: string, taskId: string, targetProjectId: string) =>
  request<TaskResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/move-to-project`, { target_project_id: targetProjectId });

// Roles (global)
export const listRoles = () => request<RoleResponse[]>('GET', '/api/roles');
export const getRole = (slug: string) => request<RoleResponse>('GET', `/api/roles/${slug}`);
export const createRole = (data: CreateRoleRequest) => request<RoleResponse>('POST', '/api/roles', data);
export const updateRole = (slug: string, data: UpdateRoleRequest) => request<RoleResponse>('PATCH', `/api/roles/${slug}`, data);
export const deleteRole = (slug: string) => request<void>('DELETE', `/api/roles/${slug}`);
export const cloneRole = (slug: string, data: CloneRoleRequest) =>
  request<RoleResponse>('POST', `/api/roles/${slug}/clone`, data);

// Roles (per-project)
export const listProjectRoles = (projectId: string) => request<RoleResponse[]>('GET', `/api/projects/${projectId}/roles`);
export const createProjectRole = (projectId: string, data: CreateRoleRequest) => request<RoleResponse>('POST', `/api/projects/${projectId}/roles`, data);
export const updateProjectRole = (projectId: string, slug: string, data: UpdateRoleRequest) => request<RoleResponse>('PATCH', `/api/projects/${projectId}/roles/${slug}`, data);
export const deleteProjectRole = (projectId: string, slug: string) => request<void>('DELETE', `/api/projects/${projectId}/roles/${slug}`);

// Comments
export const listComments = (projectId: string, taskId: string) => request<CommentResponse[]>('GET', `/api/projects/${projectId}/tasks/${taskId}/comments`);
export const createComment = (projectId: string, taskId: string, data: CreateCommentRequest) => request<CommentResponse>('POST', `/api/projects/${projectId}/tasks/${taskId}/comments`, data);
export const updateComment = (projectId: string, taskId: string, commentId: string, data: UpdateCommentRequest) => request<CommentResponse>('PATCH', `/api/projects/${projectId}/tasks/${taskId}/comments/${commentId}`, data);
export const deleteComment = (projectId: string, taskId: string, commentId: string) => request<void>('DELETE', `/api/projects/${projectId}/tasks/${taskId}/comments/${commentId}`);

// Images
export async function uploadImage(projectId: string, file: File): Promise<{ url: string }> {
  const form = new FormData();
  form.append('image', file);
  const res = await fetch(`/api/projects/${projectId}/images`, { method: 'POST', body: form });
  if (!res.ok) throw new Error('Image upload failed');
  const json = await res.json();
  return json.data as { url: string };
}

// Statistics
export const getToolUsage = (projectId: string) => request<ToolUsageStatResponse[]>('GET', `/api/projects/${projectId}/tool-usage`);
export const getColdStartStats = (projectId: string) => request<unknown[]>('GET', `/api/projects/${projectId}/stats/cold-start`);
export const getTimeline = (projectId: string, days?: number) => {
  const params = days ? `?days=${days}` : '';
  return request<TimelineEntryResponse[]>('GET', `/api/projects/${projectId}/stats/timeline${params}`);
};

// Skills (global)
export const listSkills = () => request<SkillResponse[]>('GET', '/api/skills');
export const getSkill = (slug: string) => request<SkillResponse>('GET', `/api/skills/${slug}`);
export const createSkill = (data: CreateSkillRequest) => request<SkillResponse>('POST', '/api/skills', data);
export const updateSkill = (slug: string, data: UpdateSkillRequest) => request<SkillResponse>('PATCH', `/api/skills/${slug}`, data);
export const deleteSkill = (slug: string) => request<void>('DELETE', `/api/skills/${slug}`);

// Agent-skill assignments
export const listAgentSkills = (agentSlug: string) =>
  request<SkillResponse[]>('GET', `/api/roles/${agentSlug}/skills`);
export const addSkillToAgent = (agentSlug: string, data: AddSkillToAgentRequest) =>
  request<void>('POST', `/api/roles/${agentSlug}/skills`, data);
export const removeSkillFromAgent = (agentSlug: string, skillSlug: string) =>
  request<void>('DELETE', `/api/roles/${agentSlug}/skills/${skillSlug}`);

// Project agent management
export const listProjectAgents = (projectId: string) =>
  request<RoleResponse[]>('GET', `/api/projects/${projectId}/agents`);
export const assignAgentToProject = (projectId: string, data: AssignAgentToProjectRequest) =>
  request<void>('POST', `/api/projects/${projectId}/agents`, data);
export const removeAgentFromProject = (projectId: string, agentSlug: string, data: RemoveAgentFromProjectRequest) =>
  request<void>('DELETE', `/api/projects/${projectId}/agents/${agentSlug}`, data);
export const getTasksByAgent = (projectId: string, agentSlug: string) =>
  request<TasksByAgentResponse>('GET', `/api/projects/${projectId}/agents/${agentSlug}/tasks`);
export const bulkReassignTasks = (projectId: string, data: BulkReassignTasksRequest) =>
  request<BulkReassignTasksResponse>('POST', `/api/projects/${projectId}/agents/bulk-reassign`, data);

// Dependencies
export const addDependency = (projectId: string, taskId: string, data: AddDependencyRequest) => request<unknown>('POST', `/api/projects/${projectId}/tasks/${taskId}/dependencies`, data);
export const removeDependency = (projectId: string, taskId: string, depId: string) => request<void>('DELETE', `/api/projects/${projectId}/tasks/${taskId}/dependencies/${depId}`);
export const listDependencies = (projectId: string, taskId: string) => request<TaskResponse[]>('GET', `/api/projects/${projectId}/tasks/${taskId}/dependencies`);
export const listDependents = (projectId: string, taskId: string) => request<TaskResponse[]>('GET', `/api/projects/${projectId}/tasks/${taskId}/dependents`);
