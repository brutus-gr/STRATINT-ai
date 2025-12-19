import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Header } from '../components/Header';
import { ArrowLeft, Terminal, Copy, Check } from 'lucide-react';

import { API_BASE_URL } from '../utils/api';

interface Endpoint {
  method: string;
  path: string;
  description: string;
  params?: { name: string; type: string; description: string; required?: boolean }[];
  response: string;
  example: string;
}

const endpoints: Endpoint[] = [
  {
    method: 'GET',
    path: '/api/events',
    description: 'Query intelligence events with filtering, pagination, and sorting',
    params: [
      { name: 'limit', type: 'int', description: 'Maximum number of events to return (default: 100)', required: false },
      { name: 'page', type: 'int', description: 'Page number for pagination (default: 1)', required: false },
      { name: 'status', type: 'string', description: 'Filter by status: published, rejected, pending', required: false },
      { name: 'categories', type: 'string[]', description: 'Filter by categories (comma-separated)', required: false },
      { name: 'since', type: 'timestamp', description: 'Filter events after this timestamp', required: false },
    ],
    response: '{ events: Event[], count: int, query: EventQuery }',
    example: 'curl http://localhost:8080/api/events?limit=20&status=published',
  },
  {
    method: 'GET',
    path: '/api/events/:id',
    description: 'Retrieve detailed information about a specific event by ID',
    params: [
      { name: 'id', type: 'string', description: 'Unique event identifier', required: true },
    ],
    response: 'Event',
    example: 'curl http://localhost:8080/api/events/evt-1234567890',
  },
  {
    method: 'GET',
    path: '/api/feed.rss',
    description: 'RSS 2.0 feed of the 20 most recent published intelligence events',
    response: 'application/rss+xml',
    example: 'curl http://localhost:8080/api/feed.rss',
  },
  {
    method: 'GET',
    path: '/api/stats',
    description: 'System statistics including event counts, uptime, and performance metrics',
    response: '{ uptime_seconds: int, total_events: int, total_sources: int, avg_confidence: float, enrichment_rate: float }',
    example: 'curl http://localhost:8080/api/stats',
  },
  {
    method: 'GET',
    path: '/api/sources',
    description: 'List all ingested sources with their scraping status and metadata',
    response: '{ sources: Source[], count: int }',
    example: 'curl http://localhost:8080/api/sources',
  },
  {
    method: 'GET',
    path: '/api/pipeline/metrics',
    description: 'Processing pipeline metrics including source and event counts by status, bottleneck analysis',
    response: 'PipelineMetrics',
    example: 'curl http://localhost:8080/api/pipeline/metrics',
  },
  {
    method: 'GET',
    path: '/api/thresholds',
    description: 'Current publication threshold configuration (confidence, magnitude, source age)',
    response: 'ThresholdConfig',
    example: 'curl http://localhost:8080/api/thresholds',
  },
  {
    method: 'POST',
    path: '/api/thresholds',
    description: 'Update publication thresholds for event filtering',
    params: [
      { name: 'min_confidence', type: 'float', description: 'Minimum confidence score (0.0-1.0)', required: false },
      { name: 'min_magnitude', type: 'float', description: 'Minimum event magnitude', required: false },
      { name: 'max_source_age_hours', type: 'int', description: 'Maximum source age in hours (0 = no limit)', required: false },
    ],
    response: 'ThresholdConfig',
    example: 'curl -X POST http://localhost:8080/api/thresholds -H "Content-Type: application/json" -d \'{"min_confidence": 0.6, "min_magnitude": 3.0}\'',
  },
];

