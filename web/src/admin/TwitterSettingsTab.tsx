import { useState, useEffect } from 'react';
import { Save, CheckCircle2, XCircle, Eye, EyeOff, Twitter } from 'lucide-react';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface TwitterConfig {
  id: number;
  api_key: string;
  api_secret: string;
  access_token: string;
  access_token_secret: string;
  bearer_token: string;
  min_magnitude_for_tweet: number;
  min_confidence_for_tweet: number;
  max_tweet_age_hours: number;
  enabled_categories: string[]; // Array of category strings
  enabled: boolean;
  updated_at: string;
  created_at: string;
}

export function TwitterSettingsTab() {
  const [config, setConfig] = useState<TwitterConfig | null>(null);
  const [showAPIKey, setShowAPIKey] = useState(false);
  const [showAPISecret, setShowAPISecret] = useState(false);
  const [showAccessToken, setShowAccessToken] = useState(false);
  const [showAccessTokenSecret, setShowAccessTokenSecret] = useState(false);
  const [showBearerToken, setShowBearerToken] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState<{ text: string; type: 'success' | 'error' } | null>(null);

  // Fetch current config on mount
  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/twitter-config`, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch Twitter configuration');
      const data = await response.json();

      // Parse enabled_categories if it's a JSON string
      if (typeof data.enabled_categories === 'string') {
        data.enabled_categories = JSON.parse(data.enabled_categories);
      }

      setConfig(data);
    } catch (err) {
      console.error('Error fetching Twitter config:', err);
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
      const response = await fetch(`${API_BASE_URL}/api/twitter-config`, {
        method: 'PUT',
        headers: getAuthHeaders(),
        body: JSON.stringify({
          api_key: config.api_key,
          api_secret: config.api_secret,
          access_token: config.access_token,
          access_token_secret: config.access_token_secret,
          bearer_token: config.bearer_token,
          min_magnitude_for_tweet: config.min_magnitude_for_tweet,
          min_confidence_for_tweet: config.min_confidence_for_tweet,
          max_tweet_age_hours: config.max_tweet_age_hours,
          enabled_categories: config.enabled_categories,
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

  const toggleCategory = (category: string) => {
    if (!config) return;

    const categories = [...config.enabled_categories];
    const index = categories.indexOf(category);

    if (index > -1) {
      categories.splice(index, 1);
    } else {
      categories.push(category);
    }

    setConfig({ ...config, enabled_categories: categories });
  };

  const allCategories = [
    'military',
    'geopolitics',
    'cyber',
    'terrorism',
    'disaster',
    'economic',
    'diplomacy',
    'intelligence',
    'humanitarian',
    'other'
  ];

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
        <div className="flex items-center gap-3">
          <Twitter className="w-8 h-8 text-terminal" />
          <div>
            <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
              TWITTER / X CONFIGURATION
            </h2>
            <p className="text-sm text-smoke font-mono mt-2">
              Configure Twitter/X API integration for auto-posting breaking events
            </p>
          </div>
        </div>
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

      {/* API Credentials */}
      <div className="border-2 border-steel bg-concrete">
        <div className="px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-base text-chalk">TWITTER API CREDENTIALS</h3>
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
                ENABLE TWITTER AUTO-POSTING
              </span>
            </label>
            <p className="text-xs font-mono text-fog mt-2">
              When enabled, events meeting thresholds will automatically generate and post tweets
            </p>
          </div>

          {/* API Key */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              API KEY (CONSUMER KEY)
            </label>
            <div className="relative">
              <input
                type={showAPIKey ? 'text' : 'password'}
                value={config.api_key}
                onChange={(e) => setConfig({ ...config, api_key: e.target.value })}
                placeholder="Enter Twitter API Key"
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowAPIKey(!showAPIKey)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showAPIKey ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
          </div>

          {/* API Secret */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              API SECRET (CONSUMER SECRET)
            </label>
            <div className="relative">
              <input
                type={showAPISecret ? 'text' : 'password'}
                value={config.api_secret}
                onChange={(e) => setConfig({ ...config, api_secret: e.target.value })}
                placeholder="Enter Twitter API Secret"
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowAPISecret(!showAPISecret)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showAPISecret ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
          </div>

          {/* Access Token */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              ACCESS TOKEN
            </label>
            <div className="relative">
              <input
                type={showAccessToken ? 'text' : 'password'}
                value={config.access_token}
                onChange={(e) => setConfig({ ...config, access_token: e.target.value })}
                placeholder="Enter Access Token"
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowAccessToken(!showAccessToken)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showAccessToken ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
          </div>

          {/* Access Token Secret */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              ACCESS TOKEN SECRET
            </label>
            <div className="relative">
              <input
                type={showAccessTokenSecret ? 'text' : 'password'}
                value={config.access_token_secret}
                onChange={(e) => setConfig({ ...config, access_token_secret: e.target.value })}
                placeholder="Enter Access Token Secret"
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowAccessTokenSecret(!showAccessTokenSecret)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showAccessTokenSecret ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
          </div>

          {/* Bearer Token (Optional) */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-2">
              BEARER TOKEN (OPTIONAL)
            </label>
            <div className="relative">
              <input
                type={showBearerToken ? 'text' : 'password'}
                value={config.bearer_token}
                onChange={(e) => setConfig({ ...config, bearer_token: e.target.value })}
                placeholder="Enter Bearer Token (optional)"
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono text-sm focus:border-terminal focus:outline-none transition-colors pr-12"
              />
              <button
                type="button"
                onClick={() => setShowBearerToken(!showBearerToken)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-fog hover:text-chalk transition-colors"
              >
                {showBearerToken ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
              </button>
            </div>
            <p className="text-xs font-mono text-fog mt-2">
              Get your Twitter API credentials from developer.twitter.com
            </p>
          </div>
        </div>
      </div>

      {/* Auto-Posting Thresholds */}
      <div className="border-2 border-steel bg-concrete">
        <div className="px-6 py-4 border-b-2 border-steel bg-void/50">
          <h3 className="font-display font-black text-base text-chalk">AUTO-POSTING THRESHOLDS</h3>
        </div>
        <div className="p-6 space-y-6">
          {/* Magnitude Threshold */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MINIMUM MAGNITUDE FOR TWEET</label>
                <p className="text-xs font-mono text-fog mt-1">Only events with magnitude above this threshold will be tweeted</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">{config.min_magnitude_for_tweet.toFixed(1)}</span>
            </div>
            <input
              type="range"
              min="0"
              max="10"
              step="0.5"
              value={config.min_magnitude_for_tweet}
              onChange={(e) => setConfig({ ...config, min_magnitude_for_tweet: parseFloat(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>0.0 (Low)</span>
              <span>10.0 (Critical)</span>
            </div>
          </div>

          {/* Confidence Threshold */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MINIMUM CONFIDENCE FOR TWEET</label>
                <p className="text-xs font-mono text-fog mt-1">Only events with confidence above this threshold will be tweeted</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">{config.min_confidence_for_tweet.toFixed(2)}</span>
            </div>
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              value={config.min_confidence_for_tweet}
              onChange={(e) => setConfig({ ...config, min_confidence_for_tweet: parseFloat(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>0.00 (Low)</span>
              <span>1.00 (High)</span>
            </div>
          </div>

          {/* Max Tweet Age */}
          <div className="space-y-4">
            <div className="flex justify-between items-end">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold">MAX EVENT AGE FOR TWEET (HOURS)</label>
                <p className="text-xs font-mono text-fog mt-1">Only tweet events that are at most this many hours old</p>
              </div>
              <span className="text-2xl font-mono font-bold text-terminal">{config.max_tweet_age_hours}</span>
            </div>
            <input
              type="range"
              min="1"
              max="48"
              step="1"
              value={config.max_tweet_age_hours}
              onChange={(e) => setConfig({ ...config, max_tweet_age_hours: parseInt(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs font-mono text-fog">
              <span>1 hour</span>
              <span>48 hours</span>
            </div>
          </div>

          {/* Enabled Categories */}
          <div>
            <label className="block text-sm font-mono text-chalk font-bold mb-3">
              ENABLED CATEGORIES FOR AUTO-TWEETING
            </label>
            <p className="text-xs font-mono text-fog mb-3">
              Select which event categories should trigger automatic tweets
            </p>
            <div className="grid grid-cols-2 gap-3">
              {allCategories.map((category) => (
                <label key={category} className="flex items-center gap-2 cursor-pointer px-3 py-2 border border-steel bg-void hover:border-iron transition-colors">
                  <input
                    type="checkbox"
                    checked={config.enabled_categories.includes(category)}
                    onChange={() => toggleCategory(category)}
                    className="w-4 h-4"
                  />
                  <span className="text-sm font-mono text-chalk uppercase">{category}</span>
                </label>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Context-Aware Posting Info */}
      <div className="border-2 border-terminal bg-terminal/5">
        <div className="px-6 py-4 border-b-2 border-terminal bg-terminal/10">
          <h3 className="font-display font-black text-base text-terminal">CONTEXT-AWARE POSTING</h3>
        </div>
        <div className="p-6 space-y-3">
          <p className="text-sm font-mono text-chalk">
            Tweet generation now uses AI-powered context awareness to prevent redundant posts.
          </p>
          <div className="space-y-2 text-xs font-mono text-fog">
            <p>✓ <strong className="text-chalk">SKIP</strong> - Redundant events similar to recent tweets (last 24h) are automatically skipped</p>
            <p>✓ <strong className="text-chalk">POST</strong> - Genuinely new events are posted with "BREAKING:" prefix</p>
            <p>✓ <strong className="text-chalk">UPDATE</strong> - New significant details about recent topics use "UPDATE:" prefix</p>
          </div>
          <div className="mt-4 pt-4 border-t border-terminal/30">
            <p className="text-xs font-mono text-fog">
              <strong className="text-terminal">Note:</strong> The tweet generation prompt is managed in code for optimal context-aware decision making.
              Visit the <strong>POSTED TWEETS</strong> tab to view posting history and AI decisions.
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
          <span className="text-terminal font-bold">NOTE:</span> Changes to Twitter configuration are active immediately.
          Tweets will only be posted for NEW events that meet the configured thresholds after enabling.
          Make sure to test your Twitter API credentials before enabling auto-posting.
        </p>
      </div>
    </div>
  );
}
