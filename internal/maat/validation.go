package maat

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

type validatedMarkdownObject struct {
	path         string
	title        string
	titleLine    int
	fields       map[string]string
	fieldLines   map[string]int
	sections     map[string]string
	sectionLines map[string]int
	issues       []ValidationIssue
}

type objectProjectValidation struct {
	key       string
	path      string
	goalIDs   map[string]string
	ticketIDs map[string]string
	eventIDs  map[string]string
}

// ValidateStore checks project directories in the object-per-file layout.
// Validation problems are reported in the result rather than as errors so
// callers can present the full list to users and agents.
func ValidateStore(store string) (ValidationReport, error) {
	var report ValidationReport
	if err := validateObjectProjectDirectories(store, &report); err != nil {
		return ValidationReport{}, err
	}
	return report, nil
}

func validateObjectProjectDirectories(store string, report *ValidationReport) error {
	projectsDir := contentPath(store, "projects")
	entries, err := os.ReadDir(projectsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		state := objectProjectValidation{
			key:       entry.Name(),
			goalIDs:   map[string]string{},
			ticketIDs: map[string]string{},
			eventIDs:  map[string]string{},
		}
		projectPath := filepath.Join(projectsDir, entry.Name(), "project.md")
		state.path = relPath(store, projectPath)
		if _, err := os.Stat(projectPath); err != nil {
			if os.IsNotExist(err) {
				addValidationIssue(report, store, projectPath, 0, "missing_project_file", "object project directory must contain project.md")
				continue
			}
			return err
		}
		if err := validateObjectProjectFile(store, projectPath, &state, report); err != nil {
			return err
		}
		if err := validateObjectGoalFiles(store, entry.Name(), &state, report); err != nil {
			return err
		}
		if err := validateObjectTicketFiles(store, entry.Name(), &state, report); err != nil {
			return err
		}
		if err := validateObjectEventFiles(store, entry.Name(), &state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateObjectProjectFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Project", "missing_project_heading", "malformed_project_heading", "missing_project_title")
	requireObjectFields(report, doc, "missing_project_field", "Project Key", "Display Name", "Status", "Created", "Updated")

	projectKey := doc.field("Project Key")
	if projectKey != "" && projectKey != state.key {
		addValidationIssue(report, store, path, doc.fieldLine("Project Key"), "project_key_mismatch", fmt.Sprintf("project key %q does not match directory %q", projectKey, state.key))
	}
	validateObjectStatus(report, doc, "Status", "invalid_project_status", "project")
	validateRFC3339Field(report, doc, "Created", "invalid_project_timestamp")
	validateRFC3339Field(report, doc, "Updated", "invalid_project_timestamp")
	return nil
}

func validateObjectGoalFiles(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "goals", "*.md"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		if err := validateObjectGoalFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateObjectGoalFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Goal", "missing_goal_heading", "malformed_goal_heading", "missing_goal_title")
	requireObjectFields(report, doc, "missing_goal_field", "Goal ID", "Project", "Status", "Created")

	goalID := doc.field("Goal ID")
	if goalID != "" {
		if firstPath, ok := state.goalIDs[goalID]; ok {
			addValidationIssue(report, store, path, doc.fieldLine("Goal ID"), "duplicate_goal_id", fmt.Sprintf("goal ID %q is already used by %s", goalID, firstPath))
		} else {
			state.goalIDs[goalID] = doc.path
		}
		if filenameID(path) != goalID {
			addValidationIssue(report, store, path, doc.fieldLine("Goal ID"), "goal_id_filename_mismatch", fmt.Sprintf("goal ID %q does not match filename %q", goalID, filenameID(path)))
		}
	}
	if project := doc.field("Project"); project != "" && project != state.key {
		addValidationIssue(report, store, path, doc.fieldLine("Project"), "goal_project_mismatch", fmt.Sprintf("goal project %q does not match directory %q", project, state.key))
	}
	validateObjectStatus(report, doc, "Status", "invalid_goal_status", "goal")
	validateRFC3339Field(report, doc, "Created", "invalid_goal_timestamp")
	if strings.TrimSpace(doc.sections["outcome"]) == "" {
		addValidationIssue(report, store, path, doc.sectionLines["outcome"], "missing_goal_outcome", "goal must include a non-empty Outcome section")
	}
	return nil
}

func validateObjectTicketFiles(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "tickets", "*.md"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		if err := validateObjectTicketFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateObjectTicketFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Ticket", "missing_ticket_heading", "malformed_ticket_heading", "missing_ticket_title")
	requireObjectFields(report, doc, "missing_ticket_field", "Ticket ID", "Project", "Goal", "Status", "Created")

	ticketID := doc.field("Ticket ID")
	if ticketID != "" {
		if firstPath, ok := state.ticketIDs[ticketID]; ok {
			addValidationIssue(report, store, path, doc.fieldLine("Ticket ID"), "duplicate_ticket_id", fmt.Sprintf("ticket ID %q is already used by %s", ticketID, firstPath))
		} else {
			state.ticketIDs[ticketID] = doc.path
		}
		if filenameID(path) != ticketID {
			addValidationIssue(report, store, path, doc.fieldLine("Ticket ID"), "ticket_id_filename_mismatch", fmt.Sprintf("ticket ID %q does not match filename %q", ticketID, filenameID(path)))
		}
	}
	if project := doc.field("Project"); project != "" && project != state.key {
		addValidationIssue(report, store, path, doc.fieldLine("Project"), "ticket_project_mismatch", fmt.Sprintf("ticket project %q does not match directory %q", project, state.key))
	}
	goalID := optionalObjectLink(doc.field("Goal"))
	if goalID != "" {
		if _, ok := state.goalIDs[goalID]; !ok {
			addValidationIssue(report, store, path, doc.fieldLine("Goal"), "unknown_ticket_goal", fmt.Sprintf("ticket references missing goal %q", goalID))
		}
	}
	validateObjectStatus(report, doc, "Status", "invalid_ticket_status", "ticket")
	validateRFC3339Field(report, doc, "Created", "invalid_ticket_timestamp")
	if strings.TrimSpace(doc.sections["description"]) == "" {
		addValidationIssue(report, store, path, doc.sectionLines["description"], "missing_ticket_description", "ticket must include a non-empty Description section")
	}
	if len(parseBulletList(doc.sections["acceptance"])) == 0 {
		addValidationIssue(report, store, path, doc.sectionLines["acceptance"], "missing_ticket_acceptance", "ticket must include at least one Acceptance bullet")
	}
	return nil
}

func validateObjectEventFiles(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	var paths []string
	eventsDir := contentPath(store, "projects", projectKey, "events")
	err := filepath.WalkDir(eventsDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") || isSkippedMarkdownFile(path) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := validateObjectEventFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateObjectEventFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Event", "missing_event_heading", "malformed_event_heading", "missing_event_type")
	requireObjectFields(report, doc, "missing_event_field", "Event ID", "Time", "Actor", "Project", "Type", "Object")

	eventID := doc.field("Event ID")
	if eventID != "" {
		if firstPath, ok := state.eventIDs[eventID]; ok {
			addValidationIssue(report, store, path, doc.fieldLine("Event ID"), "duplicate_event_id", fmt.Sprintf("event ID %q is already used by %s", eventID, firstPath))
		} else {
			state.eventIDs[eventID] = doc.path
		}
		if filenameID(path) != eventID {
			addValidationIssue(report, store, path, doc.fieldLine("Event ID"), "event_id_filename_mismatch", fmt.Sprintf("event ID %q does not match filename %q", eventID, filenameID(path)))
		}
	}
	eventTime, ok := validateRFC3339Field(report, doc, "Time", "invalid_event_time")
	if ok {
		validateEventTimePath(report, store, path, doc, eventTime)
	}
	if project := doc.field("Project"); project != "" && project != state.key {
		addValidationIssue(report, store, path, doc.fieldLine("Project"), "event_project_mismatch", fmt.Sprintf("event project %q does not match directory %q", project, state.key))
	}
	if strings.TrimSpace(doc.sections["summary"]) == "" {
		addValidationIssue(report, store, path, 0, "missing_event_summary", "event must include a non-empty Summary section")
	}
	validateEventObjectReference(report, store, path, doc, state)
	return nil
}

func scanMarkdownObjectForValidation(store, path string) (validatedMarkdownObject, error) {
	file, err := os.Open(path)
	if err != nil {
		return validatedMarkdownObject{}, err
	}
	defer file.Close()

	doc := validatedMarkdownObject{
		path:         relPath(store, path),
		fields:       map[string]string{},
		fieldLines:   map[string]int{},
		sections:     map[string]string{},
		sectionLines: map[string]int{},
	}
	var section string
	var sectionLines []string
	flushSection := func() {
		if section == "" {
			return
		}
		doc.sections[strings.ToLower(section)] = strings.TrimSpace(strings.Join(sectionLines, "\n"))
		sectionLines = nil
	}

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "# "):
			if doc.titleLine == 0 {
				doc.title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
				doc.titleLine = lineNumber
			}
			continue
		case strings.HasPrefix(trimmed, "## "):
			flushSection()
			section = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			doc.sectionLines[strings.ToLower(section)] = lineNumber
			continue
		case strings.HasPrefix(trimmed, "|"):
			key, value, ok := parseTableRow(trimmed)
			if ok {
				doc.fields[strings.ToLower(key)] = value
				doc.fieldLines[strings.ToLower(key)] = lineNumber
			} else if isMalformedTableRow(trimmed) {
				doc.issues = append(doc.issues, ValidationIssue{
					Path:    doc.path,
					Line:    lineNumber,
					Code:    "malformed_table_row",
					Message: "table rows must use '| Field | Value |' cells",
				})
			}
			continue
		}
		if section != "" {
			sectionLines = append(sectionLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return validatedMarkdownObject{}, err
	}
	flushSection()
	return doc, nil
}

func validateObjectHeading(report *ValidationReport, doc validatedMarkdownObject, kind, missingCode, malformedCode, emptyCode string) {
	if doc.titleLine == 0 {
		report.Issues = append(report.Issues, ValidationIssue{
			Path:    doc.path,
			Line:    1,
			Code:    missingCode,
			Message: fmt.Sprintf("%s file must include a '# %s: <name>' heading", strings.ToLower(kind), kind),
		})
		return
	}
	prefix := kind + ":"
	if !strings.HasPrefix(doc.title, prefix) {
		report.Issues = append(report.Issues, ValidationIssue{
			Path:    doc.path,
			Line:    doc.titleLine,
			Code:    malformedCode,
			Message: fmt.Sprintf("%s heading must use '# %s: <name>'", strings.ToLower(kind), kind),
		})
		return
	}
	if strings.TrimSpace(strings.TrimPrefix(doc.title, prefix)) == "" {
		report.Issues = append(report.Issues, ValidationIssue{
			Path:    doc.path,
			Line:    doc.titleLine,
			Code:    emptyCode,
			Message: fmt.Sprintf("%s heading must include a name", strings.ToLower(kind)),
		})
	}
}

func requireObjectFields(report *ValidationReport, doc validatedMarkdownObject, code string, fields ...string) {
	for _, field := range fields {
		if strings.TrimSpace(doc.field(field)) == "" {
			report.Issues = append(report.Issues, ValidationIssue{
				Path:    doc.path,
				Code:    code,
				Message: fmt.Sprintf("object is missing required field %q", field),
			})
		}
	}
}

func validateObjectStatus(report *ValidationReport, doc validatedMarkdownObject, field, code, kind string) {
	status := doc.field(field)
	if status == "" || validStatuses[status] {
		return
	}
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    doc.path,
		Line:    doc.fieldLine(field),
		Code:    code,
		Message: fmt.Sprintf("%s status %q is not valid", kind, status),
	})
}

