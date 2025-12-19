import { motion } from 'framer-motion';
import { MapPin, ChevronRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { Event } from '../types';
import { formatDateTime } from '../utils/dateFormat';

interface EventCardProps {
  event: Event;
  index: number;
}

export function EventCard({ event, index }: EventCardProps) {
  const getMagnitudeColor = (magnitude: number) => {
    if (magnitude >= 9.0) return 'text-threat-critical';
    if (magnitude >= 7.0) return 'text-threat-high';
    if (magnitude >= 5.0) return 'text-threat-medium';
    if (magnitude >= 3.0) return 'text-threat-low';
    return 'text-threat-info';
  };

  const getMagnitudeBarColor = (magnitude: number) => {
    if (magnitude >= 9.0) return 'bg-threat-critical';
    if (magnitude >= 7.0) return 'bg-threat-high';
    if (magnitude >= 5.0) return 'bg-threat-medium';
    if (magnitude >= 3.0) return 'bg-threat-low';
    return 'bg-threat-info';
  };

  const getMagnitudeBarWidth = (magnitude: number) => {
    return `${(magnitude / 10) * 100}%`;
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

  return (
    <Link to={`/events/${event.id}`} className="block">
      <motion.article
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: index * 0.05 }}
        className="group relative border-2 border-steel bg-concrete hover:bg-iron hover:border-fog hover:shadow-lg hover:shadow-terminal/5 transition-all cursor-pointer overflow-hidden"
      >
        {/* Top bar with metadata */}
        <div className="px-3 md:px-5 py-3 border-b-2 border-steel bg-void/50 font-mono text-xs flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div className="flex items-center gap-3 md:gap-5 flex-wrap">
            <Link
              to={`/category/${event.category}`}
              onClick={(e) => e.stopPropagation()}
              className={`px-2 md:px-3 py-1 border-2 font-bold text-xs hover:scale-105 transition-transform ${getCategoryColor(event.category)}`}
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
              <span className="truncate max-w-[150px]">{event.location.country || 'Unknown'}</span>
            </div>
          )}
        </div>

        <div className="text-smoke text-xs">
          {formatDateTime(event.timestamp)}
        </div>
      </div>

      {/* Content */}
      <div className="p-4 md:p-6 space-y-4">
        {/* Title */}
        <h2 className="text-base md:text-lg font-mono font-bold text-chalk leading-tight flex items-start gap-2 md:gap-3">
          <ChevronRight className="w-4 md:w-5 h-4 md:h-5 mt-0.5 md:mt-1 text-terminal flex-shrink-0 group-hover:translate-x-1 transition-transform" />
          <span className="group-hover:text-white transition-colors">{event.title}</span>
        </h2>


        {/* Magnitude bar */}
        <div className="pl-6 md:pl-8">
          <div className="h-2 w-full bg-steel overflow-hidden relative">
            <motion.div
              initial={{ width: 0 }}
              animate={{ width: getMagnitudeBarWidth(event.magnitude) }}
              transition={{ duration: 0.8, delay: 0.2, ease: "easeOut" }}
              className={`h-full ${getMagnitudeBarColor(event.magnitude)} relative`}
            >
              {event.magnitude >= 8.0 && (
                <div className="absolute inset-0 animate-pulse-slow opacity-50 blur-sm" style={{ backgroundColor: 'currentColor' }} />
              )}
            </motion.div>
          </div>
        </div>

        {/* Entities */}
        {event.entities.length > 0 && (
          <div className="pl-6 md:pl-8 flex items-start md:items-center gap-2 flex-wrap text-xs">
            <span className="text-smoke font-mono font-medium whitespace-nowrap">ENTITIES:</span>
            {event.entities.slice(0, 5).map((entity) => (
              <Link
                key={entity.id}
                to={`/entity/${encodeURIComponent(entity.name)}`}
                onClick={(e) => e.stopPropagation()}
                className="px-2 py-1 bg-steel text-chalk font-mono border border-iron hover:border-terminal hover:text-terminal hover:scale-105 transition-all cursor-pointer text-xs"
              >
                {entity.name}
              </Link>
            ))}
            {event.entities.length > 5 && (
              <span className="text-smoke font-mono">
                +{event.entities.length - 5} more
              </span>
            )}
          </div>
        )}

        {/* Sources */}
        <div className="pl-6 md:pl-8 flex items-center gap-2 text-xs">
          <span className="text-smoke font-mono font-medium">SOURCES:</span>
          <span className="text-terminal font-mono font-bold">
            {event.sources.length} verified
          </span>
        </div>
      </div>

      {/* Glitch effect on high magnitude events */}
      {event.magnitude >= 8.5 && (
        <div className="absolute top-0 left-0 w-1 h-full bg-threat-critical animate-pulse-slow" />
      )}
    </motion.article>
    </Link>
  );
}
