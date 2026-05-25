# Development

Maat is currently a Go CLI named `matt`.

## Run Locally

Use a writable Go build cache when running inside restricted environments:

```sh
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/matt status --storage .
```

Useful commands:

```sh
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/matt projects --storage .
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/matt project show orion --storage .
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/matt search "agent health" --storage .
GOCACHE=/private/tmp/maat-go-cache go run ./cmd/matt index rebuild --storage .
```

## Install Locally

Use the local installer when you need a `matt` binary on `PATH`:

```sh
scripts/install.sh
```

The installer copies an existing checkout binary when available, or builds `./cmd/matt` with Go in offline mode. See [Install](./install.md) for install targets, storage paths, and index paths.

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
dist/matt version
```

Build release archives locally:

```sh
make release
```

The release script builds macOS and Linux artifacts by default. Override targets with:

```sh
TARGETS="darwin/arm64 linux/amd64" scripts/build-release.sh
```

GitHub Actions runs the same test and release build path on `v*` tags and manual dispatch. It uploads artifacts only; it does not publish a release.

## Current Implementation Scope

The first executable slice is intentionally small:

- parse legacy flat project files from `projects/*.md`
- validate known status values
- compute status totals
- search through the rebuildable SQLite index, with direct Markdown search as a fallback
- write a rebuildable JSON index to `.maat/index.json`
- write a rebuildable SQLite index to `.maat/index.sqlite`

The architecture still targets optional vector search. The JSON and SQLite indexes are rebuildable artifacts; Markdown in Git remains canonical.
