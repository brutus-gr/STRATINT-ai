# STRATINT Data Sources Research

This document outlines available APIs, scraping approaches, rate limits, and terms of service for supported OSINT data sources.

## 1. Twitter / X

### API Access

**Official API Options:**
- **Free Tier**: 1,500 tweets/month read limit (severely limited)
- **Basic ($100/month)**: 10,000 tweets/month, 3,000 tweets/month write
- **Pro ($5,000/month)**: 1M tweets/month, full archive access
- **Enterprise**: Custom pricing, full access

**Authentication:** OAuth 2.0 Bearer Token

**Endpoints:**
- `GET /2/tweets/search/recent` - Search tweets from last 7 days
- `GET /2/tweets/search/all` - Full archive search (Pro+)
- `GET /2/users/:id/tweets` - User timeline
- `GET /2/tweets` - Tweet lookup by IDs

### Rate Limits (Free/Basic Tier)
- 15 requests per 15 minutes for search endpoints
- 180 requests per 15 minutes for timeline endpoints
- Monthly tweet cap applies

### Alternative Approaches
1. **Nitter instances** - Twitter frontend proxy (unofficial)
   - No API keys required
   - Rate limits vary by instance
   - Legal gray area, may violate ToS

2. **RSS Feeds** - Limited to user timelines
   - Use services like Nitter RSS or TweetDeck RSS
   - No authentication required

3. **Webscraping** - Not recommended
   - Violates Twitter ToS
   - Aggressive bot detection
   - Legal risks

### Terms of Service
- Must not exceed rate limits
- Cannot store tweets indefinitely without explicit permission
- Must respect user privacy settings
- Compliance with DMCA takedown requests

### Recommended Strategy
- Start with Free tier for prototype
- Implement aggressive caching
- Focus on high-value accounts (official government, verified journalists)
- Upgrade to Basic ($100/mo) when needed

---

## 2. Telegram

### API Access

**Official Telegram API:**
- Free, no rate limits (within reasonable use)
- MTProto protocol or HTTP Bot API
- Full access to public channels and groups

