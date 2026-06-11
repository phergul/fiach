package modimport

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mholt/archives"

	"github.com/phergul/fiach/internal/storage/dbtypes"
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
	if err := source.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() hidden file error = %v", err)
	}

	emptySource, err := NewFolderSource(t.TempDir())
	if err != nil {
		t.Fatalf("NewFolderSource() empty error = %v", err)
	}
	if err := emptySource.Validate(context.Background()); err == nil || !strings.Contains(err.Error(), "is empty") {
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
	if err := fileSource.Validate(context.Background()); err == nil || !strings.Contains(err.Error(), "is not a folder") {
		t.Fatalf("Validate() file error = %v, want not folder context", err)
	}

	ignoredOnlySource, err := NewFolderSource(makeSourceFolder(t, map[string]string{".DS_Store": "metadata"}))
	if err != nil {
		t.Fatalf("NewFolderSource() ignored-only error = %v", err)
	}
	if err := ignoredOnlySource.Validate(context.Background()); err == nil || !strings.Contains(err.Error(), "is empty") {
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

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
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

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
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

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
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

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(destinationPath, "Docs", "readme.txt"), "hello")
}

func TestArchiveSourceExtractsTarFormats(t *testing.T) {
	t.Parallel()

	extensions := []string{
		".tar",
		".tar.gz",
		".tgz",
		".tar.bz2",
		".tbz2",
		".tar.xz",
		".txz",
		".tar.zst",
		".tzst",
	}
	for _, extension := range extensions {
		t.Run(extension, func(t *testing.T) {
			t.Parallel()

			archivePath := makeTarArchive(t, extension, map[string]string{
				"SkyUI/Data/SkyUI.esp": "plugin",
				"SkyUI/readme.txt":     "hello",
			})
			source, err := NewArchiveSource(archivePath)
			if err != nil {
				t.Fatalf("NewArchiveSource() error = %v", err)
			}
			destinationPath := t.TempDir()

			if err := source.Materialize(context.Background(), destinationPath); err != nil {
				t.Fatalf("Materialize() error = %v", err)
			}

			assertFileContents(t, filepath.Join(destinationPath, "Data", "SkyUI.esp"), "plugin")
			assertFileContents(t, filepath.Join(destinationPath, "readme.txt"), "hello")
		})
	}
}

func TestArchiveSourceExtractsSevenZip(t *testing.T) {
	t.Parallel()

	source, err := NewArchiveSource(filepath.Join("testdata", "basic.7z"))
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	destinationPath := t.TempDir()

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertDirectoryHasRegularFile(t, destinationPath)
}

func TestArchiveSourceExtractsRAR(t *testing.T) {
	t.Parallel()

	source, err := NewArchiveSource(filepath.Join("testdata", "basic.rar"))
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	destinationPath := t.TempDir()

	if err := source.Materialize(context.Background(), destinationPath); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	assertFileContents(t, filepath.Join(destinationPath, "helloworld.txt"), "hello libarchive test suite!\n")
}

func TestArchiveSourceRejectsPasswordProtectedSevenZip(t *testing.T) {
	t.Parallel()

	source, err := NewArchiveSource(filepath.Join("testdata", "encrypted.7z"))
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}

	err = source.Materialize(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("Materialize() error = nil, want password-protected archive error")
	}
	if !strings.Contains(err.Error(), "password-protected archives are not supported") {
		t.Fatalf("Materialize() error = %q, want password-protected archive context", err.Error())
	}
}

func TestArchiveSourceRejectsMismatchedContent(t *testing.T) {
	t.Parallel()

	zipPath := makeZipArchive(t, map[string]string{"mod.txt": "content"})
	rarPath := filepath.Join(t.TempDir(), "mod.rar")
	contents, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read zip fixture: %v", err)
	}
	if err := os.WriteFile(rarPath, contents, 0o644); err != nil {
		t.Fatalf("write mismatched archive: %v", err)
	}

	source, err := NewArchiveSource(rarPath)
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	err = source.Validate(context.Background())
	if err == nil {
		t.Fatal("Validate() error = nil, want content mismatch error")
	}
	if !strings.Contains(err.Error(), "does not match file extension") {
		t.Fatalf("Validate() error = %q, want content mismatch context", err.Error())
	}
}

