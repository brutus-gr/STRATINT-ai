-- Twitter Configuration Table
-- Stores configuration for Twitter/X API integration and auto-posting

CREATE TABLE IF NOT EXISTS twitter_config (
    id SERIAL PRIMARY KEY,

    -- Twitter API Credentials
    api_key TEXT NOT NULL DEFAULT '',
    api_secret TEXT NOT NULL DEFAULT '',
    access_token TEXT NOT NULL DEFAULT '',
    access_token_secret TEXT NOT NULL DEFAULT '',
    bearer_token TEXT NOT NULL DEFAULT '',

    -- OpenAI Prompt for Tweet Generation
    tweet_generation_prompt TEXT NOT NULL,

    -- Auto-posting Thresholds
    min_magnitude_for_tweet DECIMAL(3,1) NOT NULL DEFAULT 7.0 CHECK (min_magnitude_for_tweet >= 0 AND min_magnitude_for_tweet <= 10),
    min_confidence_for_tweet DECIMAL(3,2) NOT NULL DEFAULT 0.80 CHECK (min_confidence_for_tweet >= 0 AND min_confidence_for_tweet <= 1),

    -- Categories to auto-tweet (JSON array)
    enabled_categories JSONB NOT NULL DEFAULT '["military", "geopolitics", "cyber", "terrorism", "disaster"]'::jsonb,

    -- Feature Toggle
    enabled BOOLEAN NOT NULL DEFAULT false,

    -- Timestamps
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Only allow one config row
CREATE UNIQUE INDEX twitter_config_single_row ON twitter_config ((id IS NOT NULL));

-- Insert default configuration
INSERT INTO twitter_config (
    api_key,
    api_secret,
    access_token,
    access_token_secret,
    bearer_token,
    tweet_generation_prompt,
    min_magnitude_for_tweet,
    min_confidence_for_tweet,
    enabled_categories,
    enabled
) VALUES (
    '',  -- Empty by default, will be set via admin panel
    '',
    '',
    '',
    '',
    'You are an expert at crafting compelling, concise tweets for breaking news and OSINT intelligence updates.

=== TASK ===

Create a tweet based on the following event information:

Title: {{.Title}}
Summary: {{.Summary}}
Category: {{.Category}}
Locations: {{.Locations}}
Magnitude: {{.Magnitude}}/10
Confidence: {{.Confidence}}

Event URL: https://stratint.ai/events/{{.EventID}}

=== TWEET REQUIREMENTS ===

1. Start with flag emoji(s) for location(s) if applicable (e.g., 🇺🇸 🇺🇦 🇷🇺)
2. Start with a siren emoji 🚨 after flags for breaking/urgent events
3. Use "BREAKING:" for truly breaking news (events within last 2 hours)
4. Keep the main content under 200 characters to leave room for link
5. Make it attention-grabbing but factual - NO sensationalism or speculation
6. End with the event link: https://stratint.ai/events/{{.EventID}}
7. Add relevant hashtags (2-3 max) if they add value
8. Use emojis sparingly and only when they add clarity

=== TONE GUIDELINES ===

- Authoritative and factual
- Urgent but not alarmist
- Professional intelligence community tone
- No speculation or editorializing
- Focus on verifiable facts from the summary

=== EXAMPLE OUTPUT ===

Example 1 (Military):
🇺🇦🇷🇺 🚨 BREAKING: Ukrainian forces report downing Russian Su-34 fighter-bomber over Donetsk region. Pilot status unknown. Escalation in aerial combat operations.

https://stratint.ai/events/abc123

#Ukraine #Russia #OSINT

Example 2 (Cyber):
🇺🇸 Major ransomware attack targeting healthcare provider affects 3M+ patient records. FBI investigating suspected APT group. Critical infrastructure alert.

https://stratint.ai/events/xyz789

#Cybersecurity #OSINT

Example 3 (Geopolitics):
🇨🇳🇹🇼 🚨 BREAKING: Chinese military begins live-fire exercises 50km from Taiwan coast. Largest drill in 6 months. Regional tensions escalating.

https://stratint.ai/events/def456

#Taiwan #China #OSINT

=== YOUR OUTPUT ===

Respond with ONLY the tweet text. No explanations, no markdown, no additional commentary. Just the tweet ready to post.

Maximum length: 280 characters (Twitter limit)',
    7.0,
    0.80,
    '["military", "geopolitics", "cyber", "terrorism", "disaster"]'::jsonb,
    false  -- Disabled by default until credentials are configured
) ON CONFLICT DO NOTHING;

-- Table to track posted tweets (prevent duplicates)
CREATE TABLE IF NOT EXISTS posted_tweets (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(50) NOT NULL UNIQUE,
    tweet_id VARCHAR(50) NOT NULL,
    tweet_text TEXT NOT NULL,
    posted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_event
        FOREIGN KEY(event_id)
        REFERENCES events(id)
        ON DELETE CASCADE
);

-- Index for quick lookup
CREATE INDEX idx_posted_tweets_event_id ON posted_tweets(event_id);
CREATE INDEX idx_posted_tweets_posted_at ON posted_tweets(posted_at DESC);
