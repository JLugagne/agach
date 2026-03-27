import { useState, type FormEvent } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../components/AuthContext';
import { useTheme } from '../components/ThemeContext';
import { completeInvite } from '../lib/api';

const MONO = "'JetBrains Mono', 'Geist Mono', ui-monospace, monospace";
const SANS = "'Geist', 'Inter', sans-serif";

export default function InvitePage() {
  const { theme } = useTheme();
  const { setSession } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';

  const [displayName, setDisplayName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const isDark = theme === 'dark';
  const bg = isDark ? '#13111C' : '#F8FAFC';
  const border = isDark ? '#2E2B3D' : '#E2E8F0';
  const accent = '#7C5CF6';
  const accentHover = '#6D4FE0';
  const textPrimary = isDark ? '#F0EEF8' : '#0F172A';
  const textSecondary = isDark ? '#9891B0' : '#64748B';
  const textMuted = isDark ? '#5C5578' : '#94A3B8';

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    if (!token) {
      setError('Invalid or missing invite link.');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }

    setLoading(true);
    try {
      const data = await completeInvite({ token, display_name: displayName.trim(), password });
      setSession(data.user, data.access_token);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete registration');
    } finally {
      setLoading(false);
    }
  }

  const inputStyle = {
    width: '100%', padding: '11px 14px', borderRadius: '8px',
    fontFamily: MONO, fontSize: '13px',
    border: `1px solid ${border}`,
    background: isDark ? '#1A1726' : '#F8FAFC',
    color: textPrimary, outline: 'none', boxSizing: 'border-box' as const,
    transition: 'border-color 0.15s, box-shadow 0.15s',
  };
  const focusStyle = (e: React.FocusEvent<HTMLInputElement>) => {
    e.target.style.borderColor = accent;
    e.target.style.boxShadow = `0 0 0 3px rgba(124,92,246,0.15)`;
  };
  const blurStyle = (e: React.FocusEvent<HTMLInputElement>) => {
    e.target.style.borderColor = border;
    e.target.style.boxShadow = 'none';
  };

  if (!token) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: bg }}>
        <style>{`input::placeholder { color: ${textMuted}; font-family: ${MONO}; }`}</style>
        <div style={{
          width: '100%', maxWidth: '400px', padding: '40px 24px', textAlign: 'center',
        }}>
          <div style={{
            width: '56px', height: '56px', borderRadius: '16px',
            background: isDark ? 'rgba(248,113,113,0.08)' : '#FEF2F2',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            margin: '0 auto 20px',
          }}>
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke={isDark ? '#FCA5A5' : '#DC2626'} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <line x1="15" y1="9" x2="9" y2="15" />
              <line x1="9" y1="9" x2="15" y2="15" />
            </svg>
          </div>
          <h1 style={{ fontSize: '20px', fontWeight: 600, color: textPrimary, fontFamily: SANS, marginBottom: '8px' }}>
            Invalid invite link
          </h1>
          <p style={{ fontSize: '14px', color: textSecondary, fontFamily: SANS, marginBottom: '24px' }}>
            This invite link is invalid or has expired. Ask your administrator for a new one.
          </p>
          <a
            href="/login"
            style={{
              fontSize: '13px', color: accent, fontFamily: MONO,
              textDecoration: 'none',
            }}
          >
            Go to login
          </a>
        </div>
      </div>
    );
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'grid',
      gridTemplateColumns: '1fr',
      background: bg,
      fontFamily: SANS,
      color: textPrimary,
    }}>
      <style>{`
        @media (min-width: 1024px) {
          .invite-grid { grid-template-columns: 1fr 480px !important; }
          .invite-left { display: flex !important; }
          .invite-right-pad { padding: 60px 72px !important; }
        }
        input::placeholder { color: ${textMuted}; font-family: ${MONO}; }
      `}</style>

      <div className="invite-grid" style={{
        minHeight: '100vh',
        display: 'grid',
        gridTemplateColumns: '1fr',
      }}>

        {/* ── LEFT (desktop only) ──────────────────────────────── */}
        <div className="invite-left" style={{
          display: 'none',
          flexDirection: 'column',
          padding: '48px 64px',
          background: bg,
          position: 'relative',
          overflow: 'hidden',
        }}>
          <div style={{
            position: 'absolute', top: '-200px', left: '-200px',
            width: '700px', height: '700px', borderRadius: '50%', pointerEvents: 'none',
            background: 'radial-gradient(circle, rgba(124,92,246,0.10) 0%, transparent 60%)',
          }} />

          {/* Logo */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '80px' }}>
            <div style={{
              width: '32px', height: '32px', borderRadius: '8px',
              background: accent,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              <img src={isDark ? '/logo-dark.svg' : '/logo-light.svg'} alt="" style={{ width: '18px', filter: 'brightness(10)' }} />
            </div>
            <span style={{ fontSize: '16px', fontWeight: 500, fontFamily: MONO, color: textPrimary }}>
              agach
            </span>
          </div>

          {/* Badge */}
          <div style={{
            display: 'inline-flex', alignItems: 'center', gap: '8px',
            border: `1px solid ${isDark ? '#3A3550' : '#C7D2FE'}`,
            borderRadius: '4px', padding: '5px 12px', marginBottom: '32px',
            alignSelf: 'flex-start',
          }}>
            <span style={{
              width: '7px', height: '7px', borderRadius: '50%',
              background: accent, boxShadow: `0 0 6px ${accent}`,
            }} />
            <span style={{
              fontSize: '11px', fontWeight: 500, letterSpacing: '0.10em',
              fontFamily: MONO, color: isDark ? '#A89FCC' : '#6366F1',
            }}>YOU'RE INVITED</span>
          </div>

          <h1 style={{
            fontSize: 'clamp(36px, 4vw, 52px)',
            lineHeight: 1.1, margin: '0 0 24px',
            fontFamily: MONO, fontWeight: 700,
            letterSpacing: '-0.02em', color: textPrimary,
          }}>
            Join the team.<br />
            <span style={{ color: accent }}>Start building.</span>
          </h1>

          <p style={{
            fontSize: '14px', lineHeight: 1.8, margin: '0 0 48px',
            fontFamily: MONO, color: textSecondary, maxWidth: '480px',
          }}>
            You've been invited to join an Agach workspace. Set up your account to collaborate
            with your team and AI agents.
          </p>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', marginBottom: 'auto' }}>
            {[
              { title: 'Secure & private', body: 'Your credentials are encrypted and stored on your team\'s own infrastructure.' },
              { title: 'Ready in seconds', body: 'Just pick a name and password — you\'ll be ready to collaborate immediately.' },
              { title: 'Full access', body: 'Browse projects, view kanban boards, and interact with AI agents right away.' },
            ].map(({ title, body }) => (
              <div key={title} style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                <span style={{
                  width: '8px', height: '8px', borderRadius: '50%',
                  background: accent, flexShrink: 0, marginTop: '5px',
                  boxShadow: `0 0 8px rgba(124,92,246,0.5)`,
                }} />
                <div>
                  <div style={{ fontSize: '13px', fontWeight: 700, fontFamily: MONO, color: textPrimary, marginBottom: '3px' }}>{title}</div>
                  <div style={{ fontSize: '12px', fontFamily: MONO, color: textSecondary, lineHeight: 1.7 }}>{body}</div>
                </div>
              </div>
            ))}
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', paddingTop: '48px', fontFamily: MONO }}>
            {['Open source', 'Self-hosted', 'MIT License'].map((tag, i, arr) => (
              <span key={tag} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '11px', color: textMuted }}>{tag}</span>
                {i < arr.length - 1 && <span style={{ color: border }}>·</span>}
              </span>
            ))}
          </div>
        </div>

        {/* ── RIGHT ─────────────────────────────────────────────── */}
        <div className="invite-right-pad" style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          padding: '40px 24px',
          background: isDark ? '#0F0D18' : '#FFFFFF',
          borderLeft: `1px solid ${border}`,
        }}>
          <div style={{ width: '100%', maxWidth: '400px' }}>

            {/* Mobile logo */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '40px' }}>
              <div style={{
                width: '30px', height: '30px', borderRadius: '7px',
                background: accent,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <img src={isDark ? '/logo-dark.svg' : '/logo-light.svg'} alt="" style={{ width: '17px', filter: 'brightness(10)' }} />
              </div>
              <span style={{ fontSize: '15px', fontWeight: 500, fontFamily: MONO, color: textPrimary }}>agach</span>
            </div>

            <h2 style={{
              fontSize: '28px', fontWeight: 700, margin: '0 0 6px',
              fontFamily: SANS, letterSpacing: '-0.02em', color: textPrimary,
            }}>
              Create your account
            </h2>
            <p style={{ fontSize: '14px', color: textSecondary, margin: '0 0 32px' }}>
              Set your name and password to get started
            </p>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* Display Name */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: textSecondary, marginBottom: '7px' }}>
                  Display Name
                </label>
                <input
                  type="text"
                  value={displayName}
                  onChange={e => setDisplayName(e.target.value)}
                  required
                  autoFocus
                  autoComplete="name"
                  data-qa="invite-display-name"
                  placeholder="How your team will see you"
                  style={inputStyle}
                  onFocus={focusStyle}
                  onBlur={blurStyle}
                />
              </div>

              {/* Password */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: textSecondary, marginBottom: '7px' }}>
                  Password
                </label>
                <div style={{ position: 'relative' }}>
                  <input
                    type={showPass ? 'text' : 'password'}
                    value={password}
                    onChange={e => setPassword(e.target.value)}
                    required
                    minLength={8}
                    autoComplete="new-password"
                    data-qa="invite-password"
                    placeholder="Min. 8 characters"
                    style={{ ...inputStyle, paddingRight: '44px' }}
                    onFocus={focusStyle}
                    onBlur={blurStyle}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPass(v => !v)}
                    data-qa="toggle-password-visibility-btn"
                    style={{
                      position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)',
                      background: 'none', border: 'none', cursor: 'pointer', padding: '2px',
                      color: textMuted, display: 'flex', alignItems: 'center',
                    }}
                  >
                    {showPass ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
              </div>

              {/* Confirm Password */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: textSecondary, marginBottom: '7px' }}>
                  Confirm Password
                </label>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={e => setConfirmPassword(e.target.value)}
                  required
                  minLength={8}
                  autoComplete="new-password"
                  data-qa="invite-confirm-password"
                  placeholder="Repeat your password"
                  style={inputStyle}
                  onFocus={focusStyle}
                  onBlur={blurStyle}
                />
                {confirmPassword && password !== confirmPassword && (
                  <p style={{ fontSize: '12px', color: isDark ? '#FCA5A5' : '#DC2626', fontFamily: MONO, marginTop: '6px' }}>
                    Passwords do not match
                  </p>
                )}
              </div>

              {error && (
                <div style={{
                  padding: '10px 14px', borderRadius: '8px', fontSize: '13px',
                  fontFamily: MONO,
                  color: isDark ? '#FCA5A5' : '#DC2626',
                  background: isDark ? 'rgba(248,113,113,0.08)' : '#FEF2F2',
                  border: `1px solid ${isDark ? 'rgba(248,113,113,0.2)' : '#FECACA'}`,
                }}>
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading || !displayName.trim() || !password || password !== confirmPassword}
                data-qa="invite-submit-btn"
                style={{
                  width: '100%', padding: '13px',
                  borderRadius: '8px', fontSize: '15px', fontWeight: 600,
                  border: 'none',
                  cursor: (loading || !displayName.trim() || !password || password !== confirmPassword) ? 'not-allowed' : 'pointer',
                  background: (loading || !displayName.trim() || !password || password !== confirmPassword) ? (isDark ? '#2E2B3D' : '#E2E8F0') : accent,
                  color: (loading || !displayName.trim() || !password || password !== confirmPassword) ? textMuted : '#FFFFFF',
                  boxShadow: (loading || !displayName.trim() || !password || password !== confirmPassword) ? 'none' : `0 4px 18px rgba(124,92,246,0.40)`,
                  transition: 'background 0.15s, box-shadow 0.15s, transform 0.1s',
                  fontFamily: SANS, letterSpacing: '0.01em',
                }}
                onMouseEnter={e => {
                  if (!loading && displayName.trim() && password && password === confirmPassword) {
                    e.currentTarget.style.background = accentHover;
                    e.currentTarget.style.boxShadow = `0 6px 22px rgba(124,92,246,0.50)`;
                    e.currentTarget.style.transform = 'translateY(-1px)';
                  }
                }}
                onMouseLeave={e => {
                  const disabled = loading || !displayName.trim() || !password || password !== confirmPassword;
                  e.currentTarget.style.background = disabled ? (isDark ? '#2E2B3D' : '#E2E8F0') : accent;
                  e.currentTarget.style.boxShadow = disabled ? 'none' : `0 4px 18px rgba(124,92,246,0.40)`;
                  e.currentTarget.style.transform = '';
                }}
              >
                {loading ? 'Creating account…' : 'Create account →'}
              </button>
            </form>

            <div style={{
              display: 'flex', alignItems: 'center', gap: '6px', justifyContent: 'center',
              marginTop: '24px',
            }}>
              <span style={{ fontSize: '12px', color: textMuted, fontFamily: MONO }}>
                Already have an account?
              </span>
              <a href="/login" style={{ fontSize: '12px', color: accent, fontFamily: MONO, textDecoration: 'none' }}>
                Sign in
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function Eye({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function EyeOff({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  );
}
