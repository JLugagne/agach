import { useState, useEffect, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { createProject, setProjectDockerfile, listDockerfiles, listAgents } from '../lib/api';
import type { DockerfileResponse, AgentResponse } from '../lib/types';

interface Props {
  onClose: () => void;
  onCreated?: () => void;
}

export default function CreateProjectDialog({ onClose, onCreated }: Props) {
  const navigate = useNavigate();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [gitUrl, setGitUrl] = useState('');
  const [selectedDockerfileId, setSelectedDockerfileId] = useState('');
  const [selectedRoleSlugs, setSelectedRoleSlugs] = useState<Set<string>>(new Set());

  const [dockerfiles, setDockerfiles] = useState<DockerfileResponse[]>([]);
  const [roles, setRoles] = useState<AgentResponse[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listDockerfiles(), listAgents()]).then(([dfs, rls]) => {
      setDockerfiles(dfs ?? []);
      setRoles(rls ?? []);
    }).catch(() => {});
  }, []);

  function toggleRole(slug: string) {
    setSelectedRoleSlugs(prev => {
      const next = new Set(prev);
      if (next.has(slug)) next.delete(slug);
      else next.add(slug);
      return next;
    });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const project = await createProject({
        name: name.trim(),
        description: description.trim() || undefined,
        git_url: gitUrl.trim() || undefined,
      });

      if (selectedDockerfileId) {
        await setProjectDockerfile(project.id, { dockerfile_id: selectedDockerfileId });
      }

      onCreated?.();
      navigate(`/projects/${project.id}/board`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create project');
      setLoading(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: 'rgba(0,0,0,0.5)' }}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
    >
      <form
        onSubmit={handleSubmit}
        className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] shadow-2xl w-full max-w-lg mx-4 flex flex-col max-h-[90vh]"
        style={{ fontFamily: 'Inter, sans-serif' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
          <h2 className="text-[15px] font-semibold text-[var(--text-primary)]">New Project</h2>
          <button
            type="button"
            onClick={onClose}
            data-qa="create-project-close-btn"
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="flex flex-col gap-5 px-6 py-5 overflow-y-auto flex-1">

          {/* Name */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--text-secondary)]">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              required
              autoFocus
              data-qa="create-project-name-input"
              placeholder="My awesome project"
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
            />
          </div>

          {/* Description */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--text-secondary)]">Description</label>
            <textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              data-qa="create-project-description-input"
              placeholder="What is this project about?"
              rows={2}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors resize-none"
            />
          </div>

          {/* Git URL */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--text-secondary)]">Git URL</label>
            <input
              type="text"
              value={gitUrl}
              onChange={e => setGitUrl(e.target.value)}
              data-qa="create-project-giturl-input"
              placeholder="https://github.com/org/repo"
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
              style={{ fontFamily: 'JetBrains Mono, monospace' }}
            />
          </div>

          {/* Dockerfile */}
          {dockerfiles.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]">Dockerfile</label>
              <select
                value={selectedDockerfileId}
                onChange={e => setSelectedDockerfileId(e.target.value)}
                data-qa="create-project-dockerfile-select"
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-primary)] bg-[var(--bg-primary)] text-[var(--text-primary)] text-[13px] outline-none focus:border-[var(--primary)] transition-colors"
              >
                <option value="">None</option>
                {dockerfiles.map(df => (
                  <option key={df.id} value={df.id}>
                    {df.name}{df.version ? ` (${df.version})` : ''}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Roles */}
          {roles.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--text-secondary)]">
                Roles
                <span className="ml-1 text-[11px] font-normal text-[var(--text-muted)]">
                  — highlight roles to feature on this project
                </span>
              </label>
              <div className="flex flex-wrap gap-2">
                {roles.map(role => {
                  const selected = selectedRoleSlugs.has(role.slug);
                  return (
                    <button
                      key={role.slug}
                      type="button"
                      onClick={() => toggleRole(role.slug)}
                      data-qa={`create-project-role-${role.slug}`}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium border transition-colors"
                      style={{
                        borderColor: selected ? (role.color || 'var(--primary)') : 'var(--border-primary)',
                        color: selected ? (role.color || 'var(--primary)') : 'var(--text-muted)',
                        backgroundColor: selected
                          ? `color-mix(in srgb, ${role.color || 'var(--primary)'} 12%, transparent)`
                          : 'transparent',
                      }}
                    >
                      {role.icon && <span>{role.icon}</span>}
                      {role.name}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {error && (
            <p className="text-[12px] text-red-500 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
              {error}
            </p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            type="button"
            onClick={onClose}
            data-qa="create-project-cancel-btn"
            className="px-4 py-2 rounded-lg text-[13px] font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)] transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading || !name.trim()}
            data-qa="create-project-submit-btn"
            className="px-4 py-2 rounded-lg text-[13px] font-semibold bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Creating…' : 'Create Project'}
          </button>
        </div>
      </form>
    </div>
  );
}
