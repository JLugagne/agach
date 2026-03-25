import type {
  JSendResponse, ProjectResponse, ProjectWithSummary, ProjectSummaryResponse,
  CreateProjectRequest, AgentResponse, CreateAgentRequest, UpdateAgentRequest, CloneAgentRequest,
  TaskResponse, TaskWithDetailsResponse, CreateTaskRequest, UpdateTaskRequest,
  MoveTaskRequest, CompleteTaskRequest, BlockTaskRequest, RequestWontDoRequest,
  RejectWontDoRequest, CommentResponse, CreateCommentRequest, UpdateCommentRequest,
  BoardResponse, ColumnResponse, AddDependencyRequest,
  UpdateProjectRequest, ToolUsageStatResponse, TimelineEntryResponse,
  SkillResponse, CreateSkillRequest, UpdateSkillRequest, AddSkillToAgentRequest,
  AssignAgentToProjectRequest, RemoveAgentFromProjectRequest,
  BulkReassignTasksRequest, BulkReassignTasksResponse, TasksByAgentResponse,
  DockerfileResponse, CreateDockerfileRequest, UpdateDockerfileRequest, SetProjectDockerfileRequest,
  ModelTokenStatResponse, ModelPricingResponse, FeatureStatsResponse,
  FeatureResponse, FeatureWithSummaryResponse, CreateFeatureRequest, UpdateFeatureRequest, UpdateFeatureStatusRequest,
  NotificationResponse, UnreadCountResponse,
  NodeResponse, OnboardingCodeResponse, GenerateOnboardingCodeRequest,
  ChatSessionResponse,
} from './types';
import { refreshAccessToken, setToken } from './auth';

export const authEvents = new EventTarget();

