import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { listAgents, assignAgentToProject } from '../lib/api';
import type { AgentResponse } from '../lib/types';

interface AddAgentToProjectDialogProps {
  projectId: string;
  assignedSlugs: Set<string>;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AddAgentToProjectDialog({ projectId, assignedSlugs, onClose, onSuccess }: AddAgentToProjectDialogProps) {
  const [allRoles, setAllRoles] = useState<AgentResponse[]>([]);
  const [selectedSlug, setSelectedSlug] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listAgents().then(data => {
      setAllRoles((data ?? []).filter(r => !assignedSlugs.has(r.slug)));
      setLoading(false);
    });
  }, [assignedSlugs]);

  const handleAdd = async () => {
    if (!selectedSlug) return;
    setSaving(true);
    try {
      await assignAgentToProject(projectId, { agent_slug: selectedSlug });
      onSuccess();
      onClose();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to assign agent');
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div
        className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg p-6 w-full max-w-sm shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        <h2 className="font-heading text-base text-[var(--text-primary)] mb-4">Add Agent to Project</h2>

        {loading ? (
          <div className="flex justify-center py-6">
            <Loader2 className="animate-spin text-[var(--text-dim)]" size={20} />
          </div>
        ) : allRoles.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] mb-4">
            All global agents are already assigned to this project.
          </p>
        ) : (
          <div className="mb-4">
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Agent</label>
            <select
              value={selectedSlug}
              onChange={e => setSelectedSlug(e.target.value)}
              data-qa="add-agent-select"
              className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
            >
              <option value="">Select an agent...</option>
              {allRoles.map(r => (
                <option key={r.slug} value={r.slug}>{r.name}</option>
              ))}
            </select>
          </div>
        )}

        {error && (
          <p className="text-xs text-[#FF3B30] mb-3">{error}</p>
        )}

        <div className="flex justify-end gap-2 mt-2">
          <button
            onClick={onClose}
            data-qa="add-agent-cancel-btn"
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
          >
            Cancel
          </button>
          {allRoles.length > 0 && (
            <button
              onClick={handleAdd}
              disabled={!selectedSlug || saving}
              data-qa="add-agent-confirm-btn"
              className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Adding...' : 'Add Agent'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
