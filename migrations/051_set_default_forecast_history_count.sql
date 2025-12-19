-- Ensure all existing strategies have forecast_history_count set to 1
UPDATE strategies
SET forecast_history_count = 1
WHERE forecast_history_count IS NULL OR forecast_history_count = 0;

-- Verify the update
SELECT COUNT(*) as total_strategies,
       COUNT(CASE WHEN forecast_history_count IS NOT NULL THEN 1 END) as with_history_count,
       COUNT(CASE WHEN forecast_history_count = 1 THEN 1 END) as defaulted_to_one
FROM strategies;
