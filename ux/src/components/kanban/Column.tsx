import { useState } from 'react';
import type { ColumnWithTasksResponse, ProjectWithSummary, TaskWithDetailsResponse } from '../../lib/types';
import TaskCard from './TaskCard';
import FeatureCard from './FeatureCard';
import { DndContext, closestCenter, PointerSensor, useSensor, useSensors } from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, arrayMove } from '@dnd-kit/sortable';
import { reorderTask } from '../../lib/api';
import { AlertTriangle, Plus } from 'lucide-react';

interface ColumnProps {
  column: ColumnWithTasksResponse;
  projectId: string;
  roleColorMap?: Record<string, string>;
  onTaskClick: (task: TaskWithDetailsResponse) => void;
  onTaskContextMenu: (task: TaskWithDetailsResponse, e: React.MouseEvent) => void;
  isTaskNew?: (taskId: string) => boolean;
  isTaskHighlighted?: (taskId: string) => boolean;
  isTaskSelected?: (taskId: string) => boolean;
  onTaskSelect?: (taskId: string, ctrlKey: boolean) => void;
  onRefresh?: () => void;
  features?: ProjectWithSummary[];
  onFeatureClick?: (feature: ProjectWithSummary) => void;
  onAddTask?: () => void;
}

const statusColorVars: Record<string, { dot: string; label: string; countBg: string; countText: string }> = {
  todo: { dot: 'var(--status-todo)', label: 'var(--status-todo)', countBg: 'var(--status-todo-bg)', countText: 'var(--status-todo)' },
  in_progress: { dot: 'var(--status-progress)', label: 'var(--status-progress)', countBg: 'var(--status-progress-bg)', countText: 'var(--status-progress)' },
  done: { dot: 'var(--status-done)', label: 'var(--status-done)', countBg: 'var(--status-done-bg)', countText: 'var(--status-done)' },
  blocked: { dot: 'var(--status-blocked)', label: 'var(--status-blocked)', countBg: 'var(--status-blocked-bg)', countText: 'var(--status-blocked)' },
};

export default function Column({
  column,
  projectId,
  roleColorMap,
  onTaskClick,
  onTaskContextMenu,
  isTaskNew,
  isTaskHighlighted,
  isTaskSelected,
  onTaskSelect,
  onRefresh,
  features,
  onFeatureClick,
  onAddTask,
}: ColumnProps) {
  const colors = statusColorVars[column.slug] || statusColorVars.todo;
  const [localTasks, setLocalTasks] = useState<TaskWithDetailsResponse[] | null>(null);

  const tasks = localTasks ?? column.tasks ?? [];
  const wipLimit = column.wip_limit;
  const isAtWip = wipLimit > 0 && tasks.length >= wipLimit;

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  // Reset local order whenever the column prop changes (e.g. after board refresh)
  const prevColumnRef = column.tasks;
  if (localTasks !== null && prevColumnRef !== undefined) {
    // Only reset if the set of task IDs changed (a real board refresh, not our optimistic update)
    const localIds = localTasks.map((t) => t.id).join(',');
    const serverIds = (column.tasks ?? []).map((t) => t.id).join(',');
    if (localIds !== serverIds && localIds.split(',').sort().join(',') !== serverIds.split(',').sort().join(',')) {
      setLocalTasks(null);
    }
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = tasks.findIndex((t) => t.id === active.id);
    const newIndex = tasks.findIndex((t) => t.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    // Optimistic update
    const reordered = arrayMove(tasks, oldIndex, newIndex);
    setLocalTasks(reordered);

    // Determine the task's project id
    const task = tasks[oldIndex];
    const pid = task.project_id || projectId;

    try {
      await reorderTask(pid, task.id as string, newIndex);
    } catch {
      // Revert on failure
      setLocalTasks(null);
    } finally {
      onRefresh?.();
    }
  };

  return (
    <div
      data-qa="column"
      className="flex flex-col min-w-0 flex-1 h-full"
    >
      {/* Header */}
      <div
        className="flex items-center justify-between mb-4 flex-shrink-0"
      >
        <div className="flex items-center gap-2.5">
          <div
            className="w-2.5 h-2.5 rounded-full"
            style={{ backgroundColor: colors.dot }}
          />
          <span
            data-qa="column-title"
            className="text-xs font-bold uppercase tracking-wider"
            style={{ color: colors.label, fontFamily: 'JetBrains Mono, monospace' }}
          >
            {column.name}
          </span>
          {wipLimit > 0 ? (
            <span
              className="px-1.5 py-0.5 rounded-md text-[10px] font-bold min-w-[28px] text-center flex items-center gap-0.5"
              style={{
                backgroundColor: isAtWip
                  ? 'color-mix(in srgb, var(--status-blocked) 15%, transparent)'
                  : colors.countBg,
                color: isAtWip ? 'var(--status-blocked)' : colors.countText,
                fontFamily: 'JetBrains Mono, monospace',
              }}
            >
              {isAtWip && (
                <AlertTriangle size={10} className="flex-shrink-0" />
              )}
              {tasks.length}/{wipLimit}
            </span>
          ) : (
            <span
              className="px-1.5 py-0.5 rounded-md text-[10px] font-bold min-w-[20px] text-center"
              style={{
                backgroundColor: colors.countBg,
                color: colors.countText,
                fontFamily: 'JetBrains Mono, monospace',
              }}
            >
              {tasks.length}
            </span>
          )}
        </div>
        {onAddTask && (
          <button
            onClick={onAddTask}
            className="text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors cursor-pointer p-0.5"
          >
            <Plus size={16} />
          </button>
        )}
      </div>

      {/* Cards */}
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-3 overflow-y-auto flex-1">
            {tasks.length === 0 && (features ?? []).length === 0 ? (
              <p className="text-[var(--text-muted)] text-xs text-center py-8" style={{ fontFamily: 'Inter, sans-serif' }}>
                No tasks
              </p>
            ) : (
              <>
                {(features ?? []).map((f) => (
                  <FeatureCard key={f.id} feature={f} onClick={() => onFeatureClick?.(f)} />
                ))}
                {tasks.map((task) => (
                  <TaskCard
                    key={task.id}
                    task={task}
                    columnSlug={column.slug}
                    isNew={isTaskNew ? isTaskNew(task.id) : false}
                    isHighlighted={isTaskHighlighted ? isTaskHighlighted(task.id) : false}
                    selected={isTaskSelected ? isTaskSelected(task.id) : false}
                    roleColor={task.assigned_role ? roleColorMap?.[task.assigned_role] : undefined}
                    onClick={() => onTaskClick(task)}
                    onContextMenu={(e) => onTaskContextMenu(task, e)}
                    onSelect={onTaskSelect}
                  />
                ))}
              </>
            )}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}
