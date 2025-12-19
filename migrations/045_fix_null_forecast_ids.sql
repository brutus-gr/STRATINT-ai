-- Fix null forecast_ids in existing strategies
-- Migration 044 created the column with DEFAULT '{}' but existing rows may still have null

UPDATE strategies
SET forecast_ids = '{}'
WHERE forecast_ids IS NULL;
