package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunday-studio/maat/internal/maat"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "matt: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
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
		projects, err := maat.LoadProjects(store)
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(projects)
		}
		for _, project := range projects {
			fmt.Printf("%-12s %-9s %s\n", project.ID, project.Status, project.Title)
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
		if jsonOut {
			return writeJSON(summary)
		}
		fmt.Printf("Projects: %d\n", summary.Projects)
		fmt.Printf("Goals:    %d active, %d done, %d total\n", summary.ActiveGoals, summary.DoneGoals, summary.Goals)
		fmt.Printf("Tickets:  %d open, %d done, %d total\n", summary.OpenTickets, summary.DoneTickets, summary.Tickets)
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
	fmt.Print(`matt is the Maat CLI.

Usage:
  matt init [storage-path]
  matt storage link <storage-path>
  matt index rebuild [--storage <path>]
  matt projects [--storage <path>] [--json]
  matt project show <project-id> [--storage <path>]
  matt goal create <project-key> <title> [--storage <path>] [--json]
  matt ticket create <project-key> <title> [--goal <goal-id>] [--storage <path>] [--json]
  matt ticket claim <ticket-id> [--agent <agent>] [--ttl <duration>] [--project <project-key>] [--storage <path>] [--json]
  matt ticket comment <ticket-id> <comment> [--project <project-key>] [--storage <path>] [--json]
  matt ticket complete <ticket-id> --evidence <text> [--project <project-key>] [--storage <path>] [--json]
  matt agent instructions [--json] [--output <path>]
  matt status [--storage <path>] [--json]
  matt validate [--storage <path>] [--json]
  matt sync [--storage <path>] [--message <msg>] [--push] [--status] [--json]
  matt migrate plan [--storage <path>] [--json]
  matt migrate apply --dest <path> [--storage <path>]
  matt search <query> [--storage <path>] [--json]
  matt tui [--storage <path>]

Git plus Markdown remains the source of truth. The local index is rebuildable.
`)
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
	idx, err := maat.BuildIndex(store)
	if err != nil {
		return err
	}
	path, err := maat.WriteIndex(store, idx)
	if err != nil {
		return err
	}
	sqliteInfo, err := maat.RebuildSQLiteIndex(store)
	if err != nil {
		return err
	}
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
	report, err := maat.ValidateStore(store)
	if err != nil {
		return err
	}
	if jsonOut {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else if report.OK() {
		fmt.Printf("validated %d files: ok\n", report.Files)
	} else {
		fmt.Printf("validated %d files: %d issues\n", report.Files, len(report.Issues))
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
		plan, err := maat.PlanLegacyMigration(store, maat.MigrationOptions{})
		if err != nil {
			return err
		}
		if jsonOut {
			return writeJSON(plan)
		}
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
		plan, err := maat.ApplyLegacyMigration(store, absDest, maat.MigrationOptions{})
		if err != nil {
			return err
		}
		fmt.Printf("migrated %d projects into %s\n", len(plan.Projects), absDest)
		fmt.Printf("wrote %d files\n", len(plan.Files))
		return nil
	default:
		return fmt.Errorf("unknown migrate command %q", args[0])
	}
}

func projectCommand(args []string) error {
	if len(args) < 2 || args[0] != "show" {
		return errors.New("usage: matt project show <project-id> [--storage <path>]")
	}
	projectID := args[1]
	store, err := loadStore(args[2:])
	if err != nil {
		return err
	}
	project, err := maat.LoadProject(store, projectID)
	if err != nil {
		return err
	}
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
	if len(rest) != 2 {
		return errors.New("project key and goal title are required; usage: matt goal create <project-key> <title> [--storage <path>] [--json]")
	}
	writer := maat.NewWriteStore(store)
	goal, event, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: rest[0],
		Title:      rest[1],
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
		return errors.New("usage: matt ticket <create|claim|comment|complete>")
	}
	switch args[0] {
	case "create":
		return ticketCreateCommand(args[1:])
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
	if len(rest) != 2 {
		return errors.New("project key and ticket title are required; usage: matt ticket create <project-key> <title> [--goal <goal-id>] [--storage <path>] [--json]")
	}
	writer := maat.NewWriteStore(store)
	ticket, event, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey: rest[0],
		Title:      rest[1],
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
	if len(args) == 0 || args[0] != "instructions" {
		return errors.New("usage: matt agent instructions [--json] [--output <path>]")
	}
	jsonOut := false
	outputPath := ""
	for i := 1; i < len(args); i++ {
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
	if jsonOut {
		return writeJSON(map[string]string{"instructions": snippet})
	}
	fmt.Println(snippet)
	return nil
}

func searchCommand(args []string) error {
	jsonOut := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
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
	results, err := searchWithSQLite(store, query)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(results)
	}
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

type writeCommandResult struct {
	Action     string `json:"action"`
	ProjectKey string `json:"project_key"`
	GoalID     string `json:"goal_id,omitempty"`
	TicketID   string `json:"ticket_id,omitempty"`
	EventID    string `json:"event_id"`
	Agent      string `json:"agent,omitempty"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

func finishWrite(store, projectKey string, result writeCommandResult, jsonOut bool) error {
	if _, err := maat.LoadObjectProject(store, projectKey); err != nil {
		return fmt.Errorf("post-write validation failed for project %q: %w", projectKey, err)
	}
	if err := refreshIndexes(store); err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(result)
	}
	printWriteResult(result)
	return nil
}

func printWriteResult(result writeCommandResult) {
	switch result.Action {
	case "goal.created":
		fmt.Printf("created goal %s\n", result.GoalID)
		fmt.Printf("project %s\n", result.ProjectKey)
	case "ticket.created":
		fmt.Printf("created ticket %s\n", result.TicketID)
		fmt.Printf("project %s\n", result.ProjectKey)
		if result.GoalID != "" {
			fmt.Printf("goal %s\n", result.GoalID)
		}
	case "ticket.claimed":
		fmt.Printf("claimed ticket %s\n", result.TicketID)
		fmt.Printf("project %s\n", result.ProjectKey)
		fmt.Printf("agent %s\n", result.Agent)
		fmt.Printf("expires %s\n", result.ExpiresAt)
	case "ticket.commented":
		fmt.Printf("commented on ticket %s\n", result.TicketID)
		fmt.Printf("project %s\n", result.ProjectKey)
	case "ticket.completed":
		fmt.Printf("completed ticket %s\n", result.TicketID)
		fmt.Printf("project %s\n", result.ProjectKey)
	default:
		fmt.Printf("%s\n", result.Action)
		fmt.Printf("project %s\n", result.ProjectKey)
	}
	fmt.Printf("event %s\n", result.EventID)
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

func resolveTicketProject(store, projectKey, ticketID string) (string, error) {
	if strings.TrimSpace(projectKey) != "" {
		return projectKey, nil
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
