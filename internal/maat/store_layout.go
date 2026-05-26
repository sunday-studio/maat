package maat

import "path/filepath"

func contentPath(store string, parts ...string) string {
	all := append([]string{store}, parts...)
	return filepath.Join(all...)
}

func logicalRelPath(store, path string) string {
	return filepath.ToSlash(relPath(store, path))
}
