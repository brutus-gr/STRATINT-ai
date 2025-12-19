import { X } from 'lucide-react';
import { useState, useEffect } from 'react';

import { API_BASE_URL } from '../utils/api';
import { getAuthHeaders } from '../utils/auth';

interface ConfigModalProps {
  connectorId: string;
  connectorName: string;
  onClose: () => void;
  onSave: (config: Record<string, string>) => void;
}

export function ConfigModal({ connectorId, connectorName, onClose, onSave }: ConfigModalProps) {
  const [config, setConfig] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  // Load existing configuration
  useEffect(() => {
    fetchConfig();
  }, [connectorId]);

  const fetchConfig = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/connectors/${connectorId}/config`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setConfig(data.config || {});
      }
    } catch (error) {
      console.error('Failed to fetch config:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const response = await fetch(`${API_BASE_URL}/api/connectors/${connectorId}/config`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ config }),
      });

      if (response.ok) {
        onSave(config);
        onClose();
      } else {
        alert('Failed to save configuration');
      }
    } catch (error) {
      console.error('Failed to save config:', error);
      alert('Failed to save configuration');
    } finally {
      setLoading(false);
    }
  };

  const getFields = () => {
    switch (connectorId) {
      case 'twitter':
        return [
          {
            key: 'bearer_token',
            label: 'Bearer Token',
            type: 'password',
            placeholder: 'Enter Twitter API Bearer Token',
            required: true,
          },
        ];
      case 'rss':
        return []; // RSS has no platform-level config
      case 'telegram':
        return [
          {
            key: 'bot_token',
            label: 'Bot Token',
            type: 'password',
            placeholder: 'Enter Telegram Bot Token',
            required: true,
          },
        ];
      default:
        return [];
    }
  };

  const fields = getFields();

  return (
    <div className="fixed inset-0 bg-void/90 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="border-4 border-terminal bg-concrete max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="px-6 py-4 border-b-4 border-terminal bg-void/50 flex items-center justify-between sticky top-0 z-10">
          <h3 className="font-display font-black text-xl text-chalk">
            CONFIGURE {connectorName.toUpperCase()}
          </h3>
          <button
            onClick={onClose}
            className="p-2 border-2 border-threat-critical text-threat-critical hover:bg-threat-critical hover:text-void transition-all"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-6 space-y-6">
          {fields.map((field) => (
            <div key={field.key}>
              <label className="block text-sm font-mono text-chalk font-bold mb-2">
                {field.label} {field.required && <span className="text-threat-critical">*</span>}
              </label>
              {field.type === 'textarea' ? (
                <textarea
                  value={config[field.key] || ''}
                  onChange={(e) => setConfig({ ...config, [field.key]: e.target.value })}
                  className="w-full px-4 py-3 border-2 border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                  placeholder={field.placeholder}
                  rows={6}
                  required={field.required}
                />
              ) : (
                <input
                  type={field.type}
                  value={config[field.key] || ''}
                  onChange={(e) => setConfig({ ...config, [field.key]: e.target.value })}
                  className="w-full px-4 py-3 border-2 border-steel bg-void text-chalk font-mono text-sm focus:border-terminal focus:outline-none"
                  placeholder={field.placeholder}
                  required={field.required}
                />
              )}
            </div>
          ))}

          {/* Info Box */}
          {fields.length > 0 && (
            <div className="border-2 border-electric bg-electric/5 p-4">
              <p className="text-xs font-mono text-fog leading-relaxed">
                {connectorId === 'twitter' && 'Configure your Twitter API v2 Bearer Token. Get it from: https://developer.twitter.com/en/portal/dashboard'}
                {connectorId === 'telegram' && 'Configure your Telegram Bot Token. Get it from: @BotFather'}
              </p>
            </div>
          )}

          {/* RSS has no config */}
          {fields.length === 0 && (
            <div className="border-2 border-steel bg-void p-4">
              <p className="text-xs font-mono text-fog leading-relaxed">
                RSS feeds don't require platform-level configuration. Add individual RSS feed URLs in the <strong className="text-chalk">SOURCES</strong> tab instead.
              </p>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-3 border-2 border-steel text-smoke hover:bg-steel hover:text-void transition-all font-mono text-sm font-bold"
            >
              {fields.length === 0 ? 'CLOSE' : 'CANCEL'}
            </button>
            {fields.length > 0 && (
              <button
                type="submit"
                disabled={loading}
                className="flex-1 py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'SAVING...' : '[SAVE CONFIGURATION]'}
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}
