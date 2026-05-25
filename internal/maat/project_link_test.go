package maat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkProjectCreatesProjectFromGitRemote(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	initLinkGitRepo(t, source)
	runLinkGit(t, source, "remote", "add", "origin", "git@github.com:sunday-studio/orion.git")

	linked, err := LinkProject(context.Background(), LinkProjectInput{
		Store:      store,
		SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !linked.Created || linked.ProjectKey != "orion" || linked.DisplayName != "Orion" {
		t.Fatalf("unexpected link result: %#v", linked)
	}
	if linked.Project.Identity["Remote"] != "git@github.com:sunday-studio/orion.git" {
		t.Fatalf("expected remote identity, got %#v", linked.Project.Identity)
	}
	if linked.Project.Identity["Primary Repo"] != source {
		t.Fatalf("expected source path identity, got %#v", linked.Project.Identity)
	}
}

func TestLinkProjectIsIdempotent(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()

	first, err := LinkProject(context.Background(), LinkProjectInput{
		Store:      store,
		SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := LinkProject(context.Background(), LinkProjectInput{
		Store:      store,
		SourcePath: source,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created || !second.Existing || second.ProjectKey != first.ProjectKey {
		t.Fatalf("unexpected idempotent result: first=%#v second=%#v", first, second)
	}
}

func TestLinkProjectAllowsExplicitKeyAndName(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()

	linked, err := LinkProject(context.Background(), LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "photo-system",
		DisplayName: "Photo System",
	})
	if err != nil {
		t.Fatal(err)
	}
	if linked.ProjectKey != "photo-system" || linked.DisplayName != "Photo System" {
		t.Fatalf("unexpected explicit link result: %#v", linked)
	}
}

func initLinkGitRepo(t *testing.T, dir string) {
	t.Helper()
	runLinkGit(t, dir, "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runLinkGit(t, dir, "config", "user.email", "maat@example.test")
	runLinkGit(t, dir, "config", "user.name", "Maat Test")
	runLinkGit(t, dir, "add", ".")
	runLinkGit(t, dir, "commit", "-m", "test: seed")
}

func runLinkGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
