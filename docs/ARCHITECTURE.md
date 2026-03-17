# Subcults Architecture

## Overview

Subcults is a privacy-first platform for mapping underground music communities. The system consists of three main services: a Go backend API, a Jetstream indexer for AT Protocol data ingestion, and a React frontend with MapLibre for map-based discovery.

## System Components

### Backend Services

#### API Service (`cmd/api`)
- **Language**: Go 1.24+
- **Router**: net/http (standard library)
- **Database**: Neon Postgres 16 with PostGIS
- **Auth**: JWT access + refresh tokens
- **Purpose**: REST API for scenes, events, payments (Stripe), auth, and media (R2)

#### Indexer Service (`cmd/indexer`)
- **Purpose**: Jetstream consumer ingesting AT Protocol records into Postgres
- **Data Flow**: AT Protocol → Jetstream → Indexer → Postgres

### Frontend Application (`web/`)

#### Tech Stack
- **Framework**: Vite + React 19 + TypeScript
- **Build**: SWC for fast compilation
- **Maps**: MapLibre + MapTiler tiles
- **Routing**: React Router v6
- **State**: Zustand 5
- **Testing**: Vitest + React Testing Library

## Frontend Architecture

### Routing Structure

The application uses React Router v6 for client-side routing with the following structure:

```
/ (AppLayout)
├── / (HomePage) - Map view of scenes and events
├── /scenes/:id (SceneDetailPage) - Scene details
├── /events/:id (EventDetailPage) - Event details
├── /account/login (LoginPage) - Authentication
├── /account (AccountPage) [Protected] - Account management
├── /settings (SettingsPage) [Protected] - User settings
├── /streams/:id (StreamPage) [Protected, Lazy] - Live audio streaming
├── /scenes/:id/settings (SceneSettingsPage) [Protected] - Scene settings
├── /demo/streaming (DemoStreamPage) - Streaming demo
├── /demo/stream-ui (DemoStreamUIPage) - Stream UI demo
├── /admin (AdminPage) [Admin Only, Lazy] - Admin dashboard
└── * (NotFoundPage) - 404 fallback
```

#### Route Protection Levels

1. **Public Routes**: Accessible to all users
   - `/` - Home page with map
   - `/scenes/:id` - Scene details
   - `/events/:id` - Event details
   - `/account/login` - Login page

2. **Protected Routes**: Require authentication
   - `/account` - Account management
   - `/settings` - User settings
   - `/stream/:room` - Live audio streaming

3. **Admin Routes**: Require admin role
   - `/admin` - Admin dashboard

#### Route Guards

Two guard components enforce access control:

- **RequireAuth**: Redirects unauthenticated users to `/account/login`
  - Preserves intended destination for post-login redirect
  - Used for all protected routes

- **RequireAdmin**: Enforces admin role requirement
  - Redirects unauthenticated users to `/account/login`
  - Redirects authenticated non-admin users to `/`
  - Used for admin-only routes

### Code Splitting

Heavy routes are lazy-loaded to improve initial load performance:

- **StreamPage**: LiveKit integration (WebRTC, heavy dependencies)
- **AdminPage**: Admin dashboard (limited audience)

Lazy loading uses React's `lazy()` and `Suspense` with a loading skeleton fallback.

### Layout Structure

#### AppLayout Component

The main layout shell provides:

- **Header**: Logo, search placeholder, auth status, navigation
- **Content Outlet**: Dynamic content area for routed pages
- **Mobile Navigation**: Hamburger menu for small screens
- **Bottom Nav**: Optional mobile bottom navigation bar

#### Accessibility Features

1. **Skip to Content Link**
   - Hidden by default, visible on focus
   - Allows keyboard users to bypass navigation
   - Links to `#main-content`

2. **Semantic HTML Landmarks**
   - `<header role="banner">` - Site header
   - `<nav role="navigation">` - Navigation areas with aria-labels
   - `<main role="main">` - Main content area

3. **ARIA Labels**
   - Navigation sections have descriptive labels
   - Mobile menu toggle has `aria-expanded` state
   - Loading states use `aria-live` regions

### Error Handling

#### ErrorBoundary Component

- Catches React rendering errors
- Displays user-friendly error message
- Shows error details in expandable section
- Provides refresh button to recover
- Logs errors to console for debugging

#### 404 Handling

- Catch-all route (`*`) renders `NotFoundPage`
- Clear messaging with link back to home
- Consistent with app styling

