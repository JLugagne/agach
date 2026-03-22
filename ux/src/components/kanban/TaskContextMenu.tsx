import { useEffect, useRef, useCallback } from 'react';
import {
  Pencil,
  ArrowRight,
  Ban,
  Trash2,
  CheckCircle2,
  Unlock,
  CircleSlash,
  Copy,
  FolderOutput,
  UserCircle,
} from 'lucide-react';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface RoleOption {
  slug: string;
  name: string;
  color: string;
}

interface TaskContextMenuProps {
  task: TaskWithDetailsResponse;
  column: string; // slug: todo, in_progress, done, blocked
  position: { x: number; y: number };
  projectId: string;
  roles: RoleOption[];
  onClose: () => void;
  onAction: (action: string) => void;
}

interface MenuItem {
  label: string;
  icon: React.ReactNode;
  action: string;
  danger?: boolean;
  currentPriority?: boolean;
  currentRole?: boolean;
}

const priorityDot = (color: string) => (
  <span
    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
    style={{ backgroundColor: color }}
  />
);

const roleDot = (color: string) => (
  <span
    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
    style={{ backgroundColor: color }}
  />
);

function getPriorityItems(currentPriority: string): MenuItem[] {
  const priorities = [
    { label: 'Critical', action: 'priority_critical', color: 'var(--priority-critical)' },
    { label: 'High', action: 'priority_high', color: 'var(--priority-high)' },
    { label: 'Medium', action: 'priority_medium', color: 'var(--priority-medium)' },
    { label: 'Low', action: 'priority_low', color: 'var(--priority-low)' },
  ];
  return priorities.map((p) => ({
    label: `Set ${p.label}`,
    icon: priorityDot(p.color),
    action: p.action,
    currentPriority: currentPriority === p.action.replace('priority_', ''),
  }));
}

function getRoleItems(roles: RoleOption[], currentRole: string): (MenuItem | 'divider')[] {
  if (roles.length === 0) return [];
  const items: (MenuItem | 'divider')[] = ['divider'];
  items.push({
    label: 'Unassign Role',
    icon: <UserCircle size={14} />,
    action: 'role_unassign',
    currentRole: !currentRole,
  });
  for (const role of roles) {
    items.push({
      label: `Assign ${role.name}`,
      icon: roleDot(role.color),
      action: `role_${role.slug}`,
      currentRole: currentRole === role.slug,
    });
  }
  return items;
}

function getMenuItems(
  column: string,
  currentPriority: string,
  roles: RoleOption[],
  currentRole: string,
): (MenuItem | 'divider')[] {
  const priorityItems = getPriorityItems(currentPriority);
  const roleItems = getRoleItems(roles, currentRole);
  const duplicateItem: MenuItem = { label: 'Duplicate', icon: <Copy size={14} />, action: 'duplicate' };
  const moveToProjectItem: MenuItem = { label: 'Move to Project', icon: <FolderOutput size={14} />, action: 'move_to_project' };

  switch (column) {
    case 'todo':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        ...priorityItems,
        duplicateItem,
        ...roleItems,
        'divider',
        { label: 'Move to In Progress', icon: <ArrowRight size={14} />, action: 'move_in_progress' },
        { label: 'Block Task', icon: <Ban size={14} />, action: 'block' },
        { label: "Won't Do", icon: <CircleSlash size={14} />, action: 'wontdo' },
        'divider',
        moveToProjectItem,
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'in_progress':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        ...priorityItems,
        duplicateItem,
        ...roleItems,
        'divider',
        { label: 'Move to Todo', icon: <ArrowRight size={14} />, action: 'move_todo' },
        { label: 'Complete', icon: <CheckCircle2 size={14} />, action: 'complete' },
        { label: 'Block Task', icon: <Ban size={14} />, action: 'block' },
        'divider',
        moveToProjectItem,
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'blocked':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        ...priorityItems,
        duplicateItem,
        ...roleItems,
        'divider',
        { label: 'Unblock', icon: <Unlock size={14} />, action: 'unblock' },
        'divider',
        moveToProjectItem,
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'done':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        ...priorityItems,
        duplicateItem,
        ...roleItems,
        'divider',
        { label: 'Move to Todo', icon: <ArrowRight size={14} />, action: 'move_todo' },
        'divider',
        moveToProjectItem,
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    default:
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        ...priorityItems,
        duplicateItem,
        ...roleItems,
        'divider',
        moveToProjectItem,
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
  }
}

export default function TaskContextMenu({
  task,
  column,
  position,
  projectId: _projectId,
  roles,
  onClose,
  onAction,
}: TaskContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
      onClose();
    }
  }, [onClose]);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  }, [onClose]);

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleClickOutside, handleKeyDown]);

  // Adjust position to stay within viewport
  useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      if (rect.right > viewportWidth) {
        menuRef.current.style.left = `${position.x - rect.width}px`;
      }
      if (rect.bottom > viewportHeight) {
        menuRef.current.style.top = `${position.y - rect.height}px`;
      }
    }
  }, [position]);

  const items = getMenuItems(column, task.priority, roles, task.assigned_role || '');

  return (
    <div
      ref={menuRef}
      className="fixed z-[60] min-w-[200px] rounded-lg bg-[#1A1A1A] border border-[#2A2A2A] shadow-2xl py-1 overflow-hidden"
      style={{ left: position.x, top: position.y }}
    >
      {items.map((item, idx) => {
        if (item === 'divider') {
          return (
            <div key={`divider-${idx}`} className="h-px bg-[#2A2A2A] my-1" />
          );
        }

        const isCurrent = item.currentPriority || item.currentRole;

        return (
          <button
            key={item.action}
            data-qa={`context-menu-${item.action}-btn`}
            onClick={() => {
              onAction(item.action);
              onClose();
            }}
            className={`w-full h-9 flex items-center gap-2 px-3 text-sm font-['Inter'] whitespace-nowrap transition-colors ${
              item.danger
                ? 'text-[#F06060] hover:bg-[#F0606015]'
                : isCurrent
                  ? 'text-[var(--text-primary)] bg-[#252525]'
                  : 'text-[#E0E0E0] hover:bg-[#252525]'
            }`}
          >
            {item.icon}
            {item.label}
            {isCurrent && (
              <span className="ml-auto text-[9px] font-['JetBrains_Mono'] text-[var(--text-muted)] uppercase tracking-wider">
                current
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
