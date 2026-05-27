package maat

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WriteStore struct {
	Root    string
	Now     func() time.Time
	Entropy io.Reader
}

type CreateProjectInput struct {
	Key         string
	DisplayName string
	Status      string
	Tags        []string
	Summary     string
	PrimaryRepo string
	Remote      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateGoalInput struct {
	ProjectKey string
	Title      string
	Status     string
	Tags       []string
	Outcome    string
	Actor      string
	Commit     string
	Summary    string
	At         time.Time
}

type CreateTicketInput struct {
	ProjectKey  string
	GoalID      string
	Title       string
	Status      string
	Tags        []string
	Description string
	Acceptance  []string
	Actor       string
	Commit      string
	Summary     string
	At          time.Time
}

type TicketCommentInput struct {
	ProjectKey string
	TicketID   string
	Actor      string
	Comment    string
	Commit     string
	At         time.Time
}

type CompleteTicketInput struct {
	ProjectKey string
	TicketID   string
	Actor      string
	Summary    string
	Commit     string
	Evidence   []string
	At         time.Time
}

type ClaimTicketInput struct {
	ProjectKey string
	TicketID   string
	Actor      string
	Summary    string
	Commit     string
	ExpiresAt  time.Time
	At         time.Time
}

func NewWriteStore(root string) WriteStore {
	return WriteStore{Root: root}
}

func (store WriteStore) CreateProject(input CreateProjectInput) (ObjectProject, error) {
	now := store.timeOrNow(input.CreatedAt)
	updated := input.UpdatedAt
	if updated.IsZero() {
		updated = now
	}
	project := ObjectProject{
		Key:         store.projectKey(input.Key, input.DisplayName),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Status:      defaultStatus(input.Status, "active"),
		Created:     now.Format(time.RFC3339),
		Updated:     updated.Format(time.RFC3339),
		Tags:        cleanStrings(input.Tags),
		Summary:     strings.TrimSpace(input.Summary),
		Identity:    cleanIdentity(input.PrimaryRepo, input.Remote),
	}
	if project.Key == "" {
		return ObjectProject{}, fmt.Errorf("project key or display name is required")
	}
	if project.DisplayName == "" {
		project.DisplayName = project.Key
	}
	if err := requireValidStatus("project", project.Status); err != nil {
		return ObjectProject{}, err
	}

	rel := filepath.ToSlash(filepath.Join("projects", project.Key, "project.md"))
	path := store.abs(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ObjectProject{}, err
	}
	if err := os.MkdirAll(filepath.Join(filepath.Dir(path), "goals"), 0o755); err != nil {
		return ObjectProject{}, err
	}
	if err := os.MkdirAll(filepath.Join(filepath.Dir(path), "tickets"), 0o755); err != nil {
		return ObjectProject{}, err
	}
	if err := os.MkdirAll(filepath.Join(filepath.Dir(path), "events"), 0o755); err != nil {
		return ObjectProject{}, err
	}
	if err := writeNewFile(path, renderProjectMarkdown(project)); err != nil {
		return ObjectProject{}, err
	}
	return ParseObjectProjectFile(store.Root, path)
}

func (store WriteStore) CreateGoal(input CreateGoalInput) (ObjectGoal, ObjectEvent, error) {
	projectKey := NormalizeIDPart(input.ProjectKey)
	if err := store.requireProject(projectKey); err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	at := store.timeOrNow(input.At)
	goalID, err := store.newID(GoalIDPrefix, at)
	if err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	goal := ObjectGoal{
		ID:         goalID,
		ProjectKey: projectKey,
		Title:      strings.TrimSpace(input.Title),
		Status:     defaultStatus(input.Status, "active"),
		Created:    at.Format(time.RFC3339),
		Tags:       cleanStrings(input.Tags),
		Outcome:    strings.TrimSpace(input.Outcome),
	}
	if goal.Title == "" {
		return ObjectGoal{}, ObjectEvent{}, fmt.Errorf("goal title is required")
	}
	if goal.Outcome == "" {
		return ObjectGoal{}, ObjectEvent{}, fmt.Errorf("goal outcome is required")
	}
	if err := requireActor(input.Actor); err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	if err := requireValidStatus("goal", goal.Status); err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}

	path := store.abs(filepath.ToSlash(filepath.Join("projects", projectKey, "goals", goal.ID+".md")))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	if err := writeNewFile(path, renderGoalMarkdown(goal)); err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	parsedGoal, err := ParseObjectGoalFile(store.Root, path)
	if err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	event, err := store.writeEvent(projectKey, Event{
		Time:    at,
		Actor:   input.Actor,
		Project: projectKey,
		Type:    "goal.created",
		Object:  goal.ID,
		Commit:  input.Commit,
		Summary: defaultText(input.Summary, fmt.Sprintf("Created goal %s.", goal.ID)),
	})
	if err != nil {
		return ObjectGoal{}, ObjectEvent{}, err
	}
	return parsedGoal, event, nil
}

