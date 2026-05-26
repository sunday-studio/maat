package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunday-studio/maat/internal/maat"
	"github.com/sunday-studio/maat/internal/version"
)

func TestStatusJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var summary maat.StatusSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Projects != 1 || summary.Goals != 1 || summary.Tickets != 2 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestStatusJSONIncludesObjectProjects(t *testing.T) {
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)
	writer := maat.NewWriteStore(store)
	if _, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		GoalID:      goalID,
		Title:       "Object status ticket",
		Description: "Exercise object status ticket counting.",
		Acceptance:  []string{"Status output includes the object ticket."},
		Actor:       "test",
	}); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("status", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var summary maat.StatusSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Projects != 1 || summary.Goals != 1 || summary.ActiveGoals != 1 || summary.Tickets != 1 || summary.OpenTickets != 1 {
		t.Fatalf("unexpected object summary: %#v", summary)
	}
}

func TestStatusAgentUseOutputsJSONUpdates(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store, "--agent-use")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, "Projects:") {
		t.Fatalf("agent output should not include human prose: %q", output)
	}
	var update map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &update); err != nil {
		t.Fatal(err)
	}
	if update["type"] != "maat.update" || update["step"] != "status.ready" || update["status"] != "ok" {
		t.Fatalf("unexpected update: %#v", update)
	}
}

func TestAgentUseRejectsJSONFlag(t *testing.T) {
	if _, err := captureRun("status", "--agent-use", "--json"); err == nil {
		t.Fatal("expected --agent-use with --json to fail")
	}
}

func TestHumanOutputCanUseColor(t *testing.T) {
	t.Setenv("MAAT_COLOR", "always")
	store := writeCommandFixture(t)

	output, err := captureRun("status", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "\x1b[") || !strings.Contains(output, "status ready") {
		t.Fatalf("expected colored human output, got %q", output)
	}
}

func TestProjectsJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("projects", "--json", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}

	var projects []projectListItem
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != "sample" || projects[0].Title != "Sample" || projects[0].Layout != "object" {
		t.Fatalf("unexpected projects: %#v", projects)
	}
}

func TestProjectsJSONIncludesObjectProjects(t *testing.T) {
	store := writeObjectCommandFixture(t)

	output, err := captureRun("projects", "--json", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}

	var projects []projectListItem
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].ID != "sample" || projects[0].Title != "Sample" || projects[0].Layout != "object" {
		t.Fatalf("unexpected object projects: %#v", projects)
	}
}

func TestVersionCommand(t *testing.T) {
	output, err := captureRun("version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "maat dev") {
		t.Fatalf("unexpected version output: %q", output)
	}

	output, err = captureRun("version", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatal(err)
	}
	if got["version"] != "dev" || got["commit"] == "" || got["date"] == "" {
		t.Fatalf("unexpected version json: %#v", got)
	}
}

func TestUpdateCommandInstallsSourceBinary(t *testing.T) {
	source := filepath.Join(t.TempDir(), "maat-new")
	if err := os.WriteFile(source, []byte("new binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	installDir := t.TempDir()

	output, err := captureRun("update", "--source", source, "--install-dir", installDir, "--binary-name", "maat-test", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result installCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(installDir, "maat-test")
	if result.Action != "update.installed" || result.SourcePath != source || result.TargetPath != target {
		t.Fatalf("unexpected update result: %#v", result)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new binary\n" {
		t.Fatalf("unexpected installed binary content: %q", string(data))
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected installed binary to be executable, mode=%v", info.Mode())
	}
}

func TestUpdateCommandDownloadsLatestGitHubRelease(t *testing.T) {
	installDir := t.TempDir()
	assetName := fmt.Sprintf("maat-v9.9.9-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive := writeTestReleaseArchive(t, "maat-v9.9.9-"+runtime.GOOS+"-"+runtime.GOARCH, "release binary\n")
	checksum := fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), assetName)
	oldURL := latestReleaseURL
	oldVersion := version.Version
	oldHTTPClient := updateHTTPClient
	baseURL := "https://maat.test"
	version.Version = "v1.0.0"
	updateHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case baseURL + "/latest":
			if got := request.Header.Get("Accept"); got != "application/json" {
				return nil, fmt.Errorf("latest release Accept header = %q, want application/json", got)
			}
			payload := map[string]any{
				"tag_name": "v9.9.9",
				"assets": []map[string]string{
					{
						"name":                 assetName,
						"browser_download_url": baseURL + "/assets/" + assetName,
					},
					{
						"name":                 "checksums-v9.9.9.txt",
						"browser_download_url": baseURL + "/assets/checksums-v9.9.9.txt",
					},
				},
			}
			data, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			return testHTTPResponse(http.StatusOK, data), nil
		case baseURL + "/assets/" + assetName:
			if got := request.Header.Get("Accept"); got != "application/octet-stream" {
				return nil, fmt.Errorf("asset Accept header = %q, want application/octet-stream", got)
			}
			return testHTTPResponse(http.StatusOK, archive), nil
		case baseURL + "/assets/checksums-v9.9.9.txt":
			if got := request.Header.Get("Accept"); got != "application/octet-stream" {
				return nil, fmt.Errorf("checksum Accept header = %q, want application/octet-stream", got)
			}
			return testHTTPResponse(http.StatusOK, []byte(checksum)), nil
		default:
			return testHTTPResponse(http.StatusNotFound, []byte("not found")), nil
		}
	})}
	latestReleaseURL = baseURL + "/latest"
	defer func() {
		latestReleaseURL = oldURL
		version.Version = oldVersion
		updateHTTPClient = oldHTTPClient
	}()

	output, err := captureRun("update", "--install-dir", installDir, "--binary-name", "maat", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result installCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(installDir, "maat")
	if result.Action != "update.installed" || result.LatestVersion != "v9.9.9" || result.AssetName != assetName || !result.ChecksumVerified || result.TargetPath != target {
		t.Fatalf("unexpected release update result: %#v", result)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "release binary\n" {
		t.Fatalf("unexpected downloaded binary content: %q", string(data))
	}
}

func TestUpdateCommandSkipsWhenAlreadyLatest(t *testing.T) {
	installDir := t.TempDir()
	oldURL := latestReleaseURL
	oldVersion := version.Version
	oldHTTPClient := updateHTTPClient
	latestReleaseURL = "https://maat.test/latest"
	version.Version = "v1.2.3"
	updateHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		data, err := json.Marshal(map[string]any{
			"tag_name": "v1.2.3",
			"assets":   []map[string]string{},
		})
		if err != nil {
			return nil, err
		}
		return testHTTPResponse(http.StatusOK, data), nil
	})}
	defer func() {
		latestReleaseURL = oldURL
		version.Version = oldVersion
		updateHTTPClient = oldHTTPClient
	}()

	output, err := captureRun("update", "--install-dir", installDir, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result installCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "update.current" || result.CurrentVersion != "v1.2.3" || result.LatestVersion != "v1.2.3" {
		t.Fatalf("unexpected current update result: %#v", result)
	}
}

