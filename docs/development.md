# Development

Maat is currently a Go CLI named `maat`.

## Run Locally

Use a writable Go build cache when running inside restricted environments:

```sh
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat status --storage .
```

Useful commands against a local or external Maat storage repo:

```sh
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat setup
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat projects --storage /path/to/maat-state
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat project show maat --storage /path/to/maat-state
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat search "agent health" --storage /path/to/maat-state
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/maat index rebuild --storage /path/to/maat-state
```

Use `go run ./cmd/maat setup --storage /absolute/path/to/maat-state` in tests or agent scripts when setup must remain non-interactive.

## Install Locally

Use the local installer when you need a `maat` binary on `PATH`:

```sh
scripts/install.sh
```

The installer copies an existing checkout binary when available, or builds `./cmd/maat` with Go in offline mode. See [Install](./install.md) for install targets, storage paths, and index paths.

## Test

```sh
GOCACHE=/private/tmp/maat-go-cache go test ./...
```

## Build

Build the local binary:

```sh
make build
```

Use explicit metadata when testing release stamping:

```sh
VERSION=v0.0.0-dev COMMIT=local DATE=2026-05-25T00:00:00Z make build
dist/maat version
```

Build release archives locally:

```sh
make release
```

The release script builds macOS and Linux artifacts by default. Override targets with:

```sh
TARGETS="darwin/arm64 linux/amd64" scripts/build-release.sh
```

GitHub Actions runs the same test and release build path on `v*` tags and manual dispatch. Tag builds publish the generated tarballs and checksum file to the matching GitHub Release; manual dispatch uploads artifacts only.

## Current Implementation Scope

The current executable slice includes:

- setup config with storage path and auto-sync defaults
- project registration with `maat initialize` or `maat project link`
- object-layout project, goal, ticket, and event files under `projects/<project-key>/`
- legacy flat project reads for older storage repos
- validation for required fields, status values, timestamps, duplicate IDs, object links, malformed tables, and event paths
- status totals across legacy and object-layout projects
- search through the rebuildable SQLite index, with direct Markdown search as a fallback
- rebuildable JSON and SQLite indexes under `.maat/`
- write commands for goals, tickets, comments, claims, completion, and explicit sync

The architecture still targets optional vector search. The JSON and SQLite indexes are rebuildable artifacts; Markdown in Git remains canonical.