func (store WriteStore) CreateTicket(input CreateTicketInput) (ObjectTicket, ObjectEvent, error) {
	projectKey := NormalizeIDPart(input.ProjectKey)
	if err := store.requireProject(projectKey); err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	at := store.timeOrNow(input.At)
	ticketID, err := store.newID(TicketIDPrefix, at)
	if err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	ticket := ObjectTicket{
		ID:          ticketID,
		ProjectKey:  projectKey,
		GoalID:      strings.TrimSpace(input.GoalID),
		Title:       strings.TrimSpace(input.Title),
		Status:      defaultStatus(input.Status, "active"),
		Created:     at.Format(time.RFC3339),
		Tags:        cleanStrings(input.Tags),
		Description: strings.TrimSpace(input.Description),
		Acceptance:  cleanStrings(input.Acceptance),
	}
	if ticket.Title == "" {
		return ObjectTicket{}, ObjectEvent{}, fmt.Errorf("ticket title is required")
	}
	if ticket.Description == "" {
		return ObjectTicket{}, ObjectEvent{}, fmt.Errorf("ticket description is required")
	}
	if len(ticket.Acceptance) == 0 {
		return ObjectTicket{}, ObjectEvent{}, fmt.Errorf("ticket acceptance is required")
	}
	if err := requireActor(input.Actor); err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	if err := requireValidStatus("ticket", ticket.Status); err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}

	path := store.abs(filepath.ToSlash(filepath.Join("projects", projectKey, "tickets", ticket.ID+".md")))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	if err := writeNewFile(path, renderTicketMarkdown(ticket)); err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	parsedTicket, err := ParseObjectTicketFile(store.Root, path)
	if err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	event, err := store.writeEvent(projectKey, Event{
		Time:    at,
		Actor:   input.Actor,
		Project: projectKey,
		Type:    "ticket.created",
		Object:  ticket.ID,
		Commit:  input.Commit,
		Summary: defaultText(input.Summary, fmt.Sprintf("Created ticket %s.", ticket.ID)),
	})
	if err != nil {
		return ObjectTicket{}, ObjectEvent{}, err
	}
	return parsedTicket, event, nil
}

func (store WriteStore) CommentTicket(input TicketCommentInput) (ObjectEvent, error) {
	projectKey, ticketID, err := store.requireTicket(input.ProjectKey, input.TicketID)
	if err != nil {
		return ObjectEvent{}, err
	}
	if err := requireActor(input.Actor); err != nil {
		return ObjectEvent{}, err
	}
	return store.writeEvent(projectKey, Event{
		Time:    store.timeOrNow(input.At),
		Actor:   input.Actor,
		Project: projectKey,
		Type:    "ticket.commented",
		Object:  ticketID,
		Commit:  input.Commit,
		Summary: input.Comment,
	})
}

func (store WriteStore) CompleteTicket(input CompleteTicketInput) (ObjectEvent, error) {
	projectKey, ticketID, err := store.requireTicket(input.ProjectKey, input.TicketID)
	if err != nil {
		return ObjectEvent{}, err
	}
	if len(cleanStrings(input.Evidence)) == 0 {
		return ObjectEvent{}, fmt.Errorf("completion evidence is required")
	}
	if err := requireActor(input.Actor); err != nil {
		return ObjectEvent{}, err
	}
	if err := store.setTicketStatus(projectKey, ticketID, "done"); err != nil {
		return ObjectEvent{}, err
	}
	return store.writeEvent(projectKey, Event{
		Time:     store.timeOrNow(input.At),
		Actor:    input.Actor,
		Project:  projectKey,
		Type:     "ticket.completed",
		Object:   ticketID,
		Commit:   input.Commit,
		Summary:  defaultText(input.Summary, fmt.Sprintf("Completed ticket %s.", ticketID)),
		Evidence: input.Evidence,
	})
}

