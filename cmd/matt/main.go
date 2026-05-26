package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunday-studio/maat/internal/maat"
	"github.com/sunday-studio/maat/internal/version"
)

var agentUse bool
var jsonUse bool

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "matt: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	agentUse = false
	jsonUse = false
	var err error
	args, err = splitGlobalFlags(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "version":
		return versionCommand(args[1:])
	case "initialize":
		return agentInitializeCommand(args[1:])
	case "init":
		return initConfig(args[1:])
	case "storage":
		return storageCommand(args[1:])
	case "index":
		return indexCommand(args[1:])
	case "projects":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		progress("projects.load", "loading projects", map[string]string{"storage": store})
		projects, err := loadProjectListItems(store)
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("projects.loaded", "ok", "projects loaded", map[string]any{
				"projects": projects,
				"count":    len(projects),
			})
		}
		if jsonOut {
			return writeJSON(projects)
		}
		ok("projects.loaded", fmt.Sprintf("loaded %d projects", len(projects)), nil)
		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return nil
		}
		fmt.Printf("%-18s %-9s %-8s %s\n", "Project", "Status", "Layout", "Title")
		for _, project := range projects {
			fmt.Printf("%-18s %-9s %-8s %s\n", project.ID, colorStatus(project.Status), project.Layout, project.Title)
		}
		return nil
	case "project":
		return projectCommand(args[1:])
	case "goal":
		return goalCommand(args[1:])
	case "ticket":
		return ticketCommand(args[1:])
	case "agent":
		return agentCommand(args[1:])
	case "status":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		summary, err := maat.Status(store)
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("status.ready", "ok", "status ready", summary)
		}
		if jsonOut {
			return writeJSON(summary)
		}
		ok("status.ready", "status ready", nil)
		fmt.Println("Maat status")
		printField("Projects", colorNumber(summary.Projects))
		printField("Goals", fmt.Sprintf("%s active, %s done, %s total", colorNumber(summary.ActiveGoals), colorNumber(summary.DoneGoals), colorNumber(summary.Goals)))
		printField("Tickets", fmt.Sprintf("%s open, %s done, %s total", colorNumber(summary.OpenTickets), colorNumber(summary.DoneTickets), colorNumber(summary.Tickets)))
		return nil
	case "validate":
		return validateCommand(args[1:])
	case "migrate":
		return migrateCommand(args[1:])
	case "sync":
		return syncCommand(args[1:])
	case "search":
		return searchCommand(args[1:])
	case "tui":
		return tuiCommand(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Print(`matt - Git-backed project state for agents

Usage:
  matt <command> [flags]

Common:
  matt initialize [--project <project-key>] [--storage <path>] [--json]
  matt status [--storage <path>] [--json]
  matt projects [--storage <path>] [--json]
  matt search <query> [--storage <path>] [--json]
  matt sync [--storage <path>] [--message <msg>] [--push] [--status] [--json]

Projects:
  matt project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]
  matt project show <project-id> [--storage <path>] [--json]
  matt goal create [project-key] <title> [--storage <path>] [--json]

Tickets:
  matt ticket create [project-key] <title> [--goal <goal-id>] [--storage <path>] [--json]
  matt ticket list [--project <project-key>] [--storage <path>] [--json]
  matt ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]
  matt ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]
  matt ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]
  matt ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]

Setup and maintenance:
  matt init [storage-path]
  matt storage link <storage-path>
  matt index rebuild [--storage <path>]
  matt validate [--storage <path>] [--json]
  matt migrate plan [--storage <path>] [--json]
  matt migrate apply --dest <path> [--storage <path>]
  matt agent initialize [--project <project-key>] [--storage <path>] [--json]
  matt agent instructions [--json] [--output <path>]
  matt tui [--storage <path>]
  matt version [--json]

Global flags:
  --agent-use   emit newline-delimited JSON updates for agents

Markdown plus Git is the source of truth. The SQLite index is a rebuildable local cache.
`)
}

func versionCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	if len(filtered) > 0 {
		return errors.New("usage: matt version [--json]")
	}
	info := version.Current()
	if agentUse {
		return agentUpdate("version.ready", "ok", "version ready", info)
	}
	if jsonOut {
		return writeJSON(info)
	}
	fmt.Println(info.String())
	return nil
}

func initConfig(args []string) error {
	storagePath := ""
	if len(args) > 0 {
		storagePath = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		storagePath = cwd
	}
	return writeConfig(storagePath)
}

func storageCommand(args []string) error {
	if len(args) != 2 || args[0] != "link" {
		return errors.New("usage: matt storage link <storage-path>")
	}
	return writeConfig(args[1])
}

