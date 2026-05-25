package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
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

func TestProjectLinkCommand(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Orion\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitStore(t, source)
	runGit(t, source, "remote", "add", "origin", "git@github.com:sunday-studio/orion.git")

	output, err := captureRun("project", "link", source, "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "linked project orion") || !strings.Contains(output, "remote git@github.com:sunday-studio/orion.git") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "orion")
	if err != nil {
		t.Fatal(err)
	}
	if project.DisplayName != "Orion" || project.Identity["Remote"] != "git@github.com:sunday-studio/orion.git" {
		t.Fatalf("unexpected linked project: %#v", project)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index after link: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index after link: %v", err)
	}
}

func TestProjectLinkCommandJSONAndIdempotent(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()

	output, err := captureRun("project", "link", source, "--storage", store, "--key", "photo-system", "--name", "Photo System", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var linked maat.LinkedProject
	if err := json.Unmarshal([]byte(output), &linked); err != nil {
		t.Fatal(err)
	}
	if !linked.Created || linked.ProjectKey != "photo-system" || linked.DisplayName != "Photo System" {
		t.Fatalf("unexpected link json: %#v", linked)
	}

	output, err = captureRun("project", "link", source, "--storage", store, "--key", "photo-system", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(output), &linked); err != nil {
		t.Fatal(err)
	}
	if !linked.Existing || linked.Created {
		t.Fatalf("expected idempotent existing project, got %#v", linked)
	}
}

func TestGoalCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)

	output, err := captureRun("goal", "create", "orion", "Ship command writes", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created goal") || !strings.Contains(output, "event") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "orion")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Goals) != 1 || project.Goals[0].Title != "Ship command writes" {
		t.Fatalf("unexpected goals: %#v", project.Goals)
	}
	if len(project.Events) != 1 || project.Events[0].Type != "goal.created" {
		t.Fatalf("unexpected events: %#v", project.Events)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index after write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index after write: %v", err)
	}
}

func TestGoalCreateCommandJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)

	output, err := captureRun("goal", "create", "orion", "Ship json writes", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "goal.created" || result.ProjectKey != "orion" || result.GoalID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
}

func TestTicketCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)

	output, err := captureRun("ticket", "create", "orion", "Wire ticket command", "--goal", goalID, "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created ticket") || !strings.Contains(output, "event") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "orion")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Tickets) != 1 || project.Tickets[0].Title != "Wire ticket command" {
		t.Fatalf("unexpected tickets: %#v", project.Tickets)
	}
	if project.Tickets[0].GoalID != goalID {
		t.Fatalf("expected goal link %q, got %q", goalID, project.Tickets[0].GoalID)
	}
}

func TestTicketCreateCommandJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)

	output, err := captureRun("ticket", "create", "orion", "Wire json ticket", "--goal", goalID, "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "ticket.created" || result.ProjectKey != "orion" || result.GoalID != goalID || result.TicketID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
}

