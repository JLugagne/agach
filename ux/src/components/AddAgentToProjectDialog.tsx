import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { listRoles, assignAgentToProject } from '../lib/api';
import type { RoleResponse } from '../lib/types';

interface AddAgentToProjectDialogProps {
  projectId: string;
  assignedSlugs: Set<string>;
  onClose: () => void;
  onSuccess: () => void;
}

export default function AddAgentToProjectDialog({ projectId, assignedSlugs, onClose, onSuccess }: AddAgentToProjectDialogProps) {
  const [allRoles, setAllRoles] = useState<RoleResponse[]>([]);
  const [selectedSlug, setSelectedSlug] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listRoles().then(data => {
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
        className="bg-[#1A1A1A] border border-[#252525] rounded-lg p-6 w-full max-w-sm shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        <h2 className="font-heading text-base text-[#F0F0F0] mb-4">Add Agent to Project</h2>

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
              className="w-full bg-[#111] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] focus:outline-none focus:border-[#00C896]/50"
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
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#F0F0F0] transition-colors"
          >
            Cancel
          </button>
          {allRoles.length > 0 && (
            <button
              onClick={handleAdd}
              disabled={!selectedSlug || saving}
              data-qa="add-agent-confirm-btn"
              className="px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Adding...' : 'Add Agent'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
