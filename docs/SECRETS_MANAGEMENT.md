# Secrets Management

This document describes how secrets are managed, rotated, and protected in Subcults.

## Overview

Subcults follows a secrets-in-environment-variables approach: **no secrets are ever committed to version control**. All sensitive values are passed at runtime via environment variables, `.env` files excluded from git, or a secrets manager.

## Secret Categories

| Secret | Variable | Rotation Interval |
|--------|----------|------------------|
| JWT signing key (current) | `JWT_SECRET_CURRENT` | Every 90 days |
| JWT signing key (previous) | `JWT_SECRET_PREVIOUS` | Removed after rotation window |
| Database password | In `DATABASE_URL` | Every 6 months |
| Redis password | In `REDIS_URL` | Every 6 months |
| Stripe API key | `STRIPE_API_KEY` | On compromise or annually |
| Stripe webhook secret | `STRIPE_WEBHOOK_SECRET` | On compromise |
| LiveKit API key | `LIVEKIT_API_KEY` | Annually |
| LiveKit API secret | `LIVEKIT_API_SECRET` | Annually |
| MapTiler API key | `MAPTILER_API_KEY` | Annually |
| R2 access key | `R2_ACCESS_KEY_ID` | Annually |
| R2 secret key | `R2_SECRET_ACCESS_KEY` | Annually |
| Internal service token | `INTERNAL_SERVICE_TOKEN` | On compromise |

## Where Secrets Live

### Development

1. Copy `configs/dev.env.example` to `configs/dev.env`
2. Fill in values; this file is excluded from git via `.gitignore`
3. Never commit `configs/dev.env` or any `*.env` file (except `*.env.example`)

### Production

Use a secrets manager. Supported options:

- **GitHub Actions Secrets** — for CI/CD pipelines
- **HashiCorp Vault** — for server-side secret injection at startup
- **Kubernetes Secrets** — when deployed on Kubernetes
- **AWS Secrets Manager / GCP Secret Manager** — for cloud deployments

Inject secrets as environment variables at container startup. Never bake them into Docker images.

## JWT Secret Rotation

Zero-downtime rotation is supported via dual-key configuration. See [JWT_ROTATION_GUIDE.md](./JWT_ROTATION_GUIDE.md) for full details.

Quick steps:

```bash
# 1. Generate a new secret
./scripts/rotate-jwt-secret.sh

# 2. Set in your deployment environment:
JWT_SECRET_CURRENT=<new secret>
JWT_SECRET_PREVIOUS=<current secret>

# 3. Deploy to all instances
# 4. Wait 7 days (refresh token lifetime)
# 5. Remove JWT_SECRET_PREVIOUS
```

## Log Scrubbing

The `Config` struct implements `slog.LogValuer` so that all sensitive fields are **automatically masked** when the config is logged:

```go
// Safe — secrets are automatically redacted by LogValue()
slog.Info("server started", "config", cfg)

// Also safe — LogSummary() explicitly masks all secret fields
for k, v := range cfg.LogSummary() {
    slog.Info("config", k, v)
}
```

The following fields are masked as `****` or `<field[:4]>****` in all log output:
- `JWT_SECRET`, `JWT_SECRET_CURRENT`, `JWT_SECRET_PREVIOUS`
- `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`
- `STRIPE_API_KEY`, `STRIPE_WEBHOOK_SECRET`
- `MAPTILER_API_KEY`
- `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`
- `INTERNAL_SERVICE_TOKEN`
- Passwords embedded in `DATABASE_URL` and `REDIS_URL`

**Never log raw secret values.** If a structured log statement requires a secret, use `maskSecret()` or replace with a placeholder.

## CI/CD Integration

### GitHub Actions Secrets

Store all production secrets in **GitHub Actions → Settings → Secrets and variables → Actions**.

```yaml
# Reference secrets in workflows — never hard-code values
env:
  JWT_SECRET_CURRENT: ${{ secrets.JWT_SECRET_CURRENT }}
  DATABASE_URL: ${{ secrets.DATABASE_URL }}
```

GitHub Actions automatically masks secrets in workflow logs. For dynamically constructed secrets, add explicit masking:

```yaml
- name: Mask dynamic secret
  run: echo "::add-mask::${{ steps.get-secret.outputs.value }}"
```

### Automated Secret Scanning