### State Management

#### Auth Store (`stores/authStore.ts`)

Production-ready authentication state management with secure token handling:

```typescript
interface User {
  did: string;
  role: 'user' | 'admin';
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isLoading: boolean;
  accessToken: string | null;
}
```

**Security Architecture**:
- **Access Token**: Stored in memory only (15min expiry) - prevents XSS token theft
- **Refresh Token**: Stored in httpOnly secure cookie (7d expiry) - set by backend
- **Cookie Flags**: SameSite=Lax, Secure=true for CSRF protection

**Methods**:
- `getState()` - Get current auth state
- `subscribe(listener)` - Subscribe to auth changes
- `setUser(user, accessToken)` - Set authenticated user with access token
- `logout()` - Clear user session (calls backend to clear refresh token cookie)
- `initialize()` - Check for existing session on app startup

**Hook**: `useAuth()` - React hook for accessing auth state with logout function

**Token Refresh Flow**:

```
┌─────────────┐
│  API Call   │
│  (401)      │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  Detect 401         │
│  Unauthorized       │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐     ┌──────────────────┐
│  Call Refresh       │────▶│  Exponential     │
│  Endpoint           │     │  Backoff Retry   │
│  (httpOnly cookie)  │◀────│  (3 attempts)    │
└──────┬──────────────┘     └──────────────────┘
       │
       ├─Success────▶ Update access token in memory
       │              Retry original request
       │              
       └─Failure────▶ Clear auth state
                      Broadcast logout to tabs
                      Redirect to /account/login
```

**Exponential Backoff**:
- **Max Retries**: 3 attempts
- **Initial Delay**: 1 second
- **Max Delay**: 10 seconds
- **Formula**: delay = min(initial * 2^retry, maxDelay)
- **Retry Triggers**: 5xx errors, network failures
- **No Retry**: 401 errors (invalid refresh token)

**Multi-Tab Synchronization**:
- Uses `BroadcastChannel` API for cross-tab communication
- Logout in one tab immediately logs out all other tabs
- Prevents inconsistent auth state across browser tabs
- Gracefully degrades if BroadcastChannel not supported

**Request Deduplication**:
- Multiple concurrent 401 responses share single refresh attempt
- Prevents refresh token endpoint overload
- All waiting requests receive same new token

#### API Client (`lib/api-client.ts`)

Type-safe HTTP client with automatic authentication and error handling:

**Features**:
- Automatic Authorization header injection
- Transparent token refresh on 401 responses
- Request/response interceptors
- Error handling with structured error types
- Convenience methods (GET, POST, PUT, PATCH, DELETE)

**Configuration**:
```typescript
apiClient.initialize({
  baseURL: '/api',
  getAccessToken: () => string | null,
  refreshToken: () => Promise<string | null>,
  onUnauthorized: () => void
});
```

**Usage Examples**:
```typescript
// GET request
const scenes = await apiClient.get<Scene[]>('/scenes');

// POST with body
const newScene = await apiClient.post('/scenes', { name: 'My Scene' });

// Skip auth for public endpoints
const data = await apiClient.get('/public/data', { skipAuth: true });

// Skip retry for specific requests
const response = await apiClient.get('/endpoint', { skipRetry: true });
```

**Error Handling**:
```typescript
try {
  await apiClient.get('/scenes');
} catch (error) {
  if (error instanceof ApiClientError) {
    console.error(error.status, error.code, error.message);
  }
}
```

**Automatic Retry Behavior**:
- 401 responses trigger token refresh + retry (unless `skipRetry: true`)
- Non-401 errors throw immediately (no automatic retry)
- Refresh failures call `onUnauthorized` callback

### Component Organization

```
web/src/
├── components/         # Reusable UI components
│   ├── MapView.tsx
│   ├── ClusteredMapView.tsx
│   ├── DetailPanel.tsx
│   ├── ErrorBoundary.tsx
│   ├── LoadingSkeleton.tsx
│   └── streaming/     # Streaming UI components
├── pages/             # Route-level page components
│   ├── HomePage.tsx
│   ├── SceneDetailPage.tsx
│   ├── SceneSettingsPage.tsx
│   ├── EventDetailPage.tsx
│   ├── AccountPage.tsx
│   ├── LoginPage.tsx
│   ├── SettingsPage.tsx
│   ├── StreamPage.tsx
│   ├── AdminPage.tsx
│   └── NotFoundPage.tsx
├── layouts/           # Layout shells
│   └── AppLayout.tsx
├── guards/            # Route protection
│   ├── RequireAuth.tsx
│   └── RequireAdmin.tsx
├── routes/            # Routing configuration
│   └── index.tsx
├── stores/            # State management (Zustand)
│   ├── authStore.ts
│   ├── entityStore.ts
│   ├── streamingStore.ts
│   ├── participantStore.ts
│   ├── toastStore.ts
│   └── notificationStore.ts
├── lib/               # Core libraries
│   └── api-client.ts
├── hooks/             # Custom React hooks
├── utils/             # Utility functions
└── types/             # TypeScript type definitions
```