func validateRFC3339Field(report *ValidationReport, doc validatedMarkdownObject, field, code string) (time.Time, bool) {
	value := doc.field(field)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, true
	}
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    doc.path,
		Line:    doc.fieldLine(field),
		Code:    code,
		Message: fmt.Sprintf("%s must be an RFC3339 timestamp", field),
	})
	return time.Time{}, false
}

func validateEventTimePath(report *ValidationReport, store, path string, doc validatedMarkdownObject, eventTime time.Time) {
	rel := filepath.ToSlash(relPath(store, path))
	parts := strings.Split(rel, "/")
	if len(parts) < 6 || parts[0] != "projects" || parts[2] != "events" {
		return
	}
	wantYear := eventTime.Format("2006")
	wantMonth := eventTime.Format("01")
	if parts[3] != wantYear || parts[4] != wantMonth {
		addValidationIssue(report, store, path, doc.fieldLine("Time"), "event_time_path_mismatch", fmt.Sprintf("event time belongs under events/%s/%s", wantYear, wantMonth))
	}
}

func validateEventObjectReference(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation) {
	eventType := doc.field("Type")
	objectID := doc.field("Object")
	if eventType == "" || objectID == "" {
		return
	}
	switch {
	case strings.HasPrefix(eventType, "goal."):
		if _, ok := state.goalIDs[objectID]; !ok {
			addValidationIssue(report, store, path, doc.fieldLine("Object"), "unknown_event_object", fmt.Sprintf("event references missing goal %q", objectID))
		}
	case strings.HasPrefix(eventType, "ticket."):
		if _, ok := state.ticketIDs[objectID]; !ok {
			addValidationIssue(report, store, path, doc.fieldLine("Object"), "unknown_event_object", fmt.Sprintf("event references missing ticket %q", objectID))
		}
	case strings.HasPrefix(eventType, "project."):
		if objectID != state.key {
			addValidationIssue(report, store, path, doc.fieldLine("Object"), "unknown_event_object", fmt.Sprintf("event references project %q, expected %q", objectID, state.key))
		}
	}
}

func (doc validatedMarkdownObject) field(name string) string {
	return doc.fields[strings.ToLower(name)]
}

func (doc validatedMarkdownObject) fieldLine(name string) int {
	return doc.fieldLines[strings.ToLower(name)]
}

func addValidationIssue(report *ValidationReport, store, path string, line int, code, message string) {
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    relPath(store, path),
		Line:    line,
		Code:    code,
		Message: message,
	})
}

func filenameID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func isSkippedMarkdownFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "_") || base == "README.md"
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
