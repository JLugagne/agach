import { useState } from 'react';
import { X, Clock, Cpu, Database, DollarSign } from 'lucide-react';
import type { ChatSessionResponse } from '../../lib/types';

interface SessionPickerModalProps {
  sessions: ChatSessionResponse[];
  onSelect: (sessionId: string | null) => void;
  onClose: () => void;
}

const STATE_BADGE: Record<string, { label: string; bg: string }> = {
  active: { label: 'Active', bg: 'var(--status-progress)' },
  ended: { label: 'Ended', bg: 'var(--text-muted)' },
  timeout: { label: 'Timeout', bg: 'var(--status-blocked)' },
};

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
}

function estimateCost(s: ChatSessionResponse): string {
  // Rough estimate using claude pricing (~$3/$15 per 1M for input/output)
  const cost =
    (s.input_tokens * 3) / 1_000_000 +
    (s.output_tokens * 15) / 1_000_000 +
    (s.cache_read_tokens * 0.3) / 1_000_000 +
    (s.cache_write_tokens * 3.75) / 1_000_000;
  return '$' + cost.toFixed(4);
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export default function SessionPickerModal({ sessions, onSelect, onClose }: SessionPickerModalProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  return (
    <div
      data-qa="session-picker-modal"
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'rgba(0,0,0,0.5)',
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        style={{
          backgroundColor: 'var(--bg-primary)',
          borderRadius: '12px',
          border: '1px solid var(--border-subtle)',
          width: '100%',
          maxWidth: '520px',
          maxHeight: '80vh',
          display: 'flex',
          flexDirection: 'column',
          fontFamily: 'Inter, sans-serif',
          boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '16px 20px',
            borderBottom: '1px solid var(--border-subtle)',
            flexShrink: 0,
          }}
        >
          <h2 style={{ margin: 0, fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
            Previous Sessions
          </h2>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text-muted)',
              padding: '4px',
              display: 'flex',
              alignItems: 'center',
              borderRadius: '4px',
            }}
          >
            <X size={18} />
          </button>
        </div>

        {/* Session list */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '12px 20px' }}>
          <p style={{ fontSize: '13px', color: 'var(--text-muted)', margin: '0 0 12px 0' }}>
            Select a session to restore or start a new one.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {sessions.map((s) => {
              const badge = STATE_BADGE[s.state] ?? STATE_BADGE.ended;
              const isSelected = selectedId === s.id;
              return (
                <div
                  key={s.id}
                  data-qa="session-option"
                  onClick={() => setSelectedId(s.id)}
                  style={{
                    padding: '12px',
                    borderRadius: '8px',
                    border: isSelected ? '2px solid var(--primary-hover)' : '1px solid var(--border-subtle)',
                    backgroundColor: isSelected ? 'var(--bg-secondary)' : 'var(--bg-primary)',
                    cursor: 'pointer',
                    transition: 'border-color 0.15s, background-color 0.15s',
                  }}
                >
                  {/* Top row: date + state badge */}
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '13px', color: 'var(--text-primary)', fontWeight: 500 }}>
                      <Clock size={13} style={{ color: 'var(--text-muted)' }} />
                      {formatDate(s.created_at)}
                    </div>
                    <span
                      style={{
                        fontSize: '10px',
                        fontWeight: 600,
                        padding: '2px 8px',
                        borderRadius: '9999px',
                        backgroundColor: badge.bg,
                        color: 'var(--primary-text)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.5px',
                      }}
                    >
                      {badge.label}
                    </span>
                  </div>

                  {/* Bottom row: model, tokens, cost */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', fontSize: '11px', color: 'var(--text-muted)' }}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                      <Cpu size={11} /> {s.model || '--'}
                    </span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                      <Database size={11} /> {formatTokens(s.input_tokens + s.output_tokens)} tokens
                    </span>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '3px' }}>
                      <DollarSign size={11} /> {estimateCost(s)}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Footer */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            gap: '8px',
            padding: '12px 20px',
            borderTop: '1px solid var(--border-subtle)',
            flexShrink: 0,
          }}
        >
          <button
            data-qa="start-new-session-btn"
            onClick={() => onSelect(null)}
            style={{
              padding: '8px 16px',
              borderRadius: '6px',
              border: '1px solid var(--border-subtle)',
              backgroundColor: 'var(--bg-secondary)',
              color: 'var(--text-primary)',
              fontSize: '13px',
              fontWeight: 500,
              cursor: 'pointer',
            }}
          >
            Start New Session
          </button>
          <button
            data-qa="restore-session-btn"
            disabled={!selectedId}
            onClick={() => {
              if (selectedId) onSelect(selectedId);
            }}
            style={{
              padding: '8px 16px',
              borderRadius: '6px',
              border: 'none',
              backgroundColor: selectedId ? 'var(--primary-hover)' : 'var(--text-muted)',
              color: 'var(--primary-text)',
              fontSize: '13px',
              fontWeight: 500,
              cursor: selectedId ? 'pointer' : 'not-allowed',
              opacity: selectedId ? 1 : 0.5,
            }}
          >
            Restore Session
          </button>
        </div>
      </div>
    </div>
  );
}
