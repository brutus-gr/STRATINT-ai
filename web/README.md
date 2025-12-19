# STRATINT Web Interface

**Brutalist Cyberpunk Intelligence Dashboard**

## Overview

Post-modern intelligence feed interface for visualizing OSINT events from the MCP server. Data-dense, terminal-inspired, unapologetically technical.

**Note:** This is the web visualization layer. The main product is the **MCP Server** at `mcp.stratint.ai` which provides programmatic access to intelligence data for AI assistants.

## Design Philosophy

**"Brutalist Intelligence"**
- Unapologetically technical
- Data density over white space
- Monospace fonts as primary typography
- Dark theme only (no light mode)
- Subtle glitch effects on high-magnitude events
- Terminal/command-line aesthetic (elevated)

**Influences:**
- Brutalist architecture
- Intelligence agency war rooms (CIA/NSA)
- Cyberpunk (Blade Runner 2049, not corny matrix)
- Terminal interfaces

## Tech Stack

- **Framework:** React 18 + TypeScript
- **Build Tool:** Vite
- **Styling:** TailwindCSS (custom brutalist theme)
- **Animations:** Framer Motion
- **Icons:** Lucide React
- **Dates:** date-fns

## Getting Started

### Install Dependencies

```bash
npm install
```

### Development Server

```bash
npm run dev
```

Runs on `http://localhost:5173`

### Build for Production

```bash
npm run build
```

### Preview Production Build

```bash
npm run preview
```

## Features

### ✅ Implemented

**Core UI:**
- Fixed header with live system metrics
- Real-time event feed with terminal-style cards
- Magnitude indicators with color-coded bars
- Category badges with accent colors
- Entity extraction display
- Source attribution

**Filtering:**
- Magnitude range slider
- Confidence threshold slider
- Time range selection (1h, 6h, 24h, 7d, 30d)
- Category multi-select
- Real-time filter application

**Visualizations:**
- Threat map placeholder (ready for Mapbox/Deck.gl)
- Live statistics panel
- Magnitude color-coding
- Confidence indicators

**Effects:**
- Scan line animation (subtle CRT effect)
- Pulse animations on breaking news
- Glitch text effect on logo
- Terminal green selection color
- Brutalist scrollbar styling

### 🔄 Coming Soon

- Command palette (Cmd+K)
- WebSocket real-time updates
- Infinite scroll with lazy loading
- Event detail modal
- Mapbox integration for threat map
- Keyboard shortcuts
- Export options

## Color Palette

```css
/* Base - Brutalist Dark */
--void:       #0a0a0a  /* Background */
--concrete:   #1a1a1a  /* Elevated surfaces */
--steel:      #2a2a2a  /* Borders */
--chalk:      #e0e0e0  /* Primary text */

/* Threat Levels */
--threat-critical:  #ff0844  /* 9.0+ magnitude */
--threat-high:      #ff6b35  /* 7.0-8.9 */
--threat-medium:    #ffbe0b  /* 5.0-6.9 */
--threat-low:       #4ecdc4  /* 3.0-4.9 */

/* Accents */
--terminal:  #00ff41  /* Matrix green (sparingly) */
--electric:  #0066ff  /* Links */
--cyber:     #ff0080  /* Rare accents */
```

## Typography

**Fonts:**
- **Primary:** JetBrains Mono (monospace for data)
- **Secondary:** Inter (sans-serif for readability)
- **Display:** Space Grotesk (brutalist headers)

## Project Structure

```
web/
├── src/
│   ├── components/        # React components
│   │   ├── Header.tsx          # Top metrics bar
│   │   ├── EventCard.tsx       # Terminal-style event cards
│   │   ├── FilterPanel.tsx     # Sidebar filters
│   │   ├── ThreatMap.tsx       # Geographic visualization
│   │   └── StatsPanel.tsx      # Live statistics
│   ├── types.ts           # TypeScript interfaces
│   ├── App.tsx            # Main application
│   ├── index.css          # Global styles + Tailwind
│   └── main.tsx           # React entry point
├── tailwind.config.js     # Brutalist color theme
└── package.json
```

## Component Overview

### Header
Fixed top bar with:
- Glitching logo
- Live system metrics (uptime, events, sources, latency)
- Pulsing "LIVE" indicator

### EventCard
Terminal-style event display with:
- Category badge
- Magnitude/confidence scores
- UTC timestamp
- Animated magnitude bar
- Entity tags
- Source count
- Hover effects

### FilterPanel
Sticky sidebar with:
- Magnitude range slider
- Confidence threshold
- Time range buttons
- Category checkboxes
- Active filter summary

### ThreatMap
Geographic visualization (placeholder):
- Grid overlay (cyberpunk aesthetic)
- Color-coded location pins
- Magnitude legend
- Ready for Mapbox GL integration

### StatsPanel
Real-time metrics:
- Total events with trend
- Average confidence
- Enrichment rate
- Animated progress bars

## Environment Variables

Create `.env` file:

```bash
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws
VITE_MAPBOX_TOKEN=pk.eyJ1...  # Optional for map
```

## API Integration

Currently using mock data. To integrate with backend:

```typescript
// In App.tsx or custom hook
const fetchEvents = async (filters: EventFilters) => {
  const params = new URLSearchParams();
  if (filters.minMagnitude) params.append('min_magnitude', filters.minMagnitude.toString());
  if (filters.categories) params.append('categories', filters.categories.join(','));
  
  const response = await fetch(`${import.meta.env.VITE_API_URL}/api/events?${params}`);
  return response.json();
};
```

## Deployment

### Docker

```dockerfile
FROM node:20-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
```

### Google Cloud Run

```bash
gcloud builds submit --tag gcr.io/PROJECT_ID/stratint-web
gcloud run deploy stratint-web \
  --image gcr.io/PROJECT_ID/stratint-web \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

## Design Documentation

See `/docs/FRONTEND_DESIGN.md` for complete design system specifications.