func indexCommand(args []string) error {
	if len(args) == 0 || args[0] != "rebuild" {
		return errors.New("usage: matt index rebuild [--storage <path>]")
	}
	store, err := loadStore(args[1:])
	if err != nil {
		return err
	}
	progress("index.read", "reading markdown store", map[string]string{"storage": store})
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return err
	}
	progress("index.json", "writing json index", nil)
	path, err := maat.WriteIndex(store, idx)
	if err != nil {
		return err
	}
	progress("index.sqlite", "building sqlite index", nil)
	sqliteInfo, err := maat.RebuildSQLiteIndex(store)
	if err != nil {
		return err
	}
	if agentUse {
		return agentUpdate("index.ready", "ok", "index rebuilt", map[string]any{
			"projects":     len(idx.Projects),
			"documents":    len(idx.Documents),
			"json_path":    path,
			"sqlite_index": sqliteInfo,
		})
	}
	ok("index.ready", fmt.Sprintf("indexed %d projects and %d documents", len(idx.Projects), len(idx.Documents)), nil)
	fmt.Printf("indexed %d projects and %d documents\n", len(idx.Projects), len(idx.Documents))
	fmt.Printf("json:   %s\n", path)
	fts := "disabled"
	if sqliteInfo.FTS {
		fts = "enabled"
	}
	fmt.Printf("sqlite: %s (%d documents, fts %s)\n", sqliteInfo.Path, sqliteInfo.Documents, fts)
	return nil
}

func syncCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	message, rest, err := consumeFlagValue(rest, "--message", false)
	if err != nil {
		return err
	}

	push := false
	statusOnly := false
	for _, arg := range rest {
		switch arg {
		case "--push":
			push = true
		case "--status":
			statusOnly = true
		default:
			return fmt.Errorf("unexpected sync argument %q", arg)
		}
	}
	if statusOnly && push {
		return errors.New("--push cannot be used with --status")
	}
	if !statusOnly && strings.TrimSpace(message) == "" {
		message = "status(maat): sync store"
	}

	progress("sync.start", "checking git storage", map[string]any{
		"storage":     store,
		"status_only": statusOnly,
		"push":        push,
	})
	result, err := maat.SyncStore(context.Background(), maat.StoreSyncOptions{
		Store:        store,
		Message:      message,
		Push:         push,
		SkipIndex:    statusOnly,
		SkipValidate: statusOnly,
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(result)
	}
	if agentUse {
		return agentUpdate("sync.ready", "ok", "sync complete", result)
	}
	printSyncResult(result, statusOnly)
	return nil
}

func printSyncResult(result maat.StoreSyncResult, statusOnly bool) {
	repo := "not a git repository"
	if result.Repository.IsRepository {
		repo = "git repository"
		if result.Repository.Branch != "" {
			repo += " on " + result.Repository.Branch
		}
	}
	fmt.Printf("Repository: %s\n", repo)
	if result.Validation.Files > 0 {
		fmt.Printf("Validation: %d files, ok\n", result.Validation.Files)
	}
	if result.JSONIndexPath != "" {
		fmt.Printf("Index:      %s\n", result.JSONIndexPath)
	}
	if result.SQLiteIndex.Path != "" {
		fts := "disabled"
		if result.SQLiteIndex.FTS {
			fts = "enabled"
		}
		fmt.Printf("SQLite:     %s (%d documents, fts %s)\n", result.SQLiteIndex.Path, result.SQLiteIndex.Documents, fts)
	}
	if statusOnly {
		printSyncDirty("Dirty", result.DirtyBeforeCommit)
		return
	}
	if result.Committed {
		fmt.Printf("Committed:  %s\n", result.CommitMessage)
	} else {
		fmt.Println("Committed:  no changes")
	}
	if result.Pushed {
		fmt.Println("Pushed:     yes")
	}
	printSyncDirty("Remaining", result.DirtyAfterSync)
}

func printSyncDirty(label string, entries []maat.GitStatusEntry) {
	if len(entries) == 0 {
		fmt.Printf("%s:      clean\n", label)
		return
	}
	fmt.Printf("%s:      %d changes\n", label, len(entries))
	for _, entry := range entries {
		fmt.Printf("  %c%c %s\n", entry.Index, entry.Work, entry.Path)
	}
}

func validateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, err := loadStore(filtered)
	if err != nil {
		return err
	}
	progress("validate.start", "validating markdown store", map[string]string{"storage": store})
	report, err := maat.ValidateStore(store)
	if err != nil {
		return err
	}
	if agentUse {
		status := "ok"
		message := "validation passed"
		if !report.OK() {
			status = "error"
			message = "validation failed"
		}
		if err := agentUpdate("validate.ready", status, message, report); err != nil {
			return err
		}
		if !report.OK() {
			return fmt.Errorf("validation failed with %d issues", len(report.Issues))
		}
		return nil
	}
	if jsonOut {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else if report.OK() {
		ok("validate.ready", fmt.Sprintf("validated %d files: ok", report.Files), nil)
	} else {
		warn("validate.ready", fmt.Sprintf("validated %d files: %d issues", report.Files, len(report.Issues)), nil)
		for _, issue := range report.Issues {
			location := issue.Path
			if issue.Line > 0 {
				location = fmt.Sprintf("%s:%d", location, issue.Line)
			}
			fmt.Printf("%s [%s] %s\n", location, issue.Code, issue.Message)
		}
	}
	if !report.OK() {
		return fmt.Errorf("validation failed with %d issues", len(report.Issues))
	}
	return nil
}

func migrateCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: matt migrate plan [--storage <path>] [--json] or matt migrate apply --dest <path> [--storage <path>]")
	}
	switch args[0] {
	case "plan":
		filtered, jsonOut := splitJSONFlag(args[1:])
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		progress("migrate.plan", "planning legacy migration", map[string]string{"storage": store})
		plan, err := maat.PlanLegacyMigration(store, maat.MigrationOptions{})
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("migrate.plan.ready", "ok", "migration plan ready", plan)
		}
		if jsonOut {
			return writeJSON(plan)
		}
		ok("migrate.plan.ready", fmt.Sprintf("planned %d projects and %d files", len(plan.Projects), len(plan.Files)), nil)
		fmt.Printf("planned %d projects and %d files\n", len(plan.Projects), len(plan.Files))
		for _, project := range plan.Projects {
			fmt.Printf("%s -> %s (%d goals, %d tickets, %d events)\n",
				project.LegacyPath,
				project.ProjectPath,
				len(project.GoalPaths),
				len(project.TicketPaths),
				len(project.EventPaths),
			)
		}
		return nil
	case "apply":
		filtered, dest, err := splitDestinationFlag(args[1:])
		if err != nil {
			return err
		}
		store, err := loadStore(filtered)
		if err != nil {
			return err
		}
		absDest, err := filepath.Abs(dest)
		if err != nil {
			return err
		}
		progress("migrate.apply", "applying legacy migration", map[string]string{"destination": absDest})
		plan, err := maat.ApplyLegacyMigration(store, absDest, maat.MigrationOptions{})
		if err != nil {
			return err
		}
		if agentUse {
			return agentUpdate("migrate.apply.ready", "ok", "migration applied", plan)
		}
		ok("migrate.apply.ready", fmt.Sprintf("migrated %d projects", len(plan.Projects)), nil)
		fmt.Printf("migrated %d projects into %s\n", len(plan.Projects), absDest)
		fmt.Printf("wrote %d files\n", len(plan.Files))
		return nil
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func projectCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: matt project <show|link>")
	}
	switch args[0] {
	case "show":
		return projectShowCommand(args[1:])
	case "link":
		return projectLinkCommand(args[1:])
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func projectShowCommand(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: matt project show <project-id> [--storage <path>] [--json]")
	}
	filtered, jsonOut := splitJSONFlag(args)
	projectID := filtered[0]
	store, err := loadStore(filtered[1:])
	if err != nil {
		return err
	}
	project, err := maat.LoadProject(store, projectID)
	if err == nil {
		if agentUse {
			return agentUpdate("project.ready", "ok", "project loaded", project)
		}
		if jsonOut {
			return writeJSON(project)
		}
		printLegacyProject(project)
		return nil
	}
	objectProject, objectErr := maat.LoadObjectProject(store, projectID)
	if objectErr != nil {
		return err
	}
	if agentUse {
		return agentUpdate("project.ready", "ok", "project loaded", objectProject)
	}
	if jsonOut {
		return writeJSON(objectProject)
	}
	printObjectProject(objectProject)
	return nil
}

func printLegacyProject(project maat.Project) {
	fmt.Printf("# %s\n\n", project.Title)
	fmt.Printf("ID:      %s\n", project.ID)
	fmt.Printf("Status:  %s\n", project.Status)
	fmt.Printf("Updated: %s\n", project.Updated)
	fmt.Printf("Tags:    %s\n\n", strings.Join(project.Tags, " "))
	if project.Current != "" {
		fmt.Println(project.Current)
		fmt.Println()
	}
	for _, goal := range project.Goals {
		fmt.Printf("- %s [%s] %s\n", goal.ID, goal.Status, goal.Title)
		for _, ticket := range goal.Tickets {
			box := " "
			if ticket.Done {
				box = "x"
			}
			fmt.Printf("  - [%s] %s %s\n", box, ticket.ID, ticket.Title)
		}
	}
}

func printObjectProject(project maat.ObjectProject) {
	fmt.Printf("# %s\n\n", project.DisplayName)
	fmt.Printf("Key:     %s\n", project.Key)
	fmt.Printf("Status:  %s\n", project.Status)
	fmt.Printf("Updated: %s\n", project.Updated)
	if len(project.Tags) > 0 {
		fmt.Printf("Tags:    %s\n", strings.Join(project.Tags, " "))
	}
	if project.Identity["Primary Repo"] != "" {
		fmt.Printf("Repo:    %s\n", project.Identity["Primary Repo"])
	}
	if project.Identity["Remote"] != "" {
		fmt.Printf("Remote:  %s\n", project.Identity["Remote"])
	}
	if project.Summary != "" {
		fmt.Println()
		fmt.Println(project.Summary)
	}
	if len(project.Goals) > 0 {
		fmt.Println()
		for _, goal := range project.Goals {
			fmt.Printf("- %s [%s] %s\n", goal.ID, goal.Status, goal.Title)
		}
	}
	if len(project.Tickets) > 0 {
		fmt.Println()
		for _, ticket := range project.Tickets {
			goal := "standalone"
			if ticket.GoalID != "" {
				goal = ticket.GoalID
			}
			fmt.Printf("- %s [%s] %s (%s)\n", ticket.ID, ticket.Status, ticket.Title, goal)
		}
	}
}

func projectLinkCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	key, rest, err := consumeFlagValue(rest, "--key", false)
	if err != nil {
		return err
	}
	name, rest, err := consumeFlagValue(rest, "--name", false)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return errors.New("usage: matt project link [source-path] [--storage <path>] [--key <project-key>] [--name <display-name>] [--json]")
	}
	sourcePath := "."
	if len(rest) == 1 {
		sourcePath = rest[0]
	}
	progress("project.link", "linking source repo", map[string]string{"source": sourcePath})
	linked, err := maat.LinkProject(context.Background(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  sourcePath,
		ProjectKey:  key,
		DisplayName: name,
	})
	if err != nil {
		return err
	}
	warnIndexRefresh(refreshIndexesBestEffort(store))
	if agentUse {
		return agentUpdate("project.linked", "ok", "project linked", linked)
	}
	if jsonOut {
		return writeJSON(linked)
	}
	message := "linked project " + linked.ProjectKey
	if linked.Created {
		ok("project.linked", message, nil)
	} else {
		message = "project " + linked.ProjectKey + " already linked"
		ok("project.linked", message, nil)
	}
	printField("Project", linked.ProjectKey)
	printField("Name", linked.DisplayName)
	printField("Source", linked.SourcePath)
	if linked.RemoteURL != "" {
		printField("Remote", linked.RemoteURL)
	}
	return nil
}

func goalCommand(args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage: matt goal create <project-key> <title> [--storage <path>] [--json]")
	}
	filtered, jsonOut := splitJSONFlag(args[1:])
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey, title, err := resolveCreateProjectAndTitle(store, rest, "goal")
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	goal, event, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: projectKey,
		Title:      title,
		Actor:      defaultActor(),
	})
	if err != nil {
		return err
	}
	return finishWrite(store, goal.ProjectKey, writeCommandResult{
		Action:     "goal.created",
		ProjectKey: goal.ProjectKey,
		GoalID:     goal.ID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: matt ticket <create|list|show|claim|comment|complete>")
	}
	switch args[0] {
	case "create":
		return ticketCreateCommand(args[1:])
	case "list":
		return ticketListCommand(args[1:])
	case "show":
		return ticketShowCommand(args[1:])
	case "claim":
		return ticketClaimCommand(args[1:])
	case "comment":
		return ticketCommentCommand(args[1:])
	case "complete":
		return ticketCompleteCommand(args[1:])
	default:
		return fmt.Errorf("unknown ticket command %q", args[0])
	}
}

