-- Backfill created_at and updated_at for existing events
-- Set created_at and updated_at to the event's timestamp field (when the event actually occurred)
-- Handle both NULL and zero timestamp values
UPDATE events
SET
    created_at = timestamp,
    updated_at = timestamp
WHERE
    created_at IS NULL
    OR updated_at IS NULL
    OR created_at < '1970-01-01'::timestamp
    OR updated_at < '1970-01-01'::timestamp;
