import { useState, useEffect } from 'react';
import { ExternalLink, Clock, Hash } from 'lucide-react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

interface PostedTweet {
  id: number;
  event_id: string;
  tweet_id: string;
  tweet_text: string;
  posted_at: string;
}

export function PostedTweetsTab() {
  const [tweets, setTweets] = useState<PostedTweet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchTweets();
  }, []);

  const fetchTweets = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('admin_token');
      const response = await fetch(`${API_BASE_URL}/api/admin/posted-tweets`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });

      if (!response.ok) throw new Error('Failed to fetch posted tweets');

      const data = await response.json();
      setTweets(data || []);
      setError(null);
    } catch (err) {
      console.error('Error fetching tweets:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch tweets');
    } finally {
      setLoading(false);
    }
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) {
      return `${diffMins}m ago`;
    } else if (diffHours < 24) {
      return `${diffHours}h ago`;
    } else if (diffDays < 7) {
      return `${diffDays}d ago`;
    } else {
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
      });
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="border-2 border-terminal bg-terminal/5 p-4">
          <h2 className="text-2xl font-display font-black text-terminal mb-2">
            POSTED TWEETS
          </h2>
          <p className="font-mono text-sm text-fog">
            History of auto-posted tweets from events (last 100)
          </p>
        </div>

        <div className="border border-steel bg-concrete/30 p-8 text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-terminal"></div>
          <p className="mt-4 font-mono text-sm text-fog">Loading tweets...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="border-2 border-terminal bg-terminal/5 p-4">
          <h2 className="text-2xl font-display font-black text-terminal mb-2">
            POSTED TWEETS
          </h2>
          <p className="font-mono text-sm text-fog">
            History of auto-posted tweets from events
          </p>
        </div>

        <div className="border-2 border-threat-critical bg-threat-critical/10 p-4">
          <p className="font-mono text-sm text-threat-critical">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="border-2 border-terminal bg-terminal/5 p-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-display font-black text-terminal mb-2">
              POSTED TWEETS
            </h2>
            <p className="font-mono text-sm text-fog">
              History of auto-posted tweets from events • {tweets.length} total
            </p>
          </div>
          <button
            onClick={fetchTweets}
            className="px-4 py-2 border-2 border-terminal bg-terminal/10 text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold"
          >
            REFRESH
          </button>
        </div>
      </div>

      {/* Info Box */}
      <div className="border border-steel bg-concrete/20 p-4">
        <p className="font-mono text-sm text-fog">
          <strong className="text-chalk">Context-Aware Posting:</strong> Each event is evaluated against recent tweets (last 24h) to prevent redundant posts.
          The AI can SKIP redundant events, POST new information, or send an UPDATE if there are significant new details.
        </p>
      </div>

      {/* Tweets List */}
      {tweets.length === 0 ? (
        <div className="border border-steel bg-concrete/30 p-8 text-center">
          <p className="font-mono text-fog">No tweets posted yet</p>
        </div>
      ) : (
        <div className="space-y-3">
          {tweets.map((tweet) => (
            <div
              key={tweet.id}
              className="border border-steel bg-concrete/30 hover:bg-concrete/50 transition-colors"
            >
              <div className="p-4">
                {/* Header: Time and Links */}
                <div className="flex items-center justify-between mb-3 pb-3 border-b border-steel/50">
                  <div className="flex items-center gap-2 text-xs font-mono text-fog">
                    <Clock className="w-3 h-3" />
                    <span>{formatTime(tweet.posted_at)}</span>
                    <span className="text-steel">•</span>
                    <span>{new Date(tweet.posted_at).toLocaleString()}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <a
                      href={`https://twitter.com/i/web/status/${tweet.tweet_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 px-2 py-1 border border-terminal text-terminal hover:bg-terminal hover:text-void transition-all text-xs font-mono"
                    >
                      <ExternalLink className="w-3 h-3" />
                      <span>VIEW TWEET</span>
                    </a>
                    <a
                      href={`/events/${tweet.event_id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 px-2 py-1 border border-steel text-fog hover:border-chalk hover:text-chalk transition-all text-xs font-mono"
                    >
                      <Hash className="w-3 h-3" />
                      <span>EVENT</span>
                    </a>
                  </div>
                </div>

                {/* Tweet Text */}
                <div className="font-mono text-sm text-chalk whitespace-pre-wrap leading-relaxed">
                  {tweet.tweet_text}
                </div>

                {/* Footer: IDs */}
                <div className="mt-3 pt-3 border-t border-steel/50 flex items-center gap-4 text-xs font-mono text-fog">
                  <span>Tweet ID: <span className="text-chalk">{tweet.tweet_id}</span></span>
                  <span className="text-steel">•</span>
                  <span>Event ID: <span className="text-chalk">{tweet.event_id.substring(0, 8)}...</span></span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
