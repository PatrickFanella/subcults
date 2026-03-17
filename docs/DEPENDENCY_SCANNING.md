# Dependency Vulnerability Scanning

This document provides technical details about the vulnerability scanning implementation in Subcults.

## Overview

The project uses a multi-layered approach to dependency vulnerability scanning:
- **Go**: govulncheck for Go module vulnerabilities
- **JavaScript**: npm audit for NPM package vulnerabilities  
- **Docker**: Trivy for container image vulnerabilities
- **Automation**: Dependabot for automated dependency updates

## Architecture

### Workflow Structure

The main scanning workflow is defined in `.github/workflows/dependency-scan.yml` and consists of three parallel jobs:

```yaml
jobs:
  govulncheck:     # Scans Go dependencies
  npm-audit:       # Scans NPM dependencies (matrix: web, e2e)
  docker-scan:     # Scans Docker images (matrix: api, frontend, indexer)
```

### Trigger Conditions

The workflow runs on:
- **Pull Requests**: When dependency files are modified
- **Push**: To `main` or `develop` branches
- **Schedule**: Weekly on Mondays at 9:00 AM UTC (cron: `0 9 * * 1`)
- **Manual**: Via workflow_dispatch

## Job Details

### 1. Go Vulnerability Scanning (govulncheck)

**Purpose**: Detect known vulnerabilities in Go dependencies using the official Go vulnerability database.

**Implementation**:
```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan with JSON output
govulncheck -json ./... > govulncheck-results.json

# Generate human-readable summary
govulncheck ./... > scan-summary.md
```

**Output**:
- JSON results with OSV (Open Source Vulnerability) entries
- Human-readable markdown summary
- Vulnerability count for severity assessment

**Failure Threshold**:
- Any detected vulnerability fails the CI build
- Zero tolerance policy for Go vulnerabilities

**Artifacts**:
- `govulncheck-results.json` - Machine-readable scan results
- `scan-summary.md` - Human-readable summary

**PR Comments**:
- Posted automatically when vulnerabilities are found
- Includes full scan output for immediate visibility

### 2. NPM Vulnerability Scanning (npm audit)

**Purpose**: Detect known vulnerabilities in NPM dependencies for frontend and E2E tests.

**Matrix Strategy**:
```yaml
strategy:
  matrix:
    directory: ['web', 'e2e']
```

**Implementation**:
```bash
# JSON output for parsing
npm audit --json > npm-audit-results.json

# Human-readable output
npm audit > npm-audit-human.txt
```

**Output Parsing**:
```bash
# Extract vulnerability counts by severity
CRITICAL=$(jq '.metadata.vulnerabilities.critical // 0' npm-audit-results.json)
HIGH=$(jq '.metadata.vulnerabilities.high // 0' npm-audit-results.json)
MODERATE=$(jq '.metadata.vulnerabilities.moderate // 0' npm-audit-results.json)
LOW=$(jq '.metadata.vulnerabilities.low // 0' npm-audit-results.json)
```

**Failure Thresholds**:
- **CRITICAL**: Exit code 1 (fails CI)
- **HIGH**: Exit code 0 but warning logged
- **MODERATE/LOW**: Exit code 0, reported in comments

**Artifacts** (per directory):
- `npm-audit-results-{directory}.json` - Machine-readable results
- `npm-audit-human.txt` - Full text report
- `scan-summary.md` - Formatted summary with severity table

**PR Comments**:
- Severity breakdown table
- Expandable details section with full audit output
- Posted only when vulnerabilities are detected

### 3. Docker Image Scanning (Trivy)

**Purpose**: Scan Docker base images and OS packages for vulnerabilities.

**Matrix Strategy**:
```yaml
strategy:
  matrix:
    dockerfile: ['Dockerfile.api', 'Dockerfile.frontend', 'Dockerfile.indexer']
```

**Implementation**:
```bash
# Build the image
docker build -f ${DOCKERFILE} -t subcults-${SERVICE}:scan .

# Scan with Trivy (JSON for parsing)
trivy image --format json --output trivy-results.json subcults-${SERVICE}:scan

# Scan with Trivy (table for humans)
trivy image --format table --output trivy-results.txt subcults-${SERVICE}:scan
```

**Trivy Configuration**:
- **Severities**: CRITICAL, HIGH, MEDIUM, LOW
- **Exit Code**: 0 (non-blocking for parsing)
- **Formats**: JSON (machine-readable) + Table (human-readable)

**Output Parsing**:
```bash
# Count by severity using jq
CRITICAL=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity=="CRITICAL")] | length' trivy-results.json)
HIGH=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity=="HIGH")] | length' trivy-results.json)
```

