package maat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func BuildIndex(store string) (Index, error) {
	projects, err := LoadObjectProjects(store)
	if err != nil {
		return Index{}, err
	}
	documents, err := collectDocuments(store)
	if err != nil {
		return Index{}, err
	}
	return Index{
		Version:   1,
		Projects:  projects,
		Documents: documents,
	}, nil
}

func WriteIndex(store string, idx Index) (string, error) {
	dir := filepath.Join(store, ".maat")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "index.json")
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func collectDocuments(store string) ([]Document, error) {
	var documents []Document
	err := filepath.WalkDir(store, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			base := entry.Name()
			if base == ".git" || base == ".maat" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		documents = append(documents, Document{
			Type:    documentType(store, path),
			Path:    relPath(store, path),
			Title:   firstHeading(string(data)),
			Content: string(data),
		})
		return nil
	})
	return documents, err
}

func documentType(store, path string) string {
	rel := logicalRelPath(store, path)
	parts := strings.Split(filepath.ToSlash(rel), "/")
	switch {
	case isTargetProjectFile(parts):
		return "project"
	case isTargetObjectFile(parts, "goals"):
		return "goal"
	case isTargetObjectFile(parts, "tickets"):
		return "ticket"
	case isTargetEventFile(parts):
		return "event"
	case isTargetObjectFile(parts, "decisions"):
		return "decision"
	case isTargetObjectFile(parts, "reports"):
		return "report"
	case strings.HasPrefix(rel, "decisions/"):
		return "decision"
	case strings.HasPrefix(rel, "reports/"):
		return "report"
	case strings.HasPrefix(rel, "agents/"):
		return "agent"
	case strings.HasPrefix(rel, "docs/"):
		return "doc"
	default:
		return "markdown"
	}
}

func isTargetProjectFile(parts []string) bool {
	return len(parts) == 3 &&
		parts[0] == "projects" &&
		parts[2] == "project.md"
}

func isTargetObjectFile(parts []string, dir string) bool {
	return len(parts) == 4 &&
		parts[0] == "projects" &&
		parts[2] == dir &&
		strings.EqualFold(filepath.Ext(parts[3]), ".md")
}

func isTargetEventFile(parts []string) bool {
	return len(parts) >= 6 &&
		parts[0] == "projects" &&
		parts[2] == "events" &&
		strings.EqualFold(filepath.Ext(parts[len(parts)-1]), ".md")
}

func firstHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
