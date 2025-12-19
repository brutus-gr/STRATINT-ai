import { useState, useEffect } from 'react';
import { TrendingUp, Plus, X, Play, Loader, Edit, Copy } from 'lucide-react';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';
import { formatDateTime } from '../utils/dateFormat';

interface Strategy {
  id: string;
  name: string;
  prompt: string;
  investment_symbols: string[];
  categories: string[];
  headline_count: number;
  iterations: number;
  forecast_ids: string[];
  forecast_history_count: number;
  models?: StrategyModel[];
  active: boolean;
  public: boolean;
  display_order: number;
  schedule_enabled: boolean;
  schedule_interval: number;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

interface StrategyRun {
  id: string;
  strategy_id: string;
  run_at: string;
  headline_count: number;
  status: string;
  error_message?: string;
  completed_at?: string;
}

interface StrategyResult {
  averaged_allocations: Record<string, number>;
  normalized_allocations: Record<string, number>;
  normalization_reasoning: string;
  model_count: number;
  iteration_count: number;
  consensus_variance: Record<string, number>;
}

interface StrategyRunDetail {
  run: StrategyRun;
  responses: any[];
  result?: StrategyResult;
}

interface StrategyModel {
  provider: string;
  model_name: string;
  api_key: string;
  weight: number;
}

interface Forecast {
  id: string;
  name: string;
  proposition: string;
  active: boolean;
}

export function StrategiesTab() {
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingStrategy, setEditingStrategy] = useState<Strategy | null>(null);
  const [duplicatingStrategy, setDuplicatingStrategy] = useState<Strategy | null>(null);

  useEffect(() => {
    fetchStrategies();
  }, []);

  const fetchStrategies = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch strategies');
      const data = await response.json();
      // Ensure data is an array and has required fields
      const strategies = Array.isArray(data) ? data : [];
      const sanitizedStrategies = strategies.map(s => ({
        ...s,
        investment_symbols: Array.isArray(s.investment_symbols) ? s.investment_symbols : [],
        categories: Array.isArray(s.categories) ? s.categories : [],
        forecast_ids: Array.isArray(s.forecast_ids) ? s.forecast_ids : [],
      }));
      setStrategies(sanitizedStrategies);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching strategies:', err);
      setStrategies([]);
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-4 md:pl-6 flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4">
        <div>
          <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
            STRATEGY MANAGEMENT
          </h2>
          <p className="text-xs md:text-sm text-smoke font-mono mt-2">
            AI-generated portfolio allocation strategies
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 md:px-6 py-2 md:py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs md:text-sm font-bold whitespace-nowrap"
        >
          <Plus className="w-4 h-4" />
          CREATE STRATEGY
        </button>
      </div>

      {/* Strategies List */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING STRATEGIES...</p>
        </div>
      ) : strategies.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <TrendingUp className="w-16 h-16 text-steel/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO STRATEGIES FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            Create a new strategy to start generating portfolio allocations
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {strategies.map((strategy) => (
            <StrategyCard
              key={strategy.id}
              strategy={strategy}
              onRefresh={fetchStrategies}
              onEdit={setEditingStrategy}
              onDuplicate={setDuplicatingStrategy}
            />
          ))}
        </div>
      )}

      {/* Create Strategy Modal */}
      {showCreateModal && (
        <CreateStrategyModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchStrategies();
          }}
        />
      )}

      {/* Edit Strategy Modal */}
      {editingStrategy && (
        <EditStrategyModal
          strategy={editingStrategy}
          onClose={() => setEditingStrategy(null)}
          onSuccess={() => {
            setEditingStrategy(null);
            fetchStrategies();
          }}
        />
      )}

      {/* Duplicate Strategy Modal */}
      {duplicatingStrategy && (
        <DuplicateStrategyModal
          strategy={duplicatingStrategy}
          onClose={() => setDuplicatingStrategy(null)}
          onSuccess={() => {
            setDuplicatingStrategy(null);
            fetchStrategies();
          }}
        />
      )}
    </div>
  );
}

