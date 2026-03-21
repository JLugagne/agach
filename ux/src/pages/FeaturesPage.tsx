import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Plus,
  X,
  Loader2,
  ExternalLink,
  ChevronRight,
  Pencil,
} from 'lucide-react';
import {
  getProject,
  listFeatures,
  createFeature,
  deleteFeature,
  updateFeature,
} from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { ProjectResponse, ProjectWithSummary, UpdateProjectRequest } from '../lib/types';

export default function FeaturesPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [features, setFeatures] = useState<ProjectWithSummary[]>([]);
  const [loading, setLoading] = useState(true);

  // Create state
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);

  // Edit state
  const [editTarget, setEditTarget] = useState<ProjectWithSummary | null>(null);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [saving, setSaving] = useState(false);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<ProjectWithSummary | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Detail drawer
  const [selectedFeature, setSelectedFeature] = useState<ProjectWithSummary | null>(null);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    try {
      const [p, feats] = await Promise.all([
        getProject(projectId),
        listFeatures(projectId),
      ]);
      setProject(p);
      setFeatures(feats ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => { fetchData(); }, [fetchData]);

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
      if (selectedFeature?.id === deleteTarget.id) setSelectedFeature(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[#0F0F0F] flex items-center justify-center">
        <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
      </div>
    );
  }

  const rightDrawer = selectedFeature ? (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[#1E1E1E]">
        <h3 className="font-heading text-sm text-[#F0F0F0]">{selectedFeature.name}</h3>
        <button
          onClick={() => setSelectedFeature(null)}
          className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
        >
          <X size={16} />
        </button>
      </div>
      <div className="flex-1 overflow-auto p-6">
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Name</label>
            <p className="text-sm text-[#F0F0F0]">{selectedFeature.name}</p>
          </div>
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Description</label>
            <p className="text-sm text-[var(--text-muted)]">
              {selectedFeature.description || 'No description'}
            </p>
          </div>
          {selectedFeature.summary && (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Tasks</label>
              <div className="flex flex-wrap gap-3 text-xs">
                <span className="text-[var(--text-muted)]">
                  {selectedFeature.summary.todo_count} todo
                </span>
                <span className="text-[#00C896]">
                  {selectedFeature.summary.in_progress_count} in progress
                </span>
                <span className="text-[var(--text-muted)]">
                  {selectedFeature.summary.done_count} done
                </span>
                <span className="text-[#F06060]">
                  {selectedFeature.summary.blocked_count} blocked
                </span>
              </div>
            </div>
          )}
          <div className="pt-4 flex gap-3 flex-wrap">
            <Link
              to={`/projects/${selectedFeature.id}`}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-[#00C896] text-[#0F0F0F] text-xs font-medium rounded-md hover:bg-[#00C896]/80 transition-colors"
            >
              Open Board
              <ExternalLink size={11} />
            </Link>
            <button
              onClick={() => {
                setEditName(selectedFeature.name);
                setEditDesc(selectedFeature.description || '');
                setEditTarget(selectedFeature);
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-[#252525] rounded-md hover:border-[#3A3A3A] transition-colors text-[var(--text-muted)]"
            >
              <Pencil size={11} />
              Edit
            </button>
            <button
              onClick={() => setDeleteTarget(selectedFeature)}
              className="px-3 py-1.5 text-xs text-[#F06060] hover:text-[#FF3B30] border border-[#FF3B30]/30 rounded-md transition-colors"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>
  ) : undefined;

  return (
    <SettingsLayout projectName={project?.name ?? 'Project'} rightDrawer={rightDrawer}>
      <div className="flex items-center justify-between mb-8">
        <h1 className="font-heading text-2xl text-[#F0F0F0]">Features</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-[#00C896] text-[#0F0F0F] text-xs font-medium rounded-md hover:bg-[#00C896]/80 transition-colors"
        >
          <Plus size={13} />
          Add Feature
        </button>
      </div>

      {features.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-[var(--text-dim)] text-sm mb-4">No features yet.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="text-sm text-[#00C896] hover:text-[#00C896]/80 transition-colors"
          >
            Create your first feature
          </button>
        </div>
      ) : (
        <div className="space-y-2">
          {features.map((feat) => {
            const summary = feat.task_summary ?? feat.summary;
            const total =
              (summary?.todo_count ?? 0) +
              (summary?.in_progress_count ?? 0) +
              (summary?.done_count ?? 0) +
              (summary?.blocked_count ?? 0);
            const active =
              (summary?.todo_count ?? 0) +
              (summary?.in_progress_count ?? 0) +
              (summary?.blocked_count ?? 0);
            return (
              <button
                key={feat.id}
                onClick={() => setSelectedFeature(feat)}
                className={`w-full text-left rounded-lg border p-4 flex items-center gap-3 transition-colors cursor-pointer ${
                  selectedFeature?.id === feat.id
                    ? 'bg-[#1A1A1A] border-[#00C896]/30'
                    : 'bg-[#111111] border-[#1E1E1E] hover:border-[#252525]'
                }`}
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm text-[#F0F0F0] truncate">{feat.name}</h3>
                    {active > 0 && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-[#00C896]/15 text-[#00C896] font-mono shrink-0">
                        active
                      </span>
                    )}
                  </div>
                  {feat.description && (
                    <p className="text-xs text-[var(--text-muted)] truncate mt-0.5">
                      {feat.description}
                    </p>
                  )}
                </div>
                <span className="text-xs text-[var(--text-dim)] shrink-0">
                  {total} task{total !== 1 ? 's' : ''}
                </span>
                <ChevronRight size={14} className="text-[var(--text-dim)] shrink-0" />
              </button>
            );
          })}
        </div>
      )}

      {/* Create Feature Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowCreate(false)} />
          <div className="relative bg-[#111111] border border-[#1E1E1E] rounded-lg w-full max-w-md p-6">
            <h2 className="font-heading text-lg text-[#F0F0F0] mb-4">New Feature</h2>
            <div className="mb-4">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Feature name"
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
                autoFocus
                onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
              />
            </div>
            <div className="mb-6">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
              <textarea
                value={newDesc}
                onChange={(e) => setNewDesc(e.target.value)}
                placeholder="Optional description"
                rows={3}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50 resize-y"
              />
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCreate(false)}
                className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={!newName.trim() || creating}
                className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
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
          <div className="absolute inset-0 bg-black/60" onClick={() => setEditTarget(null)} />
          <div className="relative bg-[#111111] border border-[#1E1E1E] rounded-lg w-full max-w-md p-6">
            <h2 className="font-heading text-lg text-[#F0F0F0] mb-4">Edit Feature</h2>
            <div className="mb-4">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50"
                autoFocus
              />
            </div>
            <div className="mb-6">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Description</label>
              <textarea
                value={editDesc}
                onChange={(e) => setEditDesc(e.target.value)}
                rows={3}
                className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50 resize-y"
              />
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setEditTarget(null)}
                className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleEdit}
                disabled={!editName.trim() || saving}
                className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
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
    </SettingsLayout>
  );
}
