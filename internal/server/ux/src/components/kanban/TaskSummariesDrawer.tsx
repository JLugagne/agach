import { useEffect, useState } from 'react';
import { X, ChevronDown, ChevronRight, FileText, Loader2 } from 'lucide-react';
import { getFeatureTaskSummaries } from '../../lib/api';
import type { TaskSummaryResponse } from '../../lib/types';

interface TaskSummariesDrawerProps {
  open: boolean;
  onClose: () => void;
  projectId: string;
  featureId: string;
  featureName: string;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function TaskSummaryCard({ summary }: { summary: TaskSummaryResponse }) {
  const [filesOpen, setFilesOpen] = useState(false);

  return (
    <div
      className="border border-[var(--border-primary)] rounded-lg p-4 bg-[var(--surface-bg)]"
      data-qa="task-summary-card"
    >
      <h3 className="text-[var(--text-primary)] text-sm font-['Inter'] font-semibold mb-1">
        {summary.title}
      </h3>
      <div className="flex items-center gap-3 mb-3">
        <span className="text-[var(--text-muted)] text-xs font-['Inter']">
          {summary.completed_by_agent}
        </span>
        <span className="text-[var(--text-dim)] text-xs font-['Inter']">
          {formatDate(summary.completed_at)}
        </span>
      </div>
      {summary.completion_summary && (
        <p className="text-[var(--text-secondary)] text-sm font-['Inter'] whitespace-pre-wrap mb-3">
          {summary.completion_summary}
        </p>
      )}
      {summary.files_modified && summary.files_modified.length > 0 && (
        <div>
          <button
            onClick={() => setFilesOpen(!filesOpen)}
            className="flex items-center gap-1.5 text-[var(--text-muted)] hover:text-[var(--text-secondary)] text-xs font-['Inter'] transition-colors"
          >
            {filesOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
            <FileText size={12} />
            {summary.files_modified.length} file{summary.files_modified.length !== 1 ? 's' : ''} modified
          </button>
          {filesOpen && (
            <ul className="mt-2 ml-5 space-y-0.5">
              {summary.files_modified.map((f) => (
                <li key={f} className="text-[var(--text-dim)] text-xs font-mono truncate" title={f}>
                  {f}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

export default function TaskSummariesDrawer({ open, onClose, projectId, featureId, featureName }: TaskSummariesDrawerProps) {
  const [summaries, setSummaries] = useState<TaskSummaryResponse[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setLoading(true);
    setError(null);
    getFeatureTaskSummaries(projectId, featureId)
      .then((data) => setSummaries(data ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load task summaries'))
      .finally(() => setLoading(false));
  }, [open, projectId, featureId]);

  if (!open) return null;

  return (
    <>
      {/* Overlay */}
      <div className="fixed inset-0 z-40 bg-[rgba(0,0,0,0.5)]" onClick={onClose} />

      {/* Drawer */}
      <div
        className="fixed top-0 right-0 z-50 h-full w-[720px] max-w-full bg-[var(--card-bg)] border-l border-[var(--border-primary)] shadow-2xl flex flex-col overflow-hidden animate-slide-in"
        data-qa="task-summaries-drawer"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-[var(--text-primary)] text-lg font-['Newsreader'] font-medium">Implementation Summary</h2>
            <span className="text-[var(--text-muted)] text-xs font-['Inter']">{featureName}</span>
          </div>
          <button
            onClick={onClose}
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
            aria-label="Close"
          >
            <X size={20} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-5">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={24} className="animate-spin text-[var(--text-muted)]" />
            </div>
          )}
          {error && (
            <p className="text-[var(--text-error,#ef4444)] text-sm font-['Inter']">{error}</p>
          )}
          {!loading && !error && summaries.length === 0 && (
            <p className="text-[var(--text-dim)] text-sm font-['Inter'] italic">No completed tasks yet</p>
          )}
          {!loading && !error && summaries.length > 0 && (
            <div className="flex flex-col gap-3">
              {summaries.map((s) => (
                <TaskSummaryCard key={s.task_id} summary={s} />
              ))}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
