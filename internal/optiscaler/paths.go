package optiscaler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/filetxn"
)

func ResolveWithinRoot(root string, relativePath string) (resolved string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("resolve path inside game root: %w", err)
		}
	}()

	return filetxn.ResolveWithinRoot(root, relativePath)
}

func RelativeToRoot(root string, path string) (relative string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("make path relative to game root: %w", err)
		}
	}()

	return filetxn.RelativeToRoot(root, path)
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
	for part := range strings.SplitSeq(relative, string(filepath.Separator)) {
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
