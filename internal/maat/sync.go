package maat

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type StoreSyncOptions struct {
	Store        string
	Runner       GitCommandRunner
	Message      string
	Pathspecs    []string
	Push         bool
	Remote       string
	Branch       string
	SkipIndex    bool
	SkipValidate bool
}

type StoreSyncResult struct {
	Repository        GitRepoInfo      `json:"repository"`
	Validation        ValidationReport `json:"validation"`
	JSONIndexPath     string           `json:"json_index_path,omitempty"`
	SQLiteIndex       SQLiteIndexInfo  `json:"sqlite_index"`
	DirtyBeforeCommit []GitStatusEntry `json:"dirty_before_commit,omitempty"`
	DirtyAfterSync    []GitStatusEntry `json:"dirty_after_sync,omitempty"`
	Committed         bool             `json:"committed"`
	Pushed            bool             `json:"pushed"`
	CommitMessage     string           `json:"commit_message,omitempty"`
	CommitPathspecs   []string         `json:"commit_pathspecs,omitempty"`
}

type ValidationFailedError struct {
	Report ValidationReport
}

func (err ValidationFailedError) Error() string {
	return fmt.Sprintf("validation failed with %d issues", len(err.Report.Issues))
}

func SyncStore(ctx context.Context, opts StoreSyncOptions) (StoreSyncResult, error) {
	store, err := cleanRequiredStore(opts.Store)
	if err != nil {
		return StoreSyncResult{}, err
	}

	git := GitSync{Store: store, Runner: opts.Runner}
	repo, err := git.Info(ctx)
	if err != nil {
		return StoreSyncResult{}, err
	}
	if !repo.IsRepository {
		return StoreSyncResult{}, errors.New("storage is not a git repository")
	}

	result := StoreSyncResult{Repository: repo}
	if !opts.SkipValidate {
		report, err := ValidateStore(store)
		if err != nil {
			return StoreSyncResult{}, err
		}
		result.Validation = report
		if !report.OK() {
			return result, ValidationFailedError{Report: report}
		}
	}

	if !opts.SkipIndex {
		index, err := BuildIndex(store)
		if err != nil {
			return StoreSyncResult{}, err
		}
		jsonPath, err := WriteIndex(store, index)
		if err != nil {
			return StoreSyncResult{}, err
		}
		sqliteInfo, err := RebuildSQLiteIndex(store)
		if err != nil {
			return StoreSyncResult{}, err
		}
		result.JSONIndexPath = jsonPath
		result.SQLiteIndex = sqliteInfo
	}

	dirty, err := git.DirtyStatus(ctx)
	if err != nil {
		return StoreSyncResult{}, err
	}
	result.DirtyBeforeCommit = dirty

	message := strings.TrimSpace(opts.Message)
	if message != "" && len(dirty) > 0 {
		pathspecs := syncCommitPathspecs(opts.Pathspecs)
		if err := git.Commit(ctx, message, pathspecs...); err != nil {
			return result, err
		}
		result.Committed = true
		result.CommitMessage = message
		result.CommitPathspecs = pathspecs
	}

	if opts.Push {
		remote := strings.TrimSpace(opts.Remote)
		branch := strings.TrimSpace(opts.Branch)
		if remote == "" && repo.RemoteURL != "" {
			remote = "origin"
		}
		if branch == "" {
			branch = repo.Branch
		}
		if remote == "" {
			branch = ""
		}
		if err := git.Push(ctx, remote, branch); err != nil {
			return result, err
		}
		result.Pushed = true
	}

	after, err := git.DirtyStatus(ctx)
	if err != nil {
		return result, err
	}
	result.DirtyAfterSync = after
	return result, nil
}

func cleanRequiredStore(store string) (string, error) {
	store = strings.TrimSpace(store)
	if store == "" {
		return "", errors.New("store is required")
	}
	return filepath.Clean(store), nil
}

func syncCommitPathspecs(pathspecs []string) []string {
	cleaned := make([]string, 0, len(pathspecs))
	for _, pathspec := range pathspecs {
		pathspec = strings.TrimSpace(pathspec)
		if pathspec != "" {
			cleaned = append(cleaned, pathspec)
		}
	}
	if len(cleaned) == 0 {
		return []string{"."}
	}
	return cleaned
}
