import { useState, useEffect } from 'react';
import { AlertTriangle, CheckCircle, Trash2 } from 'lucide-react';
import { formatDateTime } from '../utils/dateFormat';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface IngestionError {
  id: string;
  platform: string;
  error_type: string;
  url: string;
  error_msg: string;
  metadata: string;
  created_at: string;
  resolved: boolean;
  resolved_at?: string;
}

export function IngestionErrorsTab() {
  const [errors, setErrors] = useState<IngestionError[]>([]);
  const [loading, setLoading] = useState(true);
  const [unresolvedOnly, setUnresolvedOnly] = useState(true);
  const [unresolvedCount, setUnresolvedCount] = useState(0);

  const fetchErrors = async () => {
    try {
      const url = `${API_BASE_URL}/api/ingestion-errors?limit=100&unresolved_only=${unresolvedOnly}`;
      const response = await fetch(url, {
        headers: getAuthHeaders(),
      });
      if (!response.ok) throw new Error('Failed to fetch errors');
      const data = await response.json();
      setErrors(data.errors || []);
      setUnresolvedCount(data.unresolved_count || 0);
      setLoading(false);
    } catch (err) {
      console.error('Error fetching ingestion errors:', err);
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchErrors();
    // Refresh every 30 seconds
    const interval = setInterval(fetchErrors, 30000);
    return () => clearInterval(interval);
  }, [unresolvedOnly]);

  const handleResolve = async (errorId: string) => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/ingestion-errors/${errorId}/resolve`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to resolve error');
      }

      // Refresh the errors list
      fetchErrors();
    } catch (err) {
      console.error('Error resolving error:', err);
      alert('Failed to resolve error');
    }
  };

  const handleDelete = async (errorId: string) => {
    if (!confirm('Are you sure you want to delete this error?')) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE_URL}/api/ingestion-errors/${errorId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to delete error');
      }

      // Refresh the errors list
      fetchErrors();
    } catch (err) {
      console.error('Error deleting error:', err);
      alert('Failed to delete error');
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-l-4 border-terminal pl-6">
        <h2 className="font-display font-black text-3xl text-chalk tracking-tight">
          INGESTION ERRORS
        </h2>
        <p className="text-sm text-smoke font-mono mt-2">
          RSS feed and scraping errors requiring attention
        </p>
      </div>

      {/* Stats Summary */}
      <div className="grid grid-cols-3 gap-6">
        <div className="border-2 border-steel bg-concrete p-6">
          <div className="flex items-center gap-3 mb-2">
            <AlertTriangle className="w-5 h-5 text-warning" />
            <span className="text-xs font-mono text-smoke font-medium">UNRESOLVED ERRORS</span>
          </div>
          <span className="text-3xl font-mono font-bold text-warning">{unresolvedCount}</span>
        </div>
        <div className="border-2 border-steel bg-concrete p-6">
          <div className="flex items-center gap-3 mb-2">
            <CheckCircle className="w-5 h-5 text-terminal" />
            <span className="text-xs font-mono text-smoke font-medium">TOTAL ERRORS</span>
          </div>
          <span className="text-3xl font-mono font-bold text-chalk">{errors.length}</span>
        </div>
      </div>

      {/* Filter */}
      <div className="flex gap-2">
        <button
          onClick={() => setUnresolvedOnly(true)}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            unresolvedOnly
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          UNRESOLVED ONLY ({unresolvedCount})
        </button>
        <button
          onClick={() => setUnresolvedOnly(false)}
          className={`px-4 py-2 border font-mono text-xs font-bold uppercase transition-all ${
            !unresolvedOnly
              ? 'border-terminal bg-terminal text-void'
              : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
          }`}
        >
          ALL ERRORS ({errors.length})
        </button>
      </div>

      {/* Errors Table */}
      {loading ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <AlertTriangle className="w-16 h-16 text-terminal/50 mx-auto mb-4 animate-pulse" />
          <p className="text-lg font-mono text-chalk font-bold">LOADING ERRORS...</p>
        </div>
      ) : errors.length === 0 ? (
        <div className="border-2 border-steel bg-concrete p-16 text-center">
          <CheckCircle className="w-16 h-16 text-terminal/50 mx-auto mb-4" />
          <p className="text-lg font-mono text-chalk font-bold">NO ERRORS FOUND</p>
          <p className="text-sm font-mono text-fog mt-2">
            {unresolvedOnly
              ? 'All errors have been resolved!'
              : 'No ingestion errors have been logged yet.'
            }
          </p>
        </div>
      ) : (
        <div className="border-2 border-steel bg-concrete overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-void border-b-2 border-steel">
                <tr className="text-left">
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">TIMESTAMP</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">PLATFORM</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">ERROR TYPE</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">URL</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">MESSAGE</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">STATUS</th>
                  <th className="px-4 py-3 text-xs font-mono font-bold text-smoke">ACTIONS</th>
                </tr>
              </thead>
              <tbody>
                {errors.map((error, idx) => (
                  <tr
                    key={error.id}
                    className={`border-b border-steel hover:bg-void/50 transition-colors ${
                      idx % 2 === 0 ? 'bg-void/20' : ''
                    }`}
                  >
                    <td className="px-4 py-3 text-xs font-mono text-fog">
                      {formatDateTime(error.created_at)}
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-1 text-xs font-mono font-bold border border-steel text-chalk uppercase">
                        {error.platform}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-fog">
                      {error.error_type}
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-electric max-w-xs truncate">
                      <a
                        href={error.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:underline"
                      >
                        {error.url}
                      </a>
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-fog max-w-md truncate">
                      {error.error_msg}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 py-1 text-xs font-mono font-bold border uppercase ${
                          error.resolved
                            ? 'border-terminal text-terminal'
                            : 'border-warning text-warning'
                        }`}
                      >
                        {error.resolved ? 'RESOLVED' : 'UNRESOLVED'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        {!error.resolved && (
                          <button
                            onClick={() => handleResolve(error.id)}
                            className="px-2 py-1 text-xs font-mono font-bold border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all"
                            title="Mark as resolved"
                          >
                            <CheckCircle className="w-4 h-4" />
                          </button>
                        )}
                        <button
                          onClick={() => handleDelete(error.id)}
                          className="px-2 py-1 text-xs font-mono font-bold border border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
                          title="Delete error"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
