import { useEffect, useCallback, useState, useRef, useMemo } from 'react';
import { X, FileCode2, Tag, ArrowRight, ArrowLeft, Pencil, Check, XCircle, Paperclip, Loader2 } from 'lucide-react';
import type { TaskWithDetailsResponse, TaskResponse, ColumnWithTasksResponse } from '../../lib/types';
import { getTask, listDependencies, listDependents, updateTask } from '../../lib/api';
import BlockedBanner from './BlockedBanner';
import CommentSection from './CommentSection';
import MarkdownContent from '../ui/MarkdownContent';
import { useImageUpload } from '../../hooks/useImageUpload';

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

export default function TaskDrawer({ projectId, taskId, columns, onClose, onAction, onTaskNavigate }: TaskDrawerProps) {
  const [task, setTask] = useState<TaskWithDetailsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [commentRefreshKey] = useState(0);
  const [dependencies, setDependencies] = useState<TaskResponse[]>([]);
  const [dependents, setDependents] = useState<TaskResponse[]>([]);

  const columnSlugById = useMemo(() => {
    const map: Record<string, string> = {};
    for (const col of columns) map[col.id] = col.slug;
    return map;
  }, [columns]);

  // Priority edit state
  const [priorityOpen, setPriorityOpen] = useState(false);
  const priorityRef = useRef<HTMLDivElement>(null);

  // Description edit state
  const [descEditMode, setDescEditMode] = useState(false);
  const [descDraft, setDescDraft] = useState('');
  const [descSaving, setDescSaving] = useState(false);
  const [descSaveError, setDescSaveError] = useState<string | null>(null);
  const [descDragOver, setDescDragOver] = useState(false);
  const descTextareaRef = useRef<HTMLTextAreaElement>(null);
  const descFileInputRef = useRef<HTMLInputElement>(null);
  const { upload: uploadImg, uploading: imgUploading, error: imgError } = useImageUpload(projectId);

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

  useEffect(() => {
    setLoading(true);
    setDependencies([]);
    setDependents([]);
    fetchTask();
    fetchDependencies();
    fetchDependents();
  }, [projectId, taskId]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (descEditMode) {
          setDescEditMode(false);
        } else {
          onClose();
        }
      }
    },
    [onClose, descEditMode],
  );

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
                <h2 className="text-[var(--text-primary)] text-xl font-['Newsreader'] font-medium leading-snug mb-2">
                  {task.title}
                </h2>
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
                  {task.assigned_role && (
                    <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono']">
                      @{task.assigned_role}
                    </span>
                  )}
                  {/* Estimated effort */}
                  {task.estimated_effort && (
                    <span className="text-[var(--text-dim)] text-xs font-['JetBrains_Mono']">
                      {task.estimated_effort}
                    </span>
                  )}
                  {/* Unresolved deps */}
                  {task.has_unresolved_deps && (
                    <span className="text-[var(--status-progress)] text-[10px] font-['JetBrains_Mono'] uppercase px-1.5 py-0.5 bg-[var(--status-progress-bg)] rounded">
                      has deps
                    </span>
                  )}
                </div>
              </div>

              {/* Tags */}
              {task.tags && task.tags.length > 0 && (
                <div className="flex items-center gap-2 flex-wrap">
                  <Tag size={12} className="text-[var(--text-dim)]" />
                  {task.tags.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-0.5 rounded bg-[var(--bg-elevated)] text-[var(--text-secondary)] text-[10px] font-['JetBrains_Mono']"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              )}

              {/* Dependencies */}
              {(dependencies.length > 0 || dependents.length > 0) && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Dependencies
                  </h3>
                  <div className="flex flex-col gap-1">
                    {dependencies.map((dep) => {
                      const url = `?task=${dep.id}`;
                      const status = getDepStatus(dep, columnSlugById);
                      return (
                        <a
                          key={dep.id}
                          href={url}
                          onClick={(e) => {
                            e.preventDefault();
                            onTaskNavigate?.(dep.id);
                          }}
                          className="flex items-center gap-2 px-2 py-1.5 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--primary)] hover:bg-[var(--status-done-bg)] transition-colors group"
                        >
                          <ArrowRight size={11} className="text-[var(--primary)] flex-shrink-0" />
                          <span className="text-[var(--text-secondary)] text-xs font-['Inter'] truncate group-hover:text-[var(--text-primary)] transition-colors">
                            {dep.title}
                          </span>
                          <span className={`text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded flex-shrink-0 ${status.className}`}>
                            {status.label}
                          </span>
                          <span className="text-[var(--text-dim)] text-[10px] font-['JetBrains_Mono'] flex-shrink-0">
                            depends on
                          </span>
                        </a>
                      );
                    })}
                    {dependents.map((dep) => {
                      const url = `?task=${dep.id}`;
                      const status = getDepStatus(dep, columnSlugById);
                      return (
                        <a
                          key={dep.id}
                          href={url}
                          onClick={(e) => {
                            e.preventDefault();
                            onTaskNavigate?.(dep.id);
                          }}
                          className="flex items-center gap-2 px-2 py-1.5 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)] hover:border-[var(--status-progress)] hover:bg-[var(--status-progress-bg)] transition-colors group"
                        >
                          <ArrowLeft size={11} className="text-[var(--status-progress)] flex-shrink-0" />
                          <span className="text-[var(--text-secondary)] text-xs font-['Inter'] truncate group-hover:text-[var(--text-primary)] transition-colors">
                            {dep.title}
                          </span>
                          <span className={`text-[10px] font-['JetBrains_Mono'] px-1.5 py-0.5 rounded flex-shrink-0 ${status.className}`}>
                            {status.label}
                          </span>
                          <span className="text-[var(--text-dim)] text-[10px] font-['JetBrains_Mono'] flex-shrink-0">
                            needed by
                          </span>
                        </a>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Blocked Banner */}
              {task.is_blocked && (
                <BlockedBanner
                  task={task}
                  onUnblock={() => onAction('unblock')}
                />
              )}

              {/* Summary */}
              {task.summary && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Summary
                  </h3>
                  <MarkdownContent content={task.summary} />
                </div>
              )}

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
              {task.resolution && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Resolution
                  </h3>
                  <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-md p-3">
                    <MarkdownContent content={task.resolution} />
                  </div>
                </div>
              )}

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
              {task.context_files && task.context_files.length > 0 && (
                <div>
                  <h3 className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] uppercase tracking-wider mb-2">
                    Context Files
                  </h3>
                  <div className="flex flex-col gap-1">
                    {task.context_files.map((f) => (
                      <div
                        key={f}
                        className="flex items-center gap-2 px-2 py-1 rounded bg-[var(--bg-secondary)] border border-[var(--border-primary)]"
                      >
                        <FileCode2 size={12} className="text-[var(--text-dim)] flex-shrink-0" />
                        <span className="text-[var(--text-secondary)] text-xs font-['JetBrains_Mono'] truncate">
                          {f}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

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

              {/* Meta info */}
              <div className="text-[var(--text-muted)] text-[10px] font-['Inter'] flex flex-col gap-0.5">
                {task.created_by_role && <span>Created by role: {task.created_by_role}</span>}
                {task.created_by_agent && <span>Created by agent: {task.created_by_agent}</span>}
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