The [`secret-scan.yml`](../.github/workflows/secret-scan.yml) workflow runs [Gitleaks](https://github.com/gitleaks/gitleaks) on every pull request and push to `main`/`develop` to detect accidentally committed secrets. CI will fail if any secrets are detected.

To add patterns for project-specific secrets, create a `.gitleaks.toml` at the repository root:

```toml
[allowlist]
  description = "Global allow list"
  # Allowlist known safe patterns (e.g., example/test values)
  regexes = [
    "supersecret32characterlongvalue!",  # test fixture
  ]
```

## Secret Access Audit Log

Secret access is not logged directly (logging secrets would defeat the purpose). However, **access to resources protected by secrets** is recorded in the audit log (see [AUDIT_LOGGING_IMPLEMENTATION.md](./AUDIT_LOGGING_IMPLEMENTATION.md)):

- Authentication attempts: `user_login`, `user_logout`
- Admin panel access: `view_admin_panel`, `admin_login`, `admin_action`
- Payment operations: `payment_create`, `payment_success`, `payment_failure`

Review audit logs regularly for suspicious authentication patterns (e.g., unusual login times, geographic anomalies, repeated failures).

---

## 🚨 Emergency: Secret Compromise Runbook

Follow this runbook **immediately** if a secret is suspected or confirmed to be compromised.

### Step 1: Assess the Scope

Determine which secret was exposed:
- Was it committed to a public/private git repository?
- Was it exposed in logs, error messages, or API responses?
- Was it shared insecurely (email, Slack, etc.)?

### Step 2: Immediate Revocation

**Do this first, before anything else.**

| Secret Type | Action |
|-------------|--------|
| **JWT signing key** | Rotate using `./scripts/rotate-jwt-secret.sh`; all existing tokens become invalid after rotation window |
| **Database password** | Rotate via your database provider immediately; update `DATABASE_URL` in all environments |
| **Stripe API key** | Revoke at [Stripe Dashboard](https://dashboard.stripe.com/apikeys) and issue a new key |
| **Stripe webhook secret** | Regenerate at [Stripe Dashboard](https://dashboard.stripe.com/webhooks) |
| **LiveKit credentials** | Revoke at [LiveKit Cloud](https://cloud.livekit.io) and issue new credentials |
| **MapTiler API key** | Revoke at [MapTiler Cloud](https://cloud.maptiler.com/account/keys/) and issue a new key |
| **R2 credentials** | Revoke at [Cloudflare Dashboard](https://dash.cloudflare.com) and generate new R2 API tokens |
| **Internal service token** | Generate a new random value and deploy; old value immediately rejected |

### Step 3: Deploy Updated Secrets

1. Generate new secret value (use `openssl rand -base64 32` for random secrets)
2. Update the secret in your secrets manager / GitHub Actions Secrets
3. Deploy to **all instances simultaneously** to minimise downtime
4. Verify the application starts and authenticates correctly

### Step 4: Investigate and Remediate

1. **Check git history** for committed secrets:
   ```bash
   git log --all --full-history -- "*.env" "*.env.*"
   git grep -i "secret\|password\|api_key" $(git log --all --oneline | awk '{print $1}')
   ```
2. **Purge from git history** if committed (requires force-push to all branches — coordinate with the team):
   ```bash
   # Use git-filter-repo (preferred) or BFG Repo Cleaner
   git filter-repo --path configs/dev.env --invert-paths
   ```
3. **Review access logs** for the compromised service during the exposure window
4. **Notify affected parties** (users, partners) if their data may have been accessed

### Step 5: Post-Incident Review

After the immediate response:

1. Document the incident: what happened, timeline, impact, resolution
2. Update secret rotation schedule if rotation was overdue
3. Add the exposed pattern to Gitleaks configuration to prevent recurrence
4. Conduct a broader audit of other secrets for similar exposure risks

### Step 6: Rotate Related Secrets (Precautionary)

If the exposure mechanism could have affected other secrets (e.g., a compromised developer machine or CI environment), rotate all secrets proactively.

---

## Rotation Schedule Reference

| Secret | Interval | Next Action |
|--------|----------|-------------|
| JWT signing key | Every 90 days | Run `./scripts/rotate-jwt-secret.sh` |
| Database password | Every 6 months | Coordinate with DBA; update `DATABASE_URL` |
| All API keys | Annually (minimum) | Review each service dashboard |

Track rotations in your team's security calendar or ticketing system.
