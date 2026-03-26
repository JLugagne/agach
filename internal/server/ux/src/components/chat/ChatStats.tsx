import { Cpu, MessageSquare, Database, DollarSign, Clock } from 'lucide-react';

interface ChatStatsProps {
  model: string;
  messageCount: number;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  totalCost: number;
  durationSeconds: number;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
}

function formatCost(cost: number): string {
  return '$' + cost.toFixed(4);
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return s > 0 ? `${m}m ${s}s` : `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm > 0 ? `${h}h ${rm}m` : `${h}h`;
}

function StatItem({
  icon,
  label,
  value,
  qa,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  qa: string;
}) {
  return (
    <div
      data-qa={qa}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '4px',
        whiteSpace: 'nowrap',
      }}
    >
      {icon}
      <span style={{ fontWeight: 500 }}>{label}:</span>
      <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{value}</span>
    </div>
  );
}

export default function ChatStats({
  model,
  messageCount,
  inputTokens,
  outputTokens,
  cacheReadTokens,
  totalCost,
  durationSeconds,
}: ChatStatsProps) {
  return (
    <div
      data-qa="chat-stats"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '16px',
        padding: '8px 16px',
        borderBottom: '1px solid var(--border-subtle)',
        backgroundColor: 'var(--bg-secondary)',
        fontSize: '12px',
        color: 'var(--text-muted)',
        flexShrink: 0,
        overflowX: 'auto',
        flexWrap: 'nowrap',
      }}
    >
      <StatItem icon={<Cpu size={13} />} label="Model" value={model} qa="stat-model" />
      <StatItem icon={<MessageSquare size={13} />} label="Messages" value={String(messageCount)} qa="stat-messages" />
      <StatItem icon={<Database size={13} />} label="In" value={formatTokens(inputTokens)} qa="stat-input-tokens" />
      <StatItem icon={<Database size={13} />} label="Out" value={formatTokens(outputTokens)} qa="stat-output-tokens" />
      <StatItem icon={<Database size={13} />} label="Cache" value={formatTokens(cacheReadTokens)} qa="stat-cache-tokens" />
      <StatItem icon={<DollarSign size={13} />} label="Cost" value={formatCost(totalCost)} qa="stat-cost" />
      <StatItem icon={<Clock size={13} />} label="Duration" value={formatDuration(durationSeconds)} qa="stat-duration" />
    </div>
  );
}
