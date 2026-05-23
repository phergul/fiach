package modimport

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	name, err := NormalizeName("  SkyUI  ")
	if err != nil {
		t.Fatalf("NormalizeName() error = %v", err)
	}
	if name != "SkyUI" {
		t.Fatalf("NormalizeName() = %q, want SkyUI", name)
	}

	if _, err := NormalizeName(" \t\n "); err == nil {
		t.Fatal("NormalizeName() empty error = nil, want error")
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

func TestFolderSourceValidation(t *testing.T) {
	t.Parallel()

	sourcePath := makeSourceFolder(t, map[string]string{".hidden": "ok"})
	source, err := NewFolderSource(sourcePath)
	if err != nil {
		t.Fatalf("NewFolderSource() error = %v", err)
	}
	if err := source.Validate(); err != nil {
		t.Fatalf("Validate() hidden file error = %v", err)
	}

	emptySource, err := NewFolderSource(t.TempDir())
	if err != nil {
		t.Fatalf("NewFolderSource() empty error = %v", err)
	}
	if err := emptySource.Validate(); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("Validate() empty error = %v, want empty context", err)
	}

	filePath := filepath.Join(t.TempDir(), "mod.zip")
	if err := os.WriteFile(filePath, []byte("zip"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	fileSource, err := NewFolderSource(filePath)
	if err != nil {
		t.Fatalf("NewFolderSource() file error = %v", err)
	}
	if err := fileSource.Validate(); err == nil || !strings.Contains(err.Error(), "is not a folder") {
		t.Fatalf("Validate() file error = %v, want not folder context", err)
	}

	ignoredOnlySource, err := NewFolderSource(makeSourceFolder(t, map[string]string{".DS_Store": "metadata"}))
	if err != nil {
		t.Fatalf("NewFolderSource() ignored-only error = %v", err)
	}
	if err := ignoredOnlySource.Validate(); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("Validate() ignored-only error = %v, want empty context", err)
	}
}

func TestFolderSourceCopiesNestedFiles(t *testing.T) {
	t.Parallel()

	sourcePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"Data/.DS_Store": "metadata",
		".DS_Store":      "metadata",
		"readme.txt":     "hello",
	})
	source, err := NewFolderSource(sourcePath)
	if err != nil {
		t.Fatalf("NewFolderSource() error = %v", err)
	}
	destinationPath := t.TempDir()

	if err := source.Materialize(destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(destinationPath, "readme.txt"), "hello")
	if _, err := os.Stat(filepath.Join(destinationPath, ".DS_Store")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ignored root file stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(destinationPath, "Data", ".DS_Store")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ignored nested file stat error = %v, want not exist", err)
	}
}

func TestFolderSourceFollowsSymlinkTargets(t *testing.T) {
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
	source, err := NewFolderSource(sourcePath)
	if err != nil {
		t.Fatalf("NewFolderSource() error = %v", err)
	}

	if err := source.Materialize(destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "linked.txt"), "target")
}

func TestArchiveSourceExtractsAndStripsSingleRoot(t *testing.T) {
	t.Parallel()

	archivePath := makeZipArchive(t, map[string]string{
		"SkyUI/Data/SkyUI.esp": "plugin",
		"SkyUI/Data/.DS_Store": "metadata",
		"SkyUI/.DS_Store":      "metadata",
		"SkyUI/readme.txt":     "hello",
	})
	source, err := NewArchiveSource(archivePath)
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	destinationPath := t.TempDir()

	if err := source.Materialize(destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(destinationPath, "readme.txt"), "hello")
	if _, err := os.Stat(filepath.Join(destinationPath, ".DS_Store")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ignored archive root file stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(destinationPath, "Data", ".DS_Store")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ignored archive nested file stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(destinationPath, "SkyUI")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wrapper folder stat error = %v, want not exist", err)
	}
}

func TestArchiveSourcePreservesMultiRootLayout(t *testing.T) {
	t.Parallel()

	archivePath := makeZipArchive(t, map[string]string{
		"Data/SkyUI.esp":  "plugin",
		"Docs/readme.txt": "hello",
	})
	source, err := NewArchiveSource(archivePath)
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	destinationPath := t.TempDir()

	if err := source.Materialize(destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(destinationPath, "Docs", "readme.txt"), "hello")
}

func TestArchiveSourceRejectsInvalidArchives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      func(t *testing.T) string
		wantError string
	}{
		{
			name: "invalid extension",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "mod.rar")
				if err := os.WriteFile(path, []byte("not zip"), 0o644); err != nil {
					t.Fatalf("write invalid extension: %v", err)
				}
				return path
			},
			wantError: "is not a .zip file",
		},
		{
			name: "corrupt",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "mod.zip")
				if err := os.WriteFile(path, []byte("not zip"), 0o644); err != nil {
					t.Fatalf("write corrupt zip: %v", err)
				}
				return path
			},
			wantError: "open zip archive",
		},
		{
			name: "empty",
			path: func(t *testing.T) string {
				return makeZipArchive(t, nil)
			},
			wantError: "zip archive is empty",
		},
		{
			name: "traversal",
			path: func(t *testing.T) string {
				return makeZipArchive(t, map[string]string{"../evil.txt": "bad"})
			},
			wantError: "escapes the archive root",
		},
		{
			name: "absolute",
			path: func(t *testing.T) string {
				return makeZipArchive(t, map[string]string{"/evil.txt": "bad"})
			},
			wantError: "absolute path",
		},
		{
			name: "volume",
			path: func(t *testing.T) string {
				return makeZipArchive(t, map[string]string{`C:\evil.txt`: "bad"})
			},
			wantError: "absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source, err := NewArchiveSource(tt.path(t))
			if err != nil {
				t.Fatalf("NewArchiveSource() error = %v", err)
			}
			err = source.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestArchiveSourceRejectsSymlinkEntries(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "linked.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "linked.txt"}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatalf("create symlink entry: %v", err)
	}
	if _, err := entry.Write([]byte("target.txt")); err != nil {
		t.Fatalf("write symlink entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}

	source, err := NewArchiveSource(archivePath)
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	err = source.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want symlink error")
	}
	if !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Validate() error = %q, want symlink context", err.Error())
	}
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

func makeZipArchive(t *testing.T, files map[string]string) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "mod.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip archive: %v", err)
	}
	writer := zip.NewWriter(file)
	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	return archivePath
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
