import { useEffect, useCallback, useState, useRef, useMemo } from 'react';
import { X, FileCode2, Tag, ArrowRight, ArrowLeft, Pencil, Check, XCircle, Paperclip, Loader2, Plus } from 'lucide-react';
import type { TaskWithDetailsResponse, TaskResponse, ColumnWithTasksResponse, RoleResponse, FeatureResponse } from '../../lib/types';
import { getTask, listDependencies, listDependents, updateTask, addDependency, removeDependency, listTasks, listProjectRoles } from '../../lib/api';
import BlockedBanner from './BlockedBanner';
import CommentSection from './CommentSection';
import MarkdownContent from '../ui/MarkdownContent';
import { useImageUpload } from '../../hooks/useImageUpload';
import { formatDuration } from '../../lib/utils';

const STATUS_STYLES: Record<string, { label: string; className: string }> = {
  done: { label: 'done', className: 'bg-[var(--status-done-bg)] text-[var(--status-done)]' },
  in_progress: { label: 'in progress', className: 'bg-[var(--status-progress-bg)] text-[var(--status-progress)]' },
  blocked: { label: 'blocked', className: 'bg-[var(--status-blocked-bg)] text-[var(--status-blocked)]' },
  wont_do: { label: "won't do", className: 'bg-[var(--status-blocked-bg)] text-[var(--status-blocked)]' },
  todo: { label: 'todo', className: 'bg-[var(--status-todo-bg)] text-[var(--status-todo)]' },
};

function getDepStatus(dep: TaskResponse, columnSlugById: Record<string, string>): { label: string; className: string } {
  const slug = columnSlugById[dep.column_id] ?? '';
  if (dep.wont_do_requested && slug === 'done') return STATUS_STYLES.wont_do;
  if (slug === 'done') return STATUS_STYLES.done;
  if (slug === 'blocked') return STATUS_STYLES.blocked;
  if (slug === 'in_progress') return STATUS_STYLES.in_progress;
  return STATUS_STYLES.todo;
}

interface TaskDrawerProps {
  projectId: string;
  taskId: string;
  columns: ColumnWithTasksResponse[];
  features?: FeatureResponse[];
  onClose: () => void;
  onAction: (action: string) => void;
  onTaskNavigate?: (taskId: string) => void;
}

const priorityStyles: Record<string, { text: string; bg: string }> = {
  critical: { text: 'var(--priority-critical)', bg: 'var(--priority-critical-bg)' },
  high: { text: 'var(--priority-high)', bg: 'var(--priority-high-bg)' },
  medium: { text: 'var(--priority-medium)', bg: 'var(--priority-medium-bg)' },
  low: { text: 'var(--priority-low)', bg: 'var(--priority-low-bg)' },
};

const columnStatusStyles: Record<string, { text: string; bg: string }> = {
  todo: { text: 'var(--status-todo)', bg: 'var(--status-todo-bg)' },
  in_progress: { text: 'var(--status-progress)', bg: 'var(--status-progress-bg)' },
  done: { text: 'var(--status-done)', bg: 'var(--status-done-bg)' },
  blocked: { text: 'var(--status-blocked)', bg: 'var(--status-blocked-bg)' },
};

const EFFORT_OPTIONS = ['XS', 'S', 'M', 'L', 'XL'];

function columnSlugForTask(taskId: string, columns: ColumnWithTasksResponse[]): string {
  for (const col of columns) {
    if (col.tasks?.some((t) => t.id === taskId)) return col.slug;
  }
  return 'todo';
}

function columnNameForTask(taskId: string, columns: ColumnWithTasksResponse[]): string {
  for (const col of columns) {
    if (col.tasks?.some((t) => t.id === taskId)) return col.name;
  }
  return 'Unknown';
}