**Failure Thresholds**:
- **CRITICAL**: Exit code 1 (fails CI)
- **HIGH**: Exit code 0 but warning logged
- **MEDIUM/LOW**: Exit code 0, reported in artifacts

**GitHub Security Integration**:
- SARIF format results uploaded to GitHub Security tab
- Integrated with GitHub Code Scanning
- Results visible in repository Security > Code scanning

**Artifacts** (per Dockerfile):
- `trivy-results-Dockerfile.{service}.json` - JSON results
- `trivy-results.txt` - Table format report
- `scan-summary.md` - Severity breakdown + details

**PR Comments**:
- Image name and severity table
- Expandable details with full Trivy output
- Posted for any detected vulnerabilities

## Dependabot Configuration

**File**: `.github/dependabot.yml`

**Ecosystems Monitored**:
1. **gomod**: Go dependencies in `go.mod`
2. **npm**: NPM packages in `web/` and `e2e/`
3. **github-actions**: GitHub Actions versions
4. **docker**: Base images in Dockerfiles

**Schedule**: Weekly on Mondays at 9:00 AM UTC

**Grouping Strategy**:
- Minor and patch updates grouped together
- Major updates created as individual PRs
- Excludes critical packages (React, Vite, TypeScript) from auto-grouping

**PR Configuration**:
- Limit: 10 PRs for Go/NPM, 5 for Actions/Docker
- Auto-assign: `@subculture-collective/maintainers`
- Labels: `dependencies`, ecosystem-specific tags, `security`
- Commit prefix: `chore(deps)` for dependencies, `chore(deps-dev)` for dev deps

## Maintenance

### Weekly Tasks
- Review Dependabot PRs for security updates
- Check scheduled scan results for new vulnerabilities
- Update base images if security patches available

### Monthly Tasks
- Audit Dependabot configuration for effectiveness
- Review grouped dependency updates
- Assess vulnerability scanning coverage

### Quarterly Tasks
- Review security scanning tooling versions
- Evaluate new security scanning tools
- Update security policy documentation

## Troubleshooting

### govulncheck Fails to Build
**Issue**: Code doesn't compile due to missing dependencies or build errors

**Solution**: Use package-specific scanning:
```bash
# Scan specific packages that compile
govulncheck -scan package ./cmd/api
govulncheck -scan package ./cmd/indexer
```

### npm audit Shows Dev Dependencies
**Issue**: Dev dependencies flagged but not used in production

**Solution**: Review if vulnerability affects build/test process:
```bash
# Audit production only
npm audit --production

# Suppress dev dependency warnings if safe
npm audit --audit-level=moderate
```

### Trivy Scans Time Out
**Issue**: Large images cause timeout in CI

**Solution**: Reduce image size:
- Use multi-stage builds
- Switch to distroless base images
- Remove unnecessary packages

### False Positives
**Issue**: Vulnerability reported but not applicable

**Solution**:
1. Verify the vulnerability affects your code path
2. Check if fixed version is available
3. Document suppression in PR if false positive
4. Consider using ignore files (trivy.yaml, .npmauditignore)

## Performance Considerations

### Scan Duration
- **govulncheck**: ~30-60 seconds for module scan
- **npm audit**: ~10-20 seconds per directory
- **Trivy**: ~2-5 minutes per image (including build)

**Total workflow time**: ~10-15 minutes for all jobs (parallel execution)

### Caching Strategy
- Go modules: Cached by `actions/setup-go`
- NPM packages: Cached by `actions/setup-node`
- Docker layers: Build-time cache only (not cached between runs)

### Optimization Opportunities
1. Cache Trivy database for faster scans
2. Run only changed ecosystem scans on PRs
3. Use sparse checkouts for large repos
4. Implement incremental scanning for unchanged deps

## Integration with Security Tools

### GitHub Security Tab
- **Dependabot Alerts**: Automated dependency vulnerability alerts
- **Code Scanning**: Trivy SARIF uploads for image vulnerabilities
- **Secret Scanning**: Separate GitHub feature (not covered by this workflow)

### Artifact Retention
- **Duration**: 30 days for all scan results
- **Storage**: Compressed JSON + text reports
- **Access**: Available via GitHub Actions artifacts API

## References

- [govulncheck Documentation](https://go.dev/security/vuln/)
- [npm audit Documentation](https://docs.npmjs.com/cli/v10/commands/npm-audit)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [GitHub Security Best Practices](https://docs.github.com/en/code-security)

---

**Maintained by**: Security Team  
**Last Updated**: 2026-02-03  
**Review Schedule**: Quarterly
