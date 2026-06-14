package filetxn

import (
	"path/filepath"
	"testing"
)

func TestResolveWithinRootAndRelativeToRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	resolved, err := ResolveWithinRoot(root, filepath.Join("bin", "Game.exe"))
	if err != nil {
		t.Fatal(err)
	}
	relative, err := RelativeToRoot(root, resolved)
	if err != nil || relative != filepath.Join("bin", "Game.exe") {
		t.Fatalf("RelativeToRoot() = %q, %v", relative, err)
	}
}

func TestResolveWithinRootRejectsEscape(t *testing.T) {
	t.Parallel()
	if _, err := ResolveWithinRoot(t.TempDir(), filepath.Join("..", "outside")); err == nil {
		t.Fatal("ResolveWithinRoot() error = nil")
	}
}

func TestRelativeToRootRejectsOutsidePath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := RelativeToRoot(root, filepath.Join(filepath.Dir(root), "outside")); err == nil {
		t.Fatal("RelativeToRoot() error = nil")
	}
}
