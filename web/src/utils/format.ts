/**
 * Format units for display
 * Converts snake_case to Title Case
 */
export function formatUnits(units: string): string {
  if (!units) return '';

  // Special cases
  const specialCases: Record<string, string> = {
    'percent_change': 'Percent Change',
    'usd': 'USD',
    'eur': 'EUR',
    'gbp': 'GBP',
    'btc': 'BTC',
    'eth': 'ETH',
  };

  const lowerUnits = units.toLowerCase();
  if (specialCases[lowerUnits]) {
    return specialCases[lowerUnits];
  }

  // Convert snake_case to Title Case
  return units
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ');
}

/**
 * Format schedule interval for display
 * Converts minutes to human-readable format
 */
export function formatScheduleInterval(minutes: number): string {
  if (!minutes) return '';

  if (minutes < 60) {
    return `Updated every ${minutes} minute${minutes !== 1 ? 's' : ''}`;
  }

  const hours = minutes / 60;
  if (hours < 24) {
    const hourInt = Math.floor(hours);
    if (hours === hourInt) {
      return `Updated every ${hourInt} hour${hourInt !== 1 ? 's' : ''}`;
    }
    return `Updated every ${minutes} minutes`;
  }

  const days = hours / 24;
  const dayInt = Math.floor(days);
  if (days === dayInt) {
    return `Updated every ${dayInt} day${dayInt !== 1 ? 's' : ''}`;
  }

  return `Updated every ${hours} hours`;
}