export default function TaskDrawer({ projectId, taskId, columns, features, onClose, onAction, onTaskNavigate }: TaskDrawerProps) {
  const [task, setTask] = useState<TaskWithDetailsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [commentRefreshKey] = useState(0);
  const [dependencies, setDependencies] = useState<TaskResponse[]>([]);
  const [dependents, setDependents] = useState<TaskResponse[]>([]);
  const [roles, setRoles] = useState<RoleResponse[]>([]);

  const columnSlugById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const col of columns) map[col.id] = col.slug;
    return map;
  }, [columns]);

  // Dependency management state
  const [depSearchOpen, setDepSearchOpen] = useState(false);
  const [depSearchQuery, setDepSearchQuery] = useState('');
  const [depSearchResults, setDepSearchResults] = useState<TaskResponse[]>([]);
  const [depSearchLoading, setDepSearchLoading] = useState(false);
  const [depRemoving, setDepRemoving] = useState<string | null>(null);
  const [depAdding, setDepAdding] = useState(false);
  const depSearchRef = useRef<HTMLDivElement>(null);
  const depSearchInputRef = useRef<HTMLInputElement>(null);
  const depSearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Title edit state
  const [titleEditMode, setTitleEditMode] = useState(false);
  const [titleDraft, setTitleDraft] = useState('');
  const [titleSaving, setTitleSaving] = useState(false);
  const titleInputRef = useRef<HTMLInputElement>(null);

  // Summary edit state
  const [summaryEditMode, setSummaryEditMode] = useState(false);
  const [summaryDraft, setSummaryDraft] = useState('');
  const [summarySaving, setSummarySaving] = useState(false);
  const [summarySaveError, setSummarySaveError] = useState<string | null>(null);

  // Priority edit state
  const [priorityOpen, setPriorityOpen] = useState(false);
  const priorityRef = useRef<HTMLDivElement>(null);

  // Assigned role edit state
  const [roleOpen, setRoleOpen] = useState(false);
  const roleRef = useRef<HTMLDivElement>(null);

  // Estimated effort edit state
  const [effortOpen, setEffortOpen] = useState(false);
  const effortRef = useRef<HTMLDivElement>(null);

  // Tags input state
  const [tagInput, setTagInput] = useState('');

  // Context files input state
  const [contextFileInput, setContextFileInput] = useState('');

  // Description edit state
  const [descEditMode, setDescEditMode] = useState(false);
  const [descDraft, setDescDraft] = useState('');
  const [descSaving, setDescSaving] = useState(false);
  const [descSaveError, setDescSaveError] = useState<string | null>(null);
  const [descDragOver, setDescDragOver] = useState(false);
  const descTextareaRef = useRef<HTMLTextAreaElement>(null);
  const descFileInputRef = useRef<HTMLInputElement>(null);
  const { upload: uploadImg, uploading: imgUploading, error: imgError } = useImageUpload(projectId);

  // Resolution edit state
  const [resEditMode, setResEditMode] = useState(false);
  const [resDraft, setResDraft] = useState('');
  const [resSaving, setResSaving] = useState(false);
  const [resSaveError, setResSaveError] = useState<string | null>(null);

  const fetchTask = async () => {
    try {
      const data = await getTask(projectId, taskId);
      setTask(data);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  };

  const fetchDependencies = async () => {
    try {
      const data = await listDependencies(projectId, taskId);
      setDependencies(data ?? []);
    } catch {
      setDependencies([]);
    }
  };

  const fetchDependents = async () => {
    try {
      const data = await listDependents(projectId, taskId);
      setDependents(data ?? []);
    } catch {
      // endpoint may not exist yet — silently ignore
      setDependents([]);
    }
  };

  const fetchRoles = async () => {
    try {
      const data = await listProjectRoles(projectId);
      setRoles(data ?? []);
    } catch {
      setRoles([]);
    }
  };

  const handleRemoveDependency = async (depTaskId: string) => {
    setDepRemoving(depTaskId);
    try {
      await removeDependency(projectId, taskId, depTaskId);
      await fetchDependencies();
    } catch {
      /* ignore */
    } finally {
      setDepRemoving(null);
    }
  };

  const handleDepSearchChange = (query: string) => {
    setDepSearchQuery(query);
    if (depSearchTimerRef.current) clearTimeout(depSearchTimerRef.current);
    if (!query.trim()) {
      setDepSearchResults([]);
      return;
    }
    depSearchTimerRef.current = setTimeout(async () => {
      setDepSearchLoading(true);
      try {
        const results = await listTasks(projectId, { search: query });
        // Filter out current task and already-added dependencies
        const depIds = new Set(dependencies.map((d) => d.id));
        setDepSearchResults(
          (results ?? []).filter((t) => t.id !== taskId && !depIds.has(t.id))
        );
      } catch {
        setDepSearchResults([]);
      } finally {
        setDepSearchLoading(false);
      }
    }, 300);
  };

  const handleSelectDep = async (selectedTask: TaskResponse) => {
    setDepAdding(true);
    try {
      await addDependency(projectId, taskId, { depends_on_task_id: selectedTask.id });
      await fetchDependencies();
      setDepSearchQuery('');
      setDepSearchResults([]);
      setDepSearchOpen(false);
    } catch {
      /* ignore */
    } finally {
      setDepAdding(false);
    }
  };

  const openDepSearch = () => {
    setDepSearchOpen(true);
    setDepSearchQuery('');
    setDepSearchResults([]);
    setTimeout(() => depSearchInputRef.current?.focus(), 0);
  };

  const closeDepSearch = () => {
    setDepSearchOpen(false);
    setDepSearchQuery('');
    setDepSearchResults([]);
  };

  useEffect(() => {
    setLoading(true);
    setDependencies([]);
    setDependents([]);
    fetchTask();
    fetchDependencies();
    fetchDependents();
    fetchRoles();
  }, [projectId, taskId]);

  // Focus title input when entering edit mode
  useEffect(() => {
    if (titleEditMode && titleInputRef.current) {
      titleInputRef.current.focus();
      titleInputRef.current.select();
    }
  }, [titleEditMode]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (titleEditMode) {
          setTitleEditMode(false);
        } else if (summaryEditMode) {
          setSummaryEditMode(false);
          setSummarySaveError(null);
        } else if (descEditMode) {
          setDescEditMode(false);
        } else if (resEditMode) {
          setResEditMode(false);
        } else {
          onClose();
        }
      }
    },
    [onClose, titleEditMode, summaryEditMode, descEditMode, resEditMode],
  );

  // Title handlers
  const startTitleEdit = () => {
    setTitleDraft(task?.title ?? '');
    setTitleEditMode(true);
  };

  const cancelTitleEdit = () => {
    setTitleEditMode(false);
  };

  const saveTitle = async () => {
    if (!task || !titleDraft.trim() || titleDraft.trim() === task.title) {
      setTitleEditMode(false);
      return;
    }
    setTitleSaving(true);
    try {
      const updated = await updateTask(projectId, task.id, { title: titleDraft.trim() });
      setTask((prev) => prev ? { ...prev, title: updated.title } : prev);
      setTitleEditMode(false);
    } catch {
      /* ignore */
    } finally {
      setTitleSaving(false);
    }
  };

  const handleTitleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveTitle();
    } else if (e.key === 'Escape') {
      cancelTitleEdit();
    }
  };

  // Summary handlers
  const startSummaryEdit = () => {
    setSummaryDraft(task?.summary ?? '');
    setSummarySaveError(null);
    setSummaryEditMode(true);
  };

  const cancelSummaryEdit = () => {
    setSummaryEditMode(false);
    setSummarySaveError(null);
  };

  const saveSummary = async () => {
    if (!task) return;
    setSummarySaving(true);
    setSummarySaveError(null);
    try {
      const updated = await updateTask(projectId, task.id, { summary: summaryDraft });
      setTask((prev) => prev ? { ...prev, summary: updated.summary } : prev);
      setSummaryEditMode(false);
    } catch (err) {
      setSummarySaveError(err instanceof Error ? err.message : 'Failed to save summary');
    } finally {
      setSummarySaving(false);
    }
  };

  // Role handler
  const handleRoleChange = async (newRole: string) => {
    if (!task) return;
    try {
      const updated = await updateTask(projectId, task.id, { assigned_role: newRole });
      setTask((prev) => prev ? { ...prev, assigned_role: updated.assigned_role } : prev);
    } catch {
      /* ignore */
    }
    setRoleOpen(false);
  };

  // Effort handler
  const handleEffortChange = async (newEffort: string) => {
    if (!task || newEffort === task.estimated_effort) {
      setEffortOpen(false);
      return;
    }
    try {
      const updated = await updateTask(projectId, task.id, { estimated_effort: newEffort });
      setTask((prev) => prev ? { ...prev, estimated_effort: updated.estimated_effort } : prev);
    } catch {
      /* ignore */
    }
    setEffortOpen(false);
  };

  // Tags handlers
  const addTag = async () => {
    const trimmed = tagInput.trim();
    if (!task || !trimmed) return;
    if (task.tags?.includes(trimmed)) {
      setTagInput('');
      return;
    }
    const newTags = [...(task.tags ?? []), trimmed];
    try {
      const updated = await updateTask(projectId, task.id, { tags: newTags });
      setTask((prev) => prev ? { ...prev, tags: updated.tags } : prev);
      setTagInput('');
    } catch {
      /* ignore */
    }
  };

  const removeTag = async (tag: string) => {
    if (!task) return;
    const newTags = (task.tags ?? []).filter((t) => t !== tag);
    try {
      const updated = await updateTask(projectId, task.id, { tags: newTags });
      setTask((prev) => prev ? { ...prev, tags: updated.tags } : prev);
    } catch {
      /* ignore */
    }
  };

  const handleTagKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag();
    }
  };

  // Context files handlers
  const addContextFile = async () => {
    const trimmed = contextFileInput.trim();
    if (!task || !trimmed) return;
    if (task.context_files?.includes(trimmed)) {
      setContextFileInput('');
      return;
    }
    const newFiles = [...(task.context_files ?? []), trimmed];
    try {
      const updated = await updateTask(projectId, task.id, { context_files: newFiles });
      setTask((prev) => prev ? { ...prev, context_files: updated.context_files } : prev);
      setContextFileInput('');
    } catch {
      /* ignore */
    }
  };

  const removeContextFile = async (file: string) => {
    if (!task) return;
    const newFiles = (task.context_files ?? []).filter((f) => f !== file);
    try {
      const updated = await updateTask(projectId, task.id, { context_files: newFiles });
      setTask((prev) => prev ? { ...prev, context_files: updated.context_files } : prev);
    } catch {
      /* ignore */
    }
  };

  const handleContextFileKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addContextFile();
    }
  };

  const startDescEdit = () => {
    setDescDraft(task?.description ?? '');
    setDescSaveError(null);
    setDescEditMode(true);
  };

  const cancelDescEdit = () => {
    setDescEditMode(false);
    setDescSaveError(null);
  };

  const saveDesc = async () => {
    if (!task) return;
    setDescSaving(true);
    setDescSaveError(null);
    try {
      const updated = await updateTask(projectId, task.id, { description: descDraft });
      setTask((prev) => prev ? { ...prev, description: updated.description } : prev);
      setDescEditMode(false);
    } catch (err) {
      setDescSaveError(err instanceof Error ? err.message : 'Failed to save description');
    } finally {
      setDescSaving(false);
    }
  };

  const startResEdit = () => {
    setResDraft(task?.resolution ?? '');
    setResSaveError(null);
    setResEditMode(true);
  };

  const cancelResEdit = () => {
    setResEditMode(false);
    setResSaveError(null);
  };

  const saveRes = async () => {
    if (!task) return;
    setResSaving(true);
    setResSaveError(null);
    try {
      const updated = await updateTask(projectId, task.id, { resolution: resDraft });
      setTask((prev) => prev ? { ...prev, resolution: updated.resolution } : prev);
      setResEditMode(false);
    } catch (err) {
      setResSaveError(err instanceof Error ? err.message : 'Failed to save resolution');
    } finally {
      setResSaving(false);
    }
  };

  const handlePriorityChange = async (newPriority: string) => {
    if (!task || newPriority === task.priority) {
      setPriorityOpen(false);
      return;
    }
    try {
      const updated = await updateTask(projectId, task.id, { priority: newPriority });
      setTask((prev) => prev ? { ...prev, priority: updated.priority, priority_score: updated.priority_score } : prev);
    } catch {
      /* ignore */
    }
    setPriorityOpen(false);
  };

  // Close priority dropdown on outside click
  useEffect(() => {
    if (!priorityOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (priorityRef.current && !priorityRef.current.contains(e.target as Node)) {
        setPriorityOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [priorityOpen]);

  // Close role dropdown on outside click
  useEffect(() => {
    if (!roleOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (roleRef.current && !roleRef.current.contains(e.target as Node)) {
        setRoleOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [roleOpen]);

  // Close effort dropdown on outside click
  useEffect(() => {
    if (!effortOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (effortRef.current && !effortRef.current.contains(e.target as Node)) {
        setEffortOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [effortOpen]);

  // Close dep search dropdown on outside click
  useEffect(() => {
    if (!depSearchOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (depSearchRef.current && !depSearchRef.current.contains(e.target as Node)) {
        closeDepSearch();
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [depSearchOpen]);

  const insertImageMarkdown = useCallback(
    async (file: File) => {
      const url = await uploadImg(file);
      if (!url) return;
      const markdown = `![](${url})`;
      const textarea = descTextareaRef.current;
      if (textarea) {
        const start = textarea.selectionStart ?? descDraft.length;
        const end = textarea.selectionEnd ?? descDraft.length;
        const newDraft = descDraft.slice(0, start) + markdown + descDraft.slice(end);
        setDescDraft(newDraft);
      } else {
        setDescDraft((prev) => prev + markdown);
      }
    },
    [uploadImg, descDraft],
  );

  const handleDescDragOver = (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDescDragOver(true);
  };

  const handleDescDragLeave = () => {
    setDescDragOver(false);
  };

  const handleDescDrop = async (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setDescDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) {
      await insertImageMarkdown(file);
    }
  };

  const handleDescFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      await insertImageMarkdown(file);
    }
    e.target.value = '';
  };

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  const colSlug = columnSlugForTask(taskId, columns);
  const colName = columnNameForTask(taskId, columns);
  const statusStyle = columnStatusStyles[colSlug] || { text: 'var(--text-muted)', bg: 'var(--status-todo-bg)' };

  return (
    <>
      {/* Overlay */}
      <div className="fixed inset-0 z-40 bg-[rgba(0,0,0,0.5)]" onClick={onClose} />

      {/* Drawer */}
      <div className="fixed top-0 right-0 z-50 h-full w-[1160px] max-w-full bg-[var(--card-bg)] border-l border-[var(--border-primary)] shadow-2xl flex flex-col overflow-hidden animate-slide-in">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)] bg-[var(--bg-secondary)] flex-shrink-0">
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
            >
              <X size={18} />
            </button>
            <span className="text-[var(--text-muted)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
              Task Detail
            </span>
          </div>
          <div
            className="px-2 py-0.5 rounded text-[10px] font-['JetBrains_Mono'] font-bold uppercase tracking-wider"
            style={{ color: statusStyle.text, backgroundColor: statusStyle.bg }}
          >
            {colName}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="w-6 h-6 border-2 border-[var(--primary)] border-t-transparent rounded-full animate-spin" />
            </div>
          ) : task ? (
            <>
              {/* Title */}
              <div>
                {titleEditMode ? (
                  <div className="flex items-center gap-2 mb-2">
                    <input
                      ref={titleInputRef}
                      value={titleDraft}
                      onChange={(e) => setTitleDraft(e.target.value)}
                      onKeyDown={handleTitleKeyDown}
                      onBlur={saveTitle}
                      disabled={titleSaving}
                      className="flex-1 bg-[var(--bg-secondary)] border border-[var(--primary)] rounded-md px-3 py-1.5 text-[var(--text-primary)] text-xl font-['Newsreader'] font-medium focus:outline-none disabled:opacity-40"
                    />
                    {titleSaving && <Loader2 size={14} className="text-[var(--primary)] animate-spin flex-shrink-0" />}
                  </div>
                ) : (
                  <button
                    onClick={startTitleEdit}
                    className="group w-full text-left mb-2"
                    title="Click to edit title"
                  >
                    <h2 className="text-[var(--text-primary)] text-xl font-['Newsreader'] font-medium leading-snug group-hover:text-[var(--primary)] transition-colors">
                      {task.title}
                    </h2>
                  </button>
                )}
                <div className="flex items-center gap-3 flex-wrap">
                  {/* Priority */}
                  {task.priority && (
                    <div className="relative" ref={priorityRef}>
                      <button
                        onClick={() => setPriorityOpen((v) => !v)}
                        className="px-2 py-0.5 rounded text-[10px] font-['JetBrains_Mono'] font-bold uppercase cursor-pointer hover:opacity-80 transition-opacity"
                        style={{
                          color: priorityStyles[task.priority]?.text || 'var(--text-muted)',
                          backgroundColor: priorityStyles[task.priority]?.bg || 'var(--status-todo-bg)',
                        }}
                      >
                        {task.priority}
                      </button>
                      {priorityOpen && (
                        <div className="absolute top-full left-0 mt-1 z-10 bg-[var(--card-bg)] border border-[var(--border-primary)] rounded-md shadow-lg py-1 min-w-[120px]">
                          {(['critical', 'high', 'medium', 'low'] as const).map((p) => (
                            <button
                              key={p}
                              onClick={() => handlePriorityChange(p)}
                              className={`w-full text-left px-3 py-1.5 text-[11px] font-['JetBrains_Mono'] font-bold uppercase flex items-center gap-2 transition-colors cursor-pointer ${
                                p === task.priority ? 'bg-[var(--bg-elevated)]' : 'hover:bg-[var(--bg-secondary)]'
                              }`}
                            >
                              <span
                                className="w-2 h-2 rounded-full flex-shrink-0"
                                style={{ backgroundColor: priorityStyles[p]?.text || 'var(--text-muted)' }}
                              />
                              <span style={{ color: priorityStyles[p]?.text || 'var(--text-muted)' }}>
                                {p}
                              </span>
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                  {/* Assigned role */}
                  <div className="relative" ref={roleRef}>
                    <button
                      onClick={() => setRoleOpen((v) => !v)}
                      className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] hover:text-[var(--primary)] transition-colors cursor-pointer"
                      title="Click to change assigned role"
                    >
                      {task.assigned_role ? `@${task.assigned_role}` : (
                        <span className="text-[var(--text-dim)] italic">assign role</span>
                      )}
                    </button>
                    {roleOpen && (
                      <div className="absolute top-full left-0 mt-1 z-10 bg-[var(--card-bg)] border border-[var(--border-primary)] rounded-md shadow-lg py-1 min-w-[160px] max-h-60 overflow-y-auto">
                        <button
                          onClick={() => handleRoleChange('')}
                          className="w-full text-left px-3 py-1.5 text-[11px] font-['JetBrains_Mono'] text-[var(--text-dim)] italic hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer"
                        >
                          unassign
                        </button>
                        {roles.map((role) => (
                          <button
                            key={role.slug}
                            onClick={() => handleRoleChange(role.slug)}
                            className={`w-full text-left px-3 py-1.5 text-[11px] font-['JetBrains_Mono'] flex items-center gap-2 transition-colors cursor-pointer ${
                              role.slug === task.assigned_role ? 'bg-[var(--bg-elevated)] text-[var(--primary)]' : 'text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)]'
                            }`}
                          >
                            {role.icon && <span>{role.icon}</span>}
                            @{role.slug}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                  {/* Estimated effort */}
                  <div className="relative" ref={effortRef}>
                    <button
                      onClick={() => setEffortOpen((v) => !v)}
                      className="text-[var(--text-dim)] text-xs font-['JetBrains_Mono'] hover:text-[var(--primary)] transition-colors cursor-pointer"
                      title="Click to change estimated effort"
                    >
                      {task.estimated_effort || <span className="italic">effort?</span>}
                    </button>
                    {effortOpen && (
                      <div className="absolute top-full left-0 mt-1 z-10 bg-[var(--card-bg)] border border-[var(--border-primary)] rounded-md shadow-lg py-1 min-w-[100px]">
                        <button
                          onClick={() => handleEffortChange('')}
                          className="w-full text-left px-3 py-1.5 text-[11px] font-['JetBrains_Mono'] text-[var(--text-dim)] italic hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer"
                        >
                          clear
                        </button>
                        {EFFORT_OPTIONS.map((opt) => (
                          <button
                            key={opt}
                            onClick={() => handleEffortChange(opt)}
                            className={`w-full text-left px-3 py-1.5 text-[11px] font-['JetBrains_Mono'] font-bold transition-colors cursor-pointer ${
                              opt === task.estimated_effort
                                ? 'bg-[var(--bg-elevated)] text-[var(--primary)]'
                                : 'text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)]'
                            }`}
                          >
                            {opt}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                  {/* Unresolved deps */}
                  {task.has_unresolved_deps && (
                    <span className="text-[var(--status-progress)] text-[10px] font-['JetBrains_Mono'] uppercase px-1.5 py-0.5 bg-[var(--status-progress-bg)] rounded">
                      has deps
                    </span>
                  )}
                </div>
                {/* Feature badge */}
                {task.feature_id && (
                  <div className="mt-2">
                    <label className="block text-xs font-mono text-[var(--text-dim)] mb-1">Feature</label>
                    <span className="inline-flex items-center gap-1.5 px-2 py-0.5 bg-[#00C896]/10 border border-[#00C896]/20 rounded text-xs text-[#00C896]">
                      {features?.find((f) => f.id === task.feature_id)?.name ?? task.feature_id}
                    </span>
                  </div>
                )}
              </div>

              {/* Tags */}
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <Tag size={12} className="text-[var(--text-dim)]" />
                  <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">Tags</span>
                </div>
                <div className="flex items-center gap-2 flex-wrap">
                  {(task.tags ?? []).map((tag) => (
                    <span
                      key={tag}
                      className="flex items-center gap-1 px-2 py-0.5 rounded bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-[10px] font-['JetBrains_Mono']"
                    >
                      {tag}
                      <button
                        onClick={() => removeTag(tag)}
                        className="text-[var(--text-dim)] hover:text-[var(--status-blocked)] transition-colors ml-0.5"
                        title="Remove tag"
                      >
                        <X size={10} />
                      </button>
                    </span>
                  ))}
                  <div className="flex items-center gap-1">
                    <input
                      value={tagInput}
                      onChange={(e) => setTagInput(e.target.value)}
                      onKeyDown={handleTagKeyDown}
                      placeholder="add tag…"
                      className="bg-transparent border-b border-[var(--border-primary)] focus:border-[var(--primary)] text-[10px] font-['JetBrains_Mono'] text-[var(--text-secondary)] placeholder-[var(--text-dim)] focus:outline-none w-20 pb-0.5 transition-colors"
                    />
                    <button
                      onClick={addTag}
                      className="text-[var(--text-dim)] hover:text-[var(--primary)] transition-colors"
                      title="Add tag"
                    >
                      <Plus size={11} />
                    </button>
                  </div>
                </div>
              </div>

              {/* Dependencies */}
              <div>
                <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                  Dependencies
                </h3>
                <div className="flex flex-col gap-1">
                  {dependencies.map((dep) => {
                    const url = `?task=${dep.id}`;
                    const status = getDepStatus(dep, columnSlugById);
                    const isRemoving = depRemoving === dep.id;
                    return (
                      <div
                        key={dep.id}
                        className="flex items-center gap-2 px-2 py-1.5 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--primary)] hover:bg-[var(--status-done-bg)] transition-colors group"
                      >
                        <ArrowRight size={11} className="text-[var(--primary)] flex-shrink-0" />
                        <a
                          href={url}
                          onClick={(e) => {
                            e.preventDefault();
                            onTaskNavigate?.(dep.id);
                          }}
                          className="text-[var(--text-secondary)] text-xs font-['Inter'] truncate group-hover:text-[var(--text-primary)] transition-colors flex-1 min-w-0"
                        >
                          {dep.title}
                        </a>
                        <span className={`text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded flex-shrink-0 ${status.className}`}>
                          {status.label}
                        </span>
                        <span className="text-[var(--text-dim)] text-[10px] font-['JetBrains_Mono'] flex-shrink-0">
                          depends on
                        </span>
                        <button
                          onClick={() => handleRemoveDependency(dep.id)}
                          disabled={isRemoving}
                          title="Remove dependency"
                          className="flex-shrink-0 opacity-0 group-hover:opacity-100 text-[var(--text-dim)] hover:text-[var(--status-blocked)] transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          {isRemoving ? (
                            <Loader2 size={12} className="animate-spin" />
                          ) : (
                            <X size={12} />
                          )}
                        </button>
                      </div>
                    );
                  })}
                  {dependents.map((dep) => {
                    const url = `?task=${dep.id}`;
                    const status = getDepStatus(dep, columnSlugById);
                    return (
                      <div
                        key={dep.id}
                        className="flex items-center gap-2 px-2 py-1.5 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--status-progress)] hover:bg-[var(--status-progress-bg)] transition-colors group"
                      >
                        <ArrowLeft size={11} className="text-[var(--status-progress)] flex-shrink-0" />
                        <a
                          href={url}
                          onClick={(e) => {
                            e.preventDefault();
                            onTaskNavigate?.(dep.id);
                          }}
                          className="text-[var(--text-secondary)] text-xs font-['Inter'] truncate group-hover:text-[var(--text-primary)] transition-colors flex-1 min-w-0"
                        >
                          {dep.title}
                        </a>
                        <span className={`text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded flex-shrink-0 ${status.className}`}>
                          {status.label}
                        </span>
                        <span className="text-[var(--text-dim)] text-[10px] font-['JetBrains_Mono'] flex-shrink-0">
                          needed by
                        </span>
                      </div>
                    );
                  })}

                  {/* Add dependency */}
                  {!depSearchOpen ? (
                    <button
                      onClick={openDepSearch}
                      className="flex items-center gap-1.5 px-2 py-1 text-[var(--text-dim)] hover:text-[var(--text-secondary)] text-xs font-['Inter'] transition-colors self-start mt-0.5"
                    >
                      <Plus size={12} />
                      Add dependency
                    </button>
                  ) : (
                    <div className="relative mt-1" ref={depSearchRef}>
                      <div className="flex items-center gap-2">
                        <input
                          ref={depSearchInputRef}
                          type="text"
                          value={depSearchQuery}
                          onChange={(e) => handleDepSearchChange(e.target.value)}
                          placeholder="Search tasks..."
                          className="flex-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] focus:border-[var(--primary)] rounded-md px-3 py-1.5 text-[var(--text-primary)] text-xs font-['Inter'] placeholder-[var(--text-dim)] focus:outline-none transition-colors"
                        />
                        <button
                          onClick={closeDepSearch}
                          className="text-[var(--text-dim)] hover:text-[var(--text-secondary)] transition-colors flex-shrink-0"
                        >
                          <X size={14} />
                        </button>
                      </div>

                      {/* Dropdown results */}
                      {(depSearchResults.length > 0 || depSearchLoading) && (
                        <div className="absolute top-full left-0 right-0 mt-1 z-20 bg-[var(--card-bg)] border border-[var(--border-primary)] rounded-md shadow-lg max-h-52 overflow-y-auto">
                          {depSearchLoading ? (
                            <div className="flex items-center justify-center py-4">
                              <Loader2 size={14} className="animate-spin text-[var(--text-dim)]" />
                            </div>
                          ) : (
                            depSearchResults.map((t) => {
                              const slugForTask = columnSlugById[t.column_id] ?? 'todo';
                              const colStyle = columnStatusStyles[slugForTask] || { text: 'var(--text-muted)', bg: 'var(--status-todo-bg)' };
                              const pri = priorityStyles[t.priority] || { text: 'var(--text-muted)', bg: 'var(--status-todo-bg)' };
                              return (
                                <button
                                  key={t.id}
                                  onClick={() => handleSelectDep(t)}
                                  disabled={depAdding}
                                  className="w-full flex items-center gap-2 px-3 py-2 hover:bg-[var(--bg-secondary)] transition-colors text-left disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                  <span className="text-[var(--text-secondary)] text-xs font-['Inter'] truncate flex-1 min-w-0">
                                    {t.title}
                                  </span>
                                  <span
                                    className="text-[10px] font-['JetBrains_Mono'] font-bold px-1.5 py-0.5 rounded flex-shrink-0"
                                    style={{ color: pri.text, backgroundColor: pri.bg }}
                                  >
                                    {t.priority}
                                  </span>
                                  <span
                                    className="text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded flex-shrink-0"
                                    style={{ color: colStyle.text, backgroundColor: colStyle.bg }}
                                  >
                                    {slugForTask.replace('_', ' ')}
                                  </span>
                                </button>
                              );
                            })
                          )}
                        </div>
                      )}

                      {/* No results message */}
                      {!depSearchLoading && depSearchQuery.trim() && depSearchResults.length === 0 && (
                        <div className="absolute top-full left-0 right-0 mt-1 z-20 bg-[var(--card-bg)] border border-[var(--border-primary)] rounded-md shadow-lg px-3 py-3">
                          <p className="text-[var(--text-dim)] text-xs font-['Inter'] text-center">No tasks found</p>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>

              {/* Blocked Banner */}
              {task.is_blocked && (
                <BlockedBanner
                  task={task}
                  onUnblock={() => onAction('unblock')}
                />
              )}

              {/* Summary */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
                    Summary
                  </h3>
                  {!summaryEditMode && (
                    <button
                      onClick={startSummaryEdit}
                      className="flex items-center gap-1 text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors text-xs font-['Inter']"
                    >
                      <Pencil size={12} />
                      Edit
                    </button>
                  )}
                </div>

                {summaryEditMode ? (
                  <div className="space-y-2">
                    <textarea
                      value={summaryDraft}
                      onChange={(e) => setSummaryDraft(e.target.value)}
                      rows={4}
                      className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] focus:border-[var(--primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none transition-colors"
                      placeholder="Brief summary of the task…"
                    />
                    {summarySaveError && (
                      <p className="text-[var(--status-blocked)] text-xs font-['Inter']">{summarySaveError}</p>
                    )}
                    <div className="flex items-center gap-2">
                      <button
                        onClick={saveSummary}
                        disabled={summarySaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
                      >
                        {summarySaving ? (
                          <Loader2 size={12} className="animate-spin" />
                        ) : (
                          <Check size={12} />
                        )}
                        Save
                      </button>
                      <button
                        onClick={cancelSummaryEdit}
                        disabled={summarySaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-40 transition-colors rounded-md"
                      >
                        <XCircle size={12} />
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : task.summary ? (
                  <MarkdownContent content={task.summary} />
                ) : (
                  <p className="text-[var(--text-dim)] text-sm font-['Inter'] italic">No summary</p>
                )}
              </div>

              {/* Description */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
                    Description
                  </h3>
                  {!descEditMode && (
                    <button
                      onClick={startDescEdit}
                      className="flex items-center gap-1 text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors text-xs font-['Inter']"
                    >
                      <Pencil size={12} />
                      Edit
                    </button>
                  )}
                </div>

                {descEditMode ? (
                  <div className="space-y-2">
                    <div className="flex items-center justify-end gap-1.5 mb-1">
                      {imgUploading && (
                        <Loader2 size={13} className="text-[var(--primary)] animate-spin" />
                      )}
                      <button
                        type="button"
                        onClick={() => descFileInputRef.current?.click()}
                        title="Attach image"
                        className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
                      >
                        <Paperclip size={14} />
                      </button>
                      <input
                        ref={descFileInputRef}
                        type="file"
                        accept="image/*"
                        className="hidden"
                        onChange={handleDescFileSelect}
                      />
                    </div>
                    <textarea
                      ref={descTextareaRef}
                      value={descDraft}
                      onChange={(e) => setDescDraft(e.target.value)}
                      onDragOver={handleDescDragOver}
                      onDragLeave={handleDescDragLeave}
                      onDrop={handleDescDrop}
                      rows={8}
                      className={`w-full bg-[var(--bg-secondary)] border rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none transition-colors ${descDragOver ? 'border-[var(--primary)]' : 'border-[var(--border-primary)] focus:border-[var(--primary)]'}`}
                      placeholder="Add a description..."
                    />
                    {imgError && (
                      <p className="text-[var(--status-blocked)] text-xs font-['Inter']">{imgError}</p>
                    )}
                    {descSaveError && (
                      <p className="text-[var(--status-blocked)] text-xs font-['Inter']">{descSaveError}</p>
                    )}
                    <div className="flex items-center gap-2">
                      <button
                        onClick={saveDesc}
                        disabled={descSaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
                      >
                        {descSaving ? (
                          <Loader2 size={12} className="animate-spin" />
                        ) : (
                          <Check size={12} />
                        )}
                        Save
                      </button>
                      <button
                        onClick={cancelDescEdit}
                        disabled={descSaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-40 transition-colors rounded-md"
                      >
                        <XCircle size={12} />
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : task.description ? (
                  <MarkdownContent content={task.description} />
                ) : (
                  <p className="text-[var(--text-dim)] text-sm font-['Inter'] italic">No description</p>
                )}
              </div>

              {/* Resolution */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider">
                    Resolution
                  </h3>
                  {!resEditMode && (
                    <button
                      onClick={startResEdit}
                      className="flex items-center gap-1 text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors text-xs font-['Inter']"
                    >
                      <Pencil size={12} />
                      {task.resolution ? 'Edit' : 'Add'}
                    </button>
                  )}
                </div>

                {resEditMode ? (
                  <div className="space-y-2">
                    <textarea
                      value={resDraft}
                      onChange={(e) => setResDraft(e.target.value)}
                      rows={6}
                      className="w-full bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md px-3 py-2 text-[var(--text-primary)] text-sm font-['Inter'] placeholder-[var(--text-dim)] resize-y focus:outline-none focus:border-[var(--primary)] transition-colors"
                      placeholder="Add a resolution..."
                    />
                    {resSaveError && (
                      <p className="text-[var(--status-blocked)] text-xs font-['Inter']">{resSaveError}</p>
                    )}
                    <div className="flex items-center gap-2">
                      <button
                        onClick={saveRes}
                        disabled={resSaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] font-medium text-[var(--primary-text)] bg-[var(--primary)] hover:bg-[var(--primary-hover)] disabled:opacity-40 disabled:cursor-not-allowed rounded-md transition-colors"
                      >
                        {resSaving ? (
                          <Loader2 size={12} className="animate-spin" />
                        ) : (
                          <Check size={12} />
                        )}
                        Save
                      </button>
                      <button
                        onClick={cancelResEdit}
                        disabled={resSaving}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-['Inter'] text-[var(--text-secondary)] hover:text-[var(--text-primary)] disabled:opacity-40 transition-colors rounded-md"
                      >
                        <XCircle size={12} />
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : task.resolution ? (
                  <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
                    <MarkdownContent content={task.resolution} />
                  </div>
                ) : (
                  <p className="text-[var(--text-dim)] text-sm font-['Inter'] italic">No resolution</p>
                )}
              </div>

              {/* Completion */}
              {task.completion_summary && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Completion Summary
                  </h3>
                  <div className="bg-[var(--status-done-bg)] border border-[var(--border-subtle)] rounded-md p-3">
                    <MarkdownContent content={task.completion_summary} linkColor="var(--primary)" />
                    {task.completed_by_agent && (
                      <p className="text-[var(--text-dim)] text-xs font-['Inter'] mt-2">
                        Completed by {task.completed_by_agent}
                        {task.completed_at &&
                          ` on ${new Date(task.completed_at).toLocaleDateString()}`}
                      </p>
                    )}
                  </div>
                </div>
              )}

              {/* Context Files */}
              <div>
                <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                  Context Files
                </h3>
                <div className="flex flex-col gap-1 mb-2">
                  {(task.context_files ?? []).map((f) => (
                    <div
                      key={f}
                      className="flex items-center gap-2 px-2 py-1 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)] group"
                    >
                      <FileCode2 size={12} className="text-[var(--text-dim)] flex-shrink-0" />
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] truncate flex-1">
                        {f}
                      </span>
                      <button
                        onClick={() => removeContextFile(f)}
                        className="text-[var(--text-dim)] hover:text-[var(--status-blocked)] transition-colors opacity-0 group-hover:opacity-100 flex-shrink-0"
                        title="Remove file"
                      >
                        <X size={12} />
                      </button>
                    </div>
                  ))}
                </div>
                <div className="flex items-center gap-2">
                  <input
                    value={contextFileInput}
                    onChange={(e) => setContextFileInput(e.target.value)}
                    onKeyDown={handleContextFileKeyDown}
                    placeholder="add file path…"
                    className="flex-1 bg-[var(--bg-secondary)] border border-[var(--border-primary)] focus:border-[var(--primary)] rounded-md px-2 py-1 text-xs font-['JetBrains_Mono'] text-[var(--text-secondary)] placeholder-[var(--text-dim)] focus:outline-none transition-colors"
                  />
                  <button
                    onClick={addContextFile}
                    className="text-[var(--text-dim)] hover:text-[var(--primary)] transition-colors flex-shrink-0"
                    title="Add file"
                  >
                    <Plus size={14} />
                  </button>
                </div>
              </div>

              {/* Files Modified */}
              {task.files_modified && task.files_modified.length > 0 && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Files Modified
                  </h3>
                  <div className="flex flex-col gap-1">
                    {task.files_modified.map((f) => (
                      <div
                        key={f}
                        className="flex items-center gap-2 px-2 py-1 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)]"
                      >
                        <FileCode2 size={12} className="text-[var(--primary)] flex-shrink-0" />
                        <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] truncate">
                          {f}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Token Usage */}
              {(task.input_tokens > 0 || task.output_tokens > 0) && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Token Usage
                  </h3>
                  <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3 flex flex-wrap gap-x-5 gap-y-1.5">
                    {task.model && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        model: <span className="text-[var(--text-primary)]">{task.model}</span>
                      </span>
                    )}
                    {task.input_tokens > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        in: <span className="text-[var(--text-primary)]">{task.input_tokens.toLocaleString()}</span>
                      </span>
                    )}
                    {task.output_tokens > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        out: <span className="text-[var(--text-primary)]">{task.output_tokens.toLocaleString()}</span>
                      </span>
                    )}
                    {task.cache_read_tokens > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        cache read: <span className="text-[var(--text-primary)]">{task.cache_read_tokens.toLocaleString()}</span>
                      </span>
                    )}
                    {task.cache_write_tokens > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        cache write: <span className="text-[var(--text-primary)]">{task.cache_write_tokens.toLocaleString()}</span>
                      </span>
                    )}
                  </div>
                </div>
              )}

              {/* Timing */}
              {(task.started_at || task.duration_seconds > 0 || task.human_estimate_seconds > 0) && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Timing
                  </h3>
                  <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3 flex flex-wrap gap-x-5 gap-y-1.5">
                    <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                      started:{' '}
                      <span className="text-[var(--text-primary)]">
                        {task.started_at
                          ? new Date(task.started_at).toLocaleString()
                          : 'Not started'}
                      </span>
                    </span>
                    {task.duration_seconds > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        duration: <span className="text-[var(--text-primary)]">{formatDuration(task.duration_seconds)}</span>
                      </span>
                    )}
                    {task.human_estimate_seconds > 0 && (
                      <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                        human estimate: <span className="text-[var(--text-primary)]">{formatDuration(task.human_estimate_seconds)}</span>
                      </span>
                    )}
                    {task.duration_seconds > 0 && task.human_estimate_seconds > 0 && (() => {
                      const pct = ((task.human_estimate_seconds - task.duration_seconds) / task.human_estimate_seconds * 100).toFixed(0);
                      const isPositive = task.human_estimate_seconds > task.duration_seconds;
                      return (
                        <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                          time saved:{' '}
                          <span style={{ color: isPositive ? 'var(--status-done)' : 'var(--status-blocked)' }}>
                            {isPositive ? '+' : ''}{pct}%
                          </span>
                        </span>
                      );
                    })()}
                  </div>
                </div>
              )}

              {/* Meta info */}
              <div className="text-[var(--text-muted)] text-[10px] font-['Inter'] flex flex-col gap-0.5">
                {task.created_by_role && <span>Created by role: {task.created_by_role}</span>}
                {task.created_by_agent && <span>Created by agent: {task.created_by_agent}</span>}
                {task.session_id && <span>Session: <code className="font-mono text-[9px] bg-[var(--bg-secondary)] px-1 py-0.5 rounded break-all">{task.session_id}</code></span>}
                <span>Created: {new Date(task.created_at).toLocaleString()}</span>
                <span>Updated: {new Date(task.updated_at).toLocaleString()}</span>
              </div>

              {/* Comments */}
              <CommentSection
                projectId={projectId}
                taskId={taskId}
                refreshKey={commentRefreshKey}
              />
            </>
          ) : (
            <p className="text-[var(--text-muted)] text-sm font-['Inter'] text-center py-10">
              Task not found.
            </p>
          )}
        </div>
      </div>
    </>
  );
}
