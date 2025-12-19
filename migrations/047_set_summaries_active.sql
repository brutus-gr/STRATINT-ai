-- Migration 047: Set existing summaries to active
-- This fixes summaries created before the frontend was sending active: true

UPDATE summaries SET active = true WHERE active = false;
