-- First, check if the column exists
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name = 'strategies'
AND column_name = 'forecast_history_count';

-- If it doesn't exist, add it
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies'
        AND column_name = 'forecast_history_count'
    ) THEN
        ALTER TABLE strategies ADD COLUMN forecast_history_count INTEGER NOT NULL DEFAULT 1;
        RAISE NOTICE 'Added forecast_history_count column';
    ELSE
        RAISE NOTICE 'Column forecast_history_count already exists';
    END IF;
END $$;

-- Update any null or invalid values
UPDATE strategies SET forecast_history_count = 1
WHERE forecast_history_count IS NULL OR forecast_history_count < 1;

-- Verify the fix
SELECT id, name, forecast_history_count FROM strategies;
