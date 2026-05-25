# Project: Aether

| Field | Value |
|---|---|
| ID | aether |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #product #frontend #backend #release |

## Current

Aether is a local-first desktop knowledge and productivity app built with Tauri, Rust, React, and TypeScript. The repo includes a desktop app and an optional end-to-end encrypted sync server.

The current direction is a smaller sealed v1 release surface: Journal, Tasks, Goals, Settings, self-hosted encrypted Sync, Updater, command palette search, local search model setup, and scoped resource links. Canvas, Bookmarks, Graph, journal audio/transcription, and unfinished diagnostics/model-management surfaces are deferred or hidden for v1.

## Goals

### G-001: Finish the v1 release surface

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #release #frontend #backend |

#### Tasks

- [ ] T-001: Add or verify first-launch onboarding.
- [ ] T-002: Verify AI provider and local search model status from the UI.
- [ ] T-003: Verify updater checks, update available state, skipped versions, download, install, and preferences.
- [ ] T-004: Verify self-hosted encrypted sync setup, reconnect, manual sync, media policy, and failure messages.
- [ ] T-005: Confirm navigation and command palette expose only v1-ready destinations.
- [ ] T-006: Run release verification checks for Rust, frontend, onboarding, journal, tasks, goals, sync, updater, search, and persistence.

## Blockers

- None recorded in Maat. Some v1 items remain unverified.

## Decisions

- Canvas is out for v1.
- Updater is in scope for v1.
- First-run onboarding is in scope for v1.
- AI key setup is visible for v1 while journal audio/transcription remains deferred.
- Bookmarks and Graph should be finished or hidden for v1.

## Links

- [Aether docs index](../../aether/docs/readme.md)
- [Project README](../../aether/docs/reference/project-readme.md)
- [V1 release checklist](../../aether/docs/milestones/v1-release-checklist.md)
- [Completed work](../../aether/docs/milestones/completed-work.md)
