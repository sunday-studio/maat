package maat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadObjectStoreParsesTargetLayout(t *testing.T) {
	root := t.TempDir()
	writeObjectFixture(t, root)

	store, err := LoadObjectStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Projects) != 1 {
		t.Fatalf("expected one project, got %d", len(store.Projects))
	}

	project := store.Projects[0]
	if project.Key != "orion-a31f" || project.DisplayName != "Orion" || project.Status != "active" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if project.Path != "projects/orion-a31f/project.md" {
		t.Fatalf("unexpected project path: %q", project.Path)
	}
	if project.Summary != "Self-hosted monitoring app with Agent, Core, and Console." {
		t.Fatalf("unexpected summary: %q", project.Summary)
	}
	if project.Identity["Remote"] != "git@github.com:sunday-studio/orion.git" {
		t.Fatalf("unexpected identity: %#v", project.Identity)
	}

	if len(project.Goals) != 1 {
		t.Fatalf("expected one goal, got %d", len(project.Goals))
	}
	goal := project.Goals[0]
	if goal.ID != "G-20260525-190533-a7f3" || goal.ProjectKey != project.Key || goal.Status != "active" {
		t.Fatalf("unexpected goal: %#v", goal)
	}
	if !strings.Contains(goal.Outcome, "Agent health should explain") {
		t.Fatalf("unexpected goal outcome: %q", goal.Outcome)
	}

	if len(project.Tickets) != 1 {
		t.Fatalf("expected one ticket, got %d", len(project.Tickets))
	}
	ticket := project.Tickets[0]
	if ticket.ID != "T-20260525-190700-b91c" || ticket.GoalID != goal.ID || ticket.Status != "waiting" {
		t.Fatalf("unexpected ticket: %#v", ticket)
	}
	if len(ticket.Acceptance) != 3 {
		t.Fatalf("expected three acceptance items, got %#v", ticket.Acceptance)
	}

	if len(project.Events) != 1 {
		t.Fatalf("expected one event, got %d", len(project.Events))
	}
	event := project.Events[0]
	if event.ID != "E-20260525-191100-codex-4c9a" || event.Type != "ticket.completed" || event.ObjectID != ticket.ID {
		t.Fatalf("unexpected event: %#v", event)
	}
	if len(event.Evidence) != 1 || !strings.Contains(event.Evidence[0], "go test") {
		t.Fatalf("unexpected evidence: %#v", event.Evidence)
	}
}

func TestParseObjectTicketFileRejectsInvalidStatus(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "projects", "orion-a31f", "tickets", "T-bad.md")
	writeFile(t, path, `# Ticket: Bad Status

| Field | Value |
|---|---|
| Ticket ID | T-bad |
| Project | orion-a31f |
| Status | almost |
| Created | 2026-05-25T19:07:00+02:00 |
`)

	_, err := ParseObjectTicketFile(root, path)
	if err == nil {
		t.Fatal("expected invalid status error")
	}
	if !strings.Contains(err.Error(), "invalid ticket status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseObjectEventFileRequiresObject(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "projects", "orion-a31f", "events", "2026", "05", "E-bad.md")
	writeFile(t, path, `# Event: ticket.completed

| Field | Value |
|---|---|
| Event ID | E-bad |
| Time | 2026-05-25T19:11:00+02:00 |
| Actor | codex |
| Project | orion-a31f |
| Type | ticket.completed |
`)

	_, err := ParseObjectEventFile(root, path)
	if err == nil {
		t.Fatal("expected missing object error")
	}
	if !strings.Contains(err.Error(), "missing Object") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeObjectFixture(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "projects", "orion-a31f", "project.md"), `# Project: Orion

| Field | Value |
|---|---|
| Project Key | orion-a31f |
| Display Name | Orion |
| Status | active |
| Created | 2026-05-25T19:05:00+02:00 |
| Updated | 2026-05-25T19:05:00+02:00 |
| Tags | #infra #agent-run |

## Summary

Self-hosted monitoring app with Agent, Core, and Console.

## Identity

| Field | Value |
|---|---|
| Primary Repo | R-20260525-190100-a31f |
| Remote | git@github.com:sunday-studio/orion.git |
`)
	writeFile(t, filepath.Join(root, "projects", "orion-a31f", "goals", "G-20260525-190533-a7f3.md"), `# Goal: Improve Agent Health Clarity

| Field | Value |
|---|---|
| Goal ID | G-20260525-190533-a7f3 |
| Project | orion-a31f |
| Status | active |
| Created | 2026-05-25T19:05:33+02:00 |
| Tags | #backend #frontend |

## Outcome

Agent health should explain whether the problem is the agent, monitor rollup, stale data, or check failure.
`)
	writeFile(t, filepath.Join(root, "projects", "orion-a31f", "tickets", "T-20260525-190700-b91c.md"), `# Ticket: Separate Agent Availability From Monitor Health

| Field | Value |
|---|---|
| Ticket ID | T-20260525-190700-b91c |
| Project | orion-a31f |
| Goal | G-20260525-190533-a7f3 |
| Status | waiting |
| Created | 2026-05-25T19:07:00+02:00 |
| Tags | #backend |

## Description

Make agent availability distinct from monitor health in computed status and UI presentation.

## Acceptance

- Agent heartbeat/report freshness is visible.
- Monitor failures do not make a reporting agent look down by themselves.
- The UI explains degraded and down causes.
`)
	writeFile(t, filepath.Join(root, "projects", "orion-a31f", "events", "2026", "05", "E-20260525-191100-codex-4c9a.md"), `# Event: ticket.completed

| Field | Value |
|---|---|
| Event ID | E-20260525-191100-codex-4c9a |
| Time | 2026-05-25T19:11:00+02:00 |
| Actor | codex |
| Project | orion-a31f |
| Type | ticket.completed |
| Object | T-20260525-190700-b91c |
| Commit | abc1234 |

## Summary

Completed the ticket after backend tests passed.

## Evidence

- go test ./... passed in apps/core.
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
