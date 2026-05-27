package maat

import (
	"path/filepath"
	"testing"
)

func TestLoadProjectCatalogParsesMarkdownObjects(t *testing.T) {
	root := t.TempDir()
	writeObjectFixture(t, root)
	writeCatalogFixture(t, root, "sample-a31f")

	project, err := LoadObjectProject(root, "sample-a31f")
	if err != nil {
		t.Fatal(err)
	}
	if project.Catalog == nil {
		t.Fatal("expected project catalog")
	}
	if len(project.Catalog.Apps) != 4 {
		t.Fatalf("expected 4 catalog apps, got %d", len(project.Catalog.Apps))
	}
	if project.Catalog.Apps[0].Slug != "btop" || project.Catalog.Apps[0].SourceURL == "" {
		t.Fatalf("unexpected first app: %#v", project.Catalog.Apps[0])
	}
	if len(project.Catalog.Patterns) != 4 {
		t.Fatalf("expected 4 catalog patterns, got %d", len(project.Catalog.Patterns))
	}
	if project.Catalog.Patterns[0].Slug != "background-refresh" {
		t.Fatalf("unexpected first pattern: %#v", project.Catalog.Patterns[0])
	}
	if len(project.Catalog.Decisions) != 1 || project.Catalog.Decisions[0].State != "adopt" {
		t.Fatalf("unexpected decisions: %#v", project.Catalog.Decisions)
	}
	if len(project.Catalog.Opportunities) != 1 || project.Catalog.Opportunities[0].Status != "ticketed" {
		t.Fatalf("unexpected opportunities: %#v", project.Catalog.Opportunities)
	}
	if len(project.Catalog.Events) != 1 || project.Catalog.Events[0].ObjectID != "lazygit" {
		t.Fatalf("unexpected catalog events: %#v", project.Catalog.Events)
	}
}

func TestValidateStorePassesCatalogFixture(t *testing.T) {
	root := t.TempDir()
	writeObjectFixture(t, root)
	writeCatalogFixture(t, root, "sample-a31f")

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("expected catalog fixture to validate, got %#v", report.Issues)
	}
}

func TestValidateStoreDetectsCatalogIssues(t *testing.T) {
	root := t.TempDir()
	writeObjectFixture(t, root)
	writeMinimalCatalogApp(t, root, "sample-a31f", "lazygit", "CA-lazygit")
	writeFile(t, filepath.Join(root, "projects", "sample-a31f", "catalog", "apps", "duplicate.md"), `# Catalog App: Duplicate

| Field | Value |
|---|---|
| App ID | CA-duplicate |
| Project | sample-a31f |
| Slug | lazygit |
| Name | Duplicate |
| Summary | Duplicate slug for validation coverage. |
| Source URL | not-a-url |
| Category | git |
| Last Reviewed | today |
`)
	writeFile(t, filepath.Join(root, "projects", "sample-a31f", "catalog", "patterns", "bad-pattern.md"), `# Catalog Pattern: Bad Pattern

| Field | Value |
|---|---|
| Pattern ID | CP-bad |
| Project | sample-a31f |
| Slug | bad-pattern |
| Title | Bad pattern |
| Category | navigation |

## Problem

References objects that do not exist.

## Observed In

- missing-app

## Maat Relevance

This should fail validation.

## Related Tickets

- T-missing
`)
	writeFile(t, filepath.Join(root, "projects", "sample-a31f", "catalog", "decisions", "bad-decision.md"), `# Catalog Decision: Bad Decision

| Field | Value |
|---|---|
| Decision ID | CD-bad |
| Project | sample-a31f |
| State | maybe |
| Pattern | missing-pattern |
| Date | 2026-05-27 |

## Rationale

This should fail validation.

## Evidence

- Validation fixture.
`)

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected catalog validation issues")
	}
	for _, code := range []string{
		"duplicate_catalog_slug",
		"catalog_app_slug_filename_mismatch",
		"invalid_catalog_source_url",
		"invalid_catalog_review_date",
		"unknown_catalog_app",
		"unknown_catalog_ticket",
		"invalid_catalog_decision_state",
		"unknown_catalog_pattern",
	} {
		assertIssue(t, report, code)
	}
}

