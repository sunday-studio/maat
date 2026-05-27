# Desktop Sync Controls

This document defines the macOS desktop design for sync status, validation, and
index rebuild controls.

The desktop app remains a thin interface over the `maat` CLI. It must not read
or write Markdown directly, inspect Git itself, or maintain a second source of
truth for validation or index health. The app should run parseable CLI commands,
map their JSON to typed view models, and show the safest next action.

## Command Surface

The first implementation should use these commands:

```sh
maat sync --status --json
maat sync --message "status(<project-key>): update maat" --json
maat sync --message "status(<project-key>): update maat" --push --json
maat validate --json
maat index rebuild
maat setup doctor --json
```

Use the app-configured storage path for every command when the user selected a
storage repo explicitly:

```sh
maat sync --status --storage <path> --json
```

The command runner should execute one sync or index mutation at a time per
storage repo. Status polling and validation may run concurrently only when there
is no active sync, push, or index rebuild. If a mutation starts while a poll is
queued, cancel or drop the queued poll and refresh after the mutation finishes.

## Sync Status

The app should poll `maat sync --status --json` after setup succeeds, when the
window becomes active, after every write command, after manual commit or push,
and after an external-change notification from the storage repo. Use a default
poll interval of 60 seconds while the app is visible and 5 minutes while it is in
the background. Do not poll while the machine is offline or while Git credentials
are known to be failing.

The status result is a `StoreSyncResult` with these UI-relevant fields:

| JSON field | UI use |
| --- | --- |
| `repository.is_repository` | If false, show setup repair and disable sync actions. |
| `repository.branch` | Show the active storage branch in the sync popover. |
| `repository.remote_url` | If empty, show "Local only" and disable push. |
| `repository.upstream` | If empty with a remote, require explicit upstream setup before push. |
| `repository.ahead` | Show local commits waiting to push. |
| `repository.behind` | Show remote commits waiting to pull or rebase outside the app. |
| `repository.pull_rebase` | Show Git policy detail in the sync popover. |
| `repository.pull_strategy_warning` | Promote to a warning row and disable one-click push. |
| `dirty_before_commit[]` | Show uncommitted storage changes. |
| `committed` | After manual sync, show whether a commit was created. |
| `pushed` | After manual push, show whether push completed. |
| `dirty_after_sync[]` | Show remaining uncommitted changes after sync. |

Primary status labels:

| Condition | Label | Affordance |
| --- | --- | --- |
| Not a Git repository | Storage is local only | Open setup repair; disable commit and push. |
| No remote URL | Local changes only | Allow commit; disable push; offer setup guidance. |
| Remote exists, no upstream | Remote not linked | Disable push; offer Git instructions from CLI warning. |
| `ahead == 0`, `behind == 0`, no dirty entries | Synced | No primary action. |
| Dirty entries present | Uncommitted changes | Enable "Commit changes". |
| `ahead > 0`, `behind == 0` | Ready to push | Enable "Push" when no pull strategy warning exists. |
| `behind > 0`, `ahead == 0` | Remote updates available | Disable writes that would auto-push; show pull/rebase guidance. |
| `ahead > 0`, `behind > 0` | Branch diverged | Disable push; show conflict-safe manual Git guidance. |
| `pull_strategy_warning != ""` | Sync needs review | Show warning text; disable one-click push. |

The compact toolbar indicator should be intentionally plain:

- Green dot and "Synced" when clean, connected, and not ahead or behind.
- Yellow dot and "Uncommitted" when dirty entries exist.
- Yellow dot and "Push available" when `ahead > 0` and `behind == 0`.
- Yellow dot and "Remote ahead" when `behind > 0` and `ahead == 0`.
- Red dot and "Sync blocked" when the repo is invalid, diverged, or a CLI error
  blocks status.

The sync popover should show branch, upstream, ahead, behind, dirty file count,
and the last status poll time. Dirty file rows should use the CLI status entry
codes as detail, for example `M README.md`, `A projects/maat/tickets/...`, or
`R old.md -> new.md` when `rename` is present.

## Manual Sync And Push

"Commit changes" runs:

```sh
maat sync --message "status(<project-key>): update maat" --json
```

"Push" runs:

```sh
maat sync --message "status(<project-key>): update maat" --push --json
```

The app should choose the project key from the active project when available. If
there is no active project context, use `maat` in the commit message scope.

During a manual sync or push:

- Disable create, claim, comment, complete, validation repair, sync, push, and
  index rebuild controls for the same storage repo.
- Keep read-only navigation enabled against the last loaded data.
- Show a single progress state in the sync popover.
- Preserve the previous status summary until the command returns, then replace
  it with the command result and immediately schedule a fresh status poll.

Manual push should be disabled when any of these are true:

- `repository.is_repository` is false.
- `repository.remote_url` is empty.
- `repository.upstream` is empty.
- `repository.behind > 0`.
- `repository.pull_strategy_warning` is non-empty.
- A validation run is active, a sync is active, or an index rebuild is active.

Manual commit should be disabled when any of these are true:

- `repository.is_repository` is false.
- No dirty entries exist.
- Validation is currently running.
- Sync or index rebuild is currently running.

If `maat sync --json` returns `committed: false` with dirty entries still present,
show the remaining dirty list from `dirty_after_sync[]`. This usually means the
CLI pathspec did not match the dirty files, so the user should see the exact
files that still need attention instead of a generic failure.

## Validation Panel

The validation panel should run `maat validate --json` on demand, after setup
doctor reports a validation error, and after a sync fails with a validation
error. It may also run automatically after a write command if no other mutation
is active.

The JSON result is a `ValidationReport`:

| JSON field | UI use |
| --- | --- |
| `files` | Show the number of Markdown files scanned. |
| `issues[]` | Empty means valid. Non-empty means the panel is in an error state. |
| `issues[].path` | File path to show and copy. |
| `issues[].line` | Optional line number to append to the path. |
| `issues[].code` | Stable issue identifier for grouping and filtering. |
| `issues[].message` | Human-readable fix guidance. |

Validation issues should be displayed as a table or list with:

- Path, including `:<line>` when `line > 0`.
- Issue code.
- Message.
- Project key inferred from `projects/<project-key>/...` when the path matches
  the object layout.

The panel should group issues by project, then by file path, while preserving the
CLI order within each file. Each row should offer "Copy path" and "Reveal in
Finder". Do not offer in-app Markdown editing in the first version.

Validation state should map as follows:

| Condition | Panel state | App behavior |
| --- | --- | --- |
| No validation has run | Not checked | Show "Run validation". |
| `issues[]` empty | Valid | Show files scanned and last run time. |
| `issues[]` non-empty | Issues found | Show issue count; block sync commit and push. |
| CLI exits non-zero with JSON report | Issues found | Parse and show issues, then show the non-zero exit as expected. |
| CLI exits non-zero without JSON | Validation unavailable | Show stderr and a retry action. |

When validation issues exist, write actions that would create more state may stay
enabled if the command itself can validate and reject unsafe writes. Commit and
push controls must be disabled until validation passes, because `maat sync`
validates before committing.

## Index Rebuild Prompt

SQLite and JSON indexes are rebuildable caches. The app should never present
index failure as lost project state.

Offer `maat index rebuild` when any of these warning signals appear:

| Source | Detection | UI affordance |
| --- | --- | --- |
| `maat setup doctor --json` | A check with `id == "indexes"` and `status == "warning"` or `status == "error"` | Show "Rebuild index" when `can_fix` is true or the message says indexes are missing/stale. |
| Write command JSON | `index_warning` is non-empty or `index_refreshed == false` | Keep the write result visible and show "Rebuild index". |
| Agent-use progress | Step `index.refresh` with `status == "warning"` | Attach a warning banner to the completed write. |
| Search failure | CLI error opening or rebuilding SQLite cache | Offer rebuild, then rerun the search after success. |

The prompt copy should make the cache boundary clear: "Project files were kept;
the local search index needs to be rebuilt." The primary action runs:

```sh
maat index rebuild --storage <path>
```

During rebuild:

- Disable search input, sync commit, push, validate, and other rebuild buttons
  for the same storage repo.
- Keep project and ticket views visible from the last loaded read model.
- Show progress from the command runner if available.
- On success, clear index warnings and rerun the last search or dashboard load.
- On failure, show stderr and keep the rebuild action available.

If `setup doctor --json --fix` is already running as part of setup repair, do not
also run `maat index rebuild`. Doctor repair can rebuild missing or stale indexes
itself.

## Warning Mapping

The command runner should normalize warnings into a common UI shape:

| Warning source | Normalized severity | User action |
| --- | --- | --- |
| `repository.pull_strategy_warning` | Warning | Read guidance; push disabled until Git policy is fixed. |
| `setup doctor` check with `status == "warning"` and `requires_approval == true` | Warning | Show suggestion; require user confirmation outside automatic repair. |
| `setup doctor` check with `status == "warning"` and `can_fix == true` | Repairable warning | Offer the specific repair action. |
| `setup doctor` check with `status == "error"` | Error | Show the check message and disable dependent actions. |
| `index_warning` | Repairable warning | Offer `maat index rebuild`. |
| Validation `issues[]` | Error | Show validation panel; block commit and push. |
| Git credential stderr | Error | Show failed Git action and retry after the user fixes credentials. |

Warnings should remain visible until the next successful command proves the
condition is gone. A user may dismiss a non-blocking banner for the current
session, but the sync popover and validation panel should still show the current
source state.

## Disabled States

Controls should be disabled by source condition rather than by generic app mode.

| Control | Disable when |
| --- | --- |
| Run validation | Setup is incomplete, validation is active, sync is active, or index rebuild is active. |
| Commit changes | Setup incomplete, non-Git storage, no dirty entries, validation issues exist, validation active, sync active, or index rebuild active. |
| Push | Commit disabled, no remote, no upstream, behind remote, diverged, pull strategy warning exists, sync active, or index rebuild active. |
| Rebuild index | Setup incomplete, index rebuild active, sync active, validation active, or storage path is not writable. |
| Search | Setup incomplete or index rebuild active. |
| Write actions | Setup incomplete, storage not writable, sync active, or index rebuild active. |

Disabled controls should expose the blocking reason in a tooltip or adjacent
status line. Avoid stacking multiple warnings on the button itself; keep detailed
diagnostics in the sync popover, validation panel, or setup repair view.

## Refresh Rules

After command completion:

| Command | Refresh |
| --- | --- |
| `maat sync --status --json` | Update sync indicator only. |
| `maat sync --json` | Update sync result, rerun dashboard read, then poll status. |
| `maat sync --push --json` | Update pushed state, rerun dashboard read, then poll status. |
| `maat validate --json` | Update validation panel and sync button disabled state. |
| `maat index rebuild` | Clear index warning, rerun last dashboard or search read, then poll status. |
| `maat setup doctor --json` | Update setup health, map index and validation checks into their panels. |

The app should store only ephemeral UI timestamps and dismissed-banner state. It
must not cache `ahead`, `behind`, validation issues, or index warnings as durable
application state; those values are snapshots from the CLI and should be
replaced by the next command result.
