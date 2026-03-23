import { useState, useEffect, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  Plus,
  X,
  Loader2,
  ExternalLink,
  ChevronRight,
} from 'lucide-react';
import {
  getProject,
  listSubProjects,
  createProject,
  deleteProject,
} from '../lib/api';
import SettingsLayout from '../components/settings/SettingsLayout';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { ProjectResponse, ProjectWithSummary } from '../lib/types';

export default function SubProjectsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [project, setProject] = useState<ProjectResponse | null>(null);
  const [subProjects, setSubProjects] = useState<ProjectWithSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ProjectWithSummary | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [selectedProject, setSelectedProject] = useState<ProjectWithSummary | null>(null);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    try {
      const [p, subs] = await Promise.all([
        getProject(projectId),
        listSubProjects(projectId),
      ]);
      setProject(p);
      setSubProjects(subs ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleCreate = async () => {
    if (!projectId || !newName.trim()) return;
    setCreating(true);
    try {
      await createProject({
        name: newName.trim(),
        description: newDesc.trim(),
        parent_id: projectId,
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

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteProject(deleteTarget.id);
      setDeleteTarget(null);
      if (selectedProject?.id === deleteTarget.id) {
        setSelectedProject(null);
      }
      fetchData();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[var(--bg-secondary)] flex items-center justify-center">
        <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
      </div>
    );
  }

  const rightDrawer = selectedProject ? (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
        <h3 className="font-heading text-sm text-[var(--text-primary)]">{selectedProject.name}</h3>
        <button
          onClick={() => setSelectedProject(null)}
          className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
        >
          <X size={16} />
        </button>
      </div>
      <div className="flex-1 overflow-auto p-6">
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Name</label>
            <p className="text-sm text-[var(--text-primary)]">{selectedProject.name}</p>
          </div>
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Description</label>
            <p className="text-sm text-[var(--text-muted)]">
              {selectedProject.description || 'No description'}
            </p>
          </div>
          {selectedProject.summary && (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Tasks</label>
              <div className="flex flex-wrap gap-3 text-xs">
                <span className="text-[var(--text-muted)]">
                  {selectedProject.summary.todo_count} todo
                </span>
                <span className="text-[var(--primary)]">
                  {selectedProject.summary.in_progress_count} in progress
                </span>
                <span className="text-[var(--text-muted)]">
                  {selectedProject.summary.done_count} done
                </span>
                <span className="text-[#F06060]">
                  {selectedProject.summary.blocked_count} blocked
                </span>
              </div>
            </div>
          )}
          <div className="pt-4 flex gap-3">
            <Link
              to={`/projects/${selectedProject.id}`}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-[var(--primary)] text-[var(--primary-text)] text-xs font-medium rounded-md hover:bg-[var(--primary-hover)]/80 transition-colors"
            >
              Open Board
              <ExternalLink size={11} />
            </Link>
            <button
              onClick={() => setDeleteTarget(selectedProject)}
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
        <h1 className="font-heading text-2xl text-[var(--text-primary)]">Sub-Projects</h1>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-[var(--primary)] text-[var(--primary-text)] text-xs font-medium rounded-md hover:bg-[var(--primary-hover)]/80 transition-colors"
        >
          <Plus size={13} />
          Add Sub-Project
        </button>
      </div>

      {subProjects.length === 0 ? (
        <div className="text-center py-16">
          <p className="text-[var(--text-dim)] text-sm mb-4">No sub-projects yet.</p>
          <button
            onClick={() => setShowCreate(true)}
            className="text-sm text-[var(--primary)] hover:text-[var(--primary)]/80 transition-colors"
          >
            Create your first sub-project
          </button>
        </div>
      ) : (
        <div className="space-y-2">
          {subProjects.map((sub) => {
            const total =
              (sub.summary?.todo_count ?? 0) +
              (sub.summary?.in_progress_count ?? 0) +
              (sub.summary?.done_count ?? 0) +
              (sub.summary?.blocked_count ?? 0);
            return (
              <button
                key={sub.id}
                onClick={() => setSelectedProject(sub)}
                className={`w-full text-left rounded-lg border p-4 flex items-center gap-3 transition-colors cursor-pointer ${
                  selectedProject?.id === sub.id
                    ? 'bg-[var(--bg-secondary)] border-[var(--primary)]/30'
                    : 'bg-[var(--bg-primary)] border-[var(--border-primary)] hover:border-[var(--border-secondary)]'
                }`}
              >
                <div className="flex-1 min-w-0">
                  <h3 className="text-sm text-[var(--text-primary)] truncate">{sub.name}</h3>
                  {sub.description && (
                    <p className="text-xs text-[var(--text-muted)] truncate mt-0.5">{sub.description}</p>
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

      {/* Create Sub-Project Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setShowCreate(false)} />
          <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-md p-6">
            <h2 className="font-heading text-lg text-[var(--text-primary)] mb-4">New Sub-Project</h2>

            <div className="mb-4">
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
              <input
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Sub-project name"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
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
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 resize-y"
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
                className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
              >
                {creating ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      <DeleteConfirmModal
        open={!!deleteTarget}
        title="Delete Sub-Project?"
        description={`This will permanently delete "${deleteTarget?.name}" and all its data.`}
        confirmLabel="Delete Sub-Project"
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
        loading={deleting}
      />
    </SettingsLayout>
  );
}
