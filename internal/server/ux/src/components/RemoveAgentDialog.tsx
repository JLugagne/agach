import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { getTasksByAgent, removeAgentFromProject } from '../lib/api';
import type { AgentResponse, TasksByAgentResponse } from '../lib/types';

interface RemoveAgentDialogProps {
  projectId: string;
  agent: AgentResponse;
  projectAgents: AgentResponse[];
  onClose: () => void;
  onSuccess: () => void;
}

export default function RemoveAgentDialog({ projectId, agent, projectAgents, onClose, onSuccess }: RemoveAgentDialogProps) {
  const [taskData, setTaskData] = useState<TasksByAgentResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState<'reassign' | 'clear' | 'none'>('none');
  const [reassignTarget, setReassignTarget] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTasksByAgent(projectId, agent.slug)
      .then(data => setTaskData(data))
      .catch(() => setTaskData({ agent_slug: agent.slug, task_count: 0, tasks: [] }))
      .finally(() => setLoading(false));
  }, [projectId, agent.slug]);

  const handleConfirm = async () => {
    if ((taskData?.task_count ?? 0) > 0 && action === 'none') {
      setError('Please choose what to do with the existing tasks.');
      return;
    }
    if (action === 'reassign' && !reassignTarget) {
      setError('Please select a target agent for reassignment.');
      return;
    }

    setSaving(true);
    try {
      await removeAgentFromProject(projectId, agent.slug, {
        reassign_to: action === 'reassign' ? reassignTarget : undefined,
        clear_assignment: action === 'clear',
      });
      onSuccess();
      onClose();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to remove agent');
      setSaving(false);
    }
  };

  const isConfirmDisabled =
    saving ||
    (taskData !== null && taskData.task_count > 0 && action === 'none') ||
    (action === 'reassign' && !reassignTarget);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div
        className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg p-6 w-full max-w-md shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        <h2 className="font-heading text-base text-[var(--text-primary)] mb-3">
          Remove &ldquo;{agent.name}&rdquo; from project?
        </h2>

        {loading ? (
          <div className="flex justify-center py-6">
            <Loader2 className="animate-spin text-[var(--text-dim)]" size={20} />
          </div>
        ) : (
          <>
            {(taskData?.task_count ?? 0) > 0 ? (
              <div className="mb-4">
                <p className="text-sm text-[var(--text-muted)] mb-3">
                  This agent has <span className="text-[var(--text-primary)] font-medium">{taskData!.task_count}</span> task{taskData!.task_count !== 1 ? 's' : ''} assigned in this project.
                  What would you like to do with those tasks?
                </p>
                <div className="space-y-2">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="action"
                      value="reassign"
                      checked={action === 'reassign'}
                      onChange={() => setAction('reassign')}
                      data-qa="remove-agent-reassign-radio"
                      className="accent-[var(--primary)]"
                    />
                    <span className="text-sm text-[var(--text-muted)]">Reassign to:</span>
                    <select
                      value={reassignTarget}
                      onChange={e => setReassignTarget(e.target.value)}
                      disabled={action !== 'reassign'}
                      data-qa="remove-agent-reassign-select"
                      className="ml-1 text-sm rounded border border-border bg-[var(--bg-primary)] text-[var(--text-primary)] px-2 py-0.5 disabled:opacity-40"
                    >
                      <option value="">Select agent...</option>
                      {projectAgents.filter(r => r.slug !== agent.slug).map(r => (
                        <option key={r.slug} value={r.slug}>{r.name}</option>
                      ))}
                    </select>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="action"
                      value="clear"
                      checked={action === 'clear'}
                      onChange={() => setAction('clear')}
                      data-qa="remove-agent-clear-radio"
                      className="accent-[var(--primary)]"
                    />
                    <span className="text-sm text-[var(--text-muted)]">Clear assignment (tasks become unassigned)</span>
                  </label>
                </div>
              </div>
            ) : (
              <p className="text-sm text-[var(--text-muted)] mb-4">
                This agent has no tasks in this project.
              </p>
            )}

            {error && (
              <p className="text-xs text-[#FF3B30] mb-3">{error}</p>
            )}

            <div className="flex justify-end gap-2 mt-2">
              <button
                onClick={onClose}
                data-qa="remove-agent-cancel-btn"
                className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirm}
                disabled={isConfirmDisabled}
                data-qa="remove-agent-confirm-btn"
                className="px-4 py-2 bg-[#FF3B30] text-white text-sm font-medium rounded-md hover:bg-[#FF3B30]/80 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Removing...' : 'Confirm Remove'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
