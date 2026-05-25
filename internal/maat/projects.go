package maat

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var validStatuses = map[string]bool{
	"proposed": true,
	"active":   true,
	"waiting":  true,
	"paused":   true,
	"done":     true,
	"archived": true,
}

func LoadProjects(store string) ([]Project, error) {
	paths, err := filepath.Glob(filepath.Join(store, "projects", "*.md"))
	if err != nil {
		return nil, err
	}
	projects := make([]Project, 0, len(paths))
	for _, path := range paths {
		if strings.HasPrefix(filepath.Base(path), "_") || filepath.Base(path) == "README.md" {
			continue
		}
		project, err := ParseProjectFile(store, path)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ID < projects[j].ID
	})
	return projects, nil
}

func LoadProject(store, id string) (Project, error) {
	projects, err := LoadProjects(store)
	if err != nil {
		return Project{}, err
	}
	for _, project := range projects {
		if project.ID == id {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("project %q not found", id)
}

func ParseProjectFile(store, path string) (Project, error) {
	file, err := os.Open(path)
	if err != nil {
		return Project{}, err
	}
	defer file.Close()

	project := Project{Path: relPath(store, path)}
	var currentSection string
	var currentGoal *Goal
	var collectingCurrent []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "# Project:") {
			project.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# Project:"))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			if currentSection == "Current" {
				project.Current = strings.TrimSpace(strings.Join(collectingCurrent, "\n"))
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			currentGoal = nil
			collectingCurrent = nil
			continue
		}
		if strings.HasPrefix(trimmed, "### ") && currentSection == "Goals" {
			goal := parseGoalHeading(trimmed)
			project.Goals = append(project.Goals, goal)
			currentGoal = &project.Goals[len(project.Goals)-1]
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			key, value, ok := parseTableRow(trimmed)
			if ok {
				if currentGoal != nil {
					applyGoalField(currentGoal, key, value)
				} else {
					applyProjectField(&project, key, value)
				}
			}
			continue
		}
		if currentSection == "Current" {
			collectingCurrent = append(collectingCurrent, line)
			continue
		}
		if currentSection == "Goals" && currentGoal != nil && strings.HasPrefix(trimmed, "- [") {
			if ticket, ok := parseTicket(trimmed); ok {
				currentGoal.Tickets = append(currentGoal.Tickets, ticket)
			}
			continue
		}
		if currentSection == "Blockers" && strings.HasPrefix(trimmed, "- ") {
			project.Blockers = append(project.Blockers, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			continue
		}
		if currentSection == "Decisions" && strings.HasPrefix(trimmed, "- ") {
			project.Decisions = append(project.Decisions, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return Project{}, err
	}
	if currentSection == "Current" {
		project.Current = strings.TrimSpace(strings.Join(collectingCurrent, "\n"))
	}
	if project.ID == "" {
		return Project{}, errors.New("missing project ID")
	}
	if project.Title == "" {
		project.Title = project.ID
	}
	if project.Status == "" {
		return Project{}, fmt.Errorf("%s: missing project status", project.Path)
	}
	if !validStatuses[project.Status] {
		return Project{}, fmt.Errorf("%s: invalid project status %q", project.Path, project.Status)
	}
	for _, goal := range project.Goals {
		if goal.Status != "" && !validStatuses[goal.Status] {
			return Project{}, fmt.Errorf("%s: invalid goal status %q", project.Path, goal.Status)
		}
	}
	return project, nil
}

func Status(store string) (StatusSummary, error) {
	projects, err := LoadProjects(store)
	if err != nil {
		return StatusSummary{}, err
	}
	var summary StatusSummary
	summary.Projects = len(projects)
	for _, project := range projects {
		for _, blocker := range project.Blockers {
			if blocker != "" && strings.ToLower(blocker) != "none." && strings.ToLower(blocker) != "none" {
				summary.BlockedItems++
			}
		}
		for _, decision := range project.Decisions {
			if decision != "" && strings.ToLower(decision) != "none." && strings.ToLower(decision) != "none" {
				summary.DecisionItems++
			}
		}
		for _, goal := range project.Goals {
			summary.Goals++
			if goal.Status == "done" {
				summary.DoneGoals++
			}
			if goal.Status == "active" {
				summary.ActiveGoals++
			}
			for _, ticket := range goal.Tickets {
				summary.Tickets++
				if ticket.Done {
					summary.DoneTickets++
				} else {
					summary.OpenTickets++
				}
			}
		}
	}
	return summary, nil
}

func applyProjectField(project *Project, key, value string) {
	switch strings.ToLower(key) {
	case "id":
		project.ID = value
	case "status":
		project.Status = value
	case "owner":
		project.Owner = value
	case "updated":
		project.Updated = value
	case "tags":
		project.Tags = strings.Fields(value)
	}
}

func applyGoalField(goal *Goal, key, value string) {
	switch strings.ToLower(key) {
	case "status":
		goal.Status = value
	case "updated":
		goal.Updated = value
	case "tags":
		goal.Tags = strings.Fields(value)
	}
}

func parseGoalHeading(line string) Goal {
	title := strings.TrimSpace(strings.TrimPrefix(line, "### "))
	id, rest, ok := strings.Cut(title, ":")
	if !ok {
		return Goal{Title: title}
	}
	return Goal{ID: strings.TrimSpace(id), Title: strings.TrimSpace(rest)}
}

func parseTicket(line string) (Ticket, bool) {
	done := strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]")
	open := strings.HasPrefix(line, "- [ ]")
	if !done && !open {
		return Ticket{}, false
	}
	body := strings.TrimSpace(line[5:])
	id, title, ok := strings.Cut(body, ":")
	if !ok {
		return Ticket{Title: body, Done: done}, true
	}
	return Ticket{ID: strings.TrimSpace(id), Title: strings.TrimSpace(title), Done: done}, true
}

func parseTableRow(line string) (string, string, bool) {
	cells := strings.Split(line, "|")
	if len(cells) < 4 {
		return "", "", false
	}
	key := strings.TrimSpace(cells[1])
	value := strings.TrimSpace(cells[2])
	if key == "" || key == "---" || strings.Contains(key, "---") {
		return "", "", false
	}
	if strings.EqualFold(key, "Field") && strings.EqualFold(value, "Value") {
		return "", "", false
	}
	return key, value, true
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
