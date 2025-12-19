-- Add max_tweet_age_hours column to twitter_config table
-- This column controls the maximum age of events (in hours) that can be auto-tweeted

ALTER TABLE twitter_config
ADD COLUMN IF NOT EXISTS max_tweet_age_hours INTEGER NOT NULL DEFAULT 6 CHECK (max_tweet_age_hours > 0);

-- Update the existing row to set the default value (if it exists)
UPDATE twitter_config
SET max_tweet_age_hours = 6
WHERE max_tweet_age_hours IS NULL OR max_tweet_age_hours = 0;
