package services

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
)

func importFolderMod(service *ModService, gameID int64, name string, sourcePath string) (storage.Mod, error) {
	result, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               name,
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		return storage.Mod{}, err
	}

	return result.Mod, nil
}

func importArchiveMod(service *ModService, gameID int64, name string, archivePath string) (storage.Mod, error) {
	result, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               name,
		SourceType:         storage.ModSourceTypeArchive,
		SourcePath:         archivePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		return storage.Mod{}, err
	}

	return result.Mod, nil
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
