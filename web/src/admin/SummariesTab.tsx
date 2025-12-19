import { useState, useEffect } from 'react';
import { FileText, Plus, X, Play, Loader, Edit, Twitter, Copy } from 'lucide-react';
import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';
import { formatDateTime } from '../utils/dateFormat';

interface Summary {
  id: string;
  name: string;
  prompt: string;
  time_of_day?: string;
  lookback_hours: number;
  categories: string[];
  headline_count: number;
  models: SummaryModel[];
  active: boolean;
  schedule_enabled: boolean;
  schedule_interval: number;
  auto_post_to_twitter: boolean;
  include_forecasts: boolean;
  last_run_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

interface SummaryModel {
  provider: string;
  model_name: string;
  api_key: string;
  weight: number;
}

interface SummaryRun {
  id: string;
  summary_id: string;
  run_at: string;
  headline_count: number;
  lookback_start: string;
  lookback_end: string;
  status: string;
  error_message?: string;
  completed_at?: string;
}

interface SummaryResult {
  id: string;
  run_id: string;
  summary_text: string;
  model_provider: string;
  model_name: string;
  created_at: string;
}

interface SummaryRunDetail {
  run: SummaryRun;
  results: SummaryResult[];
}

export function SummariesTab() {
  const [summaries, setSummaries] = useState<Summary[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingSummary, setEditingSummary] = useState<Summary | null>(null);

  useEffect(() => {
    fetchSummaries();
  }, []);

  const fetchSummaries = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch summaries');
      const data = await response.json();
      const summaries = Array.isArray(data) ? data : [];
      const sanitizedSummaries = summaries.map(s => ({
        ...s,
        categories: Array.isArray(s.categories) ? s.categories : [],
        models: Array.isArray(s.models) ? s.models : [],
      }));
      setSummaries(sanitizedSummaries);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching summaries:', err);
      setSummaries([]);
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4 md:space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-4 md:pl-6 flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4">
        <div>
          <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
            SUMMARY MANAGEMENT
          </h2>
          <p className="text-xs md:text-sm text-smoke font-mono mt-2">
            AI-generated briefings based on headlines
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 md:px-6 py-2 md:py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-xs md:text-sm font-bold whitespace-nowrap"
        >
          <Plus className="w-4 h-4" />
          CREATE SUMMARY
        </button>
      </div>

      {/* Summaries List */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <Loader className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-spin" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING SUMMARIES...</p>
        </div>
      ) : summaries.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <FileText className="w-16 h-16 text-steel/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO SUMMARIES FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            Create a new summary to start generating briefings
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {summaries.map((summary) => (
            <SummaryCard
              key={summary.id}
              summary={summary}
              onEdit={setEditingSummary}
              onRefresh={fetchSummaries}
            />
          ))}
        </div>
      )}

      {/* Create Summary Modal */}
      {showCreateModal && (
        <CreateSummaryModal
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchSummaries();
          }}
        />
      )}

      {/* Edit Summary Modal */}
      {editingSummary && (
        <EditSummaryModal
          summary={editingSummary}
          onClose={() => setEditingSummary(null)}
          onSuccess={() => {
            setEditingSummary(null);
            fetchSummaries();
          }}
        />
      )}
    </div>
  );
}

