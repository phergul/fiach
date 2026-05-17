package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
)

func TestModServiceListsMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	service := NewModService(store)

	mods, err := service.ListMods(gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}
	if len(mods) != 1 || mods[0].ID != modID {
		t.Fatalf("ListMods() = %+v, want inserted mod", mods)
	}
}

func TestModServiceListsImportStrategies(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewModService(store)
	strategies, err := service.ListImportStrategies()
	if err != nil {
		t.Fatalf("ListImportStrategies() error = %v", err)
	}

	if len(strategies) != 1 {
		t.Fatalf("ListImportStrategies() length = %d, want 1: %+v", len(strategies), strategies)
	}
	if strategies[0].Type != installconfig.StrategyTypeGenericCopy || strategies[0].Visibility != installconfig.StrategyVisibilitySelectable {
		t.Fatalf("ListImportStrategies()[0] = %+v, want selectable generic copy", strategies[0])
	}
}

func TestModServiceImportsModFolder(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"readme.txt":     "hello",
	})

	service := NewModService(store)
	mod, err := importFolderMod(service, gameID, " SkyUI ", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	originalPath, err := storage.CanonicalModOriginalSourcePath(sourcePath)
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	wantSourcePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}), "SkyUI")
	if mod.Name != "SkyUI" || mod.SourceType != storage.ModSourceTypeFolder || mod.SourcePath != wantSourcePath || mod.OriginalSourcePath != originalPath || mod.OriginalSourceName != nil {
		t.Fatalf("ImportMod() = %+v, want trimmed name and managed/original paths", mod)
	}
	assertFileContents(t, filepath.Join(mod.SourcePath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(mod.SourcePath, "readme.txt"), "hello")
}

func TestModServiceImportsModArchive(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	archivePath := makeZipArchive(t, map[string]string{
		"SkyUI/Data/SkyUI.esp": "plugin",
		"SkyUI/readme.txt":     "hello",
	})

	service := NewModService(store)
	mod, err := importArchiveMod(service, gameID, " SkyUI ", archivePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	originalPath, err := storage.CanonicalModOriginalSourcePath(archivePath)
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	wantSourcePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}), "SkyUI")
	if mod.Name != "SkyUI" || mod.SourceType != storage.ModSourceTypeArchive || mod.SourcePath != wantSourcePath || mod.OriginalSourcePath != originalPath {
		t.Fatalf("ImportMod() = %+v, want archive metadata and managed/original paths", mod)
	}
	if mod.OriginalSourceName == nil || *mod.OriginalSourceName != filepath.Base(archivePath) {
		t.Fatalf("OriginalSourceName = %v, want archive filename", mod.OriginalSourceName)
	}
	assertFileContents(t, filepath.Join(mod.SourcePath, "Data", "SkyUI.esp"), "plugin")
	assertFileContents(t, filepath.Join(mod.SourcePath, "readme.txt"), "hello")
	if _, err := os.Stat(filepath.Join(mod.SourcePath, "SkyUI")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("wrapper folder stat error = %v, want not exist", err)
	}
}

func TestModServicePreviewsFolderImportConfiguration(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	sourcePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"readme.txt":     "hello",
	})
	service := NewModService(store)

	preview, err := service.PreviewImportConfiguration(PreviewImportConfigurationInput{
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Mods/SkyUI",
	})
	if err != nil {
		t.Fatalf("PreviewImportConfiguration() error = %v", err)
	}

	wantPaths := []string{"Mods/SkyUI/Data/SkyUI.esp", "Mods/SkyUI/readme.txt"}
	if preview.TotalFileCount != 2 || preview.TargetRelativePath != "Mods/SkyUI" || strings.Join(preview.TargetFilePaths, "|") != strings.Join(wantPaths, "|") {
		t.Fatalf("PreviewImportConfiguration() = %+v, want mapped target paths %+v", preview, wantPaths)
	}
}

