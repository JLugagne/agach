import { useState, useEffect, useCallback, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { X as XIcon, Plus, X as XClose } from 'lucide-react';
import { completeTask } from '../../lib/api';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface CompleteTaskModalProps {
  task: TaskWithDetailsResponse;
  projectId: string;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CompleteTaskModal({ task, projectId, onClose, onSuccess }: CompleteTaskModalProps) {
  const [summary, setSummary] = useState('');
  const [filesModified, setFilesModified] = useState<string[]>([]);
  const [fileInput, setFileInput] = useState('');
  const [agentName, setAgentName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addFile = () => {
    const trimmed = fileInput.trim();
    if (trimmed && !filesModified.includes(trimmed)) {
      setFilesModified([...filesModified, trimmed]);
      setFileInput('');
    }
  };

  const removeFile = (index: number) => {
    setFilesModified(filesModified.filter((_, i) => i !== index));
  };

  const handleFileKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addFile();
    }
  };

  const handleSubmit = async () => {
    if (summary.trim().length < 100) {
      setError('Completion summary must be at least 100 characters.');
      return;
    }
    if (agentName.trim().length === 0) {
      setError('Completed by agent is required.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await completeTask(projectId, task.id, {
        completion_summary: summary.trim(),
        files_modified: filesModified,
        completed_by_agent: agentName.trim(),
      });
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete task');
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  }, [onClose]);

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
      <div className="relative w-[480px] rounded-xl bg-[var(--bg-elevated)] border border-[var(--border-primary)] shadow-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[var(--text-primary)] text-lg font-semibold font-['Newsreader']">
            Complete Task
          </h2>
          <button
            onClick={onClose}
            data-qa="complete-task-close-btn"
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <XIcon size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-4">
          <div>
            <p className="text-[var(--text-secondary)] text-sm font-['Inter'] mb-1">Task</p>
            <p className="text-[var(--text-primary)] text-sm font-['Inter'] font-medium">{task.title}</p>
          </div>

          <div>
            <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
              Completion Summary
            </label>
            <textarea
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="Describe what was accomplished (min 100 chars)..."
              rows={5}
              data-qa="complete-summary-input"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
            <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-1">
              {summary.length} / 100 minimum characters
            </p>
          </div>

          <div>
            <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
              Files Modified
            </label>
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                value={fileInput}
                onChange={(e) => setFileInput(e.target.value)}
                onKeyDown={handleFileKeyDown}
                placeholder="path/to/file.go"
                data-qa="complete-file-path-input"
                className="flex-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['JetBrains_Mono'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
              />
              <button
                onClick={addFile}
                disabled={!fileInput.trim()}
                data-qa="complete-add-file-btn"
                className="px-3 py-2 bg-[var(--bg-tertiary)] border border-[var(--border-primary)] rounded-md text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--primary)] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              >
                <Plus size={16} />
              </button>
            </div>
            {filesModified.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {filesModified.map((file, idx) => (
                  <span
                    key={idx}
                    className="inline-flex items-center gap-1 px-2 py-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded text-[var(--text-primary)] text-xs font-['JetBrains_Mono']"
                  >
                    {file}
                    <button
                      onClick={() => removeFile(idx)}
                      data-qa="complete-remove-file-btn"
                      className="text-[var(--text-muted)] hover:text-[var(--status-blocked)] transition-colors"
                    >
                      <XClose size={12} />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>

          <div>
            <label className="block text-[var(--text-primary)] text-sm font-['Inter'] font-medium mb-2">
              Completed By Agent
            </label>
            <input
              type="text"
              value={agentName}
              onChange={(e) => setAgentName(e.target.value)}
              placeholder="e.g. human or agent identifier"
              data-qa="complete-agent-name-input"
              className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          {error && (
            <p className="text-[var(--status-blocked)] text-sm font-['Inter']">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="complete-task-cancel-btn"
            className="px-4 py-2 text-sm font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || summary.trim().length < 100 || agentName.trim().length === 0}
            data-qa="complete-task-submit-btn"
            className="px-4 py-2 text-sm font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
          >
            {loading ? 'Completing...' : 'Complete Task'}
          </button>
        </div>
      </div>
    </div>
  );
}
