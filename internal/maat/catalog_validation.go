package maat

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var validCatalogDecisionStates = map[string]bool{
	"adopt":          true,
	"adopt later":    true,
	"reject":         true,
	"needs research": true,
}

var validCatalogOpportunityStatuses = map[string]bool{
	"proposed":    true,
	"ticketed":    true,
	"in progress": true,
	"verified":    true,
	"declined":    true,
}

func validateObjectCatalogFiles(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	if err := validateCatalogApps(store, projectKey, state, report); err != nil {
		return err
	}
	if err := validateCatalogPatterns(store, projectKey, state, report); err != nil {
		return err
	}
	if err := validateCatalogDecisions(store, projectKey, state, report); err != nil {
		return err
	}
	if err := validateCatalogOpportunities(store, projectKey, state, report); err != nil {
		return err
	}
	if err := validateCatalogEvents(store, projectKey, state, report); err != nil {
		return err
	}
	return nil
}

func validateCatalogApps(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := catalogMarkdownFiles(store, projectKey, "apps")
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := validateCatalogAppFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func catalogMarkdownFiles(store, projectKey, dir string) ([]string, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "catalog", dir, "*.md"))
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		filtered = append(filtered, path)
	}
	sort.Strings(filtered)
	return filtered, nil
}

func validateCatalogAppFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Catalog App", "missing_catalog_app_heading", "malformed_catalog_app_heading", "missing_catalog_app_title")
	requireObjectFields(report, doc, "missing_catalog_app_field", "App ID", "Project", "Slug", "Name", "Summary", "Source URL", "Category", "Last Reviewed")
	validateCatalogProject(report, doc, state, "catalog_app_project_mismatch")
	validateCatalogID(report, store, path, doc, state, "App ID", "duplicate_catalog_id")
	validateCatalogSlug(report, store, path, doc, state, "Slug", "duplicate_catalog_slug", "catalog_app_slug_filename_mismatch")
	if slug := doc.field("Slug"); slug != "" {
		state.catalogAppSlugs[slug] = doc.path
	}
	validateCatalogURLField(report, doc, "Source URL", "invalid_catalog_source_url")
	validateCatalogURLField(report, doc, "Website URL", "invalid_catalog_website_url")
	validateCatalogDateField(report, doc, "Last Reviewed", "invalid_catalog_review_date")
	return nil
}

func validateCatalogPatterns(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := catalogMarkdownFiles(store, projectKey, "patterns")
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := validateCatalogPatternFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogPatternFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Catalog Pattern", "missing_catalog_pattern_heading", "malformed_catalog_pattern_heading", "missing_catalog_pattern_title")
	requireObjectFields(report, doc, "missing_catalog_pattern_field", "Pattern ID", "Project", "Slug", "Title", "Category")
	validateCatalogProject(report, doc, state, "catalog_pattern_project_mismatch")
	validateCatalogID(report, store, path, doc, state, "Pattern ID", "duplicate_catalog_id")
	validateCatalogSlug(report, store, path, doc, state, "Slug", "duplicate_catalog_slug", "catalog_pattern_slug_filename_mismatch")
	patternID := doc.field("Pattern ID")
	slug := doc.field("Slug")
	if patternID != "" {
		state.catalogPatterns[patternID] = doc.path
	}
	if slug != "" {
		state.catalogPatterns[slug] = doc.path
	}
	requireCatalogSection(report, doc, "Problem", "missing_catalog_pattern_problem", "catalog pattern must include a non-empty Problem section")
	requireCatalogSection(report, doc, "Maat Relevance", "missing_catalog_pattern_relevance", "catalog pattern must include a non-empty Maat Relevance section")
	observedIn := parseBulletList(doc.sections["observed in"])
	if len(observedIn) == 0 {
		addValidationIssue(report, store, path, doc.sectionLines["observed in"], "missing_catalog_pattern_observed_in", "catalog pattern must include at least one Observed In bullet")
	}
	for _, appSlug := range observedIn {
		if _, ok := state.catalogAppSlugs[appSlug]; !ok {
			addValidationIssue(report, store, path, doc.sectionLines["observed in"], "unknown_catalog_app", fmt.Sprintf("catalog pattern references missing app slug %q", appSlug))
		}
	}
	validateCatalogGoalLinks(report, store, path, doc, state, "Related Goals")
	validateCatalogTicketLinks(report, store, path, doc, state, "Related Tickets")
	return nil
}

