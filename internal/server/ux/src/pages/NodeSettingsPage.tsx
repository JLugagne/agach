import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { getNode, listDockerfiles } from '../lib/api';
import { wsClient } from '../lib/ws';
import type { NodeResponse, DockerfileResponse } from '../lib/types';
import {
  ChevronRight, ChevronDown as ChevronDownIcon, LayoutDashboard, Container, ListChecks,
  TriangleAlert, Search, ChevronDown, Info, Clock, User, FolderOpen, GitBranch,
  Plus, Trash2, RefreshCw, Terminal, Loader2, X, Check,
} from 'lucide-react';

type Tab = 'overview' | 'dockerfiles' | 'tasks';

export default function NodeSettingsPage() {
  const { nodeId } = useParams<{ nodeId: string }>();
  const navigate = useNavigate();

  const [node, setNode] = useState<NodeResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>('overview');

  const fetchNode = useCallback(async () => {
    if (!nodeId) return;
    try {
      const data = await getNode(nodeId);
      setNode(data.node);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load node');
    } finally {
      setLoading(false);
    }
  }, [nodeId]);

  useEffect(() => { fetchNode(); }, [fetchNode]);

  if (loading) {
    return (
      <div className="flex justify-center items-center h-full text-[var(--text-muted)] font-mono">
        Loading...
      </div>
    );
  }

  if (error || !node) {
    return (
      <div className="flex flex-col justify-center items-center h-full gap-4">
        <p className="text-[var(--severity-error)] font-mono text-sm">
          {error || 'Node not found'}
        </p>
        <button
          onClick={() => navigate('/nodes')}
          className="px-4 py-2 rounded-lg border border-[var(--border-primary)] bg-transparent text-[var(--text-muted)] font-mono text-[13px] cursor-pointer hover:border-[var(--border-secondary)] transition-colors"
        >
          Back to Nodes
        </button>
      </div>
    );
  }

  const tabs: { key: Tab; label: string; icon: typeof LayoutDashboard; badge?: string }[] = [
    { key: 'overview', label: 'Overview', icon: LayoutDashboard },
    { key: 'dockerfiles', label: 'Dockerfiles', icon: Container },
    { key: 'tasks', label: 'Tasks', icon: ListChecks },
  ];

  return (
    <div data-qa="node-settings-page" className="h-full flex flex-col bg-[var(--bg-primary)] overflow-hidden">
      <div className="flex-1 overflow-y-auto px-10 py-8">
        <div className="flex flex-col gap-4 mb-7">
          {/* Breadcrumb */}
          <div data-qa="breadcrumb" className="flex items-center gap-2">
            <Link to="/nodes" className="text-[var(--text-dim)] text-sm no-underline hover:text-[var(--text-muted)] transition-colors" style={{ fontFamily: 'Inter, sans-serif' }}>
              Nodes
            </Link>
            <ChevronRight size={12} className="text-[var(--text-dim)]" />
            <span className="text-[var(--text-primary)] text-sm" style={{ fontFamily: 'Inter, sans-serif' }}>
              {node.name || 'Unnamed node'}
            </span>
          </div>

          {/* Title row */}
          <div data-qa="title-row" className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="text-[26px] font-semibold text-[var(--text-primary)] m-0" style={{ fontFamily: 'Inter, sans-serif' }}>
                {node.name || 'Unnamed node'}
              </h1>
              {node.status === 'active' && (
                <div data-qa="status-badge" className="flex items-center gap-1.5">
                  <div className="w-2 h-2 rounded-full bg-[var(--severity-success)]" />
                  <span className="text-[var(--severity-success)] text-[13px]" style={{ fontFamily: 'Inter, sans-serif' }}>Active</span>
                </div>
              )}
            </div>
          </div>

          {/* Warning banner */}
          <div data-qa="warning-banner" className="flex items-start gap-3 p-4 rounded-lg bg-[var(--bg-elevated)]/50 border border-[var(--severity-warning)]">
            <TriangleAlert size={20} className="text-[var(--severity-warning)] shrink-0 mt-px" />
            <div className="flex flex-col gap-1">
              <span className="text-[var(--severity-warning)] text-sm font-semibold" style={{ fontFamily: 'Inter, sans-serif' }}>
                Remote System Configuration
              </span>
              <span className="text-[var(--text-dim)] text-[13px]" style={{ fontFamily: 'Inter, sans-serif' }}>
                Changes to this node affect a remote daemon instance. Proceed with caution.
              </span>
            </div>
          </div>

          {/* Tab bar */}
          <div data-qa="tab-bar" className="flex border-b border-[var(--border-primary)]">
            {tabs.map((tab) => {
              const isActive = activeTab === tab.key;
              const Icon = tab.icon;
              return (
                <button
                  key={tab.key}
                  data-qa={`tab-${tab.key}`}
                  onClick={() => setActiveTab(tab.key)}
                  className={`flex items-center gap-1.5 px-5 py-3 bg-transparent border-none cursor-pointer text-sm transition-colors ${
                    isActive
                      ? 'text-[var(--text-primary)] font-medium border-b-2 border-b-[var(--primary)]'
                      : 'text-[var(--text-dim)] border-b-2 border-b-transparent hover:text-[var(--text-muted)]'
                  }`}
                  style={{ fontFamily: 'Inter, sans-serif' }}
                >
                  <Icon size={16} className={isActive ? 'text-[var(--primary)]' : 'text-[var(--text-dim)]'} />
                  {tab.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Tab content */}
        <div className="flex flex-col gap-6">
          {activeTab === 'overview' && <OverviewTab node={node} />}
          {activeTab === 'dockerfiles' && <DockerfilesTab />}
          {activeTab === 'tasks' && <TasksTab nodeId={nodeId!} />}
        </div>
      </div>
    </div>
  );
}

function OverviewTab({ node }: { node: NodeResponse }) {
  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return 'Never';
    return new Date(dateStr).toLocaleString();
  };

  const infoRows: { label: string; value: string; mono?: boolean }[] = [
    { label: 'Node ID', value: node.id, mono: true },
    { label: 'Name', value: node.name || 'Unnamed' },
    { label: 'Mode', value: node.mode === 'shared' ? 'Shared' : 'Personal' },
    { label: 'Status', value: node.status },
    { label: 'Last Seen', value: formatDate(node.last_seen_at) },
    { label: 'Created', value: formatDate(node.created_at) },
  ];

  return (
    <div data-qa="overview-tab" className="bg-[var(--bg-secondary)] rounded-xl border border-[var(--border-primary)] p-7 flex flex-col gap-5">
      <h2 className="text-lg font-semibold text-[var(--text-primary)] m-0" style={{ fontFamily: 'Inter, sans-serif' }}>
        Node Information
      </h2>
      <div className="flex flex-col gap-4">
        {infoRows.map((row) => (
          <div key={row.label} className="flex items-center gap-4">
            <span className="w-[120px] text-[13px] font-mono text-[var(--text-muted)]">
              {row.label}
            </span>
            <span className={`text-[13px] text-[var(--text-primary)] ${row.mono ? 'font-mono' : ''}`} style={{ fontFamily: row.mono ? undefined : 'Inter, sans-serif' }}>
              {row.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

interface DaemonBuild {
  build_id: string;
  version: string;
  status: string;
  image_hash?: string;
  size_bytes?: number;
  created_at: string;
}

interface DaemonDockerfile {
  slug: string;
  latest_version: string;
  version_count: number;
  is_healthy: boolean;
  builds: DaemonBuild[];
}

function DockerfilesTab() {
  const [dockerfiles, setDockerfiles] = useState<DaemonDockerfile[]>([]);
  const [serverDockerfiles, setServerDockerfiles] = useState<DockerfileResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [buildingSlug, setBuildingSlug] = useState<string | null>(null);
  const [pruning, setPruning] = useState(false);
  const [logPanel, setLogPanel] = useState<{ buildId: string; slug: string; log: string; status: string; loading: boolean } | null>(null);
  const requestIdRef = useRef(0);

  const fetchDaemonList = useCallback(() => {
    const reqId = `docker-list-${++requestIdRef.current}`;
    wsClient.send({ type: 'docker.list', request_id: reqId, data: {} });
  }, []);

  useEffect(() => {
    wsClient.connect();
    listDockerfiles().then(dfs => setServerDockerfiles(dfs ?? [])).catch(() => {});

    // Initial fetch with a small delay to let WS connect
    const timer = setTimeout(fetchDaemonList, 500);

    const unsub = wsClient.subscribe((event) => {
      if (event.type === 'docker.list' && event.data) {
        const payload = event.data as { dockerfiles?: DaemonDockerfile[] };
        setDockerfiles(payload.dockerfiles ?? []);
        setLoading(false);
      }
      if (event.type === 'docker.build_event' && event.data) {
        const evt = event.data as { dockerfile_slug: string; status: string };
        if (evt.status === 'success' || evt.status === 'failed') {
          setBuildingSlug(null);
          fetchDaemonList();
        }
      }
      if (event.type === 'docker.prune_event' && event.data) {
        const evt = event.data as { status: string };
        if (evt.status === 'completed') {
          setPruning(false);
          fetchDaemonList();
        }
      }
      if (event.type === 'docker.logs' && event.data) {
        const evt = event.data as { build_id: string; slug: string; log: string; status: string };
        setLogPanel(prev => prev && prev.buildId === evt.build_id ? { ...prev, log: evt.log, status: evt.status, loading: false } : prev);
      }
    });

    return () => { unsub(); clearTimeout(timer); };
  }, [fetchDaemonList]);

  const handleRebuild = (slug: string) => {
    setBuildingSlug(slug);
    wsClient.send({ type: 'docker.rebuild', request_id: `rebuild-${Date.now()}`, data: { slug } });
  };

  const handleAddDockerfiles = (slugs: string[]) => {
    setShowAddDialog(false);
    for (const slug of slugs) {
      handleRebuild(slug);
    }
  };

  const handlePrune = () => {
    setPruning(true);
    wsClient.send({ type: 'docker.prune', request_id: `prune-${Date.now()}`, data: {} });
  };

  const handleConsole = (buildId: string, slug: string) => {
    setLogPanel({ buildId, slug, log: '', status: '', loading: true });
    wsClient.send({ type: 'docker.logs', request_id: `logs-${Date.now()}`, data: { build_id: buildId } });
  };

  const toggleExpand = (slug: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(slug)) next.delete(slug); else next.add(slug);
      return next;
    });
  };

  const formatSize = (bytes?: number) => {
    if (!bytes) return '—';
    const mb = bytes / (1024 * 1024);
    return `${Math.round(mb)} MB`;
  };

  const formatDate = (iso: string) => {
    try {
      const d = new Date(iso);
      return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) + ' ' +
        d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
    } catch { return iso; }
  };

  const abbreviateHash = (hash?: string) => {
    if (!hash) return '—';
    if (hash.startsWith('sha256:') && hash.length > 20) {
      return hash.slice(0, 13) + '...' + hash.slice(-4);
    }
    return hash.length > 16 ? hash.slice(0, 12) + '...' + hash.slice(-4) : hash;
  };

  const builtSlugs = new Set(dockerfiles.map(d => d.slug));

  return (
    <div data-qa="dockerfiles-tab" className="flex flex-col gap-4">
      {/* Section header */}
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold text-[var(--text-primary)] m-0" style={{ fontFamily: 'Inter, sans-serif' }}>
          Dockerfile Builds
        </h2>
        {dockerfiles.length > 0 && (
          <span className="px-2 py-1 rounded-md text-xs font-medium text-[var(--text-muted)] bg-[var(--bg-elevated)]">
            {dockerfiles.length} dockerfile{dockerfiles.length !== 1 ? 's' : ''}
          </span>
        )}
        <div className="flex-1" />
        <button
          data-qa="add-docker-btn"
          onClick={() => setShowAddDialog(true)}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer border-none"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          <Plus size={14} />
          Add Docker
        </button>
        <button
          data-qa="prune-images-btn"
          onClick={handlePrune}
          disabled={pruning || dockerfiles.length === 0}
          className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-[13px] font-medium bg-[var(--severity-error)]/10 text-[var(--severity-error)] border border-[var(--severity-error)] hover:bg-[var(--severity-error)]/20 transition-colors cursor-pointer disabled:opacity-50"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          <Trash2 size={14} />
          {pruning ? 'Pruning...' : 'Prune Images'}
        </button>
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="animate-spin text-[var(--text-muted)]" size={24} />
        </div>
      ) : dockerfiles.length === 0 ? (
        <div className="bg-[var(--bg-secondary)] rounded-xl border border-[var(--border-primary)] flex flex-col items-center py-12 px-6">
          <Container size={40} className="text-[var(--text-muted)] mb-4" />
          <p className="text-[var(--text-muted)] font-mono text-sm m-0">
            No dockerfile builds for this node yet
          </p>
          <p className="text-[var(--text-dim)] text-[13px] mt-2 m-0" style={{ fontFamily: 'Inter, sans-serif' }}>
            Use the "Add Docker" button to select a dockerfile and trigger its first build.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {dockerfiles.map(df => {
            const isExpanded = expanded.has(df.slug);
            const isBuilding = buildingSlug === df.slug;
            return (
              <div
                key={df.slug}
                className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-primary)] overflow-hidden"
              >
                {/* Accordion header */}
                <button
                  onClick={() => toggleExpand(df.slug)}
                  className="flex items-center gap-3 w-full px-5 py-4 bg-transparent border-none cursor-pointer text-left"
                  style={{ fontFamily: 'Inter, sans-serif' }}
                >
                  {isExpanded
                    ? <ChevronDownIcon size={16} className="text-[var(--text-muted)] shrink-0" />
                    : <ChevronRight size={16} className="text-[var(--text-muted)] shrink-0" />
                  }
                  <Container size={18} className="text-[var(--primary)] shrink-0" />
                  <span className="text-[14px] font-medium text-[var(--text-primary)]">{df.slug}</span>
                  {df.latest_version && (
                    <span className="px-1.5 py-0.5 rounded text-[11px] font-medium text-[var(--primary-hover)] bg-[var(--primary)]/20">
                      latest
                    </span>
                  )}
                  <div className="flex-1" />
                  <span className="text-[12px] text-[var(--text-dim)]">
                    {df.version_count} version{df.version_count !== 1 ? 's' : ''}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-[11px] font-medium ${
                    df.is_healthy
                      ? 'text-[var(--severity-success)] bg-[var(--severity-success)]/20'
                      : 'text-[var(--severity-error)] bg-[var(--severity-error)]/20'
                  }`}>
                    {df.is_healthy ? 'Healthy' : 'Failed'}
                  </span>
                  <button
                    onClick={(e) => { e.stopPropagation(); handleRebuild(df.slug); }}
                    disabled={isBuilding}
                    data-qa={`rebuild-${df.slug}`}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[12px] font-medium text-[var(--text-muted)] border border-[var(--border-primary)] bg-transparent cursor-pointer hover:border-[var(--border-secondary)] hover:text-[var(--text-primary)] transition-colors disabled:opacity-50"
                    style={{ fontFamily: 'Inter, sans-serif' }}
                  >
                    <RefreshCw size={12} className={isBuilding ? 'animate-spin' : ''} />
                    {isBuilding ? 'Building...' : 'Rebuild'}
                  </button>
                </button>

                {/* Expanded version rows */}
                {isExpanded && df.builds.length > 0 && (
                  <div className="border-t border-[var(--border-primary)]">
                    {df.builds.map((build, idx) => (
                      <div
                        key={build.build_id}
                        className={`flex items-center gap-3 px-5 py-3.5 pl-[52px] ${
                          idx < df.builds.length - 1 ? 'border-b border-[var(--border-primary)]' : ''
                        }`}
                        style={{ fontFamily: 'Inter, sans-serif' }}
                      >
                        <div className="w-[80px] flex items-center gap-1.5">
                          <span className="px-2 py-0.5 rounded text-[12px] font-mono text-[var(--text-primary)] bg-[var(--bg-elevated)]">
                            {build.version.length > 8 ? build.version.slice(0, 8) : build.version}
                          </span>
                          {idx === 0 && (
                            <span className="text-[10px] text-[var(--primary-hover)]">latest</span>
                          )}
                        </div>
                        <span className="w-[140px] text-[12px] text-[var(--text-dim)]">
                          {formatDate(build.created_at)}
                        </span>
                        <span className="w-[180px] text-[11px] text-[var(--primary-hover)] font-mono">
                          {abbreviateHash(build.image_hash)}
                        </span>
                        <div className="w-[80px]">
                          <span className={`px-2 py-0.5 rounded text-[11px] font-medium ${
                            build.status === 'success'
                              ? 'text-[var(--severity-success)] bg-[var(--severity-success)]/20'
                              : build.status === 'failed'
                              ? 'text-[var(--severity-error)] bg-[var(--severity-error)]/20'
                              : 'text-[var(--text-muted)] bg-[var(--bg-elevated)]'
                          }`}>
                            {build.status === 'success' ? 'Success' : build.status === 'failed' ? 'Failed' : build.status}
                          </span>
                        </div>
                        <span className="w-[80px] text-[12px] text-[var(--text-dim)]">
                          {formatSize(build.size_bytes)}
                        </span>
                        <button
                          onClick={() => handleConsole(build.build_id, df.slug)}
                          className="flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium text-[var(--text-muted)] border border-[var(--border-primary)] bg-transparent cursor-pointer hover:border-[var(--border-secondary)] transition-colors"
                          style={{ fontFamily: 'Inter, sans-serif' }}
                        >
                          <Terminal size={12} />
                          Console
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
      {showAddDialog && (
        <AddDockerDialog
          serverDockerfiles={serverDockerfiles}
          builtSlugs={builtSlugs}
          onClose={() => setShowAddDialog(false)}
          onAdd={handleAddDockerfiles}
        />
      )}
      {logPanel && (
        <div className="fixed inset-0 z-50 flex items-start justify-center pt-[120px]">
          <div className="absolute inset-0 bg-black/50" onClick={() => setLogPanel(null)} />
          <div className="relative z-10 w-[720px] max-h-[70vh] rounded-xl bg-[var(--bg-primary)] border border-[var(--border-primary)] flex flex-col overflow-hidden shadow-2xl" style={{ fontFamily: 'Inter, sans-serif' }}>
            <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-primary)]">
              <div className="flex items-center gap-3">
                <Terminal size={16} className="text-[var(--text-muted)]" />
                <span className="text-sm font-medium text-[var(--text-primary)]">{logPanel.slug}</span>
                {logPanel.status && (
                  <span className={`px-2 py-0.5 rounded text-[11px] font-medium ${
                    logPanel.status === 'success' ? 'text-[var(--severity-success)] bg-[var(--severity-success)]/20'
                    : logPanel.status === 'failed' ? 'text-[var(--severity-error)] bg-[var(--severity-error)]/20'
                    : 'text-[var(--text-muted)] bg-[var(--bg-elevated)]'
                  }`}>{logPanel.status}</span>
                )}
              </div>
              <button onClick={() => setLogPanel(null)} className="flex items-center justify-center w-8 h-8 rounded-md bg-transparent border-none cursor-pointer text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors">
                <X size={18} />
              </button>
            </div>
            <div className="flex-1 overflow-y-auto p-4">
              {logPanel.loading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="animate-spin text-[var(--text-muted)]" size={20} />
                </div>
              ) : logPanel.log ? (
                <pre className="text-[12px] leading-5 text-[var(--text-primary)] whitespace-pre-wrap font-mono m-0 bg-[var(--bg-secondary)] rounded-lg p-4">{logPanel.log}</pre>
              ) : (
                <p className="text-[13px] text-[var(--text-muted)] italic text-center py-8">No logs available for this build.</p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function AddDockerDialog({
  serverDockerfiles,
  builtSlugs,
  onClose,
  onAdd,
}: {
  serverDockerfiles: DockerfileResponse[];
  builtSlugs: Set<string>;
  onClose: () => void;
  onAdd: (slugs: string[]) => void;
}) {
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const filtered = serverDockerfiles.filter(df => {
    const q = search.toLowerCase();
    return !q || df.name.toLowerCase().includes(q) || df.slug.toLowerCase().includes(q);
  });

  const toggleSelect = (slug: string) => {
    if (builtSlugs.has(slug)) return;
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(slug)) next.delete(slug); else next.add(slug);
      return next;
    });
  };

  const handleAdd = () => {
    if (selected.size === 0) return;
    onAdd(Array.from(selected));
  };

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[200px]">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div
        data-qa="add-docker-dialog"
        className="relative z-10 w-[520px] rounded-xl bg-[var(--bg-primary)] border border-[var(--border-primary)] flex flex-col overflow-hidden shadow-2xl"
        style={{ fontFamily: 'Inter, sans-serif' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-5 border-b border-[var(--border-primary)]">
          <div className="flex flex-col gap-1">
            <h3 className="text-lg font-semibold text-[var(--text-primary)] m-0">Add Dockerfile</h3>
            <p className="text-[13px] text-[var(--text-muted)] m-0">Select a dockerfile to build on this node</p>
          </div>
          <button
            onClick={onClose}
            data-qa="close-add-docker-dialog"
            className="flex items-center justify-center w-8 h-8 rounded-md bg-transparent border-none cursor-pointer text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Search */}
        <div className="px-6 py-4">
          <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border-primary)]">
            <Search size={16} className="text-[var(--text-muted)] shrink-0" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search dockerfiles..."
              data-qa="search-dockerfiles-input"
              className="flex-1 bg-transparent border-none outline-none text-[13px] text-[var(--text-primary)] placeholder-[var(--text-muted)]"
              autoFocus
            />
          </div>
        </div>

        {/* List */}
        <div className="flex flex-col px-6 max-h-[340px] overflow-y-auto">
          {filtered.map(df => {
            const alreadyAdded = builtSlugs.has(df.slug);
            const isSelected = selected.has(df.slug);
            return (
              <button
                key={df.id}
                onClick={() => toggleSelect(df.slug)}
                disabled={alreadyAdded}
                data-qa={`dockerfile-item-${df.slug}`}
                className={`flex items-center gap-3 w-full px-4 py-3 rounded-lg bg-transparent border cursor-pointer transition-colors text-left ${
                  alreadyAdded
                    ? 'opacity-50 cursor-default border-transparent'
                    : isSelected
                    ? 'bg-[var(--primary)]/10 border-[var(--primary)]'
                    : 'border-transparent hover:bg-[var(--bg-tertiary)]'
                }`}
                style={{ fontFamily: 'Inter, sans-serif' }}
              >
                {/* Checkbox */}
                <div className={`flex items-center justify-center w-5 h-5 rounded shrink-0 ${
                  alreadyAdded || isSelected
                    ? 'bg-[var(--primary)]'
                    : 'border border-[var(--border-primary)]'
                }`}>
                  {(alreadyAdded || isSelected) && <Check size={12} className="text-[var(--primary-text)]" />}
                </div>
                <Container size={18} className={`shrink-0 ${isSelected ? 'text-[var(--primary)]' : 'text-[var(--text-muted)]'}`} />
                <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                  <span className="text-[13px] font-medium text-[var(--text-primary)] truncate">{df.name}</span>
                  <span className="text-[11px] text-[var(--text-muted)] truncate">
                    {df.slug} &middot; {df.version}{alreadyAdded ? ' &middot; already built' : ''}
                  </span>
                </div>
              </button>
            );
          })}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t border-[var(--border-primary)] mt-auto">
          <span className="text-[13px] text-[var(--text-muted)]">
            {selected.size} dockerfile{selected.size !== 1 ? 's' : ''} selected
          </span>
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              data-qa="cancel-add-docker-btn"
              className="px-5 py-2.5 rounded-lg text-[13px] font-medium text-[var(--text-muted)] bg-transparent border border-[var(--border-primary)] cursor-pointer hover:text-[var(--text-primary)] hover:border-[var(--border-secondary)] transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleAdd}
              disabled={selected.size === 0}
              data-qa="confirm-add-docker-btn"
              className="flex items-center gap-1.5 px-5 py-2.5 rounded-lg text-[13px] font-medium bg-[var(--primary)] text-[var(--primary-text)] hover:bg-[var(--primary-hover)] transition-colors cursor-pointer border-none disabled:opacity-50"
            >
              <Plus size={14} />
              Add & Build
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

type TaskStatus = 'completed' | 'failed' | 'cancelled' | 'running' | 'blocked' | 'todo';

interface PastTask {
  id: string;
  title: string;
  initiator: string;
  project: string;
  feature: string;
  started: string;
  timeSpent: string;
  estCost: string;
  status: TaskStatus;
  restricted?: boolean;
  costDetails?: {
    tokensIn: string;
    tokensOut: string;
    cacheRead: string;
    cacheWrite: string;
    subagents: number;
  };
}

interface CurrentTask {
  id: string;
  title: string;
  initiator: string;
  project: string;
  feature: string;
  elapsed: string;
  progress: number;
}

function TasksTab({ nodeId: _nodeId }: { nodeId: string }) {
  const [searchQuery, setSearchQuery] = useState('');
  const [tooltipTask, setTooltipTask] = useState<string | null>(null);

  // Placeholder data matching pen design
  const currentTasks: CurrentTask[] = [
    { id: '1', title: 'Implement user authentication', initiator: 'Jeremy L.', project: 'agach-mcp', feature: 'Auth System', elapsed: '24m', progress: 100 },
    { id: '2', title: 'Generate API documentation', initiator: 'Sarah K.', project: 'docs-platform', feature: 'API Docs v2', elapsed: '8m', progress: 45 },
  ];

  const pastTasks: PastTask[] = [
    {
      id: 'p1', title: 'Implement user authentication', initiator: 'Jeremy L.', project: 'agach-mcp', feature: 'Auth System',
      started: 'Mar 24, 09:14', timeSpent: '3h 42m', estCost: '$4.82', status: 'completed',
      costDetails: { tokensIn: '124,832', tokensOut: '18,456', cacheRead: '89,210', cacheWrite: '35,622', subagents: 3 },
    },
    {
      id: 'p2', title: 'Generate API documentation', initiator: 'Sarah K.', project: 'docs-platform', feature: 'API Docs v2',
      started: 'Mar 23, 16:42', timeSpent: '1h 18m', estCost: '$1.56', status: 'completed',
      costDetails: { tokensIn: '52,100', tokensOut: '8,230', cacheRead: '41,000', cacheWrite: '12,500', subagents: 1 },
    },
    {
      id: 'p3', title: 'Restricted', initiator: 'Alex M.', project: 'Restricted', feature: 'Restricted',
      started: 'Mar 22, 11:08', timeSpent: '5h 03m', estCost: '$7.21', status: 'completed', restricted: true,
      costDetails: { tokensIn: '210,400', tokensOut: '32,100', cacheRead: '180,000', cacheWrite: '55,000', subagents: 5 },
    },
    {
      id: 'p4', title: 'Database migration v3.2', initiator: 'Jeremy L.', project: 'agach-mcp', feature: 'DB Migrations',
      started: 'Mar 21, 14:30', timeSpent: '0h 12m', estCost: '$0.18', status: 'failed',
      costDetails: { tokensIn: '8,200', tokensOut: '1,100', cacheRead: '5,000', cacheWrite: '2,000', subagents: 0 },
    },
    {
      id: 'p5', title: 'Restricted', initiator: 'Tom B.', project: 'Restricted', feature: 'Restricted',
      started: 'Mar 20, 08:55', timeSpent: '2h 30m', estCost: '$3.45', status: 'cancelled', restricted: true,
      costDetails: { tokensIn: '95,600', tokensOut: '14,200', cacheRead: '72,000', cacheWrite: '28,000', subagents: 2 },
    },
  ];

  const runningCount = currentTasks.length;

  return (
    <div data-qa="tasks-tab" className="flex flex-col gap-6" style={{ fontFamily: 'Inter, sans-serif' }}>
      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)] w-[300px]">
          <Search size={16} className="text-[var(--text-dim)]" />
          <input
            type="text"
            placeholder="Search tasks..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            data-qa="tasks-search"
            className="bg-transparent border-none outline-none text-[13px] text-[var(--text-primary)] placeholder-[var(--text-dim)] w-full"
            style={{ fontFamily: 'Inter, sans-serif' }}
          />
        </div>
        <button
          className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)] text-[var(--text-dim)] text-[13px] cursor-pointer hover:border-[var(--border-secondary)] transition-colors"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          All statuses
          <ChevronDown size={14} className="text-[var(--text-dim)]" />
        </button>
        <button
          className="flex items-center gap-2 px-4 py-2.5 rounded-lg bg-[var(--bg-primary)] border border-[var(--border-primary)] text-[var(--text-dim)] text-[13px] cursor-pointer hover:border-[var(--border-secondary)] transition-colors"
          style={{ fontFamily: 'Inter, sans-serif' }}
        >
          All initiators
          <ChevronDown size={14} className="text-[var(--text-dim)]" />
        </button>
      </div>

      {/* Current Tasks */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center gap-3">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] m-0">Current Tasks</h3>
          {runningCount > 0 && (
            <span className="px-2 py-1 rounded-md text-xs font-medium text-[var(--primary-hover)] bg-[var(--primary)]/20">
              {runningCount} running
            </span>
          )}
        </div>

        {currentTasks.map((task) => (
          <div
            key={task.id}
            className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-primary)] p-5 flex flex-col gap-4"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="text-sm font-medium text-[var(--text-primary)]">{task.title}</span>
                <span className="px-2 py-1 rounded-md text-xs font-medium text-[var(--primary-hover)] bg-[var(--primary)]/20 flex items-center gap-1">
                  <Clock size={12} />
                  {task.elapsed}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-5">
              <div className="flex items-center gap-1 text-[var(--text-dim)] text-[13px]">
                <User size={12} />
                {task.initiator}
              </div>
              <div className="flex items-center gap-1 text-[var(--text-dim)] text-[13px]">
                <FolderOpen size={12} />
                {task.project}
              </div>
              <div className="flex items-center gap-1 text-[var(--text-dim)] text-[13px]">
                <GitBranch size={12} />
                {task.feature}
              </div>
            </div>
            <div className="w-full h-1 rounded-sm bg-[var(--border-primary)]">
              <div
                className="h-full rounded-sm bg-[var(--primary)]"
                style={{ width: `${task.progress}%` }}
              />
            </div>
          </div>
        ))}
      </div>

      {/* Past Tasks */}
      <div className="flex flex-col gap-4">
        <div className="flex items-center gap-3">
          <h3 className="text-lg font-semibold text-[var(--text-primary)] m-0">Past Tasks</h3>
          <span className="px-2 py-1 rounded-md text-xs font-medium text-[var(--text-muted)] bg-[var(--bg-elevated)]">
            {pastTasks.length} completed
          </span>
        </div>

        <div className="rounded-xl bg-[var(--bg-secondary)] border border-[var(--border-primary)] overflow-hidden">
          {/* Table header */}
          <div className="flex items-center px-5 py-3 bg-[var(--bg-elevated)]">
            <div className="w-[200px] text-xs font-medium text-[var(--text-dim)]">TASK</div>
            <div className="w-[100px] text-xs font-medium text-[var(--text-dim)]">INITIATOR</div>
            <div className="w-[110px] text-xs font-medium text-[var(--text-dim)]">PROJECT</div>
            <div className="w-[110px] text-xs font-medium text-[var(--text-dim)]">FEATURE</div>
            <div className="w-[110px] text-xs font-medium text-[var(--text-dim)]">STARTED</div>
            <div className="w-[80px] text-xs font-medium text-[var(--text-dim)]">TIME SPENT</div>
            <div className="w-[100px] text-xs font-medium text-[var(--text-dim)] flex items-center gap-1.5">
              EST. COST
              <Info size={12} className="text-[var(--text-dim)]" />
            </div>
            <div className="flex-1 text-xs font-medium text-[var(--text-dim)]">STATUS</div>
          </div>

          {/* Table rows */}
          {pastTasks.map((task, idx) => (
            <div
              key={task.id}
              className={`flex items-center px-5 py-4 relative ${idx < pastTasks.length - 1 ? 'border-b border-[var(--border-primary)]' : ''}`}
            >
              <div className="w-[200px]">
                {task.restricted ? (
                  <span className="text-[var(--severity-error)]/50 text-[13px] italic flex items-center gap-1.5">
                    <span className="text-[var(--severity-error)]">🔒</span> Restricted
                  </span>
                ) : (
                  <span className="text-[13px] font-medium text-[var(--text-primary)]">{task.title}</span>
                )}
              </div>
              <div className="w-[100px] text-[13px] text-[var(--text-dim)]">{task.initiator}</div>
              <div className="w-[110px]">
                {task.restricted && task.project === 'Restricted' ? (
                  <span className="text-[var(--severity-error)]/50 text-[13px] italic flex items-center gap-1.5">
                    <span className="text-[var(--severity-error)]">🔒</span> Restricted
                  </span>
                ) : (
                  <span className="text-[13px] text-[var(--text-dim)]">{task.project}</span>
                )}
              </div>
              <div className="w-[110px]">
                {task.restricted && task.feature === 'Restricted' ? (
                  <span className="text-[var(--severity-error)]/50 text-[13px] italic flex items-center gap-1.5">
                    <span className="text-[var(--severity-error)]">🔒</span> Restricted
                  </span>
                ) : (
                  <span className="text-[13px] text-[var(--text-dim)]">{task.feature}</span>
                )}
              </div>
              <div className="w-[110px] text-[13px] text-[var(--text-dim)]">{task.started}</div>
              <div className="w-[80px] text-[13px] text-[var(--text-primary)]">{task.timeSpent}</div>
              <div className="w-[100px] relative">
                <div className="flex items-center gap-1.5">
                  <span className="text-[13px] text-[var(--text-primary)]">{task.estCost}</span>
                  <button
                    className="text-[var(--text-dim)] hover:text-[var(--text-muted)] transition-colors bg-transparent border-none cursor-pointer p-0"
                    onMouseEnter={() => setTooltipTask(task.id)}
                    onMouseLeave={() => setTooltipTask(null)}
                  >
                    <Info size={13} />
                  </button>
                </div>
                {tooltipTask === task.id && task.costDetails && (
                  <div className="absolute left-0 top-full mt-2 z-50 w-[220px] rounded-lg bg-[var(--bg-elevated)] border border-[var(--border-secondary)] p-4 flex flex-col gap-3 shadow-lg">
                    <span className="text-[13px] font-semibold text-[var(--text-primary)]">Cost Breakdown</span>
                    <div className="w-full h-px bg-[var(--border-secondary)]" />
                    <div className="flex justify-between">
                      <span className="text-xs text-[var(--text-dim)]">Tokens In</span>
                      <span className="text-xs font-medium text-[var(--text-primary)]">{task.costDetails.tokensIn}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-xs text-[var(--text-dim)]">Tokens Out</span>
                      <span className="text-xs font-medium text-[var(--text-primary)]">{task.costDetails.tokensOut}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-xs text-[var(--text-dim)]">Cache Read</span>
                      <span className="text-xs font-medium text-[var(--text-primary)]">{task.costDetails.cacheRead}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-xs text-[var(--text-dim)]">Cache Write</span>
                      <span className="text-xs font-medium text-[var(--text-primary)]">{task.costDetails.cacheWrite}</span>
                    </div>
                    <div className="w-full h-px bg-[var(--border-secondary)]" />
                    <div className="flex justify-between">
                      <span className="text-xs text-[var(--text-dim)]">Subagents</span>
                      <span className="text-xs font-medium text-[var(--text-primary)]">{task.costDetails.subagents}</span>
                    </div>
                  </div>
                )}
              </div>
              <div className="flex-1">
                {statusBadge(task.status)}
              </div>
            </div>
          ))}
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between py-3">
          <span className="text-[13px] text-[var(--text-dim)]">
            Showing 1-{pastTasks.length} of {pastTasks.length} tasks
          </span>
          <div className="flex items-center gap-2">
            <button className="flex items-center justify-center px-3 py-1.5 rounded-md bg-[var(--primary)] text-[var(--primary-text)] text-[13px] font-medium cursor-pointer border-none">
              1
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function statusBadge(status: TaskStatus) {
  const config: Record<TaskStatus, { varName: string; label: string }> = {
    completed: { varName: 'success', label: 'Completed' },
    failed: { varName: 'error', label: 'Failed' },
    cancelled: { varName: 'warning', label: 'Cancelled' },
    running: { varName: 'info', label: 'Running' },
    blocked: { varName: 'error', label: 'Blocked' },
    todo: { varName: 'info', label: 'To Do' },
  };
  const c = config[status];
  return (
    <span
      className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium"
      style={{
        color: `var(--severity-${c.varName})`,
        backgroundColor: `var(--severity-${c.varName}-bg)`,
        fontFamily: 'Inter, sans-serif',
      }}
    >
      {c.label}
    </span>
  );
}
