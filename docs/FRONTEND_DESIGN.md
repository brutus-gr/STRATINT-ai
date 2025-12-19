# STRATINT Frontend Design System

## Aesthetic: "Brutalist Intelligence"

Post-modern cyberpunk meets intelligence agency war room. Data-dense, unapologetically technical, but refined and intentional.

**Influences:**
- Brutalist architecture (raw concrete, exposed systems)
- Terminal/command-line interfaces (elevated)
- Intelligence agency dashboards (CIA/NSA war rooms)
- Cyberpunk but tasteful (Blade Runner 2049, not Tron)
- Data visualization as art
- Glitch art (subtle, not overdone)

**Anti-influences:**
- Generic dashboards
- Bootstrap blue
- Corny "hacker" green-on-black
- Overuse of animations
- Skeuomorphism

---

## Color Palette

### Base Colors (Dark Mode Only)
```css
--void:       #0a0a0a;  /* Pure black background */
--concrete:   #1a1a1a;  /* Elevated surfaces */
--steel:      #2a2a2a;  /* Borders, dividers */
--iron:       #3a3a3a;  /* Hover states */
--smoke:      #8a8a8a;  /* Muted text */
--fog:        #b0b0b0;  /* Secondary text */
--chalk:      #e0e0e0;  /* Primary text */
--white:      #f5f5f5;  /* Highlights */
```

### Threat Level Colors
```css
--threat-critical:  #ff0844;  /* Hot red */
--threat-high:      #ff6b35;  /* Orange */
--threat-medium:    #ffbe0b;  /* Yellow */
--threat-low:       #4ecdc4;  /* Cyan */
--threat-info:      #45b7d1;  /* Blue */
```

### Accent Colors
```css
--terminal-green:   #00ff41;  /* Matrix green (sparingly) */
--electric-blue:    #0066ff;  /* Links, highlights */
--warning-amber:    #ffaa00;  /* Warnings */
--cyber-magenta:    #ff0080;  /* Rare accents */
```

### Functional Colors
```css
--success:  #00ff41;
--error:    #ff0844;
--warning:  #ffaa00;
--info:     #45b7d1;
```

---

## Typography

### Fonts
```css
/* Primary: Monospace for that "raw data" feel */
--font-mono: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;

/* Secondary: Sans-serif for readability */
--font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;

/* Display: Brutalist all-caps titles */
--font-display: 'Space Grotesk', 'Inter', sans-serif;
```

### Scale
```css
--text-xs:   0.75rem;   /* 12px - Timestamps, metadata */
--text-sm:   0.875rem;  /* 14px - Body text */
--text-base: 1rem;      /* 16px - Default */
--text-lg:   1.125rem;  /* 18px - Subheadings */
--text-xl:   1.25rem;   /* 20px - Card titles */
--text-2xl:  1.5rem;    /* 24px - Section headers */
--text-3xl:  2rem;      /* 32px - Page titles */
--text-4xl:  2.5rem;    /* 40px - Hero text */
```

### Weights
```css
--font-normal:  400;
--font-medium:  500;
--font-bold:    700;
--font-black:   900;  /* For all-caps display text */
```

---

## Layout System

### Grid
- **12-column grid** for flexibility
- **Dense layouts** - maximize information density
- **Terminal-inspired rows** - events stack like log output
- **Split-screen** - main feed + sidebar stats

### Spacing Scale (8px base)
```css
--space-0:   0;
--space-1:   0.25rem;   /* 4px */
--space-2:   0.5rem;    /* 8px */
--space-3:   0.75rem;   /* 12px */
--space-4:   1rem;      /* 16px */
--space-6:   1.5rem;    /* 24px */
--space-8:   2rem;      /* 32px */
--space-12:  3rem;      /* 48px */
--space-16:  4rem;      /* 64px */
```

### Breakpoints
```css
--sm:  640px;   /* Mobile */
--md:  768px;   /* Tablet */
--lg:  1024px;  /* Desktop */
--xl:  1280px;  /* Large desktop */
--2xl: 1536px;  /* Ultra-wide */
```

---

## Components

### 1. Event Card (Terminal Style)
```
┌─────────────────────────────────────────────────────────────┐
│ [MILITARY]  MAG: 8.7  CONF: 0.89  UTC: 2024-01-15 14:23:45 │
├─────────────────────────────────────────────────────────────┤
│ > MILITARY EXERCISES ANNOUNCED NEAR BORDER                  │
│                                                             │
│   Joint military drills involving United States and allied │
│   forces scheduled for next week. High-level diplomatic    │
│   talks to precede exercises.                              │
│                                                             │
│   ENTITIES: United States, NATO, Russia                    │
│   SOURCES: 3 verified  [EXPAND]                            │
│                                                             │
│   [VIEW DETAILS] [MAP] [SOURCES] [ENTITIES]               │
└─────────────────────────────────────────────────────────────┘
```

