import { useState, useEffect, useCallback, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { getToolUsage, getBoard, getProjectSummary, getTimeline, getColdStartStats } from '../lib/api';
import type { ToolUsageStatResponse, ProjectSummaryResponse, TaskWithDetailsResponse, TimelineEntryResponse } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';
import { formatDuration } from '../lib/utils';

interface RoleColdStartStat {
  assigned_role: string;
  count: number;
  min_input_tokens: number;
  max_input_tokens: number;
  avg_input_tokens: number;
  min_output_tokens: number;
  max_output_tokens: number;
  avg_output_tokens: number;
  min_cache_read_tokens: number;
  max_cache_read_tokens: number;
  avg_cache_read_tokens: number;
}

const TIME_RANGE_OPTIONS = [7, 14, 30] as const;
type TimeRange = (typeof TIME_RANGE_OPTIONS)[number];

export default function StatisticsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [loading, setLoading] = useState(true);
  const [toolUsage, setToolUsage] = useState<ToolUsageStatResponse[]>([]);
  const [summary, setSummary] = useState<ProjectSummaryResponse | null>(null);
  const [tokenTotals, setTokenTotals] = useState({ input: 0, output: 0, cacheRead: 0, cacheWrite: 0 });
  const [tasksByRole, setTasksByRole] = useState<Record<string, number>>({});
  const [tasksByPriority, setTasksByPriority] = useState<Record<string, number>>({});
  const [timeRange, setTimeRange] = useState<TimeRange>(14);
  const [timeline, setTimeline] = useState<TimelineEntryResponse[]>([]);
  const [timelineError, setTimelineError] = useState(false);
  const [coldStartStats, setColdStartStats] = useState<RoleColdStartStat[]>([]);
  const [timingStats, setTimingStats] = useState<{
    avgDuration: number;
    totalSaved: number;
    totalHumanEstimate: number;
    totalDuration: number;
    fastest: { title: string; duration: number } | null;
    slowest: { title: string; duration: number } | null;
  } | null>(null);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const [usage, summaryData, board, coldStart] = await Promise.all([
        getToolUsage(projectId).catch(() => [] as ToolUsageStatResponse[]),
        getProjectSummary(projectId).catch(() => null),
        getBoard(projectId).catch(() => null),
        getColdStartStats(projectId).catch(() => [] as RoleColdStartStat[]),
      ]);
      setToolUsage(usage ?? []);
      setSummary(summaryData);
      setColdStartStats((coldStart ?? []) as RoleColdStartStat[]);

      // Aggregate token usage and role/priority stats from all tasks
      if (board?.columns) {
        let input = 0, output = 0, cacheRead = 0, cacheWrite = 0;
        const roles: Record<string, number> = {};
        const priorities: Record<string, number> = {};

        // Timing aggregation
        let totalDuration = 0;
        let totalHumanEstimate = 0;
        let durationCount = 0;
        let fastestTask: { title: string; duration: number } | null = null;
        let slowestTask: { title: string; duration: number } | null = null;

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

            // Only count tasks with duration data
            if ((task.duration_seconds ?? 0) > 0) {
              const dur = task.duration_seconds;
              totalDuration += dur;
              durationCount++;
              if (!fastestTask || dur < fastestTask.duration) {
                fastestTask = { title: task.title, duration: dur };
              }
              if (!slowestTask || dur > slowestTask.duration) {
                slowestTask = { title: task.title, duration: dur };
              }
              if ((task.human_estimate_seconds ?? 0) > 0) {
                totalHumanEstimate += task.human_estimate_seconds;
              }
            }
          }
        }
        setTokenTotals({ input, output, cacheRead, cacheWrite });
        setTasksByRole(roles);
        setTasksByPriority(priorities);

        if (durationCount > 0) {
          setTimingStats({
            avgDuration: Math.round(totalDuration / durationCount),
            totalSaved: totalHumanEstimate - totalDuration,
            totalHumanEstimate,
            totalDuration,
            fastest: fastestTask,
            slowest: slowestTask,
          });
        } else {
          setTimingStats(null);
        }
      }
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  const fetchTimeline = useCallback(async () => {
    if (!projectId) return;
    setTimelineError(false);
    try {
      const data = await getTimeline(projectId, timeRange);
      setTimeline(data ?? []);
    } catch {
      setTimelineError(true);
      setTimeline([]);
    }
  }, [projectId, timeRange]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    fetchTimeline();
  }, [fetchTimeline]);

  useWebSocket(
    useCallback(
      (event) => {
        const type = event.type || '';
        if (type.startsWith('task_') || type.startsWith('tool_')) {
          fetchData();
          fetchTimeline();
        }
      },
      [fetchData, fetchTimeline],
    ),
  );

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

        {/* Timing */}
        {timingStats && (
          <Section title="Timing">
            <div className="grid grid-cols-2 gap-4 mb-4">
              <StatCard label="Avg Solve Time" value={formatDuration(timingStats.avgDuration)} />
              {timingStats.totalHumanEstimate > 0 && (
                <StatCard
                  label="Time Saved"
                  value={`${timingStats.totalSaved >= 0 ? '+' : ''}${((timingStats.totalSaved / timingStats.totalHumanEstimate) * 100).toFixed(0)}%`}
                />
              )}
              {timingStats.totalHumanEstimate > 0 && (
                <StatCard label="Total Human Est." value={formatDuration(timingStats.totalHumanEstimate)} />
              )}
              <StatCard label="Total Duration" value={formatDuration(timingStats.totalDuration)} />
            </div>
            {(timingStats.fastest || timingStats.slowest) && (
              <div className="flex gap-4 flex-wrap">
                {timingStats.fastest && (
                  <div className="bg-[var(--bg-tertiary)] rounded-md px-3 py-2 flex-1 min-w-[180px]">
                    <div className="text-xs text-[var(--text-muted)] mb-1">Fastest task</div>
                    <div className="text-sm text-[var(--text-secondary)] truncate" title={timingStats.fastest.title}>
                      {timingStats.fastest.title}
                    </div>
                    <div className="text-xs font-mono text-[var(--status-done)] mt-0.5">{formatDuration(timingStats.fastest.duration)}</div>
                  </div>
                )}
                {timingStats.slowest && timingStats.slowest.title !== timingStats.fastest?.title && (
                  <div className="bg-[var(--bg-tertiary)] rounded-md px-3 py-2 flex-1 min-w-[180px]">
                    <div className="text-xs text-[var(--text-muted)] mb-1">Slowest task</div>
                    <div className="text-sm text-[var(--text-secondary)] truncate" title={timingStats.slowest.title}>
                      {timingStats.slowest.title}
                    </div>
                    <div className="text-xs font-mono text-[var(--text-muted)] mt-0.5">{formatDuration(timingStats.slowest.duration)}</div>
                  </div>
                )}
              </div>
            )}
          </Section>
        )}

        {/* Cold Start Cost per Agent Role */}
        {coldStartStats.length > 0 && (
          <Section title="Cold Start Cost per Agent Role">
            <p className="text-xs text-[var(--text-muted)] mb-3">
              Cold start cost is the token usage of the first exchange when an agent starts fresh. Higher cache read tokens indicate better context reuse.
            </p>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-xs text-[var(--text-muted)] uppercase tracking-wider" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                    <th className="text-left py-2 pr-4">Role</th>
                    <th className="text-right py-2 px-2">Runs</th>
                    <th className="text-right py-2 px-2">Min Input</th>
                    <th className="text-right py-2 px-2">Avg Input</th>
                    <th className="text-right py-2 px-2">Max Input</th>
                    <th className="text-right py-2 px-2">Avg Cache Read</th>
                  </tr>
                </thead>
                <tbody>
                  {coldStartStats.map((stat) => (
                    <tr key={stat.assigned_role} className="border-t border-[var(--border-primary)]">
                      <td className="py-2 pr-4 text-[var(--text-secondary)]">{stat.assigned_role}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{stat.count.toLocaleString()}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{stat.min_input_tokens.toLocaleString()}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{Math.round(stat.avg_input_tokens).toLocaleString()}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{stat.max_input_tokens.toLocaleString()}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{Math.round(stat.avg_cache_read_tokens).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Section>
        )}

        {/* Time range selector + charts */}
        <div className="mb-2 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-[var(--text-muted)] uppercase tracking-wider" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
            Activity
          </h2>
          <div className="flex gap-1">
            {TIME_RANGE_OPTIONS.map((d) => (
              <button
                key={d}
                onClick={() => setTimeRange(d)}
                data-qa={`time-range-${d}d-btn`}
                className="px-3 py-1 rounded-full text-xs font-mono transition-colors"
                style={{
                  backgroundColor: timeRange === d ? 'var(--primary)' : 'var(--bg-tertiary)',
                  color: timeRange === d ? '#fff' : 'var(--text-muted)',
                }}
              >
                {d}d
              </button>
            ))}
          </div>
        </div>

        {timelineError || timeline.length === 0 ? (
          <p className="text-sm text-[var(--text-muted)] mb-8">No activity data yet.</p>
        ) : (
          <>
            {/* Velocity chart */}
            <VelocityChart data={timeline} />
            {/* Burndown chart */}
            <BurndownChart data={timeline} totalTasks={totalTasks} />
          </>
        )}
      </div>
    </div>
  );
}

// ---------- Velocity Chart ----------

function VelocityChart({ data }: { data: TimelineEntryResponse[] }) {
  const maxCompleted = Math.max(...data.map((d) => d.tasks_completed), 1);

  return (
    <div className="mb-8">
      <p className="text-xs text-[var(--text-muted)] mb-3" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
        Velocity — tasks completed per day
      </p>
      <div className="space-y-1.5">
        {data.map((entry) => {
          const pct = (entry.tasks_completed / maxCompleted) * 100;
          const label = formatDateLabel(entry.date);
          return (
            <div key={entry.date} className="flex items-center gap-3">
              <span className="text-xs text-[var(--text-muted)] w-20 shrink-0 font-mono">{label}</span>
              <div className="flex-1 h-5 bg-[var(--bg-tertiary)] rounded overflow-hidden">
                {entry.tasks_completed > 0 && (
                  <div
                    className="h-full rounded transition-all"
                    style={{ width: `${pct}%`, backgroundColor: 'var(--primary)' }}
                  />
                )}
              </div>
              <span className="text-xs text-[var(--text-muted)] w-6 text-right font-mono">
                {entry.tasks_completed}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ---------- Burndown Chart ----------

interface BurndownPoint {
  date: string;
  remaining: number;
}

function BurndownChart({ data, totalTasks }: { data: TimelineEntryResponse[]; totalTasks: number }) {
  const points = useMemo<BurndownPoint[]>(() => {
    // Work forward: start from (totalTasks - sum of all completed before the window)
    // We use a running total starting from a reasonable baseline.
    // We compute: baseline = totalTasks + (total completed in window) - (total created in window)
    // Then walk forward applying daily deltas.
    const totalCreated = data.reduce((s, d) => s + d.tasks_created, 0);
    const totalCompleted = data.reduce((s, d) => s + d.tasks_completed, 0);
    // Estimate tasks at the start of the window
    const startRemaining = Math.max(0, totalTasks + totalCompleted - totalCreated);

    let running = startRemaining;
    return data.map((entry) => {
      running = Math.max(0, running + entry.tasks_created - entry.tasks_completed);
      return { date: entry.date, remaining: running };
    });
  }, [data, totalTasks]);

  const width = 560;
  const height = 120;
  const paddingLeft = 36;
  const paddingBottom = 20;
  const chartW = width - paddingLeft;
  const chartH = height - paddingBottom;

  const maxVal = Math.max(...points.map((p) => p.remaining), 1);
  const n = points.length;

  const toX = (i: number) => paddingLeft + (i / Math.max(n - 1, 1)) * chartW;
  const toY = (v: number) => chartH - (v / maxVal) * chartH;

  const polylinePoints = points.map((p, i) => `${toX(i)},${toY(p.remaining)}`).join(' ');
  const areaPoints = `${toX(0)},${chartH} ${polylinePoints} ${toX(n - 1)},${chartH}`;

  // Y-axis labels: 0, mid, max
  const yLabels = [
    { v: 0, y: chartH },
    { v: Math.round(maxVal / 2), y: toY(maxVal / 2) },
    { v: maxVal, y: toY(maxVal) },
  ];

  // X-axis: show first, middle, last date labels
  const xLabels: { i: number; label: string }[] = [];
  if (n > 0) {
    xLabels.push({ i: 0, label: formatDateLabel(points[0].date) });
    if (n > 2) xLabels.push({ i: Math.floor((n - 1) / 2), label: formatDateLabel(points[Math.floor((n - 1) / 2)].date) });
    if (n > 1) xLabels.push({ i: n - 1, label: formatDateLabel(points[n - 1].date) });
  }

  return (
    <div className="mb-8">
      <p className="text-xs text-[var(--text-muted)] mb-3" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
        Burndown — remaining tasks over time
      </p>
      <div className="bg-[var(--bg-tertiary)] rounded-lg p-4 overflow-x-auto">
        <svg viewBox={`0 0 ${width} ${height}`} width="100%" style={{ minWidth: 280, display: 'block' }}>
          {/* Y-axis labels */}
          {yLabels.map(({ v, y }) => (
            <text
              key={v}
              x={paddingLeft - 4}
              y={y + 4}
              textAnchor="end"
              fontSize={9}
              fill="var(--text-muted)"
              fontFamily="JetBrains Mono, monospace"
            >
              {v}
            </text>
          ))}

          {/* Horizontal grid lines */}
          {yLabels.map(({ v, y }) => (
            <line
              key={`grid-${v}`}
              x1={paddingLeft}
              y1={y}
              x2={width}
              y2={y}
              stroke="var(--border-primary)"
              strokeWidth={0.5}
              strokeDasharray="3,3"
            />
          ))}

          {/* Area fill */}
          <polygon
            points={areaPoints}
            fill="var(--primary)"
            opacity={0.12}
          />

          {/* Line */}
          <polyline
            points={polylinePoints}
            fill="none"
            stroke="var(--primary)"
            strokeWidth={2}
            strokeLinejoin="round"
            strokeLinecap="round"
          />

          {/* X-axis labels */}
          {xLabels.map(({ i, label }) => (
            <text
              key={i}
              x={toX(i)}
              y={height - 4}
              textAnchor={i === 0 ? 'start' : i === n - 1 ? 'end' : 'middle'}
              fontSize={9}
              fill="var(--text-muted)"
              fontFamily="JetBrains Mono, monospace"
            >
              {label}
            </text>
          ))}
        </svg>
      </div>
    </div>
  );
}

// ---------- Shared helpers ----------

function formatDateLabel(dateStr: string): string {
  // dateStr is "YYYY-MM-DD"
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  const month = parseInt(parts[1], 10);
  const day = parseInt(parts[2], 10);
  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${monthNames[month - 1]} ${day}`;
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
