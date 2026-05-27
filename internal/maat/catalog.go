package maat

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Catalog struct {
	ProjectKey    string               `json:"project_key,omitempty"`
	Apps          []CatalogApp         `json:"apps"`
	Patterns      []CatalogPattern     `json:"patterns"`
	Decisions     []CatalogDecision    `json:"decisions"`
	Opportunities []CatalogOpportunity `json:"opportunities"`
	Events        []CatalogEvent       `json:"events,omitempty"`
}

type CatalogApp struct {
	Kind           string   `json:"kind"`
	ID             string   `json:"id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Summary        string   `json:"summary,omitempty"`
	SourceURL      string   `json:"source_url,omitempty"`
	WebsiteURL     string   `json:"website_url,omitempty"`
	Stars          string   `json:"stars,omitempty"`
	Language       string   `json:"language,omitempty"`
	License        string   `json:"license,omitempty"`
	Category       string   `json:"category,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	LastReviewed   string   `json:"last_reviewed,omitempty"`
	Patterns       []string `json:"patterns,omitempty"`
	RelatedGoals   []string `json:"related_goals,omitempty"`
	RelatedTickets []string `json:"related_tickets,omitempty"`
	Notes          string   `json:"notes,omitempty"`
	Path           string   `json:"path"`
}

type CatalogPattern struct {
	Kind                string   `json:"kind"`
	ID                  string   `json:"id"`
	Slug                string   `json:"slug"`
	Title               string   `json:"title"`
	Category            string   `json:"category,omitempty"`
	Problem             string   `json:"problem,omitempty"`
	ObservedIn          []string `json:"observed_in,omitempty"`
	MaatUse             string   `json:"maat_use,omitempty"`
	ImplementationNotes string   `json:"implementation_notes,omitempty"`
	RelatedGoals        []string `json:"related_goals,omitempty"`
	RelatedTickets      []string `json:"related_tickets,omitempty"`
	Path                string   `json:"path"`
}

type CatalogDecision struct {
	Kind           string   `json:"kind"`
	ID             string   `json:"id"`
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	State          string   `json:"state,omitempty"`
	Decision       string   `json:"decision,omitempty"`
	Rationale      string   `json:"rationale,omitempty"`
	Pattern        string   `json:"pattern,omitempty"`
	SourceApp      string   `json:"source_app,omitempty"`
	RelatedGoals   []string `json:"related_goals,omitempty"`
	RelatedTickets []string `json:"related_tickets,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	Date           string   `json:"date,omitempty"`
	Path           string   `json:"path"`
}

type CatalogOpportunity struct {
	Kind            string `json:"kind"`
	ID              string `json:"id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	SourcePattern   string `json:"source_pattern,omitempty"`
	Area            string `json:"area,omitempty"`
	Effort          string `json:"effort,omitempty"`
	Risk            string `json:"risk,omitempty"`
	SuggestedGoal   string `json:"suggested_goal,omitempty"`
	SuggestedTicket string `json:"suggested_ticket,omitempty"`
	Status          string `json:"status,omitempty"`
	Summary         string `json:"summary,omitempty"`
	Path            string `json:"path"`
}

