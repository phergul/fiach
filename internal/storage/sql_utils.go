package storage

import (
	"path/filepath"
	"strings"
)

func cleanOptionalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	return filepath.Clean(path)
}

func nullablePath(path string) any {
	if path == "" {
		return nil
	}

	return path
}

func cleanOptionalStringPath(path *string) string {
	if path == nil {
		return ""
	}

	return cleanOptionalPath(*path)
}
