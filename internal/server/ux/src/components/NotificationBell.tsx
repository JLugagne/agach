import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Bell } from 'lucide-react';
import { listNotifications, getNotificationUnreadCount, markNotificationRead, markAllNotificationsRead } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { NotificationResponse, NotificationSeverity } from '../lib/types';

const SEVERITY_COLORS: Record<NotificationSeverity, string> = {
  info: 'var(--severity-info)',
  success: 'var(--severity-success)',
  warning: 'var(--severity-warning)',
  error: 'var(--severity-error)',
};

const SEVERITY_BTN_STYLES: Record<string, { bg: string; text: string }> = {
  primary: { bg: 'var(--primary)', text: '#FFFFFF' },
  secondary: { bg: 'transparent', text: 'var(--text-muted)' },
  danger: { bg: 'var(--severity-error)', text: '#FFFFFF' },
  warning: { bg: 'var(--severity-warning)', text: '#0D0F17' },
};

function timeAgo(dateStr: string): string {
  const now = Date.now();
  const date = new Date(dateStr).getTime();
  const diff = Math.floor((now - date) / 1000);
  if (diff < 60) return 'just now';
  if (diff < 3600) return `${Math.floor(diff / 60)} min ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} hour${Math.floor(diff / 3600) > 1 ? 's' : ''} ago`;
  if (diff < 172800) return 'Yesterday';
  return `${Math.floor(diff / 86400)} days ago`;
}

export function NotificationBell() {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [notifications, setNotifications] = useState<NotificationResponse[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const fetchData = useCallback(() => {
    getNotificationUnreadCount().then((r) => setUnreadCount(r.unread_count)).catch(() => {});
    listNotifications({ limit: 5 }).then((n) => setNotifications(n ?? [])).catch(() => {});
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Refresh on WS notification events
  useWebSocket(
    useCallback((event) => {
      if (event.type === 'notification') fetchData();
    }, [fetchData]),
  );

  // Close on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    if (open) document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  const handleMarkAllRead = async () => {
    await markAllNotificationsRead();
    setUnreadCount(0);
    setNotifications((prev) => prev.map((n) => ({ ...n, read_at: new Date().toISOString() })));
  };

  const handleClickItem = async (notif: NotificationResponse) => {
    if (!notif.read_at) {
      markNotificationRead(notif.id).catch(() => {});
      setUnreadCount((c) => Math.max(0, c - 1));
      setNotifications((prev) => prev.map((n) => n.id === notif.id ? { ...n, read_at: new Date().toISOString() } : n));
    }
    if (notif.link_url) {
      setOpen(false);
      navigate(notif.link_url);
    }
  };

  return (
    <div ref={dropdownRef} className="relative">
      {/* Bell button */}
      <button
        onClick={() => setOpen((v) => !v)}
        data-qa="notification-bell-btn"
        className="relative p-1 rounded-md hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer"
      >
        <Bell size={18} className="text-[var(--text-muted)]" />
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-red-500" />
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div
          className="absolute bottom-full left-0 mb-2 w-[380px] rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] shadow-xl overflow-hidden z-50"
          style={{ boxShadow: '0 4px 24px rgba(0,0,0,0.25)' }}
        >
          {/* Header */}
          <div className="flex items-center px-5 py-4 border-b border-[var(--border-primary)]">
            <span className="text-base font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
              Notifications
            </span>
            <span className="flex-1" />
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                data-qa="notif-dropdown-mark-all-btn"
                className="text-[13px] font-medium text-[var(--primary)] hover:underline cursor-pointer"
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                Mark all read
              </button>
            )}
          </div>

          {/* Items */}
          <div className="max-h-[400px] overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="px-5 py-8 text-center text-[13px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                No notifications yet
              </div>
            ) : (
              notifications.map((notif) => (
                <div
                  key={notif.id}
                  className="flex gap-3 px-5 py-3 border-b border-[var(--border-primary)] last:border-b-0 cursor-pointer hover:bg-[var(--nav-bg-active)]/30 transition-colors"
                  style={!notif.read_at ? { backgroundColor: 'var(--notif-unread-bg)' } : undefined}
                  onClick={() => handleClickItem(notif)}
                  data-qa="notif-dropdown-item"
                >
                  {/* Severity dot */}
                  <div className="flex items-start pt-1.5 shrink-0">
                    {!notif.read_at && (
                      <div
                        className="w-2 h-2 rounded-full"
                        style={{ backgroundColor: SEVERITY_COLORS[notif.severity] }}
                      />
                    )}
                    {notif.read_at && <div className="w-2" />}
                  </div>
                  {/* Content */}
                  <div className="flex flex-col gap-1 flex-1 min-w-0">
                    <span className="text-[13px] font-medium text-[var(--text-primary)] truncate" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {notif.title}
                    </span>
                    <span className="text-[12px] text-[var(--text-dim)] line-clamp-2" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {notif.text}
                    </span>
                    {notif.link_text && (
                      <span
                        className="inline-flex self-start mt-0.5 px-2.5 py-0.5 rounded-md text-[11px] font-medium"
                        style={{
                          fontFamily: 'Inter, sans-serif',
                          backgroundColor: SEVERITY_BTN_STYLES[notif.link_style || 'primary']?.bg ?? 'var(--primary)',
                          color: SEVERITY_BTN_STYLES[notif.link_style || 'primary']?.text ?? '#FFFFFF',
                          border: notif.link_style === 'secondary' ? '1px solid var(--border-primary)' : 'none',
                        }}
                      >
                        {notif.link_text}
                      </span>
                    )}
                    <span className="text-[11px] text-[var(--text-dim)]" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {timeAgo(notif.created_at)}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Footer */}
          <div className="flex justify-center px-5 py-3 border-t border-[var(--border-primary)]">
            <button
              onClick={() => { setOpen(false); navigate('/notifications'); }}
              data-qa="notif-dropdown-view-all-btn"
              className="text-[13px] font-medium text-[var(--primary)] hover:underline cursor-pointer"
              style={{ fontFamily: 'Inter, sans-serif' }}
            >
              View all notifications
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
