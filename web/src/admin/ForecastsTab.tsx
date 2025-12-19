import { useState, useEffect } from 'react';
import { TrendingUp, Plus, X, Play, Eye, Loader, AlertTriangle, Edit, Copy, Trash2 } from 'lucide-react';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';
import { formatDateTime } from '../utils/dateFormat';
import { ForecastChart } from './ForecastChart';

interface Forecast {
  id: string;
  name: string;
  proposition: string;
  prediction_type: string; // 'percentile' or 'point_estimate'
  units: string; // e.g., 'percent_change', 'dollars', 'points'
  target_date?: string;
  categories: string[];
  headline_count: number;
  iterations: number;
  context_urls: string[];
  active: boolean;
  public: boolean; // Whether the forecast is publicly visible on homepage
  display_order: number; // Sort order for homepage display
  schedule_enabled: boolean;
  schedule_interval: number; // Interval in minutes
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

interface ForecastModel {
  provider: string;
  model_name: string;
  api_key: string;
  weight: number;
}

interface ForecastRun {
  id: string;
  forecast_id: string;
  run_at: string;
  headline_count: number;
  status: string;
  error_message?: string;
  completed_at?: string;
}

interface PercentilePredictions {
  p10: number;
  p25: number;
  p50: number;
  p75: number;
  p90: number;
}

interface ForecastRunDetail {
  run: ForecastRun;
  responses: any[];
  result?: {
    aggregated_percentiles?: PercentilePredictions;
    aggregated_point_estimate?: number;
    model_count: number;
    consensus_level?: number;
  };
}

type ChartViewMode = 'hourly' | 'daily';

export function ForecastsTab() {
  const [forecasts, setForecasts] = useState<Forecast[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [duplicatingForecast, setDuplicatingForecast] = useState<Forecast | null>(null);
  const [editingForecast, setEditingForecast] = useState<Forecast | null>(null);
  const [selectedRun, setSelectedRun] = useState<ForecastRunDetail | null>(null);
  const [chartViewMode, setChartViewMode] = useState<ChartViewMode>('hourly');

  useEffect(() => {
    fetchForecasts();
  }, []);

  const fetchForecasts = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch forecasts');
      const data = await response.json();
      setForecasts(data.forecasts || []);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching forecasts:', err);
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-4 md:pl-6">
        <div className="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4 mb-4">
          <div>
            <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
              FORECAST MANAGEMENT
            </h2>
            <p className="text-xs md:text-sm text-smoke font-mono mt-2">
              Multi-model AI forecasting for intelligence analysis
            </p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center gap-2 px-4 md:px-6 py-2 md:py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs md:text-sm font-bold whitespace-nowrap"
          >
            <Plus className="w-4 h-4" />
            CREATE FORECAST
          </button>
        </div>
        {/* Global Chart View Selector */}
        {!loading && forecasts.length > 0 && (
          <div className="flex items-center gap-2 mb-4">
            <span className="text-xs font-mono text-smoke font-bold">CHART VIEW:</span>
            <select
              value={chartViewMode}
              onChange={(e) => setChartViewMode(e.target.value as ChartViewMode)}
              className="border-2 border-steel bg-void text-chalk font-mono text-xs font-bold px-3 py-2 hover:border-iron focus:border-terminal focus:outline-none"
            >
              <option value="hourly">HOURLY (24H)</option>
              <option value="daily">DAILY (OHLC)</option>
            </select>
          </div>
        )}
      </div>

      {/* Forecasts List */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING FORECASTS...</p>
        </div>
      ) : forecasts.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <TrendingUp className="w-16 h-16 text-steel/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO FORECASTS FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            Create a new forecast to start making intelligence predictions
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {forecasts.map((forecast) => (
            <ForecastCard
              key={forecast.id}
              forecast={forecast}
              onRunSelected={setSelectedRun}
              onEdit={setEditingForecast}
              onDuplicate={setDuplicatingForecast}
              onDelete={fetchForecasts}
              chartViewMode={chartViewMode}
            />
          ))}
        </div>
      )}

      {/* Create Forecast Modal */}
      {showCreateModal && (
        <CreateForecastModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchForecasts();
          }}
        />
      )}

      {/* Duplicate Forecast Modal */}
      {duplicatingForecast && (
        <DuplicateForecastModal
          forecast={duplicatingForecast}
          onClose={() => setDuplicatingForecast(null)}
          onSuccess={() => {
            setDuplicatingForecast(null);
            fetchForecasts();
          }}
        />
      )}

      {/* Edit Forecast Modal */}
      {editingForecast && (
        <EditForecastModal
          forecast={editingForecast}
          onClose={() => setEditingForecast(null)}
          onSuccess={() => {
            setEditingForecast(null);
            fetchForecasts();
          }}
        />
      )}

      {/* Run Detail Modal */}
      {selectedRun && (
        <RunDetailModal
          runDetail={selectedRun}
          onClose={() => setSelectedRun(null)}
        />
      )}
    </div>
  );
}

