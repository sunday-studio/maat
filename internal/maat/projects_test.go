package maat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProjectFile(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, "sample.md")
	if err := os.WriteFile(path, []byte(`# Project: Sample

| Field | Value |
|---|---|
| ID | sample |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #infra #backend |

## Current

Current state.

## Goals

### G-001: Ship

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #release |

#### Tasks

- [ ] T-001: Open item
- [x] T-002: Done item

## Blockers

- None.

## Decisions

- Use Git.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := ParseProjectFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if project.ID != "sample" || project.Status != "active" || project.Title != "Sample" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if len(project.Goals) != 1 {
		t.Fatalf("expected one goal, got %d", len(project.Goals))
	}
	if project.Goals[0].ID != "G-001" || len(project.Goals[0].Tickets) != 2 {
		t.Fatalf("unexpected goal: %#v", project.Goals[0])
	}
	if !project.Goals[0].Tickets[1].Done {
		t.Fatalf("expected second ticket done")
	}
	if len(project.Decisions) != 1 {
		t.Fatalf("expected decision")
	}
}

func TestLoadProjectsUsesStateDirectory(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "state", "projects")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, "maat.md")
	if err := os.WriteFile(path, []byte(`# Project: Maat

| Field | Value |
|---|---|
| ID | maat |
| Status | active |
| Owner | agents |
| Updated | 2026-05-26 |
| Tags | #product |
`), 0o644); err != nil {
		t.Fatal(err)
	}

	projects, err := LoadProjects(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != "maat" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
	if projects[0].Path != "state/projects/maat.md" {
		t.Fatalf("unexpected path: %q", projects[0].Path)
	}
}

func TestSearchMarkdown(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "note.md"), []byte("# Note\n\nAgent health needs clarity.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err := Search(root, "health")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Path != "docs/note.md" || results[0].Line != 3 {
		t.Fatalf("unexpected result: %#v", results[0])
	}
}
