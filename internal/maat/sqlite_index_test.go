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
	if results[0].Path != "projects/orion.md" {
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
	mustWrite(t, filepath.Join(root, "projects", "orion.md"), `# Project: Orion

| Field | Value |
|---|---|
| ID | orion |
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