func (store WriteStore) setTicketStatus(projectKey, ticketID, status string) error {
	path := store.abs(filepath.ToSlash(filepath.Join("projects", projectKey, "tickets", ticketID+".md")))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		key, _, ok := parseTableRow(strings.TrimSpace(line))
		if !ok || !strings.EqualFold(key, "Status") {
			continue
		}
		lines[i] = fmt.Sprintf("| Status | %s |", status)
		content := strings.Join(lines, "\n")
		return os.WriteFile(path, []byte(content), 0o644)
	}
	return fmt.Errorf("ticket %s is missing Status field", ticketID)
}

func (store WriteStore) ClaimTicket(input ClaimTicketInput) (ObjectEvent, error) {
	projectKey, ticketID, err := store.requireTicket(input.ProjectKey, input.TicketID)
	if err != nil {
		return ObjectEvent{}, err
	}
	if input.ExpiresAt.IsZero() {
		return ObjectEvent{}, fmt.Errorf("claim expiration is required")
	}
	if err := requireActor(input.Actor); err != nil {
		return ObjectEvent{}, err
	}
	return store.writeEvent(projectKey, Event{
		Time:    store.timeOrNow(input.At),
		Actor:   input.Actor,
		Project: projectKey,
		Type:    "ticket.claimed",
		Object:  ticketID,
		Commit:  input.Commit,
		Metadata: map[string]string{
			"Expires": input.ExpiresAt.Format(time.RFC3339),
		},
		Summary: defaultText(input.Summary, fmt.Sprintf("Claimed ticket %s until %s.", ticketID, input.ExpiresAt.Format(time.RFC3339))),
	})
}

func (store WriteStore) writeEvent(projectKey string, event Event) (ObjectEvent, error) {
	event.Project = NormalizeIDPart(event.Project)
	if event.Project == "" {
		event.Project = projectKey
	}
	if event.Time.IsZero() {
		event.Time = store.timeOrNow(time.Time{})
	}
	eventID, err := store.newActorEventID(event.Time, event.Actor)
	if err != nil {
		return ObjectEvent{}, err
	}
	event.ID = eventID
	markdown, err := RenderEventMarkdown(event)
	if err != nil {
		return ObjectEvent{}, err
	}
	rel, err := EventRelativePath(projectKey, event.Time, event.ID)
	if err != nil {
		return ObjectEvent{}, err
	}
	path := store.abs(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ObjectEvent{}, err
	}
	if err := writeNewFile(path, markdown); err != nil {
		return ObjectEvent{}, err
	}
	return ParseObjectEventFile(store.Root, path)
}

func (store WriteStore) requireProject(projectKey string) error {
	if projectKey == "" {
		return fmt.Errorf("project key is required")
	}
	path := store.abs(filepath.ToSlash(filepath.Join("projects", projectKey, "project.md")))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project %q does not exist", projectKey)
		}
		return err
	}
	return nil
}

func (store WriteStore) requireTicket(projectKey, ticketID string) (string, string, error) {
	projectKey = NormalizeIDPart(projectKey)
	if err := store.requireProject(projectKey); err != nil {
		return "", "", err
	}
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return "", "", fmt.Errorf("ticket id is required")
	}
	path := store.abs(filepath.ToSlash(filepath.Join("projects", projectKey, "tickets", ticketID+".md")))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("ticket %q does not exist", ticketID)
		}
		return "", "", err
	}
	return projectKey, ticketID, nil
}

func (store WriteStore) projectKey(key, displayName string) string {
	if strings.TrimSpace(key) != "" {
		return NormalizeIDPart(key)
	}
	return NormalizeIDPart(displayName)
}

func (store WriteStore) timeOrNow(value time.Time) time.Time {
	if !value.IsZero() {
		return value
	}
	if store.Now != nil {
		return store.Now()
	}
	return time.Now()
}

func (store WriteStore) newID(prefix IDPrefix, at time.Time) (string, error) {
	if store.Entropy == nil {
		return NewID(prefix, at)
	}
	return NewIDWithReader(prefix, at, store.Entropy)
}

func (store WriteStore) newActorEventID(at time.Time, actor string) (string, error) {
	if store.Entropy == nil {
		return NewActorEventID(at, actor)
	}
	return NewActorEventIDWithReader(at, actor, store.Entropy)
}

