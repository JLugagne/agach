import { useState, useEffect, useCallback } from 'react';
import { Loader2, Plus, Trash2, Users, User } from 'lucide-react';
import {
  listTeams, createTeam, deleteTeam,
  listUsers, setUserTeam, removeUserFromTeam, setUserRole,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { TeamResponse, UserResponse } from '../lib/types';

export default function TeamsPage() {
  const [teams, setTeams] = useState<TeamResponse[]>([]);
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [loading, setLoading] = useState(true);

  // Create team
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newSlug, setNewSlug] = useState('');
  const [creating, setCreating] = useState(false);

  // Delete team
  const [deleteTarget, setDeleteTarget] = useState<TeamResponse | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Add user to team
  const [addUserTeamId, setAddUserTeamId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const [t, u] = await Promise.all([listTeams(), listUsers()]);
      setTeams(t ?? []);
      setUsers(u ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const generateSlug = (n: string) =>
    n.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 50);

  const handleCreate = async () => {
    if (!newName.trim() || !newSlug.trim()) return;
    setCreating(true);
    try {
      await createTeam({ name: newName.trim(), slug: newSlug.trim() });
      setShowCreate(false);
      setNewName('');
      setNewSlug('');
      fetchData();
    } catch {
      // ignore
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteTeam(deleteTarget.id);
      setDeleteTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setDeleting(false);
    }
  };

  const handleAssignUser = async (userId: string, teamId: string) => {
    try {
      await setUserTeam(userId, teamId);
      setAddUserTeamId(null);
      fetchData();
    } catch {
      // ignore
    }
  };

  const handleRemoveUser = async (userId: string, teamId: string) => {
    try {
      await removeUserFromTeam(userId, teamId);
      fetchData();
    } catch {
      // ignore
    }
  };

  const handleToggleRole = async (user: UserResponse) => {
    try {
      await setUserRole(user.id, user.role === 'admin' ? 'member' : 'admin');
      fetchData();
    } catch {
      // ignore
    }
  };

  const unassignedUsers = users.filter((u) => u.team_ids.length === 0);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-[var(--bg-primary)]">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto bg-[var(--bg-primary)]">
      <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
        {/* Header */}
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Teams
          </h1>
          <button
            onClick={() => setShowCreate(true)}
            data-qa="create-team-btn"
            className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            <Plus size={14} />
            New Team
          </button>
        </div>
        <p className="text-sm text-[var(--text-muted)] mb-8" style={{ fontFamily: 'Inter, sans-serif' }}>
          {teams.length} team{teams.length !== 1 ? 's' : ''} · {users.length} user{users.length !== 1 ? 's' : ''}
        </p>

        {/* Teams */}
        {teams.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-4">
            <div className="w-16 h-16 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
              <Users size={28} className="text-[var(--text-muted)]" />
            </div>
            <p className="text-sm text-[var(--text-muted)]">No teams yet. Create one to get started.</p>
          </div>
        ) : (
          <div className="space-y-4 mb-10">
            {teams.map((team) => {
              const members = users.filter((u) => u.team_ids.includes(team.id));
              const nonMembers = users.filter((u) => !u.team_ids.includes(team.id));
              return (
                <div key={team.id} className="border border-[var(--border-primary)] rounded-lg overflow-hidden">
                  {/* Team header */}
                  <div className="flex items-center justify-between px-4 py-3 bg-[var(--bg-secondary)]">
                    <div className="flex items-center gap-2.5">
                      <Users size={14} className="text-[var(--primary)]" />
                      <span className="text-sm font-medium text-[var(--text-primary)]">{team.name}</span>
                      <span className="text-xs text-[var(--text-dim)] font-mono">{team.slug}</span>
                      <span className="text-[10px] text-[var(--text-dim)]">
                        {members.length} member{members.length !== 1 ? 's' : ''}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setAddUserTeamId(addUserTeamId === team.id ? null : team.id)}
                        data-qa="add-user-to-team-btn"
                        className="text-xs text-[var(--text-muted)] hover:text-[var(--primary)] transition-colors px-2 py-0.5"
                      >
                        + Add User
                      </button>
                      <button
                        onClick={() => setDeleteTarget(team)}
                        data-qa="delete-team-btn"
                        className="p-1 text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors"
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </div>

                  {/* Add user dropdown */}
                  {addUserTeamId === team.id && nonMembers.length > 0 && (
                    <div className="px-4 py-2 border-t border-[var(--border-primary)] bg-[var(--bg-tertiary)]">
                      <select
                        onChange={(e) => {
                          if (e.target.value) handleAssignUser(e.target.value, team.id);
                        }}
                        defaultValue=""
                        data-qa="assign-user-select"
                        className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md px-3 py-1.5 text-sm text-[var(--text-primary)] focus:outline-none focus:border-[var(--primary)]/50"
                      >
                        <option value="" disabled>Select a user...</option>
                        {nonMembers.map((u) => (
                          <option key={u.id} value={u.id}>{u.display_name || u.email}</option>
                        ))}
                      </select>
                    </div>
                  )}
                  {addUserTeamId === team.id && nonMembers.length === 0 && (
                    <div className="px-4 py-2 border-t border-[var(--border-primary)] bg-[var(--bg-tertiary)]">
                      <p className="text-xs text-[var(--text-dim)] italic">All users are already in this team.</p>
                    </div>
                  )}

                  {/* Members */}
                  {members.length > 0 && (
                    <div className="divide-y divide-[var(--border-primary)]">
                      {members.map((u) => (
                        <UserRow
                          key={u.id}
                          user={u}
                          onToggleRole={() => handleToggleRole(u)}
                          onRemove={() => handleRemoveUser(u.id, team.id)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* Unassigned users */}
        {unassignedUsers.length > 0 && (
          <section>
            <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
              Unassigned Users
            </h2>
            <div className="border border-[var(--border-primary)] rounded-lg overflow-hidden divide-y divide-[var(--border-primary)]">
              {unassignedUsers.map((u) => (
                <div key={u.id} className="flex items-center justify-between px-4 py-2.5">
                  <div className="flex items-center gap-2.5">
                    <User size={13} className="text-[var(--text-dim)]" />
                    <span className="text-sm text-[var(--text-primary)]">{u.display_name || u.email}</span>
                    <span className="text-xs text-[var(--text-dim)]">{u.email}</span>
                    <button
                      onClick={() => handleToggleRole(u)}
                      data-qa="toggle-user-role-btn"
                      className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono cursor-pointer transition-colors ${
                        u.role === 'admin'
                          ? 'bg-[var(--primary)]/15 text-[var(--primary)]'
                          : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)] hover:text-[var(--text-muted)]'
                      }`}
                    >
                      {u.role}
                    </button>
                  </div>
                  {teams.length > 0 && (
                    <select
                      onChange={(e) => { if (e.target.value) handleAssignUser(u.id, e.target.value); }}
                      defaultValue=""
                      data-qa="assign-to-team-select"
                      className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-2 py-1 text-xs text-[var(--text-muted)] focus:outline-none focus:border-[var(--primary)]/50"
                    >
                      <option value="" disabled>Assign to team...</option>
                      {teams.map((t) => (
                        <option key={t.id} value={t.id}>{t.name}</option>
                      ))}
                    </select>
                  )}
                </div>
              ))}
            </div>
          </section>
        )}

        {/* Create Team Modal */}
        {showCreate && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div className="absolute inset-0 bg-black/60" onClick={() => setShowCreate(false)} />
            <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-sm p-6">
              <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>New Team</h2>
              <div className="mb-4">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => { setNewName(e.target.value); setNewSlug(generateSlug(e.target.value)); }}
                  placeholder="Team name"
                  data-qa="new-team-name-input"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && newName.trim() && handleCreate()}
                />
              </div>
              <div className="mb-6">
                <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Slug</label>
                <input
                  type="text"
                  value={newSlug}
                  onChange={(e) => setNewSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
                  placeholder="team-slug"
                  data-qa="new-team-slug-input"
                  className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 font-mono"
                />
              </div>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => { setShowCreate(false); setNewName(''); setNewSlug(''); }}
                  className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreate}
                  disabled={!newName.trim() || !newSlug.trim() || creating}
                  data-qa="confirm-create-team-btn"
                  className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
                >
                  {creating ? 'Creating...' : 'Create'}
                </button>
              </div>
            </div>
          </div>
        )}

        <DeleteConfirmModal
          open={!!deleteTarget}
          title="Delete Team?"
          description={`This will permanently delete "${deleteTarget?.name}". Members will become unassigned.`}
          confirmLabel="Delete Team"
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
          loading={deleting}
        />
      </div>
    </div>
  );
}

function UserRow({ user, onToggleRole, onRemove }: { user: UserResponse; onToggleRole: () => void; onRemove: () => void }) {
  return (
    <div className="flex items-center justify-between px-4 py-2.5">
      <div className="flex items-center gap-2.5">
        <User size={13} className="text-[var(--text-dim)]" />
        <span className="text-sm text-[var(--text-primary)]">{user.display_name || user.email}</span>
        <span className="text-xs text-[var(--text-dim)]">{user.email}</span>
        <button
          onClick={onToggleRole}
          data-qa="toggle-user-role-btn"
          className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono cursor-pointer transition-colors ${
            user.role === 'admin'
              ? 'bg-[var(--primary)]/15 text-[var(--primary)]'
              : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)] hover:text-[var(--text-muted)]'
          }`}
        >
          {user.role}
        </button>
      </div>
      <button
        onClick={onRemove}
        data-qa="remove-user-from-team-btn"
        className="text-xs text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors px-2 py-0.5"
      >
        Remove
      </button>
    </div>
  );
}
