# Incident Response Plan

## Severity Levels

| Level             | Description                                   | Response Time                     | Examples                                                    |
| ----------------- | --------------------------------------------- | --------------------------------- | ----------------------------------------------------------- |
| **P0 — Critical** | Service down or active data breach            | 15 min acknowledge, 1 hr mitigate | Database compromise, payment data leak, full outage         |
| **P1 — High**     | Major feature broken, potential data exposure | 1 hr acknowledge, 4 hr mitigate   | Auth bypass, Stripe webhook failure, persistent 5xx         |
| **P2 — Medium**   | Degraded performance, non-critical bug        | 4 hr acknowledge, 24 hr fix       | Elevated error rates, slow queries, partial feature failure |
| **P3 — Low**      | Minor issue, cosmetic, no user impact         | Next business day                 | Logging noise, non-critical deprecation warning             |

## Response Procedure

### 1. Detect

- Monitoring alerts (Prometheus / alerting rules)
- User reports
- CSP violation logs
- Dependency vulnerability scans (govulncheck, npm audit)

### 2. Triage

- Confirm the issue and assign severity level.
- Page on-call if P0/P1.
- Create a GitHub issue labeled `incident` with severity tag.

### 3. Contain

- **P0 data breach**: Rotate affected secrets immediately (JWT keys, Stripe keys, DB password). Revoke compromised sessions.
- **P0 outage**: Roll back to last known good deployment via `docker compose up -d` with previous image tag.
- **Auth bypass**: Disable affected endpoint via feature flag or Caddy `respond 503`.
- **Payment issue**: Pause Stripe webhook processing; investigate before re-enabling.

### 4. Eradicate

- Identify root cause.
- Apply fix on a branch, get review, merge.
- Deploy fix.

### 5. Recover

- Verify fix in production (smoke tests, monitoring).
- Re-enable any disabled features.
- Confirm no residual impact.

### 6. Post-Mortem

- Write post-mortem within 48 hours for P0/P1.
- Include: timeline, root cause, impact scope, what went well, what needs improvement.
- File follow-up issues for preventive measures.

## Communication

| Audience                    | Channel                      | When                     |
| --------------------------- | ---------------------------- | ------------------------ |
| Team                        | GitHub issue + Discord/Slack | Immediately on detection |
| Users (if data affected)    | Status page / email          | Within 72 hours per GDPR |
| Stripe (if payment related) | Stripe dashboard support     | As needed                |

## Key Contacts

| Role             | Responsibility                       |
| ---------------- | ------------------------------------ |
| On-call engineer | First responder, triage, containment |
| Project lead     | Escalation, communication decisions  |
| Security lead    | Breach assessment, secret rotation   |

## Secret Rotation Runbook

1. **JWT keys**: Generate new key pair, deploy with dual-key config, wait for old tokens to expire (15 min access, 7 day refresh), remove old key.
2. **Database password**: Update in Neon dashboard, update `DATABASE_URL` env var, restart services.
3. **Stripe keys**: Regenerate in Stripe dashboard, update env vars, restart API.
4. **R2 keys**: Regenerate in Cloudflare dashboard, update env vars, restart API.
