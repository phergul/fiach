package deployment

import (
	"path"
	"strings"
)

// CanonicalGameRelativePath returns a case-folded game-relative path key.
func CanonicalGameRelativePath(gameRelativePath string) string {
	cleaned := strings.TrimPrefix(path.Clean(filepathToSlash(gameRelativePath)), "/")
	if cleaned == "." {
		return ""
	}
	return strings.ToLower(cleaned)
}

func filepathToSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

// IsStrictPathPrefix reports whether parent is a strict directory prefix of child.
func IsStrictPathPrefix(parent string, child string) bool {
	if parent == "" || child == "" || parent == child {
		return false
	}
	return strings.HasPrefix(child, parent+"/")
}
