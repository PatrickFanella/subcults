# JWT Secret Rotation Guide

This guide explains how to perform zero-downtime JWT secret rotation for the Subcults API.

## Overview

JWT secret rotation allows you to update signing keys without disrupting active user sessions. The implementation supports dual-key rotation: new tokens are signed with the current key, while both current and previous keys can validate tokens.

## Key Concepts

### Dual-Key System

- **Current Secret** (`JWT_SECRET_CURRENT`): Used to sign all new tokens
- **Previous Secret** (`JWT_SECRET_PREVIOUS`): Used to validate tokens signed before rotation
- **Rotation Window**: Period during which both keys are active (recommended: 7 days)
- **Key Version Tracking**: Tokens include a `kid` (key ID) header field set to "current" to identify which key version signed the token

### Token Lifetimes

- **Access tokens**: 15 minutes
- **Refresh tokens**: 7 days

The rotation window should be at least as long as the refresh token lifetime to ensure all old tokens expire naturally.

## Prerequisites

- Access to production environment configuration
- Ability to update environment variables
- Ability to deploy updated configuration
- `openssl` for generating secrets (or use the provided script)

## Rotation Process

### Step 1: Generate New Secret

Use the provided script to generate a new secret:

```bash
./scripts/rotate-jwt-secret.sh
```

Or generate manually:

```bash
openssl rand -base64 32
```

**Security Note**: Store the generated secret securely. Never commit secrets to version control.

### Step 2: Update Environment Variables

Update your deployment configuration with the following environment variables:

```bash
# New key for signing tokens
JWT_SECRET_CURRENT=<new-secret-from-step-1>

# Current key (keep old tokens valid)
JWT_SECRET_PREVIOUS=<your-current-secret>
```

**For Kubernetes/Docker deployments**:

```yaml
env:
  - name: JWT_SECRET_CURRENT
    valueFrom:
      secretKeyRef:
        name: jwt-secrets
        key: current
  - name: JWT_SECRET_PREVIOUS
    valueFrom:
      secretKeyRef:
        name: jwt-secrets
        key: previous
```

**For AWS/Cloud deployments**: Update secrets in your secret manager (AWS Secrets Manager, GCP Secret Manager, etc.)

### Step 3: Deploy Configuration

Deploy the updated configuration to all API server instances:

```bash
# Example: Rolling update deployment
kubectl rollout restart deployment/api-server

# Example: Docker Compose
docker-compose up -d --force-recreate api

# Verify all instances are running with new config
kubectl get pods -l app=api-server
```

**During Deployment**:
- ✅ New tokens signed with `JWT_SECRET_CURRENT`
- ✅ Old tokens validated with `JWT_SECRET_PREVIOUS`
- ✅ Zero downtime for active users
- ✅ No user sessions interrupted

### Step 4: Wait for Rotation Window

Wait for the maximum token lifetime to pass. This ensures all tokens signed with the previous key expire naturally.

**Recommended Wait Times**:
- **Minimum**: 7 days (refresh token expiry)
- **Recommended**: 7-14 days (allows for extended sessions)
- **Conservative**: 30 days (covers any edge cases)

**Monitoring During Rotation**:

During the rotation window, monitor authentication behavior using your existing logs and metrics:

- Check API logs for any increase in authentication failures or unexpected "invalid token" errors
- Use your observability stack (Prometheus/Grafana or equivalent) to watch auth-related error rates and latency
- Confirm that overall login and refresh success rates remain stable before and after the rotation
- Monitor for any unusual patterns in token validation that might indicate rotation issues

Example monitoring approaches:
```bash
# Watch for authentication errors in logs
tail -f /var/log/api.log | grep -i "auth\|token"

# Check overall API error rates (adjust metric names to match your setup)
curl http://localhost:9090/metrics | grep -E "http_requests_total|http_request_duration"
```

### Step 5: Complete Rotation

After the rotation window, remove the previous secret:

```bash
# Keep only the current secret
JWT_SECRET_CURRENT=<secret-from-step-1>

# Remove or leave empty
JWT_SECRET_PREVIOUS=
```

Deploy this final configuration. At this point:
- Only tokens signed with the current key are valid
- Previous key is no longer used
- Rotation is complete

## Backward Compatibility

### Legacy Single-Key Setup

If you're currently using `JWT_SECRET` (legacy single-key setup), you can migrate gradually:

**Current Setup**:
```bash
JWT_SECRET=my-current-secret
```

**Migrate to Rotation-Ready**:
```bash
JWT_SECRET_CURRENT=my-current-secret
# Leave JWT_SECRET for backward compatibility during transition
JWT_SECRET=my-current-secret
```

**After All Services Updated**:
```bash
# Remove legacy JWT_SECRET
JWT_SECRET_CURRENT=my-current-secret
```

The system will prioritize `JWT_SECRET_CURRENT` over `JWT_SECRET` when both are present.

## Rotation Script Usage

The `scripts/rotate-jwt-secret.sh` script automates the rotation preparation:

```bash
# Run the script
./scripts/rotate-jwt-secret.sh

# Output includes:
# - Generated new secret
# - Current secret (masked)
# - Step-by-step instructions
# - Environment variable configuration
```

**Script Features**:
- Generates cryptographically secure 32-character base64 secret
- Detects current configuration
- Provides customized instructions based on setup
- Color-coded output for easy reading
- Security best practices reminder

