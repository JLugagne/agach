import { useState, useEffect, useCallback, useRef } from 'react';
import { useTheme } from '../components/ThemeContext';
import { generateOnboardingCode, listNodes } from '../lib/api';
import type { NodeResponse } from '../lib/types';

const MONO = "'JetBrains Mono', 'Geist Mono', ui-monospace, monospace";
const SANS = "'Geist', 'Inter', sans-serif";

interface OnboardingDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function OnboardingDialog({ open, onClose, onSuccess }: OnboardingDialogProps) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  const [mode, setMode] = useState<'default' | 'shared'>('default');
  const [nodeName, setNodeName] = useState('');
  const [code, setCode] = useState<string | null>(null);
  const [expiresAt, setExpiresAt] = useState<Date | null>(null);
  const [timeLeft, setTimeLeft] = useState<number>(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [connectedNode, setConnectedNode] = useState<NodeResponse | null>(null);

  const knownNodeIds = useRef<Set<string>>(new Set());

  const cardBg = isDark ? '#131520' : '#FFFFFF';
  const border = isDark ? '#1E2030' : '#E2E8F0';
  const accent = '#7C3AED';
  const textPrimary = isDark ? '#FFFFFF' : '#0F172A';
  const textSecondary = isDark ? '#6B7084' : '#64748B';
  const textMuted = isDark ? '#5C5578' : '#94A3B8';
  const greenColor = '#22C55E';

  // Reset state when dialog opens; refresh nodes when it closes
  useEffect(() => {
    if (open) {
      setMode('default');
      setNodeName('');
      setCode(null);
      setExpiresAt(null);
      setTimeLeft(0);
      setLoading(false);
      setError(null);
      setCopied(false);
      setConnectedNode(null);
    } else {
      // Always refresh the node list when dialog closes
      onSuccess();
    }
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  // Countdown timer
  useEffect(() => {
    if (!expiresAt) return;
    const interval = setInterval(() => {
      const diff = Math.max(0, Math.floor((expiresAt.getTime() - Date.now()) / 1000));
      setTimeLeft(diff);
      if (diff === 0) {
        setCode(null);
        setExpiresAt(null);
      }
    }, 1000);
    return () => clearInterval(interval);
  }, [expiresAt]);

  // Poll for new nodes while code is displayed
  useEffect(() => {
    if (!code || connectedNode) return;
    const poll = setInterval(async () => {
      try {
        const data = await listNodes();
        const nodes = data.nodes ?? [];
        const newNode = nodes.find(n => n.status === 'active' && !knownNodeIds.current.has(n.id));
        if (newNode) {
          setConnectedNode(newNode);
          setCode(null);
          setExpiresAt(null);
        }
      } catch {
        // ignore polling errors
      }
    }, 2000);
    return () => clearInterval(poll);
  }, [code, connectedNode]);

  const snapshotNodes = useCallback(async () => {
    try {
      const data = await listNodes();
      knownNodeIds.current = new Set((data.nodes ?? []).map(n => n.id));
    } catch { /* ignore */ }
  }, []);

  const handleGenerate = useCallback(async () => {
    setLoading(true);
    setError(null);
    setCopied(false);
    await snapshotNodes();
    try {
      const result = await generateOnboardingCode({ mode, node_name: nodeName || undefined });
      setCode(result.code);
      setExpiresAt(new Date(result.expires_at));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate code');
    } finally {
      setLoading(false);
    }
  }, [mode, nodeName, snapshotNodes]);

  const handleCopy = useCallback(() => {
    if (code) {
      navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [code]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleDone = () => {
    onClose();
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={!code && !connectedNode ? onClose : undefined} />
      <div
        data-qa="onboarding-dialog"
        style={{
          position: 'relative',
          background: cardBg,
          border: `1px solid ${border}`,
          borderRadius: '12px',
          padding: '32px',
          width: '500px',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        {/* Close button */}
        {!code && !connectedNode && (
          <button
            onClick={onClose}
            data-qa="onboarding-close-btn"
            style={{
              position: 'absolute',
              top: '16px',
              right: '16px',
              background: 'none',
              border: 'none',
              color: textMuted,
              cursor: 'pointer',
              fontSize: '18px',
              lineHeight: 1,
            }}
          >
            &times;
          </button>
        )}

        {connectedNode ? (
          /* ── Success state ── */
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '24px' }}>
            <div style={{
              width: '64px',
              height: '64px',
              borderRadius: '50%',
              background: `${greenColor}20`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}>
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke={greenColor} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                <polyline points="22 4 12 14.01 9 11.01" />
              </svg>
            </div>

            <div style={{ fontSize: '20px', fontWeight: 600, fontFamily: SANS, color: textPrimary }}>
              Daemon Connected Successfully!
            </div>

            <div style={{ fontSize: '14px', fontFamily: SANS, color: textSecondary, textAlign: 'center', maxWidth: '400px' }}>
              Your daemon instance has been registered and is now connected to the server.
            </div>

            <div style={{ width: '100%', height: '1px', background: border }} />

            {/* Node info rows */}
            <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '13px', fontFamily: SANS, color: textSecondary }}>Node Name</span>
                <span data-qa="success-node-name" style={{ fontSize: '13px', fontWeight: 500, fontFamily: SANS, color: textPrimary }}>
                  {connectedNode.name || 'Unnamed node'}
                </span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '13px', fontFamily: SANS, color: textSecondary }}>Mode</span>
                <span style={{
                  fontSize: '12px', fontWeight: 500, fontFamily: SANS, color: textMuted,
                  background: border, borderRadius: '6px', padding: '4px 8px',
                }}>
                  {connectedNode.mode === 'shared' ? 'Shared' : 'Personal'}
                </span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: '13px', fontFamily: SANS, color: textSecondary }}>Status</span>
                <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                  <span style={{ width: '8px', height: '8px', borderRadius: '50%', background: greenColor, display: 'inline-block' }} />
                  <span style={{ fontSize: '13px', fontWeight: 500, fontFamily: SANS, color: greenColor }}>Active</span>
                </span>
              </div>
            </div>

            <div style={{ width: '100%', height: '1px', background: border }} />

            <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: '12px' }}>
              <button
                onClick={handleDone}
                data-qa="view-nodes-btn"
                style={{
                  width: '100%', padding: '14px 24px', borderRadius: '8px',
                  fontSize: '14px', fontWeight: 500, border: 'none',
                  background: accent, color: '#FFFFFF', fontFamily: SANS, cursor: 'pointer',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px',
                }}
              >
                Done
              </button>
              <button
                onClick={() => {
                  setConnectedNode(null);
                  setCode(null);
                  setExpiresAt(null);
                  setNodeName('');
                  setMode('default');
                }}
                data-qa="add-another-btn"
                style={{
                  width: '100%', padding: '14px 24px', borderRadius: '8px',
                  fontSize: '14px', fontWeight: 500, border: `1px solid ${border}`,
                  background: 'transparent', color: textMuted, fontFamily: SANS, cursor: 'pointer',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px',
                }}
              >
                Add Another
              </button>
            </div>
          </div>
        ) : code ? (
          /* ── Code displayed state ── */
          <div style={{ textAlign: 'center' }}>
            <p style={{ fontSize: '13px', color: textSecondary, fontFamily: MONO, marginBottom: '16px' }}>
              Your onboarding code:
            </p>

            <div
              onClick={handleCopy}
              data-qa="onboarding-code-display"
              style={{
                fontSize: '48px', fontWeight: 700, fontFamily: MONO, color: accent,
                letterSpacing: '0.2em', padding: '24px',
                background: isDark ? '#0D0F17' : '#F8FAFC', borderRadius: '12px',
                cursor: 'pointer', marginBottom: '16px', transition: 'transform 0.1s',
              }}
            >
              {code}
            </div>

            <button
              onClick={handleCopy}
              data-qa="copy-code-btn"
              style={{
                padding: '10px 20px', borderRadius: '8px', fontSize: '13px', fontWeight: 500,
                border: `1px solid ${border}`, background: 'transparent',
                color: copied ? '#10B981' : textSecondary, fontFamily: MONO, cursor: 'pointer',
                marginBottom: '24px',
              }}
            >
              {copied ? 'Copied!' : 'Copy to clipboard'}
            </button>

            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px',
              color: timeLeft < 60 ? '#EF4444' : textSecondary, fontFamily: MONO, fontSize: '14px',
            }}>
              <span>Expires in</span>
              <span style={{ fontWeight: 600 }} data-qa="countdown-timer">{formatTime(timeLeft)}</span>
            </div>

            <div style={{
              marginTop: '32px', padding: '16px',
              background: isDark ? '#0D0F17' : '#F8FAFC', borderRadius: '8px', textAlign: 'left',
            }}>
              <p style={{ fontSize: '12px', color: textSecondary, fontFamily: MONO, marginBottom: '12px' }}>
                On the daemon machine, run:
              </p>
              <code data-qa="daemon-command" style={{
                display: 'block', padding: '12px',
                background: isDark ? '#0D0F17' : '#E2E8F0', borderRadius: '6px',
                fontSize: '12px', fontFamily: MONO, color: textPrimary, overflowX: 'auto',
              }}>
                AGACH_ONBOARDING_CODE={code} agach-daemon
              </code>
            </div>

            <button
              onClick={() => { setCode(null); setExpiresAt(null); }}
              data-qa="generate-new-btn"
              style={{
                marginTop: '24px', padding: '10px 20px', borderRadius: '8px', fontSize: '13px',
                border: `1px solid ${border}`, background: 'transparent',
                color: textSecondary, fontFamily: MONO, cursor: 'pointer',
              }}
            >
              Generate new code
            </button>
          </div>
        ) : (
          /* ── Form state ── */
          <div>
            <h2 style={{ fontSize: '20px', fontWeight: 600, fontFamily: SANS, color: textPrimary, marginBottom: '24px' }}>
              Add Node
            </h2>

            {/* Mode selection */}
            <div style={{ marginBottom: '20px' }}>
              <label style={{ display: 'block', fontSize: '14px', fontWeight: 500, color: textPrimary, marginBottom: '8px' }}>
                Connection Mode
              </label>
              <div style={{ display: 'flex', gap: '0' }}>
                {(['default', 'shared'] as const).map((m) => (
                  <button
                    key={m}
                    type="button"
                    onClick={() => setMode(m)}
                    data-qa={`mode-${m}-btn`}
                    style={{
                      flex: 1, padding: '10px 20px', borderRadius: '8px',
                      border: 'none',
                      background: mode === m ? accent : (isDark ? '#1E2030' : '#F1F5F9'),
                      color: mode === m ? '#FFFFFF' : textMuted,
                      fontFamily: SANS, fontSize: '13px', fontWeight: 500,
                      cursor: 'pointer', transition: 'all 0.15s',
                    }}
                  >
                    {m === 'default' ? 'Default (personal)' : 'Shared (team)'}
                  </button>
                ))}
              </div>
            </div>

            {/* Node name */}
            <div style={{ marginBottom: '24px' }}>
              <label style={{ display: 'block', fontSize: '14px', fontWeight: 500, color: textPrimary, marginBottom: '8px' }}>
                Node Name (optional)
              </label>
              <input
                type="text"
                value={nodeName}
                onChange={(e) => setNodeName(e.target.value)}
                placeholder="e.g., dev-laptop, ci-runner"
                data-qa="node-name-input"
                style={{
                  width: '100%', padding: '12px 16px', borderRadius: '8px',
                  fontFamily: SANS, fontSize: '14px',
                  border: `1px solid ${border}`,
                  background: isDark ? '#0D0F17' : '#F8FAFC',
                  color: textPrimary, outline: 'none', boxSizing: 'border-box',
                }}
              />
            </div>

            {error && (
              <div data-qa="onboarding-error" style={{
                padding: '12px', borderRadius: '8px',
                background: isDark ? 'rgba(248,113,113,0.1)' : '#FEF2F2',
                border: `1px solid ${isDark ? 'rgba(248,113,113,0.2)' : '#FECACA'}`,
                color: isDark ? '#FCA5A5' : '#DC2626',
                fontSize: '13px', fontFamily: MONO, marginBottom: '16px',
              }}>
                {error}
              </div>
            )}

            <button
              onClick={handleGenerate}
              disabled={loading}
              data-qa="generate-code-btn"
              style={{
                width: '100%', padding: '14px 24px', borderRadius: '8px',
                fontSize: '14px', fontWeight: 500, border: 'none',
                cursor: loading ? 'not-allowed' : 'pointer',
                background: loading ? (isDark ? '#2E2B3D' : '#E2E8F0') : accent,
                color: loading ? textMuted : '#FFFFFF',
                fontFamily: SANS, transition: 'all 0.15s',
              }}
            >
              {loading ? 'Generating...' : 'Generate Onboarding Code'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