type ticketView struct {
	ID         string `json:"id"`
	ProjectKey string `json:"project_key"`
	GoalID     string `json:"goal_id,omitempty"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Created    string `json:"created,omitempty"`
	Path       string `json:"path,omitempty"`
	Legacy     bool   `json:"legacy,omitempty"`
}

func ticketListCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return errors.New("usage: matt ticket list [--project <project-key>] [--storage <path>] [--json]")
	}
	if projectKey == "" {
		if project, err := inferProjectFromWorkingDirectory(store); err == nil {
			projectKey = project.Key
		}
	}
	tickets, err := loadTicketViews(store, projectKey)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(tickets)
	}
	if agentUse {
		return agentUpdate("tickets.ready", "ok", "tickets loaded", map[string]any{
			"tickets": tickets,
			"count":   len(tickets),
		})
	}
	for _, ticket := range tickets {
		goal := "standalone"
		if ticket.GoalID != "" {
			goal = ticket.GoalID
		}
		fmt.Printf("%-30s %-10s %-9s %-12s %s\n", ticket.ID, ticket.ProjectKey, ticket.Status, goal, ticket.Title)
	}
	return nil
}

func ticketShowCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("usage: matt ticket show <ticket-id> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveReadableTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	tickets, err := loadTicketViews(store, projectKey)
	if err != nil {
		return err
	}
	for _, ticket := range tickets {
		if ticket.ID != ticketID {
			continue
		}
		if agentUse {
			return agentUpdate("ticket.ready", "ok", "ticket loaded", ticket)
		}
		if jsonOut {
			return writeJSON(ticket)
		}
		fmt.Printf("# %s\n\n", ticket.Title)
		fmt.Printf("ID:      %s\n", ticket.ID)
		fmt.Printf("Project: %s\n", ticket.ProjectKey)
		fmt.Printf("Status:  %s\n", ticket.Status)
		if ticket.GoalID != "" {
			fmt.Printf("Goal:    %s\n", ticket.GoalID)
		}
		if ticket.Created != "" {
			fmt.Printf("Created: %s\n", ticket.Created)
		}
		if ticket.Path != "" {
			fmt.Printf("Path:    %s\n", ticket.Path)
		}
		return nil
	}
	return fmt.Errorf("ticket %q not found in project %q", ticketID, projectKey)
}

func ticketCreateCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	goalID, rest, err := consumeFlagValue(rest, "--goal", false)
	if err != nil {
		return err
	}
	projectKey, title, err := resolveCreateProjectAndTitle(store, rest, "ticket")
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	ticket, event, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: projectKey,
		Title:      title,
		GoalID:     goalID,
		Actor:      defaultActor(),
	})
	if err != nil {
		return err
	}
	return finishWrite(store, ticket.ProjectKey, writeCommandResult{
		Action:     "ticket.created",
		ProjectKey: ticket.ProjectKey,
		GoalID:     ticket.GoalID,
		TicketID:   ticket.ID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketClaimCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	agent, rest, err := consumeFlagValue(rest, "--agent", false)
	if err != nil {
		return err
	}
	ttlText, rest, err := consumeFlagValue(rest, "--ttl", false)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("ticket id is required; usage: matt ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]")
	}
	if agent == "" {
		agent = defaultActor()
	}
	ttl := 2 * time.Hour
	if ttlText != "" {
		ttl, err = time.ParseDuration(ttlText)
		if err != nil {
			return fmt.Errorf("invalid --ttl %q: %w", ttlText, err)
		}
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	now := time.Now()
	writer := maat.NewWriteStore(store)
	event, err := writer.ClaimTicket(maat.ClaimTicketInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      agent,
		ExpiresAt:  now.Add(ttl),
		At:         now,
	})
	if err != nil {
		return err
	}
	expiresAt := now.Add(ttl).Format(time.RFC3339)
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.claimed",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
		Agent:      agent,
		ExpiresAt:  expiresAt,
	}, jsonOut)
}

func ticketCommentCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return errors.New("ticket id and comment are required; usage: matt ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	event, err := writer.CommentTicket(maat.TicketCommentInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      defaultActor(),
		Comment:    rest[1],
	})
	if err != nil {
		return err
	}
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.commented",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
	}, jsonOut)
}

func ticketCompleteCommand(args []string) error {
	filtered, jsonOut := splitJSONFlag(args)
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	evidence, rest, err := consumeFlagValue(rest, "--evidence", true)
	if err != nil {
		return err
	}
	projectKey, rest, err := consumeFlagValue(rest, "--project", false)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return errors.New("ticket id is required; usage: matt ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]")
	}
	ticketID := rest[0]
	projectKey, err = resolveTicketProject(store, projectKey, ticketID)
	if err != nil {
		return err
	}
	writer := maat.NewWriteStore(store)
	event, err := writer.CompleteTicket(maat.CompleteTicketInput{
		ProjectKey: projectKey,
		TicketID:   ticketID,
		Actor:      defaultActor(),
		Evidence:   []string{evidence},
	})
	if err != nil {
		return err
	}
	return finishWrite(store, projectKey, writeCommandResult{
		Action:     "ticket.completed",
		ProjectKey: projectKey,
		TicketID:   ticketID,
		EventID:    event.ID,
	}, jsonOut)
}

func agentCommand(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: matt agent <initialize|instructions>")
	}
	switch args[0] {
	case "initialize":
		return agentInitializeCommand(args[1:])
	case "instructions":
		return agentInstructionsCommand(args[1:])
	default:
		return fmt.Errorf("unknown agent command %q", args[0])
	}
}

func agentInstructionsCommand(args []string) error {
	jsonOut := false
	outputPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--output":
			if i+1 >= len(args) {
				return errors.New("--output requires a path")
			}
			outputPath = args[i+1]
			i++
		default:
			return fmt.Errorf("unexpected agent instructions argument %q", args[i])
		}
	}

	snippet := maat.AgentInstructionsSnippet()
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(snippet+"\n"), 0o644); err != nil {
			return err
		}
	}
	if agentUse {
		return agentUpdate("agent.instructions.ready", "ok", "agent instructions ready", map[string]string{"instructions": snippet})
	}
	if jsonOut {
		return writeJSON(map[string]string{"instructions": snippet})
	}
	fmt.Println(snippet)
	return nil
}

func agentInitializeCommand(args []string) error {
	jsonOut := false
	projectKey := ""
	store := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--project":
			if i+1 >= len(args) {
				return errors.New("--project requires a project key")
			}
			projectKey = args[i+1]
			i++
		case "--storage":
			if i+1 >= len(args) {
				return errors.New("--storage requires a path")
			}
			abs, err := filepath.Abs(args[i+1])
			if err != nil {
				return err
			}
			store = abs
			i++
		default:
			return fmt.Errorf("unexpected agent initialize argument %q", args[i])
		}
	}
	if store == "" {
		if cfg, err := readConfig(); err == nil && cfg.StoragePath != "" {
			store = cfg.StoragePath
		}
	}
	document := maat.AgentSetupDocument(maat.AgentSetupOptions{
		ProjectKey:  projectKey,
		StoragePath: store,
	})
	if agentUse {
		return agentUpdate("agent.initialize.ready", "ok", "agent setup document ready", map[string]string{"document": document})
	}
	if jsonOut {
		return writeJSON(map[string]string{"document": document})
	}
	fmt.Println(document)
	return nil
}

func searchCommand(args []string) error {
	jsonOut := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			jsonUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	store, rest, err := loadStoreAndRest(filtered)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return errors.New("usage: matt search <query> [--storage <path>] [--json]")
	}
	query := strings.Join(rest, " ")
	progress("search.index", "refreshing search index", map[string]string{"query": query})
	results, err := searchWithSQLite(store, query)
	if err != nil {
		return err
	}
	if results == nil {
		results = []maat.SearchResult{}
	}
	if agentUse {
		return agentUpdate("search.ready", "ok", "search complete", map[string]any{
			"query":   query,
			"count":   len(results),
			"results": results,
		})
	}
	if jsonOut {
		return writeJSON(results)
	}
	ok("search.ready", fmt.Sprintf("found %d results", len(results)), nil)
	for _, result := range results {
		fmt.Printf("%s:%d [%s] %s\n", result.Path, result.Line, result.Type, result.Title)
		if result.Excerpt != "" {
			fmt.Printf("  %s\n", result.Excerpt)
		}
	}
	return nil
}

func searchWithSQLite(store, query string) ([]maat.SearchResult, error) {
	info, err := maat.RebuildSQLiteIndex(store)
	if err == nil {
		results, err := maat.SearchSQLiteIndex(info.Path, query)
		if err == nil {
			return results, nil
		}
	}
	return maat.Search(store, query)
}

type projectListItem struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Updated string `json:"updated,omitempty"`
	Path    string `json:"path"`
	Layout  string `json:"layout"`
}

func loadProjectListItems(store string) ([]projectListItem, error) {
	legacyProjects, err := maat.LoadProjects(store)
	if err != nil {
		return nil, err
	}
	objectProjects, err := maat.LoadObjectProjects(store)
	if err != nil {
		return nil, err
	}
	projects := make([]projectListItem, 0, len(legacyProjects)+len(objectProjects))
	for _, project := range legacyProjects {
		projects = append(projects, projectListItem{
			ID:      project.ID,
			Key:     project.ID,
			Title:   project.Title,
			Status:  project.Status,
			Updated: project.Updated,
			Path:    project.Path,
			Layout:  "legacy",
		})
	}
	for _, project := range objectProjects {
		projects = append(projects, projectListItem{
			ID:      project.Key,
			Key:     project.Key,
			Title:   project.DisplayName,
			Status:  project.Status,
			Updated: project.Updated,
			Path:    project.Path,
			Layout:  "object",
		})
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].ID == projects[j].ID {
			return projects[i].Layout < projects[j].Layout
		}
		return projects[i].ID < projects[j].ID
	})
	return projects, nil
}

type writeCommandResult struct {
	Action         string `json:"action"`
	ProjectKey     string `json:"project_key"`
	GoalID         string `json:"goal_id,omitempty"`
	TicketID       string `json:"ticket_id,omitempty"`
	EventID        string `json:"event_id"`
	Agent          string `json:"agent,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	IndexRefreshed bool   `json:"index_refreshed"`
	IndexWarning   string `json:"index_warning,omitempty"`
}

