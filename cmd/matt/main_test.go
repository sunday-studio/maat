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

func TestStatusAgentUseOutputsJSONUpdates(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store, "--agent-use")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "Projects:") {
		t.Fatalf("agent output should not include human prose: %q", output)
	}
	var update map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &update); err != nil {
		t.Fatal(err)
	}
	if update["type"] != "maat.update" || update["step"] != "status.ready" || update["status"] != "ok" {
		t.Fatalf("unexpected update: %#v", update)
	}
}

func TestAgentUseRejectsJSONFlag(t *testing.T) {
	if _, err := captureRun("status", "--agent-use", "--json"); err == nil {
		t.Fatal("expected --agent-use with --json to fail")
	}
}

func TestHumanOutputCanUseColor(t *testing.T) {
	t.Setenv("MATT_COLOR", "always")
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "\x1b[") || !strings.Contains(output, "status ready") {
		t.Fatalf("expected colored human output, got %q", output)
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
	if len(projects) != 1 || projects[0].ID != "sample" || projects[0].Title != "Sample" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}

func TestVersionCommand(t *testing.T) {
	output, err := captureRun("version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "matt dev") {
		t.Fatalf("unexpected version output: %q", output)
	}

	output, err = captureRun("version", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != "dev" || got["commit"] == "" || got["date"] == "" {
		t.Fatalf("unexpected version json: %#v", got)
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
	if project.LegacyPath != "projects/sample.md" || project.ProjectPath != "projects/sample/project.md" {
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
	legacyPath := filepath.Join(store, "projects", "sample.md")
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
	if _, err := os.Stat(filepath.Join(dest, "projects", "sample", "project.md")); err != nil {
		t.Fatalf("expected migrated project file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, "projects", "sample", "project.md")); !os.IsNotExist(err) {
		t.Fatalf("source store should not receive target layout file, got err=%v", err)
	}

	objectStore, err := maat.LoadObjectStore(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(objectStore.Projects) != 1 || objectStore.Projects[0].Key != "sample" {
		t.Fatalf("unexpected migrated object store: %#v", objectStore.Projects)
	}
}

func TestProjectLinkCommand(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitStore(t, source)
	runGit(t, source, "remote", "add", "origin", "git@github.com:sunday-studio/sample.git")

	output, err := captureRun("project", "link", source, "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "linked project sample") || !strings.Contains(output, "remote git@github.com:sunday-studio/sample.git") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if project.DisplayName != "Sample" || project.Identity["Remote"] != "git@github.com:sunday-studio/sample.git" {
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

func TestProjectShowCommandSupportsObjectProject(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "photo-system",
		DisplayName: "Photo System",
	}); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("project", "show", "photo-system", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Photo System", "Key:     photo-system", "Repo:    " + source} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to include %q, got %q", want, output)
		}
	}
}

func TestProjectShowCommandJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("project", "show", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var legacy maat.Project
	if err := json.Unmarshal([]byte(output), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.ID != "sample" || legacy.Title != "Sample" || len(legacy.Goals) != 1 {
		t.Fatalf("unexpected legacy project json: %#v", legacy)
	}

	objectStore := writeObjectCommandFixture(t)
	output, err = captureRun("project", "show", "sample", "--storage", objectStore, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var object maat.ObjectProject
	if err := json.Unmarshal([]byte(output), &object); err != nil {
		t.Fatal(err)
	}
	if object.Key != "sample" || object.DisplayName != "Sample" {
		t.Fatalf("unexpected object project json: %#v", object)
	}
}

func TestGoalCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)

	output, err := captureRun("goal", "create", "sample", "Ship command writes", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created goal") || !strings.Contains(output, "event") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
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

	output, err := captureRun("goal", "create", "sample", "Ship json writes", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "goal.created" || result.ProjectKey != "sample" || result.GoalID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
}

func TestGoalCreateCommandInfersLinkedProject(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("goal", "create", "Inferred goal", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.GoalID == "" {
		t.Fatalf("unexpected inferred goal result: %#v", result)
	}
}

func TestTicketCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)

	output, err := captureRun("ticket", "create", "sample", "Wire ticket command", "--goal", goalID, "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created ticket") || !strings.Contains(output, "event") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
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

	output, err := captureRun("ticket", "create", "sample", "Wire json ticket", "--goal", goalID, "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "ticket.created" || result.ProjectKey != "sample" || result.GoalID != goalID || result.TicketID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
}

func TestTicketCreateCommandInfersLinkedProject(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("ticket", "create", "Inferred ticket", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.TicketID == "" {
		t.Fatalf("unexpected inferred ticket result: %#v", result)
	}
}

func TestTicketListAndShowCommands(t *testing.T) {
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	output, err := captureRun("ticket", "list", "--project", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tickets []ticketView
	if err := json.Unmarshal([]byte(output), &tickets); err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != ticketID || tickets[0].ProjectKey != "sample" {
		t.Fatalf("unexpected tickets: %#v", tickets)
	}

	output, err = captureRun("ticket", "show", ticketID, "--project", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var ticket ticketView
	if err := json.Unmarshal([]byte(output), &ticket); err != nil {
		t.Fatal(err)
	}
	if ticket.ID != ticketID || ticket.Status != "active" || ticket.Title != "Existing ticket" {
		t.Fatalf("unexpected ticket: %#v", ticket)
	}
}

func TestTicketListInfersLinkedProject(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	writer := maat.NewWriteStore(store)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "other",
		DisplayName: "Other",
	}); err != nil {
		t.Fatal(err)
	}
	sampleTicket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: "sample",
		Title:      "Sample ticket",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: "other",
		Title:      "Other ticket",
		Actor:      "test",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("ticket", "list", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tickets []ticketView
	if err := json.Unmarshal([]byte(output), &tickets); err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != sampleTicket.ID || tickets[0].ProjectKey != "sample" {
		t.Fatalf("expected linked project ticket only, got %#v", tickets)
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

	project, err := maat.LoadObjectProject(store, "sample")
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

func TestTicketEventCommandsInferLinkedProjectWhenTicketIDIsDuplicated(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	writer := maat.NewWriteStore(store)
	ticket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: "sample",
		Title:      "Shared ticket id",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "other",
		DisplayName: "Other",
	}); err != nil {
		t.Fatal(err)
	}
	writeDuplicateTicket(t, store, "other", ticket.ID)
	t.Chdir(source)

	output, err := captureRun("ticket", "comment", ticket.ID, "Inferred duplicate", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.TicketID != ticket.ID {
		t.Fatalf("expected linked project inference, got %#v", result)
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
	if claim.Action != "ticket.claimed" || claim.TicketID != ticketID || claim.ProjectKey != "sample" || claim.Agent != "claude" || claim.ExpiresAt == "" || claim.EventID == "" {
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
	if comment.Action != "ticket.commented" || comment.TicketID != ticketID || comment.ProjectKey != "sample" || comment.EventID == "" {
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
	if complete.Action != "ticket.completed" || complete.TicketID != ticketID || complete.ProjectKey != "sample" || complete.EventID == "" {
		t.Fatalf("unexpected complete json: %#v", complete)
	}
}

func TestTicketCreateCommandRequiresProjectAndTitle(t *testing.T) {
	store := writeObjectCommandFixture(t)

	_, err := captureRun("ticket", "create", "Only title", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "project key is required") {
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
		Key:         "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func createCommandGoal(t *testing.T, store string) string {
	t.Helper()

	writer := maat.NewWriteStore(store)
	goal, _, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: "sample",
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
		ProjectKey: "sample",
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
	if err := os.WriteFile(filepath.Join(root, "projects", "sample.md"), []byte(`# Project: Sample

| Field | Value |
|---|---|
| ID | sample |
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

func writeDuplicateTicket(t *testing.T, store, projectKey, ticketID string) {
	t.Helper()
	path := filepath.Join(store, "projects", projectKey, "tickets", ticketID+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `# Ticket: Duplicate Ticket

| Field | Value |
|---|---|
| Ticket ID | ` + ticketID + ` |
| Project | ` + projectKey + ` |
| Goal | none |
| Status | active |
| Created | 2026-05-25T10:00:00Z |
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
