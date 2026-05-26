package maat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanLegacyMigrationReturnsTargetPaths(t *testing.T) {
	source := t.TempDir()
	writeLegacyMigrationFixture(t, source)

	at := time.Date(2026, 5, 25, 20, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	plan, err := PlanLegacyMigration(source, MigrationOptions{At: at, Actor: "Codex Worker 4"})
	if err != nil {
		t.Fatal(err)
	}

	if plan.Source != source {
		t.Fatalf("unexpected source: %q", plan.Source)
	}
	if len(plan.Projects) != 1 {
		t.Fatalf("expected one project plan, got %d", len(plan.Projects))
	}

	project := plan.Projects[0]
	if project.LegacyPath != "projects/sample.md" {
		t.Fatalf("unexpected legacy path: %q", project.LegacyPath)
	}
	if project.ProjectKey != "sample" {
		t.Fatalf("unexpected project key: %q", project.ProjectKey)
	}
	if project.ProjectPath != "projects/sample/project.md" {
		t.Fatalf("unexpected project path: %q", project.ProjectPath)
	}

	wantGoals := []string{
		"projects/sample/goals/G-001.md",
		"projects/sample/goals/G-002.md",
	}
	if strings.Join(project.GoalPaths, "\n") != strings.Join(wantGoals, "\n") {
		t.Fatalf("unexpected goal paths: %#v", project.GoalPaths)
	}

	wantTickets := []string{
		"projects/sample/tickets/T-002.md",
		"projects/sample/tickets/T-g-001-t-001.md",
		"projects/sample/tickets/T-g-002-t-001.md",
	}
	if strings.Join(project.TicketPaths, "\n") != strings.Join(wantTickets, "\n") {
		t.Fatalf("unexpected ticket paths: %#v", project.TicketPaths)
	}

	wantEvent := "projects/sample/events/2026/05/E-20260525-203000-codex-worker-4-sample-migrated.md"
	if len(project.EventPaths) != 1 || project.EventPaths[0] != wantEvent {
		t.Fatalf("unexpected event paths: %#v", project.EventPaths)
	}
	if len(plan.Files) != 7 {
		t.Fatalf("expected seven planned files, got %d", len(plan.Files))
	}
}

func TestApplyLegacyMigrationWritesTargetLayoutWithoutTouchingSource(t *testing.T) {
	source := t.TempDir()
	writeLegacyMigrationFixture(t, source)
	legacyPath := filepath.Join(source, "projects", "sample.md")
	before, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	at := time.Date(2026, 5, 25, 20, 30, 0, 0, time.UTC)
	plan, err := ApplyLegacyMigration(source, destination, MigrationOptions{At: at, Actor: "maat"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) == 0 {
		t.Fatal("expected planned files")
	}

	after, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("legacy source file changed")
	}

	store, err := LoadObjectStore(destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Projects) != 1 {
		t.Fatalf("expected one migrated project, got %d", len(store.Projects))
	}
	project := store.Projects[0]
	if project.Key != "sample" || project.DisplayName != "Sample" || project.Status != "active" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if len(project.Goals) != 2 {
		t.Fatalf("expected two goals, got %d", len(project.Goals))
	}
	if len(project.Tickets) != 3 {
		t.Fatalf("expected three tickets, got %d", len(project.Tickets))
	}
	if project.Tickets[0].ID != "T-002" || project.Tickets[0].Status != "active" {
		t.Fatalf("unexpected preserved ticket: %#v", project.Tickets[0])
	}
	if project.Tickets[1].ID != "T-g-001-t-001" || project.Tickets[1].Status != "done" {
		t.Fatalf("unexpected first duplicate ticket: %#v", project.Tickets[1])
	}
	if project.Tickets[2].ID != "T-g-002-t-001" || project.Tickets[2].GoalID != "G-002" {
		t.Fatalf("unexpected duplicate-id ticket migration: %#v", project.Tickets[2])
	}
	if len(project.Events) != 1 || project.Events[0].Type != migrationEventType {
		t.Fatalf("unexpected migration events: %#v", project.Events)
	}
}

func TestApplyLegacyMigrationRefusesExistingTargetFile(t *testing.T) {
	source := t.TempDir()
	writeLegacyMigrationFixture(t, source)

	destination := t.TempDir()
	writeFile(t, filepath.Join(destination, "projects", "sample", "project.md"), "existing")

	_, err := ApplyLegacyMigration(source, destination, MigrationOptions{})
	if err == nil {
		t.Fatal("expected migration to fail on existing target file")
	}
	if !strings.Contains(err.Error(), "projects/sample/project.md") {
		t.Fatalf("expected target path in error, got %v", err)
	}
}

func writeLegacyMigrationFixture(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "projects", "sample.md"), `# Project: Sample

| Field | Value |
|---|---|
| ID | sample |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| Tags | #infra #agent-run |

## Current

Sample tracks distributed monitors and agent health.

## Goals

### G-001: Improve health clarity

| Field | Value |
|---|---|
| Status | active |
| Updated | 2026-05-25 |
| Tags | #backend |

#### Tasks

- [x] T-001: Separate agent availability from monitor health.
- [ ] T-002: Add raw report drawer.

### G-002: Improve deploy docs

| Field | Value |
|---|---|
| Status | proposed |
| Updated | 2026-05-25 |
| Tags | #docs |

#### Tasks

- [ ] T-001: Document first-run bootstrap.

## Blockers

- None.

## Decisions

- Keep deploy docs in the repo.
`)
}