**Authentication:** 
- App API ID/Hash (register at https://my.telegram.org)
- Phone number verification for user accounts
- Bot tokens for bot accounts

**Methods:**
- `messages.getHistory` - Get channel/chat history
- `channels.getMessages` - Get specific messages
- `messages.search` - Search messages in channels
- `updates` - Real-time message streaming

### Rate Limits
- No official published limits for reasonable use
- Flood prevention: ~20 requests/second max
- Long polling for updates (no additional limits)

### Libraries
- **telethon** (Python) - Full MTProto implementation
- **tdlib** (C++) - Official library with bindings
- **gramjs** (JavaScript/TypeScript) - MTProto implementation

### Terms of Service
- Must respect user privacy
- Cannot spam or scrape private chats
- Public channels are fair game for monitoring
- Must comply with local laws

### Recommended Strategy
- Use official Bot API for public channels
- Focus on known OSINT/news channels
- Implement real-time streaming with `updates`
- Store channel metadata for attribution

---

## 3. Reddit

### API Access

**Official Reddit API:**
- Free tier: 100 requests/minute
- OAuth 2.0 authentication
- Rate limits enforced per client ID

**Authentication:**
- Client ID/Secret (register at https://www.reddit.com/prefs/apps)
- OAuth2 with user-agent requirement

**Endpoints:**
- `GET /r/{subreddit}/new` - New posts in subreddit
- `GET /r/{subreddit}/hot` - Hot posts
- `GET /r/{subreddit}/comments/{article}` - Post comments
- `GET /search` - Search across Reddit

### Rate Limits
- 100 requests per minute per client ID
- 600 requests per 10 minutes
- Must include descriptive User-Agent header

### Terms of Service
- Must use OAuth for authenticated requests
- Cannot circumvent rate limits
- Must respect robots.txt
- Cannot scrape for commercial resale

### Alternative Approaches
1. **Pushshift API** - Historical Reddit data
   - Currently restricted/deprecated
   - Limited to academic use
   
2. **RSS Feeds** - Limited data
   - `/r/{subreddit}/.rss` format
   - No authentication required
   - 100 posts max per request

### Recommended Strategy
- Use official API with OAuth
- Focus on specific subreddits (r/worldnews, r/geopolitics, etc.)
- Implement exponential backoff for rate limits
- Cache heavily to stay within 100 req/min

---

## 4. 4chan

### API Access

**Public Read-Only API:**
- No authentication required
- Free, public access
- JSON responses

**Endpoints:**
- `GET /{board}/catalog.json` - Board catalog
- `GET /{board}/thread/{id}.json` - Thread contents
- `GET /{board}/threads.json` - Thread list by page
- `GET /{board}/archive.json` - Archived threads

### Rate Limits
- 1 request per second per IP address
- 10 second cooldown between thread requests
- Enforced with 429 status codes

### Terms of Service
- Must respect rate limits (critical)
- Must include polite User-Agent
- Cannot make automated posts
- Content is public domain

### Special Considerations
- Content can be extremely offensive/illegal
- Requires strong content moderation
- Short thread lifespan (404s when archived)
- Image links expire quickly

### Recommended Strategy
- Monitor specific boards (/pol/, /int/, /news/)
- Poll catalog every 60 seconds
- Archive threads of interest immediately
- Implement content filtering pipeline
- **Strong moderation required**

---

## 5. Godlike Productions (GLP)

### API Access
**No official API available**

### Scraping Approach
- HTML parsing required
- Cloudflare protection in place
- User registration recommended

**Key Pages:**
- `https://www.godlikeproductions.com/` - Main forum
- Thread URLs: `/forum1/message{id}/pg{page}`
- RSS feeds available for some sections

### Rate Limits
- No official limits
- Cloudflare rate limiting
- Recommend 1 request per 5 seconds

### Terms of Service
- Check robots.txt
- Respect Cloudflare protections
- Private forum sections require login
- Content copyright varies by poster

### Recommended Strategy
- Use RSS feeds where available
- Implement polite scraping (5+ second delays)
- Focus on "Breaking News" section
- Use Cloudflare-aware HTTP client
- **Lowest priority source** due to reliability concerns

---

## 6. Government Alerts & Official Sources

### United States
- **FEMA API**: https://www.fema.gov/about/openfema/api
  - Free, no authentication for public data
  - Disaster declarations, emergency alerts
  
- **CDC**: Various RSS feeds
  - Health alerts and outbreak notifications

- **State Department**: https://www.state.gov/rss-feeds/
  - Travel advisories, press releases

### International
- **GDACS** (Global Disaster Alert): https://www.gdacs.org/xml/rss.xml
  - Free XML/RSS feeds
  - Natural disasters, humanitarian crises

- **ReliefWeb API**: https://api.reliefweb.int/
  - Free humanitarian news aggregation
  - No authentication required

### NATO/Military
- Most require manual monitoring (no APIs)
- RSS feeds for press releases
- Twitter accounts for official statements

### Recommended Strategy
- Poll RSS feeds every 15-30 minutes
- Parse structured XML/JSON where available
- High credibility score for government sources
- Prioritize for breaking alerts

---

## 7. News Media APIs

### News API (NewsAPI.org)
- **Free Tier**: 100 requests/day
- **Paid**: $449/month for commercial use
- 80,000+ sources indexed
- Breaking news alerts

### Alternative News APIs
- **Bing News Search**: Part of Azure Cognitive Services
- **GNews API**: Free tier available
- **Event Registry**: Academic/commercial licenses

### Recommended Strategy
- Use as secondary source for verification
- Cross-reference with social media
- Higher credibility weighting
- Implement duplicate detection across sources

---

## Rate Limiting Strategy (Cross-Platform)

### Priority System
1. **High Priority** (every 1-5 minutes)
   - Government alerts (RSS)
   - Twitter verified accounts
   - Telegram breaking news channels
   
2. **Medium Priority** (every 15-30 minutes)
   - Reddit hot posts
   - News API updates
   - Twitter search queries
   
3. **Low Priority** (every 1-2 hours)
   - 4chan catalog polls
   - GLP scraping
   - Historical data backfills

### Global Rate Limit Management
```
Total budget per minute:
- Twitter: 15 requests / 15min = 1 req/min
- Reddit: 100 requests / min
- Telegram: ~20 req/sec (limited by flood protection)
- 4chan: 1 req/sec = 60 req/min
- GLP: ~12 req/min (5 sec delay)
```

**Recommended Concurrent Fetches**: 3-5 connectors max

---

## Legal & Ethical Considerations

### Must-Have Protections
1. **Content Moderation**
   - Filter illegal content (CSAM, violence)
   - Flag extreme content for review
   - Respect DMCA takedowns

2. **Privacy Compliance**
   - GDPR considerations for EU users
   - Right to be forgotten (deletion requests)
   - Anonymous/pseudonymous content handling

3. **Attribution**
   - Always cite original sources
   - Preserve author attribution
   - Link back to original content

4. **Terms of Service Compliance**
   - Review ToS quarterly
   - Implement rate limit respect
   - No ToS violations in production

### Red Flags to Avoid
- Scraping private/protected content
- Bypassing authentication mechanisms
- Exceeding published rate limits
- Reselling raw data commercially
- Automated engagement (likes, shares, posts)

---

## Implementation Roadmap

### Phase 1: Foundation (Current)
- ✅ Connector interfaces defined
- ✅ Retry/backoff logic
- ✅ Deduplication system
- ✅ Storage abstractions

### Phase 2: Initial Connectors
- [ ] Telegram connector (easiest, most reliable)
- [ ] Twitter Free API connector
- [ ] Government RSS aggregator
- [ ] Reddit connector

### Phase 3: Expansion
- [ ] 4chan connector (with moderation)
- [ ] News API integration
- [ ] GLP scraper (optional)

### Phase 4: Optimization
- [ ] Intelligent polling intervals
- [ ] Source credibility scoring
- [ ] Cross-platform deduplication
- [ ] Real-time streaming where available

---

## Cost Estimates

### Minimal Viable Product
- Twitter Free: $0/month (limited)
- Telegram: $0/month
- Reddit: $0/month
- 4chan: $0/month
- Government sources: $0/month
- **Total: $0/month** (highly limited)

### Production Deployment
- Twitter Basic: $100/month
- News API Developer: $449/month (optional)
- Hosting: ~$50/month
- **Total: $150-600/month** depending on features

### Scale Deployment
- Twitter Pro: $5,000/month (for comprehensive coverage)
- Multiple News APIs: $1,000+/month
- Database/Storage: $200/month
- **Total: $6,000+/month**

---

## Next Steps

1. Implement Telegram connector first (easiest, most reliable)
2. Add Twitter Free API support for prototype
3. Aggregate government RSS feeds
4. Add Reddit connector for community intelligence
5. Evaluate need for 4chan/GLP based on user feedback
