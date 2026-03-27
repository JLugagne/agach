import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Loader2, Plus, Trash2, Users, User, Server, Shield, ShieldOff, ChevronLeft, Search, Copy, Check, Mail, ChevronDown } from 'lucide-react';
import {
  listTeams, createTeam, deleteTeam,
  listUsers, setUserTeam, removeUserFromTeam, setUserRole,
  blockUser, unblockUser,
  adminListNodes, adminRevokeNode,
  inviteUser,
} from '../lib/api';
import DeleteConfirmModal from '../components/ui/DeleteConfirmModal';
import type { TeamResponse, UserResponse, AdminNodeResponse } from '../lib/types';

type AdminTab = 'nodes' | 'teams' | 'users';

function tabFromPath(pathname: string): AdminTab {
  if (pathname.endsWith('/teams')) return 'teams';
  if (pathname.endsWith('/users')) return 'users';
  return 'nodes';
}

export default function AdminSettingsPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const activeTab = tabFromPath(location.pathname);

  const setTab = (tab: AdminTab) => {
    const path = tab === 'nodes' ? '/admin' : `/admin/${tab}`;
    navigate(path);
  };

  const tabs: { key: AdminTab; label: string; icon: typeof Server }[] = [
    { key: 'nodes', label: 'Nodes', icon: Server },
    { key: 'teams', label: 'Teams', icon: Users },
    { key: 'users', label: 'Users', icon: User },
  ];

  return (
    <div className="h-full bg-[var(--bg-secondary)] flex flex-col md:flex-row overflow-hidden">
      <aside className="md:w-56 bg-[#0D0D0D] border-b md:border-b-0 md:border-r border-[#2A2A2A] flex flex-col shrink-0">
        <div className="p-4 border-b border-[var(--border-primary)]">
          <button
            onClick={() => navigate('/')}
            className="flex items-center gap-2 text-[var(--text-muted)] hover:text-[#E0E0E0] text-sm transition-colors cursor-pointer"
          >
            <ChevronLeft size={14} />
            <span>Back</span>
          </button>
        </div>

        <div className="hidden md:block p-4">
          <p className="font-heading text-sm text-[var(--text-primary)] truncate mb-1">Administration</p>
          <p className="text-xs text-[var(--text-dim)]">System settings</p>
        </div>

        <nav className="flex md:flex-col md:flex-1 px-2 py-1 md:py-0 overflow-x-auto">
          {tabs.map((tab) => {
            const isActive = activeTab === tab.key;
            const Icon = tab.icon;
            return (
              <button
                key={tab.key}
                onClick={() => setTab(tab.key)}
                data-qa={`admin-tab-${tab.key}`}
                className={`flex items-center gap-2.5 px-3 py-2 rounded-md text-sm mb-0.5 transition-colors whitespace-nowrap cursor-pointer w-full text-left ${
                  isActive
                    ? 'bg-[var(--bg-secondary)] text-[var(--text-primary)]'
                    : 'text-[var(--text-muted)] hover:text-[#E0E0E0] hover:bg-[var(--bg-secondary)]/50'
                }`}
              >
                <Icon size={15} />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </aside>

      <div className="flex-1 min-w-0 overflow-hidden">
        <main className="h-full overflow-y-auto">
          {activeTab === 'nodes' && <AdminNodesTab />}
          {activeTab === 'teams' && <AdminTeamsTab />}
          {activeTab === 'users' && <AdminUsersTab />}
        </main>
      </div>
    </div>
  );
}

// ─── Nodes Tab ───────────────────────────────────────────────────────────────

function AdminNodesTab() {
  const [nodes, setNodes] = useState<AdminNodeResponse[]>([]);
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [revoking, setRevoking] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const [nodesData, usersData] = await Promise.all([adminListNodes(), listUsers()]);
      setNodes(nodesData.nodes ?? []);
      setUsers(usersData ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleRevoke = async () => {
    if (!revokeTarget) return;
    setRevoking(true);
    try {
      await adminRevokeNode(revokeTarget);
      setRevokeTarget(null);
      fetchData();
    } catch {
      // ignore
    } finally {
      setRevoking(false);
    }
  };

  const ownerName = (ownerUserId: string) => {
    const u = users.find(u => u.id === ownerUserId);
    return u?.display_name || u?.email || ownerUserId.slice(0, 8);
  };

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

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
      <h1 className="text-[28px] font-semibold text-[var(--text-primary)] mb-2" style={{ fontFamily: 'Inter, sans-serif' }}>
        All Nodes
      </h1>
      <p className="text-sm text-[var(--text-muted)] mb-8" style={{ fontFamily: 'Inter, sans-serif' }}>
        {activeNodes.length} active · {revokedNodes.length} revoked
      </p>

      {nodes.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 gap-4">
          <div className="w-16 h-16 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
            <Server size={28} className="text-[var(--text-muted)]" />
          </div>
          <p className="text-sm text-[var(--text-muted)]">No nodes registered.</p>
        </div>
      ) : (
        <>
          {activeNodes.length > 0 && (
            <div className="mb-8">
              <h2 className="text-xs font-semibold tracking-[2px] text-[var(--text-muted)] mb-3 uppercase font-mono">
                Active ({activeNodes.length})
              </h2>
              <div className="border border-[var(--border-primary)] rounded-lg overflow-hidden divide-y divide-[var(--border-primary)]">
                {activeNodes.map((node) => (
                  <div key={node.id} className="flex items-center justify-between px-4 py-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 shrink-0" style={{ boxShadow: '0 0 8px rgba(16,185,129,0.3)' }} />
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-[var(--text-primary)] truncate font-mono">
                            {node.name || 'Unnamed'}
                          </span>
                          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-[var(--bg-tertiary)] text-[var(--text-dim)] font-mono">
                            {node.mode}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 mt-0.5">
                          <span className="text-xs text-[var(--text-dim)]">
                            Owner: {ownerName(node.owner_user_id)}
                          </span>
                          <span className="text-xs text-[var(--text-dim)] font-mono">
                            Last seen: {formatDate(node.last_seen_at)}
                          </span>
                        </div>
                      </div>
                    </div>
                    <button
                      onClick={() => setRevokeTarget(node.id)}
                      data-qa="admin-revoke-node-btn"
                      className="text-xs font-medium px-3 py-1.5 rounded-md border border-red-500/40 text-red-400 hover:bg-red-500/10 transition-colors shrink-0 cursor-pointer"
                    >
                      Revoke
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {revokedNodes.length > 0 && (
            <div>
              <h2 className="text-xs font-semibold tracking-[2px] text-[var(--text-muted)] mb-3 uppercase font-mono">
                Revoked ({revokedNodes.length})
              </h2>
              <div className="border border-[var(--border-primary)] rounded-lg overflow-hidden divide-y divide-[var(--border-primary)] opacity-60">
                {revokedNodes.map((node) => (
                  <div key={node.id} className="flex items-center justify-between px-4 py-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="w-2.5 h-2.5 rounded-full bg-red-500 shrink-0" />
                      <div className="min-w-0">
                        <span className="text-sm text-[var(--text-primary)] truncate font-mono">
                          {node.name || 'Unnamed'}
                        </span>
                        <div className="flex items-center gap-3 mt-0.5">
                          <span className="text-xs text-[var(--text-dim)]">
                            Owner: {ownerName(node.owner_user_id)}
                          </span>
                          <span className="text-xs text-red-400 font-mono">
                            Revoked: {formatDate(node.revoked_at)}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      <DeleteConfirmModal
        open={revokeTarget !== null}
        title="Revoke Node"
        description="Are you sure you want to revoke this node? The daemon will be disconnected immediately."
        confirmLabel="Revoke"
        onConfirm={handleRevoke}
        onCancel={() => setRevokeTarget(null)}
        loading={revoking}
      />
    </div>
  );
}

// ─── Teams Tab ───────────────────────────────────────────────────────────────

function AdminTeamsTab() {
  const [teams, setTeams] = useState<TeamResponse[]>([]);
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newSlug, setNewSlug] = useState('');
  const [creating, setCreating] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TeamResponse | null>(null);
  const [deleting, setDeleting] = useState(false);
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

  const unassignedUsers = users.filter((u) => u.team_ids.length === 0);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
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

                {addUserTeamId === team.id && (
                  <div className="px-4 py-2 border-t border-[var(--border-primary)] bg-[var(--bg-tertiary)]">
                    {nonMembers.length > 0 ? (
                      <UserSearchInput
                        users={nonMembers}
                        placeholder="Search users..."
                        onSelect={(userId) => handleAssignUser(userId, team.id)}
                        data-qa="assign-user-search"
                      />
                    ) : (
                      <p className="text-xs text-[var(--text-dim)] italic">All users are already in this team.</p>
                    )}
                  </div>
                )}

                {members.length > 0 && (
                  <div className="divide-y divide-[var(--border-primary)]">
                    {members.map((u) => (
                      <div key={u.id} className="flex items-center justify-between px-4 py-2.5">
                        <div className="flex items-center gap-2.5">
                          <User size={13} className="text-[var(--text-dim)]" />
                          <span className="text-sm text-[var(--text-primary)]">{u.display_name || u.email}</span>
                          <span className="text-xs text-[var(--text-dim)]">{u.email}</span>
                        </div>
                        <button
                          onClick={() => handleRemoveUser(u.id, team.id)}
                          data-qa="remove-user-from-team-btn"
                          className="text-xs text-[var(--text-dim)] hover:text-[#FF3B30] transition-colors px-2 py-0.5"
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

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
                </div>
                {teams.length > 0 && (
                  <div className="w-48">
                    <TeamSearchInput
                      teams={teams}
                      placeholder="Assign to team..."
                      onSelect={(teamId) => handleAssignUser(u.id, teamId)}
                      data-qa="assign-to-team-search"
                    />
                  </div>
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
  );
}

// ─── Users Tab ───────────────────────────────────────────────────────────────

function AdminUsersTab() {
  const [users, setUsers] = useState<UserResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [showInvite, setShowInvite] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviting, setInviting] = useState(false);
  const [inviteLink, setInviteLink] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      const u = await listUsers();
      setUsers(u ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleSetRole = async (user: UserResponse, role: 'admin' | 'member') => {
    if (user.role === role) return;
    try {
      await setUserRole(user.id, role);
      fetchData();
    } catch {
      // ignore
    }
  };

  const handleToggleBlock = async (user: UserResponse) => {
    try {
      if (user.blocked_at) {
        await unblockUser(user.id);
      } else {
        await blockUser(user.id);
      }
      fetchData();
    } catch {
      // ignore
    }
  };

  const handleInvite = async () => {
    if (!inviteEmail.trim()) return;
    setInviting(true);
    try {
      const data = await inviteUser(inviteEmail.trim());
      const url = `${window.location.origin}/invite?token=${data.invite_token}`;
      setInviteLink(url);
      fetchData();
    } catch {
      // ignore
    } finally {
      setInviting(false);
    }
  };

  const handleCopyLink = async () => {
    if (!inviteLink) return;
    await navigator.clipboard.writeText(inviteLink);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const closeInviteModal = () => {
    setShowInvite(false);
    setInviteEmail('');
    setInviteLink(null);
    setCopied(false);
  };

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
      </div>
    );
  }

  const activeUsers = users.filter(u => !u.blocked_at);
  const blockedUsers = users.filter(u => u.blocked_at);

  return (
    <div className="max-w-3xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
      <div className="flex items-center justify-between mb-2">
        <h1 className="text-[28px] font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
          Users
        </h1>
        <button
          onClick={() => setShowInvite(true)}
          data-qa="invite-user-btn"
          className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          <Mail size={14} />
          Invite User
        </button>
      </div>
      <p className="text-sm text-[var(--text-muted)] mb-8" style={{ fontFamily: 'Inter, sans-serif' }}>
        {activeUsers.length} active · {blockedUsers.length} blocked
      </p>

      {users.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 gap-4">
          <div className="w-16 h-16 rounded-2xl bg-[var(--bg-tertiary)] flex items-center justify-center">
            <User size={28} className="text-[var(--text-muted)]" />
          </div>
          <p className="text-sm text-[var(--text-muted)]">No users found.</p>
        </div>
      ) : (
        <>
          {activeUsers.length > 0 && (
            <div className="mb-8">
              <h2 className="text-xs font-semibold tracking-[2px] text-[var(--text-muted)] mb-3 uppercase font-mono">
                Active ({activeUsers.length})
              </h2>
              <div className="border border-[var(--border-primary)] rounded-lg divide-y divide-[var(--border-primary)]">
                {activeUsers.map((u) => (
                  <UserManagementRow
                    key={u.id}
                    user={u}
                    onSetRole={(role) => handleSetRole(u, role)}
                    onToggleBlock={() => handleToggleBlock(u)}
                  />
                ))}
              </div>
            </div>
          )}

          {blockedUsers.length > 0 && (
            <div>
              <h2 className="text-xs font-semibold tracking-[2px] text-[var(--text-muted)] mb-3 uppercase font-mono">
                Blocked ({blockedUsers.length})
              </h2>
              <div className="border border-[var(--border-primary)] rounded-lg divide-y divide-[var(--border-primary)] opacity-70">
                {blockedUsers.map((u) => (
                  <UserManagementRow
                    key={u.id}
                    user={u}
                    onSetRole={(role) => handleSetRole(u, role)}
                    onToggleBlock={() => handleToggleBlock(u)}
                  />
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {/* Invite User Modal */}
      {showInvite && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={closeInviteModal} />
          <div className="relative bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-lg w-full max-w-md p-6">
            <h2 className="text-lg text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
              Invite User
            </h2>

            {!inviteLink ? (
              <>
                <p className="text-sm text-[var(--text-muted)] mb-4">
                  Enter the email address. You'll get a link to share with the user to complete their registration.
                </p>
                <div className="mb-6">
                  <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Email</label>
                  <input
                    type="email"
                    value={inviteEmail}
                    onChange={(e) => setInviteEmail(e.target.value)}
                    placeholder="user@example.com"
                    data-qa="invite-email-input"
                    className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
                    autoFocus
                    onKeyDown={(e) => e.key === 'Enter' && inviteEmail.trim() && handleInvite()}
                  />
                </div>
                <div className="flex justify-end gap-3">
                  <button
                    onClick={closeInviteModal}
                    className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleInvite}
                    disabled={!inviteEmail.trim() || inviting}
                    data-qa="confirm-invite-btn"
                    className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
                  >
                    {inviting ? 'Creating...' : 'Create Invite'}
                  </button>
                </div>
              </>
            ) : (
              <>
                <p className="text-sm text-[var(--text-muted)] mb-4">
                  Share this link with <span className="text-[var(--text-primary)] font-medium">{inviteEmail}</span>. It expires in 24 hours.
                </p>
                <div className="flex gap-2 mb-6">
                  <input
                    type="text"
                    readOnly
                    value={inviteLink}
                    data-qa="invite-link-input"
                    className="flex-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-xs text-[var(--text-muted)] font-mono focus:outline-none select-all"
                    onClick={(e) => (e.target as HTMLInputElement).select()}
                  />
                  <button
                    onClick={handleCopyLink}
                    data-qa="copy-invite-link-btn"
                    className={`flex items-center gap-1.5 px-3 py-2 rounded-md text-xs font-medium transition-colors cursor-pointer ${
                      copied
                        ? 'bg-emerald-500/15 text-emerald-400'
                        : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)] hover:text-[var(--text-primary)]'
                    }`}
                  >
                    {copied ? <Check size={13} /> : <Copy size={13} />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <div className="flex justify-end">
                  <button
                    onClick={closeInviteModal}
                    className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 transition-colors"
                  >
                    Done
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

const AVAILABLE_ROLES: Array<'admin' | 'member'> = ['admin', 'member'];

function UserManagementRow({ user, onSetRole, onToggleBlock }: {
  user: UserResponse;
  onSetRole: (role: 'admin' | 'member') => void;
  onToggleBlock: () => void;
}) {
  const isBlocked = !!user.blocked_at;
  const [roleMenuOpen, setRoleMenuOpen] = useState(false);
  const roleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (roleRef.current && !roleRef.current.contains(e.target as Node)) setRoleMenuOpen(false);
    }
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <div className="flex items-center justify-between px-4 py-3">
      <div className="flex items-center gap-2.5 min-w-0">
        <User size={14} className="text-[var(--text-dim)] shrink-0" />
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm text-[var(--text-primary)] truncate">{user.display_name || user.email}</span>
            <div ref={roleRef} className="relative">
              <button
                onClick={() => setRoleMenuOpen(!roleMenuOpen)}
                data-qa="toggle-user-role-btn"
                className={`flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full font-mono cursor-pointer transition-colors ${
                  user.role === 'admin'
                    ? 'bg-[var(--primary)]/15 text-[var(--primary)]'
                    : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)] hover:text-[var(--text-muted)]'
                }`}
              >
                {user.role}
                <ChevronDown size={10} />
              </button>
              {roleMenuOpen && (
                <div className="absolute z-50 left-0 mt-1 min-w-[120px] rounded-md border border-[var(--border-primary)] bg-[var(--bg-primary)] shadow-lg py-1">
                  {AVAILABLE_ROLES.map((role) => (
                    <button
                      key={role}
                      onMouseDown={() => { onSetRole(role); setRoleMenuOpen(false); }}
                      data-qa={`set-role-${role}-btn`}
                      className="flex items-center justify-between w-full px-3 py-1.5 text-xs text-left transition-colors cursor-pointer hover:bg-[var(--bg-tertiary)] text-[var(--text-secondary)]"
                    >
                      <span className="font-mono">{role}</span>
                      {user.role === role && <Check size={12} className="text-[var(--primary)] shrink-0" />}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-mono ${
              user.sso_provider
                ? 'bg-blue-500/15 text-blue-400'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-dim)]'
            }`}>
              {user.sso_provider || 'internal'}
            </span>
            {isBlocked && (
              <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/15 text-red-400 font-mono">
                blocked
              </span>
            )}
          </div>
          <span className="text-xs text-[var(--text-dim)]">{user.email}</span>
        </div>
      </div>
      <button
        onClick={onToggleBlock}
        data-qa={isBlocked ? 'unblock-user-btn' : 'block-user-btn'}
        className={`flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-md transition-colors cursor-pointer ${
          isBlocked
            ? 'text-emerald-400 hover:bg-emerald-500/10'
            : 'text-[var(--text-dim)] hover:text-red-400 hover:bg-red-500/10'
        }`}
      >
        {isBlocked ? <ShieldOff size={12} /> : <Shield size={12} />}
        {isBlocked ? 'Unblock' : 'Block'}
      </button>
    </div>
  );
}

// ─── Autocomplete search inputs ──────────────────────────────────────────────

function UserSearchInput({ users, placeholder, onSelect, 'data-qa': dataQa }: {
  users: UserResponse[];
  placeholder: string;
  onSelect: (userId: string) => void;
  'data-qa'?: string;
}) {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = query.trim()
    ? users.filter((u) => {
        const q = query.toLowerCase();
        return (u.display_name || '').toLowerCase().includes(q) || (u.email || '').toLowerCase().includes(q);
      })
    : users;

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  useEffect(() => { setHighlighted(0); }, [query]);

  const handleSelect = (userId: string) => {
    setQuery('');
    setOpen(false);
    onSelect(userId);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open && e.key === 'ArrowDown') { setOpen(true); return; }
    if (!open) return;
    if (e.key === 'ArrowDown') { e.preventDefault(); setHighlighted((h) => Math.min(h + 1, filtered.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setHighlighted((h) => Math.max(h - 1, 0)); }
    else if (e.key === 'Enter' && filtered[highlighted]) { e.preventDefault(); handleSelect(filtered[highlighted].id); }
    else if (e.key === 'Escape') { setOpen(false); }
  };

  return (
    <div ref={ref} className="relative">
      <div className="relative">
        <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-dim)]" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          data-qa={dataQa}
          className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md pl-8 pr-3 py-1.5 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
          autoFocus
        />
      </div>
      {open && filtered.length > 0 && (
        <div className="absolute z-50 left-0 right-0 mt-1 max-h-48 overflow-y-auto rounded-md border border-[var(--border-primary)] bg-[var(--bg-primary)] shadow-lg">
          {filtered.map((u, i) => (
            <button
              key={u.id}
              onMouseDown={() => handleSelect(u.id)}
              onMouseEnter={() => setHighlighted(i)}
              className={`flex items-center gap-2.5 w-full px-3 py-2 text-left text-sm transition-colors cursor-pointer ${
                i === highlighted ? 'bg-[var(--primary)]/10 text-[var(--text-primary)]' : 'text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50'
              }`}
            >
              <div className="w-6 h-6 rounded-full bg-[var(--bg-tertiary)] flex items-center justify-center shrink-0">
                <span className="text-[10px] font-semibold text-[var(--text-muted)]">
                  {(u.display_name || u.email || '?')[0].toUpperCase()}
                </span>
              </div>
              <div className="min-w-0">
                <div className="text-sm truncate">{u.display_name || u.email}</div>
                {u.display_name && u.email && (
                  <div className="text-[11px] text-[var(--text-dim)] truncate">{u.email}</div>
                )}
              </div>
            </button>
          ))}
        </div>
      )}
      {open && query.trim() && filtered.length === 0 && (
        <div className="absolute z-50 left-0 right-0 mt-1 rounded-md border border-[var(--border-primary)] bg-[var(--bg-primary)] shadow-lg px-3 py-2">
          <p className="text-xs text-[var(--text-dim)] italic">No matching users</p>
        </div>
      )}
    </div>
  );
}

function TeamSearchInput({ teams, placeholder, onSelect, 'data-qa': dataQa }: {
  teams: TeamResponse[];
  placeholder: string;
  onSelect: (teamId: string) => void;
  'data-qa'?: string;
}) {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [highlighted, setHighlighted] = useState(0);
  const ref = useRef<HTMLDivElement>(null);

  const filtered = query.trim()
    ? teams.filter((t) => {
        const q = query.toLowerCase();
        return t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q);
      })
    : teams;

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  useEffect(() => { setHighlighted(0); }, [query]);

  const handleSelect = (teamId: string) => {
    setQuery('');
    setOpen(false);
    onSelect(teamId);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!open && e.key === 'ArrowDown') { setOpen(true); return; }
    if (!open) return;
    if (e.key === 'ArrowDown') { e.preventDefault(); setHighlighted((h) => Math.min(h + 1, filtered.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setHighlighted((h) => Math.max(h - 1, 0)); }
    else if (e.key === 'Enter' && filtered[highlighted]) { e.preventDefault(); handleSelect(filtered[highlighted].id); }
    else if (e.key === 'Escape') { setOpen(false); }
  };

  return (
    <div ref={ref} className="relative">
      <div className="relative">
        <Search size={13} className="absolute left-2 top-1/2 -translate-y-1/2 text-[var(--text-dim)]" />
        <input
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          data-qa={dataQa}
          className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md pl-7 pr-2 py-1 text-xs text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
        />
      </div>
      {open && filtered.length > 0 && (
        <div className="absolute z-50 left-0 right-0 mt-1 max-h-40 overflow-y-auto rounded-md border border-[var(--border-primary)] bg-[var(--bg-primary)] shadow-lg">
          {filtered.map((t, i) => (
            <button
              key={t.id}
              onMouseDown={() => handleSelect(t.id)}
              onMouseEnter={() => setHighlighted(i)}
              className={`flex items-center gap-2 w-full px-3 py-1.5 text-left text-xs transition-colors cursor-pointer ${
                i === highlighted ? 'bg-[var(--primary)]/10 text-[var(--text-primary)]' : 'text-[var(--text-secondary)] hover:bg-[var(--nav-bg-active)]/50'
              }`}
            >
              <Users size={12} className="text-[var(--text-dim)] shrink-0" />
              <span className="truncate">{t.name}</span>
              <span className="text-[var(--text-dim)] font-mono shrink-0">{t.slug}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