function StrategyCard({ strategy, onRefresh, onEdit, onDuplicate }: { strategy: Strategy; onRefresh: () => void; onEdit: (strategy: Strategy) => void; onDuplicate: (strategy: Strategy) => void }) {
  const [runs, setRuns] = useState<StrategyRun[]>([]);
  const [expanded, setExpanded] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [latestResult, setLatestResult] = useState<StrategyRunDetail | null>(null);
  const [scheduleSaving, setScheduleSaving] = useState(false);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [selectedRunDetail, setSelectedRunDetail] = useState<StrategyRunDetail | null>(null);
  const [loadingRunDetail, setLoadingRunDetail] = useState(false);

  useEffect(() => {
    fetchLatestResult();
  }, []);

  useEffect(() => {
    if (expanded) {
      fetchRuns();
    }
  }, [expanded]);

  const fetchLatestResult = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/strategies/${strategy.id}/latest`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setLatestResult(data);
      }
    } catch (err) {
      console.error('Error fetching latest result:', err);
    }
  };

  const fetchRuns = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}/runs`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) return;
      const data = await response.json();
      // Ensure data is an array
      const runs = Array.isArray(data) ? data : [];
      setRuns(runs);
    } catch (err) {
      console.error('Error fetching runs:', err);
      setRuns([]);
    }
  };

  const fetchRunDetail = async (runId: string) => {
    setLoadingRunDetail(true);
    setSelectedRunId(runId);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/runs/${runId}`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch run detail');
      const data = await response.json();
      setSelectedRunDetail(data);
    } catch (err) {
      console.error('Error fetching run detail:', err);
      alert(`Failed to load run details: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setLoadingRunDetail(false);
    }
  };

  const handleExecute = async () => {
    setExecuting(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}/execute`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const result = await response.json();
      alert(`Strategy execution started! Run ID: ${result.run_id}`);
      setTimeout(() => fetchLatestResult(), 2000);
    } catch (err) {
      alert(`Failed to execute strategy: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setExecuting(false);
    }
  };

  const handleTogglePublic = async (isPublic: boolean) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}/publish`, {
        method: 'PUT',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ public: isPublic }),
      });
      if (!response.ok) throw new Error('Failed to toggle public status');
      onRefresh();
    } catch (err) {
      alert(`Failed to toggle public status: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleToggleSchedule = async (enabled: boolean, interval?: number) => {
    setScheduleSaving(true);
    try {
      // If enabling and no interval provided, use existing interval or default to 1 day (1440 minutes)
      const scheduleInterval = interval !== undefined
        ? interval
        : (strategy.schedule_interval > 0 ? strategy.schedule_interval : 1440);

      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}/schedule`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          schedule_enabled: enabled,
          schedule_interval: scheduleInterval,
        }),
      });
      if (!response.ok) throw new Error('Failed to update schedule');
      onRefresh();
    } catch (err) {
      alert(`Failed to update schedule: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setScheduleSaving(false);
    }
  };

  return (
    <div className="border-2 border-steel bg-concrete">
      {/* Header */}
      <div className="p-4 md:p-6">
        <div className="flex flex-col md:flex-row md:justify-between md:items-start gap-4">
          <div className="flex-1">
            <h3 className="font-mono font-bold text-chalk text-lg break-words">{strategy.name}</h3>
            <p className="text-sm font-mono text-fog mt-2 break-words">{strategy.prompt}</p>
            <div className="flex flex-wrap gap-2 md:gap-3 mt-4 text-xs font-mono">
              <span className="text-smoke">
                Symbols: <span className="text-terminal font-bold">{(strategy.investment_symbols || []).join(', ')}</span>
              </span>
              <span className="text-smoke">
                Headlines: <span className="text-terminal font-bold">{strategy.headline_count}</span>
              </span>
              <span className="text-smoke">
                Iterations: <span className="text-terminal font-bold">{strategy.iterations}</span>
              </span>
              {(strategy.categories || []).length > 0 && (
                <span className="text-smoke">
                  Categories: <span className="text-terminal font-bold">{(strategy.categories || []).join(', ')}</span>
                </span>
              )}
            </div>

            {/* Public & Schedule Controls */}
            <div className="mt-4 border-2 border-steel bg-void/30 p-3 space-y-3">
              {/* Public Toggle */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs font-mono text-smoke font-bold">PUBLIC:</span>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={strategy.public}
                    onChange={(e) => handleTogglePublic(e.target.checked)}
                    className="w-4 h-4"
                  />
                  <span className="text-xs font-mono text-chalk">
                    {strategy.public ? 'VISIBLE ON HOMEPAGE' : 'ADMIN ONLY'}
                  </span>
                </label>
              </div>

              {/* Schedule */}
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-xs font-mono text-smoke font-bold">SCHEDULE:</span>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={strategy.schedule_enabled}
                    onChange={(e) => handleToggleSchedule(e.target.checked)}
                    disabled={scheduleSaving}
                    className="w-4 h-4"
                  />
                  <span className="text-xs font-mono text-chalk">
                    {strategy.schedule_enabled ? 'ENABLED' : 'DISABLED'}
                  </span>
                </label>
                {strategy.schedule_enabled && (
                  <>
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-mono text-fog">Every</span>
                      <input
                        type="number"
                        min="1"
                        max="43200"
                        value={strategy.schedule_interval}
                        onChange={(e) => handleToggleSchedule(true, parseInt(e.target.value))}
                        disabled={scheduleSaving}
                        className="w-20 px-2 py-1 border border-steel bg-void text-terminal font-mono text-xs font-bold focus:border-terminal focus:outline-none"
                      />
                      <span className="text-xs font-mono text-fog">minutes</span>
                      <span className="text-xs font-mono text-fog/50">
                        ({strategy.schedule_interval === 60 ? '1 hour' :
                          strategy.schedule_interval === 1440 ? '1 day' :
                          strategy.schedule_interval === 10080 ? '1 week' :
                          strategy.schedule_interval < 60 ? `${strategy.schedule_interval} min` :
                          strategy.schedule_interval < 1440 ? `${(strategy.schedule_interval / 60).toFixed(1)} hours` :
                          `${(strategy.schedule_interval / 1440).toFixed(1)} days`})
                      </span>
                    </div>
                    {strategy.next_run_at && (
                      <span className="text-xs font-mono text-fog">
                        Next run: <span className="text-electric font-bold">{formatDateTime(strategy.next_run_at)}</span>
                      </span>
                    )}
                    {strategy.last_run_at && (
                      <span className="text-xs font-mono text-fog">
                        Last run: <span className="text-smoke">{formatDateTime(strategy.last_run_at)}</span>
                      </span>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-2">
            <button
              onClick={handleExecute}
              disabled={executing || !strategy.active}
              className="flex items-center gap-2 px-3 py-2 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all disabled:opacity-50 disabled:cursor-not-allowed text-xs font-mono font-bold"
            >
              {executing ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
              {executing ? 'EXECUTING...' : 'EXECUTE'}
            </button>

            <button
              onClick={() => onDuplicate(strategy)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-fog text-fog hover:bg-fog hover:text-void transition-all text-xs font-mono font-bold"
            >
              <Copy className="w-4 h-4" />
              COPY
            </button>

            <button
              onClick={() => onEdit(strategy)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all text-xs font-mono font-bold"
            >
              <Edit className="w-4 h-4" />
              EDIT
            </button>

            <button
              onClick={() => setExpanded(!expanded)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all text-xs font-mono font-bold"
            >
              {expanded ? 'COLLAPSE' : 'EXPAND'}
            </button>
          </div>
        </div>
      </div>

      {/* Latest Result */}
      {latestResult?.result && latestResult.result.normalized_allocations && (
        <div className="p-4 md:p-6 border-b-2 border-steel bg-void/20">
          <h4 className="font-mono font-bold text-sm text-terminal mb-3">LATEST ALLOCATION</h4>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
            {Object.entries(latestResult.result.normalized_allocations).map(([symbol, allocation]) => (
              <div key={symbol} className="bg-concrete border border-steel p-3">
                <div className="font-mono font-black text-2xl text-terminal">{allocation.toFixed(1)}%</div>
                <div className="font-mono text-xs text-fog mt-1">{symbol}</div>
                {latestResult.result!.consensus_variance?.[symbol] !== undefined && (
                  <div className="font-mono text-xs text-smoke mt-1">
                    σ: {latestResult.result!.consensus_variance[symbol].toFixed(2)}
                  </div>
                )}
              </div>
            ))}
          </div>
          {latestResult.result.normalization_reasoning && (
            <div className="mt-4 p-3 bg-steel/20 border border-steel">
              <div className="font-mono text-xs text-fog mb-1">AI NORMALIZATION REASONING:</div>
              <div className="font-mono text-sm text-smoke">{latestResult.result.normalization_reasoning}</div>
            </div>
          )}
        </div>
      )}

      {/* Run History */}
      {expanded && (
        <div className="p-4 md:p-6 bg-void/10">
          <h4 className="font-mono font-bold text-sm text-terminal mb-3">RUN HISTORY</h4>
          {runs.length === 0 ? (
            <p className="text-sm font-mono text-fog">No runs yet</p>
          ) : (
            <div className="space-y-2">
              {runs.map((run) => (
                <div key={run.id}>
                  <button
                    onClick={() => fetchRunDetail(run.id)}
                    className={`w-full text-left flex justify-between items-center p-3 border transition-all ${
                      selectedRunId === run.id
                        ? 'bg-terminal/10 border-terminal'
                        : 'bg-concrete border-steel hover:border-terminal/50'
                    }`}
                  >
                    <div className="font-mono text-xs">
                      <span className="text-chalk">{formatDateTime(run.run_at)}</span>
                      <span className={`ml-3 px-2 py-1 ${
                        run.status === 'completed' ? 'bg-terminal/20 text-terminal' :
                        run.status === 'failed' ? 'bg-red-500/20 text-red-400' :
                        'bg-steel/20 text-fog'
                      }`}>
                        {run.status.toUpperCase()}
                      </span>
                    </div>
                    {run.completed_at && (
                      <span className="font-mono text-xs text-fog">
                        Completed: {formatDateTime(run.completed_at)}
                      </span>
                    )}
                  </button>

                  {/* Run Detail */}
                  {selectedRunId === run.id && (
                    <div className="mt-2 p-4 bg-void/30 border-2 border-terminal">
                      {loadingRunDetail ? (
                        <div className="text-center py-8">
                          <Loader className="w-8 h-8 text-terminal/50 mx-auto mb-2 animate-spin" />
                          <p className="text-xs font-mono text-fog">Loading run details...</p>
                        </div>
                      ) : selectedRunDetail?.result && selectedRunDetail.result.normalized_allocations ? (
                        <div>
                          <h5 className="font-mono font-bold text-xs text-terminal mb-3">ALLOCATION RESULT</h5>
                          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2 mb-4">
                            {Object.entries(selectedRunDetail.result.normalized_allocations).map(([symbol, allocation]) => (
                              <div key={symbol} className="bg-concrete border border-steel p-2">
                                <div className="font-mono font-black text-xl text-terminal">{allocation.toFixed(1)}%</div>
                                <div className="font-mono text-xs text-fog">{symbol}</div>
                                {selectedRunDetail.result!.consensus_variance?.[symbol] !== undefined && (
                                  <div className="font-mono text-xs text-smoke">
                                    σ: {selectedRunDetail.result!.consensus_variance[symbol].toFixed(2)}
                                  </div>
                                )}
                              </div>
                            ))}
                          </div>
                          {selectedRunDetail.result.normalization_reasoning && (
                            <div className="p-3 bg-steel/20 border border-steel">
                              <div className="font-mono text-xs text-fog mb-1">AI NORMALIZATION REASONING:</div>
                              <div className="font-mono text-xs text-smoke">{selectedRunDetail.result.normalization_reasoning}</div>
                            </div>
                          )}
                          <div className="mt-3 flex gap-4 text-xs font-mono text-fog">
                            <span>Models: {selectedRunDetail.result.model_count}</span>
                            <span>Iterations: {selectedRunDetail.result.iteration_count}</span>
                          </div>
                        </div>
                      ) : selectedRunDetail?.run.status === 'failed' ? (
                        <div className="text-center py-4">
                          <p className="text-sm font-mono text-red-400">Run failed</p>
                          {selectedRunDetail.run.error_message && (
                            <p className="text-xs font-mono text-fog mt-2">{selectedRunDetail.run.error_message}</p>
                          )}
                        </div>
                      ) : (
                        <div className="text-center py-4">
                          <p className="text-sm font-mono text-fog">No result available for this run</p>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function CreateStrategyModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState('');
  const [prompt, setPrompt] = useState('');
  const [investmentSymbols, setInvestmentSymbols] = useState<string[]>(['SPY', 'TLT', 'GLD']);
  const [newSymbol, setNewSymbol] = useState('');
  const [categories, setCategories] = useState<string[]>([]);
  const [headlineCount, setHeadlineCount] = useState(100);
  const [iterations, setIterations] = useState(3);
  const [forecastHistoryCount, setForecastHistoryCount] = useState(1);
  const [models, setModels] = useState<StrategyModel[]>([
    { provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 },
  ]);
  const [creating, setCreating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

  const addSymbol = () => {
    if (newSymbol && !investmentSymbols.includes(newSymbol.toUpperCase())) {
      setInvestmentSymbols([...investmentSymbols, newSymbol.toUpperCase()]);
      setNewSymbol('');
    }
  };

  const removeSymbol = (symbol: string) => {
    setInvestmentSymbols(investmentSymbols.filter(s => s !== symbol));
  };

  const toggleCategory = (category: string) => {
    if (categories.includes(category)) {
      setCategories(categories.filter(c => c !== category));
    } else {
      setCategories([...categories, category]);
    }
  };

  const addModel = () => {
    setModels([...models, { provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 }]);
  };

  const removeModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const updateModel = (index: number, field: keyof StrategyModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleCreate = async () => {
    if (!name || !prompt || investmentSymbols.length === 0 || models.length === 0) {
      alert('Please fill in all required fields');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setCreating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies`, {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          prompt,
          investment_symbols: investmentSymbols,
          categories,
          headline_count: headlineCount,
          iterations,
          forecast_history_count: forecastHistoryCount,
          forecast_ids: [],
          models,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to create strategy: ${err instanceof Error ? err.message : 'Unknown error'}`);
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
            CREATE STRATEGY
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
              placeholder="e.g., Multi-Asset Risk Parity"
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
          </div>

          {/* Prompt */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">ALLOCATION PROMPT *</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="e.g., Allocate portfolio based on geopolitical risk and market conditions. Return percentage allocations that sum to 100%."
              rows={4}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
          </div>

          {/* Investment Symbols */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">INVESTMENT SYMBOLS *</label>
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                value={newSymbol}
                onChange={(e) => setNewSymbol(e.target.value.toUpperCase())}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addSymbol())}
                placeholder="Add ticker (e.g., SPY)"
                className="flex-1 px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
              />
              <button
                onClick={addSymbol}
                className="px-4 py-2 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono font-bold"
              >
                ADD
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              {investmentSymbols.map((symbol) => (
                <span
                  key={symbol}
                  className="px-3 py-1 bg-terminal/20 text-terminal border border-terminal font-mono text-sm flex items-center gap-2"
                >
                  {symbol}
                  <button
                    onClick={() => removeSymbol(symbol)}
                    className="text-terminal hover:text-chalk"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
            </div>
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">EVENT CATEGORIES</label>
            <p className="text-xs font-mono text-fog mb-2">Filter headlines by category (leave empty for all)</p>
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => toggleCategory(cat)}
                  className={`px-3 py-2 border-2 font-mono text-xs font-bold transition-all ${
                    categories.includes(cat)
                      ? 'border-terminal bg-terminal text-void'
                      : 'border-steel bg-void text-fog hover:border-iron'
                  }`}
                >
                  {cat.toUpperCase()}
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
            <p className="text-xs font-mono text-fog">Number of recent headlines to include in context</p>
          </div>

          {/* Iterations */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              ITERATIONS: {iterations}
            </label>
            <input
              type="range"
              min="1"
              max="10"
              step="1"
              value={iterations}
              onChange={(e) => setIterations(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1</span>
              <span>10</span>
            </div>
            <p className="text-xs font-mono text-fog">Number of times to run each model for consensus</p>
          </div>

          {/* Forecast History Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              FORECAST HISTORY COUNT: {forecastHistoryCount}
            </label>
            <input
              type="range"
              min="1"
              max="250"
              step="1"
              value={forecastHistoryCount}
              onChange={(e) => setForecastHistoryCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1</span>
              <span>250</span>
            </div>
            <p className="text-xs font-mono text-fog">Number of past forecast runs to include in strategy context</p>
          </div>

          {/* Models */}
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <label className="block text-sm font-mono text-chalk font-bold">AI MODELS *</label>
              <button
                onClick={addModel}
                className="flex items-center gap-1 px-3 py-1 border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs font-bold"
              >
                <Plus className="w-3 h-3" />
                ADD MODEL
              </button>
            </div>
            {models.map((model, index) => (
              <div key={index} className="p-4 border-2 border-steel bg-void/50 space-y-3">
                <div className="flex justify-between items-center">
                  <span className="font-mono text-sm text-chalk font-bold">MODEL {index + 1}</span>
                  {models.length > 1 && (
                    <button
                      onClick={() => removeModel(index)}
                      className="text-fog hover:text-chalk"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-mono text-fog mb-1">PROVIDER</label>
                    <select
                      value={model.provider}
                      onChange={(e) => updateModel(index, 'provider', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                    >
                      <option value="anthropic">Anthropic</option>
                      <option value="openai">OpenAI</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs font-mono text-fog mb-1">MODEL NAME</label>
                    <input
                      type="text"
                      value={model.model_name}
                      onChange={(e) => updateModel(index, 'model_name', e.target.value)}
                      placeholder="e.g., claude-sonnet-4-20250514"
                      className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-mono text-fog mb-1">API KEY</label>
                  <input
                    type="password"
                    value={model.api_key}
                    onChange={(e) => updateModel(index, 'api_key', e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs font-mono text-fog mb-1">
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
        <div className="px-6 py-4 border-t-4 border-terminal bg-steel/10 flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-6 py-3 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all font-mono font-bold"
          >
            CANCEL
          </button>
          <button
            onClick={handleCreate}
            disabled={creating}
            className="px-6 py-3 border-2 border-terminal bg-terminal text-void hover:bg-terminal/90 transition-all font-mono font-bold disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {creating ? (
              <>
                <Loader className="w-4 h-4 animate-spin" />
                CREATING...
              </>
            ) : (
              'CREATE STRATEGY'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

function EditStrategyModal({ strategy, onClose, onSuccess }: { strategy: Strategy; onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState(strategy.name);
  const [prompt, setPrompt] = useState(strategy.prompt);
  const [investmentSymbols, setInvestmentSymbols] = useState<string[]>(strategy.investment_symbols || []);
  const [newSymbol, setNewSymbol] = useState('');
  const [categories, setCategories] = useState<string[]>(strategy.categories || []);
  const [headlineCount, setHeadlineCount] = useState(strategy.headline_count);
  const [iterations, setIterations] = useState(strategy.iterations);
  const [forecastHistoryCount, setForecastHistoryCount] = useState(strategy.forecast_history_count || 1);
  const [forecastIds, setForecastIds] = useState<string[]>(strategy.forecast_ids || []);
  const [models, setModels] = useState<StrategyModel[]>(
    strategy.models && strategy.models.length > 0
      ? strategy.models
      : [{ provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 }]
  );
  const [availableForecasts, setAvailableForecasts] = useState<Forecast[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

  useEffect(() => {
    fetchModelsAndForecasts();
  }, []);

  const fetchModelsAndForecasts = async () => {
    try {
      // Fetch full strategy details (includes models)
      const strategyResponse = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}`, {
        headers: getAuthHeaders(),
      });
      if (!strategyResponse.ok) throw new Error('Failed to fetch strategy details');
      const strategyData = await strategyResponse.json();
      // Update models from the full strategy data
      if (strategyData.models && Array.isArray(strategyData.models) && strategyData.models.length > 0) {
        setModels(strategyData.models);
      }

      // Fetch available forecasts
      const forecastsResponse = await fetch(`${API_BASE_URL}/api/admin/forecasts`, {
        headers: getAuthHeaders(),
      });
      if (!forecastsResponse.ok) throw new Error('Failed to fetch forecasts');
      const forecastsData = await forecastsResponse.json();
      // API returns object with forecasts field, not direct array
      const forecasts = Array.isArray(forecastsData?.forecasts) ? forecastsData.forecasts :
                        Array.isArray(forecastsData) ? forecastsData : [];
      setAvailableForecasts(forecasts);

      setLoading(false);
    } catch (err) {
      console.error('Failed to load forecasts:', err);
      setAvailableForecasts([]);
      alert(`Failed to load forecasts: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setLoading(false);
    }
  };

  const addSymbol = () => {
    if (newSymbol && !investmentSymbols.includes(newSymbol.toUpperCase())) {
      setInvestmentSymbols([...investmentSymbols, newSymbol.toUpperCase()]);
      setNewSymbol('');
    }
  };

  const removeSymbol = (symbol: string) => {
    setInvestmentSymbols(investmentSymbols.filter(s => s !== symbol));
  };

  const toggleCategory = (category: string) => {
    if (categories.includes(category)) {
      setCategories(categories.filter(c => c !== category));
    } else {
      setCategories([...categories, category]);
    }
  };

  const toggleForecast = (forecastId: string) => {
    if (forecastIds.includes(forecastId)) {
      setForecastIds(forecastIds.filter(id => id !== forecastId));
    } else {
      setForecastIds([...forecastIds, forecastId]);
    }
  };

  const addModel = () => {
    setModels([...models, { provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 }]);
  };

  const removeModel = (index: number) => {
    setModels(models.filter((_, i) => i !== index));
  };

  const updateModel = (index: number, field: keyof StrategyModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleUpdate = async () => {
    if (!name || !prompt || investmentSymbols.length === 0 || models.length === 0) {
      alert('Please fill in all required fields');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setUpdating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          prompt,
          investment_symbols: investmentSymbols,
          categories,
          headline_count: headlineCount,
          iterations,
          forecast_history_count: forecastHistoryCount,
          forecast_ids: forecastIds,
          models,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to update strategy: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4">
        <div className="max-w-4xl w-full border-4 border-electric bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-electric/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING STRATEGY...</p>
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
            EDIT STRATEGY
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-electric transition-colors">
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
              placeholder="e.g., Multi-Asset Risk Parity"
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Prompt */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">ALLOCATION PROMPT *</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="e.g., Allocate portfolio based on geopolitical risk and market conditions. Return percentage allocations that sum to 100%."
              rows={4}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Investment Symbols */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">INVESTMENT SYMBOLS *</label>
            <div className="flex gap-2 mb-2">
              <input
                type="text"
                value={newSymbol}
                onChange={(e) => setNewSymbol(e.target.value.toUpperCase())}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addSymbol())}
                placeholder="Add ticker (e.g., SPY)"
                className="flex-1 px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
              />
              <button
                onClick={addSymbol}
                className="px-4 py-2 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono font-bold"
              >
                ADD
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              {investmentSymbols.map((symbol) => (
                <span
                  key={symbol}
                  className="px-3 py-1 bg-electric/20 text-electric border border-electric font-mono text-sm flex items-center gap-2"
                >
                  {symbol}
                  <button
                    onClick={() => removeSymbol(symbol)}
                    className="text-electric hover:text-chalk"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
            </div>
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">EVENT CATEGORIES</label>
            <p className="text-xs font-mono text-fog mb-2">Filter headlines by category (leave empty for all)</p>
            <div className="flex flex-wrap gap-2">
              {availableCategories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => toggleCategory(cat)}
                  className={`px-3 py-2 border-2 font-mono text-xs font-bold transition-all ${
                    categories.includes(cat)
                      ? 'border-electric bg-electric text-void'
                      : 'border-steel bg-void text-fog hover:border-iron'
                  }`}
                >
                  {cat.toUpperCase()}
                </button>
              ))}
            </div>
          </div>

          {/* Forecasts */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">FORECASTS</label>
            <p className="text-xs font-mono text-fog mb-2">Select forecasts to inject into strategy prompt (optional)</p>
            {availableForecasts.length === 0 ? (
              <p className="text-xs font-mono text-smoke">No active forecasts available</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {availableForecasts.map((forecast) => (
                  <button
                    key={forecast.id}
                    onClick={() => toggleForecast(forecast.id)}
                    className={`px-3 py-2 border-2 font-mono text-xs font-bold transition-all ${
                      forecastIds.includes(forecast.id)
                        ? 'border-electric bg-electric text-void'
                        : 'border-steel bg-void text-fog hover:border-iron'
                    }`}
                  >
                    {forecast.name}
                  </button>
                ))}
              </div>
            )}
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
            <p className="text-xs font-mono text-fog">Number of recent headlines to include in context</p>
          </div>

          {/* Iterations */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              ITERATIONS: {iterations}
            </label>
            <input
              type="range"
              min="1"
              max="10"
              step="1"
              value={iterations}
              onChange={(e) => setIterations(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1</span>
              <span>10</span>
            </div>
            <p className="text-xs font-mono text-fog">Number of times to run each model for consensus</p>
          </div>

          {/* Forecast History Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              FORECAST HISTORY COUNT: {forecastHistoryCount}
            </label>
            <input
              type="range"
              min="1"
              max="250"
              step="1"
              value={forecastHistoryCount}
              onChange={(e) => setForecastHistoryCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1</span>
              <span>250</span>
            </div>
            <p className="text-xs font-mono text-fog">Number of past forecast runs to include in strategy context</p>
          </div>

          {/* Models */}
          <div className="space-y-3">
            <div className="flex justify-between items-center">
              <label className="block text-sm font-mono text-chalk font-bold">AI MODELS *</label>
              <button
                onClick={addModel}
                className="flex items-center gap-1 px-3 py-1 border border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-xs font-bold"
              >
                <Plus className="w-3 h-3" />
                ADD MODEL
              </button>
            </div>
            {models.map((model, index) => (
              <div key={index} className="p-4 border-2 border-steel bg-void/50 space-y-3">
                <div className="flex justify-between items-center">
                  <span className="font-mono text-sm text-electric font-bold">MODEL {index + 1}</span>
                  {models.length > 1 && (
                    <button
                      onClick={() => removeModel(index)}
                      className="text-fog hover:text-chalk"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-mono text-fog mb-1">PROVIDER</label>
                    <select
                      value={model.provider}
                      onChange={(e) => updateModel(index, 'provider', e.target.value)}
                      className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                    >
                      <option value="anthropic">Anthropic</option>
                      <option value="openai">OpenAI</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs font-mono text-fog mb-1">MODEL NAME</label>
                    <input
                      type="text"
                      value={model.model_name}
                      onChange={(e) => updateModel(index, 'model_name', e.target.value)}
                      placeholder="e.g., claude-sonnet-4-20250514"
                      className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-xs font-mono text-fog mb-1">API KEY</label>
                  <input
                    type="password"
                    value={model.api_key}
                    onChange={(e) => updateModel(index, 'api_key', e.target.value)}
                    placeholder="sk-..."
                    className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs font-mono text-fog mb-1">
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
        <div className="px-6 py-4 border-t-4 border-electric bg-steel/10 flex justify-end gap-3">
          <button
            onClick={onClose}
            disabled={updating}
            className="px-6 py-3 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all font-mono font-bold disabled:opacity-50"
          >
            CANCEL
          </button>
          <button
            onClick={handleUpdate}
            disabled={updating}
            className="px-6 py-3 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono font-bold disabled:opacity-50 flex items-center gap-2"
          >
            {updating ? (
              <>
                <Loader className="w-4 h-4 animate-spin" />
                UPDATING...
              </>
            ) : (
              'UPDATE STRATEGY'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}


function DuplicateStrategyModal({ strategy, onClose, onSuccess }: { strategy: Strategy; onClose: () => void; onSuccess: () => void }) {
  // For now, just alert - full implementation will be added
  return (
    <div className="fixed inset-0 bg-void/90 flex items-center justify-center z-50 p-4">
      <div className="border-4 border-fog bg-concrete max-w-2xl w-full p-8">
        <div className="px-6 py-4 border-b-4 border-fog bg-fog/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-fog flex items-center gap-3">
            <Copy className="w-6 h-6" />
            DUPLICATE STRATEGY
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-fog transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>
        <div className="p-6">
          <p className="text-chalk font-mono mb-4">Duplicating: {strategy.name}</p>
          <p className="text-fog font-mono text-sm mb-6">This feature creates a copy of the strategy with all settings including API keys.</p>
          <button
            onClick={async () => {
              try {
                const strategyResponse = await fetch(`${API_BASE_URL}/api/admin/strategies/${strategy.id}`, {
                  headers: getAuthHeaders(),
                });
                const fullStrategy = await strategyResponse.json();
                const response = await fetch(`${API_BASE_URL}/api/admin/strategies`, {
                  method: 'POST',
                  headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    ...fullStrategy,
                    name: fullStrategy.name + ' (Copy)',
                    id: undefined,
                    created_at: undefined,
                    updated_at: undefined,
                  }),
                });
                if (!response.ok) throw new Error('Failed to duplicate');
                onSuccess();
              } catch {
                alert('Failed to duplicate strategy');
                onClose();
              }
            }}
            className="w-full px-6 py-3 border-2 border-fog text-fog hover:bg-fog hover:text-void transition-all font-mono font-bold"
          >
            CREATE COPY
          </button>
        </div>
      </div>
    </div>
  );
}
