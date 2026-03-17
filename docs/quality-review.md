# Quality Review -- Subcults (Full Repository)

- **Verdict: Not Ready (requires critical fixes)**
- **Scope**: Full repository audit (`uplink/prod` branch, all files)
- **Date**: 2026-02-28
- **Go files**: 263 (83K LOC) | **TS/JS files**: 216 | **Migrations**: 29

---

## Triage

- Docs-only: no
- React/Next perf review: no (Vite/React SPA, not Next.js)
- UI guidelines audit: no (full repo review, not a diff-based UI change)
- Reason:
  - Full-stack Go + React/TypeScript application
  - No Next.js (Vite SPA) -- React perf review applied inline instead
  - Review covers backend, frontend, infra, docs, and security holistically

---

## Strengths

- **Comprehensive security posture**: Input validation (SQL injection, XSS, SSRF), CSP reporting, EXIF stripping, Gitleaks scanning, SBOM generation, dependency vulnerability scanning, and audit logging with hash chains
- **Solid auth architecture**: Access tokens in memory only (not localStorage), refresh via httpOnly cookies, JWT rotation with dual-key validation
- **TypeScript strict mode** enabled with `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch`
- **Good test infrastructure**: 139 Go test files, integration tests, a11y tests (axe-core), E2E (Playwright), load tests (k6), Lighthouse CI
- **Privacy-first design**: Coarse geohashing, consent-based precision, no IP logging, GDPR compliance docs
- **Well-structured Zustand stores** with primitive selectors to avoid unnecessary re-renders
- **Docker security**: Non-root users in all containers, multi-stage builds, minimal base images
- **Observability**: OpenTelemetry tracing, Prometheus metrics, health check endpoints, canary deployment system

---

## Issues

### Critical (Must Fix)

#### C1. Anonymous post creation bypasses authentication
- **Location**: `internal/api/post_handlers.go:141-147`
- **What**: `CreatePost` allows unauthenticated users to create posts using a hardcoded `did:example:anonymous` DID. The comment acknowledges this is not production-safe.
- **Why**: Any unauthenticated caller can create arbitrary posts in any scene/event -- content injection, spam, abuse.
- **Fix**: Return `401 Unauthorized` when `authorDID` is empty, matching other handlers.

#### C2. Timing side-channel on internal service bypass token
- **Location**: `internal/middleware/ratelimit.go:321-323`
- **What**: `InternalServiceBypassFunc` compares the secret token using `==` (non-constant-time).
- **Why**: Enables progressive byte-by-byte brute-force via response time measurement.
- **Fix**: `subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1` (already used in `internal/indexer/handler.go:34`).

#### C3. Timing side-channel on metrics auth token
- **Location**: `cmd/api/main.go:846-849`
- **What**: Same issue as C2 -- metrics bearer token compared with `!=`.
- **Fix**: Use `subtle.ConstantTimeCompare`.

#### C4. No minimum JWT secret length enforcement
- **Location**: `internal/auth/jwt.go:60-67`, `internal/config/config.go:498-500`
- **What**: Config validation only checks for empty JWT secret. A 1-character secret passes validation.
- **Why**: Short HMAC-SHA256 keys are trivially brute-forceable.
- **Fix**: Add `len(secret) < 32` check in `Config.Validate()`.

#### C5. Production `DATABASE_URL` uses `sslmode=disable`
- **Location**: `deploy/.env:24`
- **What**: Database credentials transmitted in cleartext even in production.
- **Why**: If the internal Docker network is ever bridged/sniffed, the DB password is exposed.
- **Fix**: Change to `sslmode=require` (the `deploy/.env.example` already uses `require`).

#### C6. Security contact email inconsistency
- **Location**: `docs/SECURITY.md:138` vs `README.md:501`
- **What**: `SECURITY.md` says `security@subcults.dev`, `README.md` says `info@subcult.tv`. Different addresses, different domains.
- **Why**: Security researchers may report vulnerabilities to the wrong/unmonitored address.
- **Fix**: Consolidate to a single, monitored address in both files.

#### C7. `INTERNAL_AUTH_TOKEN` vs `METRICS_AUTH_TOKEN` mismatch
- **Location**: `README.md:342` (documents `INTERNAL_AUTH_TOKEN`) vs `cmd/api/main.go:842` (reads `METRICS_AUTH_TOKEN`)
- **What**: Docs tell users to set `INTERNAL_AUTH_TOKEN` to protect the API metrics endpoint, but the code reads `METRICS_AUTH_TOKEN`.
- **Why**: Users who follow the docs will have an unprotected `/metrics` endpoint.
- **Fix**: Align docs and code on a single variable name.

#### C8. Auth guard race condition -- false redirects on page refresh
- **Location**: `web/src/guards/RequireAuth.tsx:19`, `web/src/stores/authStore.ts:43`
- **What**: `authStore` initializes with `isLoading: true`, `isAuthenticated: false`. `RequireAuth` does not check `isLoading` -- it immediately redirects to login before token refresh completes.
- **Why**: Every hard refresh on a protected page flash-redirects authenticated users to login.
- **Fix**: Check `isLoading` and render a spinner before making the redirect decision.