let refreshPromise: Promise<string | null> | null = null;

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const token = localStorage.getItem('agach_access_token');
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const opts: RequestInit = { method, headers };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(path, opts);
  if (!res.ok) {
    if (res.status === 401 && token) {
      // Deduplicate concurrent refresh attempts.
      if (!refreshPromise) {
        refreshPromise = refreshAccessToken().finally(() => { refreshPromise = null; });
      }
      const newToken = await refreshPromise;
      if (newToken) {
        setToken(newToken);
        // Retry the original request with the new token.
        const retryHeaders: Record<string, string> = { 'Content-Type': 'application/json', Authorization: `Bearer ${newToken}` };
        const retryOpts: RequestInit = { method, headers: retryHeaders };
        if (body !== undefined) retryOpts.body = JSON.stringify(body);
        const retryRes = await fetch(path, retryOpts);
        if (!retryRes.ok) {
          if (retryRes.status === 401) authEvents.dispatchEvent(new Event('unauthorized'));
          const err: JSendResponse<unknown> = await retryRes.json().catch(() => ({
            status: 'error' as const, error: { code: 'UNKNOWN', message: retryRes.statusText },
          }));
          throw new Error(err.error?.message || retryRes.statusText);
        }
        if (retryRes.status === 204 || retryRes.headers.get('content-length') === '0') return undefined as T;
        const json: JSendResponse<T> = await retryRes.json();
        if (json.status !== 'success') throw new Error(json.error?.message || 'Request failed');
        return json.data as T;
      }
      authEvents.dispatchEvent(new Event('unauthorized'));
    } else if (res.status === 401) {
      authEvents.dispatchEvent(new Event('unauthorized'));
    }
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
export const listFeatures = (projectId: string, status?: string) => {
  const params = status ? `?status=${status}` : '';
  return request<FeatureWithSummaryResponse[]>('GET', `/api/projects/${projectId}/features${params}`);
};
export const getFeature = (projectId: string, featureId: string) =>
  request<FeatureResponse>('GET', `/api/projects/${projectId}/features/${featureId}`);
export const createFeature = (projectId: string, data: CreateFeatureRequest) =>
  request<FeatureResponse>('POST', `/api/projects/${projectId}/features`, data);
export const updateFeature = (projectId: string, featureId: string, data: UpdateFeatureRequest) =>
  request<FeatureResponse>('PATCH', `/api/projects/${projectId}/features/${featureId}`, data);
export const updateFeatureStatus = (projectId: string, featureId: string, data: UpdateFeatureStatusRequest) =>
  request<FeatureResponse>('PATCH', `/api/projects/${projectId}/features/${featureId}/status`, data);
export const deleteFeature = (projectId: string, featureId: string) =>
  request<void>('DELETE', `/api/projects/${projectId}/features/${featureId}`);
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

// Agents (global)
export const listAgents = () => request<AgentResponse[]>('GET', '/api/agents');
export const getAgent = (slug: string) => request<AgentResponse>('GET', `/api/agents/${slug}`);
export const createAgent = (data: CreateAgentRequest) => request<AgentResponse>('POST', '/api/agents', data);
export const updateAgent = (slug: string, data: UpdateAgentRequest) => request<AgentResponse>('PATCH', `/api/agents/${slug}`, data);
export const deleteAgent = (slug: string) => request<void>('DELETE', `/api/agents/${slug}`);
export const cloneAgent = (slug: string, data: CloneAgentRequest) =>
  request<AgentResponse>('POST', `/api/agents/${slug}/clone`, data);

// Agents (per-project)
export const listProjectAgents = (projectId: string) => request<AgentResponse[]>('GET', `/api/projects/${projectId}/agents`);
export const createProjectAgent = (projectId: string, data: CreateAgentRequest) => request<AgentResponse>('POST', `/api/projects/${projectId}/agents`, data);
export const updateProjectAgent = (projectId: string, slug: string, data: UpdateAgentRequest) => request<AgentResponse>('PATCH', `/api/projects/${projectId}/agents/${slug}`, data);
export const deleteProjectAgent = (projectId: string, slug: string) => request<void>('DELETE', `/api/projects/${projectId}/agents/${slug}`);

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

// Model stats
export const getModelTokenStats = (projectId: string) => request<ModelTokenStatResponse[]>('GET', `/api/projects/${projectId}/stats/model-tokens`);
export const getModelPricing = () => request<ModelPricingResponse[]>('GET', '/api/model-pricing');
export const getFeatureStats = (projectId: string) => request<FeatureStatsResponse>('GET', `/api/projects/${projectId}/stats/features`);

// Skills (global)
export const listSkills = () => request<SkillResponse[]>('GET', '/api/skills');
export const getSkill = (slug: string) => request<SkillResponse>('GET', `/api/skills/${slug}`);
export const createSkill = (data: CreateSkillRequest) => request<SkillResponse>('POST', '/api/skills', data);
export const updateSkill = (slug: string, data: UpdateSkillRequest) => request<SkillResponse>('PATCH', `/api/skills/${slug}`, data);
export const deleteSkill = (slug: string) => request<void>('DELETE', `/api/skills/${slug}`);

// Agent-skill assignments
export const listAgentSkills = (agentSlug: string) =>
  request<SkillResponse[]>('GET', `/api/agents/${agentSlug}/skills`);
export const addSkillToAgent = (agentSlug: string, data: AddSkillToAgentRequest) =>
  request<void>('POST', `/api/agents/${agentSlug}/skills`, data);
export const removeSkillFromAgent = (agentSlug: string, skillSlug: string) =>
  request<void>('DELETE', `/api/agents/${agentSlug}/skills/${skillSlug}`);

// Project agent management
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

// Dockerfiles (global)
export const listDockerfiles = () => request<DockerfileResponse[]>('GET', '/api/dockerfiles');
export const getDockerfile = (id: string) => request<DockerfileResponse>('GET', `/api/dockerfiles/${id}`);
export const createDockerfile = (data: CreateDockerfileRequest) => request<DockerfileResponse>('POST', '/api/dockerfiles', data);
export const updateDockerfile = (id: string, data: UpdateDockerfileRequest) => request<void>('PATCH', `/api/dockerfiles/${id}`, data);
export const deleteDockerfile = (id: string) => request<void>('DELETE', `/api/dockerfiles/${id}`);

// Account / profile
export const getMe = () => request<{ id: string; email: string; display_name: string; role: string; created_at: string }>('GET', '/api/auth/me');
export const updateProfile = (data: { display_name: string }) => request<{ id: string; email: string; display_name: string; role: string; created_at: string }>('PATCH', '/api/auth/me', data);
export const changePassword = (data: { current_password: string; new_password: string }) => request<void>('POST', '/api/auth/me/password', data);

// API Keys
export const listAPIKeys = () => request<{ id: string; name: string; scopes: string[]; expires_at: string | null; last_used_at: string | null; created_at: string }[]>('GET', '/api/auth/apikeys');
export const createAPIKey = (data: { name: string; scopes: string[]; expires_at?: string }) => request<{ api_key: string; id: string; name: string; scopes: string[]; expires_at: string | null; created_at: string }>('POST', '/api/auth/apikeys', data);
export const revokeAPIKey = (id: string) => request<void>('DELETE', `/api/auth/apikeys/${id}`);

// Project dockerfile assignment
export const getProjectDockerfile = (projectId: string) => request<DockerfileResponse | null>('GET', `/api/projects/${projectId}/dockerfile`);
export const setProjectDockerfile = (projectId: string, data: SetProjectDockerfileRequest) => request<void>('PUT', `/api/projects/${projectId}/dockerfile`, data);
export const clearProjectDockerfile = (projectId: string) => request<void>('DELETE', `/api/projects/${projectId}/dockerfile`);

// Notifications
export const listNotifications = (params?: { scope?: string; agent_slug?: string; unread?: boolean; limit?: number; offset?: number }) => {
  const qs = new URLSearchParams();
  if (params?.scope) qs.set('scope', params.scope);
  if (params?.agent_slug) qs.set('agent_slug', params.agent_slug);
  if (params?.unread) qs.set('unread', 'true');
  if (params?.limit) qs.set('limit', String(params.limit));
  if (params?.offset) qs.set('offset', String(params.offset));
  const q = qs.toString() ? `?${qs.toString()}` : '';
  return request<NotificationResponse[]>('GET', `/api/notifications${q}`);
};
export const listProjectNotifications = (projectId: string, params?: { scope?: string; agent_slug?: string; unread?: boolean; limit?: number; offset?: number }) => {
  const qs = new URLSearchParams();
  if (params?.scope) qs.set('scope', params.scope);
  if (params?.agent_slug) qs.set('agent_slug', params.agent_slug);
  if (params?.unread) qs.set('unread', 'true');
  if (params?.limit) qs.set('limit', String(params.limit));
  if (params?.offset) qs.set('offset', String(params.offset));
  const q = qs.toString() ? `?${qs.toString()}` : '';
  return request<NotificationResponse[]>('GET', `/api/projects/${projectId}/notifications${q}`);
};
export const getNotificationUnreadCount = (params?: { scope?: string; agent_slug?: string }) => {
  const qs = new URLSearchParams();
  if (params?.scope) qs.set('scope', params.scope);
  if (params?.agent_slug) qs.set('agent_slug', params.agent_slug);
  const q = qs.toString() ? `?${qs.toString()}` : '';
  return request<UnreadCountResponse>('GET', `/api/notifications/unread-count${q}`);
};
export const markNotificationRead = (id: string) => request<void>('PUT', `/api/notifications/${id}/read`);
export const markAllNotificationsRead = () => request<void>('PUT', `/api/notifications/read-all`);
export const markAllProjectNotificationsRead = (projectId: string) => request<void>('PUT', `/api/projects/${projectId}/notifications/read-all`);
export const deleteNotification = (id: string) => request<void>('DELETE', `/api/notifications/${id}`);

// Onboarding & Nodes
export const generateOnboardingCode = (data: GenerateOnboardingCodeRequest) =>
  request<OnboardingCodeResponse>('POST', '/api/onboarding/codes', data);
export const listNodes = () => request<{ nodes: NodeResponse[] }>('GET', '/api/nodes');
export const getNode = (nodeId: string) => request<{ node: NodeResponse }>('GET', `/api/nodes/${nodeId}`);
export const revokeNode = (nodeId: string) => request<void>('DELETE', `/api/nodes/${nodeId}`);
export const renameNode = (nodeId: string, name: string) =>
  request<{ node: NodeResponse }>('PATCH', `/api/nodes/${nodeId}/name`, { name });

// Chat sessions
export const listChatSessions = (projectId: string, featureId: string) =>
  request<ChatSessionResponse[]>('GET', `/api/projects/${projectId}/features/${featureId}/chats`);

export const getChatSession = (projectId: string, featureId: string, sessionId: string) =>
  request<ChatSessionResponse>('GET', `/api/projects/${projectId}/features/${featureId}/chats/${sessionId}`);

export const startChatSession = (projectId: string, featureId: string, resumeSessionId?: string) =>
  request<ChatSessionResponse>('POST', `/api/projects/${projectId}/features/${featureId}/chats`,
    resumeSessionId ? { resume_session_id: resumeSessionId } : {});

export const endChatSession = (projectId: string, featureId: string, sessionId: string) =>
  request<void>('POST', `/api/projects/${projectId}/features/${featureId}/chats/${sessionId}/end`);
