package maat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteStoreCreatesProject(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 5, 0, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{Root: root, Now: func() time.Time { return at }}

	project, err := store.CreateProject(CreateProjectInput{
		Key:         "Sample A31F",
		DisplayName: "Sample",
		Tags:        []string{"#infra", "#agent-run"},
		Summary:     "Self-hosted monitoring app.",
		PrimaryRepo: "R-20260525-190100-a31f",
		Remote:      "git@github.com:sunday-studio/sample.git",
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Key != "sample-a31f" || project.DisplayName != "Sample" || project.Status != "active" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if project.Path != "projects/sample-a31f/project.md" {
		t.Fatalf("unexpected path: %q", project.Path)
	}
	if project.Identity["Remote"] != "git@github.com:sunday-studio/sample.git" {
		t.Fatalf("unexpected identity: %#v", project.Identity)
	}
	for _, dir := range []string{"goals", "tickets", "events"} {
		if _, err := os.Stat(filepath.Join(root, "projects", "sample-a31f", dir)); err != nil {
			t.Fatalf("expected %s directory: %v", dir, err)
		}
	}
}

func TestWriteStoreUsesStateDirectoryWhenPresent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 5, 26, 10, 45, 0, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{Root: root, Now: func() time.Time { return at }}

	project, err := store.CreateProject(CreateProjectInput{
		Key:         "maat",
		DisplayName: "Maat",
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Path != "state/projects/maat/project.md" {
		t.Fatalf("unexpected path: %q", project.Path)
	}
	if _, err := os.Stat(filepath.Join(root, "state", "projects", "maat", "project.md")); err != nil {
		t.Fatalf("expected state project file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "maat", "project.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no root project file, got %v", err)
	}
}

func TestWriteStoreCreatesGoalAndEvent(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 5, 33, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{Root: root, Entropy: strings.NewReader("\xa7\xf3\x4c\x9a")}
	createProjectFixture(t, store, at)

	goal, event, err := store.CreateGoal(CreateGoalInput{
		ProjectKey: "sample-a31f",
		Title:      "Improve Agent Health Clarity",
		Tags:       []string{"#backend", "#frontend"},
		Outcome:    "Agent health should explain stale data and check failures.",
		Actor:      "Codex",
		At:         at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if goal.ID != "G-20260525-190533-a7f3" || goal.ProjectKey != "sample-a31f" {
		t.Fatalf("unexpected goal: %#v", goal)
	}
	if event.ID != "E-20260525-190533-codex-4c9a" || event.Type != "goal.created" || event.ObjectID != goal.ID {
		t.Fatalf("unexpected event: %#v", event)
	}
	if event.Path != "projects/sample-a31f/events/2026/05/E-20260525-190533-codex-4c9a.md" {
		t.Fatalf("unexpected event path: %q", event.Path)
	}
}

func TestWriteStoreCreatesStandaloneTicketAndEvent(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 7, 0, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{Root: root, Entropy: strings.NewReader("\xb9\x1c\xc3\xd4")}
	createProjectFixture(t, store, at)

	ticket, event, err := store.CreateTicket(CreateTicketInput{
		ProjectKey:  "sample-a31f",
		Title:       "Fix Broken Deploy Doc Link",
		Description: "Correct the stale installer reference.",
		Acceptance:  []string{"Link points at the current installer.", "Docs render cleanly."},
		Actor:       "codex",
		At:          at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ticket.ID != "T-20260525-190700-b91c" || ticket.GoalID != "" {
		t.Fatalf("unexpected ticket: %#v", ticket)
	}
	content := readFile(t, filepath.Join(root, "projects", "sample-a31f", "tickets", ticket.ID+".md"))
	if !strings.Contains(content, "| Goal | none |") {
		t.Fatalf("expected standalone goal marker, got:\n%s", content)
	}
	if event.ID != "E-20260525-190700-codex-c3d4" || event.Type != "ticket.created" || event.ObjectID != ticket.ID {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestWriteStoreCreatesTicketLifecycleEvents(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 7, 0, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{Root: root, Entropy: strings.NewReader("\x10\x01\x10\x02\x10\x03\x10\x04\x10\x05\x10\x06\x10\x07\x10\x08")}
	createProjectFixture(t, store, at)
	ticket, _, err := store.CreateTicket(CreateTicketInput{
		ProjectKey: "sample-a31f",
		Title:      "Separate Agent Availability From Monitor Health",
		Actor:      "codex",
		At:         at,
	})
	if err != nil {
		t.Fatal(err)
	}

	comment, err := store.CommentTicket(TicketCommentInput{
		ProjectKey: "sample-a31f",
		TicketID:   ticket.ID,
		Actor:      "codex",
		Comment:    "Found the status rollup issue.",
		At:         at.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	claim, err := store.ClaimTicket(ClaimTicketInput{
		ProjectKey: "sample-a31f",
		TicketID:   ticket.ID,
		Actor:      "claude",
		ExpiresAt:  at.Add(2 * time.Hour),
		At:         at.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := store.CompleteTicket(CompleteTicketInput{
		ProjectKey: "sample-a31f",
		TicketID:   ticket.ID,
		Actor:      "codex",
		Summary:    "Completed the ticket after backend tests passed.",
		Evidence:   []string{"go test ./... passed"},
		At:         at.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	if comment.Type != "ticket.commented" || claim.Type != "ticket.claimed" || completed.Type != "ticket.completed" {
		t.Fatalf("unexpected events: %#v %#v %#v", comment, claim, completed)
	}
	claimContent := readFile(t, filepath.Join(root, filepath.FromSlash(claim.Path)))
	if !strings.Contains(claimContent, "| Expires | 2026-05-25T21:07:00+02:00 |") {
		t.Fatalf("expected claim expiration, got:\n%s", claimContent)
	}
	completedContent := readFile(t, filepath.Join(root, filepath.FromSlash(completed.Path)))
	if !strings.Contains(completedContent, "## Evidence") || !strings.Contains(completedContent, "- go test ./... passed") {
		t.Fatalf("expected completion evidence, got:\n%s", completedContent)
	}
}

func TestWriteStoreRequiresCompletionEvidence(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 7, 0, 0, time.UTC)
	store := WriteStore{Root: root, Entropy: strings.NewReader("\x10\x01\x10\x02")}
	createProjectFixture(t, store, at)
	ticket, _, err := store.CreateTicket(CreateTicketInput{
		ProjectKey: "sample-a31f",
		Title:      "Add evidence guard",
		Actor:      "codex",
		At:         at,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.CompleteTicket(CompleteTicketInput{
		ProjectKey: "sample-a31f",
		TicketID:   ticket.ID,
		Actor:      "codex",
		At:         at.Add(time.Minute),
	})
	if err == nil {
		t.Fatal("expected completion evidence error")
	}
}

func createProjectFixture(t *testing.T, store WriteStore, at time.Time) {
	t.Helper()
	_, err := store.CreateProject(CreateProjectInput{
		Key:         "sample-a31f",
		DisplayName: "Sample",
		CreatedAt:   at,
		UpdatedAt:   at,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