func validateCatalogDecisions(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := catalogMarkdownFiles(store, projectKey, "decisions")
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := validateCatalogDecisionFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogDecisionFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Catalog Decision", "missing_catalog_decision_heading", "malformed_catalog_decision_heading", "missing_catalog_decision_title")
	requireObjectFields(report, doc, "missing_catalog_decision_field", "Decision ID", "Project", "State", "Pattern", "Date")
	validateCatalogProject(report, doc, state, "catalog_decision_project_mismatch")
	validateCatalogID(report, store, path, doc, state, "Decision ID", "duplicate_catalog_id")
	validateCatalogDateField(report, doc, "Date", "invalid_catalog_decision_date")
	if stateValue := doc.field("State"); stateValue != "" && !validCatalogDecisionStates[stateValue] {
		addValidationIssue(report, store, path, doc.fieldLine("State"), "invalid_catalog_decision_state", fmt.Sprintf("catalog decision state %q is not valid", stateValue))
	}
	validateCatalogPatternRef(report, store, path, doc, state, "Pattern")
	validateCatalogGoalField(report, store, path, doc, state, "Related Goal")
	validateCatalogTicketField(report, store, path, doc, state, "Related Ticket")
	requireCatalogSection(report, doc, "Rationale", "missing_catalog_decision_rationale", "catalog decision must include a non-empty Rationale section")
	if len(parseBulletList(doc.sections["evidence"])) == 0 {
		addValidationIssue(report, store, path, doc.sectionLines["evidence"], "missing_catalog_decision_evidence", "catalog decision must include at least one Evidence bullet")
	}
	return nil
}

func validateCatalogOpportunities(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	paths, err := catalogMarkdownFiles(store, projectKey, "opportunities")
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := validateCatalogOpportunityFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogOpportunityFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Catalog Opportunity", "missing_catalog_opportunity_heading", "malformed_catalog_opportunity_heading", "missing_catalog_opportunity_title")
	requireObjectFields(report, doc, "missing_catalog_opportunity_field", "Opportunity ID", "Project", "Status", "Source Pattern", "Area", "Effort", "Risk")
	validateCatalogProject(report, doc, state, "catalog_opportunity_project_mismatch")
	validateCatalogID(report, store, path, doc, state, "Opportunity ID", "duplicate_catalog_id")
	if status := doc.field("Status"); status != "" && !validCatalogOpportunityStatuses[status] {
		addValidationIssue(report, store, path, doc.fieldLine("Status"), "invalid_catalog_opportunity_status", fmt.Sprintf("catalog opportunity status %q is not valid", status))
	}
	validateCatalogPatternRef(report, store, path, doc, state, "Source Pattern")
	validateCatalogGoalField(report, store, path, doc, state, "Suggested Goal")
	validateCatalogTicketField(report, store, path, doc, state, "Suggested Ticket")
	requireCatalogSection(report, doc, "Description", "missing_catalog_opportunity_description", "catalog opportunity must include a non-empty Description section")
	return nil
}

func validateCatalogEvents(store, projectKey string, state *objectProjectValidation, report *ValidationReport) error {
	var paths []string
	eventsDir := contentPath(store, "projects", projectKey, "catalog", "events")
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
		if err := validateCatalogEventFile(store, path, state, report); err != nil {
			return err
		}
	}
	return nil
}

