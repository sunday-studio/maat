package maat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStoreDetectsLegacyProjectIssues(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProject(t, projectDir, "one.md", `# Project: One

| Field | Value |
|---|---|
| ID | shared |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |

## Goals

### G-001: Ship

| Field | Value |
|---|---|
| Status | active |

#### Tasks

- [ ] T-001: First task
`)
	writeProject(t, projectDir, "two.md", `# Project: Two

| Field | Value |
|---|---|
| ID | shared |
| Status | mystery |
| Updated | 2026-05-25 |

## Goals

### G-001: Ship

| Field | Value |
|---|---|
| Status | lost |

#### Tasks

- [ ] T-001: First task
- [x] T-001: Duplicate task

### G-001: Duplicate goal

| Field | Value |
|---|---|
| Status | active |
`)

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatalf("expected validation issues")
	}
	assertIssue(t, report, "duplicate_project_id")
	assertIssue(t, report, "invalid_project_status")
	assertIssue(t, report, "missing_project_field")
	assertIssue(t, report, "invalid_goal_status")
	assertIssue(t, report, "duplicate_goal_id")
	assertIssue(t, report, "duplicate_ticket_id")
}

func TestValidateStoreDetectsMalformedProjectFiles(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "projects")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeProject(t, projectDir, "broken.md", `# Wrong: Broken

| Field | Value |
|---|---|
| ID | broken |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |
| malformed

## Goals

### Broken goal heading

#### Tasks

- [ ] Missing ticket ID
`)

	report, err := ValidateStore(root)
	if err != nil {
		t.Fatal(err)
	}
	assertIssue(t, report, "malformed_project_heading")
	assertIssue(t, report, "missing_project_heading")
	assertIssue(t, report, "malformed_table_row")
	assertIssue(t, report, "malformed_goal_heading")
	assertIssue(t, report, "malformed_ticket")
}

func TestValidateStorePassesCurrentFixture(t *testing.T) {
	report, err := ValidateStore(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("expected current fixture to validate, got %#v", report.Issues)
	}
}

func writeProject(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertIssue(t *testing.T, report ValidationReport, code string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("expected issue %q in %#v", code, report.Issues)
}
