import { useState, useEffect } from 'react';
import { Save, CheckCircle2, XCircle, Eye, EyeOff } from 'lucide-react';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface OpenAIConfig {
  id: number;
  api_key: string;
  model: string;
  temperature: number;
  max_tokens: number;
  timeout_seconds: number;
  system_prompt: string;
  analysis_template: string;
  entity_extraction_prompt: string;
  enabled: boolean;
  updated_at: string;
  created_at: string;
}

export function OpenAIConfigTab() {
  const [config, setConfig] = useState<OpenAIConfig | null>(null);
  const [showApiKey, setShowApiKey] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  // Fetch current config on mount
  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/openai-config`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch OpenAI configuration');
      const data = await response.json();
      setConfig(data);
    } catch (err) {
      console.error('Error fetching OpenAI config:', err);
      setMessage({
        text: 'Failed to load configuration',
        type: 'error'
      });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!config) return;

    setSaving(true);
    setMessage(null);

    try {
      const response = await fetch(`${API_BASE_URL}/api/openai-config`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          api_key: config.api_key,
          model: config.model,
          temperature: config.temperature,
          max_tokens: config.max_tokens,
          timeout_seconds: config.timeout_seconds,
          system_prompt: config.system_prompt,
          analysis_template: config.analysis_template,
          entity_extraction_prompt: config.entity_extraction_prompt,
          enabled: config.enabled,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      const result = await response.json();
      setMessage({ text: result.message, type: 'success' });
      // Refresh config to get updated timestamps
      await fetchConfig();
    } catch (err) {
      setMessage({
        text: err instanceof Error ? err.message : 'Failed to save configuration',
        type: 'error'
      });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-fog font-mono">Loading configuration...</div>
      </div>
    );
  }

  if (!config) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-error font-mono">Failed to load configuration</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          OPENAI CONFIGURATION
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          Configure OpenAI API integration for OSINT enrichment
        </p>
      </div>

      {/* Message */}
      {message && (
        <div className={`border-2 ${message.type === 'success' ? 'border-terminal bg-terminal/10' : 'border-error bg-error/10'} p-4`}>
          <div className="flex items-center gap-3">
            {message.type === 'success' ? (
              <CheckCircle2 className="w-5 h-5 text-terminal" />
            ) : (
              <XCircle className="w-5 h-5 text-error" />
            )}
            <span className={`font-mono text-sm ${message.type === 'success' ? 'text-terminal' : 'text-error'}`}>
              {message.text}
            </span>
          </div>
        </div>
      )}

      {/* Configuration Form */}
      <div className="border-2 border-steel bg-concrete">
        <div className="px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-base text-chalk">API SETTINGS</h3>
        </div>
        <div className="p-6 space-y-6">
          {/* Enabled Toggle */}
          <div>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={config.enabled}
                onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
                className="w-5 h-5"
              />
              <span className="text-sm font-mono text-chalk font-bold">
                ENABLE OPENAI ENRICHMENT
              </span>
            </label>
            <p className="text-xs font-mono text-fog mt-2">
              When enabled, sources will be analyzed using OpenAI API
            </p>
          </div>

          {/* API Key */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              API KEY
            </label>
            <div className="relative">
              <input
                type={showApiKey ? 'text' : 'password'}
                value={config.api_key}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                placeholder="sk-..."
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowApiKey(!showApiKey)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showApiKey ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
            <p className="text-xs font-mono text-fog mt-2">
              OpenAI API key from platform.openai.com
            </p>
          </div>

          {/* Model */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              MODEL
            </label>
            <input
              type="text"
              value={config.model}
              onChange={(e) => setConfig({ ...config, model: e.target.value })}
              placeholder="gpt-4o-mini"
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Available: gpt-4o-mini, gpt-4o, gpt-5, o1-preview, o1-mini. Note: o1 models use extended reasoning (60-180s per request)
            </p>
          </div>

          {/* Temperature */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              TEMPERATURE: {config.temperature.toFixed(2)}
            </label>
            <input
              type="range"
              min="0"
              max="2"
              step="0.05"
              value={config.temperature}
              onChange={(e) => setConfig({ ...config, temperature: parseFloat(e.target.value) })}
              className="w-full"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Lower = more factual (0.0-0.5), Higher = more creative (1.0-2.0)
            </p>
          </div>

          {/* Max Tokens */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              MAX TOKENS
            </label>
            <input
              type="number"
              value={config.max_tokens}
              onChange={(e) => setConfig({ ...config, max_tokens: parseInt(e.target.value) || 0 })}
              min="100"
              max="16000"
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Maximum tokens for completion (typically 1000-4000 for analysis)
            </p>
          </div>

          {/* Timeout */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              TIMEOUT (SECONDS)
            </label>
            <input
              type="number"
              value={config.timeout_seconds}
              onChange={(e) => setConfig({ ...config, timeout_seconds: parseInt(e.target.value) || 0 })}
              min="5"
              max="300"
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Request timeout in seconds (recommended: 30-60)
            </p>
          </div>
        </div>
      </div>

      {/* Prompts Configuration */}
      <div className="border-2 border-steel bg-concrete">
        <div className="px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-base text-chalk">PROMPTS</h3>
        </div>
        <div className="p-6 space-y-6">
          {/* System Prompt */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              SYSTEM PROMPT
            </label>
            <textarea
              value={config.system_prompt}
              onChange={(e) => setConfig({ ...config, system_prompt: e.target.value })}
              rows={10}
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-xs focus:border-terminal focus:outline-none transition-colors resize-y"
            />
            <p className="text-xs font-mono text-fog mt-2">
              System prompt that defines the AI's role and output format
            </p>
          </div>

          {/* Analysis Template */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              ANALYSIS TEMPLATE
            </label>
            <textarea
              value={config.analysis_template}
              onChange={(e) => setConfig({ ...config, analysis_template: e.target.value })}
              rows={8}
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-xs focus:border-terminal focus:outline-none transition-colors resize-y"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Template for source analysis (supports variables like &#123;&#123;.SourceType&#125;&#125;, &#123;&#123;.RawContent&#125;&#125;)
            </p>
          </div>

          {/* Entity Extraction Prompt */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              ENTITY EXTRACTION PROMPT
            </label>
            <textarea
              value={config.entity_extraction_prompt}
              onChange={(e) => setConfig({ ...config, entity_extraction_prompt: e.target.value })}
              rows={6}
              className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-xs focus:border-terminal focus:outline-none transition-colors resize-y"
            />
            <p className="text-xs font-mono text-fog mt-2">
              Prompt for extracting named entities from text
            </p>
          </div>
        </div>
      </div>

      {/* Save Button */}
      <div className="flex justify-end">
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-3 px-8 py-4 bg-terminal hover:bg-terminal/90 disabled:bg-steel disabled:cursor-not-allowed text-void font-mono font-bold text-sm tracking-wide transition-colors"
        >
          <Save className="w-5 h-5" />
          {saving ? 'SAVING...' : 'SAVE CONFIGURATION'}
        </button>
      </div>

      {/* Info Footer */}
      <div className="border-2 border-steel bg-void/30 p-4">
        <p className="text-xs font-mono text-fog">
          <span className="text-terminal font-bold">NOTE:</span> Changes to OpenAI configuration will apply to new sources.
          The server must be restarted for enrichment to use the updated API key if it was previously disabled.
        </p>
      </div>
    </div>
  );
}
