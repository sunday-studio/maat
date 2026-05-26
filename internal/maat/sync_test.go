package maat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSyncStoreValidatesIndexesCommitsAndPushes(t *testing.T) {
	root := newSyncStore(t)
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "true\n"}},
		{result: GitCommandResult{Stdout: "main\n"}},
		{result: GitCommandResult{Stdout: "git@github.com:sunday-studio/maat-state.git\n"}},
		{result: GitCommandResult{Stdout: " M projects/maat.md\n?? .maat/index.json\n?? .maat/index.sqlite\n"}},
		{},
		{},
		{},
		{result: GitCommandResult{}},
	}}

	result, err := SyncStore(context.Background(), StoreSyncOptions{
		Store:   root,
		Runner:  runner,
		Message: "status(maat): sync state",
		Push:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Validation.OK() {
		t.Fatalf("expected validation to pass, got %#v", result.Validation.Issues)
	}
	if result.JSONIndexPath != filepath.Join(root, ".maat", "index.json") {
		t.Fatalf("unexpected json index path: %q", result.JSONIndexPath)
	}
	if result.SQLiteIndex.Path != filepath.Join(root, ".maat", "index.sqlite") || result.SQLiteIndex.Documents == 0 {
		t.Fatalf("unexpected sqlite index info: %#v", result.SQLiteIndex)
	}
	if !result.Committed || !result.Pushed {
		t.Fatalf("expected commit and push, got committed=%v pushed=%v", result.Committed, result.Pushed)
	}
	if !reflect.DeepEqual(result.CommitPathspecs, []string{"."}) {
		t.Fatalf("expected all-change pathspec, got %#v", result.CommitPathspecs)
	}
	if len(result.DirtyBeforeCommit) != 3 {
		t.Fatalf("expected dirty status before commit, got %#v", result.DirtyBeforeCommit)
	}
	assertGitCalls(t, runner.calls, [][]string{
		{"rev-parse", "--is-inside-work-tree"},
		{"branch", "--show-current"},
		{"remote", "get-url", "origin"},
		{"status", "--porcelain=v1"},
		{"add", "--", "."},
		{"commit", "-m", "status(maat): sync state"},
		{"push", "origin", "main"},
		{"status", "--porcelain=v1"},
	})
}

func TestSyncStoreCommitsSelectedPathspecs(t *testing.T) {
	root := newSyncStore(t)
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "true\n"}},
		{result: GitCommandResult{Stdout: "main\n"}},
		{result: GitCommandResult{}},
		{result: GitCommandResult{Stdout: " M projects/maat.md\n M docs/work-plan.md\n"}},
		{},
		{},
		{result: GitCommandResult{Stdout: " M docs/work-plan.md\n"}},
	}}

	result, err := SyncStore(context.Background(), StoreSyncOptions{
		Store:     root,
		Runner:    runner,
		Message:   "status(maat): record update",
		Pathspecs: []string{"projects/maat.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Committed {
		t.Fatal("expected selected commit")
	}
	if !reflect.DeepEqual(result.CommitPathspecs, []string{"projects/maat.md"}) {
		t.Fatalf("unexpected pathspecs: %#v", result.CommitPathspecs)
	}
	assertGitCalls(t, runner.calls, [][]string{
		{"rev-parse", "--is-inside-work-tree"},
		{"branch", "--show-current"},
		{"remote", "get-url", "origin"},
		{"status", "--porcelain=v1"},
		{"add", "--", "projects/maat.md"},
		{"commit", "-m", "status(maat): record update"},
		{"status", "--porcelain=v1"},
	})
}

func TestSyncStoreValidationFailureStopsBeforeIndexAndCommit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	brokenDir := filepath.Join(root, "projects", "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSyncProject(t, root, filepath.Join("broken", "project.md"), `# Project: Broken

| Field | Value |
|---|---|
| Project Key | wrong |
| Display Name | Broken |
| Status | mystery |
| Created | yesterday |
| Updated | 2026-05-25T19:05:00+02:00 |
`)
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "true\n"}},
		{result: GitCommandResult{Stdout: "main\n"}},
		{result: GitCommandResult{}},
	}}

	_, err := SyncStore(context.Background(), StoreSyncOptions{
		Store:   root,
		Runner:  runner,
		Message: "status(maat): should not commit",
	})
	var validationErr ValidationFailedError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if validationErr.Report.OK() {
		t.Fatal("expected validation issues")
	}
	assertGitCalls(t, runner.calls, [][]string{
		{"rev-parse", "--is-inside-work-tree"},
		{"branch", "--show-current"},
		{"remote", "get-url", "origin"},
	})
	if _, statErr := os.Stat(filepath.Join(root, ".maat")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected validation failure to skip indexing, got stat err %v", statErr)
	}
}

func TestSyncStoreWithoutMessageOnlyReportsDirtyState(t *testing.T) {
	root := newSyncStore(t)
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "true\n"}},
		{result: GitCommandResult{Stdout: "main\n"}},
		{result: GitCommandResult{}},
		{result: GitCommandResult{Stdout: " M projects/maat.md\n"}},
		{result: GitCommandResult{Stdout: " M projects/maat.md\n"}},
	}}

	result, err := SyncStore(context.Background(), StoreSyncOptions{
		Store:  root,
		Runner: runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Committed {
		t.Fatal("did not expect commit without message")
	}
	assertGitCalls(t, runner.calls, [][]string{
		{"rev-parse", "--is-inside-work-tree"},
		{"branch", "--show-current"},
		{"remote", "get-url", "origin"},
		{"status", "--porcelain=v1"},
		{"status", "--porcelain=v1"},
	})
}

func TestSyncStoreRejectsNonGitStorage(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "false\n"}},
	}}

	_, err := SyncStore(context.Background(), StoreSyncOptions{
		Store:  t.TempDir(),
		Runner: runner,
	})
	if err == nil || err.Error() != "storage is not a git repository" {
		t.Fatalf("expected non-repository error, got %v", err)
	}
}

func newSyncStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeSyncProject(t, root, "maat.md", `# Project: Maat

| Field | Value |
|---|---|
| ID | maat |
| Status | active |
| Owner | agents |
| Updated | 2026-05-25 |

## Current

Sync internals are being built.

## Goals

### G-001: Ship sync

| Field | Value |
|---|---|
| Status | active |

#### Tasks

- [ ] T-001: Wire orchestration
`)
	return root
}

func writeSyncProject(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, "projects", name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
