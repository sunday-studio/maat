# Desktop Write Actions

This document defines the first desktop write-action design for the macOS app
described in `docs/macos-app-architecture.md`.

The desktop app remains a thin interface over the `maat` CLI. It must not write
Markdown directly, mutate the SQLite cache, or implement separate sync rules.
Every write action should spawn the existing CLI command with `--json`, parse the
result, then refresh visible state from read commands.

## Scope

First release write actions:

- Create a goal.
- Create a ticket.
- Claim a ticket.
- Comment on a ticket.
- Complete a ticket.
- Run explicit sync actions.

The app should also expose validation and refresh behavior around these actions,
but the CLI remains responsible for storage writes, project validation, index
refresh, auto-commit, and auto-push.

## Command Runner Contract

All write commands run through one command runner API that accepts a command
descriptor rather than a shell string:

```ts
type MaatCommand = {
  argv: string[];
  cwd?: string;
  timeoutMs: number;
};
```

The runner must call the configured `maat` binary directly with an argument
array. It must not concatenate user input into a shell command. This avoids
quoting bugs in titles, acceptance criteria, comments, and evidence.

Every write command includes `--json`. When the user configured a storage path in
the desktop app, every command also includes `--storage <path>` even if the CLI
could infer it. The desktop app should prefer explicit state over working
directory inference because users can launch the app outside a project repo.

Expected write JSON:

```json
{
  "action": "ticket.created",
  "project_key": "maat",
  "goal_id": "G-20260527-120000-abcd",
  "ticket_id": "T-20260527-120100-ef01",
  "event_id": "E-20260527-120101-alice-2345",
  "index_refreshed": true,
  "index_warning": "",
  "auto_sync": {
    "committed": true,
    "pushed": false,
    "commit_message": "status(maat): create ticket",
    "commit_pathspecs": ["projects/maat"]
  },
  "auto_sync_warning": ""
}
```

The parser should require `action`, `project_key`, `event_id`, and
`index_refreshed`. It should require `goal_id` for `goal.created`, `ticket_id`
for every ticket action, `agent` and `expires_at` for `ticket.claimed`, and
`auto_sync` only when the field is present. Unknown fields should be ignored so
the app remains compatible with future CLI additions.

Malformed JSON after a zero exit code is a command-runner error. The app should
show stderr/stdout diagnostics and keep the pre-write UI state.

## Form Validation

The UI should validate required fields before enabling submit. Validation should
match CLI rules and remain conservative; the CLI is still the final authority.

| Action | Required fields | Desktop validation |
| --- | --- | --- |
| Create goal | project key, title, outcome | Trim whitespace; title and outcome must be non-empty. |
| Create ticket | project key, title, description, acceptance | Trim whitespace; title and description must be non-empty; at least one non-empty acceptance criterion is required. |
| Claim ticket | project key, ticket ID, agent, TTL | Ticket ID must be present; default agent comes from desktop actor settings; TTL defaults to `2h` and must use a CLI duration such as `30m`, `2h`, or `24h`. |
| Comment | project key, ticket ID, comment | Comment must be non-empty after trimming. |
| Complete ticket | project key, ticket ID, evidence | Evidence must be non-empty after trimming. |
| Sync | storage path | Storage path must be configured. |

Ticket creation should present acceptance criteria as repeatable rows. Empty rows
are ignored, but submission is disabled until at least one row remains after
trimming. Each non-empty row becomes one `--acceptance <text>` argument.

Completion evidence should be a focused text area that encourages exact
verification, for example checks run, commit hash, branch, PR, or manual QA. The
desktop app should not allow completion with placeholder evidence such as empty
text or whitespace. The CLI will also reject missing evidence.

## Command Construction

Create goal:

```sh
maat goal create <project-key> <title> --outcome <text> --storage <path> --json
```

On success, the UI must show the created goal ID from `goal_id`.

Create ticket:

```sh
maat ticket create <project-key> <title> \
  --description <text> \
  --acceptance <criterion> \
  --storage <path> \
  --json
```

If the ticket belongs to a goal, add `--goal <goal-id>`. For multiple acceptance
criteria, repeat `--acceptance <criterion>` in the argument array.

Claim ticket:

```sh
maat ticket claim <ticket-id> \
  --project <project-key> \
  --agent <actor> \
  --ttl <duration> \
  --storage <path> \
  --json
```

Comment:

```sh
maat ticket comment <ticket-id> <comment> \
  --project <project-key> \
  --storage <path> \
  --json
```

Complete:

```sh
maat ticket complete <ticket-id> \
  --project <project-key> \
  --evidence <text> \
  --storage <path> \
  --json
```

Explicit sync:

```sh
maat sync --storage <path> --json
maat sync --storage <path> --push --json
maat sync --storage <path> --status --json
```

