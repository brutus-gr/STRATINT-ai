import { useState, useEffect } from 'react';
import { Activity, ListChecks } from 'lucide-react';
import { formatDateTime } from '../utils/dateFormat';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

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

export function ActivityLogsTab() {
  const [logs, setLogs] = useState<ActivityLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [activityFilter, setActivityFilter] = useState<string>('');

  const fetchLogs = async () => {
    try {
      const params = new URLSearchParams();
      params.append('limit', '200');
      if (activityFilter) params.append('activity_type', activityFilter);

      const response = await fetch(`${API_BASE_URL}/api/activity-logs?${params}`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch activity logs');
      const data = await response.json();
      setLogs(data.logs || []);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching activity logs:', err);
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
    // Refresh every 5 seconds
    const interval = setInterval(fetchLogs, 5000);
    return () => clearInterval(interval);
  }, [activityFilter]);

  const formatDuration = (ms?: number) => {
    if (!ms) return '-';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          ACTIVITY LOGS
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          System operations and platform activities
        </p>
      </div>

      {/* Stats Summary */}
      <div className="grid grid-cols-3 gap-6">
        <div className="border-2 border-steel bg-concrete p-6">
          <div className="flex items-center gap-3 mb-2">
            <Activity className="w-5 h-5 text-terminal" />
            <span className="text-xs font-mono text-smoke font-medium">TOTAL ACTIVITIES</span>
          </div>
          <span className="text-3xl font-mono font-bold text-chalk">{logs.length}</span>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-2">
        <button
          onClick={() => setActivityFilter('')}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            activityFilter === ''
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          ALL TYPES
        </button>
        <button
          onClick={() => setActivityFilter('rss_fetch')}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            activityFilter === 'rss_fetch'
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          RSS FETCH
        </button>
        <button
          onClick={() => setActivityFilter('enrichment')}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            activityFilter === 'enrichment'
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          ENRICHMENT
        </button>
        <button
          onClick={() => setActivityFilter('playwright_scrape')}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            activityFilter === 'playwright_scrape'
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          PLAYWRIGHT
        </button>
      </div>

      {/* Activity Logs Table */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <ListChecks className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-pulse" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING ACTIVITIES...</p>
        </div>
      ) : logs.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <ListChecks className="w-16 h-16 text-steel/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO ACTIVITIES FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            Activity logs will appear as the system operates
          </p>
        </div>
      ) : (
        <div className="border-2 border-steel bg-concrete overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-void border-b-2 border-steel">
                <tr className="text-left">
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">TIMESTAMP</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">TYPE</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">PLATFORM</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">MESSAGE</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">SOURCES</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">DURATION</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log, idx) => (
                  <tr
                    key={log.id}
                    className={`border-b border-steel hover:bg-void/50 transition-colors ${
                      idx % 2 === 0 ? 'bg-void/20' : ''
                    }`}
                  >
                    <td className="px-4 py-3 text-xs font-mono text-fog">
                      {formatDateTime(log.timestamp)}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-1 text-xs font-mono font-bold border border-terminal text-terminal uppercase">
                        {log.activity_type.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {log.platform ? (
                        <span className="px-2 py-1 text-xs font-mono font-bold border border-steel text-chalk uppercase">
                          {log.platform}
                        </span>
                      ) : (
                        <span className="text-xs font-mono text-fog">-</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-chalk max-w-md truncate">
                      {log.message}
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-terminal font-bold">
                      {log.source_count ?? '-'}
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-electric">
                      {formatDuration(log.duration_ms)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className="text-xs font-mono text-smoke">
        SHOWING {logs.length} ACTIVITY LOG(S)
      </div>
    </div>
  );
}