func TestUninstallCommandRemovesBinaryAndCanPurgeConfig(t *testing.T) {
	installDir := t.TempDir()
	target := filepath.Join(installDir, "maat-test")
	if err := os.WriteFile(target, []byte("old binary\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MAAT_CONFIG", configPath)
	if err := os.WriteFile(configPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("uninstall", "--install-dir", installDir, "--binary-name", "maat-test", "--purge-config", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result installCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "uninstall.removed" || !result.Removed || !result.ConfigPurged || result.TargetPath != target {
		t.Fatalf("unexpected uninstall result: %#v", result)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected binary to be removed, got err=%v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected config to be removed, got err=%v", err)
	}
}

func TestUninstallCommandDoesNotReportRemovingMissingBinary(t *testing.T) {
	installDir := t.TempDir()

	output, err := captureRun("uninstall", "--install-dir", installDir, "--binary-name", "maat-test")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output, "removing installed binary") {
		t.Fatalf("missing binary output should not report removal progress: %q", output)
	}
	if !strings.Contains(output, "installed binary was not found") {
		t.Fatalf("expected missing binary warning, got %q", output)
	}
}

func TestSetupCommandWritesConfig(t *testing.T) {
	store := t.TempDir()
	runGit(t, store, "init", "-b", "main")
	configFile := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MAAT_CONFIG", configFile)
	t.Setenv("MAAT_ACTOR", "test-agent")

	output, err := captureRunWithInput("", "setup", "--storage", store, "--no-auto-push", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result setupCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "setup.configured" || result.StoragePath != store || result.ConfigPath != configFile {
		t.Fatalf("unexpected setup result: %#v", result)
	}
	if result.DefaultActor != "test-agent" || !result.AutoPullBeforeRead || !result.AutoCommitAfterWrite || result.AutoPushAfterCommit {
		t.Fatalf("unexpected setup defaults: %#v", result)
	}
	cfg, err := readConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StoragePath != store || cfg.DefaultActor != "test-agent" || !cfg.AutoPullBeforeRead || !cfg.AutoCommitAfterWrite || cfg.AutoPushAfterCommit {
		t.Fatalf("unexpected persisted config: %#v", cfg)
	}
}

func TestSetupCommandPromptsWhenStorageOmitted(t *testing.T) {
	store := t.TempDir()
	runGit(t, store, "init", "-b", "main")
	configFile := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MAAT_CONFIG", configFile)
	t.Setenv("MAAT_ACTOR", "prompt-agent")

	input := strings.Join([]string{
		store,
		"",
		"no",
		"yes",
		"n",
		"",
	}, "\n")
	output, err := captureRunWithInput(input, "setup")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"Storage repo path", "Default actor", "Auto-pull before reads", "Auto-commit after writes", "Auto-push after commits"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected prompt %q in output, got %q", want, output)
		}
	}
	cfg, err := readConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StoragePath != store || cfg.DefaultActor != "prompt-agent" || cfg.AutoPullBeforeRead || !cfg.AutoCommitAfterWrite || cfg.AutoPushAfterCommit {
		t.Fatalf("unexpected prompted config: %#v", cfg)
	}
}

func TestSetupCommandNoStorageRemainsNonInteractiveForJSON(t *testing.T) {
	if _, err := captureRunWithInput("", "setup", "--json"); err == nil || !strings.Contains(err.Error(), "usage: maat setup") {
		t.Fatalf("expected non-interactive json setup usage error, got %v", err)
	}
}

func TestSetupCommandRejectsInvalidStorage(t *testing.T) {
	if _, err := captureRun("setup", "--storage", "relative/path"); err == nil || !strings.Contains(err.Error(), "must be absolute") {
		t.Fatalf("expected relative storage error, got %v", err)
	}

	store := t.TempDir()
	if _, err := captureRun("setup", "--storage", store); err == nil || !strings.Contains(err.Error(), "must be a Git repository") {
		t.Fatalf("expected non-git storage error, got %v", err)
	}
}

func TestReadCommandsAutoPullConfiguredGitStorage(t *testing.T) {
	source := writeObjectCommandFixture(t)
	initGitStore(t, source)
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, filepath.Dir(remote), "init", "--bare", "-b", "main", remote)
	runGit(t, source, "remote", "add", "origin", remote)
	runGit(t, source, "push", "-u", "origin", "main")

	storeParent := t.TempDir()
	store := filepath.Join(storeParent, "store")
	runGit(t, storeParent, "clone", remote, store)
	runGit(t, store, "config", "user.email", "maat@example.test")
	runGit(t, store, "config", "user.name", "Maat Test")

	writer := maat.NewWriteStore(source)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "pulled",
		DisplayName: "Pulled Project",
	}); err != nil {
		t.Fatal(err)
	}
	runGit(t, source, "add", ".")
	runGit(t, source, "commit", "-m", "status(pulled): add project")
	runGit(t, source, "push")
	writeCommandConfig(t, config{
		StoragePath:          store,
		DefaultActor:         "test",
		AutoPullBeforeRead:   true,
		AutoCommitAfterWrite: false,
		AutoPushAfterCommit:  false,
	})

	output, err := captureRun("projects", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var projects []projectListItem
	if err := json.Unmarshal([]byte(output), &projects); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, project := range projects {
		seen[project.ID] = true
	}
	if !seen["sample"] || !seen["pulled"] {
		t.Fatalf("expected auto-pulled projects, got %#v", projects)
	}
}

