package maat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitCommandRunner interface {
	RunGit(ctx context.Context, dir string, args ...string) (GitCommandResult, error)
}

type GitCommandResult struct {
	Stdout string
	Stderr string
}

type ExecGitRunner struct{}

func (ExecGitRunner) RunGit(ctx context.Context, dir string, args ...string) (GitCommandResult, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	return GitCommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, err
}

type GitSync struct {
	Store  string
	Runner GitCommandRunner
}

type GitRepoInfo struct {
	IsRepository bool   `json:"is_repository"`
	Branch       string `json:"branch,omitempty"`
	RemoteURL    string `json:"remote_url,omitempty"`
}

type GitStatusEntry struct {
	Index  byte   `json:"index"`
	Work   byte   `json:"work"`
	Path   string `json:"path"`
	Rename string `json:"rename,omitempty"`
}

func IsGitRepository(ctx context.Context, store string) (bool, error) {
	return GitSync{Store: store}.IsRepository(ctx)
}

func GitRemoteURL(ctx context.Context, store string) (string, error) {
	return GitSync{Store: store}.RemoteURL(ctx)
}

func (sync GitSync) IsRepository(ctx context.Context) (bool, error) {
	result, err := sync.run(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if isGitNotRepository(result, err) {
			return false, nil
		}
		return false, gitCommandError("git repository check failed", result, err)
	}
	return strings.TrimSpace(result.Stdout) == "true", nil
}

func (sync GitSync) Info(ctx context.Context) (GitRepoInfo, error) {
	isRepository, err := sync.IsRepository(ctx)
	if err != nil || !isRepository {
		return GitRepoInfo{IsRepository: isRepository}, err
	}

	branch, err := sync.CurrentBranch(ctx)
	if err != nil {
		return GitRepoInfo{}, err
	}
	remoteURL, err := sync.RemoteURL(ctx)
	if err != nil {
		return GitRepoInfo{}, err
	}
	return GitRepoInfo{
		IsRepository: true,
		Branch:       branch,
		RemoteURL:    remoteURL,
	}, nil
}

func (sync GitSync) CurrentBranch(ctx context.Context) (string, error) {
	result, err := sync.run(ctx, "branch", "--show-current")
	if err != nil {
		return "", gitCommandError("git branch read failed", result, err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (sync GitSync) RemoteURL(ctx context.Context) (string, error) {
	result, err := sync.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		if isMissingRemote(result) {
			return "", nil
		}
		return "", gitCommandError("git remote read failed", result, err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (sync GitSync) DirtyStatus(ctx context.Context) ([]GitStatusEntry, error) {
	result, err := sync.run(ctx, "status", "--porcelain=v1")
	if err != nil {
		return nil, gitCommandError("git status failed", result, err)
	}
	return ParseGitPorcelainStatus(result.Stdout), nil
}

func (sync GitSync) PullRebase(ctx context.Context) error {
	result, err := sync.run(ctx, "pull", "--rebase")
	if err != nil {
		return gitCommandError("git pull --rebase failed", result, err)
	}
	return nil
}

func (sync GitSync) Commit(ctx context.Context, message string, pathspecs ...string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return errors.New("commit message is required")
	}
	if len(pathspecs) > 0 {
		addArgs := append([]string{"add", "--"}, pathspecs...)
		if err := validatePathspecs(pathspecs); err != nil {
			return err
		}
		result, err := sync.run(ctx, addArgs...)
		if err != nil {
			return gitCommandError("git add failed", result, err)
		}
	}

	result, err := sync.run(ctx, "commit", "-m", message)
	if err != nil {
		return gitCommandError("git commit failed", result, err)
	}
	return nil
}

func (sync GitSync) Push(ctx context.Context, remote, branch string) error {
	args := []string{"push"}
	remote = strings.TrimSpace(remote)
	branch = strings.TrimSpace(branch)
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		if remote == "" {
			return errors.New("remote is required when branch is provided")
		}
		args = append(args, branch)
	}
	result, err := sync.run(ctx, args...)
	if err != nil {
		return gitCommandError("git push failed", result, err)
	}
	return nil
}

func ParseGitPorcelainStatus(output string) []GitStatusEntry {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	entries := make([]GitStatusEntry, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}

		entry := GitStatusEntry{
			Index: line[0],
			Work:  line[1],
			Path:  strings.TrimSpace(line[3:]),
		}
		if (entry.Index == 'R' || entry.Index == 'C') && strings.Contains(entry.Path, " -> ") {
			parts := strings.SplitN(entry.Path, " -> ", 2)
			entry.Rename = parts[0]
			entry.Path = parts[1]
		}
		entries = append(entries, entry)
	}
	return entries
}

func (sync GitSync) run(ctx context.Context, args ...string) (GitCommandResult, error) {
	runner := sync.Runner
	if runner == nil {
		runner = ExecGitRunner{}
	}
	return runner.RunGit(ctx, filepath.Clean(sync.Store), args...)
}

func validatePathspecs(pathspecs []string) error {
	for _, pathspec := range pathspecs {
		if strings.TrimSpace(pathspec) == "" {
			return errors.New("commit pathspecs must not be empty")
		}
	}
	return nil
}

func isGitNotRepository(result GitCommandResult, err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(result.Stdout + result.Stderr)
	return strings.Contains(text, "not a git repository") || strings.Contains(text, "not a git repo")
}

func isMissingRemote(result GitCommandResult) bool {
	text := strings.ToLower(result.Stdout + result.Stderr)
	return strings.Contains(text, "no such remote") || strings.Contains(text, "no configured push destination")
}

func gitCommandError(prefix string, result GitCommandResult, err error) error {
	message := strings.TrimSpace(result.Stderr)
	if message == "" {
		message = strings.TrimSpace(result.Stdout)
	}
	if message == "" {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %s: %w", prefix, message, err)
}
