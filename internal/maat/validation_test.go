package maat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateStorePassesObjectLayoutWrittenByCLI(t *testing.T) {
	root := t.TempDir()
	at := time.Date(2026, 5, 25, 19, 5, 0, 0, time.FixedZone("CEST", 2*60*60))
	store := WriteStore{
		Root:    root,
		Now:     func() time.Time { return at },
		Entropy: strings.NewReader("\xa7\xf3\xb9\x1c\xc3\xd4\xe5\xf6\x10\x01"),
	}
	if _, err := store.CreateProject(CreateProjectInput{
		Key:         "maat",
		DisplayName: "Maat",
	}); err != nil {
		t.Fatal(err)
	}
	goal, _, err := store.CreateGoal(CreateGoalInput{
		ProjectKey: "maat",
		Title:      "Ship first deploy",
		Actor:      "codex",
		At:         at.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	ticket, _, err := store.CreateTicket(CreateTicketInput{
		ProjectKey: "maat",
		GoalID:     goal.ID,
		Title:      "Validate object layout",
		Actor:      "codex",
		At:         at.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CommentTicket(TicketCommentInput{
		ProjectKey: "maat",
		TicketID:   ticket.ID,
		Actor:      "codex",
		Comment:    "Validation smoke.",
		At:         at.Add(3 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("expected object layout to validate, got %#v", report.Issues)
	}
	if report.Files != 6 {
		t.Fatalf("expected 6 validated files, got %d", report.Files)
	}
}

func TestValidateStoreDetectsObjectLayoutIssues(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects", "maat")
	if err := os.MkdirAll(filepath.Join(projectDir, "goals"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "events", "2026", "05"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeProject(t, projectDir, "project.md", `# Project: Maat

| Field | Value |
|---|---|
| Project Key | wrong |
| Display Name | Maat |
| Status | mystery |
| Created | yesterday |
| Updated | 2026-05-25T19:05:00+02:00 |
| malformed
`)
	writeProject(t, filepath.Join(projectDir, "goals"), "G-1.md", `# Goal: Ship

| Field | Value |
|---|---|
| Goal ID | G-2 |
| Project | other |
| Status | lost |
| Created | not-a-time |
`)
	writeProject(t, filepath.Join(projectDir, "tickets"), "T-1.md", `# Ticket: Verify

| Field | Value |
|---|---|
| Ticket ID | T-2 |
| Project | other |
| Goal | G-missing |
| Status | lost |
| Created | not-a-time |
`)
	writeProject(t, filepath.Join(projectDir, "events", "2026", "05"), "E-1.md", `# Event: ticket.completed

| Field | Value |
|---|---|
| Event ID | E-2 |
| Time | 2026-06-01T10:00:00+02:00 |
| Actor | codex |
| Project | other |
| Type | ticket.completed |
| Object | T-missing |
`)

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatalf("expected object layout issues")
	}
	for _, code := range []string{
		"project_key_mismatch",
		"invalid_project_status",
		"invalid_project_timestamp",
		"malformed_table_row",
		"goal_project_mismatch",
		"goal_id_filename_mismatch",
		"invalid_goal_status",
		"invalid_goal_timestamp",
		"ticket_project_mismatch",
		"ticket_id_filename_mismatch",
		"unknown_ticket_goal",
		"invalid_ticket_status",
		"invalid_ticket_timestamp",
		"event_project_mismatch",
		"event_id_filename_mismatch",
		"event_time_path_mismatch",
		"missing_event_summary",
		"unknown_event_object",
	} {
		assertIssue(t, report, code)
	}
}

func TestValidateStorePassesCurrentFixture(t *testing.T) {
	report, err := ValidateStore(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("expected current fixture to validate, got %#v", report.Issues)
	}
}

func writeProject(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertIssue(t *testing.T, report ValidationReport, code string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("expected issue %q in %#v", code, report.Issues)
}
