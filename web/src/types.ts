// OSINT Event types matching backend models

export interface Event {
  id: string;
  timestamp: string;
  title: string;
  summary: string;
  magnitude: number;
  confidence: Confidence;
  category: Category;
  entities: Entity[];
  sources: Source[];
  tags: string[];
  location?: Location;
  status: EventStatus;
}

export interface Confidence {
  score: number;
  level: 'low' | 'medium' | 'high' | 'verified';
  reasoning: string;
  source_count: number;
}

export type Category =
  | 'geopolitics'
  | 'military'
  | 'economic'
  | 'cyber'
  | 'disaster'
  | 'terrorism'
  | 'diplomacy'
  | 'intelligence'
  | 'humanitarian'
  | 'other';

export type EventStatus = 'pending' | 'enriched' | 'published' | 'archived' | 'rejected';

export interface Entity {
  id: string;
  type: EntityType;
  name: string;
  normalized_name: string;
  confidence: number;
}

export type EntityType =
  | 'country'
  | 'city'
  | 'region'
  | 'person'
  | 'organization'
  | 'military_unit'
  | 'vessel'
  | 'weapon_system'
  | 'facility'
  | 'event'
  | 'other';

export interface Source {
  id: string;
  type: SourceType;
  url: string;
  title?: string;
  author?: string;
  published_at: string;
  retrieved_at: string;
  credibility: number;
}

export type SourceType =
  | 'twitter'
  | 'telegram'
  | 'glp'
  | 'government'
  | 'news_media'
  | 'blog'
  | 'other';

export interface Location {
  latitude: number;
  longitude: number;
  country?: string;
  city?: string;
  region?: string;
}

export interface EventResponse {
  events: Event[];
  page: number;
  limit: number;
  total: number;
  has_more: boolean;
  query?: string;
}

// UI-specific types
export interface EventFilters {
  categories?: Category[];
  minMagnitude?: number;
  minConfidence?: number;
  search?: string;
  timeRange?: '1h' | '6h' | '24h' | '7d' | '30d';
}

export interface SystemStats {
  uptime: number;
  events_total: number;
  sources_total: number;
  latency_ms: number;
  is_live: boolean;
  avg_confidence?: number;
  enrichment_rate?: number;
}