function ForecastCard({ forecast, onRunSelected, onEdit, onDuplicate, onDelete, chartViewMode }: { forecast: Forecast; onRunSelected: (run: ForecastRunDetail) => void; onEdit: (forecast: Forecast) => void; onDuplicate: (forecast: Forecast) => void; onDelete: () => void; chartViewMode: 'hourly' | 'daily' }) {
  const [runs, setRuns] = useState<ForecastRun[]>([]);
  const [expanded, setExpanded] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [latestResult, setLatestResult] = useState<ForecastRunDetail | null>(null);
  const [scheduleSaving, setScheduleSaving] = useState(false);

  useEffect(() => {
    fetchLatestRun();
  }, []);

  useEffect(() => {
    if (expanded) {
      fetchRuns();
    }
  }, [expanded]);

  const fetchLatestRun = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/runs`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) return;
      const data = await response.json();
      const completedRuns = (data.runs || []).filter((r: ForecastRun) => r.status === 'completed');
      if (completedRuns.length > 0) {
        // Fetch the latest completed run's details
        const latestRun = completedRuns[0];
        const detailResponse = await fetch(`${API_BASE_URL}/api/admin/forecasts/runs/${latestRun.id}`, {
          headers: getAuthHeaders(),
        });
        if (detailResponse.ok) {
          const detail = await detailResponse.json();
          setLatestResult(detail);
        }
      }
    } catch (err) {
      console.error('Error fetching latest run:', err);
    }
  };

  const fetchRuns = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/runs`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch runs');
      const data = await response.json();
      setRuns(data.runs || []);
    } catch (err) {
      console.error('Error fetching runs:', err);
    }
  };

  const handleExecute = async () => {
    setExecuting(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/execute`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const result = await response.json();
      alert(`Forecast execution started! Run ID: ${result.run_id}`);
      if (expanded) {
        fetchRuns();
      }
      // Refresh latest result after a delay to allow the run to complete
      setTimeout(() => {
        fetchLatestRun();
      }, 2000);
    } catch (err) {
      alert(`Failed to execute forecast: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setExecuting(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete "${forecast.name}"? This action cannot be undone and will delete all associated runs and results.`)) {
      return;
    }

    setDeleting(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      alert(`Forecast "${forecast.name}" deleted successfully`);
      onDelete();
    } catch (err) {
      alert(`Failed to delete forecast: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setDeleting(false);
    }
  };

  const handleViewRun = async (runId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/runs/${runId}`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch run detail');
      const data = await response.json();
      onRunSelected(data);
    } catch (err) {
      alert(`Failed to load run detail: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleDeleteRun = async (runId: string, runDate: string) => {
    if (!confirm(`Delete run from ${runDate}? This cannot be undone.`)) {
      return;
    }
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/runs/${runId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to delete run');
      fetchRuns();
      fetchLatestRun();
    } catch (err) {
      alert(`Failed to delete run: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleToggleSchedule = async (enabled: boolean, interval?: number) => {
    setScheduleSaving(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/schedule`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          enabled,
          interval: interval || forecast.schedule_interval || 60,
        }),
      });
      if (!response.ok) throw new Error('Failed to update schedule');
      onDelete(); // Refresh the forecast list to get updated schedule info
    } catch (err) {
      alert(`Failed to update schedule: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setScheduleSaving(false);
    }
  };

  const handleTogglePublic = async (isPublic: boolean) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/public`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          public: isPublic,
        }),
      });
      if (!response.ok) throw new Error('Failed to update public status');
      onDelete(); // Refresh the forecast list
    } catch (err) {
      alert(`Failed to update public status: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleUpdateDisplayOrder = async (displayOrder: number) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/display-order`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          display_order: displayOrder,
        }),
      });
      if (!response.ok) throw new Error('Failed to update display order');
      onDelete(); // Refresh the forecast list
    } catch (err) {
      alert(`Failed to update display order: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleDeleteAllRuns = async () => {
    if (!confirm(`Delete all runs for "${forecast.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}/runs`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to delete all runs');
      alert('All runs deleted successfully');
      fetchRuns();
      fetchLatestRun();
    } catch (err) {
      alert(`Failed to delete all runs: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  return (
    <div className="border-2 border-steel bg-concrete">
      {/* Header */}
      <div className="p-4 md:p-6">
        <div className="flex flex-col lg:flex-row lg:justify-between lg:items-start gap-4">
          <div className="flex-1 min-w-0">
            <h3 className="font-mono font-bold text-chalk text-lg break-words">{forecast.name}</h3>
            <p className="text-sm font-mono text-fog mt-2 break-words">{forecast.proposition}</p>
            <div className="flex flex-wrap gap-2 md:gap-3 mt-4 text-xs font-mono">
              <span className="text-smoke">
                Type: <span className="text-terminal font-bold">{forecast.prediction_type === 'percentile' ? 'DISTRIBUTION' : 'POINT ESTIMATE'}</span>
              </span>
              <span className="text-smoke">
                Units: <span className="text-terminal font-bold">{forecast.units}</span>
              </span>
              <span className="text-smoke">
                Categories: <span className="text-terminal font-bold">{forecast.categories.join(', ') || 'All'}</span>
              </span>
              <span className="text-smoke">
                Headlines: <span className="text-terminal font-bold">{forecast.headline_count}</span>
              </span>
              {forecast.target_date && (
                <span className="text-smoke">
                  Target: <span className="text-terminal font-bold">{new Date(forecast.target_date).toLocaleDateString()}</span>
                </span>
              )}
            </div>

            {/* Latest Result */}
            {latestResult?.result && (
              <div className="mt-4 border-2 border-terminal bg-terminal/5 p-3">
                {latestResult.result.aggregated_percentiles ? (
                  // Percentile forecast result
                  <div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs font-mono text-smoke">MEDIAN (P50):</span>
                      <span className="text-2xl md:text-3xl font-mono font-black text-terminal">
                        {latestResult.result.aggregated_percentiles.p50.toFixed(2)}
                      </span>
                    </div>
                    <div className="text-xs font-mono text-fog mt-2">
                      <div className="grid grid-cols-5 gap-2 text-center">
                        <div>
                          <div className="text-smoke">P10</div>
                          <div className="text-chalk font-bold">{latestResult.result.aggregated_percentiles.p10.toFixed(2)}</div>
                        </div>
                        <div>
                          <div className="text-smoke">P25</div>
                          <div className="text-chalk font-bold">{latestResult.result.aggregated_percentiles.p25.toFixed(2)}</div>
                        </div>
                        <div>
                          <div className="text-terminal">P50</div>
                          <div className="text-terminal font-bold">{latestResult.result.aggregated_percentiles.p50.toFixed(2)}</div>
                        </div>
                        <div>
                          <div className="text-smoke">P75</div>
                          <div className="text-chalk font-bold">{latestResult.result.aggregated_percentiles.p75.toFixed(2)}</div>
                        </div>
                        <div>
                          <div className="text-smoke">P90</div>
                          <div className="text-chalk font-bold">{latestResult.result.aggregated_percentiles.p90.toFixed(2)}</div>
                        </div>
                      </div>
                    </div>
                  </div>
                ) : latestResult.result.aggregated_point_estimate !== undefined ? (
                  // Point estimate result
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-xs font-mono text-smoke">ESTIMATE:</span>
                    <span className="text-2xl md:text-3xl font-mono font-black text-terminal">
                      {latestResult.result.aggregated_point_estimate.toFixed(2)}
                    </span>
                  </div>
                ) : null}
                <div className="text-xs font-mono text-fog mt-2 pt-2 border-t border-steel">
                  {formatDateTime(latestResult.run.run_at)} • {latestResult.result.model_count} model{latestResult.result.model_count !== 1 ? 's' : ''}
                  {latestResult.result.consensus_level !== undefined && (
                    <> • Consensus StdDev: {latestResult.result.consensus_level.toFixed(2)}</>
                  )}
                </div>
              </div>
            )}

            {/* Public & Schedule Controls */}
            <div className="mt-4 border-2 border-steel bg-void/30 p-3 space-y-3">
              {/* Public Toggle */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs font-mono text-smoke font-bold">PUBLIC:</span>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={forecast.public}
                    onChange={(e) => handleTogglePublic(e.target.checked)}
                    className="w-4 h-4"
                  />
                  <span className="text-xs font-mono text-chalk">
                    {forecast.public ? 'VISIBLE ON HOMEPAGE' : 'ADMIN ONLY'}
                  </span>
                </label>
              </div>

              {/* Display Order */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs font-mono text-smoke font-bold">DISPLAY ORDER:</span>
                <input
                  type="number"
                  value={forecast.display_order}
                  onChange={(e) => handleUpdateDisplayOrder(parseInt(e.target.value) || 0)}
                  className="w-20 px-2 py-1 bg-void border-2 border-steel text-chalk font-mono text-xs"
                  placeholder="0"
                />
                <span className="text-xs font-mono text-fog">
                  (Higher = Earlier on homepage)
                </span>
              </div>

              {/* Schedule */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs font-mono text-smoke font-bold">SCHEDULE:</span>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={forecast.schedule_enabled}
                    onChange={(e) => handleToggleSchedule(e.target.checked)}
                    disabled={scheduleSaving}
                    className="w-4 h-4"
                  />
                  <span className="text-xs font-mono text-chalk">
                    {forecast.schedule_enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </label>
                {forecast.schedule_enabled && (
                  <>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-fog">Every</span>
                      <input
                        type="number"
                        min="1"
                        max="43200"
                        value={forecast.schedule_interval}
                        onChange={(e) => handleToggleSchedule(true, parseInt(e.target.value))}
                        disabled={scheduleSaving}
                        className="w-20 px-2 py-1 border border-steel bg-void text-terminal font-mono text-xs font-bold focus:border-terminal focus:outline-none"
                      />
                      <span className="text-xs font-mono text-fog">minutes</span>
                      <span className="text-xs font-mono text-fog/50">
                        ({forecast.schedule_interval === 60 ? '1 hour' :
                          forecast.schedule_interval === 1440 ? '1 day' :
                          forecast.schedule_interval === 10080 ? '1 week' :
                          forecast.schedule_interval < 60 ? `${forecast.schedule_interval} min` :
                          forecast.schedule_interval < 1440 ? `${(forecast.schedule_interval / 60).toFixed(1)} hours` :
                          `${(forecast.schedule_interval / 1440).toFixed(1)} days`})
                      </span>
                    </div>
                    {forecast.next_run_at && (
                      <span className="text-xs font-mono text-fog">
                        Next run: <span className="text-electric font-bold">{formatDateTime(forecast.next_run_at)}</span>
                      </span>
                    )}
                    {forecast.last_run_at && (
                      <span className="text-xs font-mono text-fog">
                        Last run: <span className="text-smoke">{formatDateTime(forecast.last_run_at)}</span>
                      </span>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>

          {/* Action Buttons - Stack on mobile, row on desktop */}
          <div className="flex flex-wrap gap-2 lg:flex-nowrap">
            <button
              onClick={() => onDuplicate(forecast)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-fog text-fog hover:bg-fog hover:text-void transition-all font-mono text-xs md:text-sm font-bold"
            >
              <Copy className="w-3 h-3 md:w-4 md:h-4" />
              <span className="hidden sm:inline">COPY</span>
            </button>
            <button
              onClick={() => onEdit(forecast)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-xs md:text-sm font-bold"
            >
              <Edit className="w-3 h-3 md:w-4 md:h-4" />
              <span className="hidden sm:inline">EDIT</span>
            </button>
            <button
              onClick={handleExecute}
              disabled={executing}
              className="flex items-center gap-2 px-3 py-2 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs md:text-sm font-bold disabled:opacity-50"
            >
              {executing ? <Loader className="w-3 h-3 md:w-4 md:h-4 animate-spin" /> : <Play className="w-3 h-3 md:w-4 md:h-4" />}
              {executing ? 'RUN' : <span className="hidden sm:inline">EXECUTE</span>}
              {!executing && <span className="sm:hidden">RUN</span>}
            </button>
            <button
              onClick={() => setExpanded(!expanded)}
              className="px-3 py-2 border-2 border-steel text-chalk hover:border-iron transition-all font-mono text-xs md:text-sm font-bold"
            >
              {expanded ? 'HIDE' : 'RUNS'}
            </button>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="flex items-center gap-2 px-3 py-2 border-2 border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all font-mono text-xs md:text-sm font-bold disabled:opacity-50"
            >
              {deleting ? <Loader className="w-3 h-3 md:w-4 md:h-4 animate-spin" /> : <Trash2 className="w-3 h-3 md:w-4 md:h-4" />}
              <span className="hidden sm:inline">{deleting ? 'DELETING...' : 'DELETE'}</span>
            </button>
          </div>
        </div>
      </div>

      {/* Runs List */}
      {expanded && (
        <div className="border-t-2 border-steel bg-void/50 p-6 space-y-6">
          {/* Historical Chart - only show for percentile forecasts */}
          {forecast.prediction_type === 'percentile' && (
            <ForecastChart forecastId={forecast.id} viewMode={chartViewMode} />
          )}

          {runs.length === 0 ? (
            <p className="text-sm font-mono text-fog text-center py-8">No runs yet. Execute the forecast to create one.</p>
          ) : (
            <div className="space-y-2">
              <div className="flex justify-between items-center mb-3">
                <h4 className="text-xs font-mono text-smoke font-bold">RECENT RUNS</h4>
                <button
                  onClick={handleDeleteAllRuns}
                  className="flex items-center gap-2 px-3 py-1 border-2 border-threat-medium text-threat-medium hover:bg-threat-medium hover:text-void transition-all font-mono text-xs font-bold"
                >
                  <Trash2 className="w-3 h-3" />
                  DELETE ALL
                </button>
              </div>
              {runs.map((run) => (
                <div
                  key={run.id}
                  className="flex justify-between items-center p-3 border border-steel bg-concrete hover:border-iron transition-colors"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <span className="text-xs font-mono text-fog">{formatDateTime(run.run_at)}</span>
                      <span className={`px-2 py-1 text-xs font-mono font-bold border ${
                        run.status === 'completed' ? 'border-terminal text-terminal' :
                        run.status === 'running' ? 'border-warning text-warning' :
                        run.status === 'failed' ? 'border-threat-critical text-threat-critical' :
                        'border-steel text-steel'
                      }`}>
                        {run.status.toUpperCase()}
                      </span>
                      <span className="text-xs font-mono text-smoke">
                        {run.headline_count} headlines
                      </span>
                    </div>
                    {run.error_message && (
                      <p className="text-xs font-mono text-threat-critical mt-1">{run.error_message}</p>
                    )}
                  </div>
                  <div className="flex gap-2">
                    {run.status === 'completed' && (
                      <button
                        onClick={() => handleViewRun(run.id)}
                        className="flex items-center gap-2 px-3 py-1 border border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-xs font-bold"
                      >
                        <Eye className="w-3 h-3" />
                        VIEW
                      </button>
                    )}
                    <button
                      onClick={() => handleDeleteRun(run.id, formatDateTime(run.run_at))}
                      className="flex items-center gap-1 px-3 py-1 border border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all font-mono text-xs font-bold"
                      title="Delete this run"
                    >
                      <Trash2 className="w-3 h-3" />
                      <span className="hidden sm:inline">DELETE</span>
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function CreateForecastModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState('');
  const [proposition, setProposition] = useState('');
  const [predictionType, setPredictionType] = useState('percentile');
  const [units, setUnits] = useState('percent_change');
  const [targetDate, setTargetDate] = useState('');
  const [categories, setCategories] = useState<string[]>([]);
  const [headlineCount, setHeadlineCount] = useState(500);
  const [iterations, setIterations] = useState(1);
  const [contextUrls, setContextUrls] = useState<string[]>([]);
  const [models, setModels] = useState<ForecastModel[]>([
    { provider: 'openai', model_name: 'gpt-4', api_key: '', weight: 1.0 },
  ]);
  const [creating, setCreating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

  const addModel = () => {
    setModels([...models, { provider: 'openai', model_name: 'gpt-4', api_key: '', weight: 1.0 }]);
  };

  const removeModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const updateModel = (index: number, field: keyof ForecastModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleCreate = async () => {
    if (!name || !proposition || !units || models.length === 0) {
      alert('Please fill in all required fields and add at least one model');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setCreating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts`, {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          proposition,
          prediction_type: predictionType,
          units,
          target_date: targetDate ? `${targetDate}T00:00:00Z` : null,
          categories,
          headline_count: headlineCount,
          iterations,
          context_urls: contextUrls,
          models,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to create forecast: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setCreating(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="max-w-4xl w-full border-4 border-terminal bg-concrete my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-terminal bg-terminal/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-terminal flex items-center gap-3">
            <Plus className="w-6 h-6" />
            CREATE FORECAST
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-terminal transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Form */}
        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {/* Name */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">NAME *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., US Recession Q2 2025"
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
          </div>

          {/* Proposition */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PROPOSITION *</label>
            <textarea
              value={proposition}
              onChange={(e) => setProposition(e.target.value)}
              placeholder="e.g., What will be the % change of the S&P 500 1 year from today?"
              rows={3}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
            <p className="text-xs font-mono text-fog">
              Ask for an actual value prediction (e.g., "What will be the % change..."), not a yes/no question
            </p>
          </div>

          {/* Prediction Type */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PREDICTION TYPE *</label>
            <div className="flex gap-3">
              <button
                onClick={() => setPredictionType('percentile')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'percentile'
                    ? 'border-terminal bg-terminal text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                PERCENTILE (Distribution)
              </button>
              <button
                onClick={() => setPredictionType('point_estimate')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'point_estimate'
                    ? 'border-terminal bg-terminal text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                POINT ESTIMATE (Single Value)
              </button>
            </div>
            <p className="text-xs font-mono text-fog">
              {predictionType === 'percentile'
                ? 'Returns P10, P25, P50 (median), P75, P90 for uncertainty distribution'
                : 'Returns a single best estimate value'}
            </p>
          </div>

          {/* Units */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">UNITS *</label>
            <select
              value={units}
              onChange={(e) => setUnits(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            >
              <option value="percent_change">Percent Change (%)</option>
              <option value="percentage_points">Percentage Points (pp)</option>
              <option value="dollars">Dollars ($)</option>
              <option value="points">Points</option>
              <option value="basis_points">Basis Points (bps)</option>
              <option value="count">Count/Number</option>
              <option value="custom">Custom</option>
            </select>
            <p className="text-xs font-mono text-fog">
              What units should the forecast value be expressed in?
            </p>
          </div>

          {/* Target Date */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">TARGET DATE (optional)</label>
            <input
              type="date"
              value={targetDate}
              onChange={(e) => setTargetDate(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
            <p className="text-xs font-mono text-fog">
              When is this prediction for? (e.g., 1 year from now)
            </p>
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">CATEGORIES (leave empty for all)</label>
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setCategories(
                    categories.includes(cat)
                      ? categories.filter(c => c !== cat)
                      : [...categories, cat]
                  )}
                  className={`px-3 py-1 border font-mono text-xs font-bold uppercase transition-all ${
                    categories.includes(cat)
                      ? 'border-terminal bg-terminal text-void'
                      : 'border-steel bg-void text-fog hover:border-iron'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          {/* Headline Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              HEADLINE COUNT: {headlineCount}
            </label>
            <input
              type="range"
              min="100"
              max="1000"
              step="50"
              value={headlineCount}
              onChange={(e) => setHeadlineCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>100</span>
              <span>1000</span>
            </div>
          </div>

          {/* Iterations */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              ITERATIONS PER MODEL: {iterations}
            </label>
            <input
              type="range"
              min="1"
              max="50"
              step="1"
              value={iterations}
              onChange={(e) => setIterations(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1 (faster)</span>
              <span>50 (most consistent)</span>
            </div>
            <p className="text-xs font-mono text-fog">
              Higher iterations reduce variance by averaging multiple runs per model
            </p>
          </div>

          {/* Context URLs */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              CONTEXT URLs (optional)
            </label>
            <p className="text-xs font-mono text-fog mb-2">
              URLs to fetch and inject as context before headlines (e.g., market data, statistics)
            </p>
            {contextUrls.map((url, index) => (
              <div key={index} className="flex gap-2">
                <input
                  type="text"
                  value={url}
                  onChange={(e) => {
                    const newUrls = [...contextUrls];
                    newUrls[index] = e.target.value;
                    setContextUrls(newUrls);
                  }}
                  placeholder="https://example.com/data.json"
                  className="flex-1 px-4 py-2 border-2 border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                />
                <button
                  onClick={() => setContextUrls(contextUrls.filter((_, i) => i !== index))}
                  className="px-3 border-2 border-steel text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
            <button
              onClick={() => setContextUrls([...contextUrls, ''])}
              className="flex items-center gap-2 px-3 py-2 border border-steel text-fog hover:border-iron hover:text-chalk transition-all font-mono text-xs"
            >
              <Plus className="w-3 h-3" />
              ADD URL
            </button>
          </div>

          {/* Models */}
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <label className="block text-sm font-mono text-chalk font-bold">MODELS *</label>
              <button
                onClick={addModel}
                className="flex items-center gap-2 px-3 py-1 border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs font-bold"
              >
                <Plus className="w-3 h-3" />
                ADD MODEL
              </button>
            </div>

            {models.map((model, index) => (
              <div key={index} className="border-2 border-steel bg-void p-4 space-y-3">
                <div className="flex justify-between items-start">
                  <span className="text-sm font-mono text-terminal font-bold">MODEL {index + 1}</span>
                  {models.length > 1 && (
                    <button
                      onClick={() => removeModel(index)}
                      className="text-threat-critical hover:text-threat-high transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">PROVIDER</label>
                    <select
                      value={model.provider}
                      onChange={(e) => updateModel(index, 'provider', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                    >
                      <option value="openai">OpenAI (logprobs)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">MODEL NAME</label>
                    <input
                      type="text"
                      value={model.model_name}
                      onChange={(e) => updateModel(index, 'model_name', e.target.value)}
                      placeholder="e.g., gpt-4"
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">API KEY</label>
                  <input
                    type="password"
                    value={model.api_key}
                    onChange={(e) => updateModel(index, 'api_key', e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">
                    WEIGHT: {model.weight.toFixed(2)}
                  </label>
                  <input
                    type="range"
                    min="0.1"
                    max="2.0"
                    step="0.1"
                    value={model.weight}
                    onChange={(e) => updateModel(index, 'weight', parseFloat(e.target.value))}
                    className="w-full"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t-2 border-steel bg-void/50 flex gap-4">
          <button
            onClick={onClose}
            disabled={creating}
            className="flex-1 px-6 py-3 border-2 border-steel text-chalk hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            CANCEL
          </button>
          <button
            onClick={handleCreate}
            disabled={creating}
            className="flex-1 px-6 py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            {creating ? 'CREATING...' : 'CREATE FORECAST'}
          </button>
        </div>
      </div>
    </div>
  );
}

function DuplicateForecastModal({ forecast, onClose, onSuccess }: { forecast: Forecast; onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState(forecast.name + ' (Copy)');
  const [proposition, setProposition] = useState(forecast.proposition);
  const [predictionType, setPredictionType] = useState(forecast.prediction_type || 'percentile');
  const [units, setUnits] = useState(forecast.units || 'percent_change');
  const [targetDate, setTargetDate] = useState(forecast.target_date ? forecast.target_date.split('T')[0] : '');
  const [categories, setCategories] = useState<string[]>(forecast.categories || []);
  const [headlineCount, setHeadlineCount] = useState(forecast.headline_count);
  const [iterations, setIterations] = useState(forecast.iterations || 1);
  const [contextUrls, setContextUrls] = useState<string[]>(forecast.context_urls || []);
  const [models, setModels] = useState<ForecastModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

  useEffect(() => {
    fetchModels();
  }, []);

  const fetchModels = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch forecast models');
      const data = await response.json();
      setModels(data.models || []);
      setLoading(false);
    } catch (err) {
      alert(`Failed to load forecast models: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setLoading(false);
    }
  };

  const addModel = () => {
    setModels([...models, { provider: 'openai', model_name: 'gpt-4', api_key: '', weight: 1.0 }]);
  };

  const removeModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const updateModel = (index: number, field: keyof ForecastModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleCreate = async () => {
    if (!name || !proposition || !units || models.length === 0) {
      alert('Please fill in all required fields and add at least one model');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setCreating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts`, {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          proposition,
          prediction_type: predictionType,
          units,
          target_date: targetDate ? `${targetDate}T00:00:00Z` : null,
          categories,
          headline_count: headlineCount,
          iterations,
          context_urls: contextUrls,
          models,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to duplicate forecast: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setCreating(false);
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4">
        <div className="max-w-4xl w-full border-4 border-fog bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-fog/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING FORECAST...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="max-w-4xl w-full border-4 border-fog bg-concrete my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-fog bg-fog/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-fog flex items-center gap-3">
            <Copy className="w-6 h-6" />
            DUPLICATE FORECAST
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-fog transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Form - Same as CreateForecastModal */}
        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {/* Name */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">NAME *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-fog focus:outline-none"
            />
          </div>

          {/* Proposition */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PROPOSITION *</label>
            <textarea
              value={proposition}
              onChange={(e) => setProposition(e.target.value)}
              rows={3}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-fog focus:outline-none"
            />
          </div>

          {/* Prediction Type, Units, Target Date - same as in CreateForecastModal */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PREDICTION TYPE *</label>
            <div className="flex gap-3">
              <button
                onClick={() => setPredictionType('percentile')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'percentile'
                    ? 'border-fog bg-fog text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                PERCENTILE (Distribution)
              </button>
              <button
                onClick={() => setPredictionType('point_estimate')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'point_estimate'
                    ? 'border-fog bg-fog text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                POINT ESTIMATE (Single Value)
              </button>
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">UNITS *</label>
            <select
              value={units}
              onChange={(e) => setUnits(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-fog focus:outline-none"
            >
              <option value="percent_change">Percent Change (%)</option>
              <option value="percentage_points">Percentage Points (pp)</option>
              <option value="dollars">Dollars ($)</option>
              <option value="points">Points</option>
              <option value="basis_points">Basis Points (bps)</option>
              <option value="count">Count/Number</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">TARGET DATE (optional)</label>
            <input
              type="date"
              value={targetDate}
              onChange={(e) => setTargetDate(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-fog focus:outline-none"
            />
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">CATEGORIES (leave empty for all)</label>
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setCategories(
                    categories.includes(cat)
                      ? categories.filter(c => c !== cat)
                      : [...categories, cat]
                  )}
                  className={`px-3 py-1 border font-mono text-xs font-bold uppercase transition-all ${
                    categories.includes(cat)
                      ? 'border-fog bg-fog text-void'
                      : 'border-steel bg-void text-fog hover:border-iron'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          {/* Headline Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              HEADLINE COUNT: {headlineCount}
            </label>
            <input
              type="range"
              min="100"
              max="1000"
              step="50"
              value={headlineCount}
              onChange={(e) => setHeadlineCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>100</span>
              <span>1000</span>
            </div>
          </div>

          {/* Iterations */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              ITERATIONS PER MODEL: {iterations}
            </label>
            <input
              type="range"
              min="1"
              max="50"
              step="1"
              value={iterations}
              onChange={(e) => setIterations(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1 (faster)</span>
              <span>50 (most consistent)</span>
            </div>
            <p className="text-xs font-mono text-fog">
              Higher iterations reduce variance by averaging multiple runs per model
            </p>
          </div>

          {/* Context URLs */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              CONTEXT URLs (optional)
            </label>
            <p className="text-xs font-mono text-fog mb-2">
              URLs to fetch and inject as context before headlines (e.g., market data, statistics)
            </p>
            {contextUrls.map((url, index) => (
              <div key={index} className="flex gap-2">
                <input
                  type="text"
                  value={url}
                  onChange={(e) => {
                    const newUrls = [...contextUrls];
                    newUrls[index] = e.target.value;
                    setContextUrls(newUrls);
                  }}
                  placeholder="https://example.com/data.json"
                  className="flex-1 px-4 py-2 border-2 border-steel bg-void text-chalk font-mono text-sm focus:border-fog focus:outline-none"
                />
                <button
                  onClick={() => setContextUrls(contextUrls.filter((_, i) => i !== index))}
                  className="px-3 border-2 border-steel text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
            <button
              onClick={() => setContextUrls([...contextUrls, ''])}
              className="flex items-center gap-2 px-3 py-2 border border-steel text-fog hover:border-iron hover:text-chalk transition-all font-mono text-xs"
            >
              <Plus className="w-3 h-3" />
              ADD URL
            </button>
          </div>

          {/* Models */}
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <label className="block text-sm font-mono text-chalk font-bold">MODELS *</label>
              <button
                onClick={addModel}
                className="flex items-center gap-2 px-3 py-1 border border-fog text-fog hover:bg-fog hover:text-void transition-all font-mono text-xs font-bold"
              >
                <Plus className="w-3 h-3" />
                ADD MODEL
              </button>
            </div>

            {models.map((model, index) => (
              <div key={index} className="border-2 border-steel bg-void p-4 space-y-3">
                <div className="flex justify-between items-start">
                  <span className="text-sm font-mono text-fog font-bold">MODEL {index + 1}</span>
                  {models.length > 1 && (
                    <button
                      onClick={() => removeModel(index)}
                      className="text-threat-critical hover:text-threat-high transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">PROVIDER</label>
                    <select
                      value={model.provider}
                      onChange={(e) => updateModel(index, 'provider', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-fog focus:outline-none"
                    >
                      <option value="openai">OpenAI (logprobs)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">MODEL NAME</label>
                    <input
                      type="text"
                      value={model.model_name}
                      onChange={(e) => updateModel(index, 'model_name', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-fog focus:outline-none"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">API KEY</label>
                  <input
                    type="password"
                    value={model.api_key}
                    onChange={(e) => updateModel(index, 'api_key', e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-fog focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">
                    WEIGHT: {model.weight.toFixed(2)}
                  </label>
                  <input
                    type="range"
                    min="0.1"
                    max="2.0"
                    step="0.1"
                    value={model.weight}
                    onChange={(e) => updateModel(index, 'weight', parseFloat(e.target.value))}
                    className="w-full"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t-2 border-steel bg-void/50 flex gap-4">
          <button
            onClick={onClose}
            disabled={creating}
            className="flex-1 px-6 py-3 border-2 border-steel text-chalk hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            CANCEL
          </button>
          <button
            onClick={handleCreate}
            disabled={creating}
            className="flex-1 px-6 py-3 border-2 border-fog text-fog hover:bg-fog hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            {creating ? 'DUPLICATING...' : 'CREATE DUPLICATE'}
          </button>
        </div>
      </div>
    </div>
  );
}

function EditForecastModal({ forecast, onClose, onSuccess }: { forecast: Forecast; onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState(forecast.name);
  const [proposition, setProposition] = useState(forecast.proposition);
  const [predictionType, setPredictionType] = useState(forecast.prediction_type || 'percentile');
  const [units, setUnits] = useState(forecast.units || 'percent_change');
  const [targetDate, setTargetDate] = useState(forecast.target_date ? forecast.target_date.split('T')[0] : '');
  const [categories, setCategories] = useState<string[]>(forecast.categories || []);
  const [headlineCount, setHeadlineCount] = useState(forecast.headline_count);
  const [iterations, setIterations] = useState(forecast.iterations || 1);
  const [contextUrls, setContextUrls] = useState<string[]>(forecast.context_urls || []);
  const [models, setModels] = useState<ForecastModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

  useEffect(() => {
    fetchModels();
  }, []);

  const fetchModels = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch forecast models');
      const data = await response.json();
      setModels(data.models || []);
      setLoading(false);
    } catch (err) {
      alert(`Failed to load forecast models: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setLoading(false);
    }
  };

  const addModel = () => {
    setModels([...models, { provider: 'openai', model_name: 'gpt-4', api_key: '', weight: 1.0 }]);
  };

  const removeModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const updateModel = (index: number, field: keyof ForecastModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleUpdate = async () => {
    if (!name || !proposition || !units || models.length === 0) {
      alert('Please fill in all required fields and add at least one model');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setUpdating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/forecasts/${forecast.id}`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          proposition,
          prediction_type: predictionType,
          units,
          target_date: targetDate ? `${targetDate}T00:00:00Z` : null,
          categories,
          headline_count: headlineCount,
          iterations,
          context_urls: contextUrls,
          models,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to update forecast: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4">
        <div className="max-w-4xl w-full border-4 border-electric bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-electric/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING FORECAST...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="max-w-4xl w-full border-4 border-electric bg-concrete my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-electric bg-electric/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-electric flex items-center gap-3">
            <Edit className="w-6 h-6" />
            EDIT FORECAST
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-electric transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Form - Same as CreateForecastModal but with pre-filled values */}
        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {/* Name */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">NAME *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Proposition */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PROPOSITION *</label>
            <textarea
              value={proposition}
              onChange={(e) => setProposition(e.target.value)}
              rows={3}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Prediction Type */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">PREDICTION TYPE *</label>
            <div className="flex gap-3">
              <button
                onClick={() => setPredictionType('percentile')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'percentile'
                    ? 'border-electric bg-electric text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                PERCENTILE (Distribution)
              </button>
              <button
                onClick={() => setPredictionType('point_estimate')}
                className={`flex-1 px-4 py-3 border-2 font-mono text-sm font-bold transition-all ${
                  predictionType === 'point_estimate'
                    ? 'border-electric bg-electric text-void'
                    : 'border-steel bg-void text-fog hover:border-iron'
                }`}
              >
                POINT ESTIMATE (Single Value)
              </button>
            </div>
          </div>

          {/* Units */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">UNITS *</label>
            <select
              value={units}
              onChange={(e) => setUnits(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            >
              <option value="percent_change">Percent Change (%)</option>
              <option value="percentage_points">Percentage Points (pp)</option>
              <option value="dollars">Dollars ($)</option>
              <option value="points">Points</option>
              <option value="basis_points">Basis Points (bps)</option>
              <option value="count">Count/Number</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          {/* Target Date */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">TARGET DATE (optional)</label>
            <input
              type="date"
              value={targetDate}
              onChange={(e) => setTargetDate(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">CATEGORIES (leave empty for all)</label>
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setCategories(
                    categories.includes(cat)
                      ? categories.filter(c => c !== cat)
                      : [...categories, cat]
                  )}
                  className={`px-3 py-1 border font-mono text-xs font-bold uppercase transition-all ${
                    categories.includes(cat)
                      ? 'border-electric bg-electric text-void'
                      : 'border-steel bg-void text-fog hover:border-iron'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          {/* Headline Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              HEADLINE COUNT: {headlineCount}
            </label>
            <input
              type="range"
              min="100"
              max="1000"
              step="50"
              value={headlineCount}
              onChange={(e) => setHeadlineCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>100</span>
              <span>1000</span>
            </div>
          </div>

          {/* Iterations */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              ITERATIONS PER MODEL: {iterations}
            </label>
            <input
              type="range"
              min="1"
              max="50"
              step="1"
              value={iterations}
              onChange={(e) => setIterations(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1 (faster)</span>
              <span>50 (most consistent)</span>
            </div>
            <p className="text-xs font-mono text-fog">
              Higher iterations reduce variance by averaging multiple runs per model
            </p>
          </div>

          {/* Context URLs */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              CONTEXT URLs (optional)
            </label>
            <p className="text-xs font-mono text-fog mb-2">
              URLs to fetch and inject as context before headlines (e.g., market data, statistics)
            </p>
            {contextUrls.map((url, index) => (
              <div key={index} className="flex gap-2">
                <input
                  type="text"
                  value={url}
                  onChange={(e) => {
                    const newUrls = [...contextUrls];
                    newUrls[index] = e.target.value;
                    setContextUrls(newUrls);
                  }}
                  placeholder="https://example.com/data.json"
                  className="flex-1 px-4 py-2 border-2 border-steel bg-void text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                />
                <button
                  onClick={() => setContextUrls(contextUrls.filter((_, i) => i !== index))}
                  className="px-3 border-2 border-steel text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ))}
            <button
              onClick={() => setContextUrls([...contextUrls, ''])}
              className="flex items-center gap-2 px-3 py-2 border border-steel text-fog hover:border-iron hover:text-chalk transition-all font-mono text-xs"
            >
              <Plus className="w-3 h-3" />
              ADD URL
            </button>
          </div>

          {/* Models */}
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <label className="block text-sm font-mono text-chalk font-bold">MODELS *</label>
              <button
                onClick={addModel}
                className="flex items-center gap-2 px-3 py-1 border border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-xs font-bold"
              >
                <Plus className="w-3 h-3" />
                ADD MODEL
              </button>
            </div>

            {models.map((model, index) => (
              <div key={index} className="border-2 border-steel bg-void p-4 space-y-3">
                <div className="flex justify-between items-start">
                  <span className="text-sm font-mono text-electric font-bold">MODEL {index + 1}</span>
                  {models.length > 1 && (
                    <button
                      onClick={() => removeModel(index)}
                      className="text-threat-critical hover:text-threat-high transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">PROVIDER</label>
                    <select
                      value={model.provider}
                      onChange={(e) => updateModel(index, 'provider', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                    >
                      <option value="openai">OpenAI (logprobs)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-mono text-smoke mb-1">MODEL NAME</label>
                    <input
                      type="text"
                      value={model.model_name}
                      onChange={(e) => updateModel(index, 'model_name', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">API KEY</label>
                  <input
                    type="password"
                    value={model.api_key}
                    onChange={(e) => updateModel(index, 'api_key', e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 border border-steel bg-concrete text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-mono text-smoke mb-1">
                    WEIGHT: {model.weight.toFixed(2)}
                  </label>
                  <input
                    type="range"
                    min="0.1"
                    max="2.0"
                    step="0.1"
                    value={model.weight}
                    onChange={(e) => updateModel(index, 'weight', parseFloat(e.target.value))}
                    className="w-full"
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t-2 border-steel bg-void/50 flex gap-4">
          <button
            onClick={onClose}
            disabled={updating}
            className="flex-1 px-6 py-3 border-2 border-steel text-chalk hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            CANCEL
          </button>
          <button
            onClick={handleUpdate}
            disabled={updating}
            className="flex-1 px-6 py-3 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50"
          >
            {updating ? 'UPDATING...' : 'UPDATE FORECAST'}
          </button>
        </div>
      </div>
    </div>
  );
}

function RunDetailModal({ runDetail, onClose }: { runDetail: ForecastRunDetail; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="max-w-4xl w-full border-4 border-electric bg-concrete my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-electric bg-electric/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-electric flex items-center gap-3">
            <TrendingUp className="w-6 h-6" />
            FORECAST RUN DETAIL
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-electric transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {/* Run Info */}
          <div className="border-2 border-steel bg-void p-4 space-y-2">
            <div className="flex justify-between text-sm font-mono">
              <span className="text-smoke">Run ID:</span>
              <span className="text-chalk">{runDetail.run.id}</span>
            </div>
            <div className="flex justify-between text-sm font-mono">
              <span className="text-smoke">Executed:</span>
              <span className="text-chalk">{formatDateTime(runDetail.run.run_at)}</span>
            </div>
            <div className="flex justify-between text-sm font-mono">
              <span className="text-smoke">Headlines Used:</span>
              <span className="text-terminal font-bold">{runDetail.run.headline_count}</span>
            </div>
          </div>

          {/* Result */}
          {runDetail.result && (
            <div className="border-2 border-terminal bg-terminal/10 p-6">
              <h4 className="text-sm font-mono text-terminal font-bold mb-4">AGGREGATE RESULT</h4>
              <div className="space-y-3">
                {runDetail.result.aggregated_percentiles ? (
                  // Percentile forecast result
                  <div>
                    <div className="flex justify-between items-baseline mb-3">
                      <span className="text-sm font-mono text-chalk">Median (P50):</span>
                      <span className="text-3xl font-mono font-black text-terminal">
                        {runDetail.result.aggregated_percentiles.p50.toFixed(2)}
                      </span>
                    </div>
                    <div className="grid grid-cols-5 gap-2 text-center text-xs font-mono mt-3 pt-3 border-t border-steel">
                      <div>
                        <div className="text-smoke">P10</div>
                        <div className="text-chalk font-bold">{runDetail.result.aggregated_percentiles.p10.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-smoke">P25</div>
                        <div className="text-chalk font-bold">{runDetail.result.aggregated_percentiles.p25.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-terminal">P50</div>
                        <div className="text-terminal font-bold">{runDetail.result.aggregated_percentiles.p50.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-smoke">P75</div>
                        <div className="text-chalk font-bold">{runDetail.result.aggregated_percentiles.p75.toFixed(2)}</div>
                      </div>
                      <div>
                        <div className="text-smoke">P90</div>
                        <div className="text-chalk font-bold">{runDetail.result.aggregated_percentiles.p90.toFixed(2)}</div>
                      </div>
                    </div>
                  </div>
                ) : runDetail.result.aggregated_point_estimate !== undefined ? (
                  // Point estimate result
                  <div className="flex justify-between items-baseline">
                    <span className="text-sm font-mono text-chalk">Point Estimate:</span>
                    <span className="text-3xl font-mono font-black text-terminal">
                      {runDetail.result.aggregated_point_estimate.toFixed(2)}
                    </span>
                  </div>
                ) : null}
                <div className="flex justify-between text-sm font-mono pt-3 border-t border-steel">
                  <span className="text-smoke">Model Count:</span>
                  <span className="text-chalk">{runDetail.result.model_count}</span>
                </div>
                {runDetail.result.consensus_level !== undefined && (
                  <div className="flex justify-between text-sm font-mono">
                    <span className="text-smoke">Consensus (StdDev):</span>
                    <span className="text-chalk">{runDetail.result.consensus_level.toFixed(3)}</span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Model Responses */}
          <div className="space-y-3">
            <h4 className="text-sm font-mono text-chalk font-bold">MODEL RESPONSES</h4>
            {(runDetail.responses || []).map((response, idx) => (
              <div key={idx} className={`border-2 p-4 ${
                response.status === 'completed' ? 'border-terminal bg-terminal/5' : 'border-threat-critical bg-threat-critical/5'
              }`}>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <span className="text-sm font-mono font-bold text-chalk">{response.provider}</span>
                    <span className="text-xs font-mono text-fog ml-2">{response.model_name}</span>
                  </div>
                  <span className={`px-2 py-1 text-xs font-mono font-bold border ${
                    response.status === 'completed' ? 'border-terminal text-terminal' : 'border-threat-critical text-threat-critical'
                  }`}>
                    {response.status.toUpperCase()}
                  </span>
                </div>

                {response.status === 'completed' ? (
                  <>
                    {response.percentile_predictions && (
                      <div className="mb-2">
                        <div className="text-lg font-mono font-bold text-terminal mb-2">
                          Median: {response.percentile_predictions.p50?.toFixed(2)}
                        </div>
                        <div className="grid grid-cols-5 gap-2 text-center text-xs font-mono">
                          <div>
                            <div className="text-smoke">P10</div>
                            <div className="text-chalk">{response.percentile_predictions.p10?.toFixed(2)}</div>
                          </div>
                          <div>
                            <div className="text-smoke">P25</div>
                            <div className="text-chalk">{response.percentile_predictions.p25?.toFixed(2)}</div>
                          </div>
                          <div>
                            <div className="text-terminal">P50</div>
                            <div className="text-terminal font-bold">{response.percentile_predictions.p50?.toFixed(2)}</div>
                          </div>
                          <div>
                            <div className="text-smoke">P75</div>
                            <div className="text-chalk">{response.percentile_predictions.p75?.toFixed(2)}</div>
                          </div>
                          <div>
                            <div className="text-smoke">P90</div>
                            <div className="text-chalk">{response.percentile_predictions.p90?.toFixed(2)}</div>
                          </div>
                        </div>
                      </div>
                    )}
                    {response.point_estimate !== null && response.point_estimate !== undefined && (
                      <div className="mb-2">
                        <span className="text-2xl font-mono font-bold text-terminal">
                          {response.point_estimate.toFixed(2)}
                        </span>
                      </div>
                    )}
                    {response.reasoning && (
                      <p className="text-sm font-mono text-chalk mt-2">{response.reasoning}</p>
                    )}
                    {response.response_time_ms && (
                      <p className="text-xs font-mono text-fog mt-2">Response time: {response.response_time_ms}ms</p>
                    )}
                  </>
                ) : (
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="w-4 h-4 text-threat-critical flex-shrink-0 mt-0.5" />
                    <p className="text-sm font-mono text-threat-critical">{response.error_message || 'Unknown error'}</p>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t-2 border-steel bg-void/50">
          <button
            onClick={onClose}
            className="w-full px-6 py-3 border-2 border-steel text-chalk hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold"
          >
            CLOSE
          </button>
        </div>
      </div>
    </div>
  );
}