func TestReadAutoPullWarningEmitsAgentUpdate(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	writeCommandConfig(t, config{
		StoragePath:          store,
		DefaultActor:         "test",
		AutoPullBeforeRead:   true,
		AutoCommitAfterWrite: false,
		AutoPushAfterCommit:  false,
	})

	output, err := captureRun("status", "--agent-use")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var sawWarning, sawFinal bool
	for _, line := range lines {
		var update map[string]any
		if err := json.Unmarshal([]byte(line), &update); err != nil {
			t.Fatalf("invalid agent update %q: %v", line, err)
		}
		if update["step"] == "read.pull" && update["status"] == "warning" {
			sawWarning = true
		}
		if update["step"] == "status.ready" && update["status"] == "ok" {
			sawFinal = true
		}
	}
	if !sawWarning || !sawFinal {
		t.Fatalf("expected pull warning and final update, got %q", output)
	}
}

func TestValidateCommand(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("validate", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "validated 7 files: ok") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestIndexRebuildAndSearchCommand(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("index", "rebuild", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "json:") || !strings.Contains(output, "sqlite:") {
		t.Fatalf("unexpected output: %q", output)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index: %v", err)
	}

	output, err = captureRun("search", "agent health", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "docs/note.md:3") {
		t.Fatalf("unexpected search output: %q", output)
	}
}

func TestProjectLinkCommand(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitStore(t, source)
	runGit(t, source, "remote", "add", "origin", "git@github.com:sunday-studio/sample.git")

	output, err := captureRun("project", "link", source, "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "linked project sample") || !strings.Contains(output, "Remote:") || !strings.Contains(output, "git@github.com:sunday-studio/sample.git") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if project.DisplayName != "Sample" || project.Identity["Remote"] != "git@github.com:sunday-studio/sample.git" {
		t.Fatalf("unexpected linked project: %#v", project)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index after link: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index after link: %v", err)
	}
}

func TestProjectLinkCommandJSONAndIdempotent(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()

	output, err := captureRun("project", "link", source, "--storage", store, "--key", "photo-system", "--name", "Photo System", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var linked maat.LinkedProject
	if err := json.Unmarshal([]byte(output), &linked); err != nil {
		t.Fatal(err)
	}
	if !linked.Created || linked.ProjectKey != "photo-system" || linked.DisplayName != "Photo System" {
		t.Fatalf("unexpected link json: %#v", linked)
	}

	output, err = captureRun("project", "link", source, "--storage", store, "--key", "photo-system", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(output), &linked); err != nil {
		t.Fatal(err)
	}
	if !linked.Existing || linked.Created {
		t.Fatalf("expected idempotent existing project, got %#v", linked)
	}
}

func TestProjectShowCommandSupportsObjectProject(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "photo-system",
		DisplayName: "Photo System",
	}); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("project", "show", "photo-system", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Photo System", "Key:     photo-system", "Repo:    " + source} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to include %q, got %q", want, output)
		}
	}
}

