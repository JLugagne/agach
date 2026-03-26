import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  ArrowLeft,
  Loader2,
  MessageSquare,
  Cpu,
  Clock,
  DollarSign,
  Database,
  StopCircle,
} from 'lucide-react';
import { getFeature, listChatSessions, startChatSession, listNodes } from '../lib/api';
import type { FeatureResponse, ChatSessionResponse, NodeResponse } from '../lib/types';
import ChatMessage from '../components/chat/ChatMessage';
import ChatInput from '../components/chat/ChatInput';
import SessionPickerModal from '../components/chat/SessionPickerModal';
import { useChat } from '../hooks/useChat';

const STATUS_COLORS: Record<string, string> = {
  draft: 'var(--text-muted)',
  ready: 'var(--status-todo)',
  in_progress: 'var(--status-progress)',
  done: 'var(--status-done)',
  blocked: 'var(--status-blocked)',
};

const STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  ready: 'Ready',
  in_progress: 'In Progress',
  done: 'Done',
  blocked: 'Blocked',
};

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m < 60) return s > 0 ? `${m}m ${s}s` : `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm > 0 ? `${h}h ${rm}m` : `${h}h`;
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
}

function formatCost(cost: number): string {
  return '$' + cost.toFixed(4);
}

export default function FeatureChatPage() {
  const { projectId, featureId } = useParams<{ projectId: string; featureId: string }>();
  const [feature, setFeature] = useState<FeatureResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [showSessionPicker, setShowSessionPicker] = useState(false);
  const [existingSessions, setExistingSessions] = useState<ChatSessionResponse[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | undefined>(undefined);
  const [nodes, setNodes] = useState<NodeResponse[]>([]);
  const [showNodePicker, setShowNodePicker] = useState(false);
  const [selectedNodeId, setSelectedNodeId] = useState<string | undefined>(undefined);

  const { messages, stats, isThinking, sendMessage, endSession, refreshActivity } = useChat({
    projectId,
    featureId,
    sessionId: activeSessionId,
    nodeId: selectedNodeId,
    onSessionStarted: (sid) => setActiveSessionId(sid),
    onSessionEnded: () => setActiveSessionId(undefined),
    onError: (err) => console.error('[chat]', err),
  });

  // Fetch feature info
  useEffect(() => {
    if (!projectId || !featureId) return;
    setLoading(true);
    getFeature(projectId, featureId)
      .then(setFeature)
      .catch(() => setFeature(null))
      .finally(() => setLoading(false));
  }, [projectId, featureId]);

  // Check for existing sessions and fetch nodes on mount
  useEffect(() => {
    if (!projectId || !featureId) return;
    listChatSessions(projectId, featureId)
      .then((sessions) => {
        if (sessions && sessions.length > 0) {
          setExistingSessions(sessions);
          setShowSessionPicker(true);
        } else {
          // No existing sessions — go straight to node picker
          setShowNodePicker(true);
        }
      })
      .catch(() => {
        // No sessions or endpoint not available yet — show node picker
        setShowNodePicker(true);
      });
    listNodes()
      .then((resp) => setNodes(resp.nodes ?? []))
      .catch(() => setNodes([]));
  }, [projectId, featureId]);

  const handleStartSession = useCallback(
    (resumeSessionId: string | null) => {
      if (!projectId || !featureId) return;
      setShowSessionPicker(false);
      if (resumeSessionId) {
        // Restore existing session — no node picker needed
        startChatSession(projectId, featureId, resumeSessionId)
          .then((session) => {
            setActiveSessionId(session.id);
          })
          .catch(() => {
            // Session start failed — user can still type locally
          });
      } else {
        // New session — show node picker first
        setShowNodePicker(true);
      }
    },
    [projectId, featureId],
  );

  const handleNodeSelect = useCallback(
    (nodeId: string) => {
      if (!projectId || !featureId) return;
      setSelectedNodeId(nodeId);
      setShowNodePicker(false);
      startChatSession(projectId, featureId, undefined, nodeId)
        .then((session) => {
          setActiveSessionId(session.id);
        })
        .catch(() => {
          // Session start failed — user can still type locally
        });
    },
    [projectId, featureId],
  );

  // Auto-scroll on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Activity ping every 60s when session is active
  useEffect(() => {
    if (!activeSessionId) return;
    const interval = setInterval(() => {
      refreshActivity();
    }, 60_000);
    return () => clearInterval(interval);
  }, [activeSessionId, refreshActivity]);

  const handleEndSession = useCallback(() => {
    if (!activeSessionId) return;
    endSession();
    setActiveSessionId(undefined);
  }, [activeSessionId, endSession]);

  if (loading) {
    return (
      <div
        data-qa="feature-chat-page"
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          fontFamily: 'Inter, sans-serif',
        }}
      >
        <Loader2 size={24} className="animate-spin" style={{ color: 'var(--text-muted)' }} />
      </div>
    );
  }

  return (
    <div
      data-qa="feature-chat-page"
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        fontFamily: 'Inter, sans-serif',
        backgroundColor: 'var(--bg-primary)',
      }}
    >
      {/* Header */}
      <div
        data-qa="chat-header"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          padding: '12px 16px',
          borderBottom: '1px solid var(--border-subtle)',
          backgroundColor: 'var(--bg-primary)',
          flexShrink: 0,
        }}
      >
        <Link
          to={projectId && featureId ? `/projects/${projectId}/features/${featureId}` : '#'}
          data-qa="chat-back-link"
          style={{
            color: 'var(--text-muted)',
            display: 'flex',
            alignItems: 'center',
            textDecoration: 'none',
            transition: 'color 0.15s',
          }}
        >
          <ArrowLeft size={18} />
        </Link>

        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <h1
              data-qa="chat-feature-name"
              style={{
                fontSize: '16px',
                fontWeight: 600,
                color: 'var(--text-primary)',
                margin: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {feature?.name ?? 'Feature Chat'}
            </h1>
            {feature && (
              <span
                data-qa="chat-feature-status"
                style={{
                  fontSize: '11px',
                  fontWeight: 600,
                  padding: '2px 8px',
                  borderRadius: '9999px',
                  backgroundColor: STATUS_COLORS[feature.status] ?? 'var(--text-muted)',
                  color: 'var(--primary-text)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                  flexShrink: 0,
                }}
              >
                {STATUS_LABELS[feature.status] ?? feature.status}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Stats bar */}
      <div
        data-qa="chat-stats-bar"
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
        <StatItem icon={<Cpu size={13} />} label="Model" value={stats.model} qa="stat-model" />
        <StatItem
          icon={<MessageSquare size={13} />}
          label="Messages"
          value={String(stats.messageCount)}
          qa="stat-messages"
        />
        <StatItem
          icon={<Database size={13} />}
          label="In"
          value={formatTokens(stats.inputTokens)}
          qa="stat-input-tokens"
        />
        <StatItem
          icon={<Database size={13} />}
          label="Out"
          value={formatTokens(stats.outputTokens)}
          qa="stat-output-tokens"
        />
        <StatItem
          icon={<Database size={13} />}
          label="Cache"
          value={formatTokens(stats.cacheReadTokens)}
          qa="stat-cache-tokens"
        />
        <StatItem
          icon={<DollarSign size={13} />}
          label="Cost"
          value={formatCost(stats.totalCost)}
          qa="stat-cost"
        />
        <StatItem
          icon={<Clock size={13} />}
          label="Duration"
          value={formatDuration(stats.durationSeconds)}
          qa="stat-duration"
        />

        {activeSessionId && (
          <button
            data-qa="end-session-btn"
            onClick={handleEndSession}
            style={{
              marginLeft: 'auto',
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              padding: '4px 10px',
              borderRadius: '6px',
              border: '1px solid var(--status-blocked)',
              backgroundColor: 'transparent',
              color: 'var(--status-blocked)',
              fontSize: '12px',
              fontWeight: 600,
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              transition: 'background-color 0.15s',
            }}
            onMouseEnter={e => { e.currentTarget.style.backgroundColor = 'var(--status-blocked)'; e.currentTarget.style.color = 'var(--primary-text)'; }}
            onMouseLeave={e => { e.currentTarget.style.backgroundColor = 'transparent'; e.currentTarget.style.color = 'var(--status-blocked)'; }}
          >
            <StopCircle size={13} />
            End Session
          </button>
        )}
      </div>

      {/* Message list or node picker */}
      {showNodePicker ? (
        <NodePickerStep
          nodes={nodes}
          onSelect={handleNodeSelect}
          onCancel={existingSessions.length > 0 ? () => { setShowNodePicker(false); setShowSessionPicker(true); } : undefined}
        />
      ) : (
        <div
          data-qa="chat-message-list"
          style={{
            flex: 1,
            overflowY: 'auto',
            padding: '16px',
            display: 'flex',
            flexDirection: 'column',
            gap: '16px',
          }}
        >
          {messages.length === 0 && (
            <div
              data-qa="chat-empty-state"
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                flex: 1,
                color: 'var(--text-muted)',
                gap: '8px',
                textAlign: 'center',
              }}
            >
              <MessageSquare size={40} style={{ opacity: 0.3 }} />
              <p style={{ fontSize: '14px', margin: 0 }}>No messages yet</p>
              <p style={{ fontSize: '12px', margin: 0 }}>
                Start a conversation to begin coding on this feature.
              </p>
            </div>
          )}
          {messages.map((msg) => (
            <ChatMessage key={msg.id} message={msg} />
          ))}
          {isThinking && <ThinkingIndicator />}
          <div ref={messagesEndRef} />
        </div>
      )}

      {/* Input bar */}
      <ChatInput onSend={sendMessage} disabled={false} />

      {/* Session picker modal */}
      {showSessionPicker && existingSessions.length > 0 && (
        <SessionPickerModal
          sessions={existingSessions}
          onSelect={handleStartSession}
          onClose={() => setShowSessionPicker(false)}
        />
      )}
    </div>
  );
}

/** Node selection step shown before starting a new chat session */
function NodePickerStep({
  nodes,
  onSelect,
  onCancel,
}: {
  nodes: NodeResponse[];
  onSelect: (nodeId: string) => void;
  onCancel?: () => void;
}) {
  const activeNodes = nodes.filter(n => n.status === 'active');

  return (
    <div data-qa="node-picker" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', flex: 1, padding: '32px' }}>
      <h3 style={{ margin: '0 0 8px 0', fontSize: '16px', fontWeight: 600, color: 'var(--text-primary)' }}>
        Select a Node
      </h3>
      <p style={{ margin: '0 0 24px 0', fontSize: '13px', color: 'var(--text-muted)' }}>
        Choose which daemon will run this chat session.
      </p>

      {activeNodes.length === 0 ? (
        <p style={{ fontSize: '13px', color: 'var(--text-muted)' }}>No active nodes available.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', width: '100%', maxWidth: '400px' }}>
          {activeNodes.map(node => (
            <button
              key={node.id}
              data-qa="node-option"
              onClick={() => onSelect(node.id)}
              style={{
                padding: '12px 16px',
                borderRadius: '8px',
                border: '1px solid var(--border-subtle)',
                backgroundColor: 'var(--bg-primary)',
                cursor: 'pointer',
                textAlign: 'left',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                transition: 'border-color 0.15s, background-color 0.15s',
              }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--primary-hover)'; e.currentTarget.style.backgroundColor = 'var(--bg-secondary)'; }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--border-subtle)'; e.currentTarget.style.backgroundColor = 'var(--bg-primary)'; }}
            >
              <div>
                <div style={{ fontSize: '14px', fontWeight: 500, color: 'var(--text-primary)' }}>{node.name}</div>
                <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
                  {node.last_seen_at ? `Last seen ${new Date(node.last_seen_at).toLocaleString()}` : 'Never connected'}
                </div>
              </div>
              <span style={{
                fontSize: '10px',
                fontWeight: 600,
                padding: '2px 8px',
                borderRadius: '9999px',
                backgroundColor: 'var(--status-done)',
                color: 'var(--primary-text)',
                textTransform: 'uppercase',
              }}>
                {node.mode}
              </span>
            </button>
          ))}
        </div>
      )}

      {onCancel && (
        <button
          onClick={onCancel}
          style={{
            marginTop: '16px',
            padding: '8px 16px',
            borderRadius: '6px',
            border: '1px solid var(--border-subtle)',
            backgroundColor: 'var(--bg-secondary)',
            color: 'var(--text-primary)',
            fontSize: '13px',
            cursor: 'pointer',
          }}
        >
          Cancel
        </button>
      )}
    </div>
  );
}

/** Animated thinking indicator shown while waiting for Claude's response */
function ThinkingIndicator() {
  return (
    <div
      data-qa="chat-thinking"
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: '8px',
        fontFamily: 'Inter, sans-serif',
      }}
    >
      <div
        style={{
          width: '32px',
          height: '32px',
          borderRadius: '50%',
          backgroundColor: 'var(--severity-success)',
          color: 'var(--primary-text)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '12px',
          fontWeight: 700,
          flexShrink: 0,
        }}
      >
        A1
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <span style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 500 }}>
          Assistant
        </span>
        <div
          style={{
            padding: '12px 16px',
            borderRadius: '4px 16px 16px 16px',
            backgroundColor: 'var(--bg-elevated)',
            display: 'flex',
            alignItems: 'center',
            gap: '4px',
          }}
        >
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: 'var(--text-muted)',
                display: 'inline-block',
                animation: `thinking-bounce 1.4s ${i * 0.16}s infinite ease-in-out both`,
              }}
            />
          ))}
          <style>{`
            @keyframes thinking-bounce {
              0%, 80%, 100% { transform: scale(0.4); opacity: 0.4; }
              40% { transform: scale(1); opacity: 1; }
            }
          `}</style>
        </div>
      </div>
    </div>
  );
}

/** Small stat pill used in the stats bar */
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