---

### Important (Should Fix)

#### I1. `main()` is 1200+ lines -- monolithic, untestable
- **Location**: `cmd/api/main.go:44-1253`
- **What**: Configuration loading, repository init, metrics, 20+ route definitions, middleware, server lifecycle all in one function.
- **Fix**: Extract `setupRepositories()`, `setupRoutes()`, `setupMiddleware()`.

#### I2. Config validation errors logged as warnings, server proceeds
- **Location**: `cmd/api/main.go:266-272`
- **What**: Missing `DATABASE_URL`, `JWT_SECRET`, etc. are logged as warnings but do not prevent startup.
- **Why**: Server runs in a broken state, causing nil-pointer panics or silent failures at runtime.
- **Fix**: Fatal on required config errors.

#### I3. Hardcoded placeholder amount in checkout fee calculation
- **Location**: `internal/api/payment_handlers.go:277-278`
- **What**: Application fee calculated from hardcoded `$100` placeholder, not actual cart value.
- **Why**: Platform fee is wrong for every transaction.
- **Fix**: Fetch actual amount from Stripe Price API or session line items.

#### I4. SSRF check allows unresolvable domains through
- **Location**: `internal/validate/url.go:141-145`
- **What**: When DNS resolution fails, the URL is allowed (fail-open). Enables DNS rebinding attacks.
- **Fix**: Fail closed -- reject unresolvable domains for outgoing requests.

#### I5. In-memory rate limit store has no maximum size bound
- **Location**: `internal/middleware/ratelimit.go:122-135`
- **What**: Between cleanup cycles (5min), unlimited buckets can be created. Spoofed `X-Forwarded-For` IPs can exhaust memory.
- **Fix**: Add max bucket count with LRU eviction.

#### I6. Rate limiter trusts `X-Forwarded-For` directly
- **Location**: `internal/middleware/ratelimit.go:208-228`
- **What**: Client-provided forwarded headers are trusted for IP extraction.
- **Fix**: Only trust the rightmost IP added by the trusted proxy (Caddy).

#### I7. `livekit-client` (~200KB) imported at module level, included in initial bundle
- **Location**: `web/src/stores/streamingStore.ts:6-14`
- **What**: Heavy dependency imported at top level. Since `App.tsx` references the store, it's pulled into the initial bundle even though only `StreamPage` (lazy-loaded) needs it.
- **Fix**: Dynamic `import()` inside the `connect` action, or split the store.

#### I8. `maplibre-gl` (~200KB) eagerly loaded via non-lazy `HomePage`
- **Location**: `web/src/components/MapView.tsx:8`, `web/src/routes/index.tsx:57`
- **What**: `HomePage` is not lazy-loaded but imports `maplibre-gl`.
- **Fix**: Lazy-load `HomePage` or at minimum `MapView` with `React.lazy`.

#### I9. No container resource limits in production compose
- **Location**: `deploy/compose.yml` (all services)
- **What**: No `mem_limit`/`cpus` on any service. A memory leak can take down the entire host.
- **Fix**: Add `deploy.resources.limits` per service.

