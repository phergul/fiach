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

func cleanOptionalString(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func cleanOptionalStringPath(path *string) string {
	value := cleanOptionalString(path)
	if value == "" {
		return ""
	}

	return filepath.Clean(value)
}
