import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../components/AuthContext';
import { useTheme } from '../components/ThemeContext';

const MONO = "'JetBrains Mono', 'Geist Mono', ui-monospace, monospace";
const SANS = "'Geist', 'Inter', sans-serif";

export default function LoginPage() {
  const { login } = useAuth();
  const { theme } = useTheme();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const isDark = theme === 'dark';

  const bg      = isDark ? '#13111C' : '#F8FAFC';
  const border  = isDark ? '#2E2B3D' : '#E2E8F0';
  const accent  = '#7C5CF6';
  const accentHover = '#6D4FE0';
  const textPrimary   = isDark ? '#F0EEF8' : '#0F172A';
  const textSecondary = isDark ? '#9891B0' : '#64748B';
  const textMuted     = isDark ? '#5C5578' : '#94A3B8';

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
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
          .login-grid { grid-template-columns: 1fr 480px !important; }
          .login-left  { display: flex !important; }
          .login-right-pad { padding: 60px 72px !important; }
        }
        .btn-social:hover { background: ${isDark ? '#252235' : '#F1F5F9'} !important; }
        input::placeholder { color: ${textMuted}; font-family: ${MONO}; }
      `}</style>

      <div className="login-grid" style={{
        minHeight: '100vh',
        display: 'grid',
        gridTemplateColumns: '1fr',
      }}>

        {/* ── LEFT ──────────────────────────────────────────────── */}
        <div className="login-left" style={{
          display: 'none',
          flexDirection: 'column',
          padding: '48px 64px',
          background: bg,
          position: 'relative',
          overflow: 'hidden',
        }}>
          {/* Subtle noise/glow */}
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
              flexShrink: 0,
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
              width: '7px', height: '7px', borderRadius: '50%', flexShrink: 0,
              background: accent,
              boxShadow: `0 0 6px ${accent}`,
            }} />
            <span style={{
              fontSize: '11px', fontWeight: 500, letterSpacing: '0.10em',
              fontFamily: MONO, color: isDark ? '#A89FCC' : '#6366F1',
            }}>AI ORCHESTRATION</span>
          </div>

          {/* Headline */}
          <h1 style={{
            fontSize: 'clamp(36px, 4vw, 52px)',
            lineHeight: 1.1, margin: '0 0 24px',
            fontFamily: MONO, fontWeight: 700,
            letterSpacing: '-0.02em',
            color: textPrimary,
          }}>
            Describe the feature.<br />
            <span style={{ color: accent }}>Agents build it.</span>
          </h1>

          {/* Sub */}
          <p style={{
            fontSize: '14px', lineHeight: 1.8, margin: '0 0 48px',
            fontFamily: MONO, color: textSecondary, maxWidth: '480px',
          }}>
            Agach connects your team with AI coding agents that understand context, remember
            decisions, and ship features — together.
          </p>

          {/* Features */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px', marginBottom: 'auto' }}>
            {[
              {
                title: 'Plan through conversation',
                body: 'Chat with your team and agents to shape features before a single line is written.',
              },
              {
                title: 'Agents that remember',
                body: 'Persistent context across sessions — agents recall decisions, PRs, and patterns.',
              },
              {
                title: 'Full team visibility',
                body: 'Every agent action, decision, and PR linked back to your team\'s intent.',
              },
              {
                title: 'Self-hosted, your data stays yours',
                body: 'Deploy on your own infra. No telemetry, no lock-in, full control.',
              },
            ].map(({ title, body }) => (
              <div key={title} style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                <span style={{
                  width: '8px', height: '8px', borderRadius: '50%',
                  background: accent, flexShrink: 0, marginTop: '5px',
                  boxShadow: `0 0 8px rgba(124,92,246,0.5)`,
                }} />
                <div>
                  <div style={{ fontSize: '13px', fontWeight: 700, fontFamily: MONO, color: textPrimary, marginBottom: '3px' }}>
                    {title}
                  </div>
                  <div style={{ fontSize: '12px', fontFamily: MONO, color: textSecondary, lineHeight: 1.7 }}>
                    {body}
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Footer */}
          <div style={{
            display: 'flex', alignItems: 'center', gap: '8px',
            paddingTop: '48px', fontFamily: MONO,
          }}>
            {['Open source', 'Self-hosted', 'MIT License'].map((tag, i, arr) => (
              <span key={tag} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '11px', color: textMuted }}>{tag}</span>
                {i < arr.length - 1 && <span style={{ color: border }}>·</span>}
              </span>
            ))}
          </div>
        </div>

        {/* ── RIGHT ─────────────────────────────────────────────── */}
        <div className="login-right-pad" style={{
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          padding: '40px 24px',
          background: isDark ? '#0F0D18' : '#FFFFFF',
          borderLeft: `1px solid ${border}`,
        }}>
          <div style={{ width: '100%', maxWidth: '400px' }}>

            {/* Mobile logo */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: '10px',
              marginBottom: '40px',
            }}>
              <div style={{
                width: '30px', height: '30px', borderRadius: '7px',
                background: accent,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <img src={isDark ? '/logo-dark.svg' : '/logo-light.svg'} alt="" style={{ width: '17px', filter: 'brightness(10)' }} />
              </div>
              <span style={{ fontSize: '15px', fontWeight: 500, fontFamily: MONO, color: textPrimary }}>
                agach
              </span>
            </div>

            {/* Heading */}
            <h2 style={{
              fontSize: '28px', fontWeight: 700, margin: '0 0 6px',
              fontFamily: SANS, letterSpacing: '-0.02em', color: textPrimary,
            }}>
              Welcome back
            </h2>
            <p style={{ fontSize: '14px', color: textSecondary, margin: '0 0 32px' }}>
              Sign in to your workspace
            </p>

            <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* Email */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 500, color: textSecondary, marginBottom: '7px' }}>
                  Email
                </label>
                <input
                  type="text"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  required autoFocus autoComplete="username"
                  data-qa="login-email-input"
                  placeholder="admin@agach.local"
                  style={{
                    width: '100%', padding: '11px 14px', borderRadius: '8px',
                    fontFamily: MONO, fontSize: '13px',
                    border: `1px solid ${border}`,
                    background: isDark ? '#1A1726' : '#F8FAFC',
                    color: textPrimary, outline: 'none', boxSizing: 'border-box',
                    transition: 'border-color 0.15s, box-shadow 0.15s',
                  }}
                  onFocus={e => { e.target.style.borderColor = accent; e.target.style.boxShadow = `0 0 0 3px rgba(124,92,246,0.15)`; }}
                  onBlur={e => { e.target.style.borderColor = border; e.target.style.boxShadow = 'none'; }}
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
                    required autoComplete="current-password"
                    data-qa="login-password-input"
                    style={{
                      width: '100%', padding: '11px 44px 11px 14px', borderRadius: '8px',
                      fontFamily: MONO, fontSize: '13px',
                      border: `1px solid ${border}`,
                      background: isDark ? '#1A1726' : '#F8FAFC',
                      color: textPrimary, outline: 'none', boxSizing: 'border-box',
                      transition: 'border-color 0.15s, box-shadow 0.15s',
                    }}
                    onFocus={e => { e.target.style.borderColor = accent; e.target.style.boxShadow = `0 0 0 3px rgba(124,92,246,0.15)`; }}
                    onBlur={e => { e.target.style.borderColor = border; e.target.style.boxShadow = 'none'; }}
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
                <div style={{ textAlign: 'right', marginTop: '6px' }}>
                  <span style={{ fontSize: '12px', color: accent, cursor: 'pointer', fontFamily: MONO }}>
                    Forgot password?
                  </span>
                </div>
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

              {/* Submit */}
              <button
                type="submit"
                disabled={loading}
                data-qa="login-submit-btn"
                style={{
                  width: '100%', padding: '13px',
                  borderRadius: '8px', fontSize: '15px', fontWeight: 600,
                  border: 'none', cursor: loading ? 'not-allowed' : 'pointer',
                  background: loading ? (isDark ? '#2E2B3D' : '#E2E8F0') : accent,
                  color: loading ? textMuted : '#FFFFFF',
                  boxShadow: loading ? 'none' : `0 4px 18px rgba(124,92,246,0.40)`,
                  transition: 'background 0.15s, box-shadow 0.15s, transform 0.1s',
                  fontFamily: SANS,
                  letterSpacing: '0.01em',
                }}
                onMouseEnter={e => {
                  if (!loading) {
                    (e.currentTarget).style.background = accentHover;
                    (e.currentTarget).style.boxShadow = `0 6px 22px rgba(124,92,246,0.50)`;
                    (e.currentTarget).style.transform = 'translateY(-1px)';
                  }
                }}
                onMouseLeave={e => {
                  (e.currentTarget).style.background = loading ? (isDark ? '#2E2B3D' : '#E2E8F0') : accent;
                  (e.currentTarget).style.boxShadow = loading ? 'none' : `0 4px 18px rgba(124,92,246,0.40)`;
                  (e.currentTarget).style.transform = '';
                }}
              >
                {loading ? 'Signing in…' : 'Sign in →'}
              </button>

              {/* Divider */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', margin: '4px 0' }}>
                <div style={{ flex: 1, height: '1px', background: border }} />
                <span style={{ fontSize: '12px', color: textMuted, fontFamily: MONO }}>or continue with</span>
                <div style={{ flex: 1, height: '1px', background: border }} />
              </div>

              {/* Social */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px' }}>
                {[
                  { label: 'GitHub', icon: <GithubIcon /> },
                  { label: 'Google', icon: <GoogleIcon /> },
                ].map(({ label, icon }) => (
                  <button
                    key={label}
                    type="button"
                    disabled
                    data-qa={`login-${label.toLowerCase()}-btn`}
                    className="btn-social"
                    style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      gap: '8px', padding: '10px 14px',
                      borderRadius: '8px', fontSize: '13px', fontWeight: 500,
                      border: `1px solid ${border}`,
                      background: isDark ? '#1A1726' : '#F8FAFC',
                      color: textSecondary, cursor: 'not-allowed', opacity: 0.6,
                      fontFamily: SANS, transition: 'background 0.15s',
                    }}
                  >
                    {icon}
                    {label}
                  </button>
                ))}
              </div>
            </form>

            {/* Footer note */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: '6px', justifyContent: 'center',
              marginTop: '24px',
            }}>
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke={textMuted} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              </svg>
              <span style={{ fontSize: '11px', color: textMuted, fontFamily: MONO }}>
                Self-hosted · Your data never leaves your infrastructure
              </span>
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

function GithubIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z" />
    </svg>
  );
}

function GoogleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24">
      <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" />
      <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
      <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
      <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
    </svg>
  );
}
