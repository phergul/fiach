package filetxn

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func ResolveWithinRoot(root string, relativePath string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", err
	}
	relativePath = filepath.Clean(strings.TrimSpace(relativePath))
	if relativePath == "" || filepath.IsAbs(relativePath) ||
		relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", errors.New("relative path must stay inside the root")
	}
	resolved := filepath.Join(root, relativePath)
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("resolved path escapes the root")
	}
	return filepath.Clean(resolved), nil
}

func RelativeToRoot(root string, path string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside root %q", path, root)
	}
	if relative == "" {
		return ".", nil
	}
	return filepath.Clean(relative), nil
}
