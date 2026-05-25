# Project: Neptune

| Field | Value |
|---|---|
| ID | neptune |
| Status | proposed |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #product #backend #frontend #infra |

## Current

Neptune is a personal photo management and publishing system for indexing, organizing, and selectively publishing photos from a TrueNAS server to Cloudflare R2.

The documentation says the project is in design phase. The older README describes a three-component shape with Neptune Local, Neptune API, and Neptune Blog, while the docs describe a Go backend with PostgreSQL, imgproxy, TrueNAS storage, Cloudflare R2, Tailscale, and Docker deployment. Agents should reconcile that architecture before implementation work.

## Goals

### G-001: Reconcile architecture and implementation direction

| Field | Value |
|---|---|
| Status | waiting |
| Updated | 2026-05-25 |
| Tags | #product #infra #docs |

#### Tasks

- [ ] T-001: Decide whether the current architecture is three components or a consolidated Go backend plus web UI.
- [ ] T-002: Update the top-level README and docs so they describe the same system.
- [ ] T-003: Confirm the deployment target across TrueNAS, Tailscale, PostgreSQL, imgproxy, and Cloudflare R2.

### G-002: Build the v1 photo management MVP

| Field | Value |
|---|---|
| Status | proposed |
| Updated | 2026-05-25 |
| Tags | #backend #frontend #product |

#### Tasks

- [ ] T-001: Complete incremental indexing, background indexing progress, and manual re-index API.
- [ ] T-002: Add photo browsing with pagination, filters, sorting, search, metadata view, and image serving.
- [ ] T-003: Add file move, rename, delete, bulk move, and directory operations.
- [ ] T-004: Add Cloudflare R2 publishing and unpublishing.
- [ ] T-005: Add JWT authentication for a single-user protected API.
- [ ] T-006: Add album creation, listing, membership, and deletion.

## Blockers

- Architecture direction appears inconsistent between the top-level README and `docs/`.

## Decisions

- Use TrueNAS as the private source of photos.
- Use Cloudflare R2 for public photo publishing.
- Keep the design docs as living documentation until implementation direction is reconciled.

## Links

- [Neptune README](../../neptune/README.md)
- [Docs index](../../neptune/docs/README.md)
- [Features roadmap](../../neptune/docs/features.md)
- [Architecture](../../neptune/docs/architecture.md)
- [Technical decisions](../../neptune/docs/technical-decisions.md)
