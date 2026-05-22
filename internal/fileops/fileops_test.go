package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirePathWithinRoot(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := RequirePathWithinRoot("target", root, root); err != nil {
		t.Fatalf("RequirePathWithinRoot(root) error = %v", err)
	}
	if err := RequirePathWithinRoot("target", filepath.Join(root, "child", "file.txt"), root); err != nil {
		t.Fatalf("RequirePathWithinRoot(child) error = %v", err)
	}
	if err := RequirePathWithinRoot("target", filepath.Join(parent, "root-sibling"), root); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("RequirePathWithinRoot(sibling) error = %v, want outside-root error", err)
	}
}

func TestFileIntegrityAndMatches(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	hash, size, err := FileIntegrity(path)
	if err != nil {
		t.Fatalf("FileIntegrity() error = %v", err)
	}
	if hash != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" || size != 5 {
		t.Fatalf("FileIntegrity() = %q, %d, want SHA-256 and size", hash, size)
	}

	matches, err := FileMatchesIntegrity(path, hash, size)
	if err != nil {
		t.Fatalf("FileMatchesIntegrity() error = %v", err)
	}
	if !matches {
		t.Fatalf("FileMatchesIntegrity() = false, want true")
	}
}

func TestCopyFileAtomicCreateOnlyAndReplace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourcePath := filepath.Join(root, "source.txt")
	targetPath := filepath.Join(root, "target.txt")
	if err := os.WriteFile(sourcePath, []byte("first"), 0o640); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	if err := CopyFileAtomic(AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mode:       0o640,
	}); err != nil {
		t.Fatalf("CopyFileAtomic(create) error = %v", err)
	}
	assertFileopsContents(t, targetPath, "first")

	if err := CopyFileAtomic(AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mode:       0o640,
	}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CopyFileAtomic(create existing) error = %v, want already exists", err)
	}

	if err := os.WriteFile(sourcePath, []byte("second"), 0o640); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := CopyFileAtomic(AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mode:       0o640,
		Replace:    true,
	}); err != nil {
		t.Fatalf("CopyFileAtomic(replace) error = %v", err)
	}
	assertFileopsContents(t, targetPath, "second")
}

func TestRemoveEmptyParentDirectoriesStopsAtRoot(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "root")
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := RemoveEmptyParentDirectories(deep, root); err != nil {
		t.Fatalf("RemoveEmptyParentDirectories() error = %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root was removed or unreadable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a")); !os.IsNotExist(err) {
		t.Fatalf("Stat(root/a) error = %v, want missing", err)
	}
}

func assertFileopsContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", path, contents, want)
	}
}
