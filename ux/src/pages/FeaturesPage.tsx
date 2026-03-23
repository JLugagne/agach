import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Plus,
  X,
  Loader2,
  ExternalLink,
  Pencil,
  CheckCircle2,
  Circle,
  Clock,
  AlertTriangle,
  Eye,
  EyeOff,
  BookOpen,
} from 'lucide-react';
import {
  listFeatures,
  createFeature,
  deleteFeature,
  updateFeature,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { ProjectWithSummary, UpdateProjectRequest } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';

export default function FeaturesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [features, setFeatures] = useState<ProjectWithSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [showDone, setShowDone] = useState(false);

  // Create state
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [newTags, setNewTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [creating, setCreating] = useState(false);

  // Edit state
  const [editTarget, setEditTarget] = useState<ProjectWithSummary | null>(null);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [saving, setSaving] = useState(false);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<ProjectWithSummary | null>(null);
  const [deleting, setDeleting] = useState(false);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    try {
      const feats = await listFeatures(projectId);
      setFeatures(feats ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(
    useCallback(
      (event) => {
        const type = event.type || '';
        if (type.startsWith('task_') || type.startsWith('project_')) {
          fetchData();
        }
      },
      [fetchData],
    ),
  );

  // Collect all existing tags from features for auto-suggest
  const allTags = useMemo(() => {
    const tags = new Set<string>();
    // Features don't have tags directly, but we can extract from descriptions or add later
    return Array.from(tags);
  }, []);

  const filteredFeatures = useMemo(() => {
    if (showDone) return features;
    return features.filter((feat) => {
      const summary = feat.task_summary ?? feat.summary;
      const active =
        (summary?.todo_count ?? 0) +
        (summary?.in_progress_count ?? 0) +
        (summary?.blocked_count ?? 0);
      const total =
        active + (summary?.done_count ?? 0);
      // Show if it has active tasks OR has no tasks at all (new feature)
      return active > 0 || total === 0;
    });
  }, [features, showDone]);

  const doneCount = useMemo(() => {
    return features.filter((feat) => {
      const summary = feat.task_summary ?? feat.summary;
      const active =
        (summary?.todo_count ?? 0) +
        (summary?.in_progress_count ?? 0) +
        (summary?.blocked_count ?? 0);
      const total = active + (summary?.done_count ?? 0);
      return active === 0 && total > 0;
    }).length;
  }, [features]);

  const handleCreate = async () => {
    if (!projectId || !newName.trim()) return;
    setCreating(true);
    try {
      await createFeature(projectId, {
        name: newName.trim(),
        description: newDesc.trim(),
      });
      setShowCreate(false);
      setNewName('');
      setNewDesc('');
      setNewTags([]);
      setTagInput('');
      fetchData();
    } catch {
      // ignore
    } finally {
      setCreating(false);
    }
  };

  const handleEdit = async () => {
    if (!editTarget || !editName.trim()) return;
    setSaving(true);
    try {
      const update: UpdateProjectRequest = {
        name: editName.trim(),
        description: editDesc.trim(),
      };
      await updateFeature(editTarget.id, update);
      setEditTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteFeature(deleteTarget.id);
      setDeleteTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  const addTag = (tag: string) => {
    const trimmed = tag.trim().toLowerCase();
    if (trimmed && !newTags.includes(trimmed)) {
      setNewTags([...newTags, trimmed]);
    }
    setTagInput('');
  };

  const removeTag = (tag: string) => {
    setNewTags(newTags.filter((t) => t !== tag));
  };

  const tagSuggestions = useMemo(() => {
    if (!tagInput.trim()) return [];
    return allTags.filter(
      (t) => t.includes(tagInput.toLowerCase()) && !newTags.includes(t),
    );
  }, [tagInput, allTags, newTags]);

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-5xl mx-auto px-8 py-12">
        {/* Header */}
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Features
          </h1>
          <div className="flex items-center gap-3">
            {doneCount > 0 && (
              <button
                onClick={() => setShowDone(!showDone)}
                data-qa="toggle-done-features-btn"
                className="flex items-center gap-1.5 px-4 py-2.5 text-[13px] text-[var(--text-muted)] hover:text-[var(--text-primary)] border border-[var(--border-primary)] rounded-lg transition-colors cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {showDone ? <EyeOff size={14} /> : <Eye size={14} />}
                {showDone ? 'Hide done' : `Show done (${doneCount})`}
              </button>
            )}
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
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-10" style={{ fontFamily: 'Inter, sans-serif' }}>
          {features.length} feature{features.length !== 1 ? 's' : ''}
        </p>

        {/* Content */}
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
          </div>
        ) : filteredFeatures.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 gap-5">
            <div className="w-20 h-20 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <BookOpen size={36} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-lg font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              {features.length === 0 ? 'No features yet.' : 'No active features.'}
            </p>
            <p className="text-sm text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              {features.length === 0 ? 'Get started by creating your first feature' : 'All features are done'}
            </p>
            {features.length === 0 && (
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
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {filteredFeatures.map((feat) => (
              <FeatureCard
                key={feat.id}
                feature={feat}
                onEdit={() => {
                  setEditName(feat.name);
                  setEditDesc(feat.description || '');
                  setEditTarget(feat);
                }}
              />
            ))}
          </div>
        )}

        {/* Create Feature Modal */}
        {showCreate && (
          <CreateFeatureModal
            name={newName}
            onNameChange={setNewName}
            description={newDesc}
            onDescriptionChange={setNewDesc}
            tags={newTags}
            tagInput={tagInput}
            onTagInputChange={setTagInput}
            onAddTag={addTag}
            onRemoveTag={removeTag}
            tagSuggestions={tagSuggestions}
            creating={creating}
            onCreate={handleCreate}
            onClose={() => {
              setShowCreate(false);
              setNewName('');
              setNewDesc('');
              setNewTags([]);
              setTagInput('');
            }}
          />
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
              <div className="mb-6">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
                <textarea
                  value={editDesc}
                  onChange={(e) => setEditDesc(e.target.value)}
                  rows={3}
                  data-qa="edit-feature-description-textarea"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
                />
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

// ---------- Feature Card ----------

function FeatureCard({
  feature,
  onEdit,
}: {
  feature: ProjectWithSummary;
  onEdit: () => void;
}) {
  const summary = feature.task_summary ?? feature.summary;
  const todo = summary?.todo_count ?? 0;
  const inProgress = summary?.in_progress_count ?? 0;
  const done = summary?.done_count ?? 0;
  const blocked = summary?.blocked_count ?? 0;
  const total = todo + inProgress + done + blocked;

  const isDone = total > 0 && todo === 0 && inProgress === 0 && blocked === 0;

  // Progress bar percentages
  const pctDone = total > 0 ? (done / total) * 100 : 0;
  const pctInProgress = total > 0 ? (inProgress / total) * 100 : 0;
  const pctBlocked = total > 0 ? (blocked / total) * 100 : 0;

  return (
    <div
      className={`rounded-lg border p-5 transition-colors ${
        isDone
          ? 'bg-[var(--bg-primary)] border-[var(--border-primary)] opacity-60'
          : 'bg-[var(--bg-primary)] border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
      }`}
      data-qa="feature-card"
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-medium text-[var(--text-primary)] truncate">{feature.name}</h3>
            {isDone && (
              <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--status-done)]/15 text-[var(--status-done)] font-mono shrink-0">
                done
              </span>
            )}
            {blocked > 0 && (
              <AlertTriangle size={12} className="text-[#FF3B30] shrink-0" />
            )}
          </div>
          {feature.description && (
            <p className="text-xs text-[var(--text-muted)] mt-1 line-clamp-2">{feature.description}</p>
          )}
        </div>
        <div className="flex items-center gap-1 ml-2 shrink-0">
          <button
            onClick={onEdit}
            data-qa="edit-feature-btn"
            className="p-1 text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors rounded"
          >
            <Pencil size={12} />
          </button>
        </div>
      </div>

      {/* Task status row */}
      <div className="flex items-center gap-3 mb-3 flex-wrap">
        <StatusPill icon={<Circle size={10} />} label="Todo" count={todo} color="var(--text-muted)" />
        <StatusPill icon={<Clock size={10} />} label="In Progress" count={inProgress} color="var(--primary)" />
        <StatusPill icon={<CheckCircle2 size={10} />} label="Done" count={done} color="var(--status-done)" />
        {blocked > 0 && (
          <StatusPill icon={<AlertTriangle size={10} />} label="Blocked" count={blocked} color="#FF3B30" />
        )}
      </div>

      {/* Progress bar */}
      {total > 0 && (
        <div className="h-1.5 bg-[var(--bg-tertiary)] rounded-full overflow-hidden flex mb-3">
          {pctDone > 0 && (
            <div className="h-full" style={{ width: `${pctDone}%`, backgroundColor: 'var(--status-done)' }} />
          )}
          {pctInProgress > 0 && (
            <div className="h-full" style={{ width: `${pctInProgress}%`, backgroundColor: 'var(--primary)' }} />
          )}
          {pctBlocked > 0 && (
            <div className="h-full" style={{ width: `${pctBlocked}%`, backgroundColor: '#FF3B30' }} />
          )}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-mono text-[var(--text-dim)]">
          {total} task{total !== 1 ? 's' : ''}
          {total > 0 && ` · ${Math.round(pctDone)}% done`}
        </span>
        <div className="flex items-center gap-2">
          <Link
            to={`/projects/${feature.id}`}
            data-qa="open-feature-board-link"
            className="flex items-center gap-1 text-[10px] text-[var(--primary)] hover:text-[var(--primary-hover)] transition-colors"
          >
            Open Board
            <ExternalLink size={9} />
          </Link>
        </div>
      </div>
    </div>
  );
}

function StatusPill({ icon, label, count, color }: { icon: React.ReactNode; label: string; count: number; color: string }) {
  return (
    <div className="flex items-center gap-1" title={label}>
      <span style={{ color }}>{icon}</span>
      <span className="text-[10px] font-mono" style={{ color }}>{count}</span>
    </div>
  );
}

// ---------- Create Feature Modal ----------

function CreateFeatureModal({
  name,
  onNameChange,
  description,
  onDescriptionChange,
  tags,
  tagInput,
  onTagInputChange,
  onAddTag,
  onRemoveTag,
  tagSuggestions,
  creating,
  onCreate,
  onClose,
}: {
  name: string;
  onNameChange: (v: string) => void;
  description: string;
  onDescriptionChange: (v: string) => void;
  tags: string[];
  tagInput: string;
  onTagInputChange: (v: string) => void;
  onAddTag: (tag: string) => void;
  onRemoveTag: (tag: string) => void;
  tagSuggestions: string[];
  creating: boolean;
  onCreate: () => void;
  onClose: () => void;
}) {
  const tagInputRef = useRef<HTMLInputElement>(null);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} data-qa="create-feature-modal-backdrop" />
      <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-md p-6">
        <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>New Feature</h2>

        {/* Name */}
        <div className="mb-4">
          <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Title</label>
          <input
            type="text"
            value={name}
            onChange={(e) => onNameChange(e.target.value)}
            placeholder="Feature name"
            data-qa="new-feature-name-input"
            className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && name.trim() && onCreate()}
          />
        </div>

        {/* Description */}
        <div className="mb-4">
          <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
          <textarea
            value={description}
            onChange={(e) => onDescriptionChange(e.target.value)}
            placeholder="Optional description"
            rows={3}
            data-qa="new-feature-description-textarea"
            className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
          />
        </div>

        {/* Tags */}
        <div className="mb-6">
          <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Tags</label>
          <div className="flex flex-wrap gap-1.5 mb-2">
            {tags.map((tag) => (
              <span
                key={tag}
                className="flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full bg-[var(--bg-tertiary)] text-[var(--text-secondary)]"
              >
                {tag}
                <button
                  onClick={() => onRemoveTag(tag)}
                  className="hover:text-[#FF3B30] transition-colors"
                >
                  <X size={10} />
                </button>
              </span>
            ))}
          </div>
          <div className="relative">
            <input
              ref={tagInputRef}
              type="text"
              value={tagInput}
              onChange={(e) => onTagInputChange(e.target.value)}
              placeholder="Add tag..."
              data-qa="new-feature-tag-input"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-1.5 text-xs text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && tagInput.trim()) {
                  e.preventDefault();
                  onAddTag(tagInput);
                } else if (e.key === 'Backspace' && !tagInput && tags.length > 0) {
                  onRemoveTag(tags[tags.length - 1]);
                }
              }}
            />
            {tagSuggestions.length > 0 && (
              <div className="absolute z-10 left-0 right-0 mt-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md shadow-lg max-h-32 overflow-y-auto">
                {tagSuggestions.map((s) => (
                  <button
                    key={s}
                    onClick={() => onAddTag(s)}
                    className="w-full text-left px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] transition-colors"
                  >
                    {s}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            data-qa="cancel-create-feature-btn"
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onCreate}
            disabled={!name.trim() || creating}
            data-qa="confirm-create-feature-btn"
            className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
          >
            {creating ? 'Creating...' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  );
}