## Data Flow

### AT Protocol Integration

```
User Posts → AT Protocol → Jetstream WebSocket → Indexer → Postgres → API → Frontend
```

1. Users create content via AT Protocol
2. Jetstream streams commits in real-time
3. Indexer processes and persists to Postgres
4. API serves data to frontend
5. Frontend displays on map with clustering

### Privacy Enforcement

All location data respects consent flags:

1. **Database Level**: `allow_precise` flag with CHECK constraints
2. **Repository Level**: Automatic consent enforcement in all queries
3. **API Level**: Geohash-based jitter applied for non-consenting users
4. **Frontend Level**: Map displays jittered coordinates for privacy

## Search & Ranking System

### Ranking Formula

Subcults uses a multi-factor ranking system to surface the most relevant scenes and events. The ranking module (`internal/ranking`) provides centralized calculation functions with calibration support.

#### Components

Ranking components are normalized to produce scores in the [0, 1] range:

1. **Text Match**: Relevance to search query
   - Based on full-text search (ts_rank with tsvector), normalized to [0, 1]
   - Higher scores for title matches vs. description/tags
   - Note: The raw ts_rank score is expected to be normalized before passing to ranking functions

2. **Proximity** (`ProximityWeight`): Geographic distance from search center
   - Hyperbolic decay function: `1 / (1 + distance_km)`
   - 1.0 at exact location, 0.5 at ~1km, decays gradually

3. **Recency** (`RecencyWeight`): Time until event starts (events only)
   - Linear decay: `1 - (time_diff / window_span)`
   - 1.0 for current/past events, 0.0 at search window end

4. **Trust** (`TrustWeight`): Scene reputation via alliance graph
   - Feature-flagged via `RANK_TRUST_ENABLED`
   - Returns 0 when disabled for graceful degradation
   - Clamped to [0, 1] range

#### Composite Formulas

**Scene Ranking**:
```
composite_score = (text * 0.4) + (proximity * 0.3) + (trust * 0.1)
Max score: 0.7 (trust disabled) or 0.8 (trust enabled)
```

**Event Ranking**:
```
composite_score = (recency * 0.3) + (text * 0.4) + (proximity * 0.2) + (trust * 0.1)
Max score: 0.9 (trust disabled) or 1.0 (trust enabled)
```

The default weights prioritize:
- **Text match (40%)**: Ensures query relevance for targeted search
- **Recency (30%, events only)**: Favors upcoming events for timely discovery
- **Proximity (20-30%)**: Considers geographic convenience
- **Trust (10%)**: Adds reputation signal without dominating results

### Calibration System

#### Configuration File

Ranking weights are defined in `configs/ranking.calibration.json`:

```json
{
  "version": "1.0",
  "weights": {
    "scene": {
      "text_match": 0.4,
      "proximity": 0.3,
      "trust": 0.1
    },
    "event": {
      "recency": 0.3,
      "text_match": 0.4,
      "proximity": 0.2,
      "trust": 0.1
    }
  }
}
```

#### Loading Process

1. Application startup loads calibration file via `ranking.LoadCalibration(path)`
2. If file is missing/invalid, defaults are used (graceful degradation)
3. Overrides are logged at INFO level for observability
4. Changes require deployment (not runtime modifiable for security)

#### Tuning Workflow

To adjust ranking behavior:

1. Edit `configs/ranking.calibration.json` with new weights
2. Run tests to verify composite score behavior changes as expected
3. Deploy updated configuration file
4. Monitor search quality metrics and user engagement
5. Iterate based on feedback

**Example**: To increase proximity importance for events:
```json
{
  "version": "1.0",
  "weights": {
    "event": {
      "recency": 0.25,
      "text_match": 0.35,
      "proximity": 0.3,
      "trust": 0.1
    }
  }
}
```