`--push` should require an explicit user action unless auto-push is already
enabled in setup. The app should label sync actions by result, not by assumption:
show whether a commit happened, whether push happened, and any dirty paths the
CLI reports.

## Post-Write Refresh

Writes should use pessimistic completion:

1. User submits a valid form.
2. Disable the submitting control and show the action as pending.
3. Spawn the CLI write command.
4. Parse the JSON result.
5. Refresh visible state with read commands.
6. Clear or close the form only after refresh succeeds.

The refresh set depends on the action:

| Action | Required refresh |
| --- | --- |
| `goal.created` | `maat project show <project-key> --storage <path> --json`; refresh dashboard counts with `maat status --storage <path> --json`. |
| `ticket.created` | `maat ticket show <ticket-id> --project <project-key> --storage <path> --json`; refresh ticket list and containing project. |
| `ticket.claimed` | `maat ticket show <ticket-id> --project <project-key> --storage <path> --json`; refresh ticket list so owner/claim state updates. |
| `ticket.commented` | `maat ticket show <ticket-id> --project <project-key> --storage <path> --json`; refresh activity/timeline if visible. |
| `ticket.completed` | `maat ticket show <ticket-id> --project <project-key> --storage <path> --json`; refresh ticket list, board counts, and active goal state. |
| Sync | `maat status --storage <path> --json`; refresh current project or ticket view if open. |

If the write succeeds but a post-write read fails, the app should keep the write
success visible using the parsed write result and show a refresh error with a
retry action. It must not retry the write automatically because create and event
commands are not idempotent.

## Sync Behavior

The CLI write path validates the written project, refreshes indexes best effort,
and runs auto-sync when configured. The desktop app should surface those fields
from the write result:

- `index_refreshed: true` means searches and derived lists can be treated as
  current.
- `index_warning` means the Markdown write persisted but the rebuildable cache is
  stale; offer an index rebuild or full refresh, not another write.
- `auto_sync.committed` means the storage repo commit was created.
- `auto_sync.pushed` means the storage repo push completed.
- `auto_sync_warning` means the Markdown write persisted but commit or push did
  not finish; show the warning and offer explicit sync retry.

Read commands may auto-pull before loading state, depending on user setup. The
desktop app should treat read warnings as non-fatal and show that it is using
local Markdown state. Write commands should not run while the storage repo is in
a known merge-conflict state; in that case the app should open the storage path
and ask the user to resolve Git before retrying.

## UI State Model

Use pessimistic writes for canonical state. The app should not insert fake goals,
tickets, comments, claims, or completion rows before the CLI returns success.
Temporary pending affordances are allowed, such as a spinner on the submit button
or a pending item in an activity panel, but they must be visually distinct from
confirmed project state.

After a successful write and refresh, show a compact confirmation with the IDs
returned by the CLI:

- Goal creation: goal ID and event ID.
- Ticket creation: ticket ID, optional goal ID, and event ID.
- Claim: ticket ID, agent, expiry, and event ID.
- Comment: ticket ID and event ID.
- Completion: ticket ID and event ID.

If the CLI exits non-zero, leave form values intact, re-enable submit, and show
the CLI error. When stderr contains a validation message such as `--acceptance is
required`, map it to the matching field. Otherwise show the command-level error.

## Evidence Handling

Ticket completion requires evidence because completion writes a
`ticket.completed` event and marks the ticket done. The desktop completion dialog
should make evidence the primary input, not an optional note.

The app should preserve the exact submitted evidence text and pass it as one
`--evidence <text>` argument. The event viewer should display completion evidence
from refreshed project or ticket activity after the write. If future CLI support
allows multiple evidence values, the UI can expand to repeatable evidence rows,
but the first version should match the current single `--evidence` command.

Recommended evidence examples in placeholder or helper text:

- `go test ./... passed`.
- `Committed <hash> on <branch>`.
- `Manual QA: created ticket, claimed it, completed it, and refreshed dashboard`.

The app must not mark a ticket complete locally if the command fails. A failed
completion leaves the ticket state unchanged until a refreshed read proves
otherwise.

## Acceptance Checklist

- Goal creation calls `maat goal create` with `--outcome`, parses `goal_id`, and
  shows the created ID after refresh.
- Ticket creation keeps submit disabled until description and at least one
  acceptance criterion are present, then calls `maat ticket create` with repeated
  `--acceptance` arguments.
- Claim, comment, and complete actions call the existing ticket CLI commands and
  refresh the visible ticket state after success.
- Completion requires evidence before invoking `maat ticket complete`.
- Write result warnings for index refresh and auto-sync are visible without
  retrying the write.
- Explicit sync actions use `maat sync --json` and refresh dashboard state after
  success.