func TestModServicePreviewsArchiveWithImportLayout(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	archivePath := makeZipArchive(t, map[string]string{
		"SkyUI/Data/SkyUI.esp": "plugin",
		"SkyUI/readme.txt":     "hello",
	})
	service := NewModService(store)

	preview, err := service.PreviewImportConfiguration(PreviewImportConfigurationInput{
		SourceType:         storage.ModSourceTypeArchive,
		SourcePath:         archivePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
	})
	if err != nil {
		t.Fatalf("PreviewImportConfiguration() error = %v", err)
	}

	wantPaths := []string{"Data/SkyUI.esp", "readme.txt"}
	if preview.TargetDisplayPath != "Game root" || strings.Join(preview.TargetFilePaths, "|") != strings.Join(wantPaths, "|") {
		t.Fatalf("PreviewImportConfiguration() = %+v, want stripped archive paths %+v", preview, wantPaths)
	}
}

func TestModServiceImportsMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "plugin"})
	service := NewModService(store)

	result, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               " SkyUI ",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	if result.Mod.Name != "SkyUI" || result.Config.ModID != result.Mod.ID || result.Config.StrategyType != string(installconfig.StrategyTypeGenericCopy) || result.Config.TargetBase != installconfig.TargetBaseGameRoot || result.Config.TargetRelativePath != "Data" {
		t.Fatalf("ImportMod() = %+v, want imported mod and config", result)
	}
	assertFileContents(t, filepath.Join(result.Mod.SourcePath, "Data", "SkyUI.esp"), "plugin")
}

func TestModServiceImportReturnsExistingModAndConfig(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	service := NewModService(store)

	first, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}

	second, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               "Renamed",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Other",
	})
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if second.Mod.ID != first.Mod.ID || second.Mod.Name != first.Mod.Name || second.Config.TargetRelativePath != "Data" {
		t.Fatalf("ImportMod() repeat = %+v, want existing mod/config %+v", second, first)
	}
}

func TestModServiceImportAddsConfigToExistingUnconfiguredMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	managedPath := filepath.Join(t.TempDir(), "SkyUI")
	existing, err := store.CreateMod(context.Background(), storage.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         managedPath,
		OriginalSourcePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	service := NewModService(store)
	result, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               "Renamed",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	if result.Mod.ID != existing.ID || result.Config.ModID != existing.ID || result.Config.TargetRelativePath != "Data" {
		t.Fatalf("ImportMod() = %+v, want existing mod with created config", result)
	}
}

func TestModServiceImportReturnsExistingModForRepeatedArchivePath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	archivePath := makeZipArchive(t, map[string]string{"SkyUI/mod.esp": "one"})
	service := NewModService(store)

	first, err := importArchiveMod(service, gameID, "SkyUI", archivePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}

	second, err := importArchiveMod(service, gameID, "Renamed", archivePath)
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if second.ID != first.ID || second.SourcePath != first.SourcePath || second.OriginalSourcePath != first.OriginalSourcePath || second.Name != first.Name {
		t.Fatalf("second import = %+v, want existing %+v", second, first)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(first.SourcePath), "Renamed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("renamed destination stat error = %v, want not exist", err)
	}
}

func TestModServiceImportReturnsExistingModForRepeatedOriginalSource(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	service := NewModService(store)

	first, err := importFolderMod(service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "new.esp"), []byte("two"), 0o644); err != nil {
		t.Fatalf("write changed source file: %v", err)
	}

	second, err := importFolderMod(service, gameID, "Renamed", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if second != first {
		t.Fatalf("second import = %+v, want existing %+v", second, first)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(first.SourcePath), "Renamed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("renamed destination stat error = %v, want not exist", err)
	}
}

func TestModServiceImportCreatesUniqueManagedFolderNames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstSourcePath := makeSourceFolder(t, map[string]string{"first.esp": "one"})
	secondSourcePath := makeSourceFolder(t, map[string]string{"second.esp": "two"})
	service := NewModService(store)

	first, err := importFolderMod(service, gameID, "SkyUI", firstSourcePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	second, err := importFolderMod(service, gameID, "SkyUI", secondSourcePath)
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if filepath.Base(first.SourcePath) != "SkyUI" {
		t.Fatalf("first SourcePath = %q, want SkyUI folder", first.SourcePath)
	}
	if filepath.Base(second.SourcePath) != "SkyUI-2" {
		t.Fatalf("second SourcePath = %q, want SkyUI-2 folder", second.SourcePath)
	}
}

