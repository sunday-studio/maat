package maat

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ObjectStore struct {
	Projects []ObjectProject `json:"projects"`
}

type ObjectProject struct {
	Key         string            `json:"key"`
	DisplayName string            `json:"display_name"`
	Status      string            `json:"status"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
	Tags        []string          `json:"tags,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Identity    map[string]string `json:"identity,omitempty"`
	Path        string            `json:"path"`
	Goals       []ObjectGoal      `json:"goals,omitempty"`
	Tickets     []ObjectTicket    `json:"tickets,omitempty"`
	Events      []ObjectEvent     `json:"events,omitempty"`
}

type ObjectGoal struct {
	ID         string   `json:"id"`
	ProjectKey string   `json:"project_key"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Created    string   `json:"created"`
	Tags       []string `json:"tags,omitempty"`
	Outcome    string   `json:"outcome,omitempty"`
	Path       string   `json:"path"`
}

type ObjectTicket struct {
	ID          string   `json:"id"`
	ProjectKey  string   `json:"project_key"`
	GoalID      string   `json:"goal_id,omitempty"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Created     string   `json:"created"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
	Acceptance  []string `json:"acceptance,omitempty"`
	Path        string   `json:"path"`
}

type ObjectEvent struct {
	ID         string   `json:"id"`
	Time       string   `json:"time"`
	Actor      string   `json:"actor"`
	ProjectKey string   `json:"project_key"`
	Type       string   `json:"type"`
	ObjectID   string   `json:"object_id"`
	Commit     string   `json:"commit,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	Path       string   `json:"path"`
}

type markdownObject struct {
	Title    string
	Fields   map[string]string
	Sections map[string]string
}

func LoadObjectStore(store string) (ObjectStore, error) {
	projects, err := LoadObjectProjects(store)
	if err != nil {
		return ObjectStore{}, err
	}
	return ObjectStore{Projects: projects}, nil
}

func LoadObjectProjects(store string) ([]ObjectProject, error) {
	projectsDir := filepath.Join(store, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []ObjectProject
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		project, err := LoadObjectProject(store, entry.Name())
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Key < projects[j].Key
	})
	return projects, nil
}

func LoadObjectProject(store, projectKey string) (ObjectProject, error) {
	projectDir := filepath.Join(store, "projects", projectKey)
	project, err := ParseObjectProjectFile(store, filepath.Join(projectDir, "project.md"))
	if err != nil {
		return ObjectProject{}, err
	}
	if project.Key != projectKey {
		return ObjectProject{}, fmt.Errorf("%s: project key %q does not match directory %q", project.Path, project.Key, projectKey)
	}

	goals, err := loadObjectGoals(store, projectKey)
	if err != nil {
		return ObjectProject{}, err
	}
	tickets, err := loadObjectTickets(store, projectKey)
	if err != nil {
		return ObjectProject{}, err
	}
	events, err := loadObjectEvents(store, projectKey)
	if err != nil {
		return ObjectProject{}, err
	}

	project.Goals = goals
	project.Tickets = tickets
	project.Events = events
	return project, nil
}

func ParseObjectProjectFile(store, path string) (ObjectProject, error) {
	doc, err := parseMarkdownObject(path)
	if err != nil {
		return ObjectProject{}, err
	}
	project := ObjectProject{
		Key:         doc.Fields["project key"],
		DisplayName: doc.Fields["display name"],
		Status:      doc.Fields["status"],
		Created:     doc.Fields["created"],
		Updated:     doc.Fields["updated"],
		Tags:        splitTags(doc.Fields["tags"]),
		Summary:     strings.TrimSpace(doc.Sections["summary"]),
		Identity:    pickFields(doc.Fields, "Primary Repo", "Remote"),
		Path:        relPath(store, path),
	}
	if project.DisplayName == "" {
		project.DisplayName = strings.TrimSpace(strings.TrimPrefix(doc.Title, "Project:"))
	}
	if err := requireFields(project.Path, map[string]string{
		"Project Key":  project.Key,
		"Display Name": project.DisplayName,
		"Status":       project.Status,
		"Created":      project.Created,
		"Updated":      project.Updated,
	}); err != nil {
		return ObjectProject{}, err
	}
	if !validStatuses[project.Status] {
		return ObjectProject{}, fmt.Errorf("%s: invalid project status %q", project.Path, project.Status)
	}
	return project, nil
}

func ParseObjectGoalFile(store, path string) (ObjectGoal, error) {
	doc, err := parseMarkdownObject(path)
	if err != nil {
		return ObjectGoal{}, err
	}
	goal := ObjectGoal{
		ID:         doc.Fields["goal id"],
		ProjectKey: doc.Fields["project"],
		Title:      strings.TrimSpace(strings.TrimPrefix(doc.Title, "Goal:")),
		Status:     doc.Fields["status"],
		Created:    doc.Fields["created"],
		Tags:       splitTags(doc.Fields["tags"]),
		Outcome:    strings.TrimSpace(doc.Sections["outcome"]),
		Path:       relPath(store, path),
	}
	if err := requireFields(goal.Path, map[string]string{
		"Goal ID": goal.ID,
		"Project": goal.ProjectKey,
		"Status":  goal.Status,
		"Created": goal.Created,
	}); err != nil {
		return ObjectGoal{}, err
	}
	if !validStatuses[goal.Status] {
		return ObjectGoal{}, fmt.Errorf("%s: invalid goal status %q", goal.Path, goal.Status)
	}
	return goal, nil
}

