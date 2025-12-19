-- Add GLD (Gold) forecasts for normalized distribution analysis
-- These forecasts ask about probability of GLD being up/down by various thresholds
-- Mirrors the structure used for IBIT/SPY forecasts

-- Insert GLD forecasts with various threshold levels
INSERT INTO forecasts (id, name, proposition, confidence_interval, categories, headline_count, threshold_percent, threshold_direction, threshold_operator, active, created_at, updated_at)
VALUES
    -- Upside forecasts
    (
        'gld-up-5-gte',
        'GLD >= +5%',
        'What is the probability that GLD (gold ETF) will be trading at least 5% higher than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        5.0,
        'up',
        '>=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-up-10-gte',
        'GLD >= +10%',
        'What is the probability that GLD (gold ETF) will be trading at least 10% higher than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        10.0,
        'up',
        '>=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-up-15-gte',
        'GLD >= +15%',
        'What is the probability that GLD (gold ETF) will be trading at least 15% higher than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        15.0,
        'up',
        '>=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-up-20-gte',
        'GLD >= +20%',
        'What is the probability that GLD (gold ETF) will be trading at least 20% higher than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        20.0,
        'up',
        '>=',
        true,
        NOW(),
        NOW()
    ),
    -- Downside forecasts
    (
        'gld-down-5-lte',
        'GLD <= -5%',
        'What is the probability that GLD (gold ETF) will be trading at least 5% lower than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        5.0,
        'down',
        '<=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-down-10-lte',
        'GLD <= -10%',
        'What is the probability that GLD (gold ETF) will be trading at least 10% lower than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        10.0,
        'down',
        '<=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-down-15-lte',
        'GLD <= -15%',
        'What is the probability that GLD (gold ETF) will be trading at least 15% lower than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        15.0,
        'down',
        '<=',
        true,
        NOW(),
        NOW()
    ),
    (
        'gld-down-20-lte',
        'GLD <= -20%',
        'What is the probability that GLD (gold ETF) will be trading at least 20% lower than today by September 18, 2026?',
        0.95,
        ARRAY['economic', 'geopolitics']::TEXT[],
        500,
        20.0,
        'down',
        '<=',
        true,
        NOW(),
        NOW()
    )
ON CONFLICT (id) DO NOTHING;

-- Add context URL for GLD forecasts to pull current price and options data
-- NOTE: Update this URL to match your deployment
UPDATE forecasts
SET context_urls = ARRAY['https://YOUR-SERVICE-URL/api/market/gld-risk-analysis']::TEXT[]
WHERE id LIKE 'gld-%';

COMMENT ON TABLE forecasts IS 'Added GLD forecasts for gold price probability distribution analysis';
