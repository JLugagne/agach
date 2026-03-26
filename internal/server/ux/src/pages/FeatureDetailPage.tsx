import { useState, useEffect, useCallback, useMemo } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  Loader2,
  ArrowLeft,
  Pencil,
  Trash2,
  Check,
  X,
  MessageSquare,
  CheckCircle2,
  Circle,
  Clock,
  AlertTriangle,
  Flag,
  Calendar,
  User,
  Timer,
  DollarSign,
} from 'lucide-react';
import {
  getFeature,
  updateFeature,
  updateFeatureStatus,
  deleteFeature,
  listTasks,
  getFeatureTaskSummaries,
  getModelPricing,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import MarkdownContent from '../components/ui/MarkdownContent';
import type { FeatureResponse, FeatureStatus, TaskWithDetailsResponse, TaskSummaryResponse, ModelPricingResponse } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';

type TabKey = 'overview' | 'tasks' | 'activity' | 'changelog' | 'review';

const STATUS_BADGE_COLORS: Record<FeatureStatus, string> = {
  draft: '#8B5CF6',
  ready: 'var(--status-todo)',
  in_progress: 'var(--status-progress)',
  done: 'var(--status-done)',
  blocked: '#FF3B30',
};

const ALL_STATUSES: { value: FeatureStatus; label: string }[] = [
  { value: 'draft', label: 'Draft' },
  { value: 'ready', label: 'Ready' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'done', label: 'Done' },
  { value: 'blocked', label: 'Blocked' },
];

// Task groups in display order
const TASK_GROUPS: { slug: string; label: string; dot: string; icon: React.ReactNode }[] = [
  { slug: 'in_progress', label: 'In Progress', dot: 'var(--status-progress)', icon: <Clock size={12} /> },
  { slug: 'todo', label: 'Todo', dot: 'var(--status-todo)', icon: <Circle size={12} /> },
  { slug: 'blocked', label: 'Blocked', dot: 'var(--status-blocked)', icon: <AlertTriangle size={12} /> },
  { slug: 'done', label: 'Done', dot: 'var(--status-done)', icon: <CheckCircle2 size={12} /> },
];

// Priority pill styles
const priorityStyles: Record<string, { text: string; bg: string; icon: string }> = {
  critical: { text: 'var(--priority-critical)', bg: 'var(--priority-critical-bg)', icon: '#EF4444' },
  high: { text: 'var(--priority-high)', bg: 'var(--priority-high-bg)', icon: '#F97316' },
  medium: { text: 'var(--priority-medium)', bg: 'var(--priority-medium-bg)', icon: '#FFB547' },
  low: { text: 'var(--priority-low)', bg: 'var(--priority-low-bg)', icon: '#6B7084' },
};

function getTaskSlug(task: TaskWithDetailsResponse): string {
  if (task.is_blocked) return 'blocked';
  if (task.completed_at) return 'done';
  if (task.started_at) return 'in_progress';
  return 'todo';
}

function formatDuration(seconds: number): string {
  if (seconds <= 0) return '0m';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

function computeCost(
  summary: TaskSummaryResponse,
  pricingMap: Record<string, ModelPricingResponse>,
): number {
  let pricing = pricingMap[summary.model];
  if (!pricing) {
    for (const [modelId, p] of Object.entries(pricingMap)) {
      if (summary.model.startsWith(modelId) || modelId.startsWith(summary.model)) {
        pricing = p;
        break;
      }
    }
  }
  if (!pricing) return 0;
  return (
    (summary.input_tokens / 1_000_000) * pricing.input_price_per_1m +
    (summary.output_tokens / 1_000_000) * pricing.output_price_per_1m +
    (summary.cache_read_tokens / 1_000_000) * pricing.cache_read_price_per_1m +
    (summary.cache_write_tokens / 1_000_000) * pricing.cache_write_price_per_1m
  );
}

export default function FeatureDetailPage() {
  const { projectId, featureId } = useParams<{ projectId: string; featureId: string }>();
  const navigate = useNavigate();
  const [feature, setFeature] = useState<FeatureResponse | null>(null);
  const [tasks, setTasks] = useState<TaskWithDetailsResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [tasksLoading, setTasksLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<TabKey>('overview');

  // Edit state
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [saving, setSaving] = useState(false);

  // Status change
  const [changingStatus, setChangingStatus] = useState(false);

  // Delete state
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  // Review tab state
  const [reviewSummaries, setReviewSummaries] = useState<TaskSummaryResponse[]>([]);
  const [reviewLoading, setReviewLoading] = useState(false);
  const [modelPricing, setModelPricing] = useState<ModelPricingResponse[]>([]);

  const pricingMap = useMemo(() => {
    const map: Record<string, ModelPricingResponse> = {};
    for (const p of modelPricing) map[p.model_id] = p;
    return map;
  }, [modelPricing]);

  const fetchFeature = useCallback(async () => {
    if (!projectId || !featureId) return;
    try {
      const data = await getFeature(projectId, featureId);
      setFeature(data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId, featureId]);

  const fetchTasks = useCallback(async () => {
    if (!projectId || !featureId) return;
    try {
      const data = await listTasks(projectId, { feature_id: featureId });
      setTasks(data ?? []);
    } catch {
      // ignore
    } finally {
      setTasksLoading(false);
    }
  }, [projectId, featureId]);

  useEffect(() => {
    fetchFeature();
    fetchTasks();
  }, [fetchFeature, fetchTasks]);

  // Fetch review data when review tab is activated
  useEffect(() => {
    if (activeTab !== 'review' || !projectId || !featureId) return;
    setReviewLoading(true);
    Promise.all([
      getFeatureTaskSummaries(projectId, featureId).catch(() => [] as TaskSummaryResponse[]),
      getModelPricing().catch(() => [] as ModelPricingResponse[]),
    ]).then(([summaries, pricing]) => {
      setReviewSummaries(summaries ?? []);
      setModelPricing(pricing ?? []);
    }).finally(() => setReviewLoading(false));
  }, [activeTab, projectId, featureId]);

  useWebSocket(
    useCallback(
      (event) => {
        const type = event.type || '';
        if (type.startsWith('task_') || type.startsWith('feature_')) {
          fetchFeature();
          fetchTasks();
        }
      },
      [fetchFeature, fetchTasks],
    ),
  );

  const handleSaveEdit = async () => {
    if (!feature || !projectId || !editName.trim()) return;
    setSaving(true);
    try {
      await updateFeature(projectId, feature.id, {
        name: editName.trim(),
        description: editDesc.trim(),
      });
      setEditing(false);
      fetchFeature();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleStatusChange = async (newStatus: FeatureStatus) => {
    if (!feature || !projectId) return;
    setChangingStatus(true);
    try {
      await updateFeatureStatus(projectId, feature.id, { status: newStatus });
      fetchFeature();
    } catch {
      // ignore
    } finally {
      setChangingStatus(false);
    }
  };

  const handleDelete = async () => {
    if (!feature || !projectId) return;
    setDeleting(true);
    try {
      await deleteFeature(projectId, feature.id);
      navigate(`/projects/${projectId}/features`);
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  // Group tasks by status
  const grouped: Record<string, TaskWithDetailsResponse[]> = {};
  for (const task of tasks) {
    const slug = getTaskSlug(task);
    if (!grouped[slug]) grouped[slug] = [];
    grouped[slug].push(task);
  }

  const todoCount = (grouped['todo']?.length ?? 0);
  const inProgressCount = (grouped['in_progress']?.length ?? 0);
  const doneCount = (grouped['done']?.length ?? 0);
  const totalTasks = tasks.length;
  const progressPct = totalTasks > 0 ? (doneCount / totalTasks) * 100 : 0;

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[var(--bg-primary)]">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  if (!feature || !projectId) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[var(--bg-primary)]">
        <p className="text-[var(--text-muted)] text-sm font-['Inter']">Feature not found.</p>
      </div>
    );
  }

  const statusColor = STATUS_BADGE_COLORS[feature.status] ?? 'var(--text-muted)';
  const prioStyle = priorityStyles[feature.status] ?? priorityStyles.medium;

  const tabs: { key: TabKey; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'tasks', label: 'Tasks' },
    { key: 'activity', label: 'Activity' },
    { key: 'changelog', label: 'Changelog' },
    { key: 'review', label: 'Review' },
  ];

  return (
    <div className="flex-1 overflow-y-auto bg-[#0D0F17]">
      <div className="py-8 px-10" style={{ fontFamily: 'Inter, sans-serif' }}>
        {/* Back link */}
        <Link
          to={`/projects/${projectId}/features`}
          data-qa="feature-detail-back-link"
          className="inline-flex items-center gap-1.5 text-[13px] font-medium text-[#8B8FA3] hover:text-[#B0B5C8] transition-colors mb-4"
        >
          <ArrowLeft size={16} />
          Back to Features
        </Link>

        {/* Header: Title Row */}
        <div className="flex items-center gap-4 mb-7">
          {editing ? (
            <div className="flex-1 space-y-3">
              <input
                type="text"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                data-qa="feature-detail-edit-name-input"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-lg text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                autoFocus
              />
              <textarea
                value={editDesc}
                onChange={(e) => setEditDesc(e.target.value)}
                rows={3}
                data-qa="feature-detail-edit-description-textarea"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
                placeholder="Description"
              />
              <div className="flex items-center gap-2">
                <button
                  onClick={handleSaveEdit}
                  disabled={!editName.trim() || saving}
                  data-qa="feature-detail-save-edit-btn"
                  className="flex items-center gap-1 px-3 py-1.5 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)] disabled:opacity-50 transition-colors"
                >
                  <Check size={14} />
                  {saving ? 'Saving...' : 'Save'}
                </button>
                <button
                  onClick={() => setEditing(false)}
                  data-qa="feature-detail-cancel-edit-btn"
                  className="flex items-center gap-1 px-3 py-1.5 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                >
                  <X size={14} />
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <>
              <h1 className="text-[28px] font-bold text-white tracking-[-0.8px]">
                {feature.name}
              </h1>
              {/* Status badge */}
              <span
                className="text-[11px] font-semibold px-2.5 py-1 rounded-lg"
                style={{
                  color: statusColor,
                  backgroundColor: `color-mix(in srgb, ${statusColor} 12%, transparent)`,
                }}
                data-qa="feature-detail-status-badge"
              >
                {feature.status.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
              </span>
              <div className="flex-1" />
              {/* Action buttons */}
              <div className="flex items-center gap-2">
                <button
                  onClick={() => navigate(`/projects/${projectId}/features/${featureId}/chat`)}
                  data-qa="feature-chat-btn"
                  className="flex items-center gap-1.5 px-3.5 py-2 rounded-[10px] bg-[#1E2030] text-[#8B8FA3] text-[13px] font-medium hover:text-white transition-colors"
                >
                  <MessageSquare size={16} />
                  Chat
                </button>
                <button
                  onClick={() => {
                    setEditName(feature.name);
                    setEditDesc(feature.description || '');
                    setEditing(true);
                  }}
                  data-qa="feature-detail-edit-btn"
                  className="flex items-center gap-1.5 px-3.5 py-2 rounded-[10px] bg-[#1E2030] text-[#8B8FA3] text-[13px] font-medium hover:text-white transition-colors"
                >
                  <Pencil size={16} />
                  Edit
                </button>
                <button
                  onClick={() => setShowDelete(true)}
                  data-qa="feature-detail-delete-btn"
                  className="flex items-center gap-1.5 px-3.5 py-2 rounded-[10px] bg-[#1E203080] text-[#E85A4F] text-[13px] font-medium hover:text-[#FF6B63] transition-colors"
                >
                  <Trash2 size={16} />
                  Delete
                </button>
              </div>
            </>
          )}
        </div>

        {/* Description Section */}
        {!editing && feature.description && (
          <div
            className="rounded-2xl bg-[#131520] border border-[#1E2030] p-6 mb-7"
            data-qa="feature-description-card"
          >
            <h3 className="text-white text-base font-semibold mb-3">Description</h3>
            <p className="text-[#8B8FA3] text-sm leading-[1.5]">
              {feature.description}
            </p>
          </div>
        )}

        {/* Divider */}
        <div className="h-px bg-[#1E2030] mb-7" />

        {/* Task Progress Card */}
        <div className="mb-7">
          <div className="rounded-2xl bg-[#131520] border border-[#1E2030] p-6">
            <div className="flex items-center mb-4">
              <h3 className="text-white text-base font-semibold">Task Progress</h3>
              <div className="flex-1" />
              <span className="text-[#6B7084] text-[13px] font-medium">
                {doneCount} / {totalTasks} tasks
              </span>
            </div>
            {/* Progress bar */}
            <div className="h-2 rounded bg-[#1E2030] mb-4">
              <div
                className="h-2 rounded"
                style={{
                  width: `${progressPct}%`,
                  background: 'linear-gradient(90deg, #7C3AED, #8B5CF6)',
                  transition: 'width 0.3s ease',
                }}
              />
            </div>
            {/* Stats row */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1.5">
                <div className="w-2 h-2 rounded-full bg-[#6B7084]" />
                <span className="text-[#6B7084] text-xs font-medium">{todoCount} To Do</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-2 h-2 rounded-full bg-[#E85A4F]" />
                <span className="text-[#6B7084] text-xs font-medium">{inProgressCount} In Progress</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-2 h-2 rounded-full bg-[#32D583]" />
                <span className="text-[#6B7084] text-xs font-medium">{doneCount} Done</span>
              </div>
            </div>
          </div>
        </div>

        {/* Status Pills */}
        <div className="flex items-center gap-2.5 flex-wrap mb-7">
          {/* Priority pill */}
          <button
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#1E2030] border border-[#2A2A2E] text-[#8B8FA3] text-xs font-medium cursor-default"
          >
            <Flag size={14} style={{ color: prioStyle.icon }} />
            {/* Use feature's first task priority or 'Medium' as default */}
            {tasks.length > 0 ? `${tasks[0].priority.charAt(0).toUpperCase() + tasks[0].priority.slice(1)} Priority` : 'No Priority'}
          </button>
          {/* Status pill */}
          <div
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs font-medium"
            style={{
              backgroundColor: `color-mix(in srgb, ${statusColor} 8%, transparent)`,
              borderColor: `color-mix(in srgb, ${statusColor} 25%, transparent)`,
              color: statusColor,
            }}
          >
            <div className="w-2 h-2 rounded-full" style={{ backgroundColor: statusColor }} />
            {feature.status.replace('_', ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
          </div>
          {/* Sprint pill - placeholder */}
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#1E2030] border border-[#2A2A2E] text-[#8B8FA3] text-xs font-medium">
            <Calendar size={14} />
            Sprint
          </div>
          {/* Owner pill */}
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-[#1E2030] border border-[#2A2A2E] text-[#8B8FA3] text-xs font-medium">
            <User size={14} />
            {feature.created_by_agent ? `Assigned to ${feature.created_by_agent}` : 'Unassigned'}
          </div>
        </div>

        {/* Divider */}
        <div className="h-px bg-[#1E2030] mb-0" />

        {/* Tabs Bar */}
        <div className="flex items-center border-b border-[#1E2030]">
          {tabs.map((tab) => {
            const isActive = activeTab === tab.key;
            return (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                data-qa={`feature-tab-${tab.key}`}
                className="relative px-4 py-3 text-sm font-medium transition-colors"
                style={{
                  color: isActive ? '#FFFFFF' : '#6B7084',
                  fontWeight: isActive ? 600 : 500,
                }}
              >
                {tab.label}
                {isActive && (
                  <div className="absolute bottom-0 left-4 right-4 h-0.5 rounded bg-[#7C3AED]" />
                )}
              </button>
            );
          })}
        </div>

        {/* Divider after tabs */}
        <div className="h-px bg-[#1E2030] mb-7" />

        {/* Tab Content */}
        {activeTab === 'overview' && (
          <div>
            {/* Status controls */}
            <div className="mb-8">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-2">Status</label>
              <div className="flex items-center gap-2 flex-wrap">
                {ALL_STATUSES.map((s) => {
                  const isActive = feature.status === s.value;
                  const sColor = STATUS_BADGE_COLORS[s.value];
                  return (
                    <button
                      key={s.value}
                      onClick={() => !isActive && handleStatusChange(s.value)}
                      disabled={changingStatus || isActive}
                      data-qa="feature-detail-status-btn"
                      className={`px-3 py-1.5 rounded-full text-[12px] font-medium transition-colors cursor-pointer border ${
                        isActive
                          ? 'border-current'
                          : 'border-[var(--border-primary)] hover:border-current'
                      }`}
                      style={{
                        color: isActive ? sColor : 'var(--text-muted)',
                        backgroundColor: isActive ? `color-mix(in srgb, ${sColor} 15%, transparent)` : 'var(--bg-tertiary)',
                        opacity: changingStatus ? 0.5 : 1,
                      }}
                    >
                      {s.label}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'tasks' && (
          <div>
            {tasksLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 size={20} className="animate-spin text-[var(--text-muted)]" />
              </div>
            ) : tasks.length === 0 ? (
              <p className="text-sm text-[var(--text-muted)] py-4 font-['Inter']">
                No tasks assigned to this feature.
              </p>
            ) : (
              <div className="space-y-6">
                {TASK_GROUPS.map((group) => {
                  const groupTasks = grouped[group.slug];
                  if (!groupTasks || groupTasks.length === 0) return null;
                  return (
                    <div key={group.slug}>
                      <div className="flex items-center gap-2 mb-2">
                        <div
                          className="w-2 h-2 rounded-full flex-shrink-0"
                          style={{ backgroundColor: group.dot }}
                        />
                        <span
                          className="text-[11px] font-bold uppercase tracking-wider"
                          style={{ color: group.dot, fontFamily: 'JetBrains Mono, monospace' }}
                        >
                          {group.label}
                        </span>
                        <span
                          className="text-[10px] font-mono px-1.5 py-0.5 rounded-md"
                          style={{
                            color: group.dot,
                            backgroundColor: `color-mix(in srgb, ${group.dot} 12%, transparent)`,
                          }}
                        >
                          {groupTasks.length}
                        </span>
                      </div>
                      <div className="space-y-1">
                        {groupTasks.map((task) => {
                          const prio = priorityStyles[task.priority] || priorityStyles.medium;
                          return (
                            <Link
                              key={task.id}
                              to={`/projects/${projectId}?task=${task.id}`}
                              data-qa="feature-detail-task-link"
                              className="flex items-center gap-3 px-3 py-2 rounded-md transition-colors hover:bg-[var(--bg-tertiary)] group"
                            >
                              <p className="text-sm text-[var(--text-primary)] truncate flex-1 min-w-0 group-hover:text-[var(--primary)] font-['Inter']">
                                {task.title}
                              </p>
                              <span
                                className="flex-shrink-0 px-1.5 py-[1px] rounded text-[9px] font-bold uppercase tracking-wider"
                                style={{ color: prio.text, backgroundColor: prio.bg, fontFamily: 'JetBrains Mono, monospace' }}
                              >
                                {task.priority}
                              </span>
                              {task.assigned_role && (
                                <span className="flex-shrink-0 text-[10px] px-1.5 py-0.5 rounded font-medium text-[var(--text-secondary)] bg-[var(--bg-tertiary)] truncate max-w-[80px] font-['JetBrains_Mono']">
                                  @{task.assigned_role}
                                </span>
                              )}
                            </Link>
                          );
                        })}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {activeTab === 'activity' && (
          <div className="py-4">
            <p className="text-sm text-[#6B7084] font-['Inter'] italic">Activity feed coming soon.</p>
          </div>
        )}

        {activeTab === 'changelog' && (
          <div className="flex gap-5">
            {/* User Changelog */}
            <div className="flex-1 rounded-2xl bg-[#131520] border border-[#1E2030] p-6">
              <h3 className="text-white text-lg font-semibold mb-5">User Changelog</h3>
              {feature.user_changelog?.trim() ? (
                <div className="text-[#8B8FA3] text-sm leading-[1.6]">
                  <MarkdownContent content={feature.user_changelog} />
                </div>
              ) : (
                <p className="text-[#6B7084] text-sm italic">No user changelog available</p>
              )}
            </div>
            {/* Technical Changelog */}
            <div className="flex-1 rounded-2xl bg-[#131520] border border-[#1E2030] p-6">
              <h3 className="text-white text-lg font-semibold mb-5">Technical Changelog</h3>
              {feature.tech_changelog?.trim() ? (
                <div className="text-[#8B8FA3] text-sm leading-[1.6]">
                  <MarkdownContent content={feature.tech_changelog} />
                </div>
              ) : (
                <p className="text-[#6B7084] text-sm italic">No technical changelog available</p>
              )}
            </div>
          </div>
        )}

        {activeTab === 'review' && (
          <div>
            {reviewLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 size={20} className="animate-spin text-[var(--text-muted)]" />
              </div>
            ) : reviewSummaries.length === 0 ? (
              <p className="text-sm text-[#6B7084] py-4 font-['Inter'] italic">
                No completed tasks to review.
              </p>
            ) : (
              <div className="flex flex-col gap-4">
                {reviewSummaries.map((summary) => {
                  const cost = computeCost(summary, pricingMap);
                  return (
                    <div
                      key={summary.task_id}
                      className="rounded-2xl bg-[#131520] border border-[#1E2030] p-5"
                      data-qa="review-card"
                    >
                      <h4 className="text-white text-base font-bold mb-3">
                        {summary.title}
                      </h4>
                      {summary.completion_summary && (
                        <p className="text-[#8B8FA3] text-[13px] leading-[1.5] mb-3">
                          {summary.completion_summary}
                        </p>
                      )}
                      <div className="flex items-center gap-3">
                        {summary.duration_seconds > 0 && (
                          <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-[#1E2030] text-[#6B7084] text-xs font-medium">
                            <Timer size={14} />
                            {formatDuration(summary.duration_seconds)}
                          </div>
                        )}
                        {cost > 0 && (
                          <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-[#1E2030] text-[#22C55E] text-xs font-medium">
                            <DollarSign size={14} />
                            ${cost.toFixed(2)}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        <DeleteConfirmModal
          open={showDelete}
          title="Delete Feature?"
          description={`This will permanently delete "${feature.name}" and unlink all its tasks.`}
          confirmLabel="Delete Feature"
          onConfirm={handleDelete}
          onCancel={() => setShowDelete(false)}
          loading={deleting}
        />
      </div>
    </div>
  );
}
