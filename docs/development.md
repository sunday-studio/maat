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

## Test

```sh
GOCACHE=/private/tmp/maat-go-cache go test ./...
```

## Current Implementation Scope

The first executable slice is intentionally small:

- parse legacy flat project files from `projects/*.md`
- validate known status values
- compute status totals
- search Markdown files directly
- write a rebuildable JSON index to `.maat/index.json`

The architecture still targets SQLite FTS and optional vector search. The JSON index is a bootstrap artifact so the CLI can work before the SQLite layer lands.
