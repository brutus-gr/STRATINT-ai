-- Add text fields for location data (country, city, region)
-- The existing location GEOGRAPHY field stores lat/lon coordinates

ALTER TABLE events ADD COLUMN IF NOT EXISTS location_country VARCHAR(255);
ALTER TABLE events ADD COLUMN IF NOT EXISTS location_city VARCHAR(255);
ALTER TABLE events ADD COLUMN IF NOT EXISTS location_region VARCHAR(255);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_events_location_country ON events(location_country);
CREATE INDEX IF NOT EXISTS idx_events_location_city ON events(location_city);

-- Comment for documentation
COMMENT ON COLUMN events.location IS 'PostGIS geography point storing latitude/longitude coordinates (WGS84)';
COMMENT ON COLUMN events.location_country IS 'Country name extracted from event content';
COMMENT ON COLUMN events.location_city IS 'City name extracted from event content';
COMMENT ON COLUMN events.location_region IS 'Region/state/province name extracted from event content';