function SummaryCard({ summary, onEdit, onRefresh }: { summary: Summary; onEdit: (summary: Summary) => void; onRefresh: () => void }) {
  const [runs, setRuns] = useState<SummaryRun[]>([]);
  const [expanded, setExpanded] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [latestResult, setLatestResult] = useState<SummaryRunDetail | null>(null);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [selectedRunDetail, setSelectedRunDetail] = useState<SummaryRunDetail | null>(null);
  const [loadingRunDetail, setLoadingRunDetail] = useState(false);
  const [postingToTwitter, setPostingToTwitter] = useState<Set<string>>(new Set());
  const [cloning, setCloning] = useState(false);

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
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/${summary.id}/latest`, {
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
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/${summary.id}/runs`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) return;
      const data = await response.json();
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
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/runs/${runId}`, {
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
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/${summary.id}/execute`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const result = await response.json();
      alert(`Summary execution started! Run ID: ${result.run_id}`);
      setTimeout(() => fetchLatestResult(), 2000);
    } catch (err) {
      alert(`Failed to execute summary: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setExecuting(false);
    }
  };

  const postToTwitter = async (runId: string, resultId: string) => {
    setPostingToTwitter(prev => new Set(prev).add(resultId));
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/runs/${runId}/tweet`, {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ result_id: resultId }),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      const result = await response.json();
      alert(`Posted to X successfully! Tweet URL: ${result.tweet_url}`);
    } catch (err) {
      alert(`Failed to post to X: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setPostingToTwitter(prev => {
        const newSet = new Set(prev);
        newSet.delete(resultId);
        return newSet;
      });
    }
  };

  const handleClone = async () => {
    setCloning(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/${summary.id}/clone`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      alert(`Summary cloned successfully!`);
      onRefresh();
    } catch (err) {
      alert(`Failed to clone summary: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setCloning(false);
    }
  };

  return (
    <div className="border-2 border-steel bg-concrete">
      {/* Header */}
      <div className="p-4 md:p-6">
        <div className="flex flex-col md:flex-row md:justify-between md:items-start gap-4">
          <div className="flex-1">
            <h3 className="font-mono font-bold text-chalk text-lg break-words">{summary.name}</h3>
            <p className="text-sm font-mono text-fog mt-2 break-words">{summary.prompt}</p>
            <div className="flex flex-wrap gap-2 md:gap-3 mt-4 text-xs font-mono">
              {summary.time_of_day && (
                <span className="text-smoke">
                  Time: <span className="text-terminal font-bold">{summary.time_of_day}</span>
                </span>
              )}
              <span className="text-smoke">
                Lookback: <span className="text-terminal font-bold">{summary.lookback_hours}h</span>
              </span>
              <span className="text-smoke">
                Headlines: <span className="text-terminal font-bold">{summary.headline_count}</span>
              </span>
              {(summary.categories || []).length > 0 && (
                <span className="text-smoke">
                  Categories: <span className="text-terminal font-bold">{(summary.categories || []).join(', ')}</span>
                </span>
              )}
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-2">
            <button
              onClick={handleExecute}
              disabled={executing || !summary.active}
              className="flex items-center gap-2 px-3 py-2 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all disabled:opacity-50 disabled:cursor-not-allowed text-xs font-mono font-bold"
            >
              {executing ? <Loader className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
              {executing ? 'EXECUTING...' : 'EXECUTE'}
            </button>

            <button
              onClick={() => onEdit(summary)}
              className="flex items-center gap-2 px-3 py-2 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all text-xs font-mono font-bold"
            >
              <Edit className="w-4 h-4" />
              EDIT
            </button>

            <button
              onClick={handleClone}
              disabled={cloning}
              className="flex items-center gap-2 px-3 py-2 border-2 border-steel text-fog hover:bg-steel hover:text-chalk transition-all text-xs font-mono font-bold disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {cloning ? <Loader className="w-4 h-4 animate-spin" /> : <Copy className="w-4 h-4" />}
              {cloning ? 'CLONING...' : 'CLONE'}
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
      {latestResult?.results && latestResult.results.length > 0 && (
        <div className="p-4 md:p-6 border-t-2 border-steel bg-void/20">
          <h4 className="font-mono font-bold text-sm text-terminal mb-3">LATEST SUMMARY</h4>
          <div className="space-y-3">
            {latestResult.results.map((result) => (
              <div key={result.id} className="bg-concrete border border-steel p-4">
                <div className="flex justify-between items-start mb-2">
                  <span className="font-mono text-xs text-terminal font-bold">
                    {result.model_provider.toUpperCase()} / {result.model_name}
                  </span>
                  <span className="font-mono text-xs text-fog">
                    {formatDateTime(result.created_at)}
                  </span>
                </div>
                <div className="font-mono text-sm text-chalk whitespace-pre-wrap mb-3">
                  {result.summary_text}
                </div>
                <button
                  onClick={() => postToTwitter(latestResult.run.id, result.id)}
                  disabled={postingToTwitter.has(result.id)}
                  className="flex items-center gap-2 px-3 py-1.5 bg-void hover:bg-steel border border-steel font-mono text-xs text-chalk disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {postingToTwitter.has(result.id) ? (
                    <>
                      <Loader className="w-3.5 h-3.5 animate-spin" />
                      <span>POSTING...</span>
                    </>
                  ) : (
                    <>
                      <Twitter className="w-3.5 h-3.5" />
                      <span>POST TO X</span>
                    </>
                  )}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Run History */}
      {expanded && (
        <div className="p-4 md:p-6 bg-void/10 border-t-2 border-steel">
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
                      ) : selectedRunDetail?.results && selectedRunDetail.results.length > 0 ? (
                        <div className="space-y-4">
                          {selectedRunDetail.results.map((result) => (
                            <div key={result.id} className="bg-concrete border border-steel p-4">
                              <div className="flex justify-between items-start mb-2">
                                <span className="font-mono text-xs text-terminal font-bold">
                                  {result.model_provider.toUpperCase()} / {result.model_name}
                                </span>
                                <span className="font-mono text-xs text-fog">
                                  {formatDateTime(result.created_at)}
                                </span>
                              </div>
                              <div className="font-mono text-sm text-chalk whitespace-pre-wrap mb-3">
                                {result.summary_text}
                              </div>
                              <button
                                onClick={() => postToTwitter(selectedRunDetail.run.id, result.id)}
                                disabled={postingToTwitter.has(result.id)}
                                className="flex items-center gap-2 px-3 py-1.5 bg-void hover:bg-steel border border-steel font-mono text-xs text-chalk disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                              >
                                {postingToTwitter.has(result.id) ? (
                                  <>
                                    <Loader className="w-3.5 h-3.5 animate-spin" />
                                    <span>POSTING...</span>
                                  </>
                                ) : (
                                  <>
                                    <Twitter className="w-3.5 h-3.5" />
                                    <span>POST TO X</span>
                                  </>
                                )}
                              </button>
                            </div>
                          ))}
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
                          <p className="text-sm font-mono text-fog">No results available for this run</p>
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

function CreateSummaryModal({ onClose, onSuccess }: { onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState('');
  const [prompt, setPrompt] = useState('');
  const [timeOfDay, setTimeOfDay] = useState('');
  const [lookbackHours, setLookbackHours] = useState(24);
  const [categories, setCategories] = useState<string[]>([]);
  const [headlineCount, setHeadlineCount] = useState(100);
  const [models, setModels] = useState<SummaryModel[]>([
    { provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 },
  ]);
  const [autoPostToTwitter, setAutoPostToTwitter] = useState(false);
  const [includeForecasts, setIncludeForecasts] = useState(false);
  const [creating, setCreating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

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

  const updateModel = (index: number, field: keyof SummaryModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleCreate = async () => {
    if (!name || !prompt || models.length === 0) {
      alert('Please fill in all required fields');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setCreating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries`, {
        method: 'POST',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          prompt,
          time_of_day: timeOfDay || null,
          lookback_hours: lookbackHours,
          categories,
          headline_count: headlineCount,
          models,
          active: true,
          auto_post_to_twitter: autoPostToTwitter,
          include_forecasts: includeForecasts,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to create summary: ${err instanceof Error ? err.message : 'Unknown error'}`);
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
            CREATE SUMMARY
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
              placeholder="e.g., Daily Geopolitical Briefing"
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
          </div>

          {/* Prompt */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">SUMMARY PROMPT *</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="e.g., Summarize the key geopolitical developments and their potential market impacts. Focus on actionable insights."
              rows={4}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
          </div>

          {/* Time of Day */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">TIME OF DAY (OPTIONAL)</label>
            <input
              type="time"
              value={timeOfDay}
              onChange={(e) => setTimeOfDay(e.target.value)}
              placeholder="09:00"
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-terminal focus:outline-none"
            />
            <p className="text-xs font-mono text-fog">When to generate this summary daily (leave empty for manual execution only)</p>
          </div>

          {/* Lookback Hours */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              LOOKBACK HOURS: {lookbackHours}h
            </label>
            <input
              type="range"
              min="1"
              max="168"
              step="1"
              value={lookbackHours}
              onChange={(e) => setLookbackHours(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1h</span>
              <span>168h (7 days)</span>
            </div>
            <p className="text-xs font-mono text-fog">How far back to look for headlines</p>
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
              min="10"
              max="500"
              step="10"
              value={headlineCount}
              onChange={(e) => setHeadlineCount(parseInt(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>10</span>
              <span>500</span>
            </div>
            <p className="text-xs font-mono text-fog">Number of headlines to include in summary</p>
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

          {/* Auto-post to Twitter */}
          <div className="flex items-center gap-3 p-4 border-2 border-steel bg-void/20">
            <input
              type="checkbox"
              id="auto-post-twitter"
              checked={autoPostToTwitter}
              onChange={(e) => setAutoPostToTwitter(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="auto-post-twitter" className="text-sm font-mono text-chalk cursor-pointer">
              <span className="font-bold">Auto-post to X/Twitter</span>
              <p className="text-xs text-fog mt-1">Automatically post the first summary result to Twitter when run completes</p>
            </label>
          </div>

          {/* Include Forecasts */}
          <div className="flex items-center gap-3 p-4 border-2 border-steel bg-void/20">
            <input
              type="checkbox"
              id="include-forecasts"
              checked={includeForecasts}
              onChange={(e) => setIncludeForecasts(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="include-forecasts" className="text-sm font-mono text-chalk cursor-pointer">
              <span className="font-bold">Include Forecasts</span>
              <p className="text-xs text-fog mt-1">Include current forecast probabilities in the summary prompt</p>
            </label>
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
              'CREATE SUMMARY'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

function EditSummaryModal({ summary, onClose, onSuccess }: { summary: Summary; onClose: () => void; onSuccess: () => void }) {
  const [name, setName] = useState(summary.name);
  const [prompt, setPrompt] = useState(summary.prompt);
  const [timeOfDay, setTimeOfDay] = useState(summary.time_of_day || '');
  const [lookbackHours, setLookbackHours] = useState(summary.lookback_hours);
  const [categories, setCategories] = useState<string[]>(summary.categories || []);
  const [headlineCount, setHeadlineCount] = useState(summary.headline_count);
  const [models, setModels] = useState<SummaryModel[]>(
    summary.models && summary.models.length > 0
      ? summary.models
      : [{ provider: 'anthropic', model_name: 'claude-sonnet-4-20250514', api_key: '', weight: 1.0 }]
  );
  const [autoPostToTwitter, setAutoPostToTwitter] = useState(summary.auto_post_to_twitter || false);
  const [includeForecasts, setIncludeForecasts] = useState(summary.include_forecasts || false);
  const [updating, setUpdating] = useState(false);

  const availableCategories = ['geopolitics', 'military', 'economic', 'cyber', 'disaster', 'terrorism', 'diplomacy', 'intelligence', 'humanitarian'];

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

  const updateModel = (index: number, field: keyof SummaryModel, value: any) => {
    const newModels = [...models];
    newModels[index] = { ...newModels[index], [field]: value };
    setModels(newModels);
  };

  const handleUpdate = async () => {
    if (!name || !prompt || models.length === 0) {
      alert('Please fill in all required fields');
      return;
    }

    if (models.some(m => !m.api_key)) {
      alert('Please provide API keys for all models');
      return;
    }

    setUpdating(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/summaries/${summary.id}`, {
        method: 'PUT',
        headers: {
          ...getAuthHeaders(),
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          prompt,
          time_of_day: timeOfDay || null,
          lookback_hours: lookbackHours,
          categories,
          headline_count: headlineCount,
          models,
          active: true,
          auto_post_to_twitter: autoPostToTwitter,
          include_forecasts: includeForecasts,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      onSuccess();
    } catch (err) {
      alert(`Failed to update summary: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setUpdating(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-void/90 backdrop-blur-sm p-4 overflow-y-auto">
      <div className="max-w-4xl w-full border-4 border-electric bg-concrete my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-electric bg-electric/10 flex justify-between items-center">
          <h3 className="font-display font-black text-2xl text-electric flex items-center gap-3">
            <Edit className="w-6 h-6" />
            EDIT SUMMARY
          </h3>
          <button onClick={onClose} className="text-chalk hover:text-electric transition-colors">
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Form - similar to CreateSummaryModal */}
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

          {/* Prompt */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">SUMMARY PROMPT *</label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              rows={4}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
          </div>

          {/* Time of Day */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">TIME OF DAY (OPTIONAL)</label>
            <input
              type="time"
              value={timeOfDay}
              onChange={(e) => setTimeOfDay(e.target.value)}
              className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono focus:border-electric focus:outline-none"
            />
            <p className="text-xs font-mono text-fog">When to generate this summary daily (leave empty for manual execution only)</p>
          </div>

          {/* Lookback Hours */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              LOOKBACK HOURS: {lookbackHours}h
            </label>
            <input
              type="range"
              min="1"
              max="168"
              step="1"
              value={lookbackHours}
              onChange={(e) => setLookbackHours(parseInt(e.target.value))}
              className="w-full"
            />
          </div>

          {/* Categories */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">EVENT CATEGORIES</label>
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

          {/* Headline Count */}
          <div className="space-y-2">
            <label className="block text-sm font-mono text-chalk font-bold">
              HEADLINE COUNT: {headlineCount}
            </label>
            <input
              type="range"
              min="10"
              max="500"
              step="10"
              value={headlineCount}
              onChange={(e) => setHeadlineCount(parseInt(e.target.value))}
              className="w-full"
            />
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
                    className="w-full px-3 py-2 border border-steel bg-void text-chalk font-mono text-sm focus:border-electric focus:outline-none"
                  />
                </div>
              </div>
            ))}
          </div>

          {/* Auto-post to Twitter */}
          <div className="flex items-center gap-3 p-4 border-2 border-steel bg-void/20">
            <input
              type="checkbox"
              id="edit-auto-post-twitter"
              checked={autoPostToTwitter}
              onChange={(e) => setAutoPostToTwitter(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="edit-auto-post-twitter" className="text-sm font-mono text-chalk cursor-pointer">
              <span className="font-bold">Auto-post to X/Twitter</span>
              <p className="text-xs text-fog mt-1">Automatically post the first summary result to Twitter when run completes</p>
            </label>
          </div>

          {/* Include Forecasts */}
          <div className="flex items-center gap-3 p-4 border-2 border-steel bg-void/20">
            <input
              type="checkbox"
              id="edit-include-forecasts"
              checked={includeForecasts}
              onChange={(e) => setIncludeForecasts(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="edit-include-forecasts" className="text-sm font-mono text-chalk cursor-pointer">
              <span className="font-bold">Include Forecasts</span>
              <p className="text-xs text-fog mt-1">Include current forecast probabilities in the summary prompt</p>
            </label>
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
              'UPDATE SUMMARY'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
