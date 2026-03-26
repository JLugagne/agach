import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Plus,
  Loader2,
  Pencil,
  Trash2,
  CheckCircle2,
  Circle,
  Clock,
  AlertTriangle,
  BookOpen,
  GripVertical,
} from 'lucide-react';
import { DndContext, closestCenter, PointerSensor, useSensor, useSensors } from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, arrayMove, useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
  listFeatures,
  createFeature,
  deleteFeature,
  updateFeature,
  updateFeatureStatus,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { FeatureWithSummaryResponse, FeatureStatus, UpdateFeatureRequest } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';

const STATUSES: { value: string; label: string }[] = [
  { value: '', label: 'All' },
  { value: 'draft', label: 'Draft' },
  { value: 'ready', label: 'Ready' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'done', label: 'Done' },
  { value: 'blocked', label: 'Blocked' },
];

const STATUS_BADGE_COLORS: Record<FeatureStatus, string> = {
  draft: 'var(--text-muted)',
  ready: 'var(--status-todo)',
  in_progress: 'var(--status-progress)',
  done: 'var(--status-done)',
  blocked: '#FF3B30',
};

export default function FeaturesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [features, setFeatures] = useState<FeatureWithSummaryResponse[]>([]);
  const [localOrder, setLocalOrder] = useState<FeatureWithSummaryResponse[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('');

  // Create state
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);

  // Edit state
  const [editTarget, setEditTarget] = useState<FeatureWithSummaryResponse | null>(null);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [editStatus, setEditStatus] = useState<FeatureStatus>('draft');
  const [saving, setSaving] = useState(false);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<FeatureWithSummaryResponse | null>(null);
  const [deleting, setDeleting] = useState(false);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    try {
      const feats = await listFeatures(projectId, statusFilter || undefined);
      setFeatures(feats ?? []);
      setLocalOrder(null);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId, statusFilter]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(
    useCallback(
      (event) => {
        const type = event.type || '';
        if (type.startsWith('task_') || type.startsWith('feature_') || type.startsWith('project_')) {
          fetchData();
        }
      },
      [fetchData],
    ),
  );

  const handleCreate = async () => {
    if (!projectId || !newName.trim()) return;
    setCreating(true);
    try {
      await createFeature(projectId, {
        name: newName.trim(),
        description: newDesc.trim() || undefined,
      });
      setShowCreate(false);
      setNewName('');
      setNewDesc('');
      fetchData();
    } catch {
      // ignore
    } finally {
      setCreating(false);
    }
  };

  const handleEdit = async () => {
    if (!editTarget || !editName.trim() || !projectId) return;
    setSaving(true);
    try {
      const update: UpdateFeatureRequest = {
        name: editName.trim(),
        description: editDesc.trim(),
      };
      await updateFeature(projectId, editTarget.id, update);
      // Update status if changed
      if (editStatus !== editTarget.status) {
        await updateFeatureStatus(projectId, editTarget.id, { status: editStatus });
      }
      setEditTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget || !projectId) return;
    setDeleting(true);
    try {
      await deleteFeature(projectId, deleteTarget.id);
      setDeleteTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  // Drag and drop
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const displayFeatures = localOrder ?? features;

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = displayFeatures.findIndex((f) => f.id === active.id);
    const newIndex = displayFeatures.findIndex((f) => f.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;
    setLocalOrder(arrayMove(displayFeatures, oldIndex, newIndex));
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-4xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
        {/* Header */}
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Features
          </h1>
          <button
            onClick={() => setShowCreate(true)}
            data-qa="add-feature-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Feature
          </button>
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-6" style={{ fontFamily: 'Inter, sans-serif' }}>
          {features.length} feature{features.length !== 1 ? 's' : ''}
        </p>

        {/* Status filter tabs */}
        <div className="flex items-center gap-2 mb-8 flex-wrap">
          {STATUSES.map((s) => (
            <button
              key={s.value}
              onClick={() => { setStatusFilter(s.value); setLoading(true); }}
              data-qa="feature-status-filter-btn"
              className={`px-3 py-1.5 rounded-full text-[12px] font-medium transition-colors cursor-pointer border ${
                statusFilter === s.value
                  ? 'bg-[var(--primary)]/15 text-[var(--primary)] border-[var(--primary)]/30'
                  : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] border-[var(--border-primary)] hover:text-[var(--text-secondary)]'
              }`}
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              {s.label}
            </button>
          ))}
        </div>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
          </div>
        ) : features.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-5">
            <div className="w-20 h-20 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <BookOpen size={36} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-lg font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              {statusFilter ? 'No features with this status.' : 'No features yet.'}
            </p>
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              {statusFilter ? 'Try a different filter' : 'Get started by creating your first feature'}
            </p>
            {!statusFilter && (
              <button
                onClick={() => setShowCreate(true)}
                data-qa="create-first-feature-btn"
                className="flex items-center gap-2 px-6 py-3 rounded-lg text-sm font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                <Plus size={16} />
                Create your first feature
              </button>
            )}
          </div>
        ) : (
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext items={displayFeatures.map((f) => f.id)} strategy={verticalListSortingStrategy}>
              <div className="space-y-1">
                {displayFeatures.map((feat, index) => (
                  <FeatureRow
                    key={feat.id}
                    feature={feat}
                    index={index}
                    projectId={projectId!}
                    onEdit={() => {
                      setEditName(feat.name);
                      setEditDesc(feat.description || '');
                      setEditStatus(feat.status);
                      setEditTarget(feat);
                    }}
                    onDelete={() => setDeleteTarget(feat)}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}

        {/* Create Feature Modal */}
        {showCreate && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/60" onClick={() => { setShowCreate(false); setNewName(''); setNewDesc(''); }} data-qa="create-feature-modal-backdrop" />
            <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-md p-6">
              <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>New Feature</h2>

              {/* Name */}
              <div className="mb-4">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Title</label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="Feature name"
                  data-qa="new-feature-name-input"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && newName.trim() && handleCreate()}
                />
              </div>

              {/* Description */}
              <div className="mb-6">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
                <textarea
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  placeholder="Optional description"
                  rows={3}
                  data-qa="new-feature-description-textarea"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
                />
              </div>

              <div className="flex justify-end gap-3">
                <button
                  onClick={() => { setShowCreate(false); setNewName(''); setNewDesc(''); }}
                  data-qa="cancel-create-feature-btn"
                  className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreate}
                  disabled={!newName.trim() || creating}
                  data-qa="confirm-create-feature-btn"
                  className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
                >
                  {creating ? 'Creating...' : 'Create'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Edit Feature Modal */}
        {editTarget && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/60" onClick={() => setEditTarget(null)} data-qa="edit-feature-modal-backdrop" />
            <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-md p-6">
              <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>Edit Feature</h2>
              <div className="mb-4">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  data-qa="edit-feature-name-input"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                  autoFocus
                />
              </div>
              <div className="mb-4">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
                <textarea
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  rows={3}
                  data-qa="edit-feature-description-textarea"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
                />
              </div>
              <div className="mb-6">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Status</label>
                <select
                  value={editStatus}
                  onChange={(e) => setEditStatus(e.target.value as FeatureStatus)}
                  data-qa="edit-feature-status-select"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                >
                  <option value="draft">Draft</option>
                  <option value="ready">Ready</option>
                  <option value="in_progress">In Progress</option>
                  <option value="done">Done</option>
                  <option value="blocked">Blocked</option>
                </select>
              </div>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setEditTarget(null)}
                  data-qa="cancel-edit-feature-btn"
                  className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleEdit}
                  disabled={!editName.trim() || saving}
                  data-qa="confirm-edit-feature-btn"
                  className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
                >
                  {saving ? 'Saving...' : 'Save'}
                </button>
              </div>
            </div>
          </div>
        )}

        <DeleteConfirmModal
          open={!!deleteTarget}
          title="Delete Feature?"
          description={`This will permanently delete "${deleteTarget?.name}" and all its tasks.`}
          confirmLabel="Delete Feature"
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
          loading={deleting}
        />
      </div>
    </div>
  );
}

