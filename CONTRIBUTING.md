# Contributing to Subcults

Thank you for your interest in contributing to Subcults. This guide covers the development workflow, conventions, and expectations for all contributors.

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.24+ | `go version` to verify |
| Node.js | 20+ | LTS recommended |
| Docker | 24+ | Docker Compose v2 included |
| libvips | 8.15+ | Image processing (CGO dependency) |
| Make | Any | Build orchestration |

Optional:
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (falls back to Docker if absent)
- Redis (only for distributed rate limiting)
- k6 (load testing only)

## Getting Started

```bash
# Clone the repository
git clone https://github.com/subculture-collective/subcults.git
cd subcults

# Set up environment
cp configs/dev.env.example configs/dev.env
# Fill in required secrets (DATABASE_URL, JWT_SECRET, etc.)

# Start infrastructure
make compose-up

# Apply database migrations
make migrate-up

# Install frontend dependencies
cd web && npm ci && cd ..

# Run everything
make dev
```

The API runs at `http://localhost:8080` and the frontend at `http://localhost:5173`.

## Branching Model

All work happens on feature branches off `main`. The `main` branch is protected.

### Branch Naming

```
feature/issue-123-short-description
fix/issue-456-bug-description
docs/issue-789-update-readme
chore/issue-101-dependency-update
```

Always include the issue number. Keep descriptions short and hyphenated.

## Commit Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): brief description

Longer explanation if needed.

Fixes #123
```

### Types

| Type | Use For |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, no logic change |
| `refactor` | Code restructuring, no behavior change |
| `test` | Adding or updating tests |
| `chore` | Build, CI, tooling changes |
| `perf` | Performance improvement |

### Scope

Use the package or area name: `auth`, `geo`, `scene`, `api`, `frontend`, `middleware`, `config`, `docker`, `ci`.

### Examples

```
feat(scene): add proximity-based search endpoint
fix(auth): handle expired refresh tokens gracefully
test(geo): add table-driven tests for jitter calculation
docs(api): document payment webhook flow
chore(ci): add dependency scanning workflow
```

## Pull Request Process

### Before Opening a PR

1. Ensure your branch is up to date with `main`
2. Run the full check suite locally:
   ```bash
   make lint          # Go vet + frontend linters
   make test          # All unit tests with race detector
   ```
3. Verify your changes don't lower coverage

### PR Requirements

- **Link to issue**: Include `Closes #NNN` in the PR description
- **CI must pass**: All checks (lint, test, coverage gates) must be green
- **Coverage thresholds**:
  - Backend overall: >80%
  - Frontend overall: >70%
  - Critical packages (auth, geo, payment): 95%
- **No secrets committed**: Never commit `.env` files, API keys, or credentials

### PR Description Template

```markdown
## Summary

Brief description of what this PR does.

Closes #NNN

## Changes

- What was added/changed/removed

## Testing

- How was this tested?
- Which test files were added/updated?

## Privacy & Security

- Does this touch location data? If so, is consent enforced?
- Does this introduce new user input? If so, is it validated?

## Checklist

- [ ] Tests added/updated
- [ ] Documentation updated (if behavior changed)
- [ ] No secrets or PII in code/logs
- [ ] Coverage stable or improved
```

### Review Process

- Request at least one human review for production code
- Request Copilot review for automated feedback
- Address all review comments before merging
- Squash-merge to keep history clean

## Development Workflow

### Running Services

```bash
make compose-up       # Start Postgres (PostGIS)
make dev              # API + frontend dev servers
make dev-api          # API only (hot reload via go run)
make dev-frontend     # Frontend only (Vite HMR)
make dev-indexer      # Jetstream indexer
```

### Database Migrations

```bash
make migrate-up       # Apply all pending migrations
make migrate-down     # Rollback last migration
./scripts/migrate.sh version  # Check current version
```

Migrations require the `DATABASE_URL` environment variable. Source your dev env:
```bash
export $(grep -v '^#' configs/dev.env | xargs)
```

### Testing

```bash
make test             # Go + frontend tests
make test-coverage    # Generate coverage reports
make test-integration # Integration tests (requires Docker)
make test-e2e         # Playwright E2E tests
make test-load        # k6 load tests
```

### Linting

```bash
make lint             # Go vet + ESLint
make fmt              # Format Go code
cd web && npm run lint  # Frontend linting
```

### Docker

```bash
make docker-build     # Build all images
make docker-size      # Show image sizes
make compose-up       # Start local stack
make compose-down     # Stop local stack
```

## Code Quality Standards

### Go

- >80% test coverage for handlers and repositories
- 95% coverage for critical packages (`auth`, `geo`, `payment`)
- Table-driven tests with descriptive subtest names
- Race detector enabled (`-race` flag)
- All exported functions must handle errors explicitly
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Structured logging with `slog` (never `fmt.Println` or `log.Println`)

### Frontend

- >70% test coverage
- React Testing Library for component tests (behavior over snapshots)
- Accessibility: all interactive elements must have ARIA labels
- i18n: all user-facing text must use `t()` from i18next
- Tailwind CSS for styling (no inline styles, no CSS modules)

### Privacy

Every contribution touching location data or user information must:

1. Call `EnforceLocationConsent()` before persisting coordinates
2. Apply jitter for public display when `allow_precise=false`
3. Never log PII (DIDs, emails, precise coordinates)
4. Add privacy-focused tests verifying consent enforcement

### Security

- Parameterize all SQL queries (never string concatenation)
- Validate all user input via the `internal/validate` package
- No wildcard CORS origins
- No secrets in code, logs, or Docker images
- Rate limit all public endpoints

## Project Structure

```
cmd/           # Entry points (api, indexer, backfill)
internal/      # Private application code
  ├── api/     # HTTP handlers + error utilities
  ├── auth/    # JWT token management
  ├── config/  # koanf-based configuration
  ├── db/      # Database connection + utilities
  ├── geo/     # Geohash, jitter, proximity
  ├── middleware/ # HTTP middleware stack
  ├── scene/   # Domain models + repositories
  ├── validate/  # Input validation
  └── ...
web/           # Vite + React + TypeScript frontend
migrations/    # Database schema changes (golang-migrate)
scripts/       # Build and automation scripts
docs/          # Project documentation
configs/       # Environment templates
deploy/        # Production deployment config
perf/          # Performance test scenarios
```

## Reporting Issues

When opening an issue, include:

- Clear title describing the problem or feature
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Relevant logs or screenshots
- Labels: `bug`, `feature`, `docs`, `security`, `performance`

## Secret Handling

- Never commit actual secret values
- Use `configs/dev.env.example` as a template (placeholders only)
- Generate JWT secrets with: `openssl rand -base64 32`
- Rotate secrets using: `./scripts/rotate-jwt-secret.sh`
- See `docs/SECRETS_MANAGEMENT.md` for rotation procedures

## Getting Help

- Check `docs/` for existing documentation on your topic
- Search closed issues and PRs for prior solutions
- Open a discussion for architectural questions
- Tag `@onnwee` for urgent security or privacy concerns
