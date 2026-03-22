import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { X, Plus, Loader2 } from 'lucide-react';
import { listAgentSkills, listSkills, addSkillToAgent, removeSkillFromAgent } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { SkillResponse } from '../lib/types';

interface AgentSkillsPanelProps {
  agentSlug: string;
  agentName: string;
}

export default function AgentSkillsPanel({ agentSlug, agentName: _agentName }: AgentSkillsPanelProps) {
  const [assignedSkills, setAssignedSkills] = useState<SkillResponse[]>([]);
  const [allSkills, setAllSkills] = useState<SkillResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [addingSlug, setAddingSlug] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [assigned, all] = await Promise.all([
        listAgentSkills(agentSlug),
        listSkills(),
      ]);
      setAssignedSkills(assigned ?? []);
      setAllSkills(all ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [agentSlug]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(useCallback((event) => {
    if (event.type === 'agent_skill_added' || event.type === 'agent_skill_removed') {
      const eventData = event.data as { agent_slug?: string };
      if (eventData?.agent_slug === agentSlug) {
        fetchData();
      }
    }
  }, [fetchData, agentSlug]));

  const assignedSlugs = new Set(assignedSkills.map(s => s.slug));
  const availableSkills = allSkills.filter(s => !assignedSlugs.has(s.slug));

  const handleAdd = async () => {
    if (!addingSlug) return;
    setSaving(true);
    setError(null);
    try {
      await addSkillToAgent(agentSlug, { skill_slug: addingSlug });
      setAddingSlug('');
      await fetchData();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to add skill');
    } finally {
      setSaving(false);
    }
  };

  const handleRemove = async (skillSlug: string) => {
    setSaving(true);
    setError(null);
    try {
      await removeSkillFromAgent(agentSlug, skillSlug);
      await fetchData();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to remove skill');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mt-6">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wide mb-3">
        Skills
      </h3>

      {loading ? (
        <div className="flex items-center gap-2 text-muted-foreground text-sm">
          <Loader2 size={14} className="animate-spin" /> Loading...
        </div>
      ) : (
        <>
          {assignedSkills.length === 0 ? (
            <p className="text-sm text-muted-foreground italic">No skills assigned</p>
          ) : (
            <ul className="space-y-1 mb-3">
              {assignedSkills.map(skill => (
                <li key={skill.slug}
                    className="flex items-center justify-between rounded px-2 py-1 bg-muted/30 text-sm">
                  <span className="flex items-center gap-1.5">
                    {skill.icon && <span>{skill.icon}</span>}
                    <span>{skill.name}</span>
                    <span className="text-xs text-muted-foreground font-mono">{skill.slug}</span>
                  </span>
                  <button
                    onClick={() => handleRemove(skill.slug)}
                    disabled={saving}
                    data-qa={`skill-remove-btn-${skill.slug}`}
                    className="p-1 rounded hover:bg-destructive/20 hover:text-destructive transition-colors"
                    title="Remove skill"
                  >
                    <X size={12} />
                  </button>
                </li>
              ))}
            </ul>
          )}

          {allSkills.length === 0 ? (
            <p className="text-sm text-muted-foreground italic">
              Create skills in the{' '}
              <Link to="/skills" data-qa="skills-page-link" className="underline hover:text-foreground transition-colors">
                Skills page
              </Link>{' '}
              first.
            </p>
          ) : availableSkills.length === 0 ? (
            <p className="text-sm text-muted-foreground italic">All available skills are assigned.</p>
          ) : (
            <div className="flex items-center gap-2">
              <select
                value={addingSlug}
                onChange={e => setAddingSlug(e.target.value)}
                data-qa="skill-add-select"
                className="flex-1 text-sm rounded border border-border bg-background px-2 py-1"
              >
                <option value="">Select a skill...</option>
                {availableSkills.map(s => (
                  <option key={s.slug} value={s.slug}>{s.name}</option>
                ))}
              </select>
              <button
                onClick={handleAdd}
                disabled={!addingSlug || saving}
                data-qa="skill-add-btn"
                className="px-2 py-1 text-sm rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
              >
                {saving ? <Loader2 size={12} className="animate-spin" /> : <Plus size={12} />}
              </button>
            </div>
          )}

          {error && (
            <p className="mt-2 text-xs text-destructive">{error}</p>
          )}
        </>
      )}
    </div>
  );
}
