import { useState, useEffect } from 'react';
import {
  TrendingDown,
  AlertCircle,
  CheckCircle,
  Clock,
  XCircle,
  ArrowRight,
  Activity,
  Zap,
  Filter,
  BarChart3,
  RotateCcw,
  Trash2,
  ExternalLink
} from 'lucide-react';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';
import { formatDateTime } from '../utils/dateFormat';

interface EnrichmentInfo {
  source_id: string;
  source_type: string;
  source_url: string;
  source_title: string;
  published_at: string;
  enrichment_status: string;
  enrichment_error?: string;
  enriched_at?: string;
  event_id?: string;
  event_title?: string;
  event_status?: string;
}

interface PipelineMetrics {
  sources_total: number;
  sources_by_status: {
    pending: number;
    in_progress: number;
    completed: number;
    failed: number;
    skipped: number;
  };
  enrichment_by_status: {
    pending: number;
    enriching: number;
    completed: number;
    failed: number;
  };
  sources_recent_count: number;
  events_total: number;
  events_by_status: {
    pending: number;
    published: number;
    rejected: number;
  };
  events_recent_count: number;
  scrape_completion_rate: number;
  enrichment_rate: number;
  publish_rate: number;
  bottleneck: string;
  bottleneck_reason: string;
}

export function PipelineFunnelTab() {
  const [metrics, setMetrics] = useState<PipelineMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [requeueing, setRequeueing] = useState(false);
  const [requeueMessage, setRequeueMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [deletingEnrichments, setDeletingEnrichments] = useState(false);
  const [deleteEnrichmentsMessage, setDeleteEnrichmentsMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [deletingPendingSources, setDeletingPendingSources] = useState(false);
  const [deletePendingSourcesMessage, setDeletePendingSourcesMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [recentEnrichments, setRecentEnrichments] = useState<EnrichmentInfo[]>([]);

  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/pipeline/metrics`, {
          headers: getAuthHeaders(),
        });
        if (!response.ok) throw new Error('Failed to fetch pipeline metrics');
        const data = await response.json();
        setMetrics(data);
        setError(null);
      } catch (err) {
        console.error('Error fetching pipeline metrics:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
    // Refresh every 5 seconds
    const interval = setInterval(fetchMetrics, 5000);
    return () => clearInterval(interval);
  }, []);

  // Fetch recent enrichments
  useEffect(() => {
    const fetchEnrichments = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/admin/recent-enrichments`, {
          headers: getAuthHeaders(),
        });
        if (!response.ok) throw new Error('Failed to fetch enrichments');
        const data = await response.json();
        setRecentEnrichments(data.enrichments || []);
      } catch (err) {
        console.error('Error fetching enrichments:', err);
      }
    };

    fetchEnrichments();
    // Refresh every 10 seconds
    const interval = setInterval(fetchEnrichments, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleRequeueEnrichments = async () => {
    if (!confirm('Requeue all failed enrichments back to pending? This will retry AI enrichment for them.')) {
      return;
    }

    setRequeueing(true);
    setRequeueMessage(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/requeue-enrichments`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });

      const data = await response.json();

      if (response.ok) {
        setRequeueMessage({
          type: 'success',
          text: `Successfully requeued ${data.requeued_count || 0} failed enrichments (${data.total_pending || 0} total pending)`,
        });
      } else {
        setRequeueMessage({
          type: 'error',
          text: data.message || 'Failed to requeue failed enrichments',
        });
      }
    } catch (err) {
      setRequeueMessage({
        type: 'error',
        text: err instanceof Error ? err.message : 'Unknown error',
      });
    } finally {
      setRequeueing(false);
    }
  };


  const handleDeleteFailedEnrichments = async () => {
    const failedCount = metrics?.enrichment_by_status.failed || 0;

    if (!confirm(`Permanently delete ${failedCount} failed enrichments? This cannot be undone.`)) {
      return;
    }

    setDeletingEnrichments(true);
    setDeleteEnrichmentsMessage(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/delete-failed-enrichments`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      const data = await response.json();

      if (response.ok) {
        setDeleteEnrichmentsMessage({
          type: 'success',
          text: `Successfully deleted ${data.deleted_count || 0} enrichments`,
        });
      } else {
        setDeleteEnrichmentsMessage({
          type: 'error',
          text: data.message || 'Failed to delete enrichments',
        });
      }
    } catch (err) {
      setDeleteEnrichmentsMessage({
        type: 'error',
        text: err instanceof Error ? err.message : 'Unknown error',
      });
    } finally {
      setDeletingEnrichments(false);
    }
  };

  const handleDeletePendingSources = async () => {
    const pendingCount = metrics?.sources_by_status.pending || 0;

    if (!confirm(`Permanently delete ${pendingCount} pending sources? This cannot be undone. These are orphaned sources from before switching to RSS-only mode.`)) {
      return;
    }

    setDeletingPendingSources(true);
    setDeletePendingSourcesMessage(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/delete-pending-sources`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      const data = await response.json();

      if (response.ok) {
        setDeletePendingSourcesMessage({
          type: 'success',
          text: `Successfully deleted ${data.deleted_count || 0} pending sources`,
        });
      } else {
        setDeletePendingSourcesMessage({
          type: 'error',
          text: data.message || 'Failed to delete pending sources',
        });
      }
    } catch (err) {
      setDeletePendingSourcesMessage({
        type: 'error',
        text: err instanceof Error ? err.message : 'Unknown error',
      });
    } finally {
      setDeletingPendingSources(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3 text-terminal">
          <Activity className="w-6 h-6 animate-pulse" />
          <span className="font-mono text-sm">LOADING PIPELINE METRICS...</span>
        </div>
      </div>
    );
  }

  if (error || !metrics) {
    return (
      <div className="border-2 border-threat-critical bg-threat-critical/10 p-6">
        <div className="flex items-center gap-3">
          <AlertCircle className="w-6 h-6 text-threat-critical" />
          <div>
            <h3 className="font-mono font-bold text-threat-critical">ERROR LOADING METRICS</h3>
            <p className="font-mono text-sm text-chalk mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  // Calculate totals for funnel
  const sourcesCompleted = metrics.sources_by_status.completed;
  const eventsCreated = metrics.events_total;
  const eventsPublished = metrics.events_by_status.published;

  // Bottleneck styling
  const getBottleneckColor = (stage: string) => {
    if (metrics.bottleneck === 'none') return 'terminal';
    if (metrics.bottleneck === stage) return 'threat-critical';
    return 'steel';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
              PROCESSING PIPELINE
            </h2>
            <p className="text-sm text-smoke font-mono mt-2">
              Real-time view of the RSS ingestion and enrichment funnel
            </p>
          </div>
          <div className="flex gap-3">
            {metrics && metrics.sources_by_status.pending > 0 && (
              <button
                onClick={handleDeletePendingSources}
                disabled={deletingPendingSources}
                className="flex items-center gap-2 px-4 py-2 bg-threat-critical hover:bg-threat-critical/90 disabled:bg-steel disabled:cursor-not-allowed text-void font-mono font-bold text-sm transition-colors"
              >
                <Trash2 className="w-4 h-4" />
                {deletingPendingSources ? 'DELETING...' : `DELETE ${metrics.sources_by_status.pending} PENDING SOURCES`}
              </button>
            )}
            {metrics && metrics.enrichment_by_status.failed > 0 && (
              <button
                onClick={handleRequeueEnrichments}
                disabled={requeueing}
                className="flex items-center gap-2 px-4 py-2 bg-threat-medium hover:bg-threat-medium/90 disabled:bg-steel disabled:cursor-not-allowed text-void font-mono font-bold text-sm transition-colors"
              >
                <RotateCcw className={`w-4 h-4 ${requeueing ? 'animate-spin' : ''}`} />
                {requeueing ? 'REQUEUING...' : `REQUEUE ${metrics.enrichment_by_status.failed} FAILED ENRICHMENTS`}
              </button>
            )}
            {metrics && metrics.enrichment_by_status.failed > 0 && (
              <button
                onClick={handleDeleteFailedEnrichments}
                disabled={deletingEnrichments}
                className="flex items-center gap-2 px-4 py-2 bg-error hover:bg-error/90 disabled:bg-steel disabled:cursor-not-allowed text-void font-mono font-bold text-sm transition-colors"
              >
                <Trash2 className="w-4 h-4" />
                {deletingEnrichments ? 'DELETING...' : `DELETE ${metrics.enrichment_by_status.failed} FAILED ENRICHMENTS`}
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Requeue Message */}
      {requeueMessage && (
        <div className={`border-2 ${requeueMessage.type === 'success' ? 'border-terminal bg-terminal/10' : 'border-error bg-error/10'} p-4`}>
          <div className="flex items-center gap-3">
            {requeueMessage.type === 'success' ? (
              <CheckCircle className="w-5 h-5 text-terminal" />
            ) : (
              <XCircle className="w-5 h-5 text-error" />
            )}
            <span className={`font-mono text-sm ${requeueMessage.type === 'success' ? 'text-terminal' : 'text-error'}`}>
              {requeueMessage.text}
            </span>
          </div>
        </div>
      )}

      {/* Delete Enrichments Message */}
      {deleteEnrichmentsMessage && (
        <div className={`border-2 ${deleteEnrichmentsMessage.type === 'success' ? 'border-terminal bg-terminal/10' : 'border-error bg-error/10'} p-4`}>
          <div className="flex items-center gap-3">
            {deleteEnrichmentsMessage.type === 'success' ? (
              <CheckCircle className="w-5 h-5 text-terminal" />
            ) : (
              <XCircle className="w-5 h-5 text-error" />
            )}
            <span className={`font-mono text-sm ${deleteEnrichmentsMessage.type === 'success' ? 'text-terminal' : 'text-error'}`}>
              {deleteEnrichmentsMessage.text}
            </span>
          </div>
        </div>
      )}

      {/* Delete Pending Sources Message */}
      {deletePendingSourcesMessage && (
        <div className={`border-2 ${deletePendingSourcesMessage.type === 'success' ? 'border-terminal bg-terminal/10' : 'border-error bg-error/10'} p-4`}>
          <div className="flex items-center gap-3">
            {deletePendingSourcesMessage.type === 'success' ? (
              <CheckCircle className="w-5 h-5 text-terminal" />
            ) : (
              <XCircle className="w-5 h-5 text-error" />
            )}
            <span className={`font-mono text-sm ${deletePendingSourcesMessage.type === 'success' ? 'text-terminal' : 'text-error'}`}>
              {deletePendingSourcesMessage.text}
            </span>
          </div>
        </div>
      )}

      {/* Bottleneck Alert */}
      {metrics.bottleneck !== 'none' && (
        <div className="border-2 border-threat-critical bg-threat-critical/10 p-6">
          <div className="flex items-start gap-4">
            <AlertCircle className="w-6 h-6 text-threat-critical flex-shrink-0 mt-1" />
            <div>
              <h3 className="font-mono font-bold text-threat-critical text-lg">
                BOTTLENECK DETECTED: {metrics.bottleneck.toUpperCase()}
              </h3>
              <p className="font-mono text-sm text-chalk mt-2">
                {metrics.bottleneck_reason}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Visual Funnel */}
      <div className="space-y-4">
        {/* Stage 1: RSS Ingestion */}
        <FunnelStage
          title="1. RSS FEED INGESTION"
          icon={<Activity className="w-5 h-5" />}
          count={metrics.sources_total}
          width={100}
          color="terminal"
          description="Articles fetched from RSS feeds"
        />

        <ArrowRight className="w-6 h-6 text-steel mx-auto" />

        {/* Stage 2: Content Processing (using RSS descriptions) */}
        <FunnelStage
          title="2. CONTENT PROCESSING"
          icon={<Zap className="w-5 h-5" />}
          count={sourcesCompleted}
          width={metrics.sources_total > 0 ? (sourcesCompleted / metrics.sources_total) * 100 : 0}
          color={getBottleneckColor('processing')}
          description="Using RSS descriptions as content"
          substats={[
            { label: 'Pending', value: metrics.sources_by_status.pending, color: 'text-smoke' },
            { label: 'Processed', value: metrics.sources_by_status.completed, color: 'text-terminal' },
            { label: 'Skipped', value: metrics.sources_by_status.skipped, color: 'text-fog' },
          ]}
        />

        <ArrowRight className="w-6 h-6 text-steel mx-auto" />

        {/* Stage 3: AI Enrichment */}
        <FunnelStage
          title="3. AI ENRICHMENT"
          icon={<Filter className="w-5 h-5" />}
          count={eventsCreated}
          width={sourcesCompleted > 0 ? (eventsCreated / sourcesCompleted) * 100 : 0}
          color={getBottleneckColor('enrichment')}
          description={`${metrics.enrichment_rate.toFixed(1)}% enrichment rate`}
          substats={[
            { label: 'Pending', value: metrics.enrichment_by_status.pending, color: 'text-smoke' },
            { label: 'Enriching', value: metrics.enrichment_by_status.enriching, color: 'text-terminal' },
            { label: 'Failed', value: metrics.enrichment_by_status.failed, color: 'text-threat-critical' },
          ]}
        />

        <ArrowRight className="w-6 h-6 text-steel mx-auto" />

        {/* Stage 4: Publication */}
        <FunnelStage
          title="4. PUBLISHED EVENTS"
          icon={<CheckCircle className="w-5 h-5" />}
          count={eventsPublished}
          width={eventsCreated > 0 ? (eventsPublished / eventsCreated) * 100 : 0}
          color={getBottleneckColor('thresholds')}
          description={`${metrics.publish_rate.toFixed(1)}% publish rate`}
        />
      </div>

      {/* Detailed Breakdown Grid */}
      <div className="grid grid-cols-2 gap-6 mt-8">
        {/* Sources Breakdown */}
        <div className="border-2 border-steel bg-concrete/30 p-6">
          <h3 className="font-mono font-bold text-terminal text-sm mb-4 flex items-center gap-2">
            <BarChart3 className="w-4 h-4" />
            SOURCES BREAKDOWN
          </h3>
          <div className="space-y-3">
            <StatRow
              label="Total Sources"
              value={metrics.sources_total}
              icon={<Activity className="w-4 h-4" />}
            />
            <StatRow
              label="Pending Scrape"
              value={metrics.sources_by_status.pending}
              icon={<Clock className="w-4 h-4 text-smoke" />}
              color="text-smoke"
            />
            <StatRow
              label="In Progress"
              value={metrics.sources_by_status.in_progress}
              icon={<Zap className="w-4 h-4 text-terminal" />}
              color="text-terminal"
            />
            <StatRow
              label="Completed"
              value={metrics.sources_by_status.completed}
              icon={<CheckCircle className="w-4 h-4 text-terminal" />}
              color="text-terminal"
            />
            <StatRow
              label="Failed"
              value={metrics.sources_by_status.failed}
              icon={<XCircle className="w-4 h-4 text-threat-critical" />}
              color="text-threat-critical"
            />
            <StatRow
              label="Skipped"
              value={metrics.sources_by_status.skipped}
              icon={<TrendingDown className="w-4 h-4 text-fog" />}
              color="text-fog"
            />
          </div>
        </div>

        {/* Events Breakdown */}
        <div className="border-2 border-steel bg-concrete/30 p-6">
          <h3 className="font-mono font-bold text-terminal text-sm mb-4 flex items-center gap-2">
            <BarChart3 className="w-4 h-4" />
            EVENTS BREAKDOWN
          </h3>
          <div className="space-y-3">
            <StatRow
              label="Total Events"
              value={metrics.events_total}
              icon={<Activity className="w-4 h-4" />}
            />
            <StatRow
              label="Pending"
              value={metrics.events_by_status.pending}
              icon={<Clock className="w-4 h-4 text-smoke" />}
              color="text-smoke"
            />
            <StatRow
              label="Published"
              value={metrics.events_by_status.published}
              icon={<CheckCircle className="w-4 h-4 text-terminal" />}
              color="text-terminal"
            />
            <StatRow
              label="Rejected"
              value={metrics.events_by_status.rejected}
              icon={<XCircle className="w-4 h-4 text-threat-medium" />}
              color="text-threat-medium"
            />
          </div>

          {/* Conversion Metrics */}
          <div className="mt-6 pt-6 border-t-2 border-steel">
            <h4 className="font-mono font-bold text-smoke text-xs mb-3">CONVERSION RATES</h4>
            <div className="space-y-2">
              <div className="flex justify-between text-xs font-mono">
                <span className="text-fog">Scrape Completion:</span>
                <span className="text-terminal font-bold">{metrics.scrape_completion_rate.toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-xs font-mono">
                <span className="text-fog">Enrichment:</span>
                <span className="text-terminal font-bold">{metrics.enrichment_rate.toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-xs font-mono">
                <span className="text-fog">Publish:</span>
                <span className="text-terminal font-bold">{metrics.publish_rate.toFixed(1)}%</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Enrichments Section */}
      <div className="mt-8">
        <div className="border-l-4 border-electric pl-6 mb-6">
          <h3 className="font-display font-black text-2xl text-chalk tracking-tight">
            RECENT ENRICHMENTS (Last 10)
          </h3>
          <p className="text-sm text-smoke font-mono mt-2">
            Track source → event flow in real-time
          </p>
        </div>

        <div className="space-y-3">
          {recentEnrichments.slice(0, 10).map((item) => (
            <div
              key={item.source_id}
              className={`border-2 p-4 ${
                item.enrichment_status === 'completed' ? 'border-terminal bg-terminal/5' :
                item.enrichment_status === 'failed' ? 'border-threat-critical bg-threat-critical/5' :
                'border-steel bg-concrete/30'
              }`}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  {/* Source Info */}
                  <div className="flex items-center gap-2 mb-2">
                    {item.enrichment_status === 'completed' && (
                      <CheckCircle className="w-4 h-4 text-terminal flex-shrink-0" />
                    )}
                    {item.enrichment_status === 'failed' && (
                      <XCircle className="w-4 h-4 text-threat-critical flex-shrink-0" />
                    )}
                    {item.enrichment_status === 'enriching' && (
                      <Zap className="w-4 h-4 text-steel animate-pulse flex-shrink-0" />
                    )}
                    <h4 className="font-mono font-bold text-chalk text-sm truncate">
                      {item.source_title || 'Untitled Source'}
                    </h4>
                  </div>

                  {/* URL */}
                  <a
                    href={item.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs font-mono text-electric hover:text-terminal transition-colors flex items-center gap-1 mb-2 truncate"
                  >
                    <ExternalLink className="w-3 h-3 flex-shrink-0" />
                    <span className="truncate">{item.source_url}</span>
                  </a>

                  {/* Event Link */}
                  {item.event_id && item.enrichment_status === 'completed' && (
                    <div className="flex items-center gap-2 mt-2 pt-2 border-t border-steel">
                      <ArrowRight className="w-4 h-4 text-terminal flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <span className="text-xs font-mono text-terminal font-bold">EVENT: </span>
                        <span className="text-xs font-mono text-chalk truncate">{item.event_title}</span>
                        <span className={`ml-2 px-2 py-0.5 text-xs font-mono font-bold ${
                          item.event_status === 'published' ? 'bg-terminal text-void' :
                          item.event_status === 'rejected' ? 'bg-threat-medium text-void' :
                          'bg-steel text-void'
                        }`}>
                          {item.event_status?.toUpperCase()}
                        </span>
                      </div>
                    </div>
                  )}

                  {/* Error Message */}
                  {item.enrichment_error && (
                    <div className="mt-2 pt-2 border-t border-threat-critical">
                      <div className="flex items-start gap-2">
                        <AlertCircle className="w-3 h-3 text-threat-critical flex-shrink-0 mt-0.5" />
                        <p className="text-xs font-mono text-chalk break-words">
                          {item.enrichment_error}
                        </p>
                      </div>
                    </div>
                  )}
                </div>

                {/* Status Badge */}
                <div className="flex flex-col items-end gap-1">
                  <span className={`px-2 py-1 text-xs font-mono font-bold whitespace-nowrap ${
                    item.enrichment_status === 'completed' ? 'bg-terminal text-void' :
                    item.enrichment_status === 'failed' ? 'bg-threat-critical text-void' :
                    'bg-steel text-void'
                  }`}>
                    {item.enrichment_status.toUpperCase()}
                  </span>
                  {item.enriched_at && (
                    <span className="text-xs font-mono text-fog whitespace-nowrap">
                      {formatDateTime(item.enriched_at)}
                    </span>
                  )}
                </div>
              </div>
            </div>
          ))}

          {recentEnrichments.length === 0 && (
            <div className="border-2 border-steel bg-concrete/30 p-8 text-center">
              <Clock className="w-12 h-12 text-steel mx-auto mb-3" />
              <p className="font-mono text-sm text-fog">No recent enrichments</p>
            </div>
          )}
        </div>
      </div>

    </div>
  );
}

interface FunnelStageProps {
  title: string;
  icon: React.ReactNode;
  count: number;
  width: number;
  color: string;
  description: string;
  substats?: { label: string; value: number; color: string }[];
}

function FunnelStage({ title, icon, count, width, color, description, substats }: FunnelStageProps) {
  return (
    <div className="border-2 border-steel bg-concrete/30 p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`text-${color}`}>
            {icon}
          </div>
          <h3 className="font-mono font-bold text-chalk text-sm">{title}</h3>
        </div>
        <span className={`text-2xl font-display font-black text-${color}`}>
          {count.toLocaleString()}
        </span>
      </div>

      {/* Progress bar */}
      <div className="h-3 bg-void border border-steel mb-2">
        <div
          className={`h-full bg-${color} transition-all duration-500`}
          style={{ width: `${Math.min(width, 100)}%` }}
        />
      </div>

      <p className="text-xs font-mono text-fog">{description}</p>

      {/* Substats */}
      {substats && substats.length > 0 && (
        <div className="flex gap-4 mt-3 pt-3 border-t border-steel">
          {substats.map((stat, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <span className="text-xs font-mono text-fog">{stat.label}:</span>
              <span className={`text-xs font-mono font-bold ${stat.color}`}>
                {stat.value}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface StatRowProps {
  label: string;
  value: number;
  icon: React.ReactNode;
  color?: string;
}

function StatRow({ label, value, icon, color = 'text-chalk' }: StatRowProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        {icon}
        <span className="text-xs font-mono text-fog">{label}</span>
      </div>
      <span className={`text-sm font-mono font-bold ${color}`}>
        {value.toLocaleString()}
      </span>
    </div>
  );
}
