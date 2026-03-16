import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { getToolUsage, getBoard, getProjectSummary } from '../lib/api';
import type { ToolUsageStatResponse, ProjectSummaryResponse, TaskWithDetailsResponse } from '../lib/types';

export default function StatisticsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [loading, setLoading] = useState(true);
  const [toolUsage, setToolUsage] = useState<ToolUsageStatResponse[]>([]);
  const [summary, setSummary] = useState<ProjectSummaryResponse | null>(null);
  const [tokenTotals, setTokenTotals] = useState({ input: 0, output: 0, cacheRead: 0, cacheWrite: 0 });
  const [tasksByRole, setTasksByRole] = useState<Record<string, number>>({});
  const [tasksByPriority, setTasksByPriority] = useState<Record<string, number>>({});

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const [usage, summaryData, board] = await Promise.all([
        getToolUsage(projectId).catch(() => [] as ToolUsageStatResponse[]),
        getProjectSummary(projectId).catch(() => null),
        getBoard(projectId).catch(() => null),
      ]);
      setToolUsage(usage ?? []);
      setSummary(summaryData);

      // Aggregate token usage and role/priority stats from all tasks
      if (board?.columns) {
        let input = 0, output = 0, cacheRead = 0, cacheWrite = 0;
        const roles: Record<string, number> = {};
        const priorities: Record<string, number> = {};
        for (const col of board.columns) {
          for (const task of (col.tasks ?? []) as TaskWithDetailsResponse[]) {
            input += task.input_tokens || 0;
            output += task.output_tokens || 0;
            cacheRead += task.cache_read_tokens || 0;
            cacheWrite += task.cache_write_tokens || 0;
            if (task.assigned_role) {
              roles[task.assigned_role] = (roles[task.assigned_role] || 0) + 1;
            }
            if (task.priority) {
              priorities[task.priority] = (priorities[task.priority] || 0) + 1;
            }
          }
        }
        setTokenTotals({ input, output, cacheRead, cacheWrite });
        setTasksByRole(roles);
        setTasksByPriority(priorities);
      }
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={24} className="animate-spin text-[var(--text-muted)]" />
      </div>
    );
  }

  const totalTokens = tokenTotals.input + tokenTotals.output + tokenTotals.cacheRead + tokenTotals.cacheWrite;
  const totalCalls = toolUsage.reduce((sum, t) => sum + t.execution_count, 0);
  const totalTasks = summary ? summary.todo_count + summary.in_progress_count + summary.done_count + summary.blocked_count : 0;

  const priorityColors: Record<string, string> = {
    critical: '#FF3B30',
    high: '#FF9500',
    medium: '#007AFF',
    low: '#8E8E93',
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-8">
        <h1 className="text-xl font-semibold text-[var(--text-primary)] mb-6" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
          Statistics
        </h1>

        {/* Summary cards */}
        <div className="grid grid-cols-4 gap-4 mb-8">
          <StatCard label="Total Tasks" value={totalTasks} />
          <StatCard label="Done" value={summary?.done_count ?? 0} />
          <StatCard label="In Progress" value={summary?.in_progress_count ?? 0} />
          <StatCard label="Blocked" value={summary?.blocked_count ?? 0} />
        </div>

        {/* Token usage */}
        <Section title="Token Usage">
          {totalTokens === 0 ? (
            <p className="text-sm text-[var(--text-muted)]">No token usage recorded yet.</p>
          ) : (
            <div className="grid grid-cols-4 gap-4">
              <StatCard label="Input" value={formatNumber(tokenTotals.input)} />
              <StatCard label="Output" value={formatNumber(tokenTotals.output)} />
              <StatCard label="Cache Read" value={formatNumber(tokenTotals.cacheRead)} />
              <StatCard label="Cache Write" value={formatNumber(tokenTotals.cacheWrite)} />
            </div>
          )}
        </Section>

        {/* MCP Tool Usage */}
        <Section title="MCP Tool Calls">
          {toolUsage.length === 0 ? (
            <p className="text-sm text-[var(--text-muted)]">No tool usage recorded yet.</p>
          ) : (
            <>
              <p className="text-sm text-[var(--text-muted)] mb-3">{totalCalls} total calls across {toolUsage.length} tools</p>
              <div className="space-y-2">
                {toolUsage
                  .sort((a, b) => b.execution_count - a.execution_count)
                  .map((t) => {
                    const pct = totalCalls > 0 ? (t.execution_count / totalCalls) * 100 : 0;
                    return (
                      <div key={t.tool_name} className="flex items-center gap-3">
                        <span className="text-xs text-[var(--text-secondary)] w-36 truncate font-mono" title={t.tool_name}>
                          {t.tool_name}
                        </span>
                        <div className="flex-1 h-5 bg-[var(--bg-tertiary)] rounded overflow-hidden">
                          <div
                            className="h-full bg-[var(--primary)] rounded transition-all"
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        <span className="text-xs text-[var(--text-muted)] w-12 text-right font-mono">
                          {t.execution_count}
                        </span>
                      </div>
                    );
                  })}
              </div>
            </>
          )}
        </Section>

        {/* Tasks by Priority */}
        {Object.keys(tasksByPriority).length > 0 && (
          <Section title="Tasks by Priority">
            <div className="flex gap-3 flex-wrap">
              {['critical', 'high', 'medium', 'low'].filter((p) => tasksByPriority[p]).map((p) => (
                <div key={p} className="flex items-center gap-2 bg-[var(--bg-tertiary)] rounded-md px-3 py-2">
                  <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: priorityColors[p] }} />
                  <span className="text-sm text-[var(--text-secondary)] capitalize">{p}</span>
                  <span className="text-sm font-mono text-[var(--text-muted)]">{tasksByPriority[p]}</span>
                </div>
              ))}
            </div>
          </Section>
        )}

        {/* Tasks by Role */}
        {Object.keys(tasksByRole).length > 0 && (
          <Section title="Tasks by Role">
            <div className="flex gap-3 flex-wrap">
              {Object.entries(tasksByRole)
                .sort(([, a], [, b]) => b - a)
                .map(([role, count]) => (
                  <div key={role} className="flex items-center gap-2 bg-[var(--bg-tertiary)] rounded-md px-3 py-2">
                    <span className="text-sm text-[var(--text-secondary)]">{role}</span>
                    <span className="text-sm font-mono text-[var(--text-muted)]">{count}</span>
                  </div>
                ))}
            </div>
          </Section>
        )}
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-8">
      <h2 className="text-sm font-semibold text-[var(--text-muted)] uppercase tracking-wider mb-3" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
        {title}
      </h2>
      {children}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg p-4">
      <div className="text-xs text-[var(--text-muted)] mb-1">{label}</div>
      <div className="text-lg font-semibold text-[var(--text-primary)] font-mono">{value}</div>
    </div>
  );
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}
