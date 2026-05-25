package maat

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const migrationEventType = "project.migrated"

type MigrationOptions struct {
	At    time.Time
	Actor string
}

type MigrationPlan struct {
	Source   string                 `json:"source"`
	Projects []MigrationProjectPlan `json:"projects"`
	Files    []MigrationPlannedFile `json:"files"`
}

type MigrationProjectPlan struct {
	LegacyPath  string   `json:"legacy_path"`
	ProjectKey  string   `json:"project_key"`
	ProjectPath string   `json:"project_path"`
	GoalPaths   []string `json:"goal_paths,omitempty"`
	TicketPaths []string `json:"ticket_paths,omitempty"`
	EventPaths  []string `json:"event_paths,omitempty"`
}

type MigrationPlannedFile struct {
	Path    string `json:"path"`
	Content string `json:"-"`
}

func PlanLegacyMigration(source string, options MigrationOptions) (MigrationPlan, error) {
	source = filepath.Clean(source)
	options = normalizeMigrationOptions(options)

	projects, err := LoadProjects(source)
	if err != nil {
		return MigrationPlan{}, err
	}

	plan := MigrationPlan{Source: source}
	for _, project := range projects {
		projectPlan, files, err := planLegacyProjectMigration(project, options)
		if err != nil {
			return MigrationPlan{}, err
		}
		plan.Projects = append(plan.Projects, projectPlan)
		plan.Files = append(plan.Files, files...)
	}
	sort.Slice(plan.Files, func(i, j int) bool {
		return plan.Files[i].Path < plan.Files[j].Path
	})
	return plan, nil
}

func ApplyLegacyMigration(source, destination string, options MigrationOptions) (MigrationPlan, error) {
	if strings.TrimSpace(destination) == "" {
		return MigrationPlan{}, fmt.Errorf("migration destination is required")
	}
	plan, err := PlanLegacyMigration(source, options)
	if err != nil {
		return MigrationPlan{}, err
	}
	for _, file := range plan.Files {
		path := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return MigrationPlan{}, err
		}
		handle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if os.IsExist(err) {
				return MigrationPlan{}, fmt.Errorf("migration destination already has %s", file.Path)
			}
			return MigrationPlan{}, err
		}
		if _, err := handle.WriteString(file.Content); err != nil {
			_ = handle.Close()
			return MigrationPlan{}, err
		}
		if err := handle.Close(); err != nil {
			return MigrationPlan{}, err
		}
	}
	return plan, nil
}

func planLegacyProjectMigration(project Project, options MigrationOptions) (MigrationProjectPlan, []MigrationPlannedFile, error) {
	projectKey := NormalizeIDPart(project.ID)
	if projectKey == "" {
		return MigrationProjectPlan{}, nil, fmt.Errorf("%s: project id cannot become a project key", project.Path)
	}

	projectPath := filepath.ToSlash(filepath.Join("projects", projectKey, "project.md"))
	projectPlan := MigrationProjectPlan{
		LegacyPath:  project.Path,
		ProjectKey:  projectKey,
		ProjectPath: projectPath,
	}

	var files []MigrationPlannedFile
	files = append(files, MigrationPlannedFile{
		Path:    projectPath,
		Content: renderMigratedProject(project, projectKey, options.At),
	})

	ticketIDs := plannedTicketIDs(project, projectKey)
	for _, goal := range project.Goals {
		goalID := migratedGoalID(goal, projectKey)
		goalPath := filepath.ToSlash(filepath.Join("projects", projectKey, "goals", goalID+".md"))
		projectPlan.GoalPaths = append(projectPlan.GoalPaths, goalPath)
		files = append(files, MigrationPlannedFile{
			Path:    goalPath,
			Content: renderMigratedGoal(goal, projectKey, goalID, options.At),
		})

		for index, ticket := range goal.Tickets {
			ticketID := ticketIDs[ticketKey(goal.ID, index, ticket)]
			ticketPath := filepath.ToSlash(filepath.Join("projects", projectKey, "tickets", ticketID+".md"))
			projectPlan.TicketPaths = append(projectPlan.TicketPaths, ticketPath)
			files = append(files, MigrationPlannedFile{
				Path:    ticketPath,
				Content: renderMigratedTicket(ticket, projectKey, goalID, ticketID, options.At),
			})
		}
	}

	eventID := migrationEventID(options.At, options.Actor, projectKey)
	eventPath, err := EventRelativePath(projectKey, options.At, eventID)
	if err != nil {
		return MigrationProjectPlan{}, nil, err
	}
	eventMarkdown, err := RenderEventMarkdown(Event{
		ID:      eventID,
		Time:    options.At,
		Actor:   options.Actor,
		Project: projectKey,
		Type:    migrationEventType,
		Object:  projectKey,
		Summary: fmt.Sprintf("Migrated legacy project file %s into target object storage.", project.Path),
		Evidence: []string{
			fmt.Sprintf("Planned %d goals and %d tickets.", len(project.Goals), countLegacyTickets(project)),
		},
	})
	if err != nil {
		return MigrationProjectPlan{}, nil, err
	}
	projectPlan.EventPaths = append(projectPlan.EventPaths, eventPath)
	files = append(files, MigrationPlannedFile{Path: eventPath, Content: eventMarkdown})

	sort.Strings(projectPlan.GoalPaths)
	sort.Strings(projectPlan.TicketPaths)
	sort.Strings(projectPlan.EventPaths)
	return projectPlan, files, nil
}