func TestProjectShowCommandJSON(t *testing.T) {
	store := writeCommandFixture(t)

	output, err := captureRun("project", "show", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var object maat.ObjectProject
	if err := json.Unmarshal([]byte(output), &object); err != nil {
		t.Fatal(err)
	}
	if object.Key != "sample" || object.DisplayName != "Sample" {
		t.Fatalf("unexpected object project json: %#v", object)
	}
}

func TestGoalCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)

	output, err := captureRun("goal", "create", "sample", "Ship command writes", "--outcome", "Agents can see the command write outcome.", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created goal") || !strings.Contains(output, "Event:") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Goals) != 1 || project.Goals[0].Title != "Ship command writes" {
		t.Fatalf("unexpected goals: %#v", project.Goals)
	}
	if project.Goals[0].Outcome != "Agents can see the command write outcome." {
		t.Fatalf("unexpected goal outcome: %#v", project.Goals[0])
	}
	if len(project.Events) != 1 || project.Events[0].Type != "goal.created" {
		t.Fatalf("unexpected events: %#v", project.Events)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.json")); err != nil {
		t.Fatalf("expected json index after write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat", "index.sqlite")); err != nil {
		t.Fatalf("expected sqlite index after write: %v", err)
	}
}

func TestGoalCreateCommandJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)

	output, err := captureRun("goal", "create", "sample", "Ship json writes", "--outcome", "Agents can see the JSON write outcome.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "goal.created" || result.ProjectKey != "sample" || result.GoalID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
	if !result.IndexRefreshed || result.IndexWarning != "" {
		t.Fatalf("expected successful index refresh, got %#v", result)
	}
}

func TestWriteCommandsAutoCommitConfiguredGitStorage(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	initGitStore(t, store)
	writeCommandConfig(t, config{
		StoragePath:          store,
		DefaultActor:         "codex",
		AutoPullBeforeRead:   false,
		AutoCommitAfterWrite: true,
		AutoPushAfterCommit:  false,
	})

	output, err := captureRun("goal", "create", "sample", "Auto committed goal", "--outcome", "Auto commit is recorded after goal creation.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.AutoSync == nil || !result.AutoSync.Committed || result.AutoSync.Pushed {
		t.Fatalf("expected auto commit without push, got %#v", result.AutoSync)
	}
	if result.AutoSync.CommitMessage != "status(sample): create goal" {
		t.Fatalf("unexpected auto commit message: %#v", result.AutoSync)
	}
	status := runGit(t, store, "status", "--porcelain=v1")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("expected clean storage after auto commit, got %q", status)
	}
	log := strings.TrimSpace(runGit(t, store, "log", "-1", "--pretty=%s"))
	if log != "status(sample): create goal" {
		t.Fatalf("unexpected auto commit subject: %q", log)
	}
}

func TestWriteCommandsAutoPushConfiguredGitStorage(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	source := writeObjectCommandFixture(t)
	initGitStore(t, source)
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, filepath.Dir(remote), "init", "--bare", "-b", "main", remote)
	runGit(t, source, "remote", "add", "origin", remote)
	runGit(t, source, "push", "-u", "origin", "main")

	storeParent := t.TempDir()
	store := filepath.Join(storeParent, "store")
	runGit(t, storeParent, "clone", remote, store)
	runGit(t, store, "config", "user.email", "maat@example.test")
	runGit(t, store, "config", "user.name", "Maat Test")
	writeCommandConfig(t, config{
		StoragePath:          store,
		DefaultActor:         "codex",
		AutoPullBeforeRead:   false,
		AutoCommitAfterWrite: true,
		AutoPushAfterCommit:  true,
	})

	output, err := captureRun("goal", "create", "sample", "Auto pushed goal", "--outcome", "Auto push is recorded after goal creation.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.AutoSync == nil || !result.AutoSync.Committed || !result.AutoSync.Pushed {
		t.Fatalf("expected auto commit and push, got %#v", result.AutoSync)
	}
	runGit(t, source, "pull", "--rebase")
	project, err := maat.LoadObjectProject(source, "sample")
	if err != nil {
		t.Fatal(err)
	}
	var sawGoal bool
	for _, goal := range project.Goals {
		if goal.Title == "Auto pushed goal" {
			sawGoal = true
		}
	}
	if !sawGoal {
		t.Fatalf("expected pushed goal in source checkout, got %#v", project.Goals)
	}
}

