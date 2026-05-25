package maat

import (
	"strings"
	"testing"
	"time"
)

func TestNewIDWithReader(t *testing.T) {
	at := time.Date(2026, 5, 25, 19, 5, 33, 0, time.FixedZone("CEST", 2*60*60))
	id, err := NewIDWithReader(GoalIDPrefix, at, strings.NewReader("\xa7\xf3"))
	if err != nil {
		t.Fatal(err)
	}
	if id != "G-20260525-190533-a7f3" {
		t.Fatalf("unexpected id: %s", id)
	}
}

func TestNewActorEventIDWithReader(t *testing.T) {
	at := time.Date(2026, 5, 25, 19, 8, 12, 0, time.UTC)
	id, err := NewActorEventIDWithReader(at, "Codex Worker D", strings.NewReader("\x4c\x9a"))
	if err != nil {
		t.Fatal(err)
	}
	if id != "E-20260525-190812-codex-worker-d-4c9a" {
		t.Fatalf("unexpected id: %s", id)
	}
}

func TestEventRelativePath(t *testing.T) {
	at := time.Date(2026, 5, 25, 19, 8, 12, 0, time.UTC)
	path, err := EventRelativePath("Orion A31F", at, "E-20260525-190812-codex-4c9a")
	if err != nil {
		t.Fatal(err)
	}
	want := "projects/orion-a31f/events/2026/05/E-20260525-190812-codex-4c9a.md"
	if path != want {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestRenderEventMarkdown(t *testing.T) {
	at := time.Date(2026, 5, 25, 19, 11, 0, 0, time.FixedZone("CEST", 2*60*60))
	markdown, err := RenderEventMarkdown(Event{
		ID:      "E-20260525-191100-codex-4c9a",
		Time:    at,
		Actor:   "codex",
		Project: "orion-a31f",
		Type:    "ticket.completed",
		Object:  "T-20260525-190700-b91c",
		Commit:  "abc1234",
		Summary: "Completed the ticket after backend tests passed.",
		Evidence: []string{
			"`go test ./...` passed in `apps/core`.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `# Event: ticket.completed

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

- ` + "`go test ./...` passed in `apps/core`." + `
`
	if markdown != want {
		t.Fatalf("unexpected markdown:\n%s", markdown)
	}
}

func TestRenderEventMarkdownRequiresStructuredFields(t *testing.T) {
	_, err := RenderEventMarkdown(Event{
		ID:      "E-20260525-191100-codex-4c9a",
		Time:    time.Now(),
		Actor:   "codex",
		Project: "orion-a31f",
		Type:    "ticket.completed",
		Object:  "T-20260525-190700-b91c",
	})
	if err == nil {
		t.Fatal("expected missing summary error")
	}
}