func validateCatalogEventFile(store, path string, state *objectProjectValidation, report *ValidationReport) error {
	report.Files++
	doc, err := scanMarkdownObjectForValidation(store, path)
	if err != nil {
		return err
	}
	report.Issues = append(report.Issues, doc.issues...)
	validateObjectHeading(report, doc, "Catalog Event", "missing_catalog_event_heading", "malformed_catalog_event_heading", "missing_catalog_event_type")
	requireObjectFields(report, doc, "missing_catalog_event_field", "Event ID", "Time", "Actor", "Project", "Type", "Object")
	validateCatalogProject(report, doc, state, "catalog_event_project_mismatch")
	eventID := doc.field("Event ID")
	if eventID != "" {
		if firstPath, ok := state.catalogEventIDs[eventID]; ok {
			addValidationIssue(report, store, path, doc.fieldLine("Event ID"), "duplicate_catalog_event_id", fmt.Sprintf("catalog event ID %q is already used by %s", eventID, firstPath))
		} else {
			state.catalogEventIDs[eventID] = doc.path
		}
		if filenameID(path) != eventID {
			addValidationIssue(report, store, path, doc.fieldLine("Event ID"), "catalog_event_id_filename_mismatch", fmt.Sprintf("catalog event ID %q does not match filename %q", eventID, filenameID(path)))
		}
	}
	if eventTime, ok := validateRFC3339Field(report, doc, "Time", "invalid_catalog_event_time"); ok {
		validateCatalogEventTimePath(report, store, path, doc, eventTime)
	}
	objectID := doc.field("Object")
	if objectID != "" {
		if _, ok := state.catalogIDs[objectID]; !ok {
			if _, ok := state.catalogSlugs[objectID]; !ok {
				addValidationIssue(report, store, path, doc.fieldLine("Object"), "unknown_catalog_event_object", fmt.Sprintf("catalog event references missing object %q", objectID))
			}
		}
	}
	requireCatalogSection(report, doc, "Summary", "missing_catalog_event_summary", "catalog event must include a non-empty Summary section")
	return nil
}

func validateCatalogProject(report *ValidationReport, doc validatedMarkdownObject, state *objectProjectValidation, code string) {
	if project := doc.field("Project"); project != "" && project != state.key {
		report.Issues = append(report.Issues, ValidationIssue{
			Path:    doc.path,
			Line:    doc.fieldLine("Project"),
			Code:    code,
			Message: fmt.Sprintf("catalog object project %q does not match directory %q", project, state.key),
		})
	}
}

func validateCatalogID(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, field, code string) {
	id := doc.field(field)
	if id == "" {
		return
	}
	if firstPath, ok := state.catalogIDs[id]; ok {
		addValidationIssue(report, store, path, doc.fieldLine(field), code, fmt.Sprintf("catalog object ID %q is already used by %s", id, firstPath))
		return
	}
	state.catalogIDs[id] = doc.path
}

func validateCatalogSlug(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, field, duplicateCode, filenameCode string) {
	slug := doc.field(field)
	if slug == "" {
		return
	}
	if firstPath, ok := state.catalogSlugs[slug]; ok {
		addValidationIssue(report, store, path, doc.fieldLine(field), duplicateCode, fmt.Sprintf("catalog slug %q is already used by %s", slug, firstPath))
	} else {
		state.catalogSlugs[slug] = doc.path
	}
	if filenameID(path) != slug {
		addValidationIssue(report, store, path, doc.fieldLine(field), filenameCode, fmt.Sprintf("catalog slug %q does not match filename %q", slug, filenameID(path)))
	}
}

func validateCatalogURLField(report *ValidationReport, doc validatedMarkdownObject, field, code string) {
	value := strings.TrimSpace(doc.field(field))
	if value == "" || strings.EqualFold(value, "unknown") {
		return
	}
	parsed, err := url.ParseRequestURI(value)
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		return
	}
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    doc.path,
		Line:    doc.fieldLine(field),
		Code:    code,
		Message: fmt.Sprintf("%s must be an absolute URL or unknown", field),
	})
}

