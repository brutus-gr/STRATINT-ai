-- Threshold configuration table
CREATE TABLE IF NOT EXISTS threshold_config (
    id SERIAL PRIMARY KEY,
    min_confidence DECIMAL(3,2) NOT NULL DEFAULT 0.10,
    min_magnitude DECIMAL(3,1) NOT NULL DEFAULT 0.0,
    max_source_age_hours INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default values
INSERT INTO threshold_config (min_confidence, min_magnitude, max_source_age_hours)
VALUES (0.10, 0.0, 0);
