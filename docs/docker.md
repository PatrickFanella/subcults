# Docker Images

This document describes the Docker images for the Subcults application.

## Images

### API Image (`Dockerfile.api`)

Multi-stage build for the Subcults API server.

**Build:**
```bash
# Default (linux/amd64)
docker build -f Dockerfile.api -t subcults-api:latest .

# Cross-compile for ARM64
docker build -f Dockerfile.api -t subcults-api:latest \
  --build-arg TARGETARCH=arm64 .
```

**Run:**
```bash
docker run --rm subcults-api:latest --help
```

**Base Images:**
- Build stage: `golang:1.22-alpine`
- Runtime stage: `gcr.io/distroless/static-debian12:nonroot`

**Size:** ~3.4MB (target: <60MB ✓)

**Security:**
- Uses distroless base for minimal attack surface
- Runs as non-root user
- CGO disabled for static binary
- No unnecessary packages in runtime

---

### Indexer Image (`Dockerfile.indexer`)

Multi-stage build for the Subcults Jetstream Indexer.

**Build:**
```bash
# Default (linux/amd64)
docker build -f Dockerfile.indexer -t subcults-indexer:latest .

# Cross-compile for ARM64
docker build -f Dockerfile.indexer -t subcults-indexer:latest \
  --build-arg TARGETARCH=arm64 .
```

**Run:**
```bash
docker run --rm subcults-indexer:latest --help
```

**Base Images:**
- Build stage: `golang:1.22-alpine`
- Runtime stage: `gcr.io/distroless/static-debian12:nonroot`

**Size:** ~3.4MB (target: <60MB ✓)

**Security:**
- Uses distroless base for minimal attack surface
- Runs as non-root user
- CGO disabled for static binary
- No unnecessary packages in runtime

---

### Frontend Image (`Dockerfile.frontend`)

Multi-stage build for frontend static assets.

**Build:**
```bash
docker build -f Dockerfile.frontend -t subcults-frontend:latest .
```

**Base Images:**
- Build stage: `node:20-alpine`
- Output stage: `scratch` (static files only)

**Output:** Static files in `/dist` directory for serving via Caddy/nginx.

**Notes:**
- Uses `npm ci` for reproducible builds when `package-lock.json` exists
- Falls back to `npm install` when no lock file is present
- Skips install if no dependencies are defined
- `--ignore-scripts` flag prevents arbitrary code execution during install

---

## Security Scanning

Run security scans on the built images:

```bash
# Using Trivy
trivy image subcults-api:latest
trivy image subcults-indexer:latest

# Using Grype
grype subcults-api:latest
grype subcults-indexer:latest
```

**Acceptance criteria:** No HIGH or CRITICAL vulnerabilities in runtime images.

---

## .dockerignore

The `.dockerignore` file excludes:
- `node_modules/` - Dependencies (rebuilt in container)
- `bin/`, `dist/` - Build artifacts
- `.git/` - Git history
- `.env*` - Environment files (secrets should be runtime injected)
- IDE/editor files
- Temporary files
- Test and documentation files

This keeps the Docker build context small and fast.

---

## Environment Variables

All configuration should be passed via environment variables at runtime. No secrets are baked into the images.

Example:
```bash
docker run -e DATABASE_URL=postgres://... subcults-api:latest
```
