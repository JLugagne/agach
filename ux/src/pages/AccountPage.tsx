import { useState, useEffect, type FormEvent } from 'react';
import { getMe, updateProfile, changePassword } from '../lib/api';
import { useAuth } from '../components/AuthContext';

interface UserProfile {
  id: string;
  email: string;
  display_name: string;
  role: string;
  created_at: string;
}

export default function AccountPage() {
  const { user, updateUser } = useAuth();
  const [profile, setProfile] = useState<UserProfile | null>(null);

  // Profile form
  const [displayName, setDisplayName] = useState('');
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileMsg, setProfileMsg] = useState<{ ok: boolean; text: string } | null>(null);

  // Password form
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMsg, setPwMsg] = useState<{ ok: boolean; text: string } | null>(null);

  useEffect(() => {
    getMe().then(p => {
      setProfile(p);
      setDisplayName(p.display_name);
    }).catch(() => {});
  }, []);

  async function handleProfileSubmit(e: FormEvent) {
    e.preventDefault();
    setProfileSaving(true);
    setProfileMsg(null);
    try {
      const updated = await updateProfile({ display_name: displayName });
      setProfile(updated);
      updateUser({ display_name: updated.display_name });
      setProfileMsg({ ok: true, text: 'Profile updated.' });
    } catch (err) {
      setProfileMsg({ ok: false, text: err instanceof Error ? err.message : 'Failed to update profile.' });
    } finally {
      setProfileSaving(false);
    }
  }

  async function handlePasswordSubmit(e: FormEvent) {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setPwMsg({ ok: false, text: 'Passwords do not match.' });
      return;
    }
    setPwSaving(true);
    setPwMsg(null);
    try {
      await changePassword({ current_password: currentPassword, new_password: newPassword });
      setPwMsg({ ok: true, text: 'Password changed successfully.' });
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err) {
      setPwMsg({ ok: false, text: err instanceof Error ? err.message : 'Failed to change password.' });
    } finally {
      setPwSaving(false);
    }
  }

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-2xl mx-auto px-8 py-12 flex flex-col gap-8">
        <div>
          <h1 className="text-[24px] font-medium text-[var(--text-primary)]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            Account
          </h1>
          <p className="text-xs text-[var(--text-muted)] mt-1" style={{ fontFamily: 'Inter, sans-serif' }}>
            {user?.email} · {profile?.role}
          </p>
        </div>

        {/* Profile */}
        <section className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] overflow-hidden">
          <div className="px-6 py-4 border-b border-[var(--border-primary)]">
            <h2 className="text-[14px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Profile
            </h2>
          </div>
          <form onSubmit={handleProfileSubmit} className="px-6 py-5 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Display Name
              </label>
              <input
                type="text"
                value={displayName}
                onChange={e => setDisplayName(e.target.value)}
                required
                data-qa="account-display-name-input"
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Email
              </label>
              <input
                type="text"
                value={profile?.email ?? ''}
                disabled
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-tertiary)] text-[var(--text-muted)] text-[13px] cursor-not-allowed"
                style={{ fontFamily: 'Inter, sans-serif' }}
              />
            </div>
            {profileMsg && (
              <p className={`text-[12px] px-3 py-2 rounded-lg border ${profileMsg.ok ? 'text-green-500 bg-green-500/10 border-green-500/20' : 'text-red-500 bg-red-500/10 border-red-500/20'}`}
                style={{ fontFamily: 'Inter, sans-serif' }}>
                {profileMsg.text}
              </p>
            )}
            <div className="flex justify-end">
              <button
                type="submit"
                disabled={profileSaving}
                data-qa="account-save-profile-btn"
                className="px-4 py-2 rounded-lg text-[13px] font-semibold bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] disabled:opacity-50 transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {profileSaving ? 'Saving…' : 'Save'}
              </button>
            </div>
          </form>
        </section>

        {/* Password */}
        <section className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] overflow-hidden">
          <div className="px-6 py-4 border-b border-[var(--border-primary)]">
            <h2 className="text-[14px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Change Password
            </h2>
          </div>
          <form onSubmit={handlePasswordSubmit} className="px-6 py-5 flex flex-col gap-4">
            {[
              { label: 'Current Password', value: currentPassword, onChange: setCurrentPassword, qa: 'account-current-password' },
              { label: 'New Password', value: newPassword, onChange: setNewPassword, qa: 'account-new-password' },
              { label: 'Confirm New Password', value: confirmPassword, onChange: setConfirmPassword, qa: 'account-confirm-password' },
            ].map(({ label, value, onChange, qa }) => (
              <div key={qa} className="flex flex-col gap-1.5">
                <label className="text-[12px] font-medium text-[var(--text-secondary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                  {label}
                </label>
                <input
                  type="password"
                  value={value}
                  onChange={e => onChange(e.target.value)}
                  required
                  data-qa={qa}
                  className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
                  style={{ fontFamily: 'Inter, sans-serif' }}
                />
              </div>
            ))}
            {pwMsg && (
              <p className={`text-[12px] px-3 py-2 rounded-lg border ${pwMsg.ok ? 'text-green-500 bg-green-500/10 border-green-500/20' : 'text-red-500 bg-red-500/10 border-red-500/20'}`}
                style={{ fontFamily: 'Inter, sans-serif' }}>
                {pwMsg.text}
              </p>
            )}
            <div className="flex justify-end">
              <button
                type="submit"
                disabled={pwSaving}
                data-qa="account-change-password-btn"
                className="px-4 py-2 rounded-lg text-[13px] font-semibold bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] disabled:opacity-50 transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {pwSaving ? 'Saving…' : 'Change Password'}
              </button>
            </div>
          </form>
        </section>
      </div>
    </div>
  );
}