func ParseObjectTicketFile(store, path string) (ObjectTicket, error) {
	doc, err := parseMarkdownObject(path)
	if err != nil {
		return ObjectTicket{}, err
	}
	ticket := ObjectTicket{
		ID:          doc.Fields["ticket id"],
		ProjectKey:  doc.Fields["project"],
		GoalID:      doc.Fields["goal"],
		Title:       strings.TrimSpace(strings.TrimPrefix(doc.Title, "Ticket:")),
		Status:      doc.Fields["status"],
		Created:     doc.Fields["created"],
		Tags:        splitTags(doc.Fields["tags"]),
		Description: strings.TrimSpace(doc.Sections["description"]),
		Acceptance:  parseBulletList(doc.Sections["acceptance"]),
		Path:        relPath(store, path),
	}
	if err := requireFields(ticket.Path, map[string]string{
		"Ticket ID": ticket.ID,
		"Project":   ticket.ProjectKey,
		"Status":    ticket.Status,
		"Created":   ticket.Created,
	}); err != nil {
		return ObjectTicket{}, err
	}
	if !validStatuses[ticket.Status] {
		return ObjectTicket{}, fmt.Errorf("%s: invalid ticket status %q", ticket.Path, ticket.Status)
	}
	return ticket, nil
}

func ParseObjectEventFile(store, path string) (ObjectEvent, error) {
	doc, err := parseMarkdownObject(path)
	if err != nil {
		return ObjectEvent{}, err
	}
	event := ObjectEvent{
		ID:         doc.Fields["event id"],
		Time:       doc.Fields["time"],
		Actor:      doc.Fields["actor"],
		ProjectKey: doc.Fields["project"],
		Type:       doc.Fields["type"],
		ObjectID:   doc.Fields["object"],
		Commit:     doc.Fields["commit"],
		Summary:    strings.TrimSpace(doc.Sections["summary"]),
		Evidence:   parseBulletList(doc.Sections["evidence"]),
		Path:       relPath(store, path),
	}
	if err := requireFields(event.Path, map[string]string{
		"Event ID": event.ID,
		"Time":     event.Time,
		"Actor":    event.Actor,
		"Project":  event.ProjectKey,
		"Type":     event.Type,
		"Object":   event.ObjectID,
	}); err != nil {
		return ObjectEvent{}, err
	}
	return event, nil
}

func loadObjectGoals(store, projectKey string) ([]ObjectGoal, error) {
	paths, err := filepath.Glob(filepath.Join(store, "projects", projectKey, "goals", "*.md"))
	if err != nil {
		return nil, err
	}
	goals := make([]ObjectGoal, 0, len(paths))
	for _, path := range paths {
		goal, err := ParseObjectGoalFile(store, path)
		if err != nil {
			return nil, err
		}
		if goal.ProjectKey != projectKey {
			return nil, fmt.Errorf("%s: goal project %q does not match directory %q", goal.Path, goal.ProjectKey, projectKey)
		}
		goals = append(goals, goal)
	}
	sort.Slice(goals, func(i, j int) bool {
		return goals[i].ID < goals[j].ID
	})
	return goals, nil
}

func loadObjectTickets(store, projectKey string) ([]ObjectTicket, error) {
	paths, err := filepath.Glob(filepath.Join(store, "projects", projectKey, "tickets", "*.md"))
	if err != nil {
		return nil, err
	}
	tickets := make([]ObjectTicket, 0, len(paths))
	for _, path := range paths {
		ticket, err := ParseObjectTicketFile(store, path)
		if err != nil {
			return nil, err
		}
		if ticket.ProjectKey != projectKey {
			return nil, fmt.Errorf("%s: ticket project %q does not match directory %q", ticket.Path, ticket.ProjectKey, projectKey)
		}
		tickets = append(tickets, ticket)
	}
	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].ID < tickets[j].ID
	})
	return tickets, nil
}

func loadObjectEvents(store, projectKey string) ([]ObjectEvent, error) {
	var events []ObjectEvent
	eventsDir := filepath.Join(store, "projects", projectKey, "events")
	err := filepath.WalkDir(eventsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		event, err := ParseObjectEventFile(store, path)
		if err != nil {
			return err
		}
		if event.ProjectKey != projectKey {
			return fmt.Errorf("%s: event project %q does not match directory %q", event.Path, event.ProjectKey, projectKey)
		}
		events = append(events, event)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			return events[i].ID < events[j].ID
		}
		return events[i].Time < events[j].Time
	})
	return events, nil
}

func parseMarkdownObject(path string) (markdownObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return markdownObject{}, err
	}

	doc := markdownObject{
		Fields:   map[string]string{},
		Sections: map[string]string{},
	}
	var section string
	var sectionLines []string
	flushSection := func() {
		if section == "" {
			return
		}
		doc.Sections[strings.ToLower(section)] = strings.TrimSpace(strings.Join(sectionLines, "\n"))
		sectionLines = nil
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# "):
			doc.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			continue
		case strings.HasPrefix(trimmed, "## "):
			flushSection()
			section = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		case strings.HasPrefix(trimmed, "|"):
			key, value, ok := parseTableRow(trimmed)
			if ok {
				doc.Fields[strings.ToLower(key)] = value
			}
		}
		if section != "" {
			sectionLines = append(sectionLines, line)
		}
	}
	flushSection()
	return doc, nil
}

func requireFields(path string, fields map[string]string) error {
	for label, value := range fields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s: missing %s", path, label)
		}
	}
	return nil
}

func splitTags(value string) []string {
	return strings.Fields(value)
}

func pickFields(fields map[string]string, names ...string) map[string]string {
	picked := map[string]string{}
	for _, name := range names {
		key := strings.ToLower(name)
		if fields[key] != "" {
			picked[name] = fields[key]
		}
	}
	if len(picked) == 0 {
		return nil
	}
	return picked
}

func parseBulletList(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		}
	}
	return items
}