**Styling:**
- Monospace font
- ASCII box-drawing characters
- Color-coded magnitude bar
- Pulsing indicator for breaking news
- Hover: Subtle glitch effect

### 2. Magnitude Indicator
```
╔═══════════════════════════════════════════════════════╗
║ MAG: 8.7 ████████▓░                                  ║
╚═══════════════════════════════════════════════════════╝
```

**Colors by magnitude:**
- 9.0-10.0: `--threat-critical` (red)
- 7.0-8.9: `--threat-high` (orange)
- 5.0-6.9: `--threat-medium` (yellow)
- 3.0-4.9: `--threat-low` (cyan)
- 0.0-2.9: `--threat-info` (blue)

### 3. Command Palette (Cmd+K)
```
┌────────────────────────────────────────┐
│ > search_events                       │
│ ─────────────────────────────────────│
│   filter:category=military            │
│   filter:magnitude>7.0                │
│   filter:last_24h                     │
│   export:json                         │
│   help:commands                       │
└────────────────────────────────────────┘
```

**Features:**
- Fuzzy search
- Keyboard-first navigation
- Command autocomplete
- Quick filters
- Export options

### 4. Threat Map (3D Globe or Flat)
- **Dark base map** (no Google Maps colors)
- **Glowing pins** for event locations
- **Pulse animation** for recent events
- **Color-coded** by magnitude
- **Hexagonal overlay** (cyberpunk aesthetic)
- **Click to filter** by region

### 5. Live Metrics Bar
```
[SYSTEM STATUS]  UPTIME: 72:45:12  |  EVENTS: 12,847  |  SOURCES: 3,291  |  LATENCY: 45ms  [◉ LIVE]
```

**Fixed top bar:**
- Real-time metrics
- Pulsing "LIVE" indicator
- System health status
- Connection status

### 6. Category Badges
```
[MILITARY]  [CYBER]  [GEOPOLITICS]
```

**Styling:**
- Monospace text
- Square brackets
- Color-coded by category
- All-caps
- Clickable filters

---

## Animations & Effects

### 1. Glitch Effect (Subtle)
**When:** New event arrives, magnitude >8.0
**How:** RGB channel split, 150ms duration
```css
.glitch {
  animation: glitch 150ms ease-in-out;
}

@keyframes glitch {
  0% { transform: translate(0); }
  20% { transform: translate(-2px, 2px); opacity: 0.8; }
  40% { transform: translate(2px, -2px); }
  60% { transform: translate(-2px, -2px); }
  80% { transform: translate(2px, 2px); opacity: 0.8; }
  100% { transform: translate(0); }
}
```

### 2. Data Stream Effect
**When:** Page load, new events
**How:** Text types in character-by-character (fast)

### 3. Pulse Animation
**When:** Breaking news, high magnitude
**How:** Subtle glow pulse
```css
.pulse {
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
```

### 4. Scan Line (Optional)
**When:** Background effect
**How:** Subtle horizontal line moving top to bottom
- Low opacity (0.02)
- Slow speed (5s)
- Creates CRT monitor feel

### 5. Cursor Glow
**When:** Hover over interactive elements
**How:** Small glow follows cursor
```css
.cursor-glow {
  box-shadow: 0 0 20px rgba(0, 255, 65, 0.2);
}
```

---

## Page Layouts

### Homepage: Intelligence Feed
```
┌─────────────────────────────────────────────────────────────┐
│ [◉ LIVE]  STRATINT v1.0  |  EVENTS: 12847  |  45ms          │  ← Header bar
├──────────────────────────────────────┬──────────────────────┤
│                                      │  [FILTERS]           │
│  EVENT STREAM                        │  ┌────────────────┐  │
│  ═══════════════                     │  │ Magnitude: 7.0+│  │
│                                      │  │ Category: ALL  │  │
│  [CYBER] MAG: 9.2 CONF: 0.91        │  │ Last: 24h      │  │
│  > Zero-day exploit...               │  └────────────────┘  │
│                                      │                      │
│  [MILITARY] MAG: 8.7 CONF: 0.89     │  [THREAT MAP]        │
│  > Military exercises...             │  ┌────────────────┐  │
│                                      │  │   🌍 GLOBE     │  │
│  [GEOPOLITICS] MAG: 7.5 CONF: 0.82  │  │                │  │
│  > Diplomatic tensions...            │  │    📍📍📍      │  │
│                                      │  │  📍    📍📍    │  │
│  [Load More]                         │  └────────────────┘  │
│                                      │                      │
│                                      │  [STATS]             │
│                                      │  Sources: 3,291      │
│                                      │  Avg Conf: 0.76      │
│                                      │  Enrichment: 450/min │
└──────────────────────────────────────┴──────────────────────┘
```

