import { useState, useEffect, type FormEvent } from 'react';
import { listAPIKeys, createAPIKey, revokeAPIKey } from '../lib/api';
import { Plus, Trash2, Copy, Check } from 'lucide-react';

interface APIKey {
  id: string;
  name: string;
  scopes: string[];
  expires_at: string | null;
  last_used_at: string | null;
  created_at: string;
}

const AVAILABLE_SCOPES = ['kanban:read', 'kanban:write'];

function formatDate(s: string | null) {
  if (!s) return '—';
  return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);

  // Create form
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [scopes, setScopes] = useState<Set<string>>(new Set(['kanban:read', 'kanban:write']));
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [newKeyValue, setNewKeyValue] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const fetchKeys = () => {
    listAPIKeys().then(k => {
      setKeys(k ?? []);
      setLoading(false);
    }).catch(() => setLoading(false));
  };

  useEffect(() => { fetchKeys(); }, []);

  function toggleScope(s: string) {
    setScopes(prev => {
      const next = new Set(prev);
      if (next.has(s)) next.delete(s);
      else next.add(s);
      return next;
    });
  }

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      const result = await createAPIKey({ name: name.trim(), scopes: Array.from(scopes) });
      setNewKeyValue(result.api_key);
      setName('');
      setScopes(new Set(['kanban:read', 'kanban:write']));
      setShowForm(false);
      fetchKeys();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Failed to create key.');
    } finally {
      setCreating(false);
    }
  }

  async function handleRevoke(id: string) {
    try {
      await revokeAPIKey(id);
      setKeys(prev => prev.filter(k => k.id !== id));
    } catch {
      // ignore
    }
  }

  function copyKey() {
    if (!newKeyValue) return;
    navigator.clipboard.writeText(newKeyValue).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-2xl mx-auto px-4 sm:px-8 py-6 sm:py-12 flex flex-col gap-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-[24px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
              API Keys
            </h1>
            <p className="text-xs text-[var(--text-muted)] mt-1" style={{ fontFamily: 'Inter, sans-serif' }}>
              Programmatic access for agents and integrations
            </p>
          </div>
          <button
            onClick={() => { setShowForm(v => !v); setCreateError(null); }}
            data-qa="create-api-key-btn"
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Key
          </button>
        </div>

        {/* New key revealed */}
        {newKeyValue && (
          <div className="rounded-xl border border-green-500/30 bg-green-500/5 px-5 py-4 flex flex-col gap-3">
            <p className="text-[13px] font-medium text-green-500" style={{ fontFamily: 'Inter, sans-serif' }}>
              Key created — copy it now, it won't be shown again.
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-[12px] text-[var(--text-primary)] bg-[var(--bg-tertiary)] border border-[var(--border-primary)] rounded-lg px-3 py-2 truncate" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                {newKeyValue}
              </code>
              <button
                onClick={copyKey}
                data-qa="copy-api-key-btn"
                className="flex-shrink-0 p-2 rounded-lg border border-[var(--border-primary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                {copied ? <Check size={14} className="text-green-500" /> : <Copy size={14} />}
              </button>
              <button
                onClick={() => setNewKeyValue(null)}
                className="flex-shrink-0 px-3 py-2 rounded-lg text-[12px] text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                Dismiss
              </button>
            </div>
          </div>
        )}

        {/* Create form */}
        {showForm && (
          <form onSubmit={handleCreate} className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] px-6 py-5 flex flex-col gap-4">
            <h2 className="text-[13px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              New API Key
            </h2>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                required
                autoFocus
                data-qa="api-key-name-input"
                placeholder="e.g. My Agent Key"
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Scopes
              </label>
              <div className="flex gap-2">
                {AVAILABLE_SCOPES.map(s => {
                  const on = scopes.has(s);
                  return (
                    <button
                      key={s}
                      type="button"
                      onClick={() => toggleScope(s)}
                      data-qa={`scope-${s}`}
                      className="px-2.5 py-1 rounded-full text-[11px] font-medium border transition-colors"
                      style={{
                        borderColor: on ? 'var(--primary)' : 'var(--border-primary)',
                        color: on ? 'var(--primary)' : 'var(--text-muted)',
                        backgroundColor: on ? 'color-mix(in srgb, var(--primary) 12%, transparent)' : 'transparent',
                      }}
                    >
                      {s}
                    </button>
                  );
                })}
              </div>
            </div>
            {createError && (
              <p className="text-[12px] text-red-500 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2" style={{ fontFamily: 'Inter, sans-serif' }}>
                {createError}
              </p>
            )}
            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="px-3 py-2 rounded-lg text-[13px] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating || !name.trim()}
                data-qa="create-api-key-submit-btn"
                className="px-4 py-2 rounded-lg text-[13px] font-semibold bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] disabled:opacity-50 transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {creating ? 'Creating…' : 'Create'}
              </button>
            </div>
          </form>
        )}

        {/* Keys list */}
        <section className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] overflow-hidden">
          {loading ? (
            <div className="px-6 py-8 text-center text-[13px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Loading…
            </div>
          ) : keys.length === 0 ? (
            <div className="px-6 py-8 text-center text-[13px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              No API keys yet.
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--border-primary)]">
                  {['Name', 'Scopes', 'Last used', 'Created', ''].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-[11px] font-semibold text-[var(--text-muted)] tracking-wide" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {keys.map((k, i) => (
                  <tr key={k.id} className={i < keys.length - 1 ? 'border-b border-[var(--border-primary)]' : ''}>
                    <td className="px-4 py-3 text-[13px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {k.name}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(k.scopes ?? []).map(s => (
                          <span key={s} className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--bg-tertiary)] text-[var(--text-secondary)] border border-[var(--border-primary)]" style={{ fontFamily: 'JetBrains Mono, monospace' }}>
                            {s}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-[12px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {formatDate(k.last_used_at)}
                    </td>
                    <td className="px-4 py-3 text-[12px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {formatDate(k.created_at)}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => handleRevoke(k.id)}
                        data-qa={`revoke-key-${k.id}`}
                        className="p-1.5 rounded text-[var(--text-muted)] hover:text-red-500 hover:bg-red-500/10 transition-colors"
                        title="Revoke"
                      >
                        <Trash2 size={13} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </div>
  );
}
