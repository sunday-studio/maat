package maat

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Search(store, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil, nil
	}
	var results []SearchResult
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
		lines := strings.Split(string(data), "\n")
		title := firstHeading(string(data))
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), query) {
				results = append(results, SearchResult{
					Type:    documentType(store, path),
					Path:    relPath(store, path),
					Line:    i + 1,
					Title:   title,
					Excerpt: strings.TrimSpace(line),
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Path == results[j].Path {
			return results[i].Line < results[j].Line
		}
		return results[i].Path < results[j].Path
	})
	return results, nil
}