#### I10. Nginx security headers lost on cache-controlled assets
- **Location**: `Dockerfile.frontend:103-107`
- **What**: Nested `location` blocks with `add_header Cache-Control` override parent-level security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`) per nginx behavior.
- **Fix**: Repeat security headers in each location block.

#### I11. Indexer exposed on external `web` network unnecessarily
- **Location**: `deploy/compose.yml:100-104`
- **What**: Indexer is on both `internal` and `web` networks. Docs claim internal-only.
- **Fix**: Remove `web` network unless external observability access is required.

#### I12. Webhook body read has no explicit size limit
- **Location**: `internal/api/webhook_handlers.go:46`
- **What**: `io.ReadAll(r.Body)` without size restriction. While global `MaxBodySize` (1MB) middleware exists, webhook payloads should be limited to ~64KB for defense-in-depth.
- **Fix**: `io.LimitReader(r.Body, 65536)`.

#### I13. `Vary: Origin` header not set in CORS middleware
- **Location**: `internal/middleware/cors.go:72-77`
- **What**: Missing `Vary: Origin` when setting CORS headers. CDN/shared cache could serve wrong origin's response.
- **Fix**: `w.Header().Add("Vary", "Origin")`.

#### I14. README claims "chi router" -- code uses `net/http.ServeMux`
- **Location**: `README.md:20`, `docs/ARCHITECTURE.md:14` vs `cmd/api/main.go:541`
- **Fix**: Update docs to reflect standard library router.

#### I15. ARCHITECTURE.md lists 5+ "future features" that are already implemented
- **Location**: `docs/ARCHITECTURE.md:540-548`
- **What**: JWT auth, Zustand, i18n, Stripe payments, and full-text search listed as "planned" but already built.
- **Fix**: Move to "Implemented" or remove the section.

#### I16. `deploy/.env.example` missing `VITE_*` frontend build variables
- **Location**: `configs/dev.env.example` (missing `VITE_API_URL`, `VITE_MAPTILER_API_KEY`, `VITE_LIVEKIT_WS_URL`, `VITE_APP_VERSION`)
- **Fix**: Add the missing VITE variables to the dev env example.

#### I17. API_REFERENCE.md documents frontend client, not backend endpoints
- **Location**: `docs/API_REFERENCE.md`
- **What**: 40+ backend API endpoints are not documented in any central reference.
- **Fix**: Rename to "API Client Reference" and create a proper backend endpoint reference.

#### I18. Deprecated hooks used in StreamPage
- **Location**: `web/src/pages/StreamPage.tsx:13-14`
- **What**: `useStreamingConnection` and `useStreamingActions` are marked `@deprecated` in the store but still used.
- **Fix**: Replace with individual selectors as recommended by the deprecation notice.

#### I19. Stale notification subscription from localStorage
- **Location**: `web/src/stores/notificationStore.ts:44-53,86-94`
- **What**: Legacy localStorage data initializes the store as "subscribed" with potentially invalid endpoint/keys.
- **Fix**: Clear legacy data on read instead of using it.

---

### Minor (Nice to Have)

| # | Location | Issue |
|---|----------|-------|
| M1 | `internal/validate/string.go:156` | `regexp.MustCompile` in `SceneName()` recompiled on every call -- should be package-level var |
| M2 | `internal/api/webhook_handlers.go:99-111` | Webhook event type strings hardcoded -- should be constants |
| M3 | Various handler files | Inconsistent error code style: constants vs inline string literals |
| M4 | `internal/config/config.go:598-606` | `maskSecret` reveals first 4 chars of secrets -- prefer last 4 or fewer chars |
| M5 | `cmd/api/main.go:560-840` | Manual string-split routing; Go 1.22+ supports path params in `http.ServeMux` |
| M6 | `web/src/stores/toastStore.ts:108-128` | `useToasts()` returns unstable references, causing spurious effect re-fires |
| M7 | `web/src/pages/SceneSettingsPage.tsx:74-116` | `useEffect` has too many dependencies including unstable functions |
| M8 | `web/src/pages/SceneSettingsPage.tsx:427-535` | Color `<input type="color">` elements lack `aria-label` |
| M9 | `web/src/stores/streamingStore.ts:559-567` | Reconnect `setTimeout` not tracked/cancellable on disconnect |
| M10 | `web/src/stores/authStore.ts` | Custom pub/sub instead of Zustand -- inconsistent with all other stores |
| M11 | `web/src/components/ErrorBoundary.tsx:81-83` | Only recovery is full page reload -- add "Try Again" state reset |
| M12 | `docs/ARCHITECTURE.md:48` | Route path `/stream/:room` should be `/streams/:id` |
| M13 | `README.md:78-79` | Go version says 1.21+ (actual: 1.24.4), Node says 18+ (actual: 22) |
| M14 | `docker-compose.yml:1` | Deprecated `version: '3.8'` key |
| M15 | `Makefile:33-34` | `build-frontend` runs from project root, should `cd web` first |
| M16 | `docs/SECURITY.md:131` | Claims "distroless images" -- only indexer uses distroless, API/frontend use Alpine |
| M17 | Migrations 000003/5/9 | FTS indexes commented out; migration 000026 adds wrapper but originals never revisited |
| M18 | `internal/middleware/securityheaders.go:17` | CSP is `Report-Only` -- not enforcing yet |

---

## Documentation Consistency Summary

| Severity | Count | Key Themes |
|----------|-------|------------|
| Critical | 2 | Metrics env var mismatch (C7), security contact email inconsistency (C6) |
| Important | 10 | Router claim (chi vs stdlib), stale "future features", missing env vars, API docs scope, indexer network exposure claim |
| Minor | 7 | Version numbers, project structure listing, component tree, route paths |

---

## Aggregate Counts

| Severity | Count |
|----------|-------|
| **Critical** | 8 |
| **Important** | 19 |
| **Minor** | 18 |

---

## Recommended Fix Priority

**Immediate (before next deploy):**
1. C1 -- Remove anonymous post bypass
2. C2/C3 -- Constant-time token comparisons (2 locations)
3. C5 -- Enable SSL on production DB connection
4. C7 -- Fix metrics auth env var name mismatch
5. C8 -- Add `isLoading` check to auth guard

**This sprint:**
6. C4 -- JWT secret minimum length
7. C6 -- Consolidate security contact email
8. I2 -- Fatal on required config errors
9. I3 -- Fix placeholder payment amount
10. I6 -- Validate X-Forwarded-For against trusted proxy
11. I9 -- Add container resource limits
12. I10 -- Fix nginx security header inheritance

**Next sprint:**
13. I1 -- Refactor monolithic main()
14. I7/I8 -- Lazy-load heavy dependencies (livekit-client, maplibre-gl)
15. I14-I17 -- Documentation updates
16. Remaining Important and Minor items
