# Pre-deployment Validation Checklist

## Mandatory Checks

All items must be verified before deploying to production. Use the GitHub Actions **Deploy Gate** workflow for automated enforcement.

### Code Quality

- [ ] **CI pipeline green** — all unit tests passing
- [ ] **Integration tests passing** — testcontainers-based DB tests
- [ ] **Coverage thresholds met** — backend ≥80%, frontend ≥70%, critical packages ≥95%
- [ ] **Lint clean** — `go vet ./...` and frontend linters pass
- [ ] **Build succeeds** — all 3 binaries (api, indexer, backfill) and frontend build

### Security

- [ ] **Dependency scan clean** — `govulncheck ./...` and `npm audit` report no critical issues
- [ ] **Docker image scan** — Trivy reports no CRITICAL vulnerabilities
- [ ] **Secrets not in code** — no hardcoded credentials, tokens, or keys
- [ ] **CORS configuration verified** — production origins match expected domains

### Database

- [ ] **Migrations tested** — run on staging/dev first
- [ ] **Migrations reversible** — each `.up.sql` has a matching `.down.sql`
- [ ] **No destructive changes** — `DROP TABLE`, `DELETE`, column removal reviewed carefully
- [ ] **Backward compatible** — existing code works with both old and new schema during rollout

### Feature Flags

- [ ] **New features flagged** — experimental features gated behind feature flags
- [ ] **Flag defaults safe** — disabled by default, opt-in for rollout
- [ ] **Fallback behavior tested** — feature works correctly when flag is off

### Monitoring

- [ ] **Dashboards accessible** — Grafana at sentinel.subcult.tv shows data
- [ ] **Alert rules configured** — new endpoints have corresponding alerts
- [ ] **Metrics exposed** — new Prometheus metrics registered and scraped
- [ ] **Log levels appropriate** — no debug logging in production

### Operational

- [ ] **Rollback plan documented** — know how to revert if issues arise
- [ ] **Deployment tested on staging** — full deploy cycle verified
- [ ] **On-call aware** — team notified before deployment

## Automated Gate (GitHub Actions)

The `deploy-gate.yml` workflow automates these checks:

1. **Pre-checks** — verify CI status
2. **Full test suite** — unit tests with coverage validation
3. **Security scan** — govulncheck + Trivy image scan
4. **Migration validation** — check migration file consistency
5. **Manual approval** — requires human sign-off for production
6. **Deploy** — outputs deployment instructions

Trigger: `Actions → Deploy Gate → Run workflow → Select environment`

## Post-deployment Verification

After deploying:

- [ ] `/health/live` returns 200
- [ ] `/health/ready` returns 200 with all checks passing
- [ ] Grafana dashboards show normal metrics
- [ ] No new alerts firing
- [ ] Application logs show normal startup (check Loki or `docker logs`)
- [ ] Key user flows work (manual spot check)
