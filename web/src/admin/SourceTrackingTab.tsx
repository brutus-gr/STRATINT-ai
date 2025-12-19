import { useState, useEffect } from 'react';
import {
  CheckCircle,
  XCircle,
  Clock,
  Zap,
  ExternalLink,
  AlertCircle,
  Activity,
  ArrowRight
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

export function SourceTrackingTab() {
  const [enrichments, setEnrichments] = useState<EnrichmentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchEnrichments = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/admin/recent-enrichments`, {
          headers: getAuthHeaders(),
        });
        if (!response.ok) throw new Error('Failed to fetch enrichments');
        const data = await response.json();
        setEnrichments(data.enrichments || []);
        setError(null);
      } catch (err) {
        console.error('Error fetching enrichments:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchEnrichments();
    // Refresh every 10 seconds
    const interval = setInterval(fetchEnrichments, 10000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center gap-3 text-terminal">
          <Activity className="w-6 h-6 animate-pulse" />
          <span className="font-mono text-sm">LOADING SOURCE TRACKING...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="border-2 border-threat-critical bg-threat-critical/10 p-6">
        <div className="flex items-center gap-3">
          <AlertCircle className="w-6 h-6 text-threat-critical" />
          <div>
            <h3 className="font-mono font-bold text-threat-critical">ERROR LOADING DATA</h3>
            <p className="font-mono text-sm text-chalk mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  const completedCount = enrichments.filter(e => e.enrichment_status === 'completed').length;
  const failedCount = enrichments.filter(e => e.enrichment_status === 'failed').length;
  const enrichingCount = enrichments.filter(e => e.enrichment_status === 'enriching').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          SOURCE ENRICHMENT TRACKING
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Track which sources became events and which failed enrichment
        </p>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="border-2 border-steel bg-concrete/30 p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-mono text-xs text-fog mb-1">TOTAL PROCESSED</p>
              <p className="font-display text-2xl font-black text-chalk">{enrichments.length}</p>
            </div>
            <Activity className="w-8 h-8 text-steel" />
          </div>
        </div>

        <div className="border-2 border-terminal bg-terminal/10 p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-mono text-xs text-fog mb-1">COMPLETED</p>
              <p className="font-display text-2xl font-black text-terminal">{completedCount}</p>
            </div>
            <CheckCircle className="w-8 h-8 text-terminal" />
          </div>
        </div>

        <div className="border-2 border-threat-critical bg-threat-critical/10 p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-mono text-xs text-fog mb-1">FAILED</p>
              <p className="font-display text-2xl font-black text-threat-critical">{failedCount}</p>
            </div>
            <XCircle className="w-8 h-8 text-threat-critical" />
          </div>
        </div>

        <div className="border-2 border-steel bg-concrete/30 p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-mono text-xs text-fog mb-1">IN PROGRESS</p>
              <p className="font-display text-2xl font-black text-chalk">{enrichingCount}</p>
            </div>
            <Zap className="w-8 h-8 text-steel animate-pulse" />
          </div>
        </div>
      </div>

      {/* Enrichments List */}
      <div className="space-y-3">
        <h3 className="font-mono font-bold text-terminal text-sm">RECENT ENRICHMENTS ({enrichments.length})</h3>

        {enrichments.length === 0 ? (
          <div className="border-2 border-steel bg-concrete/30 p-8 text-center">
            <Clock className="w-12 h-12 text-steel mx-auto mb-3" />
            <p className="font-mono text-sm text-fog">No enrichments processed yet</p>
          </div>
        ) : (
          enrichments.map((item) => (
            <EnrichmentCard key={item.source_id} enrichment={item} />
          ))
        )}
      </div>
    </div>
  );
}

function EnrichmentCard({ enrichment }: { enrichment: EnrichmentInfo }) {
  const getStatusIcon = () => {
    switch (enrichment.enrichment_status) {
      case 'completed':
        return <CheckCircle className="w-5 h-5 text-terminal" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-threat-critical" />;
      case 'enriching':
        return <Zap className="w-5 h-5 text-steel animate-pulse" />;
      default:
        return <Clock className="w-5 h-5 text-smoke" />;
    }
  };

  const getStatusColor = () => {
    switch (enrichment.enrichment_status) {
      case 'completed':
        return 'border-terminal bg-terminal/5';
      case 'failed':
        return 'border-threat-critical bg-threat-critical/5';
      case 'enriching':
        return 'border-steel bg-concrete/30';
      default:
        return 'border-steel bg-concrete/30';
    }
  };

  const getEventStatusBadge = (status?: string) => {
    if (!status) return null;

    const colors = {
      published: 'bg-terminal text-void',
      rejected: 'bg-threat-medium text-void',
      enriched: 'bg-steel text-void',
      pending: 'bg-smoke text-void',
    };

    const color = colors[status as keyof typeof colors] || 'bg-steel text-void';

    return (
      <span className={`px-2 py-0.5 text-xs font-mono font-bold ${color}`}>
        {status.toUpperCase()}
      </span>
    );
  };

  return (
    <div className={`border-2 ${getStatusColor()} p-4`}>
      <div className="space-y-3">
        {/* Header Row */}
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 flex-1 min-w-0">
            {getStatusIcon()}
            <div className="flex-1 min-w-0">
              <h4 className="font-mono font-bold text-chalk text-sm mb-1 truncate">
                {enrichment.source_title || 'Untitled Source'}
              </h4>
              <div className="flex items-center gap-2 text-xs font-mono text-fog flex-wrap">
                <span className="px-2 py-0.5 bg-void text-smoke border border-steel">
                  {enrichment.source_type.toUpperCase()}
                </span>
                <span>{formatDateTime(enrichment.published_at)}</span>
              </div>
            </div>
          </div>
          <div className="flex flex-col items-end gap-2">
            <span className={`px-2 py-1 text-xs font-mono font-bold ${
              enrichment.enrichment_status === 'completed' ? 'bg-terminal text-void' :
              enrichment.enrichment_status === 'failed' ? 'bg-threat-critical text-void' :
              'bg-steel text-void'
            }`}>
              {enrichment.enrichment_status.toUpperCase()}
            </span>
            {enrichment.enriched_at && (
              <span className="text-xs font-mono text-fog">
                {formatDateTime(enrichment.enriched_at)}
              </span>
            )}
          </div>
        </div>

        {/* Source URL */}
        <div className="flex items-center gap-2">
          <a
            href={enrichment.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs font-mono text-electric hover:text-terminal transition-colors flex items-center gap-1 truncate"
          >
            <ExternalLink className="w-3 h-3 flex-shrink-0" />
            <span className="truncate">{enrichment.source_url}</span>
          </a>
        </div>

        {/* Event Mapping (if successful) */}
        {enrichment.event_id && enrichment.enrichment_status === 'completed' && (
          <div className="border-t border-steel pt-3 mt-3">
            <div className="flex items-center gap-3">
              <ArrowRight className="w-4 h-4 text-terminal flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-mono text-xs text-terminal font-bold">EVENT CREATED:</span>
                  {getEventStatusBadge(enrichment.event_status)}
                </div>
                <p className="text-sm font-mono text-chalk truncate">
                  {enrichment.event_title}
                </p>
                <p className="text-xs font-mono text-fog mt-1">
                  ID: {enrichment.event_id}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Error Message (if failed) */}
        {enrichment.enrichment_error && (
          <div className="border-t border-threat-critical pt-3 mt-3">
            <div className="flex items-start gap-2">
              <AlertCircle className="w-4 h-4 text-threat-critical flex-shrink-0 mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="font-mono text-xs text-threat-critical font-bold mb-1">ERROR:</p>
                <p className="text-xs font-mono text-chalk break-words">
                  {enrichment.enrichment_error}
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
