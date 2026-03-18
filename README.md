# Subcults

Subcults is a privacy-first platform for discovering underground music communities. It combines a geospatial web app, a Go API, and a real-time ingestion pipeline to help artists and fans find scenes, events, and live audio experiences without sacrificing location privacy.

## Why this project stands out

- Built as a full-stack product with Go, React, TypeScript, Postgres/PostGIS, and Docker
- Designed around privacy-first location handling, consent enforcement, and trust-based discovery
- Integrates real-world platform concerns like JWT auth, payments with Stripe Connect, media storage, and live audio streaming
- Structured as a multi-service system with an API, frontend, and indexer for decentralized data ingestion

## Technical strengths demonstrated

- **Backend engineering:** Go services, REST APIs, middleware, configuration, and observability
- **Frontend development:** React, Vite, TypeScript, Tailwind, and interactive map-based UX
- **Data & infrastructure:** Postgres, PostGIS, Docker Compose, migrations, and production-oriented deployment workflows
- **Product thinking:** privacy safeguards, direct artist support, and clear domain modeling for community-driven discovery

## Stack

- **Languages:** Go, TypeScript
- **Frontend:** React, Vite, Tailwind CSS, MapLibre
- **Backend:** Chi-style HTTP architecture, JWT auth, structured logging
- **Data:** PostgreSQL, PostGIS
- **Integrations:** AT Protocol/Jetstream, LiveKit, Stripe Connect, Cloudflare R2
- **DevOps:** Docker Compose, multi-stage Docker builds, Make-based workflows

## What employers should know

This repository showcases the ability to design and ship a modern, production-minded platform: real-time data flows, geospatial features, external service integrations, privacy-aware engineering, and a clean separation between frontend, API, and background services.

## Quick start

```bash
cp configs/dev.env.example configs/dev.env
make build
make test
```

For a deeper look at architecture and workflows, see the code in `cmd/`, `internal/`, and `web/`.
