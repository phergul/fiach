package services

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeImportedModName(t *testing.T) {
	t.Parallel()

	name, err := normalizeImportedModName("  SkyUI  ")
	if err != nil {
		t.Fatalf("normalizeImportedModName() error = %v", err)
	}
	if name != "SkyUI" {
		t.Fatalf("normalizeImportedModName() = %q, want SkyUI", name)
	}

	if _, err := normalizeImportedModName(" \t\n "); err == nil {
		t.Fatal("normalizeImportedModName() empty error = nil, want error")
	}
}

func TestManagedModFolderNameSanitizesName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "trims whitespace and dots",
			in:   "  SkyUI.  ",
			want: "SkyUI",
		},
		{
			name: "replaces unsafe filename characters",
			in:   `Better: Mod/Pack?`,
			want: "Better- Mod-Pack",
		},
		{
			name: "collapses repeated separators",
			in:   `A///B:::C`,
			want: "A-B-C",
		},
		{
			name: "falls back for empty sanitized name",
			in:   `<>:"/\|?*`,
			want: "mod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := managedModFolderName(tt.in); got != tt.want {
				t.Fatalf("managedModFolderName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPathContains(t *testing.T) {
	t.Parallel()

	root := filepath.Join("tmp", "mods")
	child := filepath.Join(root, "game", "SkyUI")
	sibling := filepath.Join("tmp", "mods-other", "SkyUI")

	if !pathContains(root, root) {
		t.Fatal("pathContains() same path = false, want true")
	}
	if !pathContains(child, root) {
		t.Fatal("pathContains() child = false, want true")
	}
	if pathContains(root, child) {
		t.Fatal("pathContains() parent in child = true, want false")
	}
	if pathContains(sibling, root) {
		t.Fatal("pathContains() sibling prefix = true, want false")
	}
}

func TestUniqueManagedModDestination(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	if err := os.Mkdir(filepath.Join(parent, "SkyUI"), 0o755); err != nil {
		t.Fatalf("create existing destination: %v", err)
	}
	if err := os.Mkdir(filepath.Join(parent, "SkyUI-2"), 0o755); err != nil {
		t.Fatalf("create second existing destination: %v", err)
	}

	path, err := uniqueManagedModDestination(parent, "SkyUI")
	if err != nil {
		t.Fatalf("uniqueManagedModDestination() error = %v", err)
	}
	if filepath.Base(path) != "SkyUI-3" {
		t.Fatalf("uniqueManagedModDestination() = %q, want SkyUI-3", path)
	}
}

func TestMakeImportTempDirCreatesHiddenFolder(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	tempPath, err := makeImportTempDir(parent, "SkyUI")
	if err != nil {
		t.Fatalf("makeImportTempDir() error = %v", err)
	}

	if !strings.HasPrefix(filepath.Base(tempPath), ".SkyUI-tmp-") {
		t.Fatalf("makeImportTempDir() = %q, want hidden SkyUI temp prefix", tempPath)
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", tempPath, err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(%q).IsDir() = false, want true", tempPath)
	}
}

func TestValidateImportSourceFolder(t *testing.T) {
	t.Parallel()

	sourcePath := makeSourceFolder(t, map[string]string{".hidden": "ok"})
	if err := validateImportSourceFolder(sourcePath); err != nil {
		t.Fatalf("validateImportSourceFolder() hidden file error = %v", err)
	}

	emptyPath := t.TempDir()
	if err := validateImportSourceFolder(emptyPath); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("validateImportSourceFolder() empty error = %v, want empty context", err)
	}

	filePath := filepath.Join(t.TempDir(), "mod.zip")
	if err := os.WriteFile(filePath, []byte("zip"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := validateImportSourceFolder(filePath); err == nil || !strings.Contains(err.Error(), "is not a folder") {
		t.Fatalf("validateImportSourceFolder() file error = %v, want not folder context", err)
	}
}

func TestCopyImportFolderCopiesNestedFiles(t *testing.T) {
	t.Parallel()

	sourcePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"readme.txt":     "hello",
	})
	destinationPath := t.TempDir()

	if err := copyImportFolder(sourcePath, destinationPath); err != nil {
		t.Fatalf("copyImportFolder() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(destinationPath, "readme.txt"), "hello")
}

func TestCopyImportFolderFollowsSymlinkTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	t.Parallel()

	sourcePath := t.TempDir()
	destinationPath := t.TempDir()
	targetPath := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(targetPath, []byte("target"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(targetPath, filepath.Join(sourcePath, "linked.txt")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if err := copyImportFolder(sourcePath, destinationPath); err != nil {
		t.Fatalf("copyImportFolder() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "linked.txt"), "target")
}

func makeSourceFolder(t *testing.T, files map[string]string) string {
	t.Helper()

	sourcePath := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(sourcePath, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create source folder parent: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write source file: %v", err)
		}
	}

	return sourcePath
}

func assertFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", path, string(contents), want)
	}
}

func assertNoManagedImportArtifacts(t *testing.T, gameStoragePath string) {
	t.Helper()

	entries, err := os.ReadDir(gameStoragePath)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", gameStoragePath, err)
	}
	if len(entries) != 0 {
		t.Fatalf("managed storage entries after failed import = %v, want none", entries)
	}
}