func TestGoalCreateCommandTreatsIndexFailureAsWarning(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	if err := os.WriteFile(filepath.Join(store, ".maat"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("goal", "create", "sample", "Survive index failure", "--outcome", "The goal persists even when index refresh fails.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "goal.created" || result.GoalID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
	if result.IndexRefreshed || !strings.Contains(result.IndexWarning, "index refresh failed after state write persisted") {
		t.Fatalf("expected index warning, got %#v", result)
	}
	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Goals) != 1 || project.Goals[0].Title != "Survive index failure" {
		t.Fatalf("expected persisted goal despite index failure, got %#v", project.Goals)
	}
}

func TestGoalCreateAgentUseEmitsIndexWarning(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	if err := os.WriteFile(filepath.Join(store, ".maat"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("goal", "create", "sample", "Agent warning", "--outcome", "Agent-use output reports index warnings.", "--storage", store, "--agent-use")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected multiple agent updates, got %q", output)
	}
	var sawWarning, sawFinal bool
	for _, line := range lines {
		var update map[string]any
		if err := json.Unmarshal([]byte(line), &update); err != nil {
			t.Fatalf("invalid agent update %q: %v", line, err)
		}
		if update["step"] == "index.refresh" && update["status"] == "warning" {
			sawWarning = true
		}
		if update["step"] == "goal.created" && update["status"] == "ok" {
			sawFinal = true
			data, ok := update["data"].(map[string]any)
			if !ok {
				t.Fatalf("expected final data object, got %#v", update["data"])
			}
			if data["index_refreshed"] != false {
				t.Fatalf("expected final update to report stale index, got %#v", data)
			}
		}
	}
	if !sawWarning || !sawFinal {
		t.Fatalf("expected warning and final updates, got %q", output)
	}
}

func TestGoalCreateAgentUseEmitsAutoSyncWarning(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	runGit(t, store, "init", "-b", "main")
	writeCommandConfig(t, config{
		StoragePath:          store,
		DefaultActor:         "codex",
		AutoPullBeforeRead:   false,
		AutoCommitAfterWrite: true,
		AutoPushAfterCommit:  true,
	})

	output, err := captureRun("goal", "create", "sample", "Agent sync warning", "--outcome", "Agent-use output reports sync warnings.", "--storage", store, "--agent-use")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var sawWarning, sawFinal bool
	for _, line := range lines {
		var update map[string]any
		if err := json.Unmarshal([]byte(line), &update); err != nil {
			t.Fatalf("invalid agent update %q: %v", line, err)
		}
		if update["step"] == "write.sync" && update["status"] == "warning" {
			sawWarning = true
		}
		if update["step"] == "goal.created" && update["status"] == "ok" {
			sawFinal = true
			data, ok := update["data"].(map[string]any)
			if !ok {
				t.Fatalf("expected final data object, got %#v", update["data"])
			}
			if data["auto_sync_warning"] == "" {
				t.Fatalf("expected final update to report auto sync warning, got %#v", data)
			}
		}
	}
	if !sawWarning || !sawFinal {
		t.Fatalf("expected auto sync warning and final updates, got %q", output)
	}
}

func TestGoalCreateCommandInfersLinkedProject(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("goal", "create", "Inferred goal", "--outcome", "The linked project is inferred for the goal.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.GoalID == "" {
		t.Fatalf("unexpected inferred goal result: %#v", result)
	}
}

func TestTicketCreateCommand(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)

	output, err := captureRun("ticket", "create", "sample", "Wire ticket command", "--goal", goalID, "--description", "Wire the ticket create command through the write store.", "--acceptance", "The ticket stores enough detail for another agent.", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "created ticket") || !strings.Contains(output, "Event:") {
		t.Fatalf("unexpected output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Tickets) != 1 || project.Tickets[0].Title != "Wire ticket command" {
		t.Fatalf("unexpected tickets: %#v", project.Tickets)
	}
	if project.Tickets[0].GoalID != goalID {
		t.Fatalf("expected goal link %q, got %q", goalID, project.Tickets[0].GoalID)
	}
	if project.Tickets[0].Description == "" || len(project.Tickets[0].Acceptance) != 1 {
		t.Fatalf("expected actionable ticket details: %#v", project.Tickets[0])
	}
}

func TestTicketCreateCommandJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	goalID := createCommandGoal(t, store)

	output, err := captureRun("ticket", "create", "sample", "Wire json ticket", "--goal", goalID, "--description", "Return a JSON result after creating a detailed ticket.", "--acceptance", "The ticket create result includes the created IDs.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "ticket.created" || result.ProjectKey != "sample" || result.GoalID != goalID || result.TicketID == "" || result.EventID == "" {
		t.Fatalf("unexpected json result: %#v", result)
	}
}

func TestTicketCreateCommandInfersLinkedProject(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("ticket", "create", "Inferred ticket", "--description", "Create a ticket using the linked project from the current directory.", "--acceptance", "The command returns the inferred project key.", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.TicketID == "" {
		t.Fatalf("unexpected inferred ticket result: %#v", result)
	}
}

func TestTicketListAndShowCommands(t *testing.T) {
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	output, err := captureRun("ticket", "list", "--project", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tickets []ticketView
	if err := json.Unmarshal([]byte(output), &tickets); err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != ticketID || tickets[0].ProjectKey != "sample" {
		t.Fatalf("unexpected tickets: %#v", tickets)
	}

	output, err = captureRun("ticket", "show", ticketID, "--project", "sample", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var ticket ticketView
	if err := json.Unmarshal([]byte(output), &ticket); err != nil {
		t.Fatal(err)
	}
	if ticket.ID != ticketID || ticket.Status != "active" || ticket.Title != "Existing ticket" {
		t.Fatalf("unexpected ticket: %#v", ticket)
	}
	if ticket.Description == "" || len(ticket.Acceptance) != 1 {
		t.Fatalf("expected ticket details in json: %#v", ticket)
	}

	output, err = captureRun("ticket", "show", ticketID, "--project", "sample", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Description:", "Existing ticket fixtures can be shown", "Acceptance:", "The ticket show command exposes actionable details."} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected show output to include %q, got %q", want, output)
		}
	}
}

func TestTicketListInfersLinkedProject(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	writer := maat.NewWriteStore(store)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "other",
		DisplayName: "Other",
	}); err != nil {
		t.Fatal(err)
	}
	sampleTicket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		Title:       "Sample ticket",
		Description: "Exercise ticket filtering for the sample project.",
		Acceptance:  []string{"Only the sample project ticket is returned."},
		Actor:       "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "other",
		Title:       "Other ticket",
		Description: "Exercise ticket filtering for another project.",
		Acceptance:  []string{"The other project ticket is excluded."},
		Actor:       "test",
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(source)

	output, err := captureRun("ticket", "list", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tickets []ticketView
	if err := json.Unmarshal([]byte(output), &tickets); err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 1 || tickets[0].ID != sampleTicket.ID || tickets[0].ProjectKey != "sample" {
		t.Fatalf("expected linked project ticket only, got %#v", tickets)
	}
}

func TestTicketEventCommands(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	if output, err := captureRun("ticket", "claim", ticketID, "--agent", "claude", "--ttl", "30m", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "claimed ticket") {
		t.Fatalf("unexpected claim output: %q", output)
	}
	if output, err := captureRun("ticket", "comment", ticketID, "Progress note", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "commented on ticket") {
		t.Fatalf("unexpected comment output: %q", output)
	}
	if output, err := captureRun("ticket", "complete", ticketID, "--evidence", "go test ./...", "--storage", store); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(output, "completed ticket") {
		t.Fatalf("unexpected complete output: %q", output)
	}

	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	eventTypes := map[string]bool{}
	for _, event := range project.Events {
		eventTypes[event.Type] = true
	}
	for _, eventType := range []string{"ticket.claimed", "ticket.commented", "ticket.completed"} {
		if !eventTypes[eventType] {
			t.Fatalf("missing event type %s in %#v", eventType, project.Events)
		}
	}
}

func TestTicketEventCommandsInferLinkedProjectWhenTicketIDIsDuplicated(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := t.TempDir()
	source := t.TempDir()
	if _, err := maat.LinkProject(t.Context(), maat.LinkProjectInput{
		Store:       store,
		SourcePath:  source,
		ProjectKey:  "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	writer := maat.NewWriteStore(store)
	ticket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		Title:       "Shared ticket id",
		Description: "Exercise duplicate ticket project inference.",
		Acceptance:  []string{"The first matching linked project is selected."},
		Actor:       "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "other",
		DisplayName: "Other",
	}); err != nil {
		t.Fatal(err)
	}
	writeDuplicateTicket(t, store, "other", ticket.ID)
	t.Chdir(source)

	output, err := captureRun("ticket", "comment", ticket.ID, "Inferred duplicate", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result writeCommandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProjectKey != "sample" || result.TicketID != ticket.ID {
		t.Fatalf("expected linked project inference, got %#v", result)
	}
}

func TestTicketEventCommandsJSON(t *testing.T) {
	t.Setenv("MAAT_ACTOR", "codex")
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	output, err := captureRun("ticket", "claim", ticketID, "--agent", "claude", "--ttl", "30m", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var claim writeCommandResult
	if err := json.Unmarshal([]byte(output), &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Action != "ticket.claimed" || claim.TicketID != ticketID || claim.ProjectKey != "sample" || claim.Agent != "claude" || claim.ExpiresAt == "" || claim.EventID == "" {
		t.Fatalf("unexpected claim json: %#v", claim)
	}

	output, err = captureRun("ticket", "comment", ticketID, "Progress note", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var comment writeCommandResult
	if err := json.Unmarshal([]byte(output), &comment); err != nil {
		t.Fatal(err)
	}
	if comment.Action != "ticket.commented" || comment.TicketID != ticketID || comment.ProjectKey != "sample" || comment.EventID == "" {
		t.Fatalf("unexpected comment json: %#v", comment)
	}

	output, err = captureRun("ticket", "complete", ticketID, "--evidence", "go test ./...", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var complete writeCommandResult
	if err := json.Unmarshal([]byte(output), &complete); err != nil {
		t.Fatal(err)
	}
	if complete.Action != "ticket.completed" || complete.TicketID != ticketID || complete.ProjectKey != "sample" || complete.EventID == "" {
		t.Fatalf("unexpected complete json: %#v", complete)
	}
}

func TestTicketCreateCommandRequiresProjectAndTitle(t *testing.T) {
	store := writeObjectCommandFixture(t)

	_, err := captureRun("ticket", "create", "Only title", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "project key is required") {
		t.Fatalf("expected project/title error, got %v", err)
	}
}

func TestGoalCreateCommandRequiresOutcome(t *testing.T) {
	store := writeObjectCommandFixture(t)

	_, err := captureRun("goal", "create", "sample", "Missing outcome", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "--outcome is required") {
		t.Fatalf("expected outcome error, got %v", err)
	}
}

func TestTicketCreateCommandRequiresDetails(t *testing.T) {
	store := writeObjectCommandFixture(t)

	_, err := captureRun("ticket", "create", "sample", "Missing description", "--acceptance", "It is accepted.", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "--description is required") {
		t.Fatalf("expected description error, got %v", err)
	}

	_, err = captureRun("ticket", "create", "sample", "Missing acceptance", "--description", "Do the work.", "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "--acceptance is required") {
		t.Fatalf("expected acceptance error, got %v", err)
	}
}

func TestTicketCompleteRequiresEvidence(t *testing.T) {
	store := writeObjectCommandFixture(t)
	ticketID := createCommandTicket(t, store)

	_, err := captureRun("ticket", "complete", ticketID, "--storage", store)
	if err == nil || !strings.Contains(err.Error(), "--evidence is required") {
		t.Fatalf("expected evidence error, got %v", err)
	}
}

func TestAgentInitializeCommand(t *testing.T) {
	store := t.TempDir()

	output, err := captureRun("initialize", "--project", "maat", "--storage", store)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"# Maat Agent Instructions",
		"maat setup --storage " + store,
		"maat initialize --project maat --storage " + store,
		"maat project show maat --storage " + store,
		"maat ticket claim <ticket-id> --project maat --agent \"<agent-id>\"",
		"Save the snippet below into `AGENTS.md`, `CLAUDE.md`, Cursor rules",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected initialize output to include %q, got %q", want, output)
		}
	}
}

func TestInitializeCommandJSON(t *testing.T) {
	store := t.TempDir()

	output, err := captureRun("initialize", "--project", "maat", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}

	var payload initializeCommandResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProjectKey != "maat" || payload.StoragePath != store || payload.LinkedProject.ProjectKey != "maat" {
		t.Fatalf("unexpected initialize payload: %#v", payload)
	}
	if !strings.Contains(payload.Document, "# Maat Agent Instructions") || !strings.Contains(payload.Document, "maat initialize --project maat --storage "+store) {
		t.Fatalf("unexpected initialize payload: %#v", payload)
	}
}

func TestInitializeCommandRegistersCurrentGitRepo(t *testing.T) {
	store := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("# Sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initGitStore(t, source)
	runGit(t, source, "remote", "add", "origin", "git@github.com:sunday-studio/sample.git")
	withWorkingDir(t, source)

	output, err := captureRun("initialize", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var payload initializeCommandResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.LinkedProject.Created || payload.ProjectKey != "sample" || payload.LinkedProject.SourcePath == "" {
		t.Fatalf("unexpected initialize link result: %#v", payload)
	}
	if payload.LinkedProject.RemoteURL != "git@github.com:sunday-studio/sample.git" {
		t.Fatalf("expected remote in initialize payload, got %#v", payload.LinkedProject)
	}
	if !strings.Contains(payload.Document, "This repo is registered as `sample`.") || !strings.Contains(payload.Document, "maat project show sample --storage "+store) {
		t.Fatalf("expected concrete project setup document, got %q", payload.Document)
	}
	project, err := maat.LoadObjectProject(store, "sample")
	if err != nil {
		t.Fatal(err)
	}
	if project.Identity["Primary Repo"] != payload.LinkedProject.SourcePath || project.Identity["Remote"] != "git@github.com:sunday-studio/sample.git" {
		t.Fatalf("unexpected registered project identity: %#v", project.Identity)
	}

	output, err = captureRun("initialize", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.LinkedProject.Existing || payload.LinkedProject.Created || payload.ProjectKey != "sample" {
		t.Fatalf("expected idempotent initialize rerun, got %#v", payload)
	}
}

func TestInitializeCommandRejectsAgentFlag(t *testing.T) {
	if _, err := captureRun("initialize", "--agent", "codex"); err == nil || !strings.Contains(err.Error(), "unexpected initialize argument") {
		t.Fatalf("expected --agent to be rejected, got %v", err)
	}
}

func TestInitializeCommandRejectsOutputFlag(t *testing.T) {
	if _, err := captureRun("initialize", "--output", "AGENTS.md"); err == nil || !strings.Contains(err.Error(), "unexpected initialize argument") {
		t.Fatalf("expected --output to be rejected, got %v", err)
	}
}

func TestAgentCommandRemoved(t *testing.T) {
	if _, err := captureRun("agent", "initialize"); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected agent command to be removed, got %v", err)
	}
}

func TestSyncStatusCommandReportsDirtyState(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "scratch.md"), []byte("# Scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--status")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Repository: git repository") || !strings.Contains(output, "Dirty:") || !strings.Contains(output, "scratch.md") {
		t.Fatalf("unexpected output: %q", output)
	}
	if _, err := os.Stat(filepath.Join(store, ".maat")); !os.IsNotExist(err) {
		t.Fatalf("status command should not rebuild indexes, got err=%v", err)
	}
}

func TestSyncCommandCommitsChanges(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "docs", "sync.md"), []byte("# Sync\n\nCommit me.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--message", "status(maat): test sync")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Validation:") || !strings.Contains(output, "Committed:  status(maat): test sync") {
		t.Fatalf("unexpected output: %q", output)
	}
	status := runGit(t, store, "status", "--porcelain=v1")
	if strings.TrimSpace(status) != "" {
		t.Fatalf("expected clean git status after sync, got %q", status)
	}
	log := runGit(t, store, "log", "-1", "--pretty=%s")
	if strings.TrimSpace(log) != "status(maat): test sync" {
		t.Fatalf("unexpected commit subject: %q", log)
	}
}

func TestSyncCommandJSON(t *testing.T) {
	store := writeCommandFixture(t)
	initGitStore(t, store)
	if err := os.WriteFile(filepath.Join(store, "docs", "json.md"), []byte("# JSON sync\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := captureRun("sync", "--storage", store, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var result maat.StoreSyncResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Committed || result.CommitMessage != "status(maat): sync store" {
		t.Fatalf("unexpected sync result: %#v", result)
	}
	if result.SQLiteIndex.Path == "" || result.JSONIndexPath == "" {
		t.Fatalf("expected rebuilt indexes: %#v", result)
	}
}

func captureRun(args ...string) (string, error) {
	return captureRunWithInput("\n", args...)
}

func captureRunWithInput(input string, args ...string) (string, error) {
	oldStdout := os.Stdout
	oldStdin := os.Stdin
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		stdoutReader.Close()
		stdoutWriter.Close()
		return "", err
	}
	if _, err := stdinWriter.WriteString(input); err != nil {
		stdoutReader.Close()
		stdoutWriter.Close()
		stdinReader.Close()
		stdinWriter.Close()
		return "", err
	}
	stdinWriter.Close()
	os.Stdout = stdoutWriter
	os.Stdin = stdinReader
	defer func() {
		os.Stdout = oldStdout
		os.Stdin = oldStdin
	}()
	runErr := run(args)
	stdoutWriter.Close()
	stdinReader.Close()

	data, readErr := io.ReadAll(stdoutReader)
	stdoutReader.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(data), runErr
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
}

func initGitStore(t *testing.T, store string) {
	t.Helper()
	runGit(t, store, "init", "-b", "main")
	runGit(t, store, "config", "user.email", "maat@example.test")
	runGit(t, store, "config", "user.name", "Maat Test")
	runGit(t, store, "add", ".")
	runGit(t, store, "commit", "-m", "test: seed store")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writeCommandConfig(t *testing.T, cfg config) {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("MAAT_CONFIG", configFile)
	if _, err := persistConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testHTTPResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

func writeTestReleaseArchive(t *testing.T, name, content string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	data := []byte(content)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(data)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func writeObjectCommandFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writer := maat.NewWriteStore(root)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func createCommandGoal(t *testing.T, store string) string {
	t.Helper()

	writer := maat.NewWriteStore(store)
	goal, _, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: "sample",
		Title:      "Existing goal",
		Outcome:    "Existing goal fixtures can be linked by command tests.",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return goal.ID
}

func createCommandTicket(t *testing.T, store string) string {
	t.Helper()

	writer := maat.NewWriteStore(store)
	ticket, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		Title:       "Existing ticket",
		Description: "Existing ticket fixtures can be shown by command tests.",
		Acceptance:  []string{"The ticket show command exposes actionable details."},
		Actor:       "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	return ticket.ID
}

func writeCommandFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "note.md"), []byte("# Note\n\nAgent health needs clarity.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer := maat.NewWriteStore(root)
	if _, err := writer.CreateProject(maat.CreateProjectInput{
		Key:         "sample",
		DisplayName: "Sample",
	}); err != nil {
		t.Fatal(err)
	}
	goal, _, err := writer.CreateGoal(maat.CreateGoalInput{
		ProjectKey: "sample",
		Title:      "Ship",
		Outcome:    "The fixture project has a goal for tickets.",
		Actor:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		GoalID:      goal.ID,
		Title:       "Open item",
		Description: "Exercise an open ticket row.",
		Acceptance:  []string{"The open ticket appears in the fixture."},
		Actor:       "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := writer.CreateTicket(maat.CreateTicketInput{
		ProjectKey:  "sample",
		GoalID:      goal.ID,
		Title:       "Second item",
		Description: "Exercise a second ticket row.",
		Acceptance:  []string{"The second ticket appears in the fixture."},
		Actor:       "test",
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeDuplicateTicket(t *testing.T, store, projectKey, ticketID string) {
	t.Helper()
	path := filepath.Join(store, "projects", projectKey, "tickets", ticketID+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `# Ticket: Duplicate Ticket

| Field | Value |
|---|---|
| Ticket ID | ` + ticketID + ` |
| Project | ` + projectKey + ` |
| Goal | none |
| Status | active |
| Created | 2026-05-25T10:00:00Z |
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
