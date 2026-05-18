package installpath

import (
	"path"
	"path/filepath"
	"strings"
)

// ResolveSourceRoot returns the effective source root for an install strategy.
func ResolveSourceRoot(managedSourcePath string, sourceSubpath *string) string {
	root := strings.TrimSpace(managedSourcePath)
	if sourceSubpath == nil {
		return root
	}

	subpath := strings.TrimSpace(*sourceSubpath)
	if subpath == "" || subpath == "." {
		return root
	}

	return filepath.Join(root, filepath.FromSlash(subpath))
}

// JoinTargetRelativePath maps a source-relative path onto a target-relative root.
func JoinTargetRelativePath(targetRelativePath string, sourceRelativePath string) string {
	cleanSourceRelativePath := strings.TrimPrefix(path.Clean(sourceRelativePath), "/")
	if targetRelativePath == "." {
		return cleanSourceRelativePath
	}

	return path.Join(targetRelativePath, cleanSourceRelativePath)
}
