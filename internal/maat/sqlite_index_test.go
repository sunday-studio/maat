package maat

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestRebuildSQLiteIndexSearchesMarkdown(t *testing.T) {
	root := writeSQLiteIndexFixture(t)

	info, err := RebuildSQLiteIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if info.Path != SQLiteIndexPath(root) {
		t.Fatalf("unexpected index path: %s", info.Path)
	}
	if info.Documents != 2 {
		t.Fatalf("expected two indexed documents, got %d", info.Documents)
	}

	results, err := SearchSQLiteIndex(info.Path, "agent health")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d: %#v", len(results), results)
	}
	if results[0].Path != "docs/health.md" {
		t.Fatalf("unexpected result path: %#v", results[0])
	}
	if results[0].Line != 3 {
		t.Fatalf("expected matching line 3, got %d", results[0].Line)
	}
	if results[0].Type != "doc" || results[0].Title != "Health" {
		t.Fatalf("unexpected result metadata: %#v", results[0])
	}
}

func TestSQLiteIndexFallbackSearch(t *testing.T) {
	root := writeSQLiteIndexFixture(t)
	path := filepath.Join(root, ".maat", "fallback.sqlite")

	info, err := RebuildSQLiteIndexWithOptions(SQLiteIndexOptions{
		Store:      root,
		Path:       path,
		DisableFTS: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.FTS {
		t.Fatalf("expected FTS to be disabled")
	}

	results, err := SearchSQLiteIndex(path, "timeline")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d: %#v", len(results), results)
	}
	if results[0].Path != "projects/sample.md" {
		t.Fatalf("unexpected fallback result: %#v", results[0])
	}
}

func TestSQLiteIndexKeepsBootstrapJSONIndexSeparate(t *testing.T) {
	root := writeSQLiteIndexFixture(t)

	idx, err := BuildIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	jsonPath, err := WriteIndex(root, idx)
	if err != nil {
		t.Fatal(err)
	}

	sqliteInfo, err := RebuildSQLiteIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	if jsonPath == sqliteInfo.Path {
		t.Fatalf("expected separate JSON and SQLite index paths")
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("bootstrap JSON index was not preserved: %v", err)
	}
	if _, err := os.Stat(sqliteInfo.Path); err != nil {
		t.Fatalf("SQLite index was not written: %v", err)
	}
}

func TestSQLiteIndexTypesTargetLayoutObjects(t *testing.T) {
	root := writeSQLiteTargetLayoutFixture(t)

	info, err := RebuildSQLiteIndexWithOptions(SQLiteIndexOptions{
		Store:      root,
		DisableFTS: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Documents != 4 {
		t.Fatalf("expected four indexed documents, got %d", info.Documents)
	}

	assertSearchResultType(t, info.Path, "monitor rollup", "ticket", "projects/sample/tickets/T-20260525-190700-b91c.md")
	assertSearchResultType(t, info.Path, "health clarity", "goal", "projects/sample/goals/G-20260525-190533-a7f3.md")
	assertSearchResultType(t, info.Path, "claim expiration", "event", "projects/sample/events/2026/05/E-20260525-191100-codex-4c9a.md")
	assertSearchResultType(t, info.Path, "github.com/sunday-studio/sample", "project", "projects/sample/project.md")
}

func TestOpenSQLiteIndexDetectsFallbackMetadata(t *testing.T) {
	root := writeSQLiteIndexFixture(t)
	path := filepath.Join(root, ".maat", "fallback.sqlite")
	if _, err := RebuildSQLiteIndexWithOptions(SQLiteIndexOptions{Store: root, Path: path, DisableFTS: true}); err != nil {
		t.Fatal(err)
	}

	idx, err := OpenSQLiteIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()

	if idx.fts {
		t.Fatalf("expected opened index to detect disabled FTS")
	}
}

func writeSQLiteIndexFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "docs", "health.md"), "# Health\n\nAgent health needs clearer status search.\n")
	mustWrite(t, filepath.Join(root, "projects", "sample.md"), `# Project: Sample

| Field | Value |
|---|---|
| ID | sample |
| Status | active |
| Updated | 2026-05-25 |

## Current

Timeline search should find project history.

## Goals

### G-001: Ship

| Field | Value |
|---|---|
| Status | active |

#### Tasks

- [ ] T-001: Build index
`)
	mustWrite(t, filepath.Join(root, ".maat", "ignored.md"), "# Ignored\n\nAgent health should not be indexed from cache.\n")

	return root
}

func writeSQLiteTargetLayoutFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "projects", "sample", "project.md"), `# Project: Sample

| Field | Value |
|---|---|
| Project Key | sample |
| Display Name | Sample |
| Status | active |
| Created | 2026-05-25 |
| Updated | 2026-05-25 |
| Remote | git@github.com:sunday-studio/sample.git |

## Summary

Tracks github.com/sunday-studio/sample agent operations.
`)
	mustWrite(t, filepath.Join(root, "projects", "sample", "goals", "G-20260525-190533-a7f3.md"), `# Goal: Improve Health Clarity

| Field | Value |
|---|---|
| Goal ID | G-20260525-190533-a7f3 |
| Project | sample |
| Status | active |
| Created | 2026-05-25T19:05:33Z |

## Outcome

Health clarity should explain agent state.
`)
	mustWrite(t, filepath.Join(root, "projects", "sample", "tickets", "T-20260525-190700-b91c.md"), `# Ticket: Fix Monitor Rollup

| Field | Value |
|---|---|
| Ticket ID | T-20260525-190700-b91c |
| Project | sample |
| Goal | G-20260525-190533-a7f3 |
| Status | active |
| Created | 2026-05-25T19:07:00Z |

## Description

Monitor rollup should separate agent availability from check health.
`)
	mustWrite(t, filepath.Join(root, "projects", "sample", "events", "2026", "05", "E-20260525-191100-codex-4c9a.md"), `# Event: ticket.claimed

| Field | Value |
|---|---|
| Event ID | E-20260525-191100-codex-4c9a |
| Time | 2026-05-25T19:11:00Z |
| Actor | codex |
| Project | sample |
| Type | ticket.claimed |
| Object | T-20260525-190700-b91c |

## Summary

Recorded claim expiration for the active ticket.
`)
	return root
}

func assertSearchResultType(t *testing.T, indexPath, query, resultType, resultPath string) {
	t.Helper()
	results, err := SearchSQLiteIndex(indexPath, query)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range results {
		if result.Path == resultPath {
			if result.Type != resultType {
				t.Fatalf("expected %s to be typed %q, got %#v", resultPath, resultType, result)
			}
			return
		}
	}
	t.Fatalf("missing result %s for query %q in %#v", resultPath, query, results)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteFTSMetadataReadable(t *testing.T) {
	root := writeSQLiteIndexFixture(t)
	info, err := RebuildSQLiteIndex(root)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", info.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRow(`SELECT value FROM index_metadata WHERE key = 'version'`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != "1" {
		t.Fatalf("unexpected index version: %s", version)
	}
}
