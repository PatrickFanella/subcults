# Security Policy

## Vulnerability Scanning

Subcults implements automated dependency vulnerability scanning to ensure the security of our codebase and dependencies.

### Automated Scanning

Our security scanning runs in the following scenarios:

1. **Pull Requests**: On every PR that modifies dependencies or Docker files
2. **Push to main/develop**: On every push to protected branches
3. **Weekly Schedule**: Every Monday at 9:00 AM UTC
4. **Manual Trigger**: Can be manually triggered via GitHub Actions

### Scanning Tools

We use the following industry-standard tools for vulnerability detection:

#### Go Dependencies (govulncheck)
- **Tool**: [govulncheck](https://go.dev/security/vuln/)
- **Scans**: Go modules in `go.mod` for known vulnerabilities
- **Threshold**: Fails CI on ANY detected vulnerability
- **Coverage**: All Go dependencies and transitive dependencies

#### NPM Dependencies (npm audit)
- **Tool**: [npm audit](https://docs.npmjs.com/cli/v10/commands/npm-audit)
- **Scans**: 
  - Frontend dependencies (`web/package.json`)
  - E2E test dependencies (`e2e/package.json`)
- **Threshold**: 
  - CRITICAL vulnerabilities: **Fail CI** ❌
  - HIGH vulnerabilities: **Warning** ⚠️
  - MODERATE/LOW vulnerabilities: **Comment on PR** 💬
- **Coverage**: All NPM dependencies and dev dependencies

#### Docker Images (Trivy)
- **Tool**: [Trivy](https://github.com/aquasecurity/trivy)
- **Scans**:
  - `Dockerfile.api` - API server image
  - `Dockerfile.frontend` - Frontend static assets image
  - `Dockerfile.indexer` - Jetstream indexer image
- **Threshold**:
  - CRITICAL vulnerabilities: **Fail CI** ❌
  - HIGH vulnerabilities: **Warning** ⚠️
  - MEDIUM/LOW vulnerabilities: **Reported** 📊
- **Coverage**: Base images, OS packages, and application dependencies
- **Integration**: Results uploaded to GitHub Security tab (SARIF format)

### Automated Dependency Updates

We use [Dependabot](https://docs.github.com/en/code-security/dependabot) to automate dependency updates:

- **Schedule**: Weekly on Mondays at 9:00 AM UTC
- **Ecosystems**:
  - Go modules (`go.mod`)
  - NPM packages (`web/`, `e2e/`)
  - GitHub Actions workflows
  - Docker base images
- **Grouping**: Minor and patch updates are grouped to reduce PR noise
- **Auto-merge**: Not enabled - all updates require manual review

### CI Integration

The vulnerability scanning workflow (`.github/workflows/dependency-scan.yml`) runs three parallel jobs:

```
dependency-scan
├── govulncheck (Go)
├── npm-audit (JavaScript)
└── docker-scan (Docker images)
```

#### Workflow Behavior

**On Pull Requests**:
- Scans run automatically when dependencies change
- Results posted as PR comments
- CI fails on CRITICAL vulnerabilities
- Warnings posted for HIGH vulnerabilities

**On main/develop Push**:
- Full scan of all dependencies
- Results archived as workflow artifacts
- Alerts created for detected issues

**Weekly Scheduled Scan**:
- Comprehensive scan of all dependencies
- Results available in workflow artifacts
- Automated Dependabot PRs created for updates

### Viewing Scan Results

#### In GitHub Actions
1. Go to the **Actions** tab
2. Select the **Dependency Vulnerability Scanning** workflow
3. Click on a specific run to view results
4. Download artifacts for detailed reports:
   - `govulncheck-results` - Go vulnerability scan
   - `npm-audit-results-web` - Frontend NPM audit
   - `npm-audit-results-e2e` - E2E NPM audit
   - `trivy-results-Dockerfile.*` - Docker image scans

#### In GitHub Security Tab
1. Go to the **Security** tab
2. Click **Dependabot alerts** for dependency vulnerabilities
3. Click **Code scanning** for Trivy Docker image results

### Responding to Vulnerabilities

When a vulnerability is detected:

1. **Review the Alert**: Check the severity, affected package, and available fixes
2. **Assess Impact**: Determine if the vulnerable code path is used in production
3. **Update Dependencies**: 
   - For Go: Run `go get -u <package>` and `go mod tidy`
   - For NPM: Run `npm audit fix` or manually update `package.json`
   - For Docker: Update base image versions in Dockerfiles
4. **Test Changes**: Run tests to ensure updates don't break functionality
5. **Create PR**: Submit changes with vulnerability fix details in description
6. **Verify Fix**: Ensure CI scans pass and vulnerability is resolved

### Security Best Practices

In addition to automated scanning, we follow these security practices:

- **Minimal Docker Images**: Use minimal base images (Alpine, distroless) to reduce attack surface
- **Dependency Pinning**: Lock dependency versions in `go.sum` and `package-lock.json`
- **Regular Updates**: Review and update dependencies weekly via Dependabot
- **Security Reviews**: All dependency updates reviewed by maintainers
- **Principle of Least Privilege**: Docker containers run as non-root users

## Reporting Security Issues

If you discover a security vulnerability in Subcults, please report it responsibly:

1. **DO NOT** open a public GitHub issue
2. Email security concerns to: [security@subcults.dev](mailto:security@subcults.dev)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

We will acknowledge your report within 48 hours and provide updates on remediation progress.

## Security Disclosure Timeline

- **T+0**: Vulnerability reported to security@subcults.dev
- **T+48h**: Acknowledgment and initial assessment
- **T+7d**: Fix developed and tested
- **T+14d**: Security patch released
- **T+30d**: Public disclosure (if applicable)

## Supported Versions

We provide security updates for:

- **Current release** (main branch): Full support
- **Previous release**: Security fixes only
- **Older releases**: No support - please upgrade

## Contact

For security-related questions or concerns:
- Email: security@subcults.dev
- PGP Key: [Available on request]

---

Last updated: 2026-02-03
