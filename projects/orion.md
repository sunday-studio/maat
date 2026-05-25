# Project: Orion

| Field | Value |
|---|---|
| ID | orion |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #infra #backend #frontend #release |

## Current

Orion is a self-hosted monitoring app for small server setups. It has an Agent for Linux/macOS hosts, Core for ingestion, health computation, incidents, alerts, and persistence, and a Console UI for operational views.

The docs show milestone work through a first deploy candidate. The next release direction focuses on making agent and monitor health more trustworthy, explainable, and debuggable.

## Goals

### G-001: Complete first self-hosted deploy verification

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #release #infra |

#### Tasks

- [ ] T-001: Start Core from the Docker Compose sample with real admin and JWT settings.
- [ ] T-002: Confirm Core health and Console login.
- [ ] T-003: Install an Agent with a reachable Core URL.
- [ ] T-004: Confirm Agent service state, local state database, Console registration, monitor rows, and restart behavior.
- [ ] T-005: Run and store a SQLite backup after the first successful report.

### G-002: Improve agent health and debugging clarity

| Field | Value |
|---|---|
| Status | proposed |
| Updated | 2026-05-25 |
| Tags | #product #frontend #backend |

#### Tasks

- [ ] T-001: Separate agent availability from monitor health in status computation and UI.
- [ ] T-002: Add threshold-based monitor rollup so a few monitor failures do not imply the agent is down.
- [ ] T-003: Make missing agent reports visible and explain degraded/down causes.
- [ ] T-004: Preserve richer monitor report metadata for failed and successful checks.
- [ ] T-005: Add raw report inspection drawers for agent and monitor report rows.
- [ ] T-006: Add incident lifecycle controls for manual resolution and removed-monitor cleanup.

## Blockers

- None recorded in Maat. First-run deploy tasks remain unchecked here because Maat has not verified them directly.

## Decisions

- Keep Core and Console deployed together from the published Docker image.
- Keep the Agent installable on Linux and macOS hosts.
- Treat the next release goal as health model trust and explainability.

## Links

- [Orion README](../../orion/README.md)
- [Milestones](../../orion/docs/milestones/README.md)
- [First run checklist](../../orion/docs/deployment/first-run-checklist.md)
- [Agent health next release plan](../../orion/docs/plans/agent-health-debugging-next-release.md)