### Future: Machine Learning Opportunities

The current calibration system uses fixed weights optimized for general use cases. Future enhancements could include:

1. **Personalized Ranking**: Per-user weights based on interaction history
   - Click-through rate on different event types
   - Geographic preferences (local vs. willing to travel)
   - Genre/tag affinity signals

2. **A/B Testing Framework**: Experiment with weight variations
   - Split users into cohorts with different weight configurations
   - Measure engagement metrics (clicks, RSVP conversions, dwell time)
   - Statistical significance testing before rollout

3. **Learning-to-Rank Models**: ML models trained on engagement data
   - Features: All current components + user context
   - Training signal: Clicks, RSVPs, time-on-page
   - Models: LambdaMART, neural ranking (BERT-based)
   - Online learning for continuous improvement

4. **Contextual Adjustments**: Dynamic weights based on query context
   - Time-sensitive queries (e.g., "tonight") → increase recency weight
   - Location-specific queries (e.g., "Brooklyn") → increase proximity weight
   - Genre queries (e.g., "techno") → increase text weight

5. **Diversity & Exploration**: Balance relevance with discovery
   - Avoid filter bubbles by injecting diverse results
   - Exploration bonus for scenes with low visibility but high quality
   - Time-decay for repeatedly shown but not engaged content

**Implementation Notes**:
- All ML features must respect privacy (no tracking without consent)
- Models should be explainable for trust and debugging
- Offline evaluation metrics before production deployment
- Fallback to rule-based ranking if ML service unavailable

## Testing Strategy

### Frontend Tests

- **Unit Tests**: Component logic, hooks, utilities
- **Integration Tests**: Route behavior, guard redirects
- **Accessibility Tests**: Landmarks, ARIA, keyboard navigation
- **Performance Tests**: Map rendering, data loading

### Test Files

- `*.test.tsx` - Component and integration tests
- `*.perf.test.tsx` - Performance benchmarks
- Setup: `web/src/test/setup.ts`
- Runner: Vitest with jsdom

### Running Tests

```bash
cd web/
npm test              # Run all tests
npm run test:ui       # Interactive UI
npm run test:coverage # Coverage report
```

## Build & Deployment

### Development

```bash
make compose-up    # Start all services
cd web && npm run dev  # Frontend dev server
```

### Production Build

```bash
cd web && npm run build  # Build for production
```

- Output: `web/dist/`
- Optimization: Code splitting, tree shaking, minification
- Target: Modern browsers (ES2020+)

## Performance Budgets

- **API Latency**: p95 < 300ms
- **FCP (First Contentful Paint)**: < 1.0s
- **Map Render**: < 1.2s
- **Stream Join**: < 2s

## Security Considerations

### Frontend Security

1. **XSS Prevention**: React's automatic escaping
2. **CSRF**: API-level protection with tokens
3. **Secure Storage**: Avoid storing sensitive data in localStorage
4. **Content Security Policy**: Report-only → enforce progression
5. **Dependency Scanning**: npm audit in CI

### Privacy Protection

- Geohash-based location jitter for non-consenting users
- No tracking of user locations
- Minimal data collection
- Explicit consent for precise location sharing

## Implemented Features

- **Full Authentication**: JWT access + refresh tokens with dual-key rotation
- **Real-time Updates**: LiveKit WebRTC for live audio streaming
- **Internationalization**: i18next integration
- **State Management**: Zustand 5 stores (auth, entity, streaming, participants, toasts, notifications)
- **Advanced Search**: Full-text search with PostGIS geo queries
- **User Profiles**: DID-based identity via AT Protocol
- **Direct Payments**: Stripe Connect integration with scene payouts
- **Rate Limiting**: Tiered rate limiting with Redis + in-memory fallback

## Future Enhancements

### Planned Features

1. **Progressive Web App**: Offline support, install prompt
2. **Lazy-load heavy dependencies**: livekit-client, maplibre-gl
3. **Go 1.22+ path params**: Migrate from manual path parsing

### Technical Debt

1. Implement proper error tracking (Sentry, etc.)
2. Add OpenAPI schema for API documentation
3. Implement comprehensive E2E tests with Playwright
4. Add performance monitoring (Web Vitals, etc.)
5. Add full-text search indexes (migration planning needed)

## References

- [Privacy Guidelines](./PRIVACY.md)
- [API Documentation](./api/)
- [Performance Baselines](../PERFORMANCE.md)
- [Docker Setup](./docker.md)
