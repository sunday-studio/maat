# Terminal Apps IA

This document defines the information architecture for a Maat terminal-app catalog and discovery surface inspired by [TerminalApps](https://terminal-apps.dev/).

TerminalApps is a simple public directory: a short title and value statement, followed by scannable app entries with name, stars, links, one-line description, language, license, and a lightweight updates signup. Maat should borrow the clarity of that structure while making the experience useful for project-memory and TUI work.

## Product Intent

Create a compact catalog that helps humans and agents understand what makes a strong terminal app, track candidate examples, and turn observations into Maat goals and tickets.

The catalog should answer:

- What terminal apps are worth studying?
- What patterns do they use?
- Which patterns should Maat adopt?
- What implementation tickets are needed?
- What evidence shows the pattern works?

## Audience

- Product builders evaluating terminal UX patterns.
- Agents turning product observations into Maat project work.
- Maintainers checking whether Maat TUI decisions are grounded in examples.

## Core Objects

### Catalog App

Represents one terminal app worth studying.

Fields:

- name
- slug
- summary
- source URL
- website URL
- stars
- language
- license
- category
- tags
- last reviewed date
- screenshots or media references when available
- notes

### Pattern

Represents a reusable UI or product pattern observed across apps.

Fields:

- title
- category
- problem solved
- observed in
- why it matters for Maat
- implementation notes
- related tickets

### Decision

Represents whether Maat should adopt, defer, or reject a pattern.

Fields:

- decision
- rationale
- linked pattern
- linked Maat goal or ticket
- evidence
- date

## Top-Level IA

```text
Terminal Apps
|-- Catalog
|-- Patterns
|-- Decisions
|-- Maat Opportunities
`-- Updates
```

## Catalog View

Purpose: fast scanning of terminal apps.

Primary content:

- app name
- stars
- one-line summary
- language
- license
- category
- links

Controls:

- search
- category filter
- language filter
- tag filter
- sort by stars, recently reviewed, name

Default sort:

1. manually pinned examples
2. stars descending
3. name ascending

Entry shape:

```text
lazygit                         57.5K stars
simple terminal UI for git commands
Go | MIT | git | dashboard | keyboard-first
[GitHub] [Website] [Notes]
```

## App Detail View

Purpose: turn a listed app into useful product learning.

Sections:

- Summary
- Links
- Metadata
- UX Patterns
- Screens and Interaction Notes
- What Maat Should Learn
- Related Goals and Tickets
- Review History

The detail page should not become a generic article. It should stay structured enough that an agent can extract work from it.

## Patterns View

Purpose: group observations by reusable design patterns.

Pattern categories:

- navigation
- layout
- keyboard model
- filtering
- inspection/detail panes
- command execution
- background refresh
- error and empty states
- onboarding/setup
- visual hierarchy
- accessibility and no-color readability

Example pattern:

```text
Focused detail pane
Problem: list views hide important object context.
Observed in: lazygit, posting, harlequin
Maat use: selected ticket detail pane with metadata, recent events, and safe actions.
Related tickets: TUI focused ticket pane, ownership display.
```

## Decisions View

Purpose: preserve why a pattern was adopted or skipped.

Decision states:

- adopt
- adopt later
- reject
- needs research

Decision entries should link to:

- source app
- pattern
- Maat goal
- ticket
- verification evidence

## Maat Opportunities View

Purpose: bridge inspiration into Maat work.

This view should list candidate improvements derived from the catalog:

- opportunity title
- source pattern
- affected Maat area
- rough effort
- risk
- suggested goal or ticket
- status

Opportunity statuses:

- proposed
- ticketed
- in progress
- verified
- declined

## Updates View

Purpose: keep the catalog current without turning it into a noisy feed.

Updates should show:

- newly reviewed apps
- changed app metadata
- new patterns
- adopted decisions
- stale entries that need review

The reference site uses an email signup for updates. Maat should start with local update history and only add external notifications if there is a real maintainer workflow.

## TUI IA

The catalog should work as a terminal-first experience before any web UI.

```text
+------------------------------------------------------------+
| Terminal Apps                                              |
| 42 apps | 18 patterns | 9 Maat opportunities               |
+------------------+----------------------+------------------+
| Apps             | Patterns             | Detail           |
| > lazygit        | > focused pane        | lazygit          |
|   btop           |   keyboard model      | 57.5K stars      |
|   gh-dash        |   background refresh  | Go | MIT         |
|   superfile      |   empty states        |                  |
|                  |                      | Lessons          |
| / search         | tab switch panes      | - fast nav       |
+------------------+----------------------+------------------+
```

Primary panes:

- apps list
- pattern list
- detail pane

Modes:

- apps
- patterns
- decisions
- opportunities

Core keys:

- `up/down` or `k/j`: move selection
- `tab`: switch pane or mode
- `/`: search
- `f`: filter
- `enter`: inspect selected item
- `r`: refresh
- `q`: quit

## Data Storage

Catalog data should be stored as Markdown object files, consistent with Maat:

```text
projects/maat/catalog/apps/<app-slug>.md
projects/maat/catalog/patterns/<pattern-slug>.md
projects/maat/catalog/decisions/<decision-id>.md
projects/maat/catalog/events/YYYY/MM/<event-id>.md
```

Do not make SQLite authoritative. SQLite may index catalog data for search, but Markdown remains the durable source.

## Minimum Viable Slice

1. Add Markdown schema for catalog apps and patterns.
2. Add loader and validation for catalog files.
3. Add `maat catalog list` and `maat catalog show`.
4. Add TUI catalog mode with app list, pattern list, and detail pane.
5. Add docs explaining how to add a new app observation.

## Acceptance Criteria

- Catalog entries are readable as plain Markdown.
- Humans can scan apps by name, summary, stars, language, and license.
- Patterns connect examples to Maat implementation opportunities.
- The TUI works with keyboard-only navigation.
- No-color output remains readable.
- Search works through the existing Markdown and SQLite search paths.
- Agent-facing JSON exposes stable object IDs and links to related Maat tickets.

## Open Questions

- Should catalog objects live under the Maat project or a separate `terminal-apps` project?
- Should stars be manually recorded, periodically refreshed, or omitted to avoid stale data?
- Should screenshots be stored as local assets, remote links, or not at all?
- Should the first interface be CLI-only or TUI-first?
- Should suggestions create Maat tickets automatically, or only draft them for review?