func (store WriteStore) abs(relative string) string {
	return filepath.Join(store.Root, filepath.FromSlash(relative))
}

func renderProjectMarkdown(project ObjectProject) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Project: %s\n\n", project.DisplayName)
	buffer.WriteString("| Field | Value |\n")
	buffer.WriteString("|---|---|\n")
	writeMarkdownField(&buffer, "Project Key", project.Key)
	writeMarkdownField(&buffer, "Display Name", project.DisplayName)
	writeMarkdownField(&buffer, "Status", project.Status)
	writeMarkdownField(&buffer, "Created", project.Created)
	writeMarkdownField(&buffer, "Updated", project.Updated)
	if len(project.Tags) > 0 {
		writeMarkdownField(&buffer, "Tags", strings.Join(project.Tags, " "))
	}
	if project.Summary != "" {
		buffer.WriteString("\n## Summary\n\n")
		buffer.WriteString(project.Summary)
		buffer.WriteString("\n")
	}
	if len(project.Identity) > 0 {
		buffer.WriteString("\n## Identity\n\n")
		buffer.WriteString("| Field | Value |\n")
		buffer.WriteString("|---|---|\n")
		if project.Identity["Primary Repo"] != "" {
			writeMarkdownField(&buffer, "Primary Repo", project.Identity["Primary Repo"])
		}
		if project.Identity["Remote"] != "" {
			writeMarkdownField(&buffer, "Remote", project.Identity["Remote"])
		}
	}
	return buffer.String()
}

func renderGoalMarkdown(goal ObjectGoal) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Goal: %s\n\n", goal.Title)
	buffer.WriteString("| Field | Value |\n")
	buffer.WriteString("|---|---|\n")
	writeMarkdownField(&buffer, "Goal ID", goal.ID)
	writeMarkdownField(&buffer, "Project", goal.ProjectKey)
	writeMarkdownField(&buffer, "Status", goal.Status)
	writeMarkdownField(&buffer, "Created", goal.Created)
	if len(goal.Tags) > 0 {
		writeMarkdownField(&buffer, "Tags", strings.Join(goal.Tags, " "))
	}
	if goal.Outcome != "" {
		buffer.WriteString("\n## Outcome\n\n")
		buffer.WriteString(goal.Outcome)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func renderTicketMarkdown(ticket ObjectTicket) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Ticket: %s\n\n", ticket.Title)
	buffer.WriteString("| Field | Value |\n")
	buffer.WriteString("|---|---|\n")
	writeMarkdownField(&buffer, "Ticket ID", ticket.ID)
	writeMarkdownField(&buffer, "Project", ticket.ProjectKey)
	if ticket.GoalID == "" {
		writeMarkdownField(&buffer, "Goal", "none")
	} else {
		writeMarkdownField(&buffer, "Goal", ticket.GoalID)
	}
	writeMarkdownField(&buffer, "Status", ticket.Status)
	writeMarkdownField(&buffer, "Created", ticket.Created)
	if len(ticket.Tags) > 0 {
		writeMarkdownField(&buffer, "Tags", strings.Join(ticket.Tags, " "))
	}
	if ticket.Description != "" {
		buffer.WriteString("\n## Description\n\n")
		buffer.WriteString(ticket.Description)
		buffer.WriteString("\n")
	}
	if len(ticket.Acceptance) > 0 {
		buffer.WriteString("\n## Acceptance\n\n")
		for _, item := range ticket.Acceptance {
			fmt.Fprintf(&buffer, "- %s\n", item)
		}
	}
	return buffer.String()
}

func writeNewFile(path, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", path)
		}
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func defaultStatus(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func requireValidStatus(kind, status string) error {
	if !validStatuses[status] {
		return fmt.Errorf("invalid %s status %q", kind, status)
	}
	return nil
}

func requireActor(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return fmt.Errorf("event actor is required")
	}
	return nil
}

func cleanStrings(values []string) []string {
	var cleaned []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func cleanIdentity(primaryRepo, remote string) map[string]string {
	identity := map[string]string{}
	if strings.TrimSpace(primaryRepo) != "" {
		identity["Primary Repo"] = strings.TrimSpace(primaryRepo)
	}
	if strings.TrimSpace(remote) != "" {
		identity["Remote"] = strings.TrimSpace(remote)
	}
	if len(identity) == 0 {
		return nil
	}
	return identity
}

func defaultText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
