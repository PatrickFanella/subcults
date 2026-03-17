# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Subcults project.

## What is an ADR?

An ADR captures the context, decision, and consequences for a significant architectural choice. ADRs are immutable once accepted — if a decision is reversed, a new ADR supersedes the original.

## Format

Each ADR follows the template in [adr-template.md](adr-template.md).

## Status Values

| Status         | Meaning                                       |
| -------------- | --------------------------------------------- |
| **Proposed**   | Under discussion, not yet accepted            |
| **Accepted**   | Decision is final and in effect               |
| **Superseded** | Replaced by a newer ADR (link to replacement) |
| **Deprecated** | No longer relevant (system changed)           |

## Naming Convention

Files are numbered sequentially: `NNNN-short-description.md`

## Index

| #                                           | Title                                                       | Status   | Date    |
| ------------------------------------------- | ----------------------------------------------------------- | -------- | ------- |
| [001](0001-stack-selection.md)              | Stack Selection (Go + React + Neon + LiveKit + Stripe + R2) | Accepted | 2025-12 |
| [002](0002-trust-ranking-feature-flag.md)   | Trust Ranking Behind Feature Flag                           | Accepted | 2025-12 |
| [003](0003-jetstream-realtime-ingestion.md) | Jetstream Real-Time Ingestion vs Periodic Polling           | Accepted | 2025-12 |
| [004](0004-stdlib-router-over-chi.md)       | Standard Library Router Over Chi                            | Accepted | 2026-01 |
| [005](0005-privacy-first-location.md)       | Privacy-First Location Consent Model                        | Accepted | 2025-12 |
| [006](0006-distroless-container-images.md)  | Distroless Container Images                                 | Accepted | 2026-01 |

## Creating a New ADR

```bash
./scripts/new-adr.sh "Short title of decision"
```

Or manually: copy `adr-template.md`, fill in fields, add to index above.

## References

- [ARCHITECTURE.md](../ARCHITECTURE.md)
- [GLOSSARY.md](../../GLOSSARY.md)
