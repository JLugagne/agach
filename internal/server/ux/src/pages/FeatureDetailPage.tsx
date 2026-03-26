import { useState, useEffect, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  Loader2,
  ArrowLeft,
  Pencil,
  Trash2,
  Check,
  X,
  MessageCircle,
  CheckCircle2,
  Circle,
  Clock,
  AlertTriangle,
  BookOpen,
  FileText,
  ListChecks,
} from 'lucide-react';
import {
  getFeature,
  updateFeature,
  updateFeatureStatus,
  deleteFeature,
  listTasks,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import FeatureChangelogDrawer from '../components/kanban/FeatureChangelogDrawer';
import TaskSummariesDrawer from '../components/kanban/TaskSummariesDrawer';
import type { FeatureResponse, FeatureStatus, TaskWithDetailsResponse } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';

const STATUS_BADGE_COLORS: Record<FeatureStatus, string> = {
  draft: 'var(--text-muted)',
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
const priorityStyles: Record<string, { text: string; bg: string }> = {
  critical: { text: 'var(--priority-critical)', bg: 'var(--priority-critical-bg)' },
  high: { text: 'var(--priority-high)', bg: 'var(--priority-high-bg)' },
  medium: { text: 'var(--priority-medium)', bg: 'var(--priority-medium-bg)' },
  low: { text: 'var(--priority-low)', bg: 'var(--priority-low-bg)' },
};

// Derive column slug from task fields
function getTaskSlug(task: TaskWithDetailsResponse): string {
  if (task.is_blocked) return 'blocked';
  if (task.completed_at) return 'done';
  if (task.started_at) return 'in_progress';
  return 'todo';
}

export default function FeatureDetailPage() {
  const { projectId, featureId } = useParams<{ projectId: string; featureId: string }>();
  const navigate = useNavigate();
  const [feature, setFeature] = useState<FeatureResponse | null>(null);
  const [tasks, setTasks] = useState<TaskWithDetailsResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [tasksLoading, setTasksLoading] = useState(true);

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

  // Drawer state
  const [showUserChangelog, setShowUserChangelog] = useState(false);
  const [showTechChangelog, setShowTechChangelog] = useState(false);
  const [showTaskSummaries, setShowTaskSummaries] = useState(false);

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
        <p className="text-[var(--text-muted)] text-sm" style={{ fontFamily: 'Inter, sans-serif' }}>Feature not found.</p>
      </div>
    );
  }

  const statusColor = STATUS_BADGE_COLORS[feature.status] ?? 'var(--text-muted)';

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-4xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
        {/* Back link */}
        <Link
          to={`/projects/${projectId}/features`}
          data-qa="feature-detail-back-link"
          className="flex items-center gap-1.5 text-sm text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors mb-6"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          <ArrowLeft size={14} />
          Back to Features
        </Link>

        {/* Header */}
        <div className="flex items-start justify-between gap-2 mb-8">
          <div className="flex-1 min-w-0">
            {editing ? (
              <div className="space-y-3">
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  data-qa="feature-detail-edit-name-input"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-lg text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                  style={{ fontFamily: 'Newsreader, Georgia, serif' }}
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
                <div className="flex items-center gap-3 mb-2">
                  <h1
                    className="text-[28px] font-semibold text-[var(--text-primary)]"
                    style={{ fontFamily: 'Newsreader, Georgia, serif' }}
                  >
                    {feature.name}
                  </h1>
                  <span
                    className="text-[11px] px-2 py-0.5 rounded-full font-mono"
                    style={{
                      color: statusColor,
                      backgroundColor: `color-mix(in srgb, ${statusColor} 15%, transparent)`,
                    }}
                    data-qa="feature-detail-status-badge"
                  >
                    {feature.status.replace('_', ' ')}
                  </span>
                </div>
                {feature.description && (
                  <p className="text-sm text-[var(--text-secondary)] leading-relaxed" style={{ fontFamily: 'Inter, sans-serif' }}>
                    {feature.description}
                  </p>
                )}
              </>
            )}
          </div>
          {!editing && (
            <div className="flex items-center gap-2 ml-4 shrink-0">
              <button
                onClick={() => navigate(`/projects/${projectId}/features/${featureId}/chat`)}
                data-qa="feature-chat-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md hover:bg-[var(--bg-tertiary)]"
                title="Chat"
              >
                <MessageCircle size={16} />
              </button>
              <div className="w-px h-4 bg-[var(--border-primary)]" />
              <button
                onClick={() => setShowUserChangelog(true)}
                disabled={!feature.user_changelog}
                data-qa="feature-user-changelog-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md hover:bg-[var(--bg-tertiary)] disabled:opacity-40 disabled:pointer-events-none"
                title="User Changelog"
              >
                <BookOpen size={16} />
              </button>
              <button
                onClick={() => setShowTechChangelog(true)}
                disabled={!feature.tech_changelog}
                data-qa="feature-tech-changelog-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md hover:bg-[var(--bg-tertiary)] disabled:opacity-40 disabled:pointer-events-none"
                title="Technical Changelog"
              >
                <FileText size={16} />
              </button>
              <button
                onClick={() => setShowTaskSummaries(true)}
                data-qa="feature-task-summaries-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md hover:bg-[var(--bg-tertiary)]"
                title="Implementation Summary"
              >
                <ListChecks size={16} />
              </button>
              <div className="w-px h-4 bg-[var(--border-primary)]" />
              <button
                onClick={() => {
                  setEditName(feature.name);
                  setEditDesc(feature.description || '');
                  setEditing(true);
                }}
                data-qa="feature-detail-edit-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md hover:bg-[var(--bg-tertiary)]"
                title="Edit"
              >
                <Pencil size={16} />
              </button>
              <button
                onClick={() => setShowDelete(true)}
                data-qa="feature-detail-delete-btn"
                className="p-2 text-[var(--text-muted)] hover:text-[#FF3B30] transition-colors rounded-md hover:bg-[var(--bg-tertiary)]"
                title="Delete"
              >
                <Trash2 size={16} />
              </button>
            </div>
          )}
        </div>

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
                    fontFamily: 'Inter, sans-serif',
                    opacity: changingStatus ? 0.5 : 1,
                  }}
                >
                  {s.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Task list grouped by status */}
        <div>
          <h2
            className="text-lg font-semibold text-[var(--text-primary)] mb-4"
            style={{ fontFamily: 'Newsreader, Georgia, serif' }}
          >
            Tasks
          </h2>
          {tasksLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 size={20} className="animate-spin text-[var(--text-muted)]" />
            </div>
          ) : tasks.length === 0 ? (
            <p className="text-sm text-[var(--text-muted)] py-4" style={{ fontFamily: 'Inter, sans-serif' }}>
              No tasks assigned to this feature.
            </p>
          ) : (
            <div className="space-y-6">
              {TASK_GROUPS.map((group) => {
                const groupTasks = grouped[group.slug];
                if (!groupTasks || groupTasks.length === 0) return null;
                return (
                  <div key={group.slug}>
                    {/* Group header */}
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
                    {/* Task rows */}
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
                            <p
                              className="text-sm text-[var(--text-primary)] truncate flex-1 min-w-0 group-hover:text-[var(--primary)]"
                              style={{ fontFamily: 'Newsreader, Georgia, serif' }}
                            >
                              {task.title}
                            </p>
                            {/* Priority pill */}
                            <span
                              className="flex-shrink-0 px-1.5 py-[1px] rounded text-[9px] font-bold uppercase tracking-wider"
                              style={{ color: prio.text, backgroundColor: prio.bg, fontFamily: 'JetBrains Mono, monospace' }}
                            >
                              {task.priority}
                            </span>
                            {/* Assigned role */}
                            {task.assigned_role && (
                              <span
                                className="flex-shrink-0 text-[10px] px-1.5 py-0.5 rounded font-medium text-[var(--text-secondary)] bg-[var(--bg-tertiary)] truncate max-w-[80px]"
                                style={{ fontFamily: 'JetBrains Mono, monospace' }}
                              >
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

      <FeatureChangelogDrawer
        open={showUserChangelog}
        onClose={() => setShowUserChangelog(false)}
        type="user"
        content={feature.user_changelog || ''}
        featureName={feature.name}
      />
      <FeatureChangelogDrawer
        open={showTechChangelog}
        onClose={() => setShowTechChangelog(false)}
        type="tech"
        content={feature.tech_changelog || ''}
        featureName={feature.name}
      />
      <TaskSummariesDrawer
        open={showTaskSummaries}
        onClose={() => setShowTaskSummaries(false)}
        projectId={projectId}
        featureId={featureId!}
        featureName={feature.name}
      />
    </div>
  );
}
