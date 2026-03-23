import { useState, useEffect, useCallback } from 'react';
import { X, FolderOutput } from 'lucide-react';
import { listProjects, moveTaskToProject } from '../../lib/api';
import type { TaskWithDetailsResponse, ProjectWithSummary } from '../../lib/types';

interface MoveToProjectModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function MoveToProjectModal({ task, projectId, onClose, onSuccess }: MoveToProjectModalProps) {
  const [projects, setProjects] = useState<ProjectWithSummary[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState('');
  const [loading, setLoading] = useState(false);
  const [fetchLoading, setFetchLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // The effective project ID for the task (may be sub-project)
  const taskProjId = task.project_id || projectId;

  useEffect(() => {
    listProjects()
      .then((all) => {
        const filtered = all.filter((p) => p.id !== taskProjId);
        setProjects(filtered);
        if (filtered.length > 0) setSelectedProjectId(filtered[0].id);
      })
      .catch(() => setError('Failed to load projects'))
      .finally(() => setFetchLoading(false));
  }, [taskProjId]);

  const handleSubmit = async () => {
    if (!selectedProjectId) return;
    setLoading(true);
    setError(null);
    try {
      await moveTaskToProject(taskProjId, task.id, selectedProjectId);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to move task');
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
    if (e.key === 'Enter' && !loading && selectedProjectId) handleSubmit();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onClose, loading, selectedProjectId]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="absolute inset-0 bg-black/60" />
      <div className="relative w-[420px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Move to Project
          </h2>
          <button
            onClick={onClose}
            data-qa="move-to-project-close-btn"
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div className="flex items-start gap-3">
            <div className="mt-0.5 p-2 bg-[var(--bg-tertiary)] rounded-lg">
              <FolderOutput size={20} className="text-[var(--primary)]" />
            </div>
            <div>
              <h3 className="text-[var(--text-primary)] text-base font-['Inter'] font-semibold mb-1">
                Move task to another project
              </h3>
              <p className="text-[var(--text-secondary)] text-sm font-['Inter'] leading-relaxed">
                Select the destination project. The task will be moved to the Todo column.
              </p>
            </div>
          </div>

          <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
            <p className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-1">
              Task
            </p>
            <p className="text-[var(--text-primary)] text-sm font-['Inter'] font-medium">
              {task.title}
            </p>
          </div>

          {fetchLoading ? (
            <div className="flex items-center justify-center py-4">
              <div className="w-5 h-5 border-2 border-[var(--primary)] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : projects.length === 0 ? (
            <p className="text-[var(--text-muted)] text-sm font-['Inter'] text-center py-2">
              No other projects available.
            </p>
          ) : (
            <div className="space-y-1.5">
              <label className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
                Destination Project
              </label>
              <select
                value={selectedProjectId}
                onChange={(e) => setSelectedProjectId(e.target.value)}
                data-qa="move-to-project-select"
                className="w-full bg-[var(--bg-secondary)] border border-[var(--border-secondary)] text-[var(--text-primary)] text-sm font-['Inter'] rounded-md px-3 py-2 focus:outline-none focus:border-[var(--primary)] cursor-pointer"
              >
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
          )}

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="move-to-project-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || fetchLoading || !selectedProjectId || projects.length === 0}
            data-qa="move-to-project-submit-btn"
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Moving...' : 'Move Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
