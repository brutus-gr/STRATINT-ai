-- Check strategies table schema
\d strategies

-- Check if forecast_history_count column exists and its values
SELECT column_name, data_type, column_default, is_nullable
FROM information_schema.columns
WHERE table_name = 'strategies'
AND column_name = 'forecast_history_count';

-- Check actual strategy data
SELECT id, name, forecast_history_count FROM strategies;