func TestModServiceImportFollowsSymlinkTargets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := t.TempDir()
	targetPath := filepath.Join(t.TempDir(), "target.txt")
	if err := os.WriteFile(targetPath, []byte("target"), 0o644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(targetPath, filepath.Join(sourcePath, "linked.txt")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	service := NewModService(store)
	mod, err := importFolderMod(service, gameID, "Linked Mod", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	assertFileContents(t, filepath.Join(mod.SourcePath, "linked.txt"), "target")
}

func TestModServiceImportBrokenSymlinkFailsAndCleansTempFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := t.TempDir()
	if err := os.Symlink(filepath.Join(sourcePath, "missing.txt"), filepath.Join(sourcePath, "broken.txt")); err != nil {
		t.Fatalf("create broken symlink: %v", err)
	}

	service := NewModService(store)
	_, err := importFolderMod(service, gameID, "Broken Link", sourcePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want broken symlink error")
	}
	if !strings.Contains(err.Error(), "read source path") {
		t.Fatalf("ImportMod() error = %q, want source path context", err.Error())
	}

	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
	assertNoManagedImportArtifacts(t, gameStoragePath)
}

func TestModServiceImportValidationErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewModService(store)
	filePath := filepath.Join(t.TempDir(), "mod.zip")
	if err := os.WriteFile(filePath, []byte("zip"), 0o644); err != nil {
		t.Fatalf("write file source: %v", err)
	}

	tests := []struct {
		name             string
		modName          string
		sourceFolderPath string
		wantError        string
	}{
		{
			name:             "missing name",
			modName:          " ",
			sourceFolderPath: makeSourceFolder(t, map[string]string{"mod.esp": "one"}),
			wantError:        "mod name is required",
		},
		{
			name:             "missing folder",
			modName:          "Missing",
			sourceFolderPath: filepath.Join(t.TempDir(), "missing"),
			wantError:        "read source folder",
		},
		{
			name:             "file instead of folder",
			modName:          "File",
			sourceFolderPath: filePath,
			wantError:        "is not a folder",
		},
		{
			name:             "empty folder",
			modName:          "Empty",
			sourceFolderPath: t.TempDir(),
			wantError:        "is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := importFolderMod(service, gameID, tt.modName, tt.sourceFolderPath)
			if err == nil {
				t.Fatal("ImportMod() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ImportMod() error = %q, want %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestModServiceImportArchiveValidationErrorsCleanManagedStorage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewModService(store)
	archivePath := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(archivePath, []byte("not zip"), 0o644); err != nil {
		t.Fatalf("write corrupt archive: %v", err)
	}

	_, err := importArchiveMod(service, gameID, "Bad Archive", archivePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want invalid archive error")
	}
	if !strings.Contains(err.Error(), "open zip archive") {
		t.Fatalf("ImportMod() error = %q, want archive context", err.Error())
	}

	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
	assertNoManagedImportArtifacts(t, gameStoragePath)
}

func TestModServiceImportUnreadableFolderReturnsClearError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod read restrictions are not reliable on Windows")
	}
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	if err := os.Chmod(sourcePath, 0o000); err != nil {
		t.Fatalf("chmod source folder: %v", err)
	}
	defer func() {
		_ = os.Chmod(sourcePath, 0o755)
	}()

	service := NewModService(store)
	_, err := importFolderMod(service, gameID, "Unreadable", sourcePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want unreadable folder error")
	}
	if !strings.Contains(err.Error(), "read source folder entries") {
		t.Fatalf("ImportMod() error = %q, want readable folder context", err.Error())
	}
}

func TestModServiceImportDatabaseFailureCleansManagedFolder(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_mod_insert
		BEFORE INSERT ON mods
		BEGIN
			SELECT RAISE(FAIL, 'forced insert failure');
		END
	`); err != nil {
		t.Fatalf("create failing insert trigger: %v", err)
	}

	service := NewModService(store)
	_, err := importFolderMod(service, gameID, "DB Fail", sourcePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want database error")
	}
	if !strings.Contains(err.Error(), "insert mod with install config rows") {
		t.Fatalf("ImportMod() error = %q, want storage insert context", err.Error())
	}
	if _, err := os.Stat(filepath.Join(gameStoragePath, "DB Fail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("managed folder stat error = %v, want cleaned destination", err)
	}
}

func TestModServiceImportArchiveDatabaseFailureCleansManagedFolder(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	archivePath := makeZipArchive(t, map[string]string{"mod.esp": "one"})
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_mod_insert
		BEFORE INSERT ON mods
		BEGIN
			SELECT RAISE(FAIL, 'forced insert failure');
		END
	`); err != nil {
		t.Fatalf("create failing insert trigger: %v", err)
	}

	service := NewModService(store)
	_, err := importArchiveMod(service, gameID, "DB Fail", archivePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want database error")
	}
	if !strings.Contains(err.Error(), "insert mod with install config rows") {
		t.Fatalf("ImportMod() error = %q, want storage insert context", err.Error())
	}
	if _, err := os.Stat(filepath.Join(gameStoragePath, "DB Fail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("managed folder stat error = %v, want cleaned destination", err)
	}
}

func TestModServiceImportConfigFailureCleansManagedFolder(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_config_insert
		BEFORE INSERT ON mod_install_configs
		BEGIN
			SELECT RAISE(FAIL, 'forced config failure');
		END
	`); err != nil {
		t.Fatalf("create failing config insert trigger: %v", err)
	}

	service := NewModService(store)
	_, err := service.ImportMod(ImportModInput{
		GameID:             gameID,
		Name:               "Config Fail",
		SourceType:         storage.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       installconfig.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err == nil {
		t.Fatal("ImportMod() error = nil, want config error")
	}
	if !strings.Contains(err.Error(), "insert mod with install config rows") {
		t.Fatalf("ImportMod() error = %q, want install config storage context", err.Error())
	}
	if _, err := os.Stat(filepath.Join(gameStoragePath, "Config Fail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("managed folder stat error = %v, want cleaned destination", err)
	}

	var count int
	if err := store.DB().Get(&count, `SELECT COUNT(*) FROM mods WHERE name = 'Config Fail'`); err != nil {
		t.Fatalf("count config fail mods: %v", err)
	}
	if count != 0 {
		t.Fatalf("config fail mod count = %d, want transaction rollback", count)
	}
}

func TestModServiceReturnsStorageConfigurationError(t *testing.T) {
	t.Parallel()

	service := NewModService(nil)

	_, err := service.ListMods(1)
	if err == nil {
		t.Fatal("ListMods() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "list mods") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("ListMods() error = %q, want service context", err.Error())
	}

	_, err = importFolderMod(service, 1, "SkyUI", "/mods/skyui")
	if err == nil {
		t.Fatal("ImportMod() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "import mod") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("ImportMod() error = %q, want service context", err.Error())
	}

	_, err = importArchiveMod(service, 1, "SkyUI", "/mods/skyui.zip")
	if err == nil {
		t.Fatal("ImportMod() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "import mod") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("ImportMod() error = %q, want service context", err.Error())
	}
}

func TestModServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.DB().Exec(`DROP TABLE mods`); err != nil {
		t.Fatalf("drop mods table: %v", err)
	}

	service := NewModService(store)
	_, err := service.ListMods(1)
	if err == nil {
		t.Fatal("ListMods() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "list mods") || !strings.Contains(err.Error(), "select game mods") {
		t.Fatalf("ListMods() error = %q, want distinct service and storage context", err.Error())
	}
}
