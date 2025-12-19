#!/bin/bash

# Script to add New York Times RSS feeds to OSINTMCP
# Usage: ./scripts/add_nyt_feeds.sh [API_URL] [ADMIN_PASSWORD]

API_URL="${1:-http://localhost:8080}"
ADMIN_PASSWORD="${2}"

if [ -z "$ADMIN_PASSWORD" ]; then
    echo "Error: Admin password required"
    echo "Usage: ./scripts/add_nyt_feeds.sh [API_URL] [ADMIN_PASSWORD]"
    exit 1
fi

echo "Adding New York Times RSS feeds to $API_URL"

# Login to get JWT token
echo "Logging in..."
TOKEN=$(curl -s -X POST "$API_URL/api/admin/login" \
    -H "Content-Type: application/json" \
    -d "{\"password\":\"$ADMIN_PASSWORD\"}" | jq -r '.token')

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "Error: Failed to authenticate. Check your admin password."
    exit 1
fi

echo "Authenticated successfully"

# Array of NYT RSS feeds
declare -a feeds=(
    "https://rss.nytimes.com/services/xml/rss/nyt/World.xml|New York Times - World"
    "https://rss.nytimes.com/services/xml/rss/nyt/US.xml|New York Times - US"
    "https://rss.nytimes.com/services/xml/rss/nyt/Politics.xml|New York Times - Politics"
    "https://rss.nytimes.com/services/xml/rss/nyt/Business.xml|New York Times - Business"
    "https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml|New York Times - Technology"
    "https://rss.nytimes.com/services/xml/rss/nyt/Science.xml|New York Times - Science"
    "https://rss.nytimes.com/services/xml/rss/nyt/Health.xml|New York Times - Health"
    "https://rss.nytimes.com/services/xml/rss/nyt/Climate.xml|New York Times - Climate"
    "https://rss.nytimes.com/services/xml/rss/nyt/AsiaPacific.xml|New York Times - Asia Pacific"
    "https://rss.nytimes.com/services/xml/rss/nyt/Europe.xml|New York Times - Europe"
    "https://rss.nytimes.com/services/xml/rss/nyt/MiddleEast.xml|New York Times - Middle East"
    "https://rss.nytimes.com/services/xml/rss/nyt/Africa.xml|New York Times - Africa"
    "https://rss.nytimes.com/services/xml/rss/nyt/Americas.xml|New York Times - Americas"
)

# Add each feed
count=0
for feed_info in "${feeds[@]}"; do
    IFS='|' read -r url name <<< "$feed_info"

    echo "Adding: $name"

    response=$(curl -s -X POST "$API_URL/api/admin/sources" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"platform\": \"rss\",
            \"account_identifier\": \"$url\",
            \"display_name\": \"$name\",
            \"enabled\": true,
            \"fetch_interval_minutes\": 15
        }")

    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        error=$(echo "$response" | jq -r '.error')
        if [[ "$error" == *"already exists"* ]]; then
            echo "  ⚠️  Already exists (skipping)"
        else
            echo "  ❌ Error: $error"
        fi
    else
        echo "  ✅ Added successfully"
        ((count++))
    fi
done

echo ""
echo "✅ Successfully added $count NYT RSS feeds"
echo "📡 Feeds will be fetched every 15 minutes"
