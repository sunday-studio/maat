package maat

import (
	"os"
	"path/filepath"
)

const stateDirectory = "state"

func contentRoot(store string) string {
	state := filepath.Join(store, stateDirectory)
	if stat, err := os.Stat(state); err == nil && stat.IsDir() {
		return state
	}
	return store
}

func contentPath(store string, parts ...string) string {
	all := append([]string{contentRoot(store)}, parts...)
	return filepath.Join(all...)
}

func logicalRelPath(store, path string) string {
	rel := filepath.ToSlash(relPath(store, path))
	prefix := stateDirectory + "/"
	if len(rel) > len(prefix) && rel[:len(prefix)] == prefix {
		return rel[len(prefix):]
	}
	return rel
}
