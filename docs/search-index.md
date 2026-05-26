# Search And Indexing

Maat uses SQLite as a local, rebuildable index over the Markdown storage repo.

The index exists for speed, search, ranking, UI queries, and agent-friendly retrieval. It is not the source of truth.

## Concurrency Model

Markdown in Git is the source of truth. SQLite is a local cache for the process or machine that built it.

The preferred many-agent setup is:

- agents write append-only Markdown object and event files
- agents coordinate through Git pull, commit, and push
- each agent, process, or machine keeps its own `.maat` SQLite cache
- stale caches are rebuilt from Markdown instead of merged or repaired by hand

This avoids turning one SQLite file into a shared write target for hundreds of agents. If a deployment later needs centralized query service behavior, that service should read from Git-backed Markdown and own its cache separately.

## Index Responsibilities

- Parse Markdown object files.
- Materialize current project, goal, and ticket state.
- Build a timeline from event files.
- Provide full-text search.
- Leave room for semantic search when embeddings are available.
- Track source file paths and hashes for incremental indexing.
- Serve fast queries to the CLI, TUI, web UI, and MCP adapter.

## Rebuild Rule

The index can always be rebuilt:

```sh
maat index rebuild
```

Deleting the SQLite file must not lose user data.

If a write succeeds but index rebuild fails, the command should treat the write as durable and report a warning. Human output should say that search or UI views may be stale until `maat index rebuild` runs. `--agent-use` should emit a structured warning update so agents can continue without repeating the state write.

## Suggested Tables

```text
files
projects
repositories
goals
tickets
events
decisions
reports
agents
claims
comments_view
object_links
search_documents
embeddings
```

`comments_view` can be materialized from `ticket.commented` events rather than stored as first-class Markdown files.

## Full-Text Search

Use SQLite FTS for keyword search.

Search documents should include:

- project summaries
- goal outcomes
- ticket title, description, acceptance criteria, and status
- event summaries and evidence
- decisions
- reports
- tags
- linked repo metadata

Current command:

```sh
maat search "agent health"
```

Future search filters may add project and document-type scoping.

## Semantic Search

Semantic search should be optional and local-first.

The index should support:

- embedding text chunks from Markdown objects
- storing embedding model metadata
- invalidating embeddings when source hashes change
- falling back to FTS when embeddings are unavailable

The first implementation can ship FTS only. Vector search can be added after the schema and query behavior are stable.

## Current State Computation

Current state is computed from:

1. object creation files
2. append-only events
3. correction events
4. active non-expired claims

For example, a ticket's current status is the latest valid status event for that ticket. If no status event exists, use the status in the ticket file.

## Staleness

The index should compute stale projects and tickets.

Suggested default:

- project stale after 14 days without events
- active ticket stale after 7 days without events
- claim stale after `expires_at`

These should be configurable later.

## Search Results

Search results should return enough context for both humans and agents:

- result type
- title
- project
- status
- path
- matching excerpt
- score
- last event time

For agent usage, JSON output should be available:

```sh
maat search "sync passphrase" --json
```
