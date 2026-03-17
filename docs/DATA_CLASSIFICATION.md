# Data Classification Policy

Subcults classifies data into four tiers. Each tier prescribes storage, access, logging, and retention rules.

## Tiers

| Tier           | Description                             | Examples                                                                               | Storage                                 | Access                                  | Logging                   |
| -------------- | --------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------------- | --------------------------------------- | ------------------------- |
| **Public**     | Freely visible to any user              | Scene names, descriptions, public event titles, jittered coordinates                   | Postgres, CDN cache                     | Unauthenticated reads                   | Standard access logs      |
| **Internal**   | Operational data not shown to end users | Analytics aggregates, error rates, trust scores, feature flags                         | Postgres, Prometheus                    | Staff / automated systems               | Structured logs, no PII   |
| **Sensitive**  | User-consented personal data            | Precise lat/lng (when `allow_precise = true`), DID, email (if collected), IP addresses | Postgres (encrypted at rest), audit log | Authenticated owner + consented viewers | Audit trail, hash-chained |
| **Restricted** | Secrets and financial data              | JWT signing keys, Stripe API keys, database credentials, payment amounts per user      | Env vars / Vault, never in code or logs | Service accounts only                   | Never logged in plaintext |

## Rules

1. **Sensitive data must never appear in logs.** Use structured logging fields like `user_did` (truncated or hashed if logged for debugging).
2. **Restricted data must never be committed to version control.** Use environment variables or a secrets manager.
3. **Location data defaults to Public (jittered).** Precise coordinates are Sensitive and only stored or returned when `allow_precise = true`.
4. **PII minimization.** Only collect what is necessary. IP addresses are retained for rate-limiting only and are not persisted beyond the request lifecycle.
5. **Encryption at rest.** The Postgres provider (Neon) encrypts storage. R2 objects are encrypted by default.
6. **Encryption in transit.** All external connections use TLS (Caddy auto-TLS, Neon requires SSL, R2 HTTPS endpoints).

## Retention

| Tier       | Default Retention                             | Notes                                                          |
| ---------- | --------------------------------------------- | -------------------------------------------------------------- |
| Public     | Indefinite                                    | Follows user deletion requests                                 |
| Internal   | 90 days (metrics), 30 days (logs)             | Aggregated data may be kept longer                             |
| Sensitive  | Until user revokes consent or deletes account | Audit log entries retained 1 year                              |
| Restricted | Rotated per schedule                          | JWT keys: dual-key rotation; Stripe keys: per Stripe dashboard |