func validateCatalogDateField(report *ValidationReport, doc validatedMarkdownObject, field, code string) {
	value := strings.TrimSpace(doc.field(field))
	if value == "" {
		return
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return
	}
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    doc.path,
		Line:    doc.fieldLine(field),
		Code:    code,
		Message: fmt.Sprintf("%s must be a YYYY-MM-DD date", field),
	})
}

func validateCatalogPatternRef(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, field string) {
	ref := strings.TrimSpace(doc.field(field))
	if ref == "" {
		return
	}
	if _, ok := state.catalogPatterns[ref]; ok {
		return
	}
	addValidationIssue(report, store, path, doc.fieldLine(field), "unknown_catalog_pattern", fmt.Sprintf("%s references missing pattern %q", field, ref))
}

func validateCatalogGoalField(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, field string) {
	goalID := optionalObjectLink(doc.field(field))
	if goalID == "" {
		return
	}
	if _, ok := state.goalIDs[goalID]; ok {
		return
	}
	addValidationIssue(report, store, path, doc.fieldLine(field), "unknown_catalog_goal", fmt.Sprintf("%s references missing goal %q", field, goalID))
}

func validateCatalogTicketField(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, field string) {
	ticketID := optionalObjectLink(doc.field(field))
	if ticketID == "" {
		return
	}
	if _, ok := state.ticketIDs[ticketID]; ok {
		return
	}
	addValidationIssue(report, store, path, doc.fieldLine(field), "unknown_catalog_ticket", fmt.Sprintf("%s references missing ticket %q", field, ticketID))
}

func validateCatalogGoalLinks(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, section string) {
	for _, goalID := range parseBulletList(doc.sections[strings.ToLower(section)]) {
		if optionalObjectLink(goalID) == "" {
			continue
		}
		if _, ok := state.goalIDs[goalID]; !ok {
			addValidationIssue(report, store, path, doc.sectionLines[strings.ToLower(section)], "unknown_catalog_goal", fmt.Sprintf("%s references missing goal %q", section, goalID))
		}
	}
}

func validateCatalogTicketLinks(report *ValidationReport, store, path string, doc validatedMarkdownObject, state *objectProjectValidation, section string) {
	for _, ticketID := range parseBulletList(doc.sections[strings.ToLower(section)]) {
		if optionalObjectLink(ticketID) == "" {
			continue
		}
		if _, ok := state.ticketIDs[ticketID]; !ok {
			addValidationIssue(report, store, path, doc.sectionLines[strings.ToLower(section)], "unknown_catalog_ticket", fmt.Sprintf("%s references missing ticket %q", section, ticketID))
		}
	}
}

func requireCatalogSection(report *ValidationReport, doc validatedMarkdownObject, section, code, message string) {
	key := strings.ToLower(section)
	if strings.TrimSpace(doc.sections[key]) != "" {
		return
	}
	report.Issues = append(report.Issues, ValidationIssue{
		Path:    doc.path,
		Line:    doc.sectionLines[key],
		Code:    code,
		Message: message,
	})
}

func validateCatalogEventTimePath(report *ValidationReport, store, path string, doc validatedMarkdownObject, eventTime time.Time) {
	rel := filepath.ToSlash(relPath(store, path))
	parts := strings.Split(rel, "/")
	if len(parts) < 7 || parts[0] != "projects" || parts[2] != "catalog" || parts[3] != "events" {
		return
	}
	wantYear := eventTime.Format("2006")
	wantMonth := eventTime.Format("01")
	if parts[4] != wantYear || parts[5] != wantMonth {
		addValidationIssue(report, store, path, doc.fieldLine("Time"), "catalog_event_time_path_mismatch", fmt.Sprintf("catalog event time belongs under catalog/events/%s/%s", wantYear, wantMonth))
	}
}
