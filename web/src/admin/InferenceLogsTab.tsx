import { useState, useEffect } from 'react';
import { Activity, TrendingUp, DollarSign, Clock, AlertCircle } from 'lucide-react';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';
import { formatDateTime } from '../utils/dateFormat';

interface InferenceLog {
  id: number;
  provider: string;
  model: string;
  operation: string;
  tokens_used: number;
  input_tokens?: number;
  output_tokens?: number;
  cost_usd?: number;
  latency_ms?: number;
  status: string;
  error_message?: string;
  metadata?: string;
  created_at: string;
}

interface InferenceStats {
  total_calls: number;
  total_tokens: number;
  total_cost_usd: number;
  successful_calls: number;
  failed_calls: number;
  avg_latency_ms: number;
}

export function InferenceLogsTab() {
  const [logs, setLogs] = useState<InferenceLog[]>([]);
  const [stats, setStats] = useState<InferenceStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [providerFilter, setProviderFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [operationFilter, setOperationFilter] = useState('');
  const [limit, setLimit] = useState(100);

  useEffect(() => {
    fetchLogs();
    fetchStats();
  }, [providerFilter, modelFilter, operationFilter, limit]);

  const fetchLogs = async () => {
    try {
      setLoading(true);
      const params = new URLSearchParams();
      if (providerFilter) params.append('provider', providerFilter);
      if (modelFilter) params.append('model', modelFilter);
      if (operationFilter) params.append('operation', operationFilter);
      params.append('limit', limit.toString());

      const response = await fetch(`${API_BASE_URL}/api/admin/inference-logs?${params}`, {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to fetch inference logs');
      }

      const data = await response.json();
      setLogs(data.logs || []);
    } catch (err) {
      console.error('Error fetching inference logs:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/inference-logs/stats`, {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to fetch stats');
      }

      const data = await response.json();
      setStats(data);
    } catch (err) {
      console.error('Error fetching stats:', err);
    }
  };

  return (
    <div className="space-y-6">
      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <Activity className="w-4 h-4 text-terminal" />
              <span className="text-xs font-mono text-fog">TOTAL CALLS</span>
            </div>
            <p className="text-2xl font-mono font-bold text-chalk">{stats.total_calls.toLocaleString()}</p>
          </div>

          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <TrendingUp className="w-4 h-4 text-electric" />
              <span className="text-xs font-mono text-fog">SUCCESS RATE</span>
            </div>
            <p className="text-2xl font-mono font-bold text-chalk">
              {stats.total_calls > 0 ? ((stats.successful_calls / stats.total_calls) * 100).toFixed(1) : '0'}%
            </p>
          </div>

          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <Activity className="w-4 h-4 text-terminal" />
              <span className="text-xs font-mono text-fog">TOTAL TOKENS</span>
            </div>
            <p className="text-2xl font-mono font-bold text-chalk">{(stats.total_tokens / 1_000_000).toFixed(2)}M</p>
          </div>

          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <DollarSign className="w-4 h-4 text-terminal" />
              <span className="text-xs font-mono text-fog">TOTAL COST</span>
            </div>
            <p className="text-2xl font-mono font-bold text-chalk">${stats.total_cost_usd.toFixed(2)}</p>
          </div>

          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <Clock className="w-4 h-4 text-fog" />
              <span className="text-xs font-mono text-fog">AVG LATENCY</span>
            </div>
            <p className="text-2xl font-mono font-bold text-chalk">{Math.round(stats.avg_latency_ms)}ms</p>
          </div>

          <div className="border-2 border-steel bg-concrete p-4">
            <div className="flex items-center gap-2 mb-2">
              <AlertCircle className="w-4 h-4 text-threat-critical" />
              <span className="text-xs font-mono text-fog">FAILURES</span>
            </div>
            <p className="text-2xl font-mono font-bold text-threat-critical">{stats.failed_calls}</p>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="border-2 border-steel bg-concrete p-4">
        <h3 className="text-sm font-mono font-bold text-chalk mb-4">FILTERS</h3>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label className="block text-xs font-mono text-fog mb-2">PROVIDER</label>
            <select
              value={providerFilter}
              onChange={(e) => setProviderFilter(e.target.value)}
              className="w-full bg-void border-2 border-steel text-chalk font-mono text-sm p-2"
            >
              <option value="">All</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-mono text-fog mb-2">OPERATION</label>
            <input
              type="text"
              value={operationFilter}
              onChange={(e) => setOperationFilter(e.target.value)}
              placeholder="event_creation, forecast..."
              className="w-full bg-void border-2 border-steel text-chalk font-mono text-sm p-2"
            />
          </div>

          <div>
            <label className="block text-xs font-mono text-fog mb-2">MODEL</label>
            <input
              type="text"
              value={modelFilter}
              onChange={(e) => setModelFilter(e.target.value)}
              placeholder="gpt-4o, claude-sonnet-4..."
              className="w-full bg-void border-2 border-steel text-chalk font-mono text-sm p-2"
            />
          </div>

          <div>
            <label className="block text-xs font-mono text-fog mb-2">LIMIT</label>
            <select
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
              className="w-full bg-void border-2 border-steel text-chalk font-mono text-sm p-2"
            >
              <option value="50">50</option>
              <option value="100">100</option>
              <option value="250">250</option>
              <option value="500">500</option>
            </select>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div className="border-2 border-steel bg-concrete">
        <div className="p-4 border-b-2 border-steel">
          <h3 className="text-sm font-mono font-bold text-chalk">INFERENCE LOGS</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="border-b-2 border-steel">
              <tr className="text-left">
                <th className="p-3 text-xs font-mono text-fog">TIME</th>
                <th className="p-3 text-xs font-mono text-fog">PROVIDER</th>
                <th className="p-3 text-xs font-mono text-fog">MODEL</th>
                <th className="p-3 text-xs font-mono text-fog">OPERATION</th>
                <th className="p-3 text-xs font-mono text-fog">TOKENS</th>
                <th className="p-3 text-xs font-mono text-fog">COST</th>
                <th className="p-3 text-xs font-mono text-fog">LATENCY</th>
                <th className="p-3 text-xs font-mono text-fog">STATUS</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={8} className="p-8 text-center text-fog font-mono">
                    Loading...
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan={8} className="p-8 text-center text-fog font-mono">
                    No inference logs found
                  </td>
                </tr>
              ) : (
                logs.map((log) => (
                  <tr key={log.id} className="border-b border-steel hover:bg-void/50">
                    <td className="p-3 text-xs font-mono text-smoke">{formatDateTime(log.created_at)}</td>
                    <td className="p-3 text-xs font-mono text-chalk">
                      <span className={`px-2 py-1 ${log.provider === 'openai' ? 'bg-electric/20 text-electric' : 'bg-terminal/20 text-terminal'}`}>
                        {log.provider.toUpperCase()}
                      </span>
                    </td>
                    <td className="p-3 text-xs font-mono text-chalk">{log.model}</td>
                    <td className="p-3 text-xs font-mono text-fog">{log.operation}</td>
                    <td className="p-3 text-xs font-mono text-chalk">
                      {log.tokens_used.toLocaleString()}
                      {log.input_tokens && log.output_tokens && (
                        <span className="text-fog ml-1">
                          ({log.input_tokens}→{log.output_tokens})
                        </span>
                      )}
                    </td>
                    <td className="p-3 text-xs font-mono text-chalk">
                      {log.cost_usd ? `$${log.cost_usd.toFixed(4)}` : '-'}
                    </td>
                    <td className="p-3 text-xs font-mono text-fog">
                      {log.latency_ms ? `${log.latency_ms}ms` : '-'}
                    </td>
                    <td className="p-3 text-xs font-mono">
                      {log.status === 'success' ? (
                        <span className="text-terminal">✓ SUCCESS</span>
                      ) : (
                        <span className="text-threat-critical" title={log.error_message}>
                          ✗ ERROR
                        </span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