// ---------- Sortable Feature Row ----------

function FeatureRow({
  feature,
  index,
  projectId,
  onEdit,
  onDelete,
}: {
  feature: FeatureWithSummaryResponse;
  index: number;
  projectId: string;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const summary = feature.task_summary;
  const todo = summary?.todo_count ?? 0;
  const inProgress = summary?.in_progress_count ?? 0;
  const done = summary?.done_count ?? 0;
  const blocked = summary?.blocked_count ?? 0;
  const total = todo + inProgress + done + blocked;
  const pctDone = total > 0 ? Math.round((done / total) * 100) : 0;

  const statusColor = STATUS_BADGE_COLORS[feature.status] ?? 'var(--text-muted)';

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: feature.id });

  const dndStyle = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={dndStyle}
      data-qa="feature-row"
      className="flex items-center gap-3 px-3 py-2.5 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] hover:border-[var(--border-secondary)] transition-colors group"
    >
      {/* Drag handle */}
      <button
        {...attributes}
        {...listeners}
        className="text-[var(--text-dim)] hover:text-[var(--text-muted)] cursor-grab active:cursor-grabbing shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
        data-qa="feature-drag-handle"
      >
        <GripVertical size={14} />
      </button>

      {/* Position number */}
      <span className="text-[11px] font-mono text-[var(--text-dim)] w-5 text-right shrink-0">
        {index + 1}
      </span>

      {/* Status dot */}
      <div
        className="w-2 h-2 rounded-full shrink-0"
        style={{ backgroundColor: statusColor }}
        title={feature.status.replace('_', ' ')}
      />

      {/* Name */}
      <Link
        to={`/projects/${projectId}/features/${feature.id}`}
        className="text-sm text-[var(--text-primary)] hover:text-[var(--primary)] transition-colors truncate min-w-0 flex-1"
        data-qa="feature-row-link"
      >
        {feature.name}
      </Link>

      {/* Status badge */}
      <span
        className="text-[10px] px-2 py-0.5 rounded-full font-medium shrink-0"
        style={{
          color: statusColor,
          backgroundColor: `color-mix(in srgb, ${statusColor} 12%, transparent)`,
        }}
        data-qa="feature-status-badge"
      >
        {feature.status.replace('_', ' ')}
      </span>

      {/* Task counts */}
      <div className="flex items-center gap-2 shrink-0">
        <StatusDot icon={<Circle size={9} />} count={todo} color="var(--text-muted)" title="Todo" />
        <StatusDot icon={<Clock size={9} />} count={inProgress} color="var(--status-progress)" title="In Progress" />
        <StatusDot icon={<CheckCircle2 size={9} />} count={done} color="var(--status-done)" title="Done" />
        {blocked > 0 && <StatusDot icon={<AlertTriangle size={9} />} count={blocked} color="#FF3B30" title="Blocked" />}
      </div>

      {/* Progress */}
      {total > 0 && (
        <span className="text-[10px] font-mono text-[var(--text-dim)] shrink-0 w-10 text-right">
          {pctDone}%
        </span>
      )}

      {/* Actions */}
      <div className="flex items-center gap-0.5 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={onEdit}
          data-qa="edit-feature-btn"
          className="p-1 text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors rounded"
        >
          <Pencil size={12} />
        </button>
        <button
          onClick={onDelete}
          data-qa="delete-feature-btn"
          className="p-1 text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors rounded"
        >
          <Trash2 size={12} />
        </button>
      </div>
    </div>
  );
}

function StatusDot({ icon, count, color, title }: { icon: React.ReactNode; count: number; color: string; title: string }) {
  return (
    <div className="flex items-center gap-0.5" title={title}>
      <span style={{ color }}>{icon}</span>
      <span className="text-[10px] font-mono" style={{ color }}>{count}</span>
    </div>
  );
}