func finishWrite(store, projectKey string, result writeCommandResult, jsonOut bool) error {
	progress("write.validate", "validating written project", map[string]string{"project": projectKey})
	if _, err := maat.LoadObjectProject(store, projectKey); err != nil {
		return fmt.Errorf("post-write validation failed for project %q: %w", projectKey, err)
	}
	progress("write.index", "refreshing indexes", nil)
	refreshResult := refreshIndexesBestEffort(store)
	result.IndexRefreshed = refreshResult.Refreshed
	result.IndexWarning = refreshResult.Warning
	warnIndexRefresh(refreshResult)
	if agentUse {
		return agentUpdate(result.Action, "ok", result.Action, result)
	}
	if jsonOut {
		return writeJSON(result)
	}
	printWriteResult(result)
	return nil
}

func printWriteResult(result writeCommandResult) {
	ok(result.Action, writeActionMessage(result), nil)
	switch result.Action {
	case "goal.created":
		printField("Goal", result.GoalID)
		printField("Project", result.ProjectKey)
	case "ticket.created":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
		if result.GoalID != "" {
			printField("Goal", result.GoalID)
		}
	case "ticket.claimed":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
		printField("Agent", result.Agent)
		printField("Expires", result.ExpiresAt)
	case "ticket.commented":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
	case "ticket.completed":
		printField("Ticket", result.TicketID)
		printField("Project", result.ProjectKey)
	default:
		printField("Action", result.Action)
		printField("Project", result.ProjectKey)
	}
	printField("Event", result.EventID)
	if result.IndexWarning != "" {
		printField("Index", result.IndexWarning)
	}
}

func writeActionMessage(result writeCommandResult) string {
	switch result.Action {
	case "goal.created":
		return "created goal"
	case "ticket.created":
		return "created ticket"
	case "ticket.claimed":
		return "claimed ticket"
	case "ticket.commented":
		return "commented on ticket"
	case "ticket.completed":
		return "completed ticket"
	default:
		return result.Action
	}
}

