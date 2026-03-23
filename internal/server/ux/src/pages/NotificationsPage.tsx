import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { listNotifications, getNotificationUnreadCount, markNotificationRead, markAllNotificationsRead } from '../lib/api';
import { useWebSocket } from '../hooks/useWebSocket';
import type { NotificationResponse, NotificationSeverity } from '../lib/types';

const SEVERITY_COLORS: Record<NotificationSeverity, string> = {
  info: 'var(--severity-info)',
  success: 'var(--severity-success)',
  warning: 'var(--severity-warning)',
  error: 'var(--severity-error)',
};

const SEVERITY_BTN_STYLES: Record<string, { bg: string; text: string; border?: string }> = {
  primary: { bg: 'var(--primary)', text: '#FFFFFF' },
  secondary: { bg: 'transparent', text: 'var(--text-muted)', border: '1px solid var(--border-primary)' },
  danger: { bg: 'var(--severity-error)', text: '#FFFFFF' },
  warning: { bg: 'var(--severity-warning)', text: '#0D0F17' },
};

const SCOPE_FILTERS: { label: string; value: string | undefined }[] = [
  { label: 'All', value: undefined },
  { label: 'Project', value: 'project' },
  { label: 'Agent', value: 'agent' },
  { label: 'Global', value: 'global' },
];

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

export default function NotificationsPage() {
  const navigate = useNavigate();
  const [notifications, setNotifications] = useState<NotificationResponse[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [scopeFilter, setScopeFilter] = useState<string | undefined>(undefined);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(() => {
    const params: { scope?: string; limit?: number } = { limit: 50 };
    if (scopeFilter) params.scope = scopeFilter;
    setLoading(true);
    Promise.all([
      listNotifications(params),
      getNotificationUnreadCount(scopeFilter ? { scope: scopeFilter } : undefined),
    ]).then(([n, c]) => {
      setNotifications(n ?? []);
      setUnreadCount(c.unread_count);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [scopeFilter]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useWebSocket(
    useCallback((event) => {
      if (event.type === 'notification') fetchData();
    }, [fetchData]),
  );

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
      navigate(notif.link_url);
    }
  };

  return (
    <div className="flex flex-col gap-6 h-full overflow-y-auto p-8 md:px-10" data-qa="notifications-page">
      {/* Header */}
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold text-[var(--text-primary)]" style={{ fontFamily: 'Inter, sans-serif' }}>
          Notifications
        </h1>
        <p className="text-sm text-[var(--text-dim)]" style={{ fontFamily: 'Inter, sans-serif' }}>
          {unreadCount > 0 ? `${unreadCount} unread notification${unreadCount !== 1 ? 's' : ''}` : 'All caught up'}
        </p>
      </div>

      {/* Filter row */}
      <div className="flex items-center gap-3 flex-wrap">
        {SCOPE_FILTERS.map((f) => (
          <button
            key={f.label}
            onClick={() => setScopeFilter(f.value)}
            data-qa={`notif-filter-${f.label.toLowerCase()}`}
            className="px-4 py-1.5 rounded-full text-[13px] transition-colors cursor-pointer"
            style={{
              fontFamily: 'Inter, sans-serif',
              backgroundColor: scopeFilter === f.value ? 'var(--primary)' : 'var(--bg-elevated)',
              color: scopeFilter === f.value ? '#FFFFFF' : 'var(--text-muted)',
            }}
          >
            {f.label}
          </button>
        ))}
        <span className="flex-1" />
        {unreadCount > 0 && (
          <button
            onClick={handleMarkAllRead}
            data-qa="notif-mark-all-read-btn"
            className="px-4 py-2 rounded-lg text-[13px] font-medium text-[var(--text-muted)] border border-[var(--border-primary)] hover:bg-[var(--nav-bg-active)]/50 transition-colors cursor-pointer"
            style={{ fontFamily: 'Inter, sans-serif' }}
          >
            Mark all as read
          </button>
        )}
      </div>

      {/* Divider */}
      <div className="h-px bg-[var(--border-primary)]" />

      {/* Notification list */}
      <div className="rounded-xl border border-[var(--border-primary)] bg-[var(--bg-secondary)] overflow-hidden flex-1">
        {loading && notifications.length === 0 ? (
          <div className="px-5 py-12 text-center text-[13px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            Loading...
          </div>
        ) : notifications.length === 0 ? (
          <div className="px-5 py-12 text-center text-[13px] text-[var(--text-muted)]" style={{ fontFamily: 'Inter, sans-serif' }}>
            No notifications
          </div>
        ) : (
          notifications.map((notif, i) => {
            const isUnread = !notif.read_at;
            return (
              <div key={notif.id}>
                <div
                  className="flex gap-4 px-5 py-4 cursor-pointer hover:bg-[var(--nav-bg-active)]/20 transition-colors"
                  style={{
                    backgroundColor: isUnread ? 'var(--notif-unread-bg)' : undefined,
                    borderLeft: isUnread ? '3px solid var(--primary)' : '3px solid transparent',
                  }}
                  onClick={() => handleClickItem(notif)}
                  data-qa="notif-list-item"
                >
                  {/* Severity dot */}
                  <div className="flex items-start pt-1.5 shrink-0">
                    <div
                      className="w-2.5 h-2.5 rounded-full"
                      style={{
                        backgroundColor: SEVERITY_COLORS[notif.severity],
                        opacity: isUnread ? 1 : 0.4,
                      }}
                    />
                  </div>

                  {/* Middle content */}
                  <div className="flex flex-col gap-1 flex-1 min-w-0">
                    <span
                      className="text-sm font-medium"
                      style={{
                        fontFamily: 'Inter, sans-serif',
                        color: isUnread ? 'var(--text-primary)' : 'var(--text-muted)',
                      }}
                    >
                      {notif.title}
                    </span>
                    <span
                      className="text-[13px] line-clamp-2"
                      style={{
                        fontFamily: 'Inter, sans-serif',
                        color: isUnread ? 'var(--text-dim)' : 'var(--text-dim)',
                      }}
                    >
                      {notif.text}
                    </span>
                    {notif.link_text && (
                      <span
                        className="inline-flex self-start mt-1 px-3 py-1 rounded-md text-[12px] font-medium"
                        style={{
                          fontFamily: 'Inter, sans-serif',
                          backgroundColor: SEVERITY_BTN_STYLES[notif.link_style || 'primary']?.bg ?? 'var(--primary)',
                          color: SEVERITY_BTN_STYLES[notif.link_style || 'primary']?.text ?? '#FFFFFF',
                          border: SEVERITY_BTN_STYLES[notif.link_style || 'primary']?.border ?? 'none',
                        }}
                      >
                        {notif.link_text}
                      </span>
                    )}
                  </div>

                  {/* Right: time + scope badge */}
                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <span className="text-[12px] text-[var(--text-dim)] whitespace-nowrap" style={{ fontFamily: 'Inter, sans-serif' }}>
                      {timeAgo(notif.created_at)}
                    </span>
                    <span
                      className="px-2 py-0.5 rounded-full text-[11px] text-[var(--text-dim)] bg-[var(--bg-elevated)]"
                      style={{ fontFamily: 'Inter, sans-serif' }}
                    >
                      {notif.scope}
                    </span>
                  </div>
                </div>
                {i < notifications.length - 1 && (
                  <div className="h-px bg-[var(--border-primary)]" />
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
