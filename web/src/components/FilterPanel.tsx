import { Filter, X } from 'lucide-react';
import type { Category, EventFilters } from '../types';

interface FilterPanelProps {
  filters: EventFilters;
  onFiltersChange: (filters: EventFilters) => void;
}

export function FilterPanel({ filters, onFiltersChange }: FilterPanelProps) {
  const categories: Category[] = [
    'military',
    'cyber',
    'geopolitics',
    'terrorism',
    'disaster',
    'economic',
    'diplomacy',
    'intelligence',
    'humanitarian',
  ];

  const toggleCategory = (category: Category) => {
    const current = filters.categories || [];
    const updated = current.includes(category)
      ? current.filter((c) => c !== category)
      : [...current, category];
    onFiltersChange({ ...filters, categories: updated });
  };

  const clearFilters = () => {
    onFiltersChange({});
  };

  const hasActiveFilters =
    (filters.categories && filters.categories.length > 0) ||
    filters.minMagnitude !== undefined ||
    filters.minConfidence !== undefined;

  return (
    <div className="border-2 border-steel bg-concrete">
      {/* Header */}
      <div className="px-5 py-3 border-b-2 border-steel bg-void/50 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Filter className="w-5 h-5 text-terminal" />
          <h3 className="font-display font-black text-base text-chalk tracking-tight">FILTERS</h3>
        </div>
        {hasActiveFilters && (
          <button
            onClick={clearFilters}
            className="text-xs font-mono text-smoke hover:text-chalk transition-colors flex items-center gap-1 font-medium"
          >
            <X className="w-3 h-3" />
            CLEAR
          </button>
        )}
      </div>

      {/* Content */}
      <div className="p-5 space-y-6">

      {/* Magnitude Range */}
      <div className="space-y-2">
        <label className="block text-xs font-mono text-smoke">MIN MAGNITUDE</label>
        <input
          type="range"
          min="0"
          max="10"
          step="0.5"
          value={filters.minMagnitude || 0}
          onChange={(e) => onFiltersChange({ ...filters, minMagnitude: parseFloat(e.target.value) })}
          className="w-full"
        />
        <div className="flex justify-between text-xs font-mono">
          <span className="text-terminal font-bold">{filters.minMagnitude?.toFixed(1) || '0.0'}</span>
          <span className="text-smoke">10.0</span>
        </div>
      </div>

      {/* Confidence Range */}
      <div className="space-y-2">
        <label className="block text-xs font-mono text-smoke">MIN CONFIDENCE</label>
        <input
          type="range"
          min="0"
          max="1"
          step="0.05"
          value={filters.minConfidence || 0}
          onChange={(e) => onFiltersChange({ ...filters, minConfidence: parseFloat(e.target.value) })}
          className="w-full"
        />
        <div className="flex justify-between text-xs font-mono">
          <span className="text-terminal font-bold">{filters.minConfidence?.toFixed(2) || '0.00'}</span>
          <span className="text-smoke">1.00</span>
        </div>
      </div>

      {/* Categories */}
      <div className="space-y-2">
        <label className="block text-xs font-mono text-smoke">CATEGORIES</label>
        <div className="space-y-1">
          {categories.map((category) => {
            const isActive = filters.categories?.includes(category);
            return (
              <button
                key={category}
                onClick={() => toggleCategory(category)}
                className={`w-full px-3 py-1.5 text-xs font-mono border-2 text-left transition-all ${
                  isActive
                    ? 'border-terminal bg-terminal/20 text-terminal'
                    : 'border-steel bg-void text-fog hover:border-iron hover:text-chalk'
                }`}
              >
                [{category.toUpperCase()}]
              </button>
            );
          })}
        </div>
      </div>

      {/* Stats Display */}
      <div className="pt-4 border-t border-steel space-y-2">
        <h3 className="text-xs font-mono text-smoke">ACTIVE FILTERS</h3>
        <div className="text-xs font-mono text-chalk space-y-1">
          {filters.minMagnitude && (
            <div className="flex justify-between">
              <span className="text-smoke">Magnitude:</span>
              <span>{filters.minMagnitude.toFixed(1)}+</span>
            </div>
          )}
          {filters.minConfidence && (
            <div className="flex justify-between">
              <span className="text-smoke">Confidence:</span>
              <span>{filters.minConfidence.toFixed(2)}+</span>
            </div>
          )}
          {filters.categories && filters.categories.length > 0 && (
            <div className="flex justify-between">
              <span className="text-smoke">Categories:</span>
              <span>{filters.categories.length}</span>
            </div>
          )}
        </div>
      </div>
      </div>
    </div>
  );
}
