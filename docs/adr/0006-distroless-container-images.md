# ADR-006: Distroless Container Images

**Status:** Accepted
**Date:** 2026-01-15

## Context

Production container images need to be small (fast pulls, less storage) and secure (minimal attack surface). The API, Indexer, and Backfill services are all compiled Go binaries with no runtime dependencies (CGO disabled for static linking).

## Decision

Use multi-stage Docker builds with `gcr.io/distroless/static-debian12:nonroot` as the runtime base image:

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
# ... compile with CGO_ENABLED=0

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/bin/api /api
ENTRYPOINT ["/api"]
```

Key properties:

- **No shell:** No `/bin/sh`, no package manager — no interactive exploit surface.
- **Nonroot:** Runs as UID 65534, not root.
- **Static binary:** CGO disabled, so no libc dependency.
- **Image size:** ~3.4 MB per service image.

## Consequences

### Positive

- Minimal attack surface — no shell, no package manager, no unnecessary libraries.
- Tiny images (~3.4 MB) mean fast container pulls and low registry storage.
- Non-root execution by default — defense in depth.
- No runtime dependencies to patch — only the Go binary and distroless base.

### Negative

- Cannot `docker exec` into the container for debugging (no shell). Must use log output, health endpoints, or sidecar debug containers.
- Cannot install ad-hoc tools at runtime (e.g., `curl` for manual health checks inside the container).

### Neutral

- Build stage uses `golang:1.24-alpine` which is larger (~300 MB) but only used during CI — not shipped to production.
- Debug variant (`gcr.io/distroless/static-debian12:debug`) is available if shell access is needed during development.

## Alternatives Considered

### Alternative 1: Alpine Runtime

Alpine (~5 MB) includes a shell and apk package manager. Rejected because the shell increases attack surface and the Go binary doesn't need libc (musl or glibc). The marginal size difference (5 MB vs 3.4 MB) isn't worth the security tradeoff.

### Alternative 2: Scratch

`FROM scratch` produces the smallest possible image but lacks CA certificates and timezone data. Distroless includes these essentials while remaining nearly as minimal.