func normalizeMigrationOptions(options MigrationOptions) MigrationOptions {
	if options.At.IsZero() {
		options.At = time.Now()
	}
	options.Actor = strings.TrimSpace(options.Actor)
	if options.Actor == "" {
		options.Actor = "matt"
	}
	return options
}

func renderMigratedProject(project Project, projectKey string, at time.Time) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Project: %s\n\n", markdownTitle(project.Title, project.ID))
	writeFieldTable(&buffer, []markdownField{
		{"Project Key", projectKey},
		{"Display Name", markdownTitle(project.Title, project.ID)},
		{"Status", project.Status},
		{"Created", at.Format(time.RFC3339)},
		{"Updated", at.Format(time.RFC3339)},
		{"Tags", strings.Join(project.Tags, " ")},
	})
	buffer.WriteString("\n## Summary\n\n")
	if strings.TrimSpace(project.Current) != "" {
		buffer.WriteString(strings.TrimSpace(project.Current))
	} else {
		fmt.Fprintf(&buffer, "Migrated from legacy project file %s.", project.Path)
	}
	buffer.WriteString("\n")
	return buffer.String()
}

func renderMigratedGoal(goal Goal, projectKey, goalID string, at time.Time) string {
	status := strings.TrimSpace(goal.Status)
	if status == "" {
		status = "proposed"
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Goal: %s\n\n", markdownTitle(goal.Title, goalID))
	writeFieldTable(&buffer, []markdownField{
		{"Goal ID", goalID},
		{"Project", projectKey},
		{"Status", status},
		{"Created", at.Format(time.RFC3339)},
		{"Tags", strings.Join(goal.Tags, " ")},
	})
	buffer.WriteString("\n## Outcome\n\n")
	fmt.Fprintf(&buffer, "Migrated from legacy goal %s.", markdownTitle(goal.ID, goalID))
	buffer.WriteString("\n")
	return buffer.String()
}

func renderMigratedTicket(ticket Ticket, projectKey, goalID, ticketID string, at time.Time) string {
	status := "active"
	if ticket.Done {
		status = "done"
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Ticket: %s\n\n", markdownTitle(ticket.Title, ticketID))
	writeFieldTable(&buffer, []markdownField{
		{"Ticket ID", ticketID},
		{"Project", projectKey},
		{"Goal", goalID},
		{"Status", status},
		{"Created", at.Format(time.RFC3339)},
	})
	buffer.WriteString("\n## Description\n\n")
	fmt.Fprintf(&buffer, "Migrated from legacy ticket %s.", markdownTitle(ticket.ID, ticketID))
	buffer.WriteString("\n")
	return buffer.String()
}

type markdownField struct {
	Key   string
	Value string
}

func writeFieldTable(buffer *bytes.Buffer, fields []markdownField) {
	buffer.WriteString("| Field | Value |\n")
	buffer.WriteString("|---|---|\n")
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		writeMarkdownField(buffer, field.Key, field.Value)
	}
}

func migratedGoalID(goal Goal, projectKey string) string {
	if clean := cleanObjectID(goal.ID); clean != "" {
		return clean
	}
	return "G-" + shortHash(projectKey+"|"+goal.Title)
}

func plannedTicketIDs(project Project, projectKey string) map[string]string {
	counts := map[string]int{}
	for _, goal := range project.Goals {
		for index, ticket := range goal.Tickets {
			_ = index
			base := cleanObjectID(ticket.ID)
			if base == "" {
				base = "T-" + shortHash(projectKey+"|"+goal.ID+"|"+ticket.Title)
			}
			counts[base]++
		}
	}

	seen := map[string]int{}
	planned := map[string]string{}
	for _, goal := range project.Goals {
		goalID := migratedGoalID(goal, projectKey)
		for index, ticket := range goal.Tickets {
			base := cleanObjectID(ticket.ID)
			if base == "" {
				base = "T-" + shortHash(projectKey+"|"+goal.ID+"|"+ticket.Title)
			}
			id := base
			if counts[base] > 1 {
				id = "T-" + NormalizeIDPart(goalID) + "-" + NormalizeIDPart(base)
			}
			seen[id]++
			if seen[id] > 1 {
				id = fmt.Sprintf("%s-%d", id, seen[id])
			}
			planned[ticketKey(goal.ID, index, ticket)] = id
		}
	}
	return planned
}

func cleanObjectID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, `/\`) {
		return ""
	}
	return value
}

func ticketKey(goalID string, index int, ticket Ticket) string {
	return fmt.Sprintf("%s\x00%d\x00%s\x00%s", goalID, index, ticket.ID, ticket.Title)
}

func migrationEventID(at time.Time, actor, projectKey string) string {
	actor = NormalizeIDPart(actor)
	if actor == "" {
		actor = "matt"
	}
	return fmt.Sprintf("E-%s-%s-%s-migrated", at.Format("20060102-150405"), actor, projectKey)
}

func markdownTitle(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func countLegacyTickets(project Project) int {
	count := 0
	for _, goal := range project.Goals {
		count += len(goal.Tickets)
	}
	return count
}
