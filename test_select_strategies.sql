-- Test if we can select all columns (see what the actual error is)
SELECT * FROM strategies LIMIT 1;

-- Test the exact query the Go code uses
SELECT id, name, prompt, investment_symbols, categories, headline_count, iterations, forecast_ids, forecast_history_count, active, public, display_order, schedule_enabled, schedule_interval, last_run_at, next_run_at, created_at, updated_at
FROM strategies
ORDER BY created_at DESC
LIMIT 1;