func TestArchiveSourceRejectsMultipartNames(t *testing.T) {
	t.Parallel()

	names := []string{"mod.part01.rar", "mod.7z.001", "mod.r00"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			archivePath := filepath.Join(t.TempDir(), name)
			if err := os.WriteFile(archivePath, []byte("archive"), 0o644); err != nil {
				t.Fatalf("write multipart archive: %v", err)
			}
			source, err := NewArchiveSource(archivePath)
			if err != nil {
				t.Fatalf("NewArchiveSource() error = %v", err)
			}

			err = source.Validate(context.Background())
			if err == nil {
				t.Fatal("Validate() error = nil, want multipart error")
			}
			if !strings.Contains(err.Error(), "multipart archives are not supported") {
				t.Fatalf("Validate() error = %q, want multipart context", err.Error())
			}
		})
	}
}

func TestArchiveSourceSuggestedNameStripsSupportedExtension(t *testing.T) {
	t.Parallel()

	names := map[string]string{
		"SkyUI.zip":     "SkyUI",
		"SkyUI.7z":      "SkyUI",
		"SkyUI.rar":     "SkyUI",
		"SkyUI.tar.gz":  "SkyUI",
		"SkyUI.TAR.ZST": "SkyUI",
		"SkyUI.tbz2":    "SkyUI",
		"SkyUI.archive": "SkyUI",
	}
	for filename, want := range names {
		t.Run(filename, func(t *testing.T) {
			t.Parallel()

			archivePath := filepath.Join(t.TempDir(), filename)
			if err := os.WriteFile(archivePath, []byte("archive"), 0o644); err != nil {
				t.Fatalf("write archive: %v", err)
			}
			source, err := NewArchiveSource(archivePath)
			if err != nil {
				t.Fatalf("NewArchiveSource() error = %v", err)
			}
			if got := source.SuggestedName(); got != want {
				t.Fatalf("SuggestedName() = %q, want %q", got, want)
			}
		})
	}
}

func TestArchiveSourceHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	source, err := NewArchiveSource(makeZipArchive(t, map[string]string{"mod.txt": "content"}))
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = source.Validate(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate() error = %v, want context canceled", err)
	}
}

func TestArchiveSourceRejectsTarHardLinks(t *testing.T) {
	t.Parallel()

	archivePath := makeTarWithHardLink(t)
	source, err := NewArchiveSource(archivePath)
	if err != nil {
		t.Fatalf("NewArchiveSource() error = %v", err)
	}

	err = source.Validate(context.Background())
	if err == nil {
		t.Fatal("Validate() error = nil, want hard link error")
	}
	if !strings.Contains(err.Error(), "is a link") {
		t.Fatalf("Validate() error = %q, want link context", err.Error())
	}
}

func TestArchiveSourceRejectsInvalidArchives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      func(t *testing.T) string
		wantError string
	}{
		{
			name: "unsupported extension",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "mod.exe")
				if err := os.WriteFile(path, []byte("not archive"), 0o644); err != nil {
					t.Fatalf("write unsupported extension: %v", err)
				}
				return path
			},
			wantError: "unsupported archive type",
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
			name: "corrupt rar",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "mod.rar")
				if err := os.WriteFile(path, []byte("not rar"), 0o644); err != nil {
					t.Fatalf("write corrupt rar: %v", err)
				}
				return path
			},
			wantError: "open RAR archive",
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
			err = source.Validate(context.Background())
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
	err = source.Validate(context.Background())
	if err == nil {
		t.Fatal("Validate() error = nil, want symlink error")
	}
	if !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Validate() error = %q, want symlink context", err.Error())
	}
}

func TestUpdateRestoresCurrentPackageWhenStorageUpdateFails(t *testing.T) {
	t.Parallel()

	gameStoragePath := t.TempDir()
	managedPath := filepath.Join(gameStoragePath, "SkyUI")
	if err := os.Mkdir(managedPath, 0o755); err != nil {
		t.Fatalf("create managed folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(managedPath, "mod.esp"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write managed file: %v", err)
	}
	replacementPath := makeSourceFolder(t, map[string]string{"mod.esp": "new"})
	source, err := NewFolderSource(replacementPath)
	if err != nil {
		t.Fatalf("NewFolderSource() error = %v", err)
	}
	store := failingUpdateStore{
		mod: dbtypes.Mod{
			ID:                 10,
			GameID:             20,
			Name:               "SkyUI",
			SourceType:         dbtypes.ModSourceTypeFolder,
			SourcePath:         managedPath,
			OriginalSourcePath: filepath.Join(t.TempDir(), "source"),
		},
		gameStoragePath: gameStoragePath,
		updateErr:       fmt.Errorf("forced update failure"),
	}

	_, err = Update(context.Background(), store, store.mod.ID, source, ImportOptions{})
	if err == nil || !strings.Contains(err.Error(), "forced update failure") {
		t.Fatalf("Update() error = %v, want storage failure", err)
	}
	assertFileContents(t, filepath.Join(managedPath, "mod.esp"), "old")
}

