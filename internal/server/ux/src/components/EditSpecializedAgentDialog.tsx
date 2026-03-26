import { useState, useEffect, useMemo } from 'react';
import { X, Search, Loader2 } from 'lucide-react';
import { listSkills, listSpecializedAgentSkills, createSpecializedAgent, updateSpecializedAgent } from '../lib/api';
import type { SpecializedAgentResponse, SkillResponse } from '../lib/types';

interface EditSpecializedAgentDialogProps {
  parentSlug: string;
  specializedAgent?: SpecializedAgentResponse | null;
  onClose: () => void;
  onSaved: () => void;
}

export default function EditSpecializedAgentDialog({ parentSlug, specializedAgent, onClose, onSaved }: EditSpecializedAgentDialogProps) {
  const isEdit = !!specializedAgent;
  const [name, setName] = useState(specializedAgent?.name ?? '');
  const [slug, setSlug] = useState(specializedAgent?.slug ?? '');
  const [autoSlug, setAutoSlug] = useState(!isEdit);
  const [allSkills, setAllSkills] = useState<SkillResponse[]>([]);
  const [selectedSlugs, setSelectedSlugs] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const [skills, assigned] = await Promise.all([
          listSkills(),
          isEdit ? listSpecializedAgentSkills(parentSlug, specializedAgent!.slug) : Promise.resolve([]),
        ]);
        setAllSkills(skills ?? []);
        if (assigned && assigned.length > 0) {
          setSelectedSlugs(new Set(assigned.map(s => s.slug)));
        }
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    })();
  }, [parentSlug, specializedAgent, isEdit]);

  const generateSlug = (n: string) =>
    n.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 50);

  const handleNameChange = (val: string) => {
    setName(val);
    if (autoSlug) {
      setSlug(generateSlug(val));
    }
  };

  const handleSlugChange = (val: string) => {
    setAutoSlug(false);
    setSlug(val.toLowerCase().replace(/[^a-z0-9-]/g, ''));
  };

  const toggleSkill = (skillSlug: string) => {
    setSelectedSlugs(prev => {
      const next = new Set(prev);
      if (next.has(skillSlug)) {
        next.delete(skillSlug);
      } else {
        next.add(skillSlug);
      }
      return next;
    });
  };

  const filteredSkills = useMemo(() => {
    if (!search.trim()) return allSkills;
    const q = search.toLowerCase();
    return allSkills.filter(s => s.name.toLowerCase().includes(q) || s.slug.toLowerCase().includes(q));
  }, [allSkills, search]);

  const handleSave = async () => {
    if (!name.trim() || !slug.trim()) return;
    setSaving(true);
    setError(null);
    try {
      if (isEdit) {
        await updateSpecializedAgent(parentSlug, specializedAgent!.slug, {
          name: name.trim(),
          skill_slugs: Array.from(selectedSlugs),
        });
      } else {
        await createSpecializedAgent(parentSlug, {
          slug: slug.trim(),
          name: name.trim(),
          skill_slugs: Array.from(selectedSlugs),
        });
      }
      onSaved();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60" onClick={onClose}>
      <div
        className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-xl p-0 w-full max-w-lg shadow-xl"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start justify-between px-6 pt-5 pb-4 border-b border-[var(--border-primary)]">
          <div>
            <h2 className="text-base font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              {isEdit ? 'Edit Specialized Agent' : 'New Specialized Agent'}
            </h2>
            <p className="text-xs text-[var(--text-muted)] mt-0.5">
              Customize skills for this agent specialization
            </p>
          </div>
          <button
            onClick={onClose}
            data-qa="edit-specialized-close-btn"
            className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors mt-0.5"
          >
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5 space-y-5 max-h-[60vh] overflow-y-auto">
          {/* Name */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Name</label>
            <input
              type="text"
              value={name}
              onChange={e => handleNameChange(e.target.value)}
              placeholder="e.g. Go Backend"
              data-qa="specialized-agent-name-input"
              className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
              autoFocus
            />
          </div>

          {/* Slug (create only) */}
          {!isEdit && (
            <div>
              <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">Slug</label>
              <input
                type="text"
                value={slug}
                onChange={e => handleSlugChange(e.target.value)}
                placeholder="go-backend"
                data-qa="specialized-agent-slug-input"
                className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50 font-mono"
              />
            </div>
          )}

          {/* Skills */}
          <div>
            <label className="block text-xs font-mono text-[var(--text-dim)] mb-1.5">
              Skills ({selectedSlugs.size} selected)
            </label>

            {/* Search */}
            <div className="relative mb-3">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-dim)]" />
              <input
                type="text"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search skills..."
                data-qa="specialized-agent-skill-search"
                className="w-full bg-[var(--bg-primary)] border border-[var(--border-primary)] rounded-md pl-9 pr-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-dim)] focus:outline-none focus:border-[var(--primary)]/50"
              />
            </div>

            {loading ? (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="animate-spin text-[var(--text-dim)]" size={20} />
              </div>
            ) : filteredSkills.length === 0 ? (
              <p className="text-sm text-[var(--text-muted)] py-3">
                {search ? 'No skills matching search.' : 'No skills available.'}
              </p>
            ) : (
              <div className="space-y-1 max-h-[240px] overflow-y-auto">
                {filteredSkills.map(skill => (
                  <label
                    key={skill.slug}
                    className="flex items-center gap-3 px-3 py-2 rounded-md hover:bg-[var(--bg-primary)] transition-colors cursor-pointer"
                    data-qa="specialized-agent-skill-item"
                  >
                    <input
                      type="checkbox"
                      checked={selectedSlugs.has(skill.slug)}
                      onChange={() => toggleSkill(skill.slug)}
                      className="w-4 h-4 rounded border-[var(--border-primary)] accent-[var(--primary)]"
                    />
                    <span className="flex items-center gap-1.5 text-sm text-[var(--text-primary)]">
                      {skill.icon && <span>{skill.icon}</span>}
                      {skill.name}
                    </span>
                  </label>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="px-6 pb-2">
            <p className="text-xs text-[#FF3B30]">{error}</p>
          </div>
        )}

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-[var(--border-primary)]">
          <button
            onClick={onClose}
            data-qa="specialized-agent-cancel-btn"
            className="px-4 py-2 text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || !slug.trim() || saving}
            data-qa="specialized-agent-save-btn"
            className="px-4 py-2 bg-[var(--primary)] text-[var(--primary-text)] text-sm font-medium rounded-md hover:bg-[var(--primary-hover)]/80 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
}