**Sections:**
1. **Fixed header** - Metrics, status, search
2. **Main feed** - 70% width, scrollable events
3. **Sidebar** - 30% width, filters + map + stats
4. **Footer** - Credits, API docs, status page

---

## Interactions

### Keyboard Shortcuts
```
Cmd+K       Open command palette
/           Focus search
F           Toggle filters
M           Toggle map
Esc         Close modals
↑↓          Navigate events
Enter       Open event details
1-9         Quick filters (categories)
R           Refresh feed
```

### Mouse Interactions
- **Hover:** Subtle highlight, cursor glow
- **Click:** Immediate feedback (no lag)
- **Drag:** Reorder filters (optional)
- **Scroll:** Infinite scroll, lazy load

---

## Technical Specs

### Tech Stack
```json
{
  "framework": "React 18",
  "language": "TypeScript",
  "build": "Vite",
  "styling": "TailwindCSS + CSS Modules",
  "animations": "Framer Motion",
  "state": "Zustand",
  "data": "React Query",
  "icons": "Lucide React",
  "map": "Mapbox GL / Deck.gl",
  "charts": "Recharts (custom styled)"
}
```

### Performance Targets
- **First Paint:** <500ms
- **Interactive:** <1s
- **60 FPS** animations
- **Lazy loading** for images/charts
- **Virtual scrolling** for long lists

### Accessibility
- **Keyboard navigation** (WCAG 2.1 AA)
- **Screen reader** support
- **High contrast** mode option
- **Reduced motion** respects prefers-reduced-motion
- **Focus indicators** visible

---

## Easter Eggs

1. **Konami Code:** Activate "full matrix" mode (green rain)
2. **Cmd+Shift+D:** Toggle "debug overlay" with frame rate, memory
3. **Triple-click logo:** ASCII art animation
4. **Type "neo":** Special color scheme
5. **Hover logo 10 times:** Glitch animation

---

## Implementation Priority

### Phase 1: Foundation (Week 1)
- [ ] Vite + React + TypeScript setup
- [ ] Tailwind config with custom theme
- [ ] Basic layout (header, main, sidebar)
- [ ] Typography system
- [ ] Color system

### Phase 2: Core Components (Week 1-2)
- [ ] Event card component
- [ ] Magnitude indicator
- [ ] Category badges
- [ ] Filter panel
- [ ] Command palette

### Phase 3: Data Integration (Week 2)
- [ ] Connect to MCP API
- [ ] Real-time updates (WebSocket or polling)
- [ ] Infinite scroll
- [ ] Error states
- [ ] Loading states

### Phase 4: Advanced Features (Week 3)
- [ ] Threat map visualization
- [ ] Live metrics
- [ ] Glitch effects
- [ ] Keyboard shortcuts
- [ ] Search functionality

### Phase 5: Polish (Week 3-4)
- [ ] Animations tuning
- [ ] Performance optimization
- [ ] Accessibility audit
- [ ] Mobile responsive
- [ ] Documentation

---

## Deployment

### Build
```bash
npm run build
# Output: dist/ (static files for Cloud Run)
```

### Environment Variables
```bash
VITE_API_URL=https://stratint-api-....run.app
VITE_WS_URL=wss://stratint-api-....run.app/ws
VITE_MAPBOX_TOKEN=pk.eyJ1...
```

### Cloud Run Deployment
- Serve via Nginx container
- CDN via Cloud CDN
- Gzip compression
- Cache headers for static assets

---

## Design Philosophy

**"Unapologetically Technical"**
- Don't hide the complexity - embrace it
- Data density over white space
- Monospace fonts are beautiful
- ASCII art is post-modern
- Glitches are features
- Dark themes only (no light mode)

**"Information First"**
- Every pixel serves a purpose
- No decorative elements
- No marketing fluff
- Direct, fast, efficient

**"Brutalist but Refined"**
- Raw aesthetics with intention
- Sharp edges, but aligned
- Monochrome with accents
- Geometric precision

This is an OSINT tool for people who want **signal, not noise**.
