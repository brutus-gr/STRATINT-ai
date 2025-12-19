import { useState, useEffect } from 'react';
import { Header } from './components/Header';
import { EventCard } from './components/EventCard';
import { FilterPanel } from './components/FilterPanel';
import { MCPInstructions } from './components/MCPInstructions';
import { PublicForecastChart } from './components/PublicForecastChart';
import { Activity, TrendingUp, Target, Twitter } from 'lucide-react';
import type { Event, EventFilters } from './types';
import { API_BASE_URL } from './utils/api';
import { formatUnits, formatScheduleInterval } from './utils/format';

type Tab = 'signals' | 'forecasts' | 'strategies';

interface Forecast {
  id: string;
  name: string;
  proposition: string;
  prediction_type: string;
  units: string;
  target_date?: string;
  categories: string[];
  headline_count: number;
  iterations: number;
  context_urls: string[];
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

interface Strategy {
  id: string;
  name: string;
  prompt: string;
  investment_symbols: string[];
  categories: string[];
  headline_count: number;
  iterations: number;
  forecast_ids: string[];
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

interface StrategyResult {
  averaged_allocations: Record<string, number>;
  normalized_allocations: Record<string, number>;
  normalization_reasoning: string;
  model_count: number;
  iteration_count: number;
  consensus_variance: Record<string, number>;
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

interface StrategyWithResult {
  strategy: Strategy;
  latestRun?: {
    run: StrategyRun;
    result?: StrategyResult;
  };
}

type ChartViewMode = 'hourly' | '4h' | 'daily';

function App() {
  // Initialize activeTab based on current pathname to avoid flash of wrong content
  const getInitialTab = (): Tab => {
    const path = window.location.pathname;
    if (path === '/forecasts') return 'forecasts';
    if (path === '/strategies') return 'strategies';
    return 'signals';
  };

  const [activeTab, setActiveTab] = useState<Tab>(getInitialTab());
  const [filters, setFilters] = useState<EventFilters>({});
  const [events, setEvents] = useState<Event[]>([]);
  const [forecasts, setForecasts] = useState<Forecast[]>([]);
  const [strategies, setStrategies] = useState<StrategyWithResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [forecastsLoading, setForecastsLoading] = useState(false);
  const [strategiesLoading, setStrategiesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMoreEvents, setHasMoreEvents] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [chartViewMode, setChartViewMode] = useState<ChartViewMode>('hourly');

  // Check URL path on mount to set initial tab and handle browser back/forward
  useEffect(() => {
    const handleLocationChange = () => {
      const path = window.location.pathname;
      if (path === '/forecasts') {
        setActiveTab('forecasts');
      } else if (path === '/strategies') {
        setActiveTab('strategies');
      } else {
        setActiveTab('signals');
      }
    };

    // Set initial tab based on URL
    handleLocationChange();

    // Listen for browser back/forward navigation
    window.addEventListener('popstate', handleLocationChange);

    return () => {
      window.removeEventListener('popstate', handleLocationChange);
    };
  }, []);

  // Update URL when tab changes
  const handleTabChange = (tab: Tab) => {
    setActiveTab(tab);
    const path = tab === 'signals' ? '/' : `/${tab}`;
    window.history.pushState({}, '', path);
  };

  // Fetch events from API with server-side filtering
  useEffect(() => {
    // Only fetch events if we're on the signals tab
    if (activeTab !== 'signals') {
      return;
    }

    let isInitialLoad = true;

    // Reset pagination state when filters change
    setHasMoreEvents(true);

    const fetchEvents = async () => {
      try {
        if (isInitialLoad) {
          setLoading(true);
        }

        // Build query parameters from filters
        const params = new URLSearchParams({
          limit: '25',
          sort_by: 'timestamp',
          sort_order: 'desc'
        });

        // Add filter parameters
        if (filters.categories && filters.categories.length > 0) {
          params.append('categories', filters.categories.join(','));
        }

        if (filters.minMagnitude !== undefined) {
          params.append('min_magnitude', filters.minMagnitude.toString());
        }

        if (filters.minConfidence !== undefined) {
          params.append('min_confidence', filters.minConfidence.toString());
        }

        if (filters.search) {
          params.append('search', filters.search);
        }

        if (filters.timeRange) {
          params.append('time_range', filters.timeRange);
        }

        const response = await fetch(`${API_BASE_URL}/api/events?${params.toString()}`);
        if (!response.ok) throw new Error('Failed to fetch events');
        const data = await response.json();
        const newEvents = data.events || [];

        // Check if we got fewer events than requested (means no more data)
        setHasMoreEvents(newEvents.length >= 25);

        if (isInitialLoad) {
          // On initial load, replace everything
          setEvents(newEvents);
          setLoading(false);
          isInitialLoad = false;
        } else {
          // On subsequent fetches, merge intelligently
          setEvents(currentEvents => {
            // Create a map of existing event IDs for fast lookup
            const existingIds = new Set(currentEvents.map((e: Event) => e.id));
            // Find truly new events (not in current list)
            const freshEvents = newEvents.filter((e: Event) => !existingIds.has(e.id));
            // Prepend new events to maintain chronological order
            return freshEvents.length > 0 ? [...freshEvents, ...currentEvents] : currentEvents;
          });
        }
      } catch (err) {
        console.error('Error fetching events:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        if (isInitialLoad) {
          setLoading(false);
        }
      }
    };

    fetchEvents();
    // Refresh every 30 seconds
    const interval = setInterval(fetchEvents, 30000);
    return () => clearInterval(interval);
  }, [filters, activeTab]);

  // Load more events function
  const loadMoreEvents = async () => {
    if (loadingMore || !hasMoreEvents) return;

    setLoadingMore(true);
    try {
      // Build query parameters with offset
      const params = new URLSearchParams({
        limit: '25',
        sort_by: 'timestamp',
        sort_order: 'desc',
        offset: events.length.toString()
      });

      // Add filter parameters
      if (filters.categories && filters.categories.length > 0) {
        params.append('categories', filters.categories.join(','));
      }

      if (filters.minMagnitude !== undefined) {
        params.append('min_magnitude', filters.minMagnitude.toString());
      }

      if (filters.minConfidence !== undefined) {
        params.append('min_confidence', filters.minConfidence.toString());
      }

      if (filters.search) {
        params.append('search', filters.search);
      }

      if (filters.timeRange) {
        params.append('time_range', filters.timeRange);
      }

      const response = await fetch(`${API_BASE_URL}/api/events?${params.toString()}`);
      if (!response.ok) throw new Error('Failed to fetch more events');
      const data = await response.json();
      const newEvents = data.events || [];

      // If we got fewer than requested, there are no more events
      setHasMoreEvents(newEvents.length >= 25);

      // Append new events to existing ones
      if (newEvents.length > 0) {
        setEvents(currentEvents => [...currentEvents, ...newEvents]);
      }
    } catch (err) {
      console.error('Error loading more events:', err);
    } finally {
      setLoadingMore(false);
    }
  };

  // Fetch public forecasts when forecasts tab is active
  useEffect(() => {
    if (activeTab === 'forecasts') {
      const fetchForecasts = async () => {
        setForecastsLoading(true);
        try {
          const response = await fetch(`${API_BASE_URL}/api/forecasts`);
          if (!response.ok) throw new Error('Failed to fetch forecasts');
          const data = await response.json();
          setForecasts(data.forecasts || []);
        } catch (err) {
          console.error('Error fetching forecasts:', err);
        } finally {
          setForecastsLoading(false);
        }
      };
      fetchForecasts();
    }
  }, [activeTab]);

  // Fetch public strategies when strategies tab is active
  useEffect(() => {
    if (activeTab === 'strategies') {
      const fetchStrategies = async () => {
        setStrategiesLoading(true);
        try {
          const response = await fetch(`${API_BASE_URL}/api/strategies`);
          if (!response.ok) throw new Error('Failed to fetch strategies');
          const strategiesData = await response.json();

          // Fetch latest result for each strategy
          const strategiesWithResults: StrategyWithResult[] = await Promise.all(
            (strategiesData || []).map(async (strategy: Strategy) => {
              try {
                const resultResponse = await fetch(`${API_BASE_URL}/api/strategies/${strategy.id}/latest`);
                if (resultResponse.ok) {
                  const latestRun = await resultResponse.json();
                  return { strategy, latestRun };
                }
                return { strategy };
              } catch (err) {
                console.error(`Error fetching latest result for strategy ${strategy.id}:`, err);
                return { strategy };
              }
            })
          );

          setStrategies(strategiesWithResults);
        } catch (err) {
          console.error('Error fetching strategies:', err);
        } finally {
          setStrategiesLoading(false);
        }
      };
      fetchStrategies();
    }
  }, [activeTab]);

  return (
    <div className="min-h-screen bg-void text-chalk">
      {/* Scan line effect */}
      <div className="scan-line" />
      
      {/* Header */}
      <Header />

      {/* Main Layout */}
      <div className="pt-16">
        <div className="flex flex-col lg:flex-row gap-0 justify-center lg:min-h-screen">
          {/* Main Content - Event Feed */}
          <main className="flex-1 p-4 md:p-8 space-y-6 max-w-5xl mx-auto w-full lg:pb-8">
            {/* Tab Navigation */}
            <div className="flex gap-2 border-2 border-steel bg-concrete/30 p-2">
              <button
                onClick={() => handleTabChange('signals')}
                className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 font-mono font-bold text-sm transition-all ${
                  activeTab === 'signals'
                    ? 'bg-terminal text-void border-2 border-terminal'
                    : 'bg-void border-2 border-steel text-chalk hover:border-terminal hover:text-terminal'
                }`}
              >
                <Activity className="w-4 h-4" />
                <span className="hidden sm:inline">SIGNALS</span>
              </button>
              <button
                onClick={() => handleTabChange('forecasts')}
                className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 font-mono font-bold text-sm transition-all ${
                  activeTab === 'forecasts'
                    ? 'bg-terminal text-void border-2 border-terminal'
                    : 'bg-void border-2 border-steel text-chalk hover:border-terminal hover:text-terminal'
                }`}
              >
                <TrendingUp className="w-4 h-4" />
                <span className="hidden sm:inline">FORECASTS</span>
              </button>
              <button
                onClick={() => handleTabChange('strategies')}
                className={`flex-1 flex items-center justify-center gap-2 px-4 py-3 font-mono font-bold text-sm transition-all ${
                  activeTab === 'strategies'
                    ? 'bg-terminal text-void border-2 border-terminal'
                    : 'bg-void border-2 border-steel text-chalk hover:border-terminal hover:text-terminal'
                }`}
              >
                <Target className="w-4 h-4" />
                <span className="hidden sm:inline">STRATEGIES</span>
              </button>
            </div>

            {/* Signals Tab */}
            {activeTab === 'signals' && (
              <>
                {/* Section Header */}
                <div className="border-l-4 border-terminal pl-4 md:pl-6">
                  <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-2">
                    <div>
                      <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
                        SIGNAL STREAM
                      </h2>
                      <p className="text-xs md:text-sm text-smoke font-mono mt-2">
                        MOST RECENT 25 SIGNALS / REAL-TIME INTELLIGENCE FEED
                      </p>
                    </div>
                    <div className="text-xs font-mono text-fog">
                      LAST UPDATE: {new Date().toLocaleTimeString('en-US', { hour12: false })}
                    </div>
                  </div>
                </div>

                {/* MCP Server Instructions */}
                <MCPInstructions />

            {/* Event Cards */}
            <div className="space-y-6">
              {loading ? (
                <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                  <div className="text-5xl text-terminal animate-pulse">◉</div>
                  <div className="space-y-2">
                    <p className="text-chalk font-mono text-lg font-bold">LOADING EVENTS...</p>
                    <p className="text-fog font-mono text-sm">Connecting to intelligence feed</p>
                  </div>
                </div>
              ) : error ? (
                <div className="border-2 border-warning bg-concrete p-16 text-center space-y-6">
                  <div className="text-5xl text-warning">⚠</div>
                  <div className="space-y-2">
                    <p className="text-chalk font-mono text-lg font-bold">CONNECTION ERROR</p>
                    <p className="text-fog font-mono text-sm">{error}</p>
                  </div>
                </div>
              ) : events.length > 0 ? (
                events.map((event, index) => (
                  <EventCard key={event.id} event={event} index={index} />
                ))
              ) : (
                <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                  <div className="text-5xl text-steel/50">◉</div>
                  <div className="space-y-2">
                    <p className="text-chalk font-mono text-lg font-bold">
                      {Object.keys(filters).length > 0 ? 'NO EVENTS MATCH FILTERS' : 'NO EVENTS AVAILABLE'}
                    </p>
                    <p className="text-fog font-mono text-sm">
                      {Object.keys(filters).length > 0
                        ? 'Adjust filters or clear to see all events'
                        : 'Sources are being ingested. Events will appear once enriched.'
                      }
                    </p>
                  </div>
                  {Object.keys(filters).length > 0 && (
                    <button
                      onClick={() => setFilters({})}
                      className="mt-4 px-6 py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold"
                    >
                      [CLEAR ALL FILTERS]
                    </button>
                  )}
                </div>
              )}
            </div>

                {/* Load More */}
                {events.length > 0 && hasMoreEvents && (
                  <button
                    onClick={loadMoreEvents}
                    disabled={loadingMore}
                    className="w-full py-4 border-2 border-steel bg-concrete hover:bg-iron hover:border-terminal transition-all font-mono text-sm text-chalk font-bold group disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <span className="group-hover:text-terminal transition-colors">
                      {loadingMore ? '[LOADING MORE...]' : '[LOAD MORE SIGNALS]'}
                    </span>
                  </button>
                )}
                {events.length > 0 && !hasMoreEvents && (
                  <div className="w-full py-4 border-2 border-steel bg-concrete font-mono text-sm text-fog text-center">
                    [END OF SIGNAL STREAM]
                  </div>
                )}
              </>
            )}

            {/* Forecasts Tab */}
            {activeTab === 'forecasts' && (
              <>
                {/* Section Header */}
                <div className="border-l-4 border-electric pl-4 md:pl-6">
                  <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4">
                    <div>
                      <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
                        PUBLIC FORECASTS
                      </h2>
                      <p className="text-xs md:text-sm text-smoke font-mono mt-2">
                        AI-POWERED PREDICTIVE INTELLIGENCE / PROBABILITY ESTIMATES
                      </p>
                    </div>

                    {/* Global Chart View Mode Selector */}
                    <select
                      value={chartViewMode}
                      onChange={(e) => setChartViewMode(e.target.value as ChartViewMode)}
                      className="border-2 border-steel bg-void text-chalk font-mono text-xs font-bold px-3 py-2 hover:border-iron focus:border-electric focus:outline-none"
                    >
                      <option value="hourly">HOURLY (24H)</option>
                      <option value="4h">4-HOUR (OHLC)</option>
                      <option value="daily">DAILY (OHLC)</option>
                    </select>
                  </div>
                </div>

                {/* Forecasts List */}
                <div className="space-y-6">
                  {forecastsLoading ? (
                    <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                      <div className="text-5xl text-electric animate-pulse">◉</div>
                      <div className="space-y-2">
                        <p className="text-chalk font-mono text-lg font-bold">LOADING FORECASTS...</p>
                        <p className="text-fog font-mono text-sm">Fetching predictive intelligence</p>
                      </div>
                    </div>
                  ) : forecasts.length > 0 ? (
                    forecasts.map((forecast) => (
                      <div key={forecast.id} className="border-2 border-steel bg-concrete p-6 space-y-4">
                        <div>
                          <h3 className="font-mono font-bold text-chalk text-lg">{forecast.name}</h3>
                          <p className="text-sm font-mono text-fog mt-2">{forecast.proposition}</p>
                        </div>
                        <div className="flex flex-wrap gap-3 text-xs font-mono">
                          <span className="text-smoke">
                            Type: <span className="text-electric font-bold">{forecast.prediction_type === 'percentile' ? 'DISTRIBUTION' : 'POINT ESTIMATE'}</span>
                          </span>
                          <span className="text-smoke">
                            Units: <span className="text-electric font-bold">{formatUnits(forecast.units)}</span>
                          </span>
                          {forecast.target_date && (
                            <span className="text-smoke">
                              Target: <span className="text-electric font-bold">{new Date(forecast.target_date).toLocaleDateString()}</span>
                            </span>
                          )}
                          {forecast.schedule_enabled && forecast.schedule_interval && (
                            <span className="text-smoke">
                              <span className="text-electric font-bold">{formatScheduleInterval(forecast.schedule_interval)}</span>
                            </span>
                          )}
                        </div>
                        {/* Chart */}
                        {forecast.prediction_type === 'percentile' && (
                          <PublicForecastChart forecastId={forecast.id} viewMode={chartViewMode} />
                        )}
                      </div>
                    ))
                  ) : (
                    <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                      <div className="flex justify-center">
                        <TrendingUp className="w-20 h-20 text-steel/50" />
                      </div>
                      <div className="space-y-2">
                        <p className="text-chalk font-mono text-lg font-bold">NO PUBLIC FORECASTS</p>
                        <p className="text-fog font-mono text-sm">
                          Public forecasts will appear here when available
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              </>
            )}

            {/* Strategies Tab */}
            {activeTab === 'strategies' && (
              <>
                {/* Tab Header */}
                <div className="border-l-4 border-terminal pl-4 md:pl-6">
                  <h2 className="font-display font-black text-2xl md:text-3xl text-chalk tracking-tight">
                    PORTFOLIO STRATEGIES
                  </h2>
                  <p className="text-xs md:text-sm text-smoke font-mono mt-2">
                    AI-generated portfolio allocation strategies based on emerging signals and market forecasts
                  </p>
                </div>

                <div className="space-y-6">
                  {strategiesLoading ? (
                    <div className="border-2 border-steel bg-concrete p-16 text-center">
                      <div className="text-5xl text-terminal animate-pulse mb-4">◉</div>
                      <p className="text-chalk font-mono text-lg font-bold">LOADING STRATEGIES...</p>
                    </div>
                  ) : strategies.length === 0 ? (
                    <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                      <div className="flex justify-center">
                        <Target className="w-20 h-20 text-steel/50" />
                      </div>
                      <div className="space-y-3">
                        <p className="text-chalk font-mono text-lg font-bold">NO PUBLIC STRATEGIES</p>
                        <p className="text-fog font-mono text-sm">
                          Public strategies will appear here when available
                        </p>
                      </div>
                    </div>
                  ) : (
                    strategies.map((item) => (
                      <div key={item.strategy.id} className="border-2 border-steel bg-concrete p-6">
                        {/* Strategy Header */}
                        <div className="mb-4">
                          <h3 className="font-display font-black text-2xl text-terminal mb-2">
                            {item.strategy.name}
                          </h3>
                          <p className="text-sm font-mono text-fog whitespace-pre-wrap">
                            {item.strategy.prompt}
                          </p>
                        </div>

                        {/* Latest Result */}
                        {item.latestRun && item.latestRun.result ? (
                          <div className="space-y-4">
                            {/* Timestamp */}
                            <div className="text-xs font-mono text-smoke">
                              LAST UPDATED: {new Date(item.latestRun.run.run_at).toLocaleString('en-US', {
                                dateStyle: 'short',
                                timeStyle: 'short'
                              })}
                            </div>

                            {/* Portfolio Allocations */}
                            <div className="space-y-2">
                              <h4 className="text-sm font-mono text-chalk font-bold mb-3">
                                RECOMMENDED ALLOCATIONS
                              </h4>
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                                {Object.entries(item.latestRun.result.normalized_allocations)
                                  .sort(([, a], [, b]) => b - a)
                                  .map(([symbol, allocation]) => (
                                    <div key={symbol} className="border border-steel bg-void p-3">
                                      <div className="flex justify-between items-center mb-2">
                                        <span className="font-mono font-bold text-terminal text-lg">
                                          {symbol}
                                        </span>
                                        <span className="font-mono font-black text-chalk text-xl">
                                          {allocation.toFixed(1)}%
                                        </span>
                                      </div>
                                      <div className="w-full bg-steel h-2">
                                        <div
                                          className="bg-terminal h-2 transition-all"
                                          style={{ width: `${allocation}%` }}
                                        />
                                      </div>
                                    </div>
                                  ))}
                              </div>
                            </div>

                            {/* Metadata */}
                            <div className="flex flex-wrap gap-4 text-xs font-mono text-smoke border-t border-steel pt-3">
                              <span>MODELS: {item.latestRun.result.model_count}</span>
                              <span>ITERATIONS: {item.latestRun.result.iteration_count}</span>
                              <span>SIGNALS: {item.latestRun.run.headline_count}</span>
                            </div>
                          </div>
                        ) : (
                          <div className="border border-steel bg-void p-6 text-center">
                            <p className="text-sm font-mono text-fog">
                              No results available yet
                            </p>
                          </div>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </>
            )}
          </main>

          {/* Sidebar - Filters & Widgets - Only show on Signals tab */}
          {activeTab === 'signals' && (
            <aside className="w-full lg:w-96 lg:sticky lg:top-16 lg:self-start lg:h-[calc(100vh-4rem)] lg:-mt-16 flex flex-col gap-6 p-4 md:p-6 lg:pt-4 border-t lg:border-t-0 lg:border-l border-steel bg-concrete/30 order-first lg:order-last">
              {/* Filters */}
              <div className="flex-1">
                <FilterPanel filters={filters} onFiltersChange={setFilters} />
              </div>
            </aside>
          )}
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-steel bg-concrete mt-12">
        <div className="px-4 md:px-6 py-4 flex flex-col md:flex-row items-center justify-between gap-3 text-xs font-mono text-smoke">
          <div className="text-center md:text-left">STRATINT v1.0</div>
          <div className="flex items-center gap-4">
            <a href="/api-docs" className="hover:text-terminal transition-colors">API DOCS</a>
            <span>|</span>
            <a href="https://x.com/STRATINT_ai" target="_blank" rel="noopener noreferrer" className="hover:text-terminal transition-colors flex items-center gap-1">
              <Twitter className="w-3 h-3" />
              <span>@STRATINT_AI</span>
            </a>
            <span>|</span>
            <a href="mailto:contact@stratint.ai" className="hover:text-terminal transition-colors">CONTACT</a>
            {/* Admin and GitHub links hidden for production */}
            {/* <span>|</span>
            <a href="/admin" className="hover:text-terminal transition-colors">ADMIN</a>
            <span>|</span>
            <a href="https://github.com/brutus-gr/STRATINT" target="_blank" rel="noopener noreferrer" className="hover:text-terminal transition-colors">GITHUB</a> */}
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
