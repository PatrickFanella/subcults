# Configuration & Environment Variables

This document provides a comprehensive reference for all configuration options and environment variables in the Subcults platform.

## Table of Contents

- [Overview](#overview)
- [Required Environment Variables](#required-environment-variables)
- [Optional Environment Variables](#optional-environment-variables)
- [Feature Flags](#feature-flags)
- [Secret Key Rotation](#secret-key-rotation)
- [Third-Party Service Configuration](#third-party-service-configuration)
- [Database Configuration](#database-configuration)
- [Logging Configuration](#logging-configuration)
- [Development vs Production](#development-vs-production)
- [Configuration Examples](#configuration-examples)

## Overview

Subcults uses environment variables for configuration, following the [12-factor app](https://12factor.net/) principles. Configuration is loaded via the [koanf](https://github.com/knadh/koanf) library, which supports multiple sources with the following precedence (highest to lowest):

1. Environment variables (highest priority)
2. YAML configuration file (optional, if provided)
3. Default values (lowest priority)

### Configuration Files

- **Template**: `configs/dev.env.example` - Example configuration with all variables documented
- **Your config**: `configs/dev.env` - Copy from example and fill in your values (git-ignored)
- **Loader**: `internal/config/config.go` - Configuration loading and validation logic

## Required Environment Variables

These variables **must** be set for the application to start successfully.

### Core Configuration

#### `DATABASE_URL`
- **Description**: Neon Postgres connection string with PostGIS extension
- **Type**: String (URL)
- **Format**: `postgres://user:password@host:port/database?sslmode=require`
- **Example**: `postgres://subcults:password@localhost:5432/subcults?sslmode=disable`
- **Validation**: Must be a valid Postgres connection string
- **When to override**: Always required; use local values for dev, production credentials for prod

#### `JWT_SECRET`
- **Description**: Secret key for signing JWT access and refresh tokens
- **Type**: String
- **Minimum length**: 32 characters (recommended for security)
- **Example**: Generate with `openssl rand -base64 32`
- **Validation**: Cannot be empty; longer is more secure
- **When to override**: Always required; use different secrets for dev/staging/prod environments
- **Security**: Never commit to version control; rotate periodically (see [Secret Key Rotation](#secret-key-rotation))

### LiveKit (WebRTC Audio/Video)

Cloud WebRTC Selective Forwarding Unit (SFU) for live audio sessions. Get credentials from [https://livekit.io/](https://livekit.io/).

#### `LIVEKIT_URL`
- **Description**: LiveKit server WebSocket URL
- **Type**: String (URL)
- **Format**: `wss://your-project.livekit.cloud`
- **Example**: `wss://subcults-dev.livekit.cloud`
- **Validation**: Must be a valid WebSocket URL
- **When to override**: Always required; use separate projects for dev/staging/prod

#### `LIVEKIT_API_KEY`
- **Description**: LiveKit API key for server-side operations
- **Type**: String
- **Example**: `APIabcdef123456`
- **Validation**: Cannot be empty
- **When to override**: Always required; use project-specific keys
- **Security**: Keep secret; obtain from LiveKit dashboard

#### `LIVEKIT_API_SECRET`
- **Description**: LiveKit API secret for token generation
- **Type**: String
- **Example**: (64-character hex string)
- **Validation**: Cannot be empty
- **When to override**: Always required; use project-specific secrets
- **Security**: Keep secret; obtain from LiveKit dashboard

### Stripe (Payments)

Payment processing for tickets, merch, and scene payouts via Stripe Connect. Get credentials from [https://dashboard.stripe.com/apikeys](https://dashboard.stripe.com/apikeys).

#### `STRIPE_API_KEY`
- **Description**: Stripe secret API key
- **Type**: String
- **Format**: Starts with `sk_test_` (test mode) or `sk_live_` (live mode)
- **Example**: `sk_test_51A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6`
- **Validation**: Cannot be empty; must be a valid Stripe secret key
- **When to override**: Always required; use test keys in dev, live keys in production
- **Security**: Never commit; rotate if exposed

#### `STRIPE_WEBHOOK_SECRET`
- **Description**: Stripe webhook signing secret for event verification
- **Type**: String
- **Format**: Starts with `whsec_`
- **Example**: `whsec_abc123def456ghi789jkl012mno345pqr678stu901vwx234yz`
- **Validation**: Cannot be empty
- **When to override**: Always required; different secrets for each environment
- **Get from**: [https://dashboard.stripe.com/webhooks](https://dashboard.stripe.com/webhooks)
- **Security**: Keep secret; used to verify webhook authenticity

#### `STRIPE_ONBOARDING_RETURN_URL`
- **Description**: URL to redirect users after successful Stripe Connect onboarding
- **Type**: String (URL)
- **Format**: `https://yourdomain.com/stripe/return`
- **Example**: `https://localhost:3000/stripe/return` (dev), `https://subcults.com/stripe/return` (prod)
- **Validation**: Cannot be empty; must be a valid HTTPS URL (HTTP allowed in dev)
- **When to override**: Always required; use environment-specific URLs

#### `STRIPE_ONBOARDING_REFRESH_URL`
- **Description**: URL to redirect users to continue incomplete Stripe Connect onboarding
- **Type**: String (URL)
- **Format**: `https://yourdomain.com/stripe/refresh`
- **Example**: `https://localhost:3000/stripe/refresh` (dev), `https://subcults.com/stripe/refresh` (prod)
- **Validation**: Cannot be empty; must be a valid HTTPS URL (HTTP allowed in dev)
- **When to override**: Always required; use environment-specific URLs

### MapTiler (Map Tiles)

Map tiles for the MapLibre frontend. Get API key from [https://cloud.maptiler.com/account/keys/](https://cloud.maptiler.com/account/keys/).

#### `MAPTILER_API_KEY`
- **Description**: MapTiler API key for tile requests
- **Type**: String
- **Example**: `abc123DEF456ghi789JKL012mno345PQR`
- **Validation**: Cannot be empty
- **When to override**: Always required; can use same key across environments for dev
- **Security**: Rate-limited by MapTiler; avoid exposing in client-side code

### Jetstream (AT Protocol)

Real-time AT Protocol firehose for decentralized data ingestion. Used by the indexer service.

#### `JETSTREAM_URL`
- **Description**: Jetstream WebSocket endpoint for AT Protocol subscription
- **Type**: String (WebSocket URL)
- **Default**: `wss://jetstream1.us-east.bsky.network/subscribe`
- **Example**: `wss://jetstream2.us-west.bsky.network/subscribe` (alternative region)
- **Validation**: If set, must be a valid WebSocket URL
- **When to override**: Use default unless connecting to a custom Jetstream instance
- **Note**: Used only by the indexer service. If `JETSTREAM_URL` is not set, the indexer falls back to the default value above. The API server does not use this variable; any validation that marks it as required for the API is a known config validation bug, not a requirement to set this env var.

## Optional Environment Variables

These variables have defaults and can be omitted in most cases.

### Server Configuration

#### `SUBCULT_PORT` / `PORT`
- **Description**: API server HTTP port
- **Type**: Integer
- **Default**: `8080`
- **Aliases**: `SUBCULT_PORT` (preferred), `PORT` (fallback for compatibility)
- **Example**: `8080` (dev), `3000` (alternative)
- **Validation**: Must be a valid integer between 1 and 65535
- **When to override**: When port 8080 is in use or deploying to platforms with specific port requirements

#### `SUBCULT_ENV` / `ENV` / `GO_ENV`
- **Description**: Environment mode affecting logging verbosity and feature behavior
- **Type**: String (enum)
- **Default**: `development`
- **Valid values**: `development`, `staging`, `production`
- **Aliases**: `SUBCULT_ENV` (preferred), `ENV`, `GO_ENV` (fallback)
- **Example**: `production`
- **When to override**: Set to `production` in production deployments for optimized logging
- **Effects**:
  - `development` / `staging`: Text-formatted logs, debug level enabled
  - `production`: JSON-formatted logs, info level default, optimized for log aggregators

### Storage (Cloudflare R2)

S3-compatible object storage for images, audio, and videos. **All R2 variables are optional**; if any R2 variable is set, all four must be provided.

#### `R2_BUCKET_NAME`
- **Description**: R2 bucket name for media assets
- **Type**: String
- **Example**: `subcults-media-dev`, `subcults-media-prod`
- **Validation**: Required if any other R2 variable is set
- **When to override**: When enabling media upload functionality
- **Get from**: Cloudflare R2 dashboard after creating a bucket

#### `R2_ACCESS_KEY_ID`
- **Description**: R2 access key ID for S3 API authentication
- **Type**: String
- **Example**: (32-character hex string)
- **Validation**: Required if any other R2 variable is set
- **When to override**: When enabling media upload functionality
- **Security**: Keep secret; obtain from Cloudflare R2 dashboard

#### `R2_SECRET_ACCESS_KEY`
- **Description**: R2 secret access key for S3 API authentication
- **Type**: String
- **Example**: (64-character base64 string)
- **Validation**: Required if any other R2 variable is set
- **When to override**: When enabling media upload functionality
- **Security**: Keep secret; obtain from Cloudflare R2 dashboard

#### `R2_ENDPOINT`
- **Description**: R2 endpoint URL for S3 API
- **Type**: String (URL)
- **Format**: `https://<account-id>.r2.cloudflarestorage.com`
- **Example**: `https://abc123def456.r2.cloudflarestorage.com`
- **Validation**: Required if any other R2 variable is set; must be a valid HTTPS URL
- **When to override**: When enabling media upload functionality
- **Get from**: Cloudflare R2 dashboard (account-specific)

#### `R2_MAX_UPLOAD_SIZE_MB`
- **Description**: Maximum file upload size in megabytes
- **Type**: Integer
- **Default**: `15`
- **Example**: `15` (default), `50` (larger uploads)
- **Validation**: Must be a positive integer
- **When to override**: When larger uploads are needed (consider storage costs)

### Rate Limiting (Redis)

Optional distributed rate limiting using Redis. If not set, in-memory rate limiting is used.

#### `REDIS_URL`
- **Description**: Redis connection URL for distributed rate limiting
- **Type**: String (URL)
- **Format**: `redis://user:password@host:port/db` or `redis://host:port`
- **Example**: `redis://localhost:6379`, `redis://:password@redis.example.com:6379/0`
- **Default**: Empty (uses in-memory rate limiting)
- **Validation**: Must be a valid Redis connection string if provided
- **When to override**: In multi-instance deployments where rate limits should be shared across instances
- **Note**: Optional; in-memory rate limiting works fine for single-instance deployments

### Observability & Metrics

#### `METRICS_PORT`
- **Description**: Prometheus metrics endpoint port (indexer service)
- **Type**: Integer
- **Default**: `9090`
- **Example**: `9090` (default), `9091` (alternative)
- **Validation**: Must be a valid integer between 1 and 65535
- **When to override**: When port 9090 is in use or running multiple indexers
- **Endpoint**: `http://localhost:9090/internal/indexer/metrics`

#### `INTERNAL_AUTH_TOKEN`
- **Description**: Internal authentication token for metrics endpoint
- **Type**: String
- **Default**: Empty (no authentication)
- **Example**: Generate with `openssl rand -hex 32`
- **Validation**: None (any non-empty string)
- **When to override**: **Recommended for production** to secure metrics endpoint
- **Security**: Keep secret; use strong random tokens
- **Usage**: Send as `Authorization: Bearer <token>` header when accessing metrics

### Payments Configuration

#### `STRIPE_APPLICATION_FEE_PERCENT`
- **Description**: Platform fee percentage on Stripe Connect transactions
- **Type**: Float
- **Default**: `5.0` (5% platform fee)
- **Example**: `5.0` (5%), `7.5` (7.5%)
- **Validation**: Must be a valid float; typically 0.0 to 30.0
- **When to override**: To adjust platform revenue model
- **Note**: This is the percentage taken from each transaction before payout to the scene

## Feature Flags

Feature flags control experimental or optional functionality. All feature flags are **optional** and default to `false`.

### `RANK_TRUST_ENABLED`
- **Description**: Enable trust-weighted ranking in search and feed endpoints
- **Type**: Boolean
- **Default**: `false`
- **Valid values**: `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off` (case-insensitive)
- **Example**: `true`
- **When to enable**: When trust graph has sufficient data and the feature is ready for users
- **Effects**:
  - `true`: Search results include trust score weighting (composite score includes `trust_weight * 0.1`)
  - `false`: Trust score is excluded from ranking (fallback to text/proximity/recency only)
- **Performance impact**: Minimal; trust scores are pre-computed
- **Related**: Trust graph computation, alliance system

### `TRACING_ENABLED`
- **Description**: Enable OpenTelemetry distributed tracing
- **Type**: Boolean
- **Default**: `false`
- **Valid values**: `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off` (case-insensitive)
- **Example**: `true`
- **When to enable**: For debugging, performance analysis, or production observability
- **Effects**: Exports trace spans to configured OTLP endpoint
- **Performance impact**: Low overhead with proper sampling rates
- **Prerequisites**: Requires OTLP collector (e.g., Jaeger, Grafana Tempo)

### `TRACING_EXPORTER_TYPE`
- **Description**: Tracing exporter protocol type
- **Type**: String (enum)
- **Default**: `otlp-http`
- **Valid values**: `otlp-http`, `otlp-grpc`
- **Example**: `otlp-http` (default), `otlp-grpc` (for gRPC endpoints)
- **When to override**: When OTLP collector requires gRPC protocol
- **Note**: Most modern OTLP collectors support both HTTP and gRPC

### `TRACING_OTLP_ENDPOINT`
- **Description**: OTLP endpoint URL for trace export
- **Type**: String (URL)
- **Default**: Empty (uses OTLP default endpoint)
- **Examples**:
  - Jaeger (HTTP): `http://localhost:4318`
  - Jaeger (gRPC): `localhost:4317`
  - Production: `https://your-collector.example.com:4318`
- **When to override**: Always when `TRACING_ENABLED=true`
- **Note**: Must match the protocol specified in `TRACING_EXPORTER_TYPE`

### `TRACING_SAMPLE_RATE`
- **Description**: Sampling rate for traces (0.0 to 1.0)
- **Type**: Float
- **Default**: `0.1` (10% sampling)
- **Examples**:
  - `1.0`: 100% of traces sampled (recommended for development)
  - `0.1`: 10% of traces sampled (recommended for production)
  - `0.01`: 1% of traces sampled (high-traffic production)
- **Validation**: Must be between 0.0 and 1.0
- **When to override**: Adjust based on traffic volume and observability needs
- **Performance impact**: Higher sampling increases overhead and storage costs

### `TRACING_INSECURE`
- **Description**: Disable TLS for OTLP connection (development only)
- **Type**: Boolean
- **Default**: `false`
- **Valid values**: `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off` (case-insensitive)
- **Example**: `true` (for local Jaeger without TLS)
- **When to enable**: Only in development with local OTLP collectors
- **Security**: **Never enable in production**; TLS is required for secure trace transmission

### Web Push Notifications (Frontend Only)

#### `VITE_VAPID_PUBLIC_KEY`
- **Description**: VAPID public key for Web Push notifications (frontend environment variable)
- **Type**: String (base64 URL-safe)
- **Format**: Starts with 'B' followed by base64 characters
- **Example**: `BEl62iUYgUivxIkv69yViEuiBIa-Ib37J8xQmr8Db5s...` (truncated)
- **Generate**: `npx web-push generate-vapid-keys`
- **When to override**: Always required for Web Push functionality
- **Security**: Public key is safe to expose in frontend code; keep the **private key** secret on the backend
- **Note**: This is a **frontend** environment variable (prefixed with `VITE_`), not used by the Go backend

## Secret Key Rotation

Regular secret rotation is a critical security practice. This section provides procedures for rotating sensitive credentials without service interruption.

### JWT Secret Rotation

The JWT secret is used to sign access and refresh tokens. Rotating it requires a dual-key strategy to avoid invalidating all active sessions.

#### Dual-Key Rotation Strategy

The current implementation supports **single-key** JWT signing. To implement zero-downtime rotation, the following strategy is recommended:

1. **Preparation Phase**:
   - Generate new JWT secret: `openssl rand -base64 32`
   - Add support for dual-key validation in `internal/auth/jwt.go` (future enhancement)

2. **Current Single-Key Rotation** (service interruption):
   - **Before rotation**: Announce to users that sessions will be invalidated
   - Update `JWT_SECRET` environment variable with new value
   - Restart API servers
   - **After rotation**: All users must log in again (existing tokens are invalid)

3. **Recommended: Dual-Key Rotation** (zero downtime, requires implementation):
   - Modify `internal/auth/jwt.go` to support `JWT_SECRET` (new) and `JWT_SECRET_OLD` (legacy)
   - Deploy code supporting dual keys
   - Set `JWT_SECRET_OLD` to current secret
   - Set `JWT_SECRET` to new secret
   - Restart API servers (tokens signed with old key remain valid)
   - Wait for old tokens to expire (7 days for refresh tokens)
   - Remove `JWT_SECRET_OLD` from configuration
   - Restart API servers

#### Rotation Frequency
- **Development**: Rotate when compromised or every 90 days
- **Production**: Rotate every 30-90 days or immediately if compromised
- **Post-incident**: Immediately if secret is exposed in logs, version control, or external systems

### Database Credentials Rotation

Neon Postgres supports credential rotation with minimal downtime.

#### Procedure

1. **Create new database user** (via Neon dashboard or SQL):
   ```sql
   CREATE USER subcults_new WITH PASSWORD 'new_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE subcults TO subcults_new;
   GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO subcults_new;
   GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO subcults_new;
   ```

2. **Update connection string**:
   - Update `DATABASE_URL` with new credentials
   - Deploy configuration update
   - Restart services

3. **Verify connectivity**:
   - Monitor logs for database connection errors
   - Verify application functionality

4. **Revoke old credentials** (after verification):
   ```sql
   REVOKE ALL PRIVILEGES ON DATABASE subcults FROM subcults_old;
   DROP USER subcults_old;
   ```

#### Rotation Frequency
- **Production**: Every 90 days or when compromised
- **Neon-specific**: Use Neon's built-in credential rotation features

### Third-Party Service Key Rotation

#### LiveKit API Keys

1. **Generate new API key** in LiveKit dashboard
2. Update `LIVEKIT_API_KEY` and `LIVEKIT_API_SECRET`
3. Restart API servers
4. **Note**: Active streams may be interrupted; schedule during low-traffic periods

#### Stripe API Keys

1. **Generate new secret key** in Stripe dashboard
2. Update `STRIPE_API_KEY`
3. Update `STRIPE_WEBHOOK_SECRET` if webhook endpoints changed
4. Restart API servers
5. **Verify webhooks** are being received successfully
6. **Revoke old keys** in Stripe dashboard after verification

#### Cloudflare R2 Keys

1. **Generate new access key** in Cloudflare R2 dashboard
2. Update `R2_ACCESS_KEY_ID` and `R2_SECRET_ACCESS_KEY`
3. Restart API servers
4. **Verify uploads** are working
5. **Revoke old keys** in Cloudflare dashboard

#### MapTiler API Key

1. **Generate new API key** in MapTiler dashboard
2. Update `MAPTILER_API_KEY`
3. Restart API servers
4. **Verify map tiles** are loading in frontend
5. **Delete old key** in MapTiler dashboard

### Internal Auth Token Rotation

The `INTERNAL_AUTH_TOKEN` protects the metrics endpoint.

1. **Generate new token**: `openssl rand -hex 32`
2. Update `INTERNAL_AUTH_TOKEN` in configuration
3. Restart indexer service
4. Update monitoring tools (Prometheus, Grafana) with new token

## Third-Party Service Configuration

### LiveKit WebRTC Setup

LiveKit provides real-time audio streaming for live performances and events.

#### Getting Started

1. **Sign up** at [https://livekit.io/](https://livekit.io/)
2. **Create a project** (e.g., "subcults-dev", "subcults-prod")
3. **Copy credentials** from the project dashboard:
   - WebSocket URL (e.g., `wss://subcults-dev.livekit.cloud`)
   - API Key (e.g., `APIabcdef123456`)
   - API Secret (64-character hex string)
4. **Set environment variables**:
   ```bash
   LIVEKIT_URL=wss://subcults-dev.livekit.cloud
   LIVEKIT_API_KEY=APIabcdef123456
   LIVEKIT_API_SECRET=your-secret-here
   ```

#### Configuration

- **Separate projects** for dev/staging/prod (recommended)
- **TURN server**: Included in LiveKit Cloud (no additional config)
- **Room settings**: Configured via API calls (see `internal/livekit/`)
- **Token generation**: Handled by `internal/livekit/token.go`

#### Monitoring

- **LiveKit dashboard**: Real-time room and participant metrics
- **Prometheus metrics**: Exposed by indexer (participant counts, room durations)

### Stripe Connect Configuration

Stripe Connect enables direct payouts to scenes while collecting platform fees.

#### Getting Started

1. **Sign up** at [https://stripe.com/](https://stripe.com/)
2. **Enable Stripe Connect** in dashboard settings
3. **Create webhook endpoint** at [https://dashboard.stripe.com/webhooks](https://dashboard.stripe.com/webhooks):
   - **URL**: `https://yourdomain.com/api/v1/stripe/webhook`
   - **Events**: `account.updated`, `checkout.session.completed`, etc. (see `internal/payment/webhook.go`)
4. **Copy credentials**:
   - Secret API key (starts with `sk_test_` or `sk_live_`)
   - Webhook signing secret (starts with `whsec_`)
5. **Set environment variables**:
   ```bash
   STRIPE_API_KEY=sk_test_...
   STRIPE_WEBHOOK_SECRET=whsec_...
   STRIPE_ONBOARDING_RETURN_URL=https://yourdomain.com/stripe/return
   STRIPE_ONBOARDING_REFRESH_URL=https://yourdomain.com/stripe/refresh
   STRIPE_APPLICATION_FEE_PERCENT=5.0
   ```

#### Configuration

- **Test mode**: Use `sk_test_` keys in development
- **Live mode**: Use `sk_live_` keys in production (requires Stripe account activation)
- **Connect account type**: Express (simplified onboarding for scenes)
- **Platform fee**: Configurable via `STRIPE_APPLICATION_FEE_PERCENT` (default: 5%)

#### Webhook Validation

Stripe webhooks are verified using `STRIPE_WEBHOOK_SECRET` to prevent spoofing. See `internal/payment/webhook.go` for implementation.

### Cloudflare R2 Storage

R2 provides S3-compatible object storage for media uploads.

#### Getting Started

1. **Sign up** at [https://cloudflare.com/](https://cloudflare.com/)
2. **Navigate to R2** in Cloudflare dashboard
3. **Create a bucket** (e.g., "subcults-media-dev")
4. **Create API token**:
   - **Permissions**: Object Read & Write
   - **Bucket**: Select your bucket
5. **Copy credentials**:
   - Access Key ID
   - Secret Access Key
   - Endpoint URL (format: `https://<account-id>.r2.cloudflarestorage.com`)
6. **Set environment variables**:
   ```bash
   R2_BUCKET_NAME=subcults-media-dev
   R2_ACCESS_KEY_ID=your-access-key
   R2_SECRET_ACCESS_KEY=your-secret-key
   R2_ENDPOINT=https://abc123.r2.cloudflarestorage.com
   R2_MAX_UPLOAD_SIZE_MB=15
   ```

#### Configuration

- **Separate buckets** for dev/staging/prod (recommended)
- **CORS policy**: Configure in R2 dashboard to allow frontend uploads
- **Public access**: Configure bucket policy for public read (images, audio)
- **S3 compatibility**: R2 implements the S3 API, so standard S3 clients work

#### Storage Costs

- **Storage**: $0.015/GB/month
- **Class A operations** (writes): $4.50 per million
- **Class B operations** (reads): $0.36 per million
- **No egress fees** (major R2 advantage over S3)

### MapTiler Map Tiles

MapTiler provides map tiles for the MapLibre frontend.

#### Getting Started

1. **Sign up** at [https://cloud.maptiler.com/](https://cloud.maptiler.com/)
2. **Navigate to Account → Keys**
3. **Create API key** or copy existing key
4. **Set environment variable**:
   ```bash
   MAPTILER_API_KEY=your-key-here
   ```

#### Configuration

- **Free tier**: 100,000 tile requests/month (sufficient for development)
- **Rate limiting**: Enforced by MapTiler based on plan
- **Tile styles**: Configured in frontend (`web/src/config/map.ts`)

### Jetstream AT Protocol

Jetstream provides real-time AT Protocol event streaming.

#### Getting Started

Jetstream is a **public service** provided by Bluesky. No signup required.

1. **Use default endpoint**:
   ```bash
   JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe
   ```

2. **Alternative endpoints** (for redundancy):
   - `wss://jetstream2.us-west.bsky.network/subscribe`
   - `wss://jetstream1.us-west.bsky.network/subscribe`

#### Configuration

- **Filtering**: Configured in `internal/indexer/filter.go` (collection types, DIDs)
- **Reconnection**: Automatic with exponential backoff (see `internal/indexer/config.go`)
- **Metrics**: Prometheus metrics exposed on `METRICS_PORT` (indexer service)

## Database Configuration

### Neon Postgres with PostGIS

Subcults uses Neon Postgres 16 with the PostGIS extension for geospatial queries.

#### Getting Started

1. **Sign up** at [https://neon.tech/](https://neon.tech/)
2. **Create a project** (e.g., "subcults-dev")
3. **Enable PostGIS extension** (automatically enabled for new Neon projects)
4. **Copy connection string** from dashboard:
   ```
   postgres://user:password@ep-cool-name-123456.us-east-2.aws.neon.tech/neondb?sslmode=require
   ```
5. **Set environment variable**:
   ```bash
   DATABASE_URL=postgres://user:password@host/database?sslmode=require
   ```

#### Connection String Format

```
postgres://[user]:[password]@[host]:[port]/[database]?[parameters]
```

**Components**:
- `user`: Database username (created automatically by Neon)
- `password`: Database password (from Neon dashboard)
- `host`: Neon endpoint hostname (e.g., `ep-cool-name-123456.us-east-2.aws.neon.tech`)
- `port`: Port number (default: `5432`, usually omitted)
- `database`: Database name (default: `neondb`)
- `parameters`: Connection parameters (e.g., `sslmode=require`)

**SSL Mode**:
- **Production**: `sslmode=require` (always use TLS)
- **Development (local)**: `sslmode=disable` (for local Postgres without TLS)

#### Migration Procedures

Subcults uses [golang-migrate](https://github.com/golang-migrate/migrate) for database schema management.

**Apply migrations**:
```bash
export DATABASE_URL="postgres://user:password@host/database?sslmode=require"
make migrate-up
```

**Rollback last migration**:
```bash
make migrate-down
```

**Check migration version**:
```bash
./scripts/migrate.sh version
```

**Create new migration**:
```bash
./scripts/migrate.sh create -ext sql -dir migrations -seq <migration_name>
```

See `migrations/README.md` for detailed migration documentation.

#### PostGIS Configuration

PostGIS is used for geospatial queries (proximity search, location consent).

**Verify PostGIS installation**:
```sql
SELECT PostGIS_version();
```

**Key PostGIS functions used**:
- `ST_MakePoint(lon, lat)`: Create point geometry
- `ST_Distance(geog1, geog2)`: Calculate distance in meters
- `ST_DWithin(geog1, geog2, distance)`: Check if points are within distance
- `ST_GeogFromText('SRID=4326;POINT(lon lat)')`: Parse geography from WKT

See `migrations/000000_initial_schema.up.sql` for PostGIS column definitions.

## Logging Configuration

### Environment-Based Log Levels

Subcults uses Go's `log/slog` for structured logging with environment-based configuration.

#### Log Format

- **Development** (`SUBCULT_ENV=development`): Human-readable text format
  ```
  2024-01-15T10:30:45.123Z INFO server started port=8080 env=development
  ```

- **Production** (`SUBCULT_ENV=production`): JSON format for log aggregators
  ```json
  {"time":"2024-01-15T10:30:45.123Z","level":"INFO","msg":"server started","port":8080,"env":"production"}
  ```

#### Log Levels

- **Development**: `DEBUG` level enabled (verbose)
- **Production**: `INFO` level default (errors, warnings, info)

**Log level hierarchy** (lowest to highest severity):
1. `DEBUG`: Detailed debugging information
2. `INFO`: General informational messages
3. `WARN`: Warning messages (potential issues)
4. `ERROR`: Error messages (request failures, exceptions)

#### Structured Logging

All logs include contextual fields for filtering and analysis:

**Required fields** (HTTP requests):
- `request_id`: Unique identifier per request (via `X-Request-ID` header)
- `method`: HTTP method (GET, POST, etc.)
- `path`: Request path
- `status`: HTTP status code
- `latency_ms`: Request duration in milliseconds
- `size`: Response size in bytes

**Optional fields**:
- `user_did`: Authenticated user DID (if logged in)
- `error_code`: Application error code (for 4xx/5xx responses)
- `error`: Error message (for failures)

**Example**:
```json
{
  "time": "2024-01-15T10:30:45.123Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "req-abc123",
  "method": "GET",
  "path": "/api/v1/scenes/search",
  "status": 200,
  "latency_ms": 45,
  "size": 1024,
  "user_did": "did:plc:abc123"
}
```

### Metrics and Observability

#### Prometheus Metrics

The indexer service exposes Prometheus metrics on the `/internal/indexer/metrics` endpoint.

**Endpoint**: `http://localhost:9090/internal/indexer/metrics` (configurable via `METRICS_PORT`)

**Authentication**: Optional via `INTERNAL_AUTH_TOKEN` (recommended for production)

**Sample metrics**:
- `jetstream_messages_total`: Total messages received from Jetstream
- `jetstream_reconnections_total`: Total reconnection attempts
- `indexer_processing_duration_seconds`: Message processing latency histogram

**Scrape configuration** (Prometheus):
```yaml
scrape_configs:
  - job_name: 'subcults-indexer'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/internal/indexer/metrics'
    bearer_token: 'your-internal-auth-token'  # if INTERNAL_AUTH_TOKEN is set
```

#### OpenTelemetry Tracing

See [Feature Flags](#feature-flags) for tracing configuration (`TRACING_ENABLED`, etc.).

**Trace export flow**:
1. API server generates trace spans for each request
2. Spans are batched and exported to OTLP endpoint
3. OTLP collector (Jaeger, Tempo) stores traces
4. Query traces via collector UI (e.g., Jaeger UI)

**Trace attributes**:
- `http.method`, `http.route`, `http.status_code`
- `user.did` (if authenticated)
- `db.statement` (for database queries)
- Custom span attributes per service

## Development vs Production

### Development Configuration

**Characteristics**:
- Text-formatted logs (human-readable)
- Debug logging enabled
- Local database (`sslmode=disable`)
- Test API keys (Stripe `sk_test_`, etc.)
- Relaxed security (HTTP allowed for some URLs)
- Higher tracing sample rate (100%)

**Example `.env` (development)**:
```bash
# Core
SUBCULT_ENV=development
SUBCULT_PORT=8080
DATABASE_URL=postgres://subcults:password@localhost:5432/subcults?sslmode=disable

# Auth
JWT_SECRET=dev-secret-please-change-me-in-production-min-32-chars

# LiveKit (dev project)
LIVEKIT_URL=wss://subcults-dev.livekit.cloud
LIVEKIT_API_KEY=APIdevkey123
LIVEKIT_API_SECRET=devsecret123456789

# Stripe (test mode)
STRIPE_API_KEY=sk_test_123456789
STRIPE_WEBHOOK_SECRET=whsec_test123
STRIPE_ONBOARDING_RETURN_URL=http://localhost:3000/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=http://localhost:3000/stripe/refresh

# MapTiler
MAPTILER_API_KEY=devkey123

# Jetstream (default)
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# R2 (optional in dev)
# R2_BUCKET_NAME=
# R2_ACCESS_KEY_ID=
# R2_SECRET_ACCESS_KEY=
# R2_ENDPOINT=

# Feature Flags
RANK_TRUST_ENABLED=false

# Tracing (full sampling for dev)
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=http://localhost:4318
TRACING_SAMPLE_RATE=1.0
TRACING_INSECURE=true
```

### Production Configuration

**Characteristics**:
- JSON-formatted logs (for aggregators like Datadog, CloudWatch)
- Info-level logging (errors, warnings, info only)
- Managed database with TLS (`sslmode=require`)
- Live API keys (Stripe `sk_live_`, etc.)
- Strict security (HTTPS required, CORS allowlist)
- Lower tracing sample rate (10% or less)
- Secrets managed via secret manager (AWS Secrets Manager, HashiCorp Vault, etc.)

**Example `.env` (production)**:
```bash
# Core
SUBCULT_ENV=production
SUBCULT_PORT=8080
DATABASE_URL=postgres://user:password@ep-prod-123.us-east-2.aws.neon.tech/neondb?sslmode=require

# Auth (rotate every 30-90 days)
JWT_SECRET=prod-secret-rotate-regularly-min-32-chars-ABC123xyz==

# LiveKit (prod project)
LIVEKIT_URL=wss://subcults-prod.livekit.cloud
LIVEKIT_API_KEY=APIprodkey456
LIVEKIT_API_SECRET=prodsecret987654321

# Stripe (live mode)
STRIPE_API_KEY=sk_live_987654321
STRIPE_WEBHOOK_SECRET=whsec_live456
STRIPE_ONBOARDING_RETURN_URL=https://subcults.com/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=https://subcults.com/stripe/refresh
STRIPE_APPLICATION_FEE_PERCENT=5.0

# MapTiler
MAPTILER_API_KEY=prodkey456

# Jetstream (default or custom endpoint)
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# R2 (production bucket)
R2_BUCKET_NAME=subcults-media-prod
R2_ACCESS_KEY_ID=prodaccesskey
R2_SECRET_ACCESS_KEY=prodsecretkey
R2_ENDPOINT=https://prodaccount.r2.cloudflarestorage.com
R2_MAX_UPLOAD_SIZE_MB=15

# Redis (for distributed rate limiting)
REDIS_URL=redis://:password@redis.example.com:6379/0

# Metrics (with auth)
METRICS_PORT=9090
INTERNAL_AUTH_TOKEN=prod-metrics-token-rotate-regularly-ABC123==

# Feature Flags
RANK_TRUST_ENABLED=true

# Tracing (10% sampling)
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=https://otlp-collector.example.com:4318
TRACING_SAMPLE_RATE=0.1
TRACING_INSECURE=false
```

### Configuration Checklist

**Before deploying to production**:
- [ ] Set `SUBCULT_ENV=production`
- [ ] Use strong `JWT_SECRET` (min 32 chars, rotated regularly)
- [ ] Use production database with `sslmode=require`
- [ ] Use live Stripe keys (`sk_live_`)
- [ ] Use separate LiveKit project for production
- [ ] Use separate R2 bucket for production
- [ ] Enable `INTERNAL_AUTH_TOKEN` for metrics endpoint
- [ ] Set appropriate `TRACING_SAMPLE_RATE` (0.1 or lower)
- [ ] Set `TRACING_INSECURE=false`
- [ ] Configure HTTPS URLs for Stripe onboarding
- [ ] Review and adjust `STRIPE_APPLICATION_FEE_PERCENT`
- [ ] Enable Redis for distributed rate limiting (if multi-instance)
- [ ] Configure secret rotation schedule
- [ ] Set up monitoring alerts (metrics, logs, traces)

## Configuration Examples

### Minimal Development Setup

For local development with minimal external dependencies:

```bash
# configs/dev.env
SUBCULT_ENV=development
DATABASE_URL=postgres://subcults:password@localhost:5432/subcults?sslmode=disable
JWT_SECRET=dev-secret-change-me-before-production

# LiveKit (required)
LIVEKIT_URL=wss://subcults-dev.livekit.cloud
LIVEKIT_API_KEY=APIdevkey123
LIVEKIT_API_SECRET=devsecret123

# Stripe (required)
STRIPE_API_KEY=sk_test_123456789
STRIPE_WEBHOOK_SECRET=whsec_test123
STRIPE_ONBOARDING_RETURN_URL=http://localhost:3000/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=http://localhost:3000/stripe/refresh

# MapTiler (required)
MAPTILER_API_KEY=devkey123

# Jetstream (required for indexer)
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# Optional: Enable tracing for debugging
TRACING_ENABLED=true
TRACING_OTLP_ENDPOINT=http://localhost:4318
TRACING_SAMPLE_RATE=1.0
TRACING_INSECURE=true
```

### Full Development Setup

With all optional features enabled:

```bash
# configs/dev.env
SUBCULT_ENV=development
SUBCULT_PORT=8080
DATABASE_URL=postgres://subcults:password@localhost:5432/subcults?sslmode=disable
JWT_SECRET=dev-secret-change-me-before-production

# LiveKit
LIVEKIT_URL=wss://subcults-dev.livekit.cloud
LIVEKIT_API_KEY=APIdevkey123
LIVEKIT_API_SECRET=devsecret123

# Stripe
STRIPE_API_KEY=sk_test_123456789
STRIPE_WEBHOOK_SECRET=whsec_test123
STRIPE_ONBOARDING_RETURN_URL=http://localhost:3000/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=http://localhost:3000/stripe/refresh
STRIPE_APPLICATION_FEE_PERCENT=5.0

# MapTiler
MAPTILER_API_KEY=devkey123

# Jetstream
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# R2 (media uploads)
R2_BUCKET_NAME=subcults-media-dev
R2_ACCESS_KEY_ID=devr2key
R2_SECRET_ACCESS_KEY=devr2secret
R2_ENDPOINT=https://devaccount.r2.cloudflarestorage.com
R2_MAX_UPLOAD_SIZE_MB=15

# Redis (distributed rate limiting)
REDIS_URL=redis://localhost:6379

# Metrics
METRICS_PORT=9090
INTERNAL_AUTH_TOKEN=dev-metrics-token

# Feature Flags
RANK_TRUST_ENABLED=true

# Tracing (full sampling)
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=http://localhost:4318
TRACING_SAMPLE_RATE=1.0
TRACING_INSECURE=true
```

### Production Setup

With production-grade security and monitoring:

```bash
# configs/prod.env (loaded from secret manager in production)
SUBCULT_ENV=production
SUBCULT_PORT=8080
DATABASE_URL=postgres://user:password@ep-prod-123.us-east-2.aws.neon.tech/neondb?sslmode=require
JWT_SECRET=prod-secret-rotate-every-90-days-ABC123xyz==

# LiveKit (production project)
LIVEKIT_URL=wss://subcults-prod.livekit.cloud
LIVEKIT_API_KEY=APIprodkey456
LIVEKIT_API_SECRET=prodsecret987654321

# Stripe (live mode)
STRIPE_API_KEY=sk_live_987654321
STRIPE_WEBHOOK_SECRET=whsec_live456
STRIPE_ONBOARDING_RETURN_URL=https://subcults.com/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=https://subcults.com/stripe/refresh
STRIPE_APPLICATION_FEE_PERCENT=5.0

# MapTiler
MAPTILER_API_KEY=prodkey456

# Jetstream
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# R2 (production bucket)
R2_BUCKET_NAME=subcults-media-prod
R2_ACCESS_KEY_ID=prodaccesskey
R2_SECRET_ACCESS_KEY=prodsecretkey
R2_ENDPOINT=https://prodaccount.r2.cloudflarestorage.com
R2_MAX_UPLOAD_SIZE_MB=15

# Redis (multi-instance rate limiting)
REDIS_URL=redis://:password@redis-cluster.example.com:6379/0

# Metrics (authenticated)
METRICS_PORT=9090
INTERNAL_AUTH_TOKEN=prod-metrics-token-rotate-every-90-days-XYZ789==

# Feature Flags
RANK_TRUST_ENABLED=true

# Tracing (10% sampling)
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=https://otlp-collector.example.com:4318
TRACING_SAMPLE_RATE=0.1
TRACING_INSECURE=false
```

### Staging Setup

Staging mirrors production but with test credentials:

```bash
# configs/staging.env
SUBCULT_ENV=staging
SUBCULT_PORT=8080
DATABASE_URL=postgres://user:password@ep-staging-123.us-east-2.aws.neon.tech/neondb?sslmode=require
JWT_SECRET=staging-secret-different-from-prod-ABC123==

# LiveKit (staging project)
LIVEKIT_URL=wss://subcults-staging.livekit.cloud
LIVEKIT_API_KEY=APIstaging456
LIVEKIT_API_SECRET=stagingsecret789

# Stripe (test mode)
STRIPE_API_KEY=sk_test_staging123
STRIPE_WEBHOOK_SECRET=whsec_test_staging456
STRIPE_ONBOARDING_RETURN_URL=https://staging.subcults.com/stripe/return
STRIPE_ONBOARDING_REFRESH_URL=https://staging.subcults.com/stripe/refresh
STRIPE_APPLICATION_FEE_PERCENT=5.0

# MapTiler
MAPTILER_API_KEY=stagingkey789

# Jetstream (default or test instance)
JETSTREAM_URL=wss://jetstream1.us-east.bsky.network/subscribe

# R2 (staging bucket)
R2_BUCKET_NAME=subcults-media-staging
R2_ACCESS_KEY_ID=stagingaccesskey
R2_SECRET_ACCESS_KEY=stagingsecretkey
R2_ENDPOINT=https://stagingaccount.r2.cloudflarestorage.com
R2_MAX_UPLOAD_SIZE_MB=15

# Redis
REDIS_URL=redis://:password@redis-staging.example.com:6379/0

# Metrics
METRICS_PORT=9090
INTERNAL_AUTH_TOKEN=staging-metrics-token-ABC==

# Feature Flags (test new features here first)
RANK_TRUST_ENABLED=true

# Tracing (higher sampling for testing)
TRACING_ENABLED=true
TRACING_EXPORTER_TYPE=otlp-http
TRACING_OTLP_ENDPOINT=https://otlp-staging.example.com:4318
TRACING_SAMPLE_RATE=0.5
TRACING_INSECURE=false
```

## Troubleshooting

### Configuration Loading Errors

**Error**: `failed to load config file`

**Solution**: Check that the config file path is correct and the file is valid YAML. Environment variables take precedence, so the file is optional.

### Validation Errors

**Error**: `DATABASE_URL is required`

**Solution**: Ensure `DATABASE_URL` is set in your environment or config file.

**Error**: `JWT_SECRET is required`

**Solution**: Set `JWT_SECRET` to a strong random value (min 32 chars).

**Error**: `PORT must be a valid integer`

**Solution**: Ensure `SUBCULT_PORT` or `PORT` is a numeric value (e.g., `8080`, not `abc`).

### Service Connection Errors

**Error**: `failed to connect to database`

**Solution**:
- Verify `DATABASE_URL` is correct
- Check network connectivity to database host
- Ensure database is running and accepting connections
- Verify SSL mode matches database configuration (`sslmode=require` for Neon)

**Error**: `failed to connect to LiveKit`

**Solution**:
- Verify `LIVEKIT_URL` is correct WebSocket URL (starts with `wss://`)
- Check `LIVEKIT_API_KEY` and `LIVEKIT_API_SECRET` are valid
- Ensure LiveKit project is active in dashboard

**Error**: `Stripe webhook validation failed`

**Solution**:
- Verify `STRIPE_WEBHOOK_SECRET` matches the webhook endpoint in Stripe dashboard
- Ensure webhook endpoint URL is correct
- Check that webhook events are being sent to the correct environment (test vs live)

### Secret Masking in Logs

All secrets are automatically masked in log output via `config.LogSummary()`:
- Passwords in database URLs are replaced with `****`
- API keys show only first 4 characters (e.g., `sk_l****` for Stripe)
- Generic secrets show first 4 characters or `****` if shorter than 8 characters

If you see secrets in logs, this is a bug—please report it immediately.

## Additional Resources

- **Codebase Guide**: See `.github/copilot-instructions.md` for architecture overview
- **API Reference**: See `docs/API_REFERENCE.md` for endpoint documentation
- **Database Schema**: See `migrations/000000_initial_schema.up.sql` for schema definition
- **Migration Guide**: See `migrations/README.md` for migration procedures
- **Deployment Guide**: See `docs/OPERATIONS.md` for deployment procedures
- **Privacy Guide**: See `docs/PRIVACY.md` for location consent and privacy practices

## Questions?

If you have questions about configuration or encounter issues not covered here:
1. Check existing [GitHub Issues](https://github.com/subculture-collective/subcults/issues)
2. Review the [README](../README.md) for setup instructions
3. Open a new issue with the `documentation` label
