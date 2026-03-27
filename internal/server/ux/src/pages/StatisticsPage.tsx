import { useState, useEffect, useCallback, useMemo } from 'react';
import { useParams } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { getBoard, getProjectSummary, getTimeline, getModelTokenStats, getModelPricing, getFeatureStats, listFeatures } from '../lib/api';
import type { ProjectSummaryResponse, TaskWithDetailsResponse, TimelineEntryResponse, ModelTokenStatResponse, ModelPricingResponse, FeatureStatsResponse, FeatureWithSummaryResponse } from '../lib/types';
import { useWebSocket } from '../hooks/useWebSocket';
import { formatDuration } from '../lib/utils';

const TIME_RANGE_OPTIONS = [7, 14, 30] as const;
type TimeRange = (typeof TIME_RANGE_OPTIONS)[number];

interface FeatureTaskBreakdown {
  id: string;
  name: string;
  status: string;
  total: number;
  done: number;
  inProgress: number;
  blocked: number;
  todo: number;
  backlog: number;
  cost: number;
}

export default function StatisticsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<ProjectSummaryResponse | null>(null);
  const [tasksByRole, setTasksByRole] = useState<Record<string, number>>({});
  const [tasksByPriority, setTasksByPriority] = useState<Record<string, number>>({});
  const [timeRange, setTimeRange] = useState<TimeRange>(14);
  const [timeline, setTimeline] = useState<TimelineEntryResponse[]>([]);
  const [timelineError, setTimelineError] = useState(false);
  const [modelTokenStats, setModelTokenStats] = useState<ModelTokenStatResponse[]>([]);
  const [modelPricing, setModelPricing] = useState<ModelPricingResponse[]>([]);
  const [featureStats, setFeatureStats] = useState<FeatureStatsResponse | null>(null);
  const [features, setFeatures] = useState<FeatureWithSummaryResponse[]>([]);
  const [allTasks, setAllTasks] = useState<TaskWithDetailsResponse[]>([]);
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
      const [summaryData, board, modelStats, pricing, featStats, featureList] = await Promise.all([
        getProjectSummary(projectId).catch(() => null),
        getBoard(projectId).catch(() => null),
        getModelTokenStats(projectId).catch(() => [] as ModelTokenStatResponse[]),
        getModelPricing().catch(() => [] as ModelPricingResponse[]),
        getFeatureStats(projectId).catch(() => null),
        listFeatures(projectId).catch(() => [] as FeatureWithSummaryResponse[]),
      ]);
      setSummary(summaryData);
      setModelTokenStats((modelStats ?? []) as ModelTokenStatResponse[]);
      setModelPricing((pricing ?? []) as ModelPricingResponse[]);
      setFeatureStats(featStats as FeatureStatsResponse | null);
      setFeatures((featureList ?? []) as FeatureWithSummaryResponse[]);

      if (board?.columns) {
        const roles: Record<string, number> = {};
        const priorities: Record<string, number> = {};
        const tasks: TaskWithDetailsResponse[] = [];

        let totalDuration = 0;
        let totalHumanEstimate = 0;
        let durationCount = 0;
        let fastestTask: { title: string; duration: number } | null = null;
        let slowestTask: { title: string; duration: number } | null = null;

        for (const col of board.columns) {
          for (const task of (col.tasks ?? []) as TaskWithDetailsResponse[]) {
            tasks.push(task);
            if (task.assigned_role) {
              roles[task.assigned_role] = (roles[task.assigned_role] || 0) + 1;
            }
            if (task.priority) {
              priorities[task.priority] = (priorities[task.priority] || 0) + 1;
            }
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
        setAllTasks(tasks);
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
        if (type.startsWith('task_') || type.startsWith('feature_')) {
          fetchData();
          fetchTimeline();
        }
      },
      [fetchData, fetchTimeline],
    ),
  );

  // Build pricing lookup by model_id
  const pricingMap = useMemo(() => {
    const map: Record<string, ModelPricingResponse> = {};
    for (const p of modelPricing) {
      map[p.model_id] = p;
    }
    return map;
  }, [modelPricing]);

  // Calculate costs per model
  const modelCosts = useMemo(() => {
    return modelTokenStats.map((stat) => {
      let pricing = pricingMap[stat.model];
      if (!pricing) {
        for (const [modelId, p] of Object.entries(pricingMap)) {
          if (stat.model.startsWith(modelId) || modelId.startsWith(stat.model)) {
            pricing = p;
            break;
          }
        }
      }

      const inputCost = pricing ? (stat.input_tokens / 1_000_000) * pricing.input_price_per_1m : 0;
      const outputCost = pricing ? (stat.output_tokens / 1_000_000) * pricing.output_price_per_1m : 0;
      const cacheReadCost = pricing ? (stat.cache_read_tokens / 1_000_000) * pricing.cache_read_price_per_1m : 0;
      const cacheWriteCost = pricing ? (stat.cache_write_tokens / 1_000_000) * pricing.cache_write_price_per_1m : 0;
      const totalCost = inputCost + outputCost + cacheReadCost + cacheWriteCost;

      return {
        ...stat,
        inputCost,
        outputCost,
        cacheReadCost,
        cacheWriteCost,
        totalCost,
        hasPricing: !!pricing,
      };
    });
  }, [modelTokenStats, pricingMap]);

  const totalEstimatedCost = useMemo(() => modelCosts.reduce((sum, m) => sum + m.totalCost, 0), [modelCosts]);
  const totalTaskCount = useMemo(() => modelCosts.reduce((sum, m) => sum + m.task_count, 0), [modelCosts]);

  // Compute per-task cost from tasks that have token data
  const costPerTask = useMemo(() => {
    if (totalTaskCount === 0) return 0;
    return totalEstimatedCost / totalTaskCount;
  }, [totalEstimatedCost, totalTaskCount]);

  // Tasks per feature breakdown
  const featureBreakdown = useMemo<FeatureTaskBreakdown[]>(() => {
    if (features.length === 0) return [];

    // Build a lookup of feature costs from tasks
    const featureCosts: Record<string, number> = {};
    for (const task of allTasks) {
      if (!task.feature_id) continue;
      // Calculate task cost
      let taskCost = 0;
      const totalTokens = (task.input_tokens || 0) + (task.output_tokens || 0) + (task.cache_read_tokens || 0) + (task.cache_write_tokens || 0);
      if (totalTokens > 0 && task.model) {
        let pricing = pricingMap[task.model];
        if (!pricing) {
          for (const [modelId, p] of Object.entries(pricingMap)) {
            if (task.model.startsWith(modelId) || modelId.startsWith(task.model)) {
              pricing = p;
              break;
            }
          }
        }
        if (pricing) {
          taskCost = ((task.input_tokens || 0) / 1_000_000) * pricing.input_price_per_1m
            + ((task.output_tokens || 0) / 1_000_000) * pricing.output_price_per_1m
            + ((task.cache_read_tokens || 0) / 1_000_000) * pricing.cache_read_price_per_1m
            + ((task.cache_write_tokens || 0) / 1_000_000) * pricing.cache_write_price_per_1m;
        }
      }
      featureCosts[task.feature_id] = (featureCosts[task.feature_id] || 0) + taskCost;
    }

    return features.map((f) => {
      const s = f.task_summary;
      return {
        id: f.id,
        name: f.name,
        status: f.status,
        total: s.backlog_count + s.todo_count + s.in_progress_count + s.done_count + s.blocked_count,
        done: s.done_count,
        inProgress: s.in_progress_count,
        blocked: s.blocked_count,
        todo: s.todo_count,
        backlog: s.backlog_count,
        cost: featureCosts[f.id] || 0,
      };
    }).sort((a, b) => b.total - a.total);
  }, [features, allTasks, pricingMap]);

  const costPerFeature = useMemo(() => {
    const doneFeatures = features.filter((f) => f.status === 'done').length;
    if (doneFeatures === 0) return 0;
    return totalEstimatedCost / doneFeatures;
  }, [features, totalEstimatedCost]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={24} className="animate-spin text-[var(--text-muted)]" />
      </div>
    );
  }

  const totalTasks = summary ? summary.todo_count + summary.in_progress_count + summary.done_count + summary.blocked_count : 0;

  const priorityColors: Record<string, string> = {
    critical: '#FF3B30',
    high: '#FF9500',
    medium: '#007AFF',
    low: '#8E8E93',
  };

  const featureStatusColors: Record<string, string> = {
    draft: 'var(--text-muted)',
    ready: 'var(--primary)',
    in_progress: '#007AFF',
    done: 'var(--status-done)',
    blocked: '#FF3B30',
  };

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-8">
        <h1 className="text-xl font-semibold text-[var(--text-primary)] mb-6" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
          Statistics
        </h1>

        {/* Cost summary cards */}
        <div className="grid grid-cols-4 gap-4 mb-8">
          <StatCard label="Total Cost" value={totalEstimatedCost > 0 ? `$${totalEstimatedCost.toFixed(2)}` : '$0'} />
          <StatCard label="Cost / Task" value={costPerTask > 0 ? `$${costPerTask.toFixed(2)}` : '-'} />
          <StatCard label="Cost / Feature (done)" value={costPerFeature > 0 ? `$${costPerFeature.toFixed(2)}` : '-'} />
          <StatCard label="Total Tasks" value={totalTasks} />
        </div>

        {/* Task summary cards */}
        <div className="grid grid-cols-4 gap-4 mb-8">
          <StatCard label="Done" value={summary?.done_count ?? 0} />
          <StatCard label="In Progress" value={summary?.in_progress_count ?? 0} />
          <StatCard label="Blocked" value={summary?.blocked_count ?? 0} />
          <StatCard label="Backlog" value={summary?.backlog_count ?? 0} />
        </div>

        {/* Feature Stats */}
        {featureStats && featureStats.total_count > 0 && (
          <Section title="Features">
            <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
              <MiniStatCard label="Total" value={featureStats.total_count} />
              <MiniStatCard label="Not Ready" value={featureStats.not_ready_count} color="var(--text-muted)" />
              <MiniStatCard label="Ready" value={featureStats.ready_count} color="var(--primary)" />
              <MiniStatCard label="In Progress" value={featureStats.in_progress_count} color="#007AFF" />
              <MiniStatCard label="Done" value={featureStats.done_count} color="var(--status-done)" />
              <MiniStatCard label="Blocked" value={featureStats.blocked_count} color="#FF3B30" />
            </div>
          </Section>
        )}

        {/* Tasks per Feature */}
        {featureBreakdown.length > 0 && (
          <Section title="Tasks per Feature">
            <div className="space-y-3">
              {featureBreakdown.map((f) => {
                const pctDone = f.total > 0 ? (f.done / f.total) * 100 : 0;
                const pctInProgress = f.total > 0 ? (f.inProgress / f.total) * 100 : 0;
                const pctBlocked = f.total > 0 ? (f.blocked / f.total) * 100 : 0;
                const pctTodo = f.total > 0 ? ((f.todo + f.backlog) / f.total) * 100 : 0;
                return (
                  <div key={f.id} className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg p-3">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 rounded-full" style={{ backgroundColor: featureStatusColors[f.status] || 'var(--text-muted)' }} />
                        <span className="text-sm text-[var(--text-primary)] font-medium truncate max-w-[300px]" title={f.name}>{f.name}</span>
                      </div>
                      <div className="flex items-center gap-3 text-xs font-mono text-[var(--text-muted)]">
                        <span>{f.done}/{f.total} tasks</span>
                        {f.cost > 0 && <span>${f.cost.toFixed(2)}</span>}
                      </div>
                    </div>
                    <div className="h-2 bg-[var(--bg-tertiary)] rounded-full overflow-hidden flex">
                      {pctDone > 0 && <div className="h-full" style={{ width: `${pctDone}%`, backgroundColor: 'var(--status-done)' }} />}
                      {pctInProgress > 0 && <div className="h-full" style={{ width: `${pctInProgress}%`, backgroundColor: '#007AFF' }} />}
                      {pctBlocked > 0 && <div className="h-full" style={{ width: `${pctBlocked}%`, backgroundColor: '#FF3B30' }} />}
                      {pctTodo > 0 && <div className="h-full" style={{ width: `${pctTodo}%`, backgroundColor: 'transparent' }} />}
                    </div>
                  </div>
                );
              })}
            </div>
          </Section>
        )}

        {/* Cost Breakdown by Model */}
        {modelTokenStats.length > 0 && (
          <Section title="Cost Breakdown by Model">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-xs text-[var(--text-muted)] uppercase tracking-wider" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                    <th className="text-left py-2 pr-4">Model</th>
                    <th className="text-right py-2 px-2">Tasks</th>
                    <th className="text-right py-2 px-2">Input</th>
                    <th className="text-right py-2 px-2">Output</th>
                    <th className="text-right py-2 px-2">Cache R</th>
                    <th className="text-right py-2 px-2">Cache W</th>
                    <th className="text-right py-2 px-2">Cost</th>
                  </tr>
                </thead>
                <tbody>
                  {modelCosts.map((stat) => (
                    <tr key={stat.model} className="border-t border-[var(--border-primary)]">
                      <td className="py-2 pr-4 text-[var(--text-secondary)] font-mono text-xs truncate max-w-[200px]" title={stat.model}>
                        {formatModelName(stat.model)}
                      </td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{stat.task_count}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{formatNumber(stat.input_tokens)}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{formatNumber(stat.output_tokens)}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{formatNumber(stat.cache_read_tokens)}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">{formatNumber(stat.cache_write_tokens)}</td>
                      <td className="py-2 px-2 text-right font-mono text-[var(--text-muted)]">
                        {stat.hasPricing ? `$${stat.totalCost.toFixed(2)}` : '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
                {totalEstimatedCost > 0 && (
                  <tfoot>
                    <tr className="border-t-2 border-[var(--border-primary)]">
                      <td colSpan={6} className="py-2 pr-4 text-xs text-[var(--text-muted)] text-right font-mono uppercase">Total</td>
                      <td className="py-2 px-2 text-right font-mono font-semibold text-[var(--text-primary)]">
                        ${totalEstimatedCost.toFixed(2)}
                      </td>
                    </tr>
                  </tfoot>
                )}
              </table>
            </div>
          </Section>
        )}

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

        {/* Feature Burndown Charts */}
        {featureBreakdown.filter((f) => f.status !== 'done' && f.total > 0).length > 0 && (
          <Section title="Feature Burndown">
            <p className="text-xs text-[var(--text-muted)] mb-4">Progress towards completion for active features</p>
            <div className="space-y-4">
              {featureBreakdown
                .filter((f) => f.status !== 'done' && f.total > 0)
                .map((f) => {
                  const remaining = f.total - f.done;
                  const pct = f.total > 0 ? (f.done / f.total) * 100 : 0;
                  return (
                    <div key={f.id} className="bg-[var(--bg-tertiary)] rounded-lg p-4">
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm text-[var(--text-primary)] font-medium">{f.name}</span>
                        <span className="text-xs font-mono text-[var(--text-muted)]">{remaining} remaining</span>
                      </div>
                      <div className="h-6 bg-[var(--bg-secondary)] rounded-full overflow-hidden relative">
                        <div
                          className="h-full rounded-full transition-all"
                          style={{
                            width: `${pct}%`,
                            backgroundColor: pct === 100 ? 'var(--status-done)' : 'var(--primary)',
                            minWidth: pct > 0 ? '8px' : undefined,
                          }}
                        />
                        <span className="absolute inset-0 flex items-center justify-center text-[10px] font-mono text-[var(--text-muted)]">
                          {pct.toFixed(0)}%
                        </span>
                      </div>
                      <div className="flex gap-4 mt-2 text-[10px] font-mono text-[var(--text-muted)]">
                        <span><span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: 'var(--status-done)' }} />{f.done} done</span>
                        <span><span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: '#007AFF' }} />{f.inProgress} in progress</span>
                        {f.blocked > 0 && <span><span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: '#FF3B30' }} />{f.blocked} blocked</span>}
                        <span><span className="inline-block w-1.5 h-1.5 rounded-full mr-1" style={{ backgroundColor: 'var(--text-muted)' }} />{f.todo + f.backlog} todo</span>
                      </div>
                    </div>
                  );
                })}
            </div>
          </Section>
        )}

        {/* Time range selector + activity charts */}
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
            <VelocityChart data={timeline} />
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
    const totalCreated = data.reduce((s, d) => s + d.tasks_created, 0);
    const totalCompleted = data.reduce((s, d) => s + d.tasks_completed, 0);
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

  const yLabels = [
    { v: 0, y: chartH },
    { v: Math.round(maxVal / 2), y: toY(maxVal / 2) },
    { v: maxVal, y: toY(maxVal) },
  ];

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
          {yLabels.map(({ v, y }) => (
            <text key={v} x={paddingLeft - 4} y={y + 4} textAnchor="end" fontSize={9} fill="var(--text-muted)" fontFamily="JetBrains Mono, monospace">{v}</text>
          ))}
          {yLabels.map(({ v, y }) => (
            <line key={`grid-${v}`} x1={paddingLeft} y1={y} x2={width} y2={y} stroke="var(--border-primary)" strokeWidth={0.5} strokeDasharray="3,3" />
          ))}
          <polygon points={areaPoints} fill="var(--primary)" opacity={0.12} />
          <polyline points={polylinePoints} fill="none" stroke="var(--primary)" strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
          {xLabels.map(({ i, label }) => (
            <text key={i} x={toX(i)} y={height - 4} textAnchor={i === 0 ? 'start' : i === n - 1 ? 'end' : 'middle'} fontSize={9} fill="var(--text-muted)" fontFamily="JetBrains Mono, monospace">{label}</text>
          ))}
        </svg>
      </div>
    </div>
  );
}

// ---------- Shared helpers ----------

function formatDateLabel(dateStr: string): string {
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  const month = parseInt(parts[1], 10);
  const day = parseInt(parts[2], 10);
  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${monthNames[month - 1]} ${day}`;
}

function formatModelName(model: string): string {
  return model
    .replace('claude-', '')
    .replace('-20250514', '')
    .replace('-20250620', '')
    .replace('-20241022', '')
    .replace('-20240229', '')
    .replace('-20240307', '');
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

function MiniStatCard({ label, value, color }: { label: string; value: number; color?: string }) {
  return (
    <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-center">
      <div className="text-[10px] text-[var(--text-muted)] mb-0.5">{label}</div>
      <div className="text-base font-semibold font-mono" style={{ color: color || 'var(--text-primary)' }}>{value}</div>
    </div>
  );
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return n.toString();
}
