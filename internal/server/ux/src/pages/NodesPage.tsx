import { useState, useCallback, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../components/ThemeContext';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import OnboardingDialog from './OnboardingPage';
import { listNodes, revokeNode, renameNode } from '../lib/api';
import type { NodeResponse } from '../lib/types';
import { Settings } from 'lucide-react';

const MONO = "'JetBrains Mono', 'Geist Mono', ui-monospace, monospace";
const SANS = "'Geist', 'Inter', sans-serif";

export default function NodesPage() {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const navigate = useNavigate();
  const [onboardingOpen, setOnboardingOpen] = useState(false);

  const [nodes, setNodes] = useState<NodeResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [confirmingRevoke, setConfirmingRevoke] = useState<string | null>(null);
  const [editingName, setEditingName] = useState<string | null>(null);
  const [newName, setNewName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const cardBg = isDark ? '#1A1726' : '#FFFFFF';
  const border = isDark ? '#2E2B3D' : '#E2E8F0';
  const accent = '#7C5CF6';
  const textPrimary = isDark ? '#F0EEF8' : '#0F172A';
  const textSecondary = isDark ? '#9891B0' : '#64748B';
  const textMuted = isDark ? '#5C5578' : '#94A3B8';
  const greenColor = '#10B981';
  const redColor = '#EF4444';

  const fetchNodes = useCallback(async () => {
    try {
      const data = await listNodes();
      setNodes(data.nodes ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load nodes');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchNodes(); }, [fetchNodes]);

  const handleRevoke = useCallback(async () => {
    if (!confirmingRevoke) return;
    const nodeId = confirmingRevoke;
    setConfirmingRevoke(null);
    setRevoking(nodeId);
    try {
      await revokeNode(nodeId);
      await fetchNodes();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke node');
    } finally {
      setRevoking(null);
    }
  }, [confirmingRevoke, fetchNodes]);

  const handleRename = useCallback(async (nodeId: string) => {
    if (!newName.trim()) return;
    try {
      await renameNode(nodeId, newName.trim());
      setEditingName(null);
      setNewName('');
      await fetchNodes();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rename node');
    }
  }, [newName, fetchNodes]);

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return 'Never';
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();

    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return date.toLocaleDateString();
  };

  const activeNodes = nodes.filter(n => n.status === 'active');
  const revokedNodes = nodes.filter(n => n.status === 'revoked');

  return (
    <div className="flex-1 overflow-y-auto">
    <div className="max-w-5xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '32px' }}>
        <div>
          <h1 style={{
            fontSize: '24px',
            fontWeight: 700,
            fontFamily: SANS,
            color: textPrimary,
            marginBottom: '4px',
          }}>
            Daemon Nodes
          </h1>
          <p style={{
            fontSize: '14px',
            fontFamily: MONO,
            color: textSecondary,
          }}>
            Manage your connected daemon instances
          </p>
        </div>
        <button
          onClick={() => setOnboardingOpen(true)}
          data-qa="add-node-btn"
          style={{
            padding: '10px 20px',
            borderRadius: '8px',
            fontSize: '14px',
            fontWeight: 600,
            border: 'none',
            background: accent,
            color: '#FFFFFF',
            fontFamily: SANS,
            cursor: 'pointer',
          }}
        >
          + Add Node
        </button>
      </div>

      {error && (
        <div style={{
          padding: '12px 16px',
          borderRadius: '8px',
          background: isDark ? 'rgba(248,113,113,0.1)' : '#FEF2F2',
          border: `1px solid ${isDark ? 'rgba(248,113,113,0.2)' : '#FECACA'}`,
          color: isDark ? '#FCA5A5' : '#DC2626',
          fontSize: '13px',
          fontFamily: MONO,
          marginBottom: '24px',
        }}>
          {error}
        </div>
      )}

      {loading ? (
        <div style={{ textAlign: 'center', padding: '48px', color: textMuted, fontFamily: MONO }}>
          Loading...
        </div>
      ) : nodes.length === 0 ? (
        <div style={{
          textAlign: 'center',
          padding: '48px',
          background: cardBg,
          border: `1px solid ${border}`,
          borderRadius: '12px',
        }}>
          <p style={{ color: textSecondary, fontFamily: MONO, marginBottom: '16px' }}>
            No daemon nodes registered yet
          </p>
          <button
            onClick={() => setOnboardingOpen(true)}
            data-qa="connect-first-daemon-btn"
            style={{
              padding: '10px 20px',
              borderRadius: '8px',
              fontSize: '14px',
              border: `1px solid ${accent}`,
              background: 'transparent',
              color: accent,
              fontFamily: MONO,
              cursor: 'pointer',
            }}
          >
            Connect your first daemon
          </button>
        </div>
      ) : (
        <>
          {activeNodes.length > 0 && (
            <div style={{ marginBottom: '32px' }}>
              <h2 data-qa="active-nodes-heading" style={{
                fontSize: '14px',
                fontWeight: 600,
                fontFamily: MONO,
                color: textSecondary,
                marginBottom: '12px',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}>
                Active ({activeNodes.length})
              </h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {activeNodes.map(node => (
                  <NodeCard
                    key={node.id}
                    node={node}
                    isDark={isDark}
                    colors={{ cardBg, border, textPrimary, textSecondary, textMuted, greenColor, redColor, accent }}
                    revoking={revoking === node.id}
                    editingName={editingName === node.id}
                    newName={newName}
                    onRevoke={() => setConfirmingRevoke(node.id)}
                    onSettings={() => navigate(`/nodes/${node.id}/settings`)}
                    onStartEdit={() => { setEditingName(node.id); setNewName(node.name); }}
                    onCancelEdit={() => { setEditingName(null); setNewName(''); }}
                    onSaveName={() => handleRename(node.id)}
                    onNameChange={setNewName}
                    formatDate={formatDate}
                  />
                ))}
              </div>
            </div>
          )}

          {revokedNodes.length > 0 && (
            <div>
              <h2 data-qa="revoked-nodes-heading" style={{
                fontSize: '14px',
                fontWeight: 600,
                fontFamily: MONO,
                color: textMuted,
                marginBottom: '12px',
                textTransform: 'uppercase',
                letterSpacing: '0.05em',
              }}>
                Revoked ({revokedNodes.length})
              </h2>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', opacity: 0.6 }}>
                {revokedNodes.map(node => (
                  <NodeCard
                    key={node.id}
                    node={node}
                    isDark={isDark}
                    colors={{ cardBg, border, textPrimary, textSecondary, textMuted, greenColor, redColor, accent }}
                    revoking={false}
                    editingName={false}
                    newName=""
                    formatDate={formatDate}
                  />
                ))}
              </div>
            </div>
          )}
        </>
      )}

      <DeleteConfirmModal
        open={confirmingRevoke !== null}
        title="Revoke Node"
        description="Are you sure you want to revoke this node? The daemon will be disconnected immediately."
        confirmLabel="Revoke"
        onConfirm={handleRevoke}
        onCancel={() => setConfirmingRevoke(null)}
        loading={revoking !== null}
      />

      <OnboardingDialog
        open={onboardingOpen}
        onClose={() => setOnboardingOpen(false)}
        onSuccess={fetchNodes}
      />
    </div>
    </div>
  );
}

interface NodeCardProps {
  node: NodeResponse;
  isDark: boolean;
  colors: Record<string, string>;
  revoking: boolean;
  editingName: boolean;
  newName: string;
  onRevoke?: () => void;
  onSettings?: () => void;
  onStartEdit?: () => void;
  onCancelEdit?: () => void;
  onSaveName?: () => void;
  onNameChange?: (name: string) => void;
  formatDate: (date: string | null) => string;
}

function NodeCard({
  node,
  isDark,
  colors,
  revoking,
  editingName,
  newName,
  onRevoke,
  onSettings,
  onStartEdit,
  onCancelEdit,
  onSaveName,
  onNameChange,
  formatDate,
}: NodeCardProps) {
  const { cardBg, border, textPrimary, textMuted, greenColor, redColor, accent } = colors;
  const isActive = node.status === 'active';

  return (
    <div
      data-qa={`node-card-${node.id}`}
      style={{
        background: cardBg,
        border: `1px solid ${border}`,
        borderRadius: '10px',
        padding: '16px 20px',
        display: 'flex',
        alignItems: 'center',
        gap: '16px',
      }}
    >
      {/* Status indicator */}
      <div data-qa={`node-status-${node.status}`} style={{
        width: '10px',
        height: '10px',
        borderRadius: '50%',
        background: isActive ? greenColor : redColor,
        boxShadow: isActive ? `0 0 8px ${greenColor}50` : undefined,
        flexShrink: 0,
      }} />

      {/* Info */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {editingName ? (
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
            <input
              type="text"
              value={newName}
              onChange={(e) => onNameChange?.(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && onSaveName?.()}
              autoFocus
              data-qa="node-name-edit-input"
              style={{
                padding: '6px 10px',
                borderRadius: '6px',
                border: `1px solid ${accent}`,
                background: isDark ? '#13111C' : '#F8FAFC',
                color: textPrimary,
                fontFamily: MONO,
                fontSize: '14px',
                outline: 'none',
              }}
            />
            <button data-qa="save-name-btn" onClick={onSaveName} style={{ color: greenColor, background: 'none', border: 'none', cursor: 'pointer', fontFamily: MONO, fontSize: '12px' }}>
              Save
            </button>
            <button data-qa="cancel-edit-btn" onClick={onCancelEdit} style={{ color: textMuted, background: 'none', border: 'none', cursor: 'pointer', fontFamily: MONO, fontSize: '12px' }}>
              Cancel
            </button>
          </div>
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <span data-qa="node-name" style={{ fontWeight: 600, fontFamily: MONO, fontSize: '14px', color: textPrimary }}>
              {node.name || 'Unnamed node'}
            </span>
            {isActive && onStartEdit && (
              <button
                onClick={onStartEdit}
                data-qa="edit-name-btn"
                style={{ color: textMuted, background: 'none', border: 'none', cursor: 'pointer', fontSize: '12px' }}
              >
                edit
              </button>
            )}
          </div>
        )}
        <div style={{ display: 'flex', gap: '16px', marginTop: '4px' }}>
          <span data-qa="node-mode" style={{ fontSize: '12px', fontFamily: MONO, color: textMuted }}>
            {node.mode === 'shared' ? 'Shared' : 'Personal'}
          </span>
          <span data-qa="node-last-seen" style={{ fontSize: '12px', fontFamily: MONO, color: textMuted }}>
            Last seen: {formatDate(node.last_seen_at)}
          </span>
          {node.revoked_at && (
            <span data-qa="node-revoked-at" style={{ fontSize: '12px', fontFamily: MONO, color: redColor }}>
              Revoked: {formatDate(node.revoked_at)}
            </span>
          )}
        </div>
      </div>

      {/* Actions */}
      {isActive && (
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {onSettings && (
            <button
              onClick={onSettings}
              data-qa="node-settings-btn"
              style={{
                display: 'flex', alignItems: 'center', gap: '6px',
                padding: '8px 16px',
                borderRadius: '8px',
                fontSize: '13px',
                fontWeight: 500,
                border: `1px solid ${border}`,
                background: 'transparent',
                color: textMuted,
                fontFamily: MONO,
                cursor: 'pointer',
              }}
            >
              <Settings size={14} />
              Settings
            </button>
          )}
          {onRevoke && (
            <button
              onClick={onRevoke}
              disabled={revoking}
              data-qa="revoke-node-btn"
              style={{
                padding: '8px 16px',
                borderRadius: '6px',
                fontSize: '12px',
                fontWeight: 500,
                border: `1px solid ${revoking ? border : redColor}`,
                background: 'transparent',
                color: revoking ? textMuted : redColor,
                fontFamily: MONO,
                cursor: revoking ? 'not-allowed' : 'pointer',
              }}
            >
              {revoking ? 'Revoking...' : 'Revoke'}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