func refreshIndexes(store string) error {
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return fmt.Errorf("rebuild json index: %w", err)
	}
	if _, err := maat.WriteIndex(store, idx); err != nil {
		return fmt.Errorf("write json index: %w", err)
	}
	if _, err := maat.RebuildSQLiteIndex(store); err != nil {
		return fmt.Errorf("rebuild sqlite index: %w", err)
	}
	return nil
}

type indexRefreshResult struct {
	Refreshed bool
	Warning   string
}

func refreshIndexesBestEffort(store string) indexRefreshResult {
	if err := refreshIndexes(store); err != nil {
		return indexRefreshResult{
			Refreshed: false,
			Warning:   fmt.Sprintf("index refresh failed after state write persisted: %v", err),
		}
	}
	return indexRefreshResult{Refreshed: true}
}

func warnIndexRefresh(result indexRefreshResult) {
	if result.Warning == "" {
		return
	}
	warn("index.refresh", result.Warning, map[string]any{
		"index_refreshed": result.Refreshed,
		"warning":         result.Warning,
	})
}

func resolveTicketProject(store, projectKey, ticketID string) (string, error) {
	if strings.TrimSpace(projectKey) != "" {
		return projectKey, nil
	}
	if project, err := inferProjectFromWorkingDirectory(store); err == nil {
		if objectProjectHasTicket(project, ticketID) {
			return project.Key, nil
		}
	}
	objectStore, err := maat.LoadObjectStore(store)
	if err != nil {
		return "", err
	}
	matches := make([]string, 0, 1)
	for _, project := range objectStore.Projects {
		for _, ticket := range project.Tickets {
			if ticket.ID == ticketID {
				matches = append(matches, project.Key)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %q not found; pass --project <project-key> if it is in a legacy project", ticketID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ticket %q exists in multiple projects; pass --project <project-key>", ticketID)
	}
}

func resolveReadableTicketProject(store, projectKey, ticketID string) (string, error) {
	if strings.TrimSpace(projectKey) != "" {
		return projectKey, nil
	}
	if project, err := inferProjectFromWorkingDirectory(store); err == nil {
		if objectProjectHasTicket(project, ticketID) {
			return project.Key, nil
		}
		for _, ticket := range legacyTicketViewsForProject(project.Key, nil) {
			if ticket.ID == ticketID {
				return project.Key, nil
			}
		}
	}
	tickets, err := loadTicketViews(store, "")
	if err != nil {
		return "", err
	}
	matches := make([]string, 0, 1)
	for _, ticket := range tickets {
		if ticket.ID == ticketID {
			matches = append(matches, ticket.ProjectKey)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("ticket %q not found", ticketID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ticket %q exists in multiple projects; pass --project <project-key>", ticketID)
	}
}

func objectProjectHasTicket(project maat.ObjectProject, ticketID string) bool {
	for _, ticket := range project.Tickets {
		if ticket.ID == ticketID {
			return true
		}
	}
	return false
}

func loadTicketViews(store, projectKey string) ([]ticketView, error) {
	var tickets []ticketView
	if projectKey != "" {
		if project, err := maat.LoadObjectProject(store, projectKey); err == nil {
			tickets = append(tickets, objectTicketViews(project)...)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if project, err := maat.LoadProject(store, projectKey); err == nil {
			tickets = append(tickets, legacyTicketViewsForProject(project.ID, project.Goals)...)
		} else if !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
		return tickets, nil
	}
	objectStore, err := maat.LoadObjectStore(store)
	if err != nil {
		return nil, err
	}
	for _, project := range objectStore.Projects {
		tickets = append(tickets, objectTicketViews(project)...)
	}
	legacyProjects, err := maat.LoadProjects(store)
	if err != nil {
		return nil, err
	}
	for _, project := range legacyProjects {
		tickets = append(tickets, legacyTicketViewsForProject(project.ID, project.Goals)...)
	}
	return tickets, nil
}

func objectTicketViews(project maat.ObjectProject) []ticketView {
	tickets := make([]ticketView, 0, len(project.Tickets))
	for _, ticket := range project.Tickets {
		tickets = append(tickets, ticketView{
			ID:         ticket.ID,
			ProjectKey: ticket.ProjectKey,
			GoalID:     ticket.GoalID,
			Title:      ticket.Title,
			Status:     ticket.Status,
			Created:    ticket.Created,
			Path:       ticket.Path,
		})
	}
	return tickets
}

func legacyTicketViewsForProject(projectKey string, goals []maat.Goal) []ticketView {
	var tickets []ticketView
	for _, goal := range goals {
		for _, ticket := range goal.Tickets {
			status := "active"
			if ticket.Done {
				status = "done"
			}
			tickets = append(tickets, ticketView{
				ID:         ticket.ID,
				ProjectKey: projectKey,
				GoalID:     goal.ID,
				Title:      ticket.Title,
				Status:     status,
				Legacy:     true,
			})
		}
	}
	return tickets
}

func resolveCreateProjectAndTitle(store string, args []string, kind string) (string, string, error) {
	switch len(args) {
	case 1:
		project, err := inferProjectFromWorkingDirectory(store)
		if err != nil {
			return "", "", fmt.Errorf("project key is required when not inside a linked project; usage: matt %s create <project-key> <title>: %w", kind, err)
		}
		return project.Key, args[0], nil
	case 2:
		return args[0], args[1], nil
	default:
		return "", "", fmt.Errorf("project key and %s title are required; usage: matt %s create [project-key] <title> [--storage <path>] [--json]", kind, kind)
	}
}

func inferProjectFromWorkingDirectory(store string) (maat.ObjectProject, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return maat.ObjectProject{}, err
	}
	return maat.InferProjectForPath(context.Background(), store, cwd)
}

func consumeFlagValue(args []string, flag string, required bool) (string, []string, error) {
	filtered := make([]string, 0, len(args))
	value := ""
	found := false
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			filtered = append(filtered, args[i])
			continue
		}
		if found {
			return "", nil, fmt.Errorf("%s can only be provided once", flag)
		}
		if i+1 >= len(args) {
			return "", nil, fmt.Errorf("%s requires a value", flag)
		}
		value = args[i+1]
		found = true
		i++
	}
	if required && !found {
		return "", nil, fmt.Errorf("%s is required", flag)
	}
	return value, filtered, nil
}

func defaultActor() string {
	for _, key := range []string{"MAAT_ACTOR", "USER", "USERNAME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return "agent"
}

func splitJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
			jsonUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonOut
}

func splitDestinationFlag(args []string) ([]string, string, error) {
	filtered := make([]string, 0, len(args))
	dest := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--dest" {
			if i+1 >= len(args) {
				return nil, "", errors.New("--dest requires a path")
			}
			dest = args[i+1]
			i++
			continue
		}
		filtered = append(filtered, args[i])
	}
	if strings.TrimSpace(dest) == "" {
		return nil, "", errors.New("usage: matt migrate apply --dest <path> [--storage <path>]")
	}
	return filtered, dest, nil
}

func writeJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func splitGlobalFlags(args []string) ([]string, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--agent-use" {
			agentUse = true
			continue
		}
		filtered = append(filtered, arg)
	}
	if agentUse {
		for _, arg := range filtered {
			if arg == "--json" {
				return nil, errors.New("--agent-use cannot be combined with --json")
			}
		}
	}
	return filtered, nil
}

func progress(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "running", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[run]", "36"), message)
}

func ok(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "ok", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[ok]", "32"), message)
}

func warn(step, message string, data any) {
	if agentUse {
		_ = agentUpdate(step, "warning", message, data)
		return
	}
	if jsonUse {
		return
	}
	fmt.Printf("%s %s\n", color("[warn]", "33"), message)
}

func printField(label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Printf("%-10s %s\n", label+":", value)
}

func agentUpdate(step, status, message string, data any) error {
	event := map[string]any{
		"type":    "maat.update",
		"step":    step,
		"status":  status,
		"message": message,
	}
	if data != nil {
		event["data"] = data
	}
	return json.NewEncoder(os.Stdout).Encode(event)
}

func colorNumber(value int) string {
	return color(fmt.Sprintf("%d", value), "36")
}

func colorStatus(status string) string {
	switch strings.ToLower(status) {
	case "done":
		return color(status, "32")
	case "active":
		return color(status, "36")
	case "waiting", "paused":
		return color(status, "33")
	default:
		return status
	}
}

func color(text, code string) string {
	if !colorEnabled() {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func colorEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MATT_COLOR"))) {
	case "always", "1", "true", "yes":
		return true
	case "never", "0", "false", "no":
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func loadStore(args []string) (string, error) {
	store, rest, err := loadStoreAndRest(args)
	if err != nil {
		return "", err
	}
	if len(rest) > 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}
	return store, nil
}

func loadStoreAndRest(args []string) (string, []string, error) {
	rest := make([]string, 0, len(args))
	store := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--storage" {
			if i+1 >= len(args) {
				return "", nil, errors.New("--storage requires a path")
			}
			store = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if store == "" {
		cfg, err := readConfig()
		if err == nil && cfg.StoragePath != "" {
			store = cfg.StoragePath
		}
	}
	if store == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil, err
		}
		store = cwd
	}
	abs, err := filepath.Abs(store)
	if err != nil {
		return "", nil, err
	}
	return abs, rest, nil
}

type config struct {
	StoragePath string `json:"storage_path"`
}

func writeConfig(storagePath string) error {
	abs, err := filepath.Abs(storagePath)
	if err != nil {
		return err
	}
	if stat, err := os.Stat(abs); err != nil {
		return err
	} else if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", abs)
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config{StoragePath: abs}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if agentUse {
		return agentUpdate("storage.linked", "ok", "storage linked", map[string]string{
			"storage_path": abs,
			"config_path":  path,
		})
	}
	fmt.Printf("linked storage: %s\n", abs)
	return nil
}

func readConfig() (config, error) {
	path, err := configPath()
	if err != nil {
		return config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func configPath() (string, error) {
	if override := os.Getenv("MAAT_CONFIG"); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "maat", "config.json"), nil
}
