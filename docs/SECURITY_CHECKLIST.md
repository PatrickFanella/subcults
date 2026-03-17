# Security Checklist — Pre-Deployment

Verify each item before deploying to production. Items marked with **[auto]** are enforced by code/config; manual items need a human check.

## Authentication & Authorization

- [ ] JWT_SECRET is at least 32 characters and randomly generated
- [ ] Access token expiry is 15 minutes; refresh token expiry is 7 days
- [ ] Dual-key rotation is configured for zero-downtime key changes
- [ ] All mutation endpoints require authenticated user (DID in context)
- [ ] Scene/event mutation handlers verify ownership before allowing changes **[audit needed]**
- [ ] Payment onboarding verifies scene ownership **[audit needed]**

## Transport Security

- [x] Caddy auto-TLS enabled for subcults.subcult.tv **[auto]**
- [x] HSTS header with preload directive **[auto — Caddy config + Go middleware]**
- [ ] Database connection uses SSL (`?sslmode=require` in DATABASE_URL)
- [ ] R2 endpoint uses HTTPS

## Security Headers

- [x] X-Content-Type-Options: nosniff **[auto]**
- [x] X-Frame-Options: DENY **[auto]**
- [x] Referrer-Policy: strict-origin-when-cross-origin **[auto]**
- [x] Permissions-Policy: camera=(), microphone=(self), geolocation=(self) **[auto]**
- [x] Content-Security-Policy-Report-Only with report-uri **[auto]**
- [x] Server header stripped **[auto — Caddy]**
- [ ] CSP graduated to enforced mode (after 2 weeks of clean report data)

## Input Validation

- [x] Request body size limits: 1MB JSON, 15MB uploads **[auto — MaxBodySize middleware]**
- [x] Rate limiting on all endpoints (general: 1000/min, search: 100/min, creation: 5-10/hr) **[auto]**
- [ ] All user input validated before use (scene names, coordinates, etc.)
- [ ] SQL queries use parameterized statements (when Postgres integration is complete)

## CORS

- [x] No wildcard origins — explicit allowlist only **[auto]**
- [x] CORS disabled if no origins configured **[auto]**
- [ ] Production CORS_ALLOWED_ORIGINS set to `https://subcults.subcult.tv` only

## Privacy

- [x] Location consent enforced via `EnforceLocationConsent()` **[auto — repository layer]**
- [x] Public coordinates use geohash-based jitter **[auto]**
- [x] Audit logging with hash chain for tamper detection **[auto]**
- [ ] No PII in application logs (grep logs for email, IP, full DID)
- [ ] Data classification policy reviewed (see DATA_CLASSIFICATION.md)

## Secrets Management

- [ ] No secrets in source code or Docker images
- [ ] All secrets loaded from environment variables
- [ ] `dev.env` is in `.gitignore`
- [ ] Stripe webhook secret configured and verified
- [ ] R2 credentials configured
- [ ] LiveKit credentials configured

## Infrastructure

- [ ] Docker images use distroless/nonroot base **[auto — Dockerfile]**
- [ ] CGO disabled for static binaries **[auto — Dockerfile]**
- [ ] External Caddy on `web` Docker network routes to subcults containers
- [ ] Health check endpoints responding (`/health/live`, `/health/ready`)
- [ ] Prometheus metrics endpoint protected with bearer token in production

## Dependency Security

- [ ] `govulncheck` passes with no known vulnerabilities
- [ ] `npm audit` passes (or known issues are documented and accepted)
- [ ] Go module checksums verified (`go mod verify`)

## Monitoring & Incident Response

- [ ] Structured logging enabled (JSON format in production)
- [ ] CSP violation reports being collected via `/api/csp-report`
- [ ] Incident response plan reviewed (see INCIDENT_RESPONSE.md)
- [ ] Secret rotation runbook tested
