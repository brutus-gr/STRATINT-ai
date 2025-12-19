import { useState, useEffect } from 'react';
import {
  Terminal,
  LogOut,
  Database,
  Activity,
  Sliders,
  FileCheck,
  TrendingUp,
  Globe,
  AlertTriangle,
  ListChecks,
  Brain,
  Trash2,
  Twitter,
  Menu,
  X,
  Zap
} from 'lucide-react';
import { TrackedSourcesTab } from './TrackedSourcesTab';
import { ConfigModal } from './ConfigModal';
import { IngestionErrorsTab } from './IngestionErrorsTab';
import { ActivityLogsTab } from './ActivityLogsTab';
import { OpenAIConfigTab } from './OpenAIConfigTab';
import { PipelineFunnelTab } from './PipelineFunnelTab';
import { TwitterSettingsTab } from './TwitterSettingsTab';
import { SourceTrackingTab } from './SourceTrackingTab';
import { ForecastsTab } from './ForecastsTab';
import { StrategiesTab } from './StrategiesTab';
import { SummariesTab } from './SummariesTab';
import { PostedTweetsTab } from './PostedTweetsTab';
import { InferenceLogsTab } from './InferenceLogsTab';
import { formatDateTime } from '../utils/dateFormat';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface AdminDashboardProps {
  onLogout: () => void;
}

type Tab = 'overview' | 'pipeline' | 'events' | 'sources' | 'tracking' | 'forecasts' | 'strategies' | 'summaries' | 'connectors' | 'thresholds' | 'ai' | 'errors' | 'activity' | 'twitter' | 'posted-tweets' | 'inference-logs';

interface SystemStats {
  total_events: number;
  total_sources: number;
  tracked_accounts: number;
  avg_confidence: number;
  avg_magnitude: number;
  category_counts: Record<string, number>;
  uptime_seconds: number;
  uptime_formatted: string;
}

