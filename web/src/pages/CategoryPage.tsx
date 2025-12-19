import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Header } from '../components/Header';
import { EventCard } from '../components/EventCard';
import { ArrowLeft, Shield, Zap, Globe, Bomb, CloudRain, TrendingUp } from 'lucide-react';
import type { Event } from '../types';

import { API_BASE_URL } from '../utils/api';

const categoryIcons: Record<string, any> = {
  military: Shield,
  cyber: Zap,
  geopolitics: Globe,
  terrorism: Bomb,
  disaster: CloudRain,
  economic: TrendingUp,
};

const categoryDescriptions: Record<string, string> = {
  military: 'Military operations, conflicts, and defense activities',
  cyber: 'Cyber attacks, data breaches, and digital security incidents',
  geopolitics: 'International relations, diplomacy, and political developments',
  terrorism: 'Terrorist activities, threats, and counter-terrorism operations',
  disaster: 'Natural disasters, humanitarian crises, and emergency situations',
  economic: 'Economic developments, market changes, and financial impacts',
};

const getCategoryColor = (category: string) => {
  const colors: Record<string, string> = {
    military: 'text-threat-high border-threat-high',
    cyber: 'text-cyber border-cyber',
    geopolitics: 'text-electric border-electric',
    terrorism: 'text-threat-critical border-threat-critical',
    disaster: 'text-warning border-warning',
    economic: 'text-threat-medium border-threat-medium',
  };
  return colors[category] || 'text-fog border-steel';
};

export function CategoryPage() {
  const { name } = useParams<{ name: string }>();
  const categoryName = name?.toLowerCase() || '';
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const CategoryIcon = categoryIcons[categoryName] || Shield;

  // Fetch events in this category
  useEffect(() => {
    const fetchEvents = async () => {
      try {
        const response = await fetch(
          `${API_BASE_URL}/api/events?limit=100&sort_by=timestamp&sort_order=desc&categories=${categoryName}`
        );
        if (!response.ok) throw new Error('Failed to fetch events');
        const data = await response.json();
        setEvents(data.events || []);
        setLoading(false);
      } catch (err) {
        console.error('Error fetching events:', err);
        setError(err instanceof Error ? err.message : 'Unknown error');
        setLoading(false);
      }
    };

    if (categoryName) {
      fetchEvents();
    }
  }, [categoryName]);

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

          {/* Category Header */}
          <div className={`border-l-4 ${getCategoryColor(categoryName)} pl-6`}>
            <div className="flex items-center gap-3 mb-2">
              <CategoryIcon className={`w-6 h-6 ${getCategoryColor(categoryName)}`} />
              <h1 className="font-display font-black text-3xl text-chalk tracking-tight">
                {categoryName.toUpperCase()}
              </h1>
            </div>
            {categoryDescriptions[categoryName] && (
              <p className="text-sm text-fog font-sans mb-2">
                {categoryDescriptions[categoryName]}
              </p>
            )}
            <p className="text-sm text-smoke font-mono">
              {loading ? 'LOADING EVENTS...' : `${events.length} EVENTS IN THIS CATEGORY`}
            </p>
          </div>

          {/* Events List */}
          <div className="space-y-6">
            {loading ? (
              <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                <div className="text-5xl text-terminal animate-pulse">◉</div>
                <div className="space-y-2">
                  <p className="text-chalk font-mono text-lg font-bold">LOADING EVENTS...</p>
                  <p className="text-fog font-mono text-sm">Searching intelligence database</p>
                </div>
              </div>
            ) : error ? (
              <div className="border-2 border-warning bg-concrete p-16 text-center space-y-6">
                <div className="text-5xl text-warning">⚠</div>
                <div className="space-y-2">
                  <p className="text-chalk font-mono text-lg font-bold">CONNECTION ERROR</p>
                  <p className="text-fog font-mono text-sm">{error}</p>
                </div>
              </div>
            ) : events.length > 0 ? (
              events.map((event, index) => (
                <EventCard key={event.id} event={event} index={index} />
              ))
            ) : (
              <div className="border-2 border-steel bg-concrete p-16 text-center space-y-6">
                <div className="text-5xl text-steel/50">◉</div>
                <div className="space-y-2">
                  <p className="text-chalk font-mono text-lg font-bold">NO EVENTS FOUND</p>
                  <p className="text-fog font-mono text-sm">
                    No {categoryName} events in the database
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
