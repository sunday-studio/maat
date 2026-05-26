package maat

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ValidationReport struct {
	Files  int               `json:"files"`
	Issues []ValidationIssue `json:"issues"`
}

func (report ValidationReport) OK() bool {
	return len(report.Issues) == 0
}

type ValidationIssue struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type validatedProjectFile struct {
	project Project
	idLine  int
	issues  []ValidationIssue
}

// ValidateStore checks the current legacy flat Markdown store and returns all
// detected schema issues. Validation problems are reported in the result rather
// than as errors so callers can present the full list to users and agents.
func ValidateStore(store string) (ValidationReport, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", "*.md"))
	if err != nil {
		return ValidationReport{}, err
	}
	sort.Strings(paths)

	var report ValidationReport
	projectsByID := make(map[string]validatedProjectFile)
	for _, path := range paths {
		if strings.HasPrefix(filepath.Base(path), "_") || filepath.Base(path) == "README.md" {
			continue
		}
		report.Files++
		validated, err := validateProjectFile(store, path)
		if err != nil {
			return ValidationReport{}, err
		}
		report.Issues = append(report.Issues, validated.issues...)
		if validated.project.ID == "" {
			continue
		}
		if first, ok := projectsByID[validated.project.ID]; ok {
			report.Issues = append(report.Issues, ValidationIssue{
				Path:    validated.project.Path,
				Line:    validated.idLine,
				Code:    "duplicate_project_id",
				Message: fmt.Sprintf("project ID %q is already used by %s", validated.project.ID, first.project.Path),
			})
			continue
		}
		projectsByID[validated.project.ID] = validated
	}

	return report, nil
}

func validateProjectFile(store, path string) (validatedProjectFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return validatedProjectFile{}, err
	}
	defer file.Close()

	result := validatedProjectFile{
		project: Project{Path: relPath(store, path)},
	}
	var currentSection string
	var currentGoal *Goal
	var currentGoalLine int
	var sawProjectHeading bool
	projectFieldLines := make(map[string]int)
	goalIDs := make(map[string]int)
	ticketIDsByGoal := make(map[string]map[string]int)

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "# Project:") {
			sawProjectHeading = true
			result.project.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# Project:"))
			if result.project.Title == "" {
				result.addIssue(lineNumber, "missing_project_title", "project heading must include a title")
			}
			continue
		}
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "# Project:") {
			result.addIssue(lineNumber, "malformed_project_heading", "project files must start with a '# Project: <name>' heading")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			currentGoal = nil
			currentGoalLine = 0
			continue
		}
		if strings.HasPrefix(trimmed, "### ") && currentSection == "Goals" {
			goal := parseGoalHeading(trimmed)
			result.project.Goals = append(result.project.Goals, goal)
			currentGoal = &result.project.Goals[len(result.project.Goals)-1]
			currentGoalLine = lineNumber
			if goal.ID == "" || goal.Title == "" {
				result.addIssue(lineNumber, "malformed_goal_heading", "goal headings must use '### <goal-id>: <title>'")
			} else if firstLine, ok := goalIDs[goal.ID]; ok {
				result.addIssue(lineNumber, "duplicate_goal_id", fmt.Sprintf("goal ID %q is already used on line %d", goal.ID, firstLine))
			} else {
				goalIDs[goal.ID] = lineNumber
			}
			continue
		}
		if strings.HasPrefix(trimmed, "|") {
			key, value, ok := parseTableRow(trimmed)
			if ok {
				if currentGoal != nil {
					applyGoalField(currentGoal, key, value)
				} else {
					applyProjectField(&result.project, key, value)
					projectFieldLines[strings.ToLower(key)] = lineNumber
					if strings.EqualFold(key, "ID") {
						result.idLine = lineNumber
					}
				}
				continue
			}
			if isMalformedTableRow(trimmed) {
				result.addIssue(lineNumber, "malformed_table_row", "table rows must use '| Field | Value |' cells")
			}
			continue
		}
		if currentSection == "Goals" && currentGoal != nil && strings.HasPrefix(trimmed, "- [") {
			ticket, ok := parseTicket(trimmed)
			if !ok || ticket.ID == "" || ticket.Title == "" {
				result.addIssue(lineNumber, "malformed_ticket", "tickets must use '- [ ] <ticket-id>: <title>' or '- [x] <ticket-id>: <title>'")
				continue
			}
			currentGoal.Tickets = append(currentGoal.Tickets, ticket)
			goalKey := currentGoal.ID
			if goalKey == "" {
				goalKey = fmt.Sprintf("line-%d", currentGoalLine)
			}
			if ticketIDsByGoal[goalKey] == nil {
				ticketIDsByGoal[goalKey] = make(map[string]int)
			}
			if firstLine, ok := ticketIDsByGoal[goalKey][ticket.ID]; ok {
				result.addIssue(lineNumber, "duplicate_ticket_id", fmt.Sprintf("ticket ID %q is already used in this goal on line %d", ticket.ID, firstLine))
			} else {
				ticketIDsByGoal[goalKey][ticket.ID] = lineNumber
			}
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return validatedProjectFile{}, err
	}

	if !sawProjectHeading {
		result.addIssue(1, "missing_project_heading", "project file must include a '# Project: <name>' heading")
	}
	for _, field := range []string{"id", "status", "owner", "updated"} {
		if projectFieldLines[field] == 0 {
			result.addIssue(0, "missing_project_field", fmt.Sprintf("project is missing required field %q", field))
		}
	}
	if result.project.Status != "" && !validStatuses[result.project.Status] {
		result.addIssue(projectFieldLines["status"], "invalid_project_status", fmt.Sprintf("project status %q is not valid", result.project.Status))
	}
	for _, goal := range result.project.Goals {
		if goal.Status != "" && !validStatuses[goal.Status] {
			result.addIssue(0, "invalid_goal_status", fmt.Sprintf("goal %q has invalid status %q", goal.ID, goal.Status))
		}
	}
	if result.idLine == 0 && result.project.ID != "" {
		result.idLine = projectFieldLines["id"]
	}
	return result, nil
}

func (result *validatedProjectFile) addIssue(line int, code, message string) {
	result.issues = append(result.issues, ValidationIssue{
		Path:    result.project.Path,
		Line:    line,
		Code:    code,
		Message: message,
	})
}

func isMalformedTableRow(line string) bool {
	if line == "" || strings.EqualFold(line, "| Field | Value |") {
		return false
	}
	withoutPipes := strings.ReplaceAll(line, "|", "")
	if strings.Trim(withoutPipes, "- ") == "" {
		return false
	}
	return strings.Count(line, "|") < 3
}