export function AdminDashboard({ onLogout }: AdminDashboardProps) {
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [stats, setStats] = useState<SystemStats>({
    total_events: 0,
    total_sources: 0,
    tracked_accounts: 0,
    avg_confidence: 0,
    avg_magnitude: 0,
    category_counts: {},
    uptime_seconds: 0,
    uptime_formatted: '00:00:00',
  });

  // Fetch stats from API
  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/stats`);
        if (!response.ok) throw new Error('Failed to fetch stats');
        const data = await response.json();
        setStats(data);
      } catch (err) {
        console.error('Error fetching stats:', err);
      }
    };

    fetchStats();
    // Refresh every 5 seconds
    const interval = setInterval(fetchStats, 5000);
    return () => clearInterval(interval);
  }, []);

  const tabs = [
    { id: 'overview' as Tab, label: 'OVERVIEW', icon: Activity },
    { id: 'pipeline' as Tab, label: 'PIPELINE', icon: TrendingUp },
    { id: 'events' as Tab, label: 'EVENTS', icon: FileCheck },
    { id: 'sources' as Tab, label: 'SOURCES', icon: Globe },
    { id: 'tracking' as Tab, label: 'ENRICHMENT TRACKING', icon: Zap },
    { id: 'forecasts' as Tab, label: 'FORECASTS', icon: TrendingUp },
    { id: 'strategies' as Tab, label: 'STRATEGIES', icon: Brain },
    { id: 'summaries' as Tab, label: 'SUMMARIES', icon: FileCheck },
    { id: 'connectors' as Tab, label: 'CONNECTORS', icon: Database },
    { id: 'thresholds' as Tab, label: 'THRESHOLDS', icon: Sliders },
    { id: 'ai' as Tab, label: 'OPENAI', icon: Brain },
    { id: 'twitter' as Tab, label: 'TWITTER / X', icon: Terminal },
    { id: 'posted-tweets' as Tab, label: 'POSTED TWEETS', icon: ListChecks },
    { id: 'inference-logs' as Tab, label: 'INFERENCE LOGS', icon: Activity },
    { id: 'activity' as Tab, label: 'ACTIVITY LOGS', icon: ListChecks },
    { id: 'errors' as Tab, label: 'ERRORS', icon: AlertTriangle },
  ];

  return (
    <div className="min-h-screen bg-void text-chalk">
      {/* Scan line effect */}
      <div className="scan-line" />
      
      {/* Header */}
      <header className="fixed top-0 left-0 right-0 z-50 border-b-2 border-steel bg-void/98 backdrop-blur-md">
        <div className="px-4 md:px-6 py-3 flex items-center justify-between">
          {/* Logo */}
          <div className="flex items-center gap-2 md:gap-4">
            {/* Mobile Menu Toggle */}
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="lg:hidden p-2 border border-steel hover:border-terminal transition-colors"
              aria-label="Toggle menu"
            >
              {mobileMenuOpen ? (
                <X className="w-5 h-5 text-terminal" />
              ) : (
                <Menu className="w-5 h-5 text-terminal" />
              )}
            </button>

            <Terminal className="w-5 md:w-6 h-5 md:h-6 text-terminal animate-pulse-slow" />
            <h1 className="text-base md:text-xl font-display font-black text-white tracking-tight">
              STRATINT
            </h1>
            <span className="hidden sm:inline-block px-2 md:px-3 py-1 border-2 border-terminal bg-terminal/10 text-terminal text-xs font-mono font-bold">
              ADMIN
            </span>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2 md:gap-4">
            <div className="hidden md:flex items-center gap-2 px-4 py-2 border border-steel bg-concrete">
              <Activity className="w-4 h-4 text-terminal animate-pulse-slow" />
              <span className="text-xs font-mono text-terminal font-bold">AUTHENTICATED</span>
            </div>
            <button
              onClick={onLogout}
              className="flex items-center gap-1 md:gap-2 px-2 md:px-4 py-2 border-2 border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all font-mono text-xs font-bold"
            >
              <LogOut className="w-4 h-4" />
              <span className="hidden sm:inline">LOGOUT</span>
            </button>
          </div>
        </div>
      </header>

      {/* Main Layout */}
      <div className="pt-16 flex">
        {/* Mobile Menu Overlay */}
        {mobileMenuOpen && (
          <div
            className="fixed inset-0 bg-void/80 z-40 lg:hidden"
            onClick={() => setMobileMenuOpen(false)}
          />
        )}

        {/* Sidebar Navigation */}
        <aside className={`
          fixed lg:sticky top-16 left-0 z-40
          w-64 h-[calc(100vh-4rem)]
          border-r-2 border-steel bg-concrete/95 lg:bg-concrete/30 p-4
          transition-transform duration-300 ease-in-out
          ${mobileMenuOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
        `}>
          <nav className="space-y-2">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => {
                    setActiveTab(tab.id);
                    setMobileMenuOpen(false);
                  }}
                  className={`w-full flex items-center gap-3 px-4 py-3 font-mono text-sm transition-all ${
                    isActive
                      ? 'border-2 border-terminal bg-terminal/10 text-terminal font-bold'
                      : 'border border-steel bg-void text-fog hover:border-iron hover:text-chalk'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                  {tab.label}
                </button>
              );
            })}
          </nav>

          {/* System Status */}
          <div className="mt-8 p-4 border-2 border-steel bg-void">
            <h3 className="text-xs font-mono text-smoke font-bold mb-3">SYSTEM STATUS</h3>
            <div className="space-y-2 text-xs font-mono">
              <div className="flex justify-between">
                <span className="text-fog">Uptime:</span>
                <span className="text-chalk">{stats.uptime_formatted}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-fog">Events:</span>
                <span className="text-terminal font-bold">{stats.total_events.toLocaleString()}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-fog">Sources:</span>
                <span className="text-terminal font-bold">{stats.total_sources.toLocaleString()}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-fog">Tracked:</span>
                <span className="text-terminal font-bold">{stats.tracked_accounts}</span>
              </div>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 p-4 md:p-8">
          {activeTab === 'overview' && <OverviewTab stats={stats} />}
          {activeTab === 'pipeline' && <PipelineFunnelTab />}
          {activeTab === 'events' && <EventsTab />}
          {activeTab === 'sources' && <TrackedSourcesTab />}
          {activeTab === 'tracking' && <SourceTrackingTab />}
          {activeTab === 'forecasts' && <ForecastsTab />}
          {activeTab === 'strategies' && <StrategiesTab />}
          {activeTab === 'summaries' && <SummariesTab />}
          {activeTab === 'connectors' && <ConnectorsTab />}
          {activeTab === 'thresholds' && <ThresholdsTab />}
          {activeTab === 'ai' && <OpenAIConfigTab />}
          {activeTab === 'twitter' && <TwitterSettingsTab />}
          {activeTab === 'posted-tweets' && <PostedTweetsTab />}
          {activeTab === 'inference-logs' && <InferenceLogsTab />}
          {activeTab === 'activity' && <ActivityLogsTab />}
          {activeTab === 'errors' && <IngestionErrorsTab />}
        </main>
      </div>
    </div>
  );
}

