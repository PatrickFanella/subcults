# ADR-001: Stack Selection

**Status:** Accepted
**Date:** 2025-12-01

## Context

Subcults is a privacy-first platform for mapping underground music communities, requiring real-time data ingestion, geospatial queries, live audio streaming, and direct artist payments. We needed a technology stack that supports low-latency APIs, spatial data, WebRTC audio, payment processing, and decentralized identity — while remaining simple enough for a small team to operate.

## Decision

We chose the following stack:

| Layer       | Technology                     | Role                        |
| ----------- | ------------------------------ | --------------------------- |
| Backend API | **Go 1.24+**                   | HTTP API, business logic    |
| Frontend    | **React + TypeScript + Vite**  | Map-based discovery UI      |
| Database    | **Neon Postgres 16 + PostGIS** | Geospatial queries, FTS     |
| Streaming   | **LiveKit Cloud**              | WebRTC SFU for live audio   |
| Payments    | **Stripe Connect Express**     | Direct artist payouts       |
| Media       | **Cloudflare R2**              | S3-compatible media storage |
| Identity    | **AT Protocol / Jetstream**    | Decentralized user identity |
| Maps        | **MapLibre + MapTiler**        | Open-source map rendering   |

## Consequences

### Positive

- Go provides low memory footprint, fast compilation, and strong concurrency for API + indexer services.
- Neon Postgres with PostGIS gives managed Postgres with native geospatial support — no separate geo service needed.
- LiveKit Cloud eliminates the need to run our own WebRTC infrastructure (TURN, SFU).
- Stripe Connect Express means artists own their payment relationship — no platform lock-in.
- R2 offers zero egress fees for media delivery.
- AT Protocol integration gives users portable, self-sovereign identity.

### Negative

- Go lacks a mature ORM ecosystem — query building is more manual.
- LiveKit Cloud is a third-party dependency for a core feature (live streaming).
- Stripe Connect has complex onboarding UX for international creators.
- Neon is a managed service — less control over Postgres configuration.

### Neutral

- MapLibre requires MapTiler (or similar) for tile hosting — vendor choice is flexible.
- React + Vite is a standard frontend stack with broad hiring pool.

## Alternatives Considered

### Alternative 1: Node.js/Express Backend

Rejected because Go's performance characteristics better suit real-time ingestion (Jetstream processing) and the low per-request overhead budget (p95 <300ms).

### Alternative 2: MongoDB for Geospatial

Rejected because Postgres + PostGIS provides the same geospatial capabilities with ACID transactions, mature tooling, and the ability to combine spatial and relational queries in one system.

### Alternative 3: Self-hosted Janus/Mediasoup for WebRTC

Rejected due to operational complexity. LiveKit Cloud provides managed SFU, TURN, and global edge network — critical for live audio quality with a small ops team.
