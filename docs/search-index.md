# Search And Indexing

Maat uses SQLite as a local, rebuildable index over the Markdown storage repo.

SQLite exists for speed and search. It is not the source of truth.

## Concurrency Model

Markdown in Git is authoritative. SQLite is a local cache for the process or machine that built it.

For many-agent use:

- agents write Markdown object and event files
- agents coordinate through Git pull, commit, and push
- each agent, process, or machine keeps its own `.maat` cache
- stale caches are rebuilt from Markdown

This avoids turning one SQLite file into a shared write target for hundreds of agents.

## Rebuild

The index can always be rebuilt:

```sh
maat index rebuild
```

Deleting the SQLite file does not delete project state.

If a write succeeds but index rebuild fails, the command treats the write as durable and reports a warning. Agents should run `maat index rebuild` later instead of repeating the state write.

## Search

Search reads the local SQLite index when it can and falls back to Markdown search when needed:

```sh
maat search "agent health"
maat search "sync passphrase" --json
```

Search documents include:

- project summaries
- goal outcomes
- ticket title, description, acceptance criteria, and status
- event summaries and evidence
- linked repo metadata

## Cache Files

Maat writes rebuildable cache files under:

```text
<storage>/.maat/index.json
<storage>/.maat/index.sqlite
```

These files should not be treated as primary state.