interface ActivityLog {
  id: string;
  timestamp: string;
  activity_type: string;
  platform?: string;
  message: string;
  details?: Record<string, any>;
  source_count?: number;
  duration_ms?: number;
}

function OverviewTab({ stats }: { stats: SystemStats }) {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [isDeleting, setIsDeleting] = useState(false);
  const [recentActivity, setRecentActivity] = useState<ActivityLog[]>([]);

  // Fetch recent activity logs
  useEffect(() => {
    const fetchActivity = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/activity-logs?limit=5`);
        if (!response.ok) throw new Error('Failed to fetch activity logs');
        const data = await response.json();
        setRecentActivity(data.logs || []);
      } catch (err) {
        console.error('Error fetching activity logs:', err);
      }
    };

    fetchActivity();
    // Refresh every 5 seconds
    const interval = setInterval(fetchActivity, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleDeleteAll = async () => {
    if (deleteConfirmText !== 'DELETE ALL DATA') {
      alert('Please type "DELETE ALL DATA" to confirm');
      return;
    }

    setIsDeleting(true);
    try {
      const token = localStorage.getItem('admin_token');
      const response = await fetch(`${API_BASE_URL}/api/admin/delete-all`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      const result = await response.json();
      alert(`Success: ${result.message}\n\nDeleted ${result.events_deleted} events and ${result.sources_deleted} sources.`);
      setShowDeleteModal(false);
      setDeleteConfirmText('');

      // Reload page to refresh stats
      window.location.reload();
    } catch (err) {
      console.error('Error deleting data:', err);
      alert(`Failed to delete data: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          SYSTEM OVERVIEW
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Real-time platform metrics and status
        </p>
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6">
        <MetricCard
          icon={Activity}
          label="ACTIVE EVENTS"
          value={stats.total_events.toLocaleString()}
          trend=""
          color="text-terminal"
        />
        <MetricCard
          icon={Database}
          label="DATA SOURCES"
          value={stats.total_sources.toLocaleString()}
          trend=""
          color="text-electric"
        />
        <MetricCard
          icon={TrendingUp}
          label="AVG CONFIDENCE"
          value={stats.avg_confidence.toFixed(2)}
          trend=""
          color="text-warning"
        />
      </div>

      {/* Recent Activity */}
      <div className="border-2 border-steel bg-concrete">
        <div className="px-4 md:px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-sm md:text-base text-chalk">RECENT ACTIVITY</h3>
        </div>
        <div className="p-4 md:p-6 space-y-3">
          {recentActivity.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-sm font-mono text-fog">No recent activity</p>
            </div>
          ) : (
            recentActivity.map((activity) => (
              <div key={activity.id} className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 p-3 border border-steel bg-void hover:border-iron transition-colors">
                <span className="text-xs font-mono text-terminal font-bold">{formatDateTime(activity.timestamp)}</span>
                <div className="flex-1">
                  <p className="text-sm font-mono text-chalk">{activity.activity_type.replace(/_/g, ' ').toUpperCase()}</p>
                  <p className="text-xs font-mono text-fog mt-0.5">{activity.message}</p>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Danger Zone */}
      <div className="border-2 border-threat-critical bg-concrete">
        <div className="px-4 md:px-6 py-4 border-b-2 border-threat-critical bg-threat-critical/10">
          <h3 className="font-display font-black text-sm md:text-base text-threat-critical flex items-center gap-2">
            <AlertTriangle className="w-5 h-5" />
            DANGER ZONE
          </h3>
        </div>
        <div className="p-4 md:p-6">
          <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
            <div className="flex-1">
              <h4 className="font-mono font-bold text-chalk text-sm">DELETE ALL DATA</h4>
              <p className="text-xs font-mono text-fog mt-2">
                Permanently delete all events and sources from the database. This action cannot be undone.
              </p>
              <p className="text-xs font-mono text-threat-critical mt-2 font-bold">
                ⚠ WARNING: This will delete {stats.total_events.toLocaleString()} events and {stats.total_sources.toLocaleString()} sources.
              </p>
            </div>
            <button
              onClick={() => setShowDeleteModal(true)}
              className="flex items-center justify-center gap-2 px-6 py-3 border-2 border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all font-mono text-sm font-bold whitespace-nowrap"
            >
              <Trash2 className="w-4 h-4" />
              DELETE ALL
            </button>
          </div>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4">
          <div className="max-w-2xl w-full border-4 border-threat-critical bg-concrete">
            <div className="px-4 md:px-6 py-4 border-b-4 border-threat-critical bg-threat-critical/20">
              <h3 className="font-display font-black text-lg md:text-2xl text-threat-critical flex items-center gap-3">
                <AlertTriangle className="w-6 md:w-8 h-6 md:h-8" />
                CONFIRM DATA DELETION
              </h3>
            </div>
            <div className="p-4 md:p-8 space-y-6">
              <div className="p-4 border-2 border-threat-critical bg-threat-critical/10">
                <p className="font-mono text-sm text-chalk font-bold">⚠ THIS ACTION IS IRREVERSIBLE ⚠</p>
              </div>

              <div className="space-y-2 font-mono text-sm text-chalk">
                <p>You are about to permanently delete:</p>
                <ul className="list-disc list-inside pl-4 space-y-1 text-threat-critical font-bold">
                  <li>{stats.total_events.toLocaleString()} events</li>
                  <li>{stats.total_sources.toLocaleString()} sources</li>
                  <li>All associated metadata and relationships</li>
                </ul>
              </div>

              <div className="space-y-3">
                <label className="block text-sm font-mono text-chalk font-bold">
                  Type "DELETE ALL DATA" to confirm:
                </label>
                <input
                  type="text"
                  value={deleteConfirmText}
                  onChange={(e) => setDeleteConfirmText(e.target.value)}
                  placeholder="DELETE ALL DATA"
                  className="w-full px-4 py-3 border-2 border-steel bg-void text-chalk font-mono focus:border-threat-critical focus:outline-none"
                  autoFocus
                />
              </div>

              <div className="flex gap-4">
                <button
                  onClick={() => {
                    setShowDeleteModal(false);
                    setDeleteConfirmText('');
                  }}
                  disabled={isDeleting}
                  className="flex-1 px-6 py-3 border-2 border-steel text-chalk hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  CANCEL
                </button>
                <button
                  onClick={handleDeleteAll}
                  disabled={isDeleting || deleteConfirmText !== 'DELETE ALL DATA'}
                  className="flex-1 px-6 py-3 border-2 border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isDeleting ? 'DELETING...' : 'DELETE ALL DATA'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function EventsTab() {
  const [events, setEvents] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<string>('all');

  useEffect(() => {
    const fetchEvents = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/events?limit=50&sort_by=timestamp&sort_order=desc`);
        if (!response.ok) throw new Error('Failed to fetch events');
        const data = await response.json();
        setEvents(data.events || []);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching events:', err);
        setLoading(false);
      }
    };

    fetchEvents();
    const interval = setInterval(fetchEvents, 10000);
    return () => clearInterval(interval);
  }, []);

  const filteredEvents = statusFilter === 'all'
    ? events
    : events.filter(e => e.status === statusFilter);

  const handleAction = async (eventId: string, action: string) => {
    try {
      // Handle special Twitter action
      if (action === 'post-to-twitter') {
        const response = await fetch(`${API_BASE_URL}/api/events/${eventId}/post-to-twitter`, {
          method: 'POST',
          headers: getAuthHeaders(),
        });

        if (!response.ok) {
          const error = await response.text();
          throw new Error(error);
        }

        const result = await response.json();
        alert(result.message);

        // Refresh events list
        const eventsResponse = await fetch(`${API_BASE_URL}/api/events?limit=50&sort_by=timestamp&sort_order=desc`);
        if (eventsResponse.ok) {
          const data = await eventsResponse.json();
          setEvents(data.events || []);
        }
        return;
      }

      // Map action verbs to status values (publish -> published, reject -> rejected, archive -> archived)
      const statusMap: Record<string, string> = {
        'publish': 'published',
        'reject': 'rejected',
        'archive': 'archived'
      };
      const status = statusMap[action] || action;

      const response = await fetch(`${API_BASE_URL}/api/events/${eventId}/status`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: JSON.stringify({ status }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      // Refresh events list
      const eventsResponse = await fetch(`${API_BASE_URL}/api/events?limit=50&sort_by=timestamp&sort_order=desc`);
      if (eventsResponse.ok) {
        const data = await eventsResponse.json();
        setEvents(data.events || []);
      }
    } catch (err) {
      console.error(`Error ${action}ing event:`, err);
      alert(`Failed to ${action} event: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          EVENT MODERATION
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Review, approve, and manage intelligence events
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        {['all', 'pending', 'enriched', 'published', 'archived', 'rejected'].map(status => (
          <button
            key={status}
            onClick={() => setStatusFilter(status)}
            className={`px-3 md:px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
              statusFilter === status
                ? 'border-terminal bg-terminal text-void'
                : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
            }`}
          >
            <span className="hidden sm:inline">{status}</span>
            <span className="sm:hidden">{status.slice(0, 3)}</span>
            {' '}({status === 'all' ? events.length : events.filter(e => e.status === status).length})
          </button>
        ))}
      </div>

      {/* Events Table */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <FileCheck className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-pulse" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING EVENTS...</p>
        </div>
      ) : filteredEvents.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <FileCheck className="w-16 h-16 text-steel/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO EVENTS FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            {statusFilter === 'all'
              ? 'Sources are being ingested. Events will appear once enriched.'
              : `No events with status "${statusFilter}"`
            }
          </p>
        </div>
      ) : (
        <div className="border-2 border-steel bg-concrete overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[640px]">
              <thead className="bg-void border-b-2 border-steel">
                <tr className="text-left">
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">TIMESTAMP</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">TITLE</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">CATEGORY</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">MAG</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">CONF</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">STATUS</th>
                  <th className="px-2 md:px-4 py-3 text-xs font-mono font-bold text-smoke">ACTIONS</th>
                </tr>
              </thead>
              <tbody>
                {filteredEvents.map((event, idx) => (
                  <tr key={event.id} className={`border-b border-steel hover:bg-void/50 transition-colors ${
                    idx % 2 === 0 ? 'bg-void/20' : ''
                  }`}>
                    <td className="px-2 md:px-4 py-3 text-xs font-mono text-fog">
                      {formatDateTime(event.timestamp)}
                    </td>
                    <td className="px-2 md:px-4 py-3 text-xs md:text-sm font-mono text-chalk max-w-xs truncate">
                      {event.title}
                    </td>
                    <td className="px-2 md:px-4 py-3">
                      <span className="px-2 py-1 text-xs font-mono font-bold border border-steel text-terminal uppercase">
                        {event.category.slice(0, 3)}
                      </span>
                    </td>
                    <td className="px-2 md:px-4 py-3 text-sm font-mono font-bold text-warning">
                      {event.magnitude.toFixed(1)}
                    </td>
                    <td className="px-2 md:px-4 py-3 text-sm font-mono font-bold text-electric">
                      {event.confidence.score.toFixed(2)}
                    </td>
                    <td className="px-2 md:px-4 py-3">
                      <span className={`px-2 py-1 text-xs font-mono font-bold border uppercase ${
                        event.status === 'published' ? 'border-terminal text-terminal' :
                        event.status === 'pending' ? 'border-warning text-warning' :
                        event.status === 'enriched' ? 'border-electric text-electric' :
                        event.status === 'archived' ? 'border-steel text-steel' :
                        'border-threat-critical text-threat-critical'
                      }`}>
                        {event.status.slice(0, 3)}
                      </span>
                    </td>
                    <td className="px-2 md:px-4 py-3">
                      <div className="flex flex-wrap gap-1 md:gap-2">
                        {event.status !== 'published' && (
                          <button
                            onClick={() => handleAction(event.id, 'publish')}
                            className="px-2 py-1 text-xs font-mono font-bold border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all"
                          >
                            PUB
                          </button>
                        )}
                        {event.status !== 'archived' && (
                          <button
                            onClick={() => handleAction(event.id, 'archive')}
                            className="px-2 py-1 text-xs font-mono font-bold border border-steel text-steel hover:bg-steel hover:text-void transition-all"
                          >
                            ARC
                          </button>
                        )}
                        {event.status !== 'rejected' && (
                          <button
                            onClick={() => handleAction(event.id, 'reject')}
                            className="px-2 py-1 text-xs font-mono font-bold border border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                          >
                            REJ
                          </button>
                        )}
                        <button
                          onClick={() => handleAction(event.id, 'post-to-twitter')}
                          className="px-2 py-1 text-xs font-mono font-bold border border-electric text-electric hover:bg-electric hover:text-void transition-all flex items-center gap-1"
                          title="Post to X/Twitter"
                        >
                          <Twitter className="w-3 h-3" />
                          <span className="hidden md:inline">POST</span>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

interface Connector {
  id: string;
  name: string;
  enabled: boolean;
  status: string;
}

function ConnectorsTab() {
  const [connectors, setConnectors] = useState<Connector[]>([]);
  const [loading, setLoading] = useState(true);
  const [configModalOpen, setConfigModalOpen] = useState<string | null>(null);

  // Fetch connectors from API
  useEffect(() => {
    const fetchConnectors = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/connectors`, {
          headers: getAuthHeaders(),
        });
        if (!response.ok) throw new Error('Failed to fetch connectors');
        const data = await response.json();
        setConnectors(data.connectors || []);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching connectors:', err);
        setLoading(false);
      }
    };

    fetchConnectors();
  }, []);

  const toggleConnector = async (id: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/connectors/${id}/toggle`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      // Refresh connectors list
      const connectorsResponse = await fetch(`${API_BASE_URL}/api/connectors`, {
        headers: getAuthHeaders(),
      });
      if (connectorsResponse.ok) {
        const data = await connectorsResponse.json();
        setConnectors(data.connectors || []);
      }
    } catch (err) {
      console.error('Error toggling connector:', err);
      alert(`Failed to toggle connector: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleConfigSave = () => {
    // Config saved successfully
    alert('Configuration saved! Restart the server for changes to take effect.');
  };

  return (
    <div className="space-y-6">
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          DATA CONNECTORS
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Manage OSINT data source integrations and credentials
        </p>
      </div>

      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <Globe className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-pulse" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING CONNECTORS...</p>
        </div>
      ) : (
        <div className="space-y-4">
          {connectors.map((connector) => (
            <div key={connector.id} className="border-2 border-steel bg-concrete">
            <div className="p-4 md:p-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
              <div className="flex items-center gap-4">
                <Globe className="w-6 h-6 text-terminal flex-shrink-0" />
                <div>
                  <h3 className="font-mono font-bold text-chalk">{connector.name}</h3>
                  <p className="text-xs font-mono text-fog mt-1">ID: {connector.id}</p>
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 md:gap-4">
                <span className={`px-3 py-1 border-2 text-xs font-mono font-bold ${
                  connector.enabled
                    ? 'border-terminal text-terminal bg-terminal/10'
                    : 'border-steel text-smoke bg-void'
                }`}>
                  {connector.status.toUpperCase()}
                </span>
                <button
                  onClick={() => toggleConnector(connector.id)}
                  className={`px-4 md:px-6 py-2 border-2 font-mono text-sm font-bold transition-all ${
                    connector.enabled
                      ? 'border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void'
                      : 'border-terminal text-terminal hover:bg-terminal hover:text-void'
                  }`}
                >
                  {connector.enabled ? 'DISABLE' : 'ENABLE'}
                </button>
                <button
                  onClick={() => setConfigModalOpen(connector.id)}
                  className="px-4 md:px-6 py-2 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-sm font-bold"
                >
                  CONFIGURE
                </button>
              </div>
            </div>
          </div>
        ))}
        </div>
      )}

      {/* Configuration Modal */}
      {configModalOpen && (
        <ConfigModal
          connectorId={configModalOpen}
          connectorName={connectors.find(c => c.id === configModalOpen)?.name || ''}
          onClose={() => setConfigModalOpen(null)}
          onSave={handleConfigSave}
        />
      )}
    </div>
  );
}

function ThresholdsTab() {
  const [minConfidence, setMinConfidence] = useState(0.1);
  const [minMagnitude, setMinMagnitude] = useState(0.0);
  const [maxSourceAgeHours, setMaxSourceAgeHours] = useState(0);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  // Fetch current thresholds on mount
  useEffect(() => {
    const fetchThresholds = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/thresholds`, {
          headers: getAuthHeaders(),
        });
        if (!response.ok) throw new Error('Failed to fetch thresholds');
        const data = await response.json();
        setMinConfidence(data.min_confidence);
        setMinMagnitude(data.min_magnitude);
        setMaxSourceAgeHours(data.max_source_age_hours || 0);
      } catch (err) {
        console.error('Error fetching thresholds:', err);
      }
    };
    fetchThresholds();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/thresholds`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          min_confidence: minConfidence,
          min_magnitude: minMagnitude,
          max_source_age_hours: maxSourceAgeHours,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      const result = await response.json();
      setMessage({ text: result.message, type: 'success' });
    } catch (err) {
      setMessage({
        text: err instanceof Error ? err.message : 'Failed to save thresholds',
        type: 'error'
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          THRESHOLD CONFIGURATION
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Adjust auto-publish thresholds for event quality control
        </p>
      </div>

      <div className="border-2 border-steel bg-concrete">
        <div className="px-4 md:px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-sm md:text-base text-chalk">AUTO-PUBLISH THRESHOLDS</h3>
        </div>

        <div className="p-4 md:p-8 space-y-6 md:space-y-8">
          {/* Confidence Threshold */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MINIMUM CONFIDENCE</label>
                <p className="text-xs font-mono text-fog mt-1">Events below this threshold require manual review</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">{minConfidence.toFixed(2)}</span>
            </div>
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={minConfidence}
              onChange={(e) => setMinConfidence(parseFloat(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>0.00 (Low)</span>
              <span>1.00 (High)</span>
            </div>
          </div>

          {/* Magnitude Threshold */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MINIMUM MAGNITUDE</label>
                <p className="text-xs font-mono text-fog mt-1">Events below this threshold are filtered out</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">{minMagnitude.toFixed(1)}</span>
            </div>
            <input
              type="range"
              min="0"
              max="10"
              step="0.5"
              value={minMagnitude}
              onChange={(e) => setMinMagnitude(parseFloat(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>0.0 (Low)</span>
              <span>10.0 (Critical)</span>
            </div>
          </div>

          {/* Maximum Source Age */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MAXIMUM SOURCE AGE (HOURS)</label>
                <p className="text-xs font-mono text-fog mt-1">Articles older than this are rejected (0 = no limit)</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">
                {maxSourceAgeHours === 0 ? 'NO LIMIT' : `${maxSourceAgeHours}h`}
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="168"
              step="6"
              value={maxSourceAgeHours}
              onChange={(e) => setMaxSourceAgeHours(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>0 (No Limit)</span>
              <span>168h (7 days)</span>
            </div>
          </div>

          {/* Message Display */}
          {message && (
            <div className={`p-4 border-2 ${
              message.type === 'success'
                ? 'border-terminal bg-terminal/10 text-terminal'
                : 'border-threat-critical bg-threat-critical/10 text-threat-critical'
            } font-mono text-sm`}>
              {message.text}
            </div>
          )}

          <button
            onClick={handleSave}
            disabled={saving}
            className={`w-full py-4 border-2 font-mono text-sm font-bold transition-all ${
              saving
                ? 'border-steel bg-steel text-void cursor-not-allowed'
                : 'border-terminal text-terminal hover:bg-terminal hover:text-void'
            }`}
          >
            {saving ? '[SAVING...]' : '[SAVE CHANGES]'}
          </button>
        </div>
      </div>
    </div>
  );
}


function MetricCard({ icon: Icon, label, value, trend, color }: any) {
  return (
    <div className="border-2 border-steel bg-concrete">
      <div className="p-4 md:p-6">
        <div className="flex items-center gap-3 mb-4">
          <Icon className={`w-5 h-5 ${color}`} />
          <span className="text-xs font-mono text-smoke font-medium">{label}</span>
        </div>
        <div className="flex items-end justify-between">
          <span className="text-2xl md:text-3xl font-mono font-bold text-chalk">{value}</span>
          <span className="text-xs font-mono text-terminal font-bold">{trend}</span>
        </div>
      </div>
    </div>
  );
}

