# ADR-004: Standard Library Router Over Chi

**Status:** Accepted
**Date:** 2026-01-15

## Context

The original plan specified chi as the HTTP router. During implementation, Go 1.22 introduced enhanced pattern matching in `net/http` (`http.NewServeMux()`), including method-based routing and path parameters — features that previously required third-party routers.

## Decision

Use Go's standard library `http.NewServeMux()` instead of chi. The middleware stack is implemented as composable `func(http.Handler) http.Handler` wrappers:

1. Request ID (first — sets context for all downstream middleware)
2. Logging (depends on Request ID)
3. Security Headers (CSP, HSTS, X-Frame-Options)
4. CORS
5. Rate Limiting
6. Authentication (conditional per route)

## Consequences

### Positive

- Zero external dependency for routing — one less package to audit and maintain.
- Standard library compatibility — any Go developer can read the routing code without learning a framework.
- Go 1.22+ pattern matching covers our routing needs (method dispatch, path params).

### Negative

- No built-in middleware chaining syntax like `r.Use()` — we compose middleware manually.
- Some convenience features (route groups, inline middleware per-group) require more boilerplate.

### Neutral

- Performance is equivalent — both chi and stdlib use a radix tree internally.
- Migration to chi (or any router) remains straightforward since handlers use the standard `http.Handler` interface.

## Alternatives Considered

### Alternative 1: Chi Router

Chi provides elegant middleware chaining (`r.Use()`, `r.Group()`) and was the original plan. Rejected because Go 1.22+ `http.NewServeMux()` now handles the critical routing features (method+path matching) that previously required chi, and eliminating the dependency reduces supply chain risk.

### Alternative 2: Gorilla Mux

Rejected because Gorilla Mux was archived (though later revived). The stdlib approach avoids dependency on third-party project maintenance decisions.
