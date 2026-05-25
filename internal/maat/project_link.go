package maat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LinkProjectInput struct {
	Store       string
	SourcePath  string
	ProjectKey  string
	DisplayName string
}

type LinkedProject struct {
	Project     ObjectProject `json:"project"`
	SourcePath  string        `json:"source_path"`
	RemoteURL   string        `json:"remote_url,omitempty"`
	Created     bool          `json:"created"`
	Existing    bool          `json:"existing"`
	DisplayName string        `json:"display_name"`
	ProjectKey  string        `json:"project_key"`
}

func InferProjectForPath(ctx context.Context, store, sourcePath string) (ObjectProject, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		sourcePath = "."
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return ObjectProject{}, err
	}
	remoteURL := ""
	if isRepo, err := IsGitRepository(ctx, absSource); err != nil {
		return ObjectProject{}, err
	} else if isRepo {
		remoteURL, err = GitRemoteURL(ctx, absSource)
		if err != nil {
			return ObjectProject{}, err
		}
	}

	objectStore, err := LoadObjectStore(store)
	if err != nil {
		return ObjectProject{}, err
	}
	var matches []ObjectProject
	for _, project := range objectStore.Projects {
		if remoteURL != "" && strings.TrimSpace(project.Identity["Remote"]) == remoteURL {
			matches = append(matches, project)
			continue
		}
		primaryRepo := strings.TrimSpace(project.Identity["Primary Repo"])
		if primaryRepo == "" {
			continue
		}
		absRepo, err := filepath.Abs(primaryRepo)
		if err != nil {
			continue
		}
		if pathContains(absRepo, absSource) {
			matches = append(matches, project)
		}
	}
	switch len(matches) {
	case 0:
		return ObjectProject{}, fmt.Errorf("no linked project found for %s", absSource)
	case 1:
		return matches[0], nil
	default:
		return ObjectProject{}, fmt.Errorf("multiple linked projects match %s; pass a project key explicitly", absSource)
	}
}

func LinkProject(ctx context.Context, input LinkProjectInput) (LinkedProject, error) {
	store := strings.TrimSpace(input.Store)
	if store == "" {
		return LinkedProject{}, fmt.Errorf("storage path is required")
	}
	sourcePath := strings.TrimSpace(input.SourcePath)
	if sourcePath == "" {
		sourcePath = "."
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return LinkedProject{}, err
	}
	stat, err := os.Stat(absSource)
	if err != nil {
		return LinkedProject{}, err
	}
	if !stat.IsDir() {
		return LinkedProject{}, fmt.Errorf("%s is not a directory", absSource)
	}

	remoteURL := ""
	if isRepo, err := IsGitRepository(ctx, absSource); err != nil {
		return LinkedProject{}, err
	} else if isRepo {
		remoteURL, err = GitRemoteURL(ctx, absSource)
		if err != nil {
			return LinkedProject{}, err
		}
	}

	projectKey := NormalizeIDPart(input.ProjectKey)
	if projectKey == "" {
		projectKey = projectKeyFromSource(absSource, remoteURL)
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = displayNameFromKey(projectKey)
	}
	if projectKey == "" {
		return LinkedProject{}, fmt.Errorf("project key could not be inferred")
	}

	if project, err := LoadObjectProject(store, projectKey); err == nil {
		return LinkedProject{
			Project:     project,
			SourcePath:  absSource,
			RemoteURL:   remoteURL,
			Existing:    true,
			DisplayName: project.DisplayName,
			ProjectKey:  project.Key,
		}, nil
	} else if !os.IsNotExist(err) {
		return LinkedProject{}, err
	}

	project, err := NewWriteStore(store).CreateProject(CreateProjectInput{
		Key:         projectKey,
		DisplayName: displayName,
		Summary:     fmt.Sprintf("Linked from `%s`.", absSource),
		PrimaryRepo: absSource,
		Remote:      remoteURL,
	})
	if err != nil {
		return LinkedProject{}, err
	}
	return LinkedProject{
		Project:     project,
		SourcePath:  absSource,
		RemoteURL:   remoteURL,
		Created:     true,
		DisplayName: project.DisplayName,
		ProjectKey:  project.Key,
	}, nil
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func projectKeyFromSource(sourcePath, remoteURL string) string {
	if remoteURL != "" {
		remoteURL = strings.TrimSuffix(remoteURL, "/")
		base := filepath.Base(remoteURL)
		base = strings.TrimSuffix(base, ".git")
		if key := NormalizeIDPart(base); key != "" {
			return key
		}
	}
	return NormalizeIDPart(filepath.Base(sourcePath))
}

func displayNameFromKey(key string) string {
	parts := strings.Fields(strings.ReplaceAll(key, "-", " "))
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
