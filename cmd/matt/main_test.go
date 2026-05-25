package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunday-studio/maat/internal/maat"
)

func TestStatusJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var summary maat.StatusSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Projects != 1 || summary.Goals != 1 || summary.Tickets != 2 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestProjectsJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("projects", "--json", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}

	var projects []maat.Project
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != "orion" || projects[0].Title != "Orion" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}

func TestValidateCommand(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("validate", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "validated 1 files: ok") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestIndexRebuildAndSearchCommand(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("index", "rebuild", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "json:") || !strings.Contains(output, "sqlite:") {
		t.Fatalf("unexpected output: %q", output)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index: %v", err)
	}

	output, err = captureRun("search", "agent health", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "docs/note.md:3") {
		t.Fatalf("unexpected search output: %q", output)
	}
}

func TestMigratePlanCommandJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("migrate", "plan", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var plan maat.MigrationPlan
	if err := json.Unmarshal([]byte(output), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Source != store {
		t.Fatalf("unexpected source: %q", plan.Source)
	}
	if len(plan.Projects) != 1 {
		t.Fatalf("expected one project plan, got %d", len(plan.Projects))
	}
	project := plan.Projects[0]
	if project.LegacyPath != "projects/orion.md" || project.ProjectPath != "projects/orion/project.md" {
		t.Fatalf("unexpected project plan: %#v", project)
	}
	if len(project.GoalPaths) != 1 || len(project.TicketPaths) != 2 || len(project.EventPaths) != 1 {
		t.Fatalf("unexpected migrated object paths: %#v", project)
	}
	if strings.Contains(output, "Content") || strings.Contains(output, "Current state.") {
		t.Fatalf("plan json should not expose planned file contents: %q", output)
	}
}

func TestMigrateApplyCommandWritesDestinationOnly(t *testing.T) {
	store := writeCommandFixture(t)
	dest := t.TempDir()
	legacyPath := filepath.Join(store, "projects", "orion.md")
	before, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("migrate", "apply", "--storage", store, "--dest", dest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "migrated 1 projects into") || !strings.Contains(output, "wrote 5 files") {
		t.Fatalf("unexpected output: %q", output)
	}

	after, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("legacy source file changed")
	}
	if _, err := os.Stat(filepath.Join(dest, "projects", "orion", "project.md")); err != nil {
		t.Fatalf("expected migrated project file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, "projects", "orion", "project.md")); !os.IsNotExist(err) {
		t.Fatalf("source store should not receive target layout file, got err=%v", err)
	}

	objectStore, err := maat.LoadObjectStore(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(objectStore.Projects) != 1 || objectStore.Projects[0].Key != "orion" {
		t.Fatalf("unexpected migrated object store: %#v", objectStore.Projects)
	}
}

func captureRun(args ...string) (string, error) {
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()
	runErr := run(args)
	writer.Close()

	data, readErr := io.ReadAll(reader)
	reader.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(data), runErr
}

func writeCommandFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "note.md"), []byte("# Note\n\nAgent health needs clarity.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "orion.md"), []byte(`# Project: Orion

| Field | Value |
|---|---|
| ID | orion |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #infra |

## Current

Current state.

## Goals

### G-001: Ship

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |

#### Tasks

- [ ] T-001: Open item
- [x] T-002: Done item
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}