export function ApiDocsPage() {
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null);

  const copyToClipboard = (text: string, index: number) => {
    navigator.clipboard.writeText(text);
    setCopiedIndex(index);
    setTimeout(() => setCopiedIndex(null), 2000);
  };

  return (
    <div className="min-h-screen bg-void text-chalk">
      <div className="scan-line" />
      <Header  />

      <div className="pt-16">
        <div className="max-w-6xl mx-auto p-4 md:p-8">
          {/* Back Button */}
          <Link
            to="/"
            className="inline-flex items-center gap-2 text-sm font-mono text-smoke hover:text-terminal transition-colors mb-6"
          >
            <ArrowLeft className="w-4 h-4" />
            BACK TO FEED
          </Link>

          {/* Header */}
          <div className="border-l-4 border-terminal pl-6 mb-8">
            <div className="flex items-center gap-3 mb-2">
              <Terminal className="w-8 h-8 text-terminal" />
              <h1 className="font-display font-black text-4xl text-chalk tracking-tight">
                API DOCUMENTATION
              </h1>
            </div>
            <p className="text-sm text-smoke font-mono mt-2">
              REST API REFERENCE / STRATINT v1.0
            </p>
          </div>

          {/* Introduction */}
          <div className="border-2 border-steel bg-concrete p-6 mb-6">
            <h2 className="font-mono font-bold text-chalk text-lg mb-4">OVERVIEW</h2>
            <div className="space-y-3 text-sm font-mono text-fog">
              <p>
                The STRATINT REST API provides programmatic access to intelligence events, sources, and system metrics.
                All endpoints return JSON responses unless otherwise specified.
              </p>
              <div className="border-l-2 border-terminal pl-4 bg-void/50 p-3">
                <p className="text-terminal font-bold mb-2">BASE URL</p>
                <code className="text-chalk">{API_BASE_URL}</code>
              </div>
              <div className="border-l-2 border-electric pl-4 bg-void/50 p-3">
                <p className="text-electric font-bold mb-2">RSS FEED</p>
                <code className="text-chalk">{API_BASE_URL}/api/feed.rss</code>
              </div>
            </div>
          </div>

          {/* Endpoints */}
          <div className="space-y-6">
            {endpoints.map((endpoint, index) => (
              <div key={index} className="border-2 border-steel bg-concrete">
                {/* Endpoint Header */}
                <div className="border-b-2 border-steel bg-iron p-4 flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <span
                      className={`px-3 py-1 font-mono text-xs font-bold border-2 ${
                        endpoint.method === 'GET'
                          ? 'border-terminal text-terminal bg-terminal/10'
                          : endpoint.method === 'POST'
                          ? 'border-electric text-electric bg-electric/10'
                          : 'border-warning text-warning bg-warning/10'
                      }`}
                    >
                      {endpoint.method}
                    </span>
                    <code className="font-mono text-chalk font-bold">{endpoint.path}</code>
                  </div>
                  <button
                    onClick={() => copyToClipboard(endpoint.example, index)}
                    className="text-smoke hover:text-terminal transition-colors"
                    title="Copy example"
                  >
                    {copiedIndex === index ? (
                      <Check className="w-4 h-4 text-terminal" />
                    ) : (
                      <Copy className="w-4 h-4" />
                    )}
                  </button>
                </div>

                {/* Endpoint Body */}
                <div className="p-4 space-y-4">
                  <p className="text-sm font-mono text-fog">{endpoint.description}</p>

                  {/* Parameters */}
                  {endpoint.params && endpoint.params.length > 0 && (
                    <div>
                      <h3 className="font-mono font-bold text-chalk text-xs mb-2 uppercase">Parameters</h3>
                      <div className="border border-steel bg-void/50 divide-y divide-steel">
                        {endpoint.params.map((param, pIndex) => (
                          <div key={pIndex} className="p-3 grid grid-cols-4 gap-4 text-xs font-mono">
                            <div>
                              <code className="text-terminal">{param.name}</code>
                              {param.required && <span className="text-warning ml-1">*</span>}
                            </div>
                            <div className="text-electric">{param.type}</div>
                            <div className="col-span-2 text-fog">{param.description}</div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Response */}
                  <div>
                    <h3 className="font-mono font-bold text-chalk text-xs mb-2 uppercase">Response</h3>
                    <div className="border border-steel bg-void/50 p-3">
                      <code className="text-sm text-electric font-mono">{endpoint.response}</code>
                    </div>
                  </div>

                  {/* Example */}
                  <div>
                    <h3 className="font-mono font-bold text-chalk text-xs mb-2 uppercase">Example</h3>
                    <div className="border border-steel bg-void p-3 overflow-x-auto">
                      <code className="text-xs text-chalk font-mono whitespace-pre">{endpoint.example}</code>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Data Models */}
          <div className="border-2 border-steel bg-concrete mt-8 p-6">
            <h2 className="font-mono font-bold text-chalk text-lg mb-4">DATA MODELS</h2>
            <div className="space-y-4 text-sm font-mono">
              <div className="border border-steel bg-void/50 p-4">
                <h3 className="text-terminal font-bold mb-2">Event</h3>
                <pre className="text-xs text-fog overflow-x-auto">
{`{
  id: string,
  title: string,
  summary: string,
  category: "geopolitics" | "terrorism" | "cyber" | "other",
  magnitude: float,
  confidence: {
    score: float,
    reasoning: string,
    source_count: int
  },
  sources: Source[],
  entities: Entity[],
  location: Location,
  timestamp: datetime,
  status: "published" | "rejected" | "pending"
}`}
                </pre>
              </div>

              <div className="border border-steel bg-void/50 p-4">
                <h3 className="text-terminal font-bold mb-2">Source</h3>
                <pre className="text-xs text-fog overflow-x-auto">
{`{
  id: string,
  url: string,
  title: string,
  content: string,
  published_at: datetime,
  scrape_status: "pending" | "completed" | "failed" | "skipped",
  credibility: {
    score: float,
    factors: string[]
  }
}`}
                </pre>
              </div>
            </div>
          </div>

          {/* Rate Limiting */}
          <div className="border-2 border-warning bg-concrete mt-6 p-6">
            <h2 className="font-mono font-bold text-warning text-lg mb-4">⚠ RATE LIMITING</h2>
            <p className="text-sm font-mono text-fog">
              Currently, there are no rate limits enforced. However, please be respectful of server resources.
              Production deployments should implement appropriate rate limiting based on their requirements.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
