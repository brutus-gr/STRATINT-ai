import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { Header } from '../components/Header';
import { ShareButtons } from '../components/ShareButtons';
import { MapPin, ArrowLeft, ExternalLink } from 'lucide-react';
import type { Event } from '../types';
import { formatDateTime } from '../utils/dateFormat';

import { API_BASE_URL } from '../utils/api';

export function EventDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [event, setEvent] = useState<Event | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch event
  useEffect(() => {
    const fetchEvent = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/events/${id}`);
        if (!response.ok) {
          if (response.status === 404) {
            throw new Error('Event not found');
          }
          throw new Error('Failed to fetch event');
        }
        const data = await response.json();
        setEvent(data);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching event:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        setLoading(false);
      }
    };

    if (id) {
      fetchEvent();
    }
  }, [id]);

  const getMagnitudeColor = (magnitude: number) => {
    if (magnitude >= 9.0) return 'text-threat-critical';
    if (magnitude >= 7.0) return 'text-threat-high';
    if (magnitude >= 5.0) return 'text-threat-medium';
    if (magnitude >= 3.0) return 'text-threat-low';
    return 'text-threat-info';
  };

  const getCategoryColor = (category: string) => {
    const colors: Record<string, string> = {
      military: 'text-threat-high border-threat-high/30 bg-threat-high/10',
      cyber: 'text-cyber border-cyber/30 bg-cyber/10',
      geopolitics: 'text-electric border-electric/30 bg-electric/10',
      terrorism: 'text-threat-critical border-threat-critical/30 bg-threat-critical/10',
      disaster: 'text-warning border-warning/30 bg-warning/10',
      economic: 'text-threat-medium border-threat-medium/30 bg-threat-medium/10',
    };
    return colors[category] || 'text-fog border-steel bg-concrete';
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-void text-chalk">
        <div className="scan-line" />
        <Header  />
        <div className="pt-16 p-8">
          <div className="max-w-5xl mx-auto border-2 border-steel bg-concrete p-16 text-center space-y-6">
            <div className="text-5xl text-terminal animate-pulse">◉</div>
            <div className="space-y-2">
              <p className="text-chalk font-mono text-lg font-bold">LOADING EVENT...</p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (error || !event) {
    return (
      <div className="min-h-screen bg-void text-chalk">
        <div className="scan-line" />
        <Header  />
        <div className="pt-16 p-8">
          <div className="max-w-5xl mx-auto border-2 border-warning bg-concrete p-16 text-center space-y-6">
            <div className="text-5xl text-warning">⚠</div>
            <div className="space-y-2">
              <p className="text-chalk font-mono text-lg font-bold">{error || 'EVENT NOT FOUND'}</p>
            </div>
            <button
              onClick={() => navigate('/')}
              className="mt-4 px-6 py-3 border-2 border-terminal text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold"
            >
              [RETURN TO FEED]
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-void text-chalk">
      <div className="scan-line" />
      <Header  />

      <div className="pt-16 p-8">
        <div className="max-w-5xl mx-auto space-y-6">
          {/* Back Button */}
          <Link
            to="/"
            className="inline-flex items-center gap-2 text-terminal hover:text-electric transition-colors font-mono text-sm font-bold"
          >
            <ArrowLeft className="w-4 h-4" />
            BACK TO FEED
          </Link>

          {/* Event Detail Card */}
          <article className="border-2 border-steel bg-concrete">
            {/* Top bar with metadata */}
            <div className="px-5 py-3 border-b-2 border-steel bg-void/50 font-mono text-xs flex items-center justify-between flex-wrap gap-3">
              <div className="flex items-center gap-5 flex-wrap">
                <Link
                  to={`/category/${event.category}`}
                  className={`px-3 py-1 border-2 font-bold text-xs hover:scale-105 transition-transform ${getCategoryColor(event.category)}`}
                >
                  [{event.category.toUpperCase()}]
                </Link>

                <div className="flex items-center gap-2">
                  <span className="text-smoke">MAG:</span>
                  <span className={`font-bold ${getMagnitudeColor(event.magnitude)}`}>
                    {event.magnitude.toFixed(1)}
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <span className="text-smoke">CONF:</span>
                  <span className="text-chalk font-medium">{event.confidence.score.toFixed(2)}</span>
                </div>

                {event.location && (
                  <div className="flex items-center gap-1 text-fog">
                    <MapPin className="w-3 h-3" />
                    <span>{event.location.country || 'Unknown'}</span>
                  </div>
                )}
              </div>

              <div className="text-smoke">
                {formatDateTime(event.timestamp)}
              </div>
            </div>

            {/* Content */}
            <div className="p-8 space-y-6">
              {/* Title */}
              <h1 className="text-2xl font-mono font-bold text-chalk leading-tight">
                {event.title}
              </h1>

              {/* Share Buttons */}
              <ShareButtons event={event} />

              {/* Confidence Details */}
              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 border border-steel bg-void/30">
                  <div className="text-xs font-mono text-smoke mb-1">CONFIDENCE SCORE</div>
                  <div className="text-2xl font-bold text-terminal">{event.confidence.score.toFixed(2)}</div>
                </div>
                <div className="p-4 border border-steel bg-void/30">
                  <div className="text-xs font-mono text-smoke mb-1">MAGNITUDE</div>
                  <div className={`text-2xl font-bold ${getMagnitudeColor(event.magnitude)}`}>
                    {event.magnitude.toFixed(1)}
                  </div>
                </div>
              </div>

              {/* Confidence Reasoning */}
              {event.confidence.reasoning && (
                <div className="space-y-2">
                  <h3 className="text-sm font-mono font-bold text-chalk">CONFIDENCE REASONING</h3>
                  <div className="p-4 bg-void border border-steel text-xs text-fog font-mono">
                    {event.confidence.reasoning}
                  </div>
                </div>
              )}

              {/* Entities */}
              {event.entities.length > 0 && (
                <div className="space-y-3">
                  <h3 className="text-sm font-mono font-bold text-chalk">ENTITIES ({event.entities.length})</h3>
                  <div className="flex items-center gap-2 flex-wrap">
                    {event.entities.map((entity) => (
                      <Link
                        key={entity.id}
                        to={`/entity/${encodeURIComponent(entity.name)}`}
                        className="px-3 py-2 bg-steel text-chalk font-mono border border-iron hover:border-terminal hover:text-terminal hover:scale-105 transition-all cursor-pointer text-sm"
                      >
                        {entity.name}
                        <span className="ml-2 text-xs text-fog">({entity.type})</span>
                      </Link>
                    ))}
                  </div>
                </div>
              )}

              {/* Location Details */}
              {event.location && (
                <div className="space-y-3">
                  <h3 className="text-sm font-mono font-bold text-chalk">LOCATION</h3>
                  <div className="grid grid-cols-2 gap-4">
                    {event.location.country && (
                      <div className="p-3 border border-steel bg-void/30">
                        <div className="text-xs font-mono text-smoke mb-1">COUNTRY</div>
                        <div className="text-sm font-mono text-chalk">{event.location.country}</div>
                      </div>
                    )}
                    {event.location.city && (
                      <div className="p-3 border border-steel bg-void/30">
                        <div className="text-xs font-mono text-smoke mb-1">CITY</div>
                        <div className="text-sm font-mono text-chalk">{event.location.city}</div>
                      </div>
                    )}
                    {event.location.latitude && event.location.longitude && (
                      <div className="p-3 border border-steel bg-void/30 col-span-2">
                        <div className="text-xs font-mono text-smoke mb-1">COORDINATES</div>
                        <div className="text-sm font-mono text-chalk">
                          {event.location.latitude.toFixed(4)}, {event.location.longitude.toFixed(4)}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Sources */}
              <div className="space-y-3">
                <h3 className="text-sm font-mono font-bold text-chalk">SOURCES ({event.sources.length})</h3>
                <div className="space-y-2">
                  {event.sources.map((source) => (
                    <div key={source.id} className="p-4 border border-steel bg-void/30 hover:bg-void/50 transition-colors">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 space-y-2">
                          <div className="font-mono text-sm text-chalk font-bold break-all">
                            {source.title || source.url || 'Untitled Source'}
                          </div>
                          <div className="text-xs text-smoke font-mono">
                            Retrieved: {formatDateTime(source.retrieved_at)}
                          </div>
                        </div>
                        {source.url && (
                          <a
                            href={source.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-1.5 px-3 py-1.5 border border-electric text-electric hover:bg-electric hover:text-void transition-all font-mono font-medium text-xs whitespace-nowrap"
                          >
                            <ExternalLink className="w-3 h-3" />
                            <span>VIEW</span>
                          </a>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </article>
        </div>
      </div>
    </div>
  );
}
