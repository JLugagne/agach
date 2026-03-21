import { useState, useEffect, useRef } from 'react';
import { Loader2, X } from 'lucide-react';
import { cloneRole } from '../lib/api';
import type { RoleResponse } from '../lib/types';

interface CloneAgentDialogProps {
  sourceRole: RoleResponse;
  onClose: () => void;
  onSuccess: (cloned: RoleResponse) => void;
}

export default function CloneAgentDialog({ sourceRole, onClose, onSuccess }: CloneAgentDialogProps) {
  const [newSlug, setNewSlug] = useState('');
  const [newName, setNewName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);
  const slugInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setNewSlug(sourceRole.slug + '-copy');
    setNewName(sourceRole.name + ' (copy)');
  }, [sourceRole]);

  useEffect(() => {
    slugInputRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const validateSlug = (value: string) => {
    if (!value) return 'Slug is required';
    if (!/^[a-z0-9-]+$/.test(value)) return 'Slug must be lowercase alphanumeric with hyphens only';
    if (value.length > 50) return 'Slug must be 50 characters or fewer';
    return null;
  };

  const handleSlugChange = (value: string) => {
    setNewSlug(value);
    setSlugError(validateSlug(value));
  };

  const handleSubmit = async () => {
    const err = validateSlug(newSlug);
    if (err) { setSlugError(err); return; }
    setLoading(true);
    setError(null);
    try {
      const cloned = await cloneRole(sourceRole.slug, {
        new_slug: newSlug,
        new_name: newName || undefined,
      });
      onSuccess(cloned);
      onClose();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Clone failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-md bg-[#111111] border border-[#1E1E1E] rounded-lg shadow-xl">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#1E1E1E]">
          <h2 className="text-base text-[#F0F0F0]" style={{ fontFamily: 'Newsreader, Georgia, serif' }}>
            Clone &ldquo;{sourceRole.name}&rdquo;
          </h2>
          <button
            onClick={onClose}
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        <div className="px-6 py-5 space-y-4">
          <p className="text-xs text-[var(--text-dim)]">
            Cloning from: <span className="font-mono">{sourceRole.slug}</span>
          </p>

          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">
              New Slug <span className="text-[#F06060]">*</span>
            </label>
            <input
              ref={slugInputRef}
              type="text"
              value={newSlug}
              onChange={(e) => handleSlugChange(e.target.value)}
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm font-mono text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
              placeholder="new-agent-slug"
            />
            {slugError && (
              <p className="mt-1 text-xs text-[#F06060]">{slugError}</p>
            )}
          </div>

          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">
              New Name <span className="text-[var(--text-dim)]">(optional)</span>
            </label>
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="w-full bg-[#1A1A1A] border border-[#252525] rounded-md px-3 py-2 text-sm text-[#F0F0F0] placeholder-[var(--text-dim)] focus:outline-none focus:border-[#00C896]/50"
              placeholder="New Agent Name"
            />
          </div>

          {error && (
            <div className="px-3 py-2 bg-[#F06060]/10 border border-[#F06060]/30 rounded-md">
              <p className="text-xs text-[#F06060]">{error}</p>
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[#1E1E1E]">
          <button
            onClick={onClose}
            disabled={loading}
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[#E0E0E0] transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || !!slugError || !newSlug}
            className="flex items-center gap-2 px-4 py-2 bg-[#00C896] text-[#0F0F0F] text-sm font-medium rounded-md hover:bg-[#00C896]/80 disabled:opacity-50 transition-colors"
          >
            {loading && <Loader2 size={14} className="animate-spin" />}
            Clone Agent
          </button>
        </div>
      </div>
    </div>
  );
}
