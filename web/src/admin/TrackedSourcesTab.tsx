import { useState, useEffect } from 'react';
import { Plus, Trash2, Power, PowerOff, RefreshCw, Download, Eye } from 'lucide-react';
import { formatDateTime, formatDate } from '../utils/dateFormat';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface TrackedAccount {
  id: string;
  platform: string;
  account_identifier: string;
  display_name: string;
  enabled: boolean;
  last_fetched_id?: string;
  last_fetched_at?: string;
  fetch_interval_minutes: number;
  created_at: string;
}

export function TrackedSourcesTab() {
  const [accounts, setAccounts] = useState<TrackedAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddForm, setShowAddForm] = useState(false);
  const [viewingSourcesFor, setViewingSourcesFor] = useState<string | null>(null);
  const [sources, setSources] = useState<any[]>([]);
  const [loadingSources, setLoadingSources] = useState(false);
  const [newAccount, setNewAccount] = useState({
    platform: 'twitter',
    account_identifier: '',
    display_name: '',
    fetch_interval_minutes: 5,
  });

  useEffect(() => {
    fetchAccounts();
  }, []);

  const fetchAccounts = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/tracked-accounts`, {
        headers: getAuthHeaders(),
      });
      const data = await response.json();
      setAccounts(data.accounts || []);
      setLoading(false);
    } catch (error) {
      console.error('Failed to fetch tracked accounts:', error);
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch(`${API_BASE_URL}/api/tracked-accounts`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(newAccount),
      });

      if (response.ok) {
        fetchAccounts();
        setShowAddForm(false);
        setNewAccount({
          platform: 'twitter',
          account_identifier: '',
          display_name: '',
          fetch_interval_minutes: 5,
        });
      }
    } catch (error) {
      console.error('Failed to add tracked account:', error);
    }
  };

  const toggleAccount = async (id: string, currentState: boolean) => {
    try {
      await fetch(`${API_BASE_URL}/api/tracked-accounts/${id}/toggle`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ enabled: !currentState }),
      });
      fetchAccounts();
    } catch (error) {
      console.error('Failed to toggle account:', error);
    }
  };

  const deleteAccount = async (id: string) => {
    if (!confirm('Are you sure you want to delete this tracked source?')) return;

    try {
      await fetch(`${API_BASE_URL}/api/tracked-accounts/${id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      fetchAccounts();
    } catch (error) {
      console.error('Failed to delete account:', error);
    }
  };

  const fetchNow = async (accountId: string, _platform: string, identifier: string) => {
    if (!confirm(`Fetch new content from ${identifier} now?`)) return;

    try {
      const response = await fetch(`${API_BASE_URL}/api/tracked-accounts/${accountId}/fetch`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        const errorText = await response.text();
        alert(`Failed to fetch: ${errorText}`);
        return;
      }

      const result = await response.json();
      alert(`Success! Fetched ${result.fetched} new items. ${result.message}`);

      // Refresh the accounts list to update last_fetched_at
      fetchAccounts();
    } catch (error) {
      console.error('Failed to trigger fetch:', error);
      alert('Failed to trigger fetch. Check console for details.');
    }
  };

  const viewSources = async (accountId: string) => {
    setViewingSourcesFor(accountId);
    setLoadingSources(true);

    try {
      const response = await fetch(`${API_BASE_URL}/api/sources?account_id=${accountId}`, {
        headers: getAuthHeaders(),
      });
      const data = await response.json();
      setSources(data.sources || []);
      setLoadingSources(false);
    } catch (error) {
      console.error('Failed to fetch sources:', error);
      setLoadingSources(false);
    }
  };

  const getPlatformColor = (platform: string) => {
    switch (platform) {
      case 'twitter': return 'text-blue-400 border-blue-400';
      case 'rss': return 'text-yellow-400 border-yellow-400';
      case 'telegram': return 'text-cyan-400 border-cyan-400';
      default: return 'text-fog border-steel';
    }
  };

  const getPlaceholder = (platform: string) => {
    switch (platform) {
      case 'twitter': return '@username (e.g., @Reuters)';
      case 'rss': return 'Feed URL (e.g., https://...)';
      case 'telegram': return '@channel (e.g., @durov)';
      default: return 'Enter identifier';
    }
  };

  return (
    <div className="space-y-6">
      <div className="border-l-4 border-terminal pl-6 flex items-center justify-between">
        <div>
          <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
            TRACKED SOURCES
          </h2>
          <p className="text-sm text-smoke font-mono mt-2">
            Monitor Twitter accounts, Telegram channels, and RSS feeds
          </p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={fetchAccounts}
            className="px-4 py-3 border-2 border-steel text-chalk hover:border-terminal hover:text-terminal transition-all font-mono text-sm font-bold flex items-center gap-2"
          >
            <RefreshCw className="w-4 h-4" />
            REFRESH
          </button>
          <button
            onClick={() => setShowAddForm(!showAddForm)}
            className="px-6 py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            {showAddForm ? 'CANCEL' : 'ADD SOURCE'}
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4">
        {['twitter', 'telegram', 'rss'].map((platform) => {
          const count = accounts.filter((a) => a.platform === platform).length;
          const enabled = accounts.filter((a) => a.platform === platform && a.enabled).length;
          return (
            <div key={platform} className="border-2 border-steel bg-concrete p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-mono font-bold text-smoke uppercase">{platform}</span>
                <span className={`text-xs font-mono font-bold ${getPlatformColor(platform)}`}>
                  {enabled}/{count}
                </span>
              </div>
              <div className="text-2xl font-display font-black text-chalk">{count}</div>
            </div>
          );
        })}
      </div>

      {/* Add Form */}
      {showAddForm && (
        <div className="border-2 border-terminal bg-concrete">
          <div className="px-6 py-4 border-b-2 border-terminal bg-void/50">
            <h3 className="font-display font-black text-base text-chalk">ADD NEW SOURCE</h3>
          </div>
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-mono text-chalk font-bold mb-2">PLATFORM *</label>
                <select
                  value={newAccount.platform}
                  onChange={(e) => setNewAccount({ ...newAccount, platform: e.target.value })}
                  className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono"
                  required
                >
                  <option value="twitter">Twitter</option>
                  <option value="telegram">Telegram (coming soon)</option>
                  <option value="rss">RSS Feed</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-mono text-chalk font-bold mb-2">
                  CHECK INTERVAL (minutes)
                </label>
                <input
                  type="number"
                  min="1"
                  max="60"
                  value={newAccount.fetch_interval_minutes}
                  onChange={(e) => setNewAccount({ ...newAccount, fetch_interval_minutes: parseInt(e.target.value) })}
                  className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono"
                />
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-mono text-chalk font-bold mb-2">
                  IDENTIFIER *
                </label>
                <input
                  type="text"
                  value={newAccount.account_identifier}
                  onChange={(e) => setNewAccount({ ...newAccount, account_identifier: e.target.value })}
                  placeholder={getPlaceholder(newAccount.platform)}
                  className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono"
                  required
                />
                <p className="text-xs text-smoke font-mono mt-1">
                  {getPlaceholder(newAccount.platform)}
                </p>
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-mono text-chalk font-bold mb-2">
                  DISPLAY NAME
                </label>
                <input
                  type="text"
                  value={newAccount.display_name}
                  onChange={(e) => setNewAccount({ ...newAccount, display_name: e.target.value })}
                  placeholder="Optional - will use identifier if empty"
                  className="w-full px-4 py-2 border-2 border-steel bg-void text-chalk font-mono"
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4">
              <button
                type="button"
                onClick={() => setShowAddForm(false)}
                className="px-6 py-2 border border-steel text-fog hover:border-iron hover:text-chalk transition-all font-mono text-sm"
              >
                CANCEL
              </button>
              <button
                type="submit"
                className="px-6 py-2 border-2 border-terminal bg-terminal text-void hover:bg-void hover:text-terminal transition-all font-mono text-sm font-bold"
              >
                ADD SOURCE
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Sources List */}
      {loading ? (
        <div className="text-center py-12 text-smoke font-mono">Loading tracked sources...</div>
      ) : accounts.length === 0 ? (
        <div className="text-center py-12 border-2 border-steel bg-concrete">
          <p className="text-smoke font-mono mb-4">No tracked sources yet</p>
          <button
            onClick={() => setShowAddForm(true)}
            className="px-6 py-2 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold"
          >
            ADD YOUR FIRST SOURCE
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {accounts.map((account) => (
            <div
              key={account.id}
              className={`border-2 bg-concrete transition-all ${
                account.enabled ? 'border-terminal' : 'border-steel opacity-60'
              }`}
            >
              <div className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <span className={`px-2 py-1 border text-xs font-mono font-bold uppercase ${getPlatformColor(account.platform)}`}>
                        {account.platform}
                      </span>
                      <span className="text-base font-mono font-bold text-chalk">
                        {account.account_identifier}
                      </span>
                      {account.display_name && (
                        <span className="text-sm font-mono text-smoke">
                          ({account.display_name})
                        </span>
                      )}
                    </div>

                    <div className="flex items-center gap-4 text-xs font-mono text-smoke">
                      <span>Check every {account.fetch_interval_minutes}min</span>
                      {account.last_fetched_at && (
                        <span>Last fetched: {formatDateTime(account.last_fetched_at)}</span>
                      )}
                      <span>Added: {formatDate(account.created_at)}</span>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => viewSources(account.id)}
                      className="p-2 border border-electric text-electric hover:bg-electric hover:text-void transition-all"
                      title="View Sources"
                    >
                      <Eye className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => fetchNow(account.id, account.platform, account.account_identifier)}
                      className="p-2 border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all"
                      title="Fetch Now"
                    >
                      <Download className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => toggleAccount(account.id, account.enabled)}
                      className={`p-2 border transition-all ${
                        account.enabled
                          ? 'border-terminal text-terminal hover:bg-terminal hover:text-void'
                          : 'border-steel text-steel hover:border-terminal hover:text-terminal'
                      }`}
                      title={account.enabled ? 'Disable' : 'Enable'}
                    >
                      {account.enabled ? <Power className="w-4 h-4" /> : <PowerOff className="w-4 h-4" />}
                    </button>
                    <button
                      onClick={() => deleteAccount(account.id)}
                      className="p-2 border border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Sources Viewer Modal */}
      {viewingSourcesFor && (
        <div className="fixed inset-0 bg-void/80 flex items-center justify-center z-50 p-4">
          <div className="bg-concrete border-2 border-terminal max-w-4xl w-full max-h-[80vh] overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b-2 border-terminal bg-void/50 flex items-center justify-between">
              <h3 className="font-display font-black text-lg text-chalk">
                SOURCES: {accounts.find(a => a.id === viewingSourcesFor)?.account_identifier}
              </h3>
              <button
                onClick={() => setViewingSourcesFor(null)}
                className="px-4 py-2 border border-steel text-fog hover:border-terminal hover:text-terminal transition-all font-mono text-sm font-bold"
              >
                CLOSE
              </button>
            </div>
            <div className="p-6 overflow-y-auto">
              {loadingSources ? (
                <div className="text-center py-12 text-smoke font-mono">Loading sources...</div>
              ) : sources.length === 0 ? (
                <div className="text-center py-12 text-smoke font-mono">
                  No sources found for this account yet
                </div>
              ) : (
                <div className="space-y-3">
                  {sources.map((source) => (
                    <div key={source.id} className="border-2 border-steel bg-void p-4">
                      <div className="flex items-start justify-between mb-2">
                        <div className="flex-1">
                          <h4 className="font-mono font-bold text-chalk mb-1 break-all">
                            {source.title || source.url || 'Untitled'}
                          </h4>
                          <p className="text-sm text-smoke font-mono mb-2">
                            {source.content?.substring(0, 200)}
                            {source.content?.length > 200 && '...'}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-4 text-xs font-mono text-smoke">
                        <span>ID: {source.id.substring(0, 8)}</span>
                        {source.source_url && (
                          <a
                            href={source.source_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-electric hover:underline"
                          >
                            View Original
                          </a>
                        )}
                        <span>Published: {formatDateTime(source.published_at)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
