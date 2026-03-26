import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ChevronRight, Pencil, Trash2, Loader2, BookOpen, ChevronDown, ChevronUp } from 'lucide-react';
import { getAgent, getSpecializedAgent, listSpecializedAgentSkills, deleteSpecializedAgent } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { AgentResponse, SpecializedAgentResponse, SkillResponse } from '../lib/types';
import EditSpecializedAgentDialog from '../components/EditSpecializedAgentDialog';

export default function SpecializedAgentDetailPage() {
  const { parentSlug, specSlug } = useParams<{ parentSlug: string; specSlug: string }>();
  const navigate = useNavigate();
  const [parent, setParent] = useState<AgentResponse | null>(null);
  const [specialized, setSpecialized] = useState<SpecializedAgentResponse | null>(null);
  const [skills, setSkills] = useState<SkillResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [hintExpanded, setHintExpanded] = useState(false);

  const fetchData = useCallback(async () => {
    if (!parentSlug || !specSlug) return;
    try {
      const [parentData, specData, skillsData] = await Promise.all([
        getAgent(parentSlug),
        getSpecializedAgent(parentSlug, specSlug),
        listSpecializedAgentSkills(parentSlug, specSlug),
      ]);
      setParent(parentData);
      setSpecialized(specData);
      setSkills(skillsData ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [parentSlug, specSlug]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(useCallback((event) => {
    if (
      event.type === 'specialized_agent_updated' ||
      event.type === 'specialized_agent_deleted'
    ) {
      fetchData();
    }
  }, [fetchData]));

  const handleDelete = async () => {
    if (!parentSlug || !specSlug) return;
    setDeleting(true);
    try {
      await deleteSpecializedAgent(parentSlug, specSlug);
      navigate('/roles');
    } catch {
      setDeleting(false);
    }
  };

  const handleEditSaved = () => {
    setEditOpen(false);
    fetchData();
  };

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <Loader2 className="animate-spin text-[var(--text-dim)]" size={24} />
      </div>
    );
  }

  if (!parent || !specialized) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <p className="text-sm text-[var(--text-muted)]">Specialized agent not found.</p>
      </div>
    );
  }

  const promptHint = parent.prompt_hint || '';
  const hintLines = promptHint.split('\n');
  const HINT_COLLAPSED_LINES = 8;
  const showToggle = hintLines.length > HINT_COLLAPSED_LINES;
  const displayedHint = hintExpanded ? promptHint : hintLines.slice(0, HINT_COLLAPSED_LINES).join('\n');

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-4xl mx-auto px-4 sm:px-8 py-6 sm:py-12">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-1.5 text-sm text-[var(--text-muted)] mb-6" data-qa="specialized-agent-breadcrumb">
          <Link to="/roles" className="hover:text-[var(--text-primary)] transition-colors">Agents</Link>
          <ChevronRight size={14} className="text-[var(--text-dim)]" />
          <Link to="/roles" className="hover:text-[var(--text-primary)] transition-colors">{parent.name}</Link>
          <ChevronRight size={14} className="text-[var(--text-dim)]" />
          <span className="text-[var(--text-primary)]">{specialized.name}</span>
        </nav>

        {/* Header */}
        <div className="flex items-start justify-between mb-2">
          <div>
            <h1
              className="text-[28px] font-semibold text-[var(--text-primary)]"
              data-qa="specialized-agent-title"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              {specialized.name}
            </h1>
            <p className="text-sm text-[var(--text-muted)] mt-1" style={{ fontFamily: 'Inter, sans-serif' }}>
              Specialized agent of {parent.name}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setEditOpen(true)}
              data-qa="specialized-agent-edit-btn"
              className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium border border-[var(--border-primary)] text-[var(--text-primary)] hover:bg-[var(--bg-secondary)] transition-colors"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              <Pencil size={14} />
              Edit
            </button>
            {confirmDelete ? (
              <div className="flex items-center gap-2">
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  data-qa="specialized-agent-delete-confirm-btn"
                  className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium bg-[#FF3B30] text-white hover:bg-[#FF3B30]/80 disabled:opacity-50 transition-colors"
                >
                  <Trash2 size={14} />
                  {deleting ? 'Deleting...' : 'Confirm'}
                </button>
                <button
                  onClick={() => setConfirmDelete(false)}
                  className="px-3 py-2 rounded-md text-sm text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <button
                onClick={() => setConfirmDelete(true)}
                data-qa="specialized-agent-delete-btn"
                className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium border border-[#FF3B30]/30 text-[#F06060] hover:bg-[#FF3B30]/10 transition-colors"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                <Trash2 size={14} />
                Delete
              </button>
            )}
          </div>
        </div>

        {/* Divider */}
        <div className="border-t border-[var(--border-primary)] my-6" />

        {/* Description */}
        <div className="mb-6">
          <label className="text-xs font-mono text-[var(--text-dim)] mb-2 block">Description</label>
          {parent.description ? (
            <p className="text-sm text-[var(--text-muted)] leading-relaxed">{parent.description}</p>
          ) : (
            <p className="text-sm text-[var(--text-dim)] italic">No description (inherited from parent agent)</p>
          )}
        </div>

        {/* Prompt Hint */}
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-2">
            <label className="text-xs font-mono text-[var(--text-dim)]">Prompt Hint</label>
            {promptHint && (
              <span className="text-[10px] font-mono text-[var(--text-dim)]">
                {hintLines.length} line{hintLines.length !== 1 ? 's' : ''}
              </span>
            )}
          </div>
          {promptHint ? (
            <div>
              <div className="rounded-md bg-[#0D0D0D] border border-[#1A1A1A] px-4 py-3">
                <pre className="text-xs font-mono text-[var(--text-muted)] whitespace-pre-wrap break-words leading-relaxed">
                  {displayedHint}
                </pre>
              </div>
              {showToggle && (
                <button
                  onClick={() => setHintExpanded(!hintExpanded)}
                  className="flex items-center gap-1 mt-2 text-xs text-[var(--primary)] hover:text-[var(--primary)]/80 transition-colors"
                >
                  {hintExpanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                  {hintExpanded ? 'Show less' : 'Show more'}
                </button>
              )}
            </div>
          ) : (
            <p className="text-sm text-[var(--text-dim)] italic">No prompt hint (inherited from parent agent)</p>
          )}
        </div>

        {/* Divider */}
        <div className="border-t border-[var(--border-primary)] my-6" />

        {/* Skills */}
        <div>
          <h2 className="text-lg font-semibold text-[var(--text-primary)] mb-4" style={{ fontFamily: 'Inter, sans-serif' }}>
            Skills
          </h2>
          {skills.length === 0 ? (
            <p className="text-sm text-[var(--text-muted)] italic">No skills assigned to this specialized agent.</p>
          ) : (
            <div className="space-y-2">
              {skills.map(skill => (
                <div
                  key={skill.slug}
                  data-qa="specialized-agent-skill-item"
                  className="flex items-center gap-3 px-4 py-3 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)]"
                >
                  <BookOpen size={16} className="text-[var(--text-dim)] shrink-0" />
                  <span className="text-sm text-[var(--text-primary)]">{skill.name}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Edit Dialog */}
      {editOpen && (
        <EditSpecializedAgentDialog
          parentSlug={parentSlug!}
          specializedAgent={specialized}
          onClose={() => setEditOpen(false)}
          onSaved={handleEditSaved}
        />
      )}
    </div>
  );
}
