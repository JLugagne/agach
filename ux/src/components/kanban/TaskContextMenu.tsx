import { useEffect, useRef, useCallback } from 'react';
import {
  Pencil,
  ArrowRight,
  Ban,
  Trash2,
  CheckCircle2,
  Unlock,
  CircleSlash,
} from 'lucide-react';
import type { TaskWithDetailsResponse } from '../../lib/types';

interface TaskContextMenuProps {
  task: TaskWithDetailsResponse;
  column: string; // slug: todo, in_progress, done, blocked
  position: { x: number; y: number };
  projectId: string;
  onClose: () => void;
  onAction: (action: string) => void;
}

interface MenuItem {
  label: string;
  icon: React.ReactNode;
  action: string;
  danger?: boolean;
}

function getMenuItems(column: string): (MenuItem | 'divider')[] {
  switch (column) {
    case 'todo':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        { label: 'Move to In Progress', icon: <ArrowRight size={14} />, action: 'move_in_progress' },
        { label: 'Block Task', icon: <Ban size={14} />, action: 'block' },
        { label: "Won't Do", icon: <CircleSlash size={14} />, action: 'wontdo' },
        'divider',
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'in_progress':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        { label: 'Move to Todo', icon: <ArrowRight size={14} />, action: 'move_todo' },
        { label: 'Complete', icon: <CheckCircle2 size={14} />, action: 'complete' },
        { label: 'Block Task', icon: <Ban size={14} />, action: 'block' },
        'divider',
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'blocked':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        { label: 'Unblock', icon: <Unlock size={14} />, action: 'unblock' },
        'divider',
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    case 'done':
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        { label: 'Move to Todo', icon: <ArrowRight size={14} />, action: 'move_todo' },
        'divider',
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
    default:
      return [
        { label: 'Edit Task', icon: <Pencil size={14} />, action: 'edit' },
        'divider',
        { label: 'Delete Task', icon: <Trash2 size={14} />, action: 'delete', danger: true },
      ];
  }
}

export default function TaskContextMenu({
  task: _task,
  column,
  position,
  projectId: _projectId,
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

  const items = getMenuItems(column);

  return (
    <div
      ref={menuRef}
      className="fixed z-[60] w-[180px] rounded-lg bg-[#1A1A1A] border border-[#2A2A2A] shadow-2xl py-1 overflow-hidden"
      style={{ left: position.x, top: position.y }}
    >
      {items.map((item, idx) => {
        if (item === 'divider') {
          return (
            <div key={`divider-${idx}`} className="h-px bg-[#2A2A2A] my-1" />
          );
        }

        return (
          <button
            key={item.action}
            onClick={() => {
              onAction(item.action);
              onClose();
            }}
            className={`w-full h-9 flex items-center gap-2 px-3 text-sm font-['Inter'] transition-colors ${
              item.danger
                ? 'text-[#F06060] hover:bg-[#F0606015]'
                : 'text-[#E0E0E0] hover:bg-[#252525]'
            }`}
          >
            {item.icon}
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