type CatalogEvent struct {
	Kind       string   `json:"kind"`
	ID         string   `json:"id"`
	Time       string   `json:"time"`
	Actor      string   `json:"actor"`
	ProjectKey string   `json:"project_key"`
	Type       string   `json:"type"`
	ObjectID   string   `json:"object_id"`
	Summary    string   `json:"summary,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	Path       string   `json:"path"`
}

func (catalog Catalog) Empty() bool {
	return len(catalog.Apps) == 0 &&
		len(catalog.Patterns) == 0 &&
		len(catalog.Decisions) == 0 &&
		len(catalog.Opportunities) == 0 &&
		len(catalog.Events) == 0
}

func LoadCatalog(store, projectKey string) (Catalog, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey != "" {
		return LoadProjectCatalog(store, projectKey)
	}
	projects, err := LoadObjectProjects(store)
	if err != nil {
		return Catalog{}, err
	}
	var catalog Catalog
	for _, project := range projects {
		projectCatalog, err := LoadProjectCatalog(store, project.Key)
		if err != nil {
			return Catalog{}, err
		}
		catalog.Apps = append(catalog.Apps, projectCatalog.Apps...)
		catalog.Patterns = append(catalog.Patterns, projectCatalog.Patterns...)
		catalog.Decisions = append(catalog.Decisions, projectCatalog.Decisions...)
		catalog.Opportunities = append(catalog.Opportunities, projectCatalog.Opportunities...)
		catalog.Events = append(catalog.Events, projectCatalog.Events...)
	}
	sortCatalog(&catalog)
	return catalog, nil
}

func LoadProjectCatalog(store, projectKey string) (Catalog, error) {
	catalog := Catalog{ProjectKey: projectKey}
	apps, err := loadCatalogApps(store, projectKey)
	if err != nil {
		return Catalog{}, err
	}
	patterns, err := loadCatalogPatterns(store, projectKey)
	if err != nil {
		return Catalog{}, err
	}
	decisions, err := loadCatalogDecisions(store, projectKey)
	if err != nil {
		return Catalog{}, err
	}
	opportunities, err := loadCatalogOpportunities(store, projectKey)
	if err != nil {
		return Catalog{}, err
	}
	events, err := loadCatalogEvents(store, projectKey)
	if err != nil {
		return Catalog{}, err
	}
	catalog.Apps = apps
	catalog.Patterns = patterns
	catalog.Decisions = decisions
	catalog.Opportunities = opportunities
	catalog.Events = events
	sortCatalog(&catalog)
	return catalog, nil
}

func FindCatalogObject(store, projectKey, idOrSlug string) (any, error) {
	idOrSlug = strings.TrimSpace(idOrSlug)
	if idOrSlug == "" {
		return nil, fmt.Errorf("catalog object id or slug is required")
	}
	catalog, err := LoadCatalog(store, projectKey)
	if err != nil {
		return nil, err
	}
	var matches []any
	for _, app := range catalog.Apps {
		if catalogIDMatches(app.ID, app.Slug, idOrSlug) {
			matches = append(matches, app)
		}
	}
	for _, pattern := range catalog.Patterns {
		if catalogIDMatches(pattern.ID, pattern.Slug, idOrSlug) {
			matches = append(matches, pattern)
		}
	}
	for _, decision := range catalog.Decisions {
		if catalogIDMatches(decision.ID, decision.Slug, idOrSlug) {
			matches = append(matches, decision)
		}
	}
	for _, opportunity := range catalog.Opportunities {
		if catalogIDMatches(opportunity.ID, opportunity.Slug, idOrSlug) {
			matches = append(matches, opportunity)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("catalog object %q not found", idOrSlug)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("catalog object %q matches multiple objects; pass --project <project-key> or a more specific id", idOrSlug)
	}
}

func loadCatalogApps(store, projectKey string) ([]CatalogApp, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "catalog", "apps", "*.md"))
	if err != nil {
		return nil, err
	}
	apps := make([]CatalogApp, 0, len(paths))
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		doc, err := parseMarkdownObject(path)
		if err != nil {
			return nil, err
		}
		slug := catalogField(doc, "Slug")
		if slug == "" {
			slug = filenameID(path)
		}
		id := firstNonEmpty(catalogField(doc, "App ID"), catalogField(doc, "ID"), slug)
		name := firstNonEmpty(catalogField(doc, "Name"), trimCatalogTitle(doc.Title, "Catalog App", "App"), slug)
		apps = append(apps, CatalogApp{
			Kind:           "app",
			ID:             id,
			Slug:           slug,
			Name:           name,
			Summary:        firstNonEmpty(catalogField(doc, "Summary"), strings.TrimSpace(doc.Sections["summary"])),
			SourceURL:      catalogField(doc, "Source URL", "Source"),
			WebsiteURL:     catalogField(doc, "Website URL", "Website"),
			Stars:          catalogField(doc, "Stars"),
			Language:       catalogField(doc, "Language"),
			License:        catalogField(doc, "License"),
			Category:       catalogField(doc, "Category"),
			Tags:           splitCatalogList(catalogField(doc, "Tags")),
			LastReviewed:   catalogField(doc, "Last Reviewed", "Reviewed"),
			Patterns:       splitCatalogList(firstNonEmpty(catalogField(doc, "Patterns"), doc.Sections["ux patterns"])),
			RelatedGoals:   splitCatalogList(firstNonEmpty(catalogField(doc, "Related Goals", "Related Goal", "Goals"), doc.Sections["related goals"])),
			RelatedTickets: splitCatalogList(firstNonEmpty(catalogField(doc, "Related Tickets", "Related Ticket", "Tickets"), doc.Sections["related tickets"])),
			Notes:          strings.TrimSpace(firstNonEmpty(doc.Sections["notes"], doc.Sections["what maat should learn"], doc.Sections["screens and interaction notes"])),
			Path:           relPath(store, path),
		})
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Slug < apps[j].Slug
	})
	return apps, nil
}

func loadCatalogPatterns(store, projectKey string) ([]CatalogPattern, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "catalog", "patterns", "*.md"))
	if err != nil {
		return nil, err
	}
	patterns := make([]CatalogPattern, 0, len(paths))
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		doc, err := parseMarkdownObject(path)
		if err != nil {
			return nil, err
		}
		slug := catalogSlug(path, doc)
		id := firstNonEmpty(catalogField(doc, "Pattern ID"), catalogField(doc, "ID"), slug)
		title := firstNonEmpty(catalogField(doc, "Title"), trimCatalogTitle(doc.Title, "Catalog Pattern", "Pattern"), slug)
		patterns = append(patterns, CatalogPattern{
			Kind:                "pattern",
			ID:                  id,
			Slug:                slug,
			Title:               title,
			Category:            catalogField(doc, "Category"),
			Problem:             firstNonEmpty(strings.TrimSpace(doc.Sections["problem"]), catalogField(doc, "Problem")),
			ObservedIn:          splitCatalogList(firstNonEmpty(catalogField(doc, "Observed In"), doc.Sections["observed in"])),
			MaatUse:             firstNonEmpty(strings.TrimSpace(doc.Sections["maat relevance"]), strings.TrimSpace(doc.Sections["maat use"]), strings.TrimSpace(doc.Sections["why it matters for maat"]), catalogField(doc, "Maat Use")),
			ImplementationNotes: strings.TrimSpace(firstNonEmpty(doc.Sections["implementation notes"], doc.Sections["notes"])),
			RelatedGoals:        splitCatalogList(firstNonEmpty(catalogField(doc, "Related Goals", "Related Goal", "Goals"), doc.Sections["related goals"])),
			RelatedTickets:      splitCatalogList(firstNonEmpty(catalogField(doc, "Related Tickets", "Related Ticket", "Tickets"), doc.Sections["related tickets"])),
			Path:                relPath(store, path),
		})
	}
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Slug < patterns[j].Slug
	})
	return patterns, nil
}

func loadCatalogDecisions(store, projectKey string) ([]CatalogDecision, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "catalog", "decisions", "*.md"))
	if err != nil {
		return nil, err
	}
	decisions := make([]CatalogDecision, 0, len(paths))
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		doc, err := parseMarkdownObject(path)
		if err != nil {
			return nil, err
		}
		slug := catalogSlug(path, doc)
		id := firstNonEmpty(catalogField(doc, "Decision ID"), catalogField(doc, "ID"), slug)
		state := catalogField(doc, "Decision", "State")
		decisions = append(decisions, CatalogDecision{
			Kind:           "decision",
			ID:             id,
			Slug:           slug,
			Title:          firstNonEmpty(catalogField(doc, "Title"), trimCatalogTitle(doc.Title, "Catalog Decision", "Decision"), slug),
			State:          state,
			Decision:       state,
			Rationale:      firstNonEmpty(strings.TrimSpace(doc.Sections["rationale"]), catalogField(doc, "Rationale")),
			Pattern:        catalogField(doc, "Pattern", "Linked Pattern"),
			SourceApp:      catalogField(doc, "Source App", "App"),
			RelatedGoals:   splitCatalogList(firstNonEmpty(catalogField(doc, "Related Goals", "Related Goal", "Goals"), doc.Sections["related goals"])),
			RelatedTickets: splitCatalogList(firstNonEmpty(catalogField(doc, "Related Tickets", "Related Ticket", "Tickets"), doc.Sections["related tickets"])),
			Evidence:       parseBulletList(firstNonEmpty(doc.Sections["evidence"], doc.Sections["notes"])),
			Date:           catalogField(doc, "Date"),
			Path:           relPath(store, path),
		})
	}
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].ID < decisions[j].ID
	})
	return decisions, nil
}

func loadCatalogOpportunities(store, projectKey string) ([]CatalogOpportunity, error) {
	paths, err := filepath.Glob(contentPath(store, "projects", projectKey, "catalog", "opportunities", "*.md"))
	if err != nil {
		return nil, err
	}
	opportunities := make([]CatalogOpportunity, 0, len(paths))
	for _, path := range paths {
		if isSkippedMarkdownFile(path) {
			continue
		}
		doc, err := parseMarkdownObject(path)
		if err != nil {
			return nil, err
		}
		slug := catalogSlug(path, doc)
		id := firstNonEmpty(catalogField(doc, "Opportunity ID"), catalogField(doc, "ID"), slug)
		opportunities = append(opportunities, CatalogOpportunity{
			Kind:            "opportunity",
			ID:              id,
			Slug:            slug,
			Title:           firstNonEmpty(catalogField(doc, "Title"), trimCatalogTitle(doc.Title, "Catalog Opportunity", "Opportunity"), slug),
			SourcePattern:   catalogField(doc, "Source Pattern", "Pattern"),
			Area:            catalogField(doc, "Area"),
			Effort:          catalogField(doc, "Effort"),
			Risk:            catalogField(doc, "Risk"),
			SuggestedGoal:   catalogField(doc, "Suggested Goal", "Goal"),
			SuggestedTicket: catalogField(doc, "Suggested Ticket", "Ticket"),
			Status:          catalogField(doc, "Status"),
			Summary:         strings.TrimSpace(firstNonEmpty(doc.Sections["description"], doc.Sections["summary"])),
			Path:            relPath(store, path),
		})
	}
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].ID < opportunities[j].ID
	})
	return opportunities, nil
}

func loadCatalogEvents(store, projectKey string) ([]CatalogEvent, error) {
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
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	events := make([]CatalogEvent, 0, len(paths))
	for _, path := range paths {
		doc, err := parseMarkdownObject(path)
		if err != nil {
			return nil, err
		}
		events = append(events, CatalogEvent{
			Kind:       "event",
			ID:         catalogField(doc, "Event ID", "ID"),
			Time:       catalogField(doc, "Time"),
			Actor:      catalogField(doc, "Actor"),
			ProjectKey: catalogField(doc, "Project"),
			Type:       catalogField(doc, "Type"),
			ObjectID:   catalogField(doc, "Object"),
			Summary:    strings.TrimSpace(doc.Sections["summary"]),
			Evidence:   parseBulletList(doc.Sections["evidence"]),
			Path:       relPath(store, path),
		})
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			return events[i].ID > events[j].ID
		}
		return events[i].Time > events[j].Time
	})
	return events, nil
}

func sortCatalog(catalog *Catalog) {
	sort.Slice(catalog.Apps, func(i, j int) bool {
		return catalog.Apps[i].Slug < catalog.Apps[j].Slug
	})
	sort.Slice(catalog.Patterns, func(i, j int) bool {
		return catalog.Patterns[i].Slug < catalog.Patterns[j].Slug
	})
	sort.Slice(catalog.Decisions, func(i, j int) bool {
		return catalog.Decisions[i].ID < catalog.Decisions[j].ID
	})
	sort.Slice(catalog.Opportunities, func(i, j int) bool {
		return catalog.Opportunities[i].ID < catalog.Opportunities[j].ID
	})
	sort.Slice(catalog.Events, func(i, j int) bool {
		if catalog.Events[i].Time == catalog.Events[j].Time {
			return catalog.Events[i].ID > catalog.Events[j].ID
		}
		return catalog.Events[i].Time > catalog.Events[j].Time
	})
}

func catalogSlug(path string, doc markdownObject) string {
	if slug := catalogField(doc, "Slug"); slug != "" {
		return slug
	}
	return filenameID(path)
}

func catalogIDMatches(id, slug, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	return strings.ToLower(id) == query || strings.ToLower(slug) == query
}

func catalogField(doc markdownObject, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(doc.Fields[strings.ToLower(name)]); value != "" {
			return value
		}
	}
	return ""
}

func trimCatalogTitle(title string, prefixes ...string) string {
	title = strings.TrimSpace(title)
	for _, prefix := range prefixes {
		label := prefix + ":"
		if strings.HasPrefix(title, label) {
			return strings.TrimSpace(strings.TrimPrefix(title, label))
		}
	}
	return title
}

func splitCatalogList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "- ") || strings.Contains(value, "\n") {
		if bullets := parseBulletList(value); len(bullets) > 0 {
			return uniqueCatalogItems(bullets)
		}
	}
	var parts []string
	if strings.Contains(value, "\n") || strings.Contains(value, ",") || strings.Contains(value, ";") {
		value = strings.ReplaceAll(value, "\n", ",")
		value = strings.ReplaceAll(value, ";", ",")
		parts = strings.Split(value, ",")
	} else {
		parts = strings.Fields(value)
	}
	return uniqueCatalogItems(parts)
}

func uniqueCatalogItems(parts []string) []string {
	items := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(strings.TrimPrefix(part, "- "))
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		items = append(items, part)
	}
	return items
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