func TestTicketEventCommands(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	if output, err := captureRun("ticket", "claim", ticketID, "--agent", "claude", "--ttl", "30m", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "claimed ticket") {
		t.Fatalf("unexpected claim output: %q", output)
	}
	if output, err := captureRun("ticket", "comment", ticketID, "Progress note", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "commented on ticket") {
		t.Fatalf("unexpected comment output: %q", output)
	}
	if output, err := captureRun("ticket", "complete", ticketID, "--evidence", "go test ./...", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "completed ticket") {
		t.Fatalf("unexpected complete output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "orion")
	if err != nil {
		t.Fatal(err)
	}
	eventTypes := map[string]bool{}
	for _, event := range project.Events {
		eventTypes[event.Type] = true
	}
	for _, eventType := range []string{"ticket.claimed", "ticket.commented", "ticket.completed"} {
		if !eventTypes[eventType] {
			t.Fatalf("missing event type %s in %#v", eventType, project.Events)
		}
	}
}

func TestTicketEventCommandsJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	output, err := captureRun("ticket", "claim", ticketID, "--agent", "claude", "--ttl", "30m", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var claim writeCommandResult
	if err := json.Unmarshal([]byte(output), &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Action != "ticket.claimed" || claim.TicketID != ticketID || claim.ProjectKey != "orion" || claim.Agent != "claude" || claim.ExpiresAt == "" || claim.EventID == "" {
		t.Fatalf("unexpected claim json: %#v", claim)
	}

	output, err = captureRun("ticket", "comment", ticketID, "Progress note", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var comment writeCommandResult
	if err := json.Unmarshal([]byte(output), &comment); err != nil {
		t.Fatal(err)
	}
	if comment.Action != "ticket.commented" || comment.TicketID != ticketID || comment.ProjectKey != "orion" || comment.EventID == "" {
		t.Fatalf("unexpected comment json: %#v", comment)
	}

	output, err = captureRun("ticket", "complete", ticketID, "--evidence", "go test ./...", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var complete writeCommandResult
	if err := json.Unmarshal([]byte(output), &complete); err != nil {
		t.Fatal(err)
	}
	if complete.Action != "ticket.completed" || complete.TicketID != ticketID || complete.ProjectKey != "orion" || complete.EventID == "" {
		t.Fatalf("unexpected complete json: %#v", complete)
	}
}

func TestTicketCreateCommandRequiresProjectAndTitle(t *testing.T) {
	store := writeObjectCommandFixture(t)

	_, err := captureRun("ticket", "create", "Only title", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "project key and ticket title are required") {
		t.Fatalf("expected project/title error, got %v", err)
	}
}

func TestTicketCompleteRequiresEvidence(t *testing.T) {
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	_, err := captureRun("ticket", "complete", ticketID, "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "--evidence is required") {
		t.Fatalf("expected evidence error, got %v", err)
	}
}

func TestAgentInstructionsCommand(t *testing.T) {
	output, err := captureRun("agent", "instructions")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"matt sync",
		"matt status",
		"matt project show <project>",
		"matt search <query>",
		"Create or claim a ticket",
		"complete the ticket with evidence",
		"Do not mark work done without evidence",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to include %q, got %q", want, output)
		}
	}
}

func TestAgentInstructionsCommandJSONAndOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")

	output, err := captureRun("agent", "instructions", "--json", "--output", path)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["instructions"] != maat.AgentInstructionsSnippet() {
		t.Fatalf("unexpected instructions payload: %#v", payload)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != maat.AgentInstructionsSnippet()+"\n" {
		t.Fatalf("unexpected written snippet: %q", string(written))
	}
}

func TestSyncStatusCommandReportsDirtyState(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "scratch.md"), []byte("# Scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Repository: git repository") || !strings.Contains(output, "Dirty:") || !strings.Contains(output, "scratch.md") {
		t.Fatalf("unexpected output: %q", output)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat")); !os.IsNotExist(err) {
		t.Fatalf("status command should not rebuild indexes, got err=%v", err)
	}
}

func TestSyncCommandCommitsChanges(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "docs", "sync.md"), []byte("# Sync\n\nCommit me.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--message", "status(maat): test sync")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Validation:") || !strings.Contains(output, "Committed:  status(maat): test sync") {
		t.Fatalf("unexpected output: %q", output)
	}
	status := runGit(t, store, "status", "--porcelain=v1")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("expected clean git status after sync, got %q", status)
	}
	log := runGit(t, store, "log", "-1", "--pretty=%s")
	if strings.TrimSpace(log) != "status(maat): test sync" {
		t.Fatalf("unexpected commit subject: %q", log)
	}
}

func TestSyncCommandJSON(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "docs", "json.md"), []byte("# JSON sync\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result maat.StoreSyncResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Committed || result.CommitMessage != "status(maat): sync store" {
		t.Fatalf("unexpected sync result: %#v", result)
	}
	if result.SQLiteIndex.Path == "" || result.JSONIndexPath == "" {
		t.Fatalf("expected rebuilt indexes: %#v", result)
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

func initGitStore(t *testing.T, store string) {
	t.Helper()
	runGit(t, store, "init", "-b", "main")
	runGit(t, store, "config", "user.email", "maat@example.test")
	runGit(t, store, "config", "user.name", "Maat Test")
	runGit(t, store, "add", ".")
	runGit(t, store, "commit", "-m", "test: seed store")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writeObjectCommandFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writer := maat.NewWriteStore(root)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "orion",
		DisplayName: "Orion",
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func createCommandGoal(t *testing.T, store string) string {
	t.Helper()

	writer := maat.NewWriteStore(store)
	goal, _, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: "orion",
		Title:      "Existing goal",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return goal.ID
}

func createCommandTicket(t *testing.T, store string) string {
	t.Helper()

	writer := maat.NewWriteStore(store)
	ticket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: "orion",
		Title:      "Existing ticket",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return ticket.ID
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