type failingUpdateStore struct {
	mod             dbtypes.Mod
	gameStoragePath string
	updateErr       error
}

func (s failingUpdateStore) GetMod(context.Context, int64) (dbtypes.Mod, bool, error) {
	return s.mod, true, nil
}

func (s failingUpdateStore) FindModByOriginalSourcePath(context.Context, int64, string) (dbtypes.Mod, bool, error) {
	return dbtypes.Mod{}, false, nil
}

func (s failingUpdateStore) GetGlobalModStorageRoot(context.Context) (string, error) {
	return filepath.Dir(s.gameStoragePath), nil
}

func (s failingUpdateStore) ResolveGameModStoragePath(context.Context, int64, string) (string, error) {
	return s.gameStoragePath, nil
}

func (s failingUpdateStore) GetModMetadata(context.Context, int64) (dbtypes.ModMetadata, bool, error) {
	return dbtypes.ModMetadata{ModID: s.mod.ID}, true, nil
}

func (s failingUpdateStore) GetModInstallConfig(context.Context, int64) (dbtypes.ModInstallConfig, bool, error) {
	return dbtypes.ModInstallConfig{
		ModID:              s.mod.ID,
		StrategyType:       "generic_copy",
		TargetBase:         "game_root",
		TargetRelativePath: ".",
	}, true, nil
}

func (s failingUpdateStore) UpdateModPackage(context.Context, dbtypes.UpdateModPackageInput) (dbtypes.Mod, error) {
	return dbtypes.Mod{}, s.updateErr
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

func makeTarArchive(t *testing.T, extension string, files map[string]string) string {
	t.Helper()

	var tarContents bytes.Buffer
	writer := tar.NewWriter(&tarContents)
	for name, contents := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(contents)),
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("create tar entry %q: %v", name, err)
		}
		if _, err := writer.Write([]byte(contents)); err != nil {
			t.Fatalf("write tar entry %q: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	archiveContents := tarContents.Bytes()
	if compression := tarCompressionForExtension(extension); compression != nil {
		var compressed bytes.Buffer
		compressor, err := compression.OpenWriter(&compressed)
		if err != nil {
			t.Fatalf("create %s compressor: %v", extension, err)
		}
		if _, err := compressor.Write(archiveContents); err != nil {
			t.Fatalf("compress %s archive: %v", extension, err)
		}
		if err := compressor.Close(); err != nil {
			t.Fatalf("close %s compressor: %v", extension, err)
		}
		archiveContents = compressed.Bytes()
	}

	archivePath := filepath.Join(t.TempDir(), "mod"+extension)
	if err := os.WriteFile(archivePath, archiveContents, 0o644); err != nil {
		t.Fatalf("write %s archive: %v", extension, err)
	}

	return archivePath
}

func tarCompressionForExtension(extension string) archives.Compression {
	switch extension {
	case ".tar.gz", ".tgz":
		return archives.Gz{}
	case ".tar.bz2", ".tbz2":
		return archives.Bz2{}
	case ".tar.xz", ".txz":
		return archives.Xz{}
	case ".tar.zst", ".tzst":
		return archives.Zstd{}
	default:
		return nil
	}
}

func makeTarWithHardLink(t *testing.T) string {
	t.Helper()

	var contents bytes.Buffer
	writer := tar.NewWriter(&contents)
	if err := writer.WriteHeader(&tar.Header{
		Name:     "linked.txt",
		Mode:     0o644,
		Typeflag: tar.TypeLink,
		Linkname: "target.txt",
	}); err != nil {
		t.Fatalf("create tar hard link: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "linked.tar")
	if err := os.WriteFile(archivePath, contents.Bytes(), 0o644); err != nil {
		t.Fatalf("write tar archive: %v", err)
	}
	return archivePath
}

func assertDirectoryHasRegularFile(t *testing.T, root string) {
	t.Helper()

	found := false
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			found = found || info.Mode().IsRegular()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk extracted archive: %v", err)
	}
	if !found {
		t.Fatal("extracted archive contains no regular files")
	}
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
