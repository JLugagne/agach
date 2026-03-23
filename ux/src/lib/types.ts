// JSend response wrapper
export interface JSendResponse<T> {
  status: 'success' | 'fail' | 'error';
  data?: T;
  error?: { code: string; message: string };
}

// Projects
export interface ProjectResponse {
  id: string;
  parent_id: string | null;
  name: string;
  description: string;
  git_url: string;
  created_by_role: string;
  created_by_agent: string;
  default_role: string;
  dockerfile_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface ProjectSummaryResponse {
  backlog_count: number;
  todo_count: number;
  in_progress_count: number;
  done_count: number;
  blocked_count: number;
}

export interface ProjectWithSummary extends ProjectResponse {
  summary: ProjectSummaryResponse;
  // Domain model fields (from direct domain serialization)
  children_count?: number;
  task_summary?: ProjectSummaryResponse;
}

export interface CreateProjectRequest {
  name: string;
  description?: string;
  git_url?: string;
  dockerfile_id?: string;
  roles?: string[];
  parent_id?: string;
  created_by_role?: string;
  created_by_agent?: string;
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
  git_url?: string;
  default_role?: string;
}

// Agents
export interface AgentResponse {
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

// Skills
export interface SkillResponse {
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

export interface CreateSkillRequest {
  slug: string;
  name: string;
  description?: string;
  content?: string;
  icon?: string;
  color?: string;
  sort_order?: number;
}

export interface UpdateSkillRequest {
  name?: string;
  description?: string;
  content?: string;
  icon?: string;
  color?: string;
  sort_order?: number;
}

export interface AddSkillToAgentRequest {
  skill_slug: string;
}

export interface CreateAgentRequest {
  slug: string;
  name: string;
  icon?: string;
  color?: string;
  description?: string;
  tech_stack?: string[];
  prompt_hint?: string;
  prompt_template?: string;
  sort_order?: number;
  skill_slugs?: string[];
}

export interface UpdateAgentRequest {
  name?: string;
  icon?: string;
  color?: string;
  description?: string;
  tech_stack?: string[];
  prompt_hint?: string;
  prompt_template?: string;
  sort_order?: number;
  skill_slugs?: string[];
}

export interface CloneAgentRequest {
  new_slug: string;
  new_name?: string;
}

// Tasks
export interface TaskResponse {
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
  blocked_at: string | null;
  blocked_by_agent: string;
  wont_do_requested: boolean;
  wont_do_reason: string;
  wont_do_requested_by: string;
  wont_do_requested_at: string | null;
  completion_summary: string;
  completed_by_agent: string;
  completed_at: string | null;
  files_modified: string[];
  resolution: string;
  context_files: string[];
  tags: string[];
  estimated_effort: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  model: string;
  seen_at: string | null;
  started_at: string | null;
  duration_seconds: number;
  human_estimate_seconds: number;
  session_id?: string;
  created_at: string;
  updated_at: string;
}

export interface TaskWithDetailsResponse extends TaskResponse {
  has_unresolved_deps: boolean;
  comment_count: number;
  project_id?: string;
  project_name?: string;
}

export interface CreateTaskRequest {
  title: string;
  summary: string;
  description?: string;
  priority?: string;
  created_by_role?: string;
  created_by_agent?: string;
  assigned_role?: string;
  context_files?: string[];
  tags?: string[];
  estimated_effort?: string;
  depends_on?: string[];
  start_in_backlog?: boolean;
  feature_id?: string | null;
}

export interface UpdateTaskRequest {
  title?: string;
  summary?: string;
  description?: string;
  priority?: string;
  assigned_role?: string;
  context_files?: string[];
  tags?: string[];
  estimated_effort?: string;
  resolution?: string;
  feature_id?: string | null;
}

// Features
export type FeatureStatus = 'draft' | 'ready' | 'in_progress' | 'done' | 'blocked';

export interface FeatureResponse {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: FeatureStatus;
  created_by_role: string;
  created_by_agent: string;
  created_at: string;
  updated_at: string;
}

export interface FeatureWithSummaryResponse extends FeatureResponse {
  task_summary: ProjectSummaryResponse;
}

export interface CreateFeatureRequest {
  name: string;
  description?: string;
  created_by_role?: string;
  created_by_agent?: string;
}

export interface UpdateFeatureRequest {
  name?: string;
  description?: string;
}

export interface UpdateFeatureStatusRequest {
  status: FeatureStatus;
}

export interface MoveTaskRequest {
  target_column: string;
  reason?: string;
}

export interface CompleteTaskRequest {
  completion_summary: string;
  files_modified?: string[];
  completed_by_agent: string;
}

export interface BlockTaskRequest {
  blocked_reason: string;
  blocked_by_agent: string;
}

export interface RequestWontDoRequest {
  wont_do_reason: string;
  wont_do_requested_by: string;
}

export interface RejectWontDoRequest {
  reason: string;
}

// Comments
export interface CommentResponse {
  id: string;
  task_id: string;
  author_role: string;
  author_name: string;
  author_type: string;
  content: string;
  edited_at: string | null;
  created_at: string;
}

export interface CreateCommentRequest {
  author_role: string;
  author_name?: string;
  content: string;
  mark_as_wont_do?: boolean;
}

export interface UpdateCommentRequest {
  content: string;
}

// Columns & Board
export interface ColumnResponse {
  id: string;
  slug: string;
  name: string;
  position: number;
  created_at: string;
}

export interface ColumnWithTasksResponse extends ColumnResponse {
  tasks: TaskWithDetailsResponse[];
}

export interface BoardResponse {
  columns: ColumnWithTasksResponse[];
}

// Dependencies
export interface AddDependencyRequest {
  depends_on_task_id: string;
}

export interface DependencyContextResponse {
  task_id: string;
  title: string;
  completion_summary: string;
  files_modified: string[];
}

export interface TaskDependentResponse {
  task_id: string;
  title: string;
  column_slug: string;
}

// Tool Usage / Statistics
export interface ToolUsageStatResponse {
  tool_name: string;
  execution_count: number;
  last_executed_at: string | null;
}

export interface TimelineEntryResponse {
  date: string;
  tasks_created: number;
  tasks_completed: number;
}

// Model token stats
export interface ModelTokenStatResponse {
  model: string;
  task_count: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// Model pricing
export interface ModelPricingResponse {
  id: string;
  model_id: string;
  input_price_per_1m: number;
  output_price_per_1m: number;
  cache_read_price_per_1m: number;
  cache_write_price_per_1m: number;
  updated_at: string;
}

// Feature stats
export interface FeatureStatsResponse {
  total_count: number;
  not_ready_count: number;
  ready_count: number;
  in_progress_count: number;
  done_count: number;
  blocked_count: number;
}

// Project-agent management
export interface AssignAgentToProjectRequest {
  agent_slug: string;
}

export interface RemoveAgentFromProjectRequest {
  reassign_to?: string;
  clear_assignment?: boolean;
}

export interface BulkReassignTasksRequest {
  old_slug: string;
  new_slug: string;
}

export interface BulkReassignTasksResponse {
  updated_count: number;
}

export interface TasksByAgentResponse {
  agent_slug: string;
  task_count: number;
  tasks: TaskResponse[];
}

// Dockerfiles
export interface DockerfileResponse {
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

export interface CreateDockerfileRequest {
  slug: string;
  name: string;
  description?: string;
  version: string;
  content?: string;
  is_latest?: boolean;
  sort_order?: number;
}

export interface UpdateDockerfileRequest {
  name?: string;
  description?: string;
  content?: string;
  is_latest?: boolean;
  sort_order?: number;
}

export interface SetProjectDockerfileRequest {
  dockerfile_id: string;
}

// WebSocket
export interface WSEvent {
  type: string;
  project_id?: string;
  data: unknown;
}
