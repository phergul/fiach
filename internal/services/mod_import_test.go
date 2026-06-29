package services

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/services/dto"
)

func importFolderMod(ctx context.Context, service *ModService, gameID int64, name string, sourcePath string) (dto.Mod, error) {
	result, err := service.ImportMod(ctx, dto.ImportModInput{
		GameID:             gameID,
		Name:               name,
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		return dto.Mod{}, err
	}

	return result.Mod, nil
}

func importArchiveMod(ctx context.Context, service *ModService, gameID int64, name string, archivePath string) (dto.Mod, error) {
	result, err := service.ImportMod(ctx, dto.ImportModInput{
		GameID:             gameID,
		Name:               name,
		SourceType:         dto.ModSourceTypeArchive,
		SourcePath:         archivePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		return dto.Mod{}, err
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

func TestModServiceResolveImportSourceDuplicatesMixedBatch(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertServiceProfileTestGame(t, store, "Fallout", "/games/fallout")
	service := NewModService(store, testLogger())
	ctx := context.Background()

	existingFolder := makeSourceFolder(t, map[string]string{"Data/file.txt": "existing"})
	existingArchive := makeZipArchive(t, map[string]string{"Data/file.txt": "archive"})
	newFolder := makeSourceFolder(t, map[string]string{"Data/file.txt": "new"})
	newArchive := makeZipArchive(t, map[string]string{"Data/file.txt": "new-archive"})

	existingMod, err := importFolderMod(ctx, service, gameID, "SkyUI", existingFolder)
	if err != nil {
		t.Fatalf("importFolderMod() existing error = %v", err)
	}
	if _, err := importArchiveMod(ctx, service, otherGameID, "Other SkyUI", existingArchive); err != nil {
		t.Fatalf("importArchiveMod() other game error = %v", err)
	}

	result, err := service.ResolveImportSourceDuplicates(ctx, dto.ResolveImportSourceDuplicatesInput{
		GameID: gameID,
		Sources: []dto.ImportSourceRef{
			{SourceType: dto.ModSourceTypeFolder, SourcePath: existingFolder},
			{SourceType: dto.ModSourceTypeArchive, SourcePath: existingArchive},
			{SourceType: dto.ModSourceTypeFolder, SourcePath: newFolder},
			{SourceType: dto.ModSourceTypeArchive, SourcePath: newArchive},
			{SourceType: dto.ModSourceTypeFolder, SourcePath: "   "},
		},
	})
	if err != nil {
		t.Fatalf("ResolveImportSourceDuplicates() error = %v", err)
	}
	if len(result.Items) != 5 {
		t.Fatalf("ResolveImportSourceDuplicates() length = %d, want 5", len(result.Items))
	}
	if !result.Items[0].IsDuplicate || result.Items[0].ExistingModID == nil || *result.Items[0].ExistingModID != existingMod.ID ||
		result.Items[0].ExistingModName == nil || *result.Items[0].ExistingModName != existingMod.Name {
		t.Fatalf("ResolveImportSourceDuplicates() existing folder = %+v, want duplicate SkyUI", result.Items[0])
	}
	if result.Items[1].IsDuplicate {
		t.Fatalf("ResolveImportSourceDuplicates() archive on other game = %+v, want not duplicate for this game", result.Items[1])
	}
	if result.Items[1].CanonicalPath == "" {
		t.Fatalf("ResolveImportSourceDuplicates() archive canonical path = empty, want resolved path")
	}
	if result.Items[2].IsDuplicate || result.Items[2].CanonicalPath == "" {
		t.Fatalf("ResolveImportSourceDuplicates() new folder = %+v, want canonical non-duplicate", result.Items[2])
	}
	if result.Items[3].IsDuplicate || result.Items[3].CanonicalPath == "" {
		t.Fatalf("ResolveImportSourceDuplicates() new archive = %+v, want canonical non-duplicate", result.Items[3])
	}
	if result.Items[4].Error == nil {
		t.Fatalf("ResolveImportSourceDuplicates() invalid archive = %+v, want per-item error", result.Items[4])
	}
}

func TestModServiceResolveImportSourceDuplicatesUsesCanonicalPaths(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewModService(store, testLogger())
	ctx := context.Background()

	sourceRoot := t.TempDir()
	sourceFolder := filepath.Join(sourceRoot, "mods", "SkyUI")
	if err := os.MkdirAll(sourceFolder, 0o755); err != nil {
		t.Fatalf("create source folder: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceFolder, "Data"), 0o755); err != nil {
		t.Fatalf("create source folder data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceFolder, "Data", "file.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	existingMod, err := importFolderMod(ctx, service, gameID, "SkyUI", sourceFolder)
	if err != nil {
		t.Fatalf("importFolderMod() error = %v", err)
	}

	aliasPath := filepath.Join(sourceRoot, "mods", ".", "SkyUI")
	result, err := service.ResolveImportSourceDuplicates(ctx, dto.ResolveImportSourceDuplicatesInput{
		GameID: gameID,
		Sources: []dto.ImportSourceRef{
			{SourceType: dto.ModSourceTypeFolder, SourcePath: aliasPath},
		},
	})
	if err != nil {
		t.Fatalf("ResolveImportSourceDuplicates() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("ResolveImportSourceDuplicates() length = %d, want 1", len(result.Items))
	}
	if !result.Items[0].IsDuplicate || result.Items[0].ExistingModID == nil || *result.Items[0].ExistingModID != existingMod.ID {
		t.Fatalf("ResolveImportSourceDuplicates() = %+v, want duplicate via canonical path", result.Items[0])
	}
}
