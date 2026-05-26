package maat

import (
	"path/filepath"
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

func Status(store string) (StatusSummary, error) {
	projects, err := LoadObjectProjects(store)
	if err != nil {
		return StatusSummary{}, err
	}
	var summary StatusSummary
	summary.Projects = len(projects)
	for _, project := range projects {
		for _, goal := range project.Goals {
			summary.Goals++
			switch goal.Status {
			case "done":
				summary.DoneGoals++
			case "active":
				summary.ActiveGoals++
			}
		}
		for _, ticket := range project.Tickets {
			summary.Tickets++
			if ticket.Status == "done" {
				summary.DoneTickets++
			} else {
				summary.OpenTickets++
			}
		}
	}
	return summary, nil
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
