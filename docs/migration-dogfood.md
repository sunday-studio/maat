# Migration Dogfood

This note records the first dogfood run of the legacy flat-project migration against the current Maat repository. The real repository was not migrated in place.

## Command Used

Temp destination:

```sh
/private/tmp/maat-migration-dogfood.KV2Vgd
```

Plan:

```sh
GOCACHE=/private/tmp/maat-go-cache GOPATH=/private/tmp/maat-go-path go run ./cmd/matt migrate plan --storage . --json
```

Apply:

```sh
GOCACHE=/private/tmp/maat-go-cache GOPATH=/private/tmp/maat-go-path go run ./cmd/matt migrate apply --storage . --dest /private/tmp/maat-migration-dogfood.KV2Vgd
```

Result:

```text
migrated 4 projects into /private/tmp/maat-migration-dogfood.KV2Vgd
wrote 87 files
```

## Generated Structure Summary

The migration generated target-layout directories for the four legacy projects:

- `projects/aether/`
- `projects/maat/`
- `projects/neptune/`
- `projects/orion/`

Generated file counts:

| Kind | Count |
|---|---:|
| Project files | 4 |
| Goal files | 12 |
| Ticket files | 67 |
| Migration event files | 4 |
| Total Markdown files | 87 |

Each project received:

- `project.md`
- one file per parsed legacy goal under `goals/`
- one file per parsed legacy task under `tickets/`
- one `project.migrated` event under `events/2026/05/`

The generated target layout loaded successfully in the existing migration tests through `LoadObjectStore`.

## Issues Found

- In-place migration is not safe yet. The apply path previously overwrote existing destination files if a target file already existed.
- The migration now refuses to overwrite existing target files and returns the conflicting target path.
- Legacy repeated ticket IDs are resolved by prefixing them with the goal ID, for example `T-g-001-t-001`. This avoids collisions, but the generated IDs are less pleasant than future native Maat IDs.
- Project `Created` and `Updated` fields use the migration timestamp, not the legacy `Updated` field.
- Goal `Updated` metadata is not preserved.
- Legacy blockers and decisions are not migrated into first-class target objects yet.
- Migration history is one event per project. There are no per-goal or per-ticket migration events.

## In-Place Migration Safety

Do not run in-place migration yet.

The migration is now safer because it refuses existing target files, but the full in-place flow still needs a deliberate command that:

- validates the legacy source before planning
- checks for all target path collisions before writing any files
- supports dry-run output by default
- writes a backup or uses a Git branch
- handles legacy blockers and decisions
- updates or archives legacy flat project files intentionally

Until that exists, use `matt migrate apply --dest <temp-path>` only.

## Recommended Next Fixes

- Add a preflight collision scan to the migration plan so all conflicts are reported before any write begins.
- Preserve legacy `Updated` values on migrated project and goal objects.
- Migrate legacy blockers and decisions into target object files.
- Add optional `--actor` and `--at` flags so migration events are reproducible.
- Add a `matt migrate apply --in-place` command only after preflight, backup, and legacy archival behavior are designed.
- Add CLI output that summarizes counts by project without requiring `--json`.