## Automated Rotation (CI/CD)

For automated deployments, integrate rotation into your CI/CD pipeline:

### Example: GitHub Actions

```yaml
name: Rotate JWT Secret

on:
  schedule:
    - cron: '0 0 1 * *'  # Monthly on 1st day
  workflow_dispatch:  # Manual trigger

jobs:
  rotate:
    runs-on: ubuntu-latest
    steps:
      - name: Generate new secret
        id: secret
        run: |
          NEW_SECRET=$(openssl rand -base64 32)
          echo "::add-mask::$NEW_SECRET"
          echo "new_secret=$NEW_SECRET" >> $GITHUB_OUTPUT

      - name: Update secrets in AWS
        run: |
          # Move current to previous
          CURRENT=$(aws secretsmanager get-secret-value --secret-id jwt-current --query SecretString --output text)
          aws secretsmanager update-secret --secret-id jwt-previous --secret-string "$CURRENT"
          
          # Set new current
          aws secretsmanager update-secret --secret-id jwt-current --secret-string "${{ steps.secret.outputs.new_secret }}"

      - name: Deploy updated config
        run: |
          # Trigger deployment with new secrets
          kubectl rollout restart deployment/api-server
```

### Example: Terraform

```hcl
resource "random_password" "jwt_secret_current" {
  length  = 32
  special = true
}

resource "aws_secretsmanager_secret_version" "jwt_current" {
  secret_id     = aws_secretsmanager_secret.jwt_current.id
  secret_string = random_password.jwt_secret_current.result
}

# Retain previous version for rotation
resource "aws_secretsmanager_secret_version" "jwt_previous" {
  secret_id     = aws_secretsmanager_secret.jwt_previous.id
  secret_string = data.aws_secretsmanager_secret_version.jwt_current_previous.secret_string
}
```

## Security Best Practices

### 1. Secret Generation
- Use cryptographically secure random generation
- Minimum 32 characters (256 bits)
- Base64 encoding for compatibility

### 2. Secret Storage
- Never commit secrets to version control
- Use secret management systems (AWS Secrets Manager, HashiCorp Vault, etc.)
- Encrypt secrets at rest
- Limit access with IAM policies

### 3. Rotation Frequency
- **Monthly**: Standard practice for production
- **Quarterly**: Acceptable for low-risk environments
- **Immediately**: After any suspected compromise
- **On-demand**: When team members with access leave

### 4. Monitoring
- Log all authentication events
- Monitor for unusual patterns during rotation
- Alert on authentication failures spike
- Track key usage metrics

### 5. Audit Trail
- Document each rotation event
- Record who initiated rotation
- Track rotation window durations
- Maintain rotation schedule

## Troubleshooting

### Issue: Tokens Not Validating After Rotation

**Symptoms**: Users getting 401 Unauthorized after deployment

**Causes**:
1. `JWT_SECRET_PREVIOUS` not set correctly
2. Deployment didn't propagate to all instances
3. Secrets not loaded by application

**Resolution**:
```bash
# Verify environment variables
kubectl exec -it api-server-pod -- env | grep JWT_SECRET

# Check logs for validation errors
kubectl logs api-server-pod | grep "jwt_validation_error"

# Rollback if necessary
kubectl rollout undo deployment/api-server
```

### Issue: Old Tokens Still Validating After Rotation Complete

**Symptoms**: Tokens signed with removed key still work

**Causes**:
1. `JWT_SECRET_PREVIOUS` still set in some instances
2. Configuration cache not cleared
3. Old deployment pods still running

**Resolution**:
```bash
# Force restart all pods
kubectl rollout restart deployment/api-server

# Verify configuration
kubectl get pods -o jsonpath='{.items[*].spec.containers[0].env}' | grep JWT_SECRET
```

### Issue: Rotation Script Fails

**Symptoms**: `rotate-jwt-secret.sh` exits with error

**Causes**:
1. `openssl` not installed
2. Insufficient permissions
3. Environment variables not accessible

**Resolution**:
```bash
# Install openssl
sudo apt-get install openssl  # Debian/Ubuntu
brew install openssl          # macOS

# Generate manually
openssl rand -base64 32
```

## Testing Rotation

Before performing production rotation, test in a staging environment:

### 1. Create Test Tokens

```bash
# Generate tokens before rotation
curl -X POST http://staging-api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'

# Save the tokens
export OLD_ACCESS_TOKEN="<token>"
export OLD_REFRESH_TOKEN="<token>"
```

### 2. Perform Rotation

Follow steps 1-3 of the rotation process in staging.

### 3. Verify Old Tokens Work

```bash
# Validate old tokens still work
curl http://staging-api/protected \
  -H "Authorization: Bearer $OLD_ACCESS_TOKEN"

# Should return 200 OK
```

### 4. Generate New Tokens

```bash
# Refresh to get new tokens
curl -X POST http://staging-api/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$OLD_REFRESH_TOKEN\"}"

export NEW_ACCESS_TOKEN="<token>"
```

### 5. Complete Rotation

Remove `JWT_SECRET_PREVIOUS` and verify:
- New tokens still work
- Old tokens (beyond expiry) are rejected
- New token generation uses only current key

## Additional Resources

- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [NIST Key Management Guidelines](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)

## Support

For questions or issues with JWT rotation:
- Open an issue: https://github.com/subculture-collective/subcults/issues
- Security concerns: security@subcults.com (use PGP key)
