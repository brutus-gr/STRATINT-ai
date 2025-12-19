import { Terminal, Copy, Check, ChevronDown, ChevronUp } from 'lucide-react';
import { useState } from 'react';

type AccessMethod = 'mcp' | 'rest' | 'rss';

export function MCPInstructions() {
  const [copied, setCopied] = useState(false);
  const [activeMethod, setActiveMethod] = useState<AccessMethod>('mcp');
  const [isExpanded, setIsExpanded] = useState(false);
  const mcpUrl = 'https://mcp.stratint.ai/mcp';
  const restUrl = 'https://stratint.ai';

  const copyToClipboard = async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="border-2 border-terminal bg-terminal/5 overflow-hidden">
      {/* Header */}
      <div
        className={`px-6 py-4 ${isExpanded ? 'border-b-2' : ''} border-terminal bg-terminal/10 cursor-pointer hover:bg-terminal/15 transition-colors`}
        onClick={() => setIsExpanded(!isExpanded)}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Terminal className="w-6 h-6 text-terminal" />
            <div>
              <h2 className="font-display font-black text-lg text-terminal tracking-tight">
                API AND MCP SERVER ACCESS
              </h2>
              <p className="text-xs font-mono text-chalk mt-0.5">
                Connect to real-time OSINT intelligence - No auth required
              </p>
            </div>
          </div>

          {/* Expand/Collapse Icon */}
          <div className="flex items-center gap-2">
            {isExpanded ? (
              <ChevronUp className="w-6 h-6 text-terminal" />
            ) : (
              <ChevronDown className="w-6 h-6 text-terminal" />
            )}
          </div>
        </div>
      </div>

      {/* Method Selector - Only show when expanded */}
      {isExpanded && (
        <div className="px-6 py-4 border-b-2 border-terminal bg-terminal/5">
          <div className="flex gap-2">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setActiveMethod('mcp');
              }}
              className={`px-4 py-2 border-2 font-mono text-xs font-bold transition-all ${
                activeMethod === 'mcp'
                  ? 'border-terminal bg-terminal/20 text-terminal'
                  : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
              }`}
            >
              MCP SERVER
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setActiveMethod('rest');
              }}
              className={`px-4 py-2 border-2 font-mono text-xs font-bold transition-all ${
                activeMethod === 'rest'
                  ? 'border-terminal bg-terminal/20 text-terminal'
                  : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
              }`}
            >
              REST API
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setActiveMethod('rss');
              }}
              className={`px-4 py-2 border-2 font-mono text-xs font-bold transition-all ${
                activeMethod === 'rss'
                  ? 'border-terminal bg-terminal/20 text-terminal'
                  : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
              }`}
            >
              RSS FEED
            </button>
          </div>
        </div>
      )}

      {/* Content - Only show when expanded */}
      {isExpanded && (
        <div className="p-6 space-y-6">
        {activeMethod === 'mcp' ? (
          <>
            {/* MCP URL */}
            <div>
              <label className="block text-xs font-mono text-smoke mb-2">MCP SERVER ENDPOINT</label>
              <div className="flex items-center gap-2">
                <div className="flex-1 px-4 py-3 bg-void border-2 border-steel font-mono text-terminal">
                  {mcpUrl}
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    copyToClipboard(mcpUrl);
                  }}
                  className="px-4 py-3 border-2 border-terminal bg-terminal/10 hover:bg-terminal hover:text-void transition-all"
                >
                  {copied ? (
                    <Check className="w-5 h-5" />
                  ) : (
                    <Copy className="w-5 h-5" />
                  )}
                </button>
              </div>
            </div>

            {/* Claude Code Quick Add */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">ADD TO CLAUDE CODE</h3>
              <div className="p-4 bg-void border-2 border-terminal font-mono text-xs">
                <div className="text-smoke mb-2">// Run this command in your terminal:</div>
                <div className="text-terminal font-bold">
                  claude mcp add --transport http stratint https://mcp.stratint.ai/mcp
                </div>
              </div>
            </div>

            {/* MCP Quick Start */}
            <div className="space-y-3">
              <h3 className="text-sm font-mono font-bold text-chalk">QUICK START (MCP)</h3>
              <div className="space-y-2 text-xs font-mono text-fog">
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">1.</span>
                  <span>Add the MCP server to your AI assistant configuration (Claude Desktop, Cline, etc.)</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">2.</span>
                  <span>No authentication required - immediate access to global intelligence feed</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">3.</span>
                  <span>Query events using natural language or the <code className="px-1 py-0.5 bg-steel">get_events</code> function</span>
                </div>
              </div>
            </div>

            {/* MCP Example */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">EXAMPLE QUERY</h3>
              <div className="p-4 bg-void border border-steel font-mono text-xs">
                <div className="text-smoke">// Natural language query</div>
                <div className="text-chalk mt-1">
                  "Show me high-magnitude geopolitics events from the last 24 hours"
                </div>
              </div>
            </div>
          </>
        ) : activeMethod === 'rest' ? (
          <>
            {/* REST URL */}
            <div>
              <label className="block text-xs font-mono text-smoke mb-2">REST API BASE URL</label>
              <div className="flex items-center gap-2">
                <div className="flex-1 px-4 py-3 bg-void border-2 border-steel font-mono text-terminal">
                  {restUrl}
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    copyToClipboard(restUrl);
                  }}
                  className="px-4 py-3 border-2 border-terminal bg-terminal/10 hover:bg-terminal hover:text-void transition-all"
                >
                  {copied ? (
                    <Check className="w-5 h-5" />
                  ) : (
                    <Copy className="w-5 h-5" />
                  )}
                </button>
              </div>
            </div>

            {/* REST Quick Start */}
            <div className="space-y-3">
              <h3 className="text-sm font-mono font-bold text-chalk">QUICK START (REST)</h3>
              <div className="space-y-2 text-xs font-mono text-fog">
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">1.</span>
                  <span>Use standard HTTP GET requests to query intelligence data</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">2.</span>
                  <span>No authentication required - CORS enabled for browser requests</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">3.</span>
                  <span>Filter using URL query parameters (13+ options available)</span>
                </div>
              </div>
            </div>

            {/* REST Example */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">EXAMPLE REQUEST</h3>
              <div className="p-4 bg-void border border-steel font-mono text-xs space-y-3">
                <div>
                  <div className="text-smoke">// Get high-magnitude geopolitics events from last 24h</div>
                  <div className="text-terminal mt-1">
                    GET {restUrl}/api/events?time_range=24h&categories=geopolitics&min_magnitude=7.0
                  </div>
                </div>
                <div>
                  <div className="text-smoke">// Using curl</div>
                  <div className="text-chalk mt-1 break-all">
                    curl "{restUrl}/api/events?time_range=24h"
                  </div>
                </div>
              </div>
            </div>

            {/* REST Endpoints */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">AVAILABLE ENDPOINTS</h3>
              <div className="space-y-1 text-xs font-mono">
                <div className="p-2 bg-void border border-steel flex items-center gap-2">
                  <span className="text-terminal font-bold">GET</span>
                  <span className="text-chalk">/api/events</span>
                  <span className="text-fog ml-auto">Query events</span>
                </div>
                <div className="p-2 bg-void border border-steel flex items-center gap-2">
                  <span className="text-terminal font-bold">GET</span>
                  <span className="text-chalk">/api/events/:id</span>
                  <span className="text-fog ml-auto">Get single event</span>
                </div>
                <div className="p-2 bg-void border border-steel flex items-center gap-2">
                  <span className="text-terminal font-bold">GET</span>
                  <span className="text-chalk">/api/stats</span>
                  <span className="text-fog ml-auto">Get statistics</span>
                </div>
              </div>
            </div>
          </>
        ) : activeMethod === 'rss' ? (
          <>
            {/* RSS Feed URL */}
            <div>
              <label className="block text-xs font-mono text-smoke mb-2">RSS FEED URL</label>
              <div className="flex items-center gap-2">
                <div className="flex-1 px-4 py-3 bg-void border-2 border-steel font-mono text-terminal">
                  {restUrl}/api/feed.rss
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    copyToClipboard(`${restUrl}/api/feed.rss`);
                  }}
                  className="px-4 py-3 border-2 border-terminal bg-terminal/10 hover:bg-terminal hover:text-void transition-all"
                >
                  {copied ? (
                    <Check className="w-5 h-5" />
                  ) : (
                    <Copy className="w-5 h-5" />
                  )}
                </button>
              </div>
            </div>

            {/* RSS Quick Start */}
            <div className="space-y-3">
              <h3 className="text-sm font-mono font-bold text-chalk">QUICK START (RSS)</h3>
              <div className="space-y-2 text-xs font-mono text-fog">
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">1.</span>
                  <span>Add the feed URL to your RSS reader (Feedly, NetNewsWire, etc.)</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">2.</span>
                  <span>Automatically receive the 20 most recent intelligence events</span>
                </div>
                <div className="flex items-start gap-3">
                  <span className="text-terminal font-bold">3.</span>
                  <span>Feed updates every time new events are published to the system</span>
                </div>
              </div>
            </div>

            {/* RSS Example */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">FEED FORMAT</h3>
              <div className="p-4 bg-void border border-steel font-mono text-xs space-y-3">
                <div>
                  <div className="text-smoke">// RSS 2.0 format with standard fields</div>
                  <div className="text-chalk mt-2 space-y-1">
                    <div>• Title: Event headline</div>
                    <div>• Description: Event summary</div>
                    <div>• Link: Direct link to event details</div>
                    <div>• PubDate: Event timestamp</div>
                    <div>• Category: Event category (geopolitics, cyber, etc.)</div>
                    <div>• GUID: Unique event identifier</div>
                  </div>
                </div>
              </div>
            </div>

            {/* RSS Readers */}
            <div className="space-y-2">
              <h3 className="text-sm font-mono font-bold text-chalk">COMPATIBLE RSS READERS</h3>
              <div className="grid grid-cols-2 gap-2 text-xs font-mono">
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> Feedly
                </div>
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> NetNewsWire
                </div>
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> Inoreader
                </div>
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> The Old Reader
                </div>
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> Newsboat (CLI)
                </div>
                <div className="p-2 bg-void border border-steel">
                  <span className="text-terminal">✓</span> Any RSS 2.0 reader
                </div>
              </div>
            </div>
          </>
        ) : null}

        {/* Features */}
        <div className="pt-4 border-t border-steel">
          <div className="grid grid-cols-2 gap-3 text-xs font-mono">
            <div className="flex items-center gap-2">
              <span className="text-terminal">✓</span>
              <span className="text-fog">No auth required</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-terminal">✓</span>
              <span className="text-fog">Real-time updates</span>
            </div>
            {activeMethod !== 'rss' && (
              <div className="flex items-center gap-2">
                <span className="text-terminal">✓</span>
                <span className="text-fog">13+ filter parameters</span>
              </div>
            )}
            {activeMethod === 'rss' && (
              <div className="flex items-center gap-2">
                <span className="text-terminal">✓</span>
                <span className="text-fog">20 most recent events</span>
              </div>
            )}
            <div className="flex items-center gap-2">
              <span className="text-terminal">✓</span>
              <span className="text-fog">AI-enriched data</span>
            </div>
          </div>
        </div>

        {/* API Docs Link */}
        <div className="pt-4">
          <a
            href="/api-docs"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center gap-2 px-4 py-2 border-2 border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono text-sm font-bold"
          >
            VIEW API DOCUMENTATION →
          </a>
        </div>
        </div>
      )}
    </div>
  );
}