func writeCatalogFixture(t *testing.T, root, projectKey string) {
	t.Helper()
	writeMinimalCatalogApp(t, root, projectKey, "lazygit", "CA-20260527-lazygit")
	writeMinimalCatalogApp(t, root, projectKey, "btop", "CA-20260527-btop")
	writeMinimalCatalogApp(t, root, projectKey, "gh-dash", "CA-20260527-gh-dash")
	writeMinimalCatalogApp(t, root, projectKey, "superfile", "CA-20260527-superfile")
	writeCatalogPattern(t, root, projectKey, "focused-detail-pane", "Focused detail pane", "inspection/detail panes", []string{"lazygit", "gh-dash", "superfile"})
	writeCatalogPattern(t, root, projectKey, "keyboard-model", "Keyboard model", "keyboard model", []string{"lazygit", "btop", "gh-dash", "superfile"})
	writeCatalogPattern(t, root, projectKey, "background-refresh", "Background refresh", "background refresh", []string{"btop", "gh-dash"})
	writeCatalogPattern(t, root, projectKey, "empty-states", "Empty states", "error and empty states", []string{"lazygit", "superfile"})
	writeFile(t, filepath.Join(root, "projects", projectKey, "catalog", "decisions", "CD-20260527-focused-detail-pane.md"), `# Catalog Decision: Adopt Focused Detail Pane

| Field | Value |
|---|---|
| Decision ID | CD-20260527-focused-detail-pane |
| Project | sample-a31f |
| State | adopt |
| Pattern | focused-detail-pane |
| Date | 2026-05-27 |
| Related Goal | G-20260525-190533-a7f3 |
| Related Ticket | T-20260525-190700-b91c |

## Rationale

Focused detail keeps reading close to navigation.

## Evidence

- Pattern appears in seeded catalog apps.
`)
	writeFile(t, filepath.Join(root, "projects", projectKey, "catalog", "opportunities", "CO-20260527-project-board-detail-flow.md"), `# Catalog Opportunity: Project Board Detail Flow

| Field | Value |
|---|---|
| Opportunity ID | CO-20260527-project-board-detail-flow |
| Project | sample-a31f |
| Status | ticketed |
| Source Pattern | focused-detail-pane |
| Area | tui |
| Effort | medium |
| Risk | low |
| Suggested Goal | G-20260525-190533-a7f3 |
| Suggested Ticket | T-20260525-190700-b91c |

## Description

Make project list, board navigation, and item detail feel like one terminal workflow.
`)
	writeFile(t, filepath.Join(root, "projects", projectKey, "catalog", "events", "2026", "05", "CE-20260527-105500-codex-a1b2.md"), `# Catalog Event: catalog.app.reviewed

| Field | Value |
|---|---|
| Event ID | CE-20260527-105500-codex-a1b2 |
| Time | 2026-05-27T10:55:00+02:00 |
| Actor | codex |
| Project | sample-a31f |
| Type | catalog.app.reviewed |
| Object | lazygit |

## Summary

Reviewed lazygit as a terminal app catalog seed.

## Evidence

- Seed object validates from Markdown.
`)
}

func writeMinimalCatalogApp(t *testing.T, root, projectKey, slug, id string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "projects", projectKey, "catalog", "apps", slug+".md"), "# Catalog App: "+slug+`

| Field | Value |
|---|---|
| App ID | `+id+` |
| Project | `+projectKey+` |
| Slug | `+slug+` |
| Name | `+slug+` |
| Summary | Terminal app catalog observation for `+slug+`. |
| Source URL | https://github.com/example/`+slug+` |
| Website URL | unknown |
| Stars | unknown |
| Language | unknown |
| License | unknown |
| Category | terminal |
| Last Reviewed | 2026-05-27 |
| Tags | #terminal-app #catalog |

## Screens

- unknown

## Notes

Metadata that has not been verified stays unknown.
`)
}

func writeCatalogPattern(t *testing.T, root, projectKey, slug, title, category string, apps []string) {
	t.Helper()
	observed := ""
	for _, app := range apps {
		observed += "- " + app + "\n"
	}
	writeFile(t, filepath.Join(root, "projects", projectKey, "catalog", "patterns", slug+".md"), `# Catalog Pattern: `+title+`

| Field | Value |
|---|---|
| Pattern ID | CP-20260527-`+slug+` |
| Project | `+projectKey+` |
| Slug | `+slug+` |
| Title | `+title+` |
| Category | `+category+` |
| Tags | #tui #catalog |

## Problem

Terminal users need this pattern to keep navigation and context understandable.

## Observed In

`+observed+`
## Maat Relevance

Maat should apply this pattern where it improves project and ticket reading.

## Implementation Notes

Keep the behavior keyboard-first and readable without color.

## Related Goals

- G-20260525-190533-a7f3

## Related Tickets

- T-20260525-190700-b91c
`)
}
