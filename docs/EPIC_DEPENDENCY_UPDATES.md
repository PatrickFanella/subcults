# Epic Dependency Normalization - Issue Updates Required

Based on the consolidation map in Issue #416, the following issue descriptions need to be updated:

## Issue #305: Jetstream Indexer

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Core entities: Scenes, Events, Posts (#10, #15, #17)
- ✅ Database migrations (#4)
- ✅ AT Protocol record schema (docs/AT_PROTOCOL_SCHEMA.md)
- ⏳ Observability infrastructure (#307 - metrics collection)
- ⏳ Error logging (#298-302)
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #305 (Jetstream Indexer - this epic)
- #4 (Database setup)
- #298-302 (Error logging integration)
- #307 (Observability, Monitoring & Operations)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #8 (AT Protocol records) - deprecated; replaced with explicit schema documentation
- Added AT Protocol schema documentation (docs/AT_PROTOCOL_SCHEMA.md)
- Changed #19 → #307 (Observability, Monitoring & Operations)
- Removed #5 self-reference (parent issue)
- Added #416 (Canonical Roadmap) reference

---

## Issue #304: Search & Discovery

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Core entities: Scenes, Events, Posts (#10, #15, #17)
- ✅ Trust scoring infrastructure (#24)
- ⏳ Frontend app shell (#303)
- ⏳ Observability for performance monitoring (#307)
- ⏳ Security hardening for rate limiting (#308)
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #304 (Search & Discovery - this epic)
- #98-102 (Search sub-tasks - if they exist)
- #24 (Trust Graph for ranking)
- #303 (Frontend UX - search bar integration)
- #298-302 (Error logging - search error tracking)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #14 (Search & Discovery - deprecated parent)
- Changed #21 → #303 (Frontend UX Shell)
- Changed #19 → #307 (Observability)
- Changed #20 → #308 (Security)
- Removed #8 reference
- Added #416 (Canonical Roadmap) reference

---

## Issue #307: Observability, Monitoring & Operations

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Prometheus client library (go-client)
- ✅ Structured logging (slog)
- ✅ Request logging middleware
- ⏳ Error logging (#298-302)
- 🔗 Grafana (needs deployment)
- 🔗 Prometheus (needs deployment)
- 🔗 Log aggregation tool (ELK/Loki/etc.)
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #307 (Observability, Monitoring & Operations - this epic)
- #298-302 (Error logging & telemetry improvements)
- #308 (Security hardening - includes audit logging)
- #385 (Testing - includes monitoring tests)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #19 (deprecated parent epic)
- Changed #20 → #308 (Security Hardening)
- Changed #23 → #385 (Testing)
- Added #416 (Canonical Roadmap) reference

---

## Issue #306: LiveKit Streaming

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ LiveKit client library (livekit-client v2.16.0)
- ✅ Token service implementation
- ✅ Stream metrics instrumentation
- ✅ Participant state management (Zustand)
- ⏳ Frontend app shell (#303 - routing, layout)
- ⏳ Observability (#307 - Prometheus metrics)
- ⏳ Error logging (#298-302)
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #306 (LiveKit Streaming - this epic)
- #62-67 (Streaming sub-tasks - may have overlaps)
- #303 (Frontend UX - stream page integration)
- #298-302 (Error logging)
- #307 (Observability, Monitoring & Operations)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #23 (deprecated parent epic)
- Changed #19 → #307 (Observability)
- Removed #299 reference (task-level, not epic)
- Added #416 (Canonical Roadmap) reference

---

## Issue #303: Frontend UX Shell

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Backend API endpoints (#10, #15, #17, #22, #306)
- ⏳ Error logging improvements (#298-302)
- ⏳ Performance metrics collection (#298-302)
- 🔗 TypeScript strict mode enabled
- 🔗 Tailwind dark mode configured
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #303 (Frontend App Shell - this epic)
- #103-117 (Design system & components - if they exist)
- #298-302 (Error logging & telemetry)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #21 (deprecated parent epic)
- Changed #23 → #306 in dependencies (LiveKit Streaming)
- Removed #299 reference (use #298-302 range)
- Added #416 (Canonical Roadmap) reference

---

## Issue #308: Security Hardening & Compliance

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Authentication infrastructure (JWT)
- ✅ Payment integration (Stripe)
- ✅ Database (Postgres)
- ⏳ Secret vault (AWS Secrets Manager or equivalent)
- ⏳ Monitoring for audit logs (#307)
- 🔗 Container registry with vulnerability scanning
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #308 (Security & Hardening - this epic)
- #298-302 (Error logging - don't leak sensitive info)
- #307 (Observability - audit logging, monitoring)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #20 (deprecated parent epic)
- Changed #19 → #307 (Observability)
- Added #416 (Canonical Roadmap) reference

---

## Issue #385: Comprehensive Testing & QA

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Test frameworks (Go testing, Jest/Vitest)
- ⏳ Mock services for LiveKit, Stripe
- ⏳ Test database with testcontainers
- ⏳ CI/CD pipeline
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #385 (Testing & QA - this epic)
- #307 (Observability - metrics for test performance)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #18 (deprecated parent epic)
- Removed #23 reference (duplicate)
- Removed #309 self-reference
- Added #416 (Canonical Roadmap) reference

---

## Issue #386: Deployment & Infrastructure

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ Docker Compose setup
- ✅ CI/CD pipeline
- ⏳ Kubernetes cluster
- ⏳ Container registry
- ⏳ Monitoring infrastructure (#307, #385)
- ⏳ Secrets management solution
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #386 (Deployment & Infrastructure - this epic)
- #307 (Observability - dashboards/metrics)
- #308 (Security - credential management)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #16 (deprecated parent epic)
- Removed #310 reference (consolidated into #386)
- Added #416 (Canonical Roadmap) reference

---

## Issue #387: Documentation & Developer Reference

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- ✅ All features implemented (#303-#308, #305, #306, #385, #386)
- ✅ API stable and ready to document
- ✅ Operational procedures established
```

**Related Issues section - UPDATE:**
```markdown
### Related Issues
- #387 (Documentation & Developer Reference - this epic)
- #303-#308, #305, #306, #385, #386 (Features to document)
- #416 (Canonical Roadmap)
```

**Changes:**
- Removed #24 (deprecated parent epic - Note: #24 Trust Graph is NOT deprecated, only the old doc epic #24 in roadmap #1)
- Removed #12 reference (consolidated into #387)
- Removed #311 reference (consolidated into #387)
- Updated feature references to use canonical epic numbers
- Added #416 (Canonical Roadmap) reference

---

## Issue #13: Backfill & Migration

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- Roadmap #416, Database & Migrations #4, Jetstream Indexer #305.
```

**Changes:**
- Changed Roadmap #1 → Roadmap #416
- Changed #5 → #305 (Jetstream Indexer)

---

## Issue #24: Trust Graph & Alliances

**Dependencies section - UPDATE:**
```markdown
### Dependencies
- Roadmap #416, Database #4, Jetstream Indexer #305.
```

**Changes:**
- Changed Roadmap #1 → Roadmap #416
- Changed #5 → #305 (Jetstream Indexer)

---

## Summary

All epic issues now reference:
- **Canonical epics** from the consolidation map in #416
- **Roadmap #416** instead of deprecated Roadmap #1
- Task-level issues (#298-302, etc.) remain unchanged as they are implementation details

## Implementation

Since the GitHub MCP integration doesn't have write permissions for issue updates, these changes need to be made manually through the GitHub web interface by a repository maintainer.

## Consolidation Map (from #416)

For reference, here is the epic consolidation map:
- #5 → #305 (Jetstream Indexer)
- #8 → #304 (Search & Discovery)
- #19 → #307 (Observability)
- #20 → #308 (Security)
- #23 → #306 (LiveKit Streaming)
- #18 → #385 (Testing)
- #21 → #303 (Frontend UX)
- #310 → #386 (Deployment)
- #311 → #387 (Documentation)
- #12 → #387 (Documentation)
