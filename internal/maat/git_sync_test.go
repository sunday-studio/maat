package maat

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

type fakeGitCall struct {
	dir  string
	args []string
}

type fakeGitResponse struct {
	result GitCommandResult
	err    error
}

type fakeGitRunner struct {
	responses []fakeGitResponse
	calls     []fakeGitCall
}

func (runner *fakeGitRunner) RunGit(ctx context.Context, dir string, args ...string) (GitCommandResult, error) {
	runner.calls = append(runner.calls, fakeGitCall{
		dir:  dir,
		args: append([]string(nil), args...),
	})
	if len(runner.responses) == 0 {
		return GitCommandResult{}, nil
	}
	response := runner.responses[0]
	runner.responses = runner.responses[1:]
	return response.result, response.err
}

func TestGitSyncInfoReadsRepositoryBranchAndRemote(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: "true\n"}},
		{result: GitCommandResult{Stdout: "main\n"}},
		{result: GitCommandResult{Stdout: "git@github.com:sunday-studio/maat-state.git\n"}},
	}}

	info, err := GitSync{Store: "/tmp/maat-state", Runner: runner}.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsRepository || info.Branch != "main" || info.RemoteURL != "git@github.com:sunday-studio/maat-state.git" {
		t.Fatalf("unexpected repo info: %#v", info)
	}
	assertGitCalls(t, runner.calls, [][]string{
		{"rev-parse", "--is-inside-work-tree"},
		{"branch", "--show-current"},
		{"remote", "get-url", "origin"},
	})
}

func TestGitSyncIsRepositoryReturnsFalseForNonRepo(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{
			result: GitCommandResult{Stderr: "fatal: not a git repository (or any of the parent directories): .git\n"},
			err:    errors.New("exit status 128"),
		},
	}}

	isRepository, err := GitSync{Store: "/tmp/nope", Runner: runner}.IsRepository(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if isRepository {
		t.Fatal("expected non-repository path")
	}
}

func TestGitSyncRemoteURLAllowsMissingOrigin(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{
			result: GitCommandResult{Stderr: "error: No such remote 'origin'\n"},
			err:    errors.New("exit status 2"),
		},
	}}

	remoteURL, err := GitSync{Store: "/tmp/repo", Runner: runner}.RemoteURL(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if remoteURL != "" {
		t.Fatalf("expected empty remote URL, got %q", remoteURL)
	}
}

func TestGitSyncDirtyStatusParsesPorcelain(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{result: GitCommandResult{Stdout: " M README.md\nA  docs/plan.md\nR  old.md -> new.md\n?? scratch.md\n"}},
	}}

	entries, err := GitSync{Store: "/tmp/repo", Runner: runner}.DirtyStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []GitStatusEntry{
		{Index: ' ', Work: 'M', Path: "README.md"},
		{Index: 'A', Work: ' ', Path: "docs/plan.md"},
		{Index: 'R', Work: ' ', Path: "new.md", Rename: "old.md"},
		{Index: '?', Work: '?', Path: "scratch.md"},
	}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("unexpected status entries:\nwant: %#v\n got: %#v", want, entries)
	}
}

func TestGitSyncPullRebaseCommitAndPushConstructCommands(t *testing.T) {
	runner := &fakeGitRunner{responses: []fakeGitResponse{
		{},
		{},
		{},
		{},
	}}
	sync := GitSync{Store: "/tmp/repo", Runner: runner}

	if err := sync.PullRebase(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := sync.Commit(context.Background(), "feat(sync): test command construction", "projects/orion/project.md", "projects/orion/events"); err != nil {
		t.Fatal(err)
	}
	if err := sync.Push(context.Background(), "origin", "main"); err != nil {
		t.Fatal(err)
	}

	assertGitCalls(t, runner.calls, [][]string{
		{"pull", "--rebase"},
		{"add", "--", "projects/orion/project.md", "projects/orion/events"},
		{"commit", "-m", "feat(sync): test command construction"},
		{"push", "origin", "main"},
	})
}

func TestGitSyncCommitRequiresMessage(t *testing.T) {
	err := GitSync{Store: "/tmp/repo", Runner: &fakeGitRunner{}}.Commit(context.Background(), " ")
	if err == nil || !strings.Contains(err.Error(), "commit message is required") {
		t.Fatalf("expected required message error, got %v", err)
	}
}

func TestGitSyncPushRequiresRemoteWithBranch(t *testing.T) {
	err := GitSync{Store: "/tmp/repo", Runner: &fakeGitRunner{}}.Push(context.Background(), "", "main")
	if err == nil || !strings.Contains(err.Error(), "remote is required") {
		t.Fatalf("expected remote error, got %v", err)
	}
}

func TestExecGitRunnerDetectsLocalGitRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	if output, err := exec.Command("git", "-C", root, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	isRepository, err := IsGitRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !isRepository {
		t.Fatal("expected initialized temp directory to be a git repository")
	}
}

func assertGitCalls(t *testing.T, calls []fakeGitCall, want [][]string) {
	t.Helper()
	if len(calls) != len(want) {
		t.Fatalf("expected %d git calls, got %d: %#v", len(want), len(calls), calls)
	}
	for index, call := range calls {
		if !reflect.DeepEqual(call.args, want[index]) {
			t.Fatalf("call %d args mismatch:\nwant: %#v\n got: %#v", index, want[index], call.args)
		}
	}
}
