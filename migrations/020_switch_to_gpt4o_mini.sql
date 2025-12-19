-- Migration 020: Switch from o4-mini to gpt-4o-mini for better JSON compliance
--
-- o4-mini is a reasoning model that:
-- - Does not support JSON mode (response_format)
-- - Often ignores or omits fields like magnitude
-- - Is much slower (30-60s vs 2-5s) and more expensive ($3/1M vs $0.15/1M)
--
-- gpt-4o-mini is optimized for structured extraction:
-- - Enforces JSON schemas via response_format
-- - Fast and reliable
-- - Much cheaper
-- - Better for OSINT event extraction

UPDATE openai_config
SET
  model = 'gpt-4o-mini',
  updated_at = NOW()
WHERE id IS NOT NULL;

-- Verify the update
SELECT
  'Migration 020 applied successfully' as message,
  model as new_model,
  CASE
    WHEN model = 'gpt-4o-mini' THEN '✓ Model updated to gpt-4o-mini'
    ELSE '✗ Model update failed'
  END as status,
  updated_at
FROM openai_config;
