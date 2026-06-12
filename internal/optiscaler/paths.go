package optiscaler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveWithinRoot(root string, relativePath string) (resolved string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve path inside game root: %w", err)
		}
	}()

	root, err = filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", err
	}
	relativePath = filepath.Clean(strings.TrimSpace(relativePath))
	if relativePath == "" || filepath.IsAbs(relativePath) ||
		relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", errors.New("relative path must stay inside the game root")
	}
	resolved = filepath.Join(root, relativePath)
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("resolved path escapes the game root")
	}
	return filepath.Clean(resolved), nil
}

func RelativeToRoot(root string, path string) (relative string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("make path relative to game root: %w", err)
		}
	}()

	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err = filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path is outside the game root")
	}
	if relative == "" {
		return ".", nil
	}
	return filepath.Clean(relative), nil
}

func requireNoSymlinkComponents(root string, path string) error {
	relative, err := RelativeToRoot(root, path)
	if err != nil {
		return err
	}
	current := filepath.Clean(root)
	if relative == "." {
		return nil
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("inspect path component %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %q is a symbolic link", current)
		}
	}
	return nil
}
