package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/modmetadata"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestModServiceListsMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	service := NewModService(store, testLogger())

	mods, err := service.ListMods(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}
	if len(mods) != 1 || mods[0].ID != modID {
		t.Fatalf("ListMods() = %+v, want inserted mod", mods)
	}
	if mods[0].Metadata == nil || mods[0].Metadata.ModID != modID {
		t.Fatalf("ListMods() metadata = %+v, want metadata for mod %d", mods[0].Metadata, modID)
	}
}

func TestModServiceListsModsWithEditableMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/mods/skyui",
		OriginalSourcePath: "/imports/skyui",
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version: serviceStringPtr("1.0.0"),
			Author:  serviceStringPtr("Detected Author"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}
	service := NewModService(store, testLogger())
	if _, err := service.UpdateModMetadata(context.Background(), dto.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("2.0.0"),
		},
		Author: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		Description: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("Updated description"),
		},
		SourceURL: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		Notes: serviceStringPtr("Local notes"),
	}); err != nil {
		t.Fatalf("UpdateModMetadata() error = %v", err)
	}

	mods, err := service.ListMods(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}
	if len(mods) != 1 || mods[0].Metadata == nil {
		t.Fatalf("ListMods() = %+v, want one mod with metadata", mods)
	}
	metadata := mods[0].Metadata
	if metadata.Version.Detected == nil || *metadata.Version.Detected != "1.0.0" ||
		metadata.Version.Effective == nil || *metadata.Version.Effective != "2.0.0" ||
		metadata.Author.Effective == nil || *metadata.Author.Effective != "Detected Author" ||
		metadata.Notes == nil || *metadata.Notes != "Local notes" {
		t.Fatalf("ListMods() metadata = %+v, want detected, user, effective, and notes fields", metadata)
	}
}

func TestModServiceRenamesMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	service := NewModService(store, testLogger())

	renamed, err := service.RenameMod(context.Background(), modID, " USSEP ")
	if err != nil {
		t.Fatalf("RenameMod() error = %v", err)
	}
	if renamed.ID != modID || renamed.Name != "USSEP" {
		t.Fatalf("RenameMod() = %+v, want renamed mod", renamed)
	}
}

func TestModServiceGetsAndUpdatesModMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/mods/skyui",
		OriginalSourcePath: "/imports/skyui",
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version:   serviceStringPtr("1.0.0"),
			Author:    serviceStringPtr("Detected Author"),
			SourceURL: serviceStringPtr("https://example.com/skyui"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	service := NewModService(store, testLogger())
	initial, err := service.GetModMetadata(context.Background(), mod.ID)
	if err != nil {
		t.Fatalf("GetModMetadata() error = %v", err)
	}
	if initial.Version.Effective == nil || *initial.Version.Effective != "1.0.0" || initial.Version.UserSet {
		t.Fatalf("GetModMetadata() = %+v, want detected effective version", initial)
	}

	updated, err := service.UpdateModMetadata(context.Background(), dto.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr(" 2.0.0 "),
		},
		Author: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeClear,
		},
		Description: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("Better menus"),
		},
		SourceURL: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		Notes: serviceStringPtr("Local install notes"),
	})
	if err != nil {
		t.Fatalf("UpdateModMetadata() error = %v", err)
	}

	if updated.Version.User == nil || *updated.Version.User != "2.0.0" || updated.Version.Effective == nil || *updated.Version.Effective != "2.0.0" {
		t.Fatalf("Version = %+v, want trimmed user override", updated.Version)
	}
	if updated.Author.User != nil || updated.Author.Effective != nil || !updated.Author.UserSet {
		t.Fatalf("Author = %+v, want explicit clear", updated.Author)
	}
	if updated.SourceURL.Effective == nil || *updated.SourceURL.Effective != "https://example.com/skyui" || updated.SourceURL.UserSet {
		t.Fatalf("SourceURL = %+v, want reset to detected", updated.SourceURL)
	}
	if updated.Notes == nil || *updated.Notes != "Local install notes" {
		t.Fatalf("Notes = %v, want saved notes", updated.Notes)
	}
}

func TestModServiceUpdateModMetadataRejectsInvalidValuesWithoutCorruptingExistingData(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/mods/skyui",
		OriginalSourcePath: "/imports/skyui",
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	service := NewModService(store, testLogger())
	_, err = service.UpdateModMetadata(context.Background(), dto.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("1.0.0"),
		},
		Author: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		Description: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		SourceURL: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
	})
	if err != nil {
		t.Fatalf("UpdateModMetadata() setup error = %v", err)
	}

	_, err = service.UpdateModMetadata(context.Background(), dto.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		Author: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("Bad\x00Author"),
		},
		Description: dto.ModMetadataFieldUpdate{
			Mode: dto.ModMetadataFieldUpdateModeReset,
		},
		SourceURL: dto.ModMetadataFieldUpdate{
			Mode:  dto.ModMetadataFieldUpdateModeUser,
			Value: serviceStringPtr("file:///mods/skyui"),
		},
	})
	if err == nil {
		t.Fatal("UpdateModMetadata() invalid error = nil, want validation error")
	}

	metadata, err := service.GetModMetadata(context.Background(), mod.ID)
	if err != nil {
		t.Fatalf("GetModMetadata() error = %v", err)
	}
	if metadata.Version.User == nil || *metadata.Version.User != "1.0.0" || !metadata.Version.UserSet {
		t.Fatalf("metadata after invalid update = %+v, want previous version preserved", metadata)
	}
}

func TestModServiceGetModMetadataReportsMissingMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewModService(store, testLogger())
	_, err := service.GetModMetadata(context.Background(), 999)
	if err == nil {
		t.Fatal("GetModMetadata() error = nil, want missing mod error")
	}
	if !strings.Contains(err.Error(), "mod 999 was not found") {
		t.Fatalf("GetModMetadata() error = %q, want missing mod context", err.Error())
	}
}

func TestModServiceGetsDeleteSummary(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modPath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}), "SkyUI")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", modPath)
	appliedProfileID := insertServiceProfileTestProfile(t, store, gameID, "Applied")
	otherProfileID := insertServiceProfileTestProfile(t, store, gameID, "Other")
	if _, err := store.AddModToProfile(context.Background(), appliedProfileID, modID); err != nil {
		t.Fatalf("AddModToProfile() applied error = %v", err)
	}
	if _, err := store.AddModToProfile(context.Background(), otherProfileID, modID); err != nil {
		t.Fatalf("AddModToProfile() other error = %v", err)
	}
	saveServiceAppliedStateWithCurrentComposition(t, store, gameID, appliedProfileID)

	service := NewModService(store, testLogger())
	summary, err := service.GetModDeleteSummary(context.Background(), modID)
	if err != nil {
		t.Fatalf("GetModDeleteSummary() error = %v", err)
	}

	if summary.ModID != modID || summary.ModName != "SkyUI" || summary.ProfileUsageCount != 2 || !summary.IsInAppliedProfile || summary.ManagedSourcePath != modPath {
		t.Fatalf("GetModDeleteSummary() = %+v, want applied mod usage summary", summary)
	}
}

func TestModServiceDeletesManagedModFilesThenRow(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modPath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}), "SkyUI")
	if err := os.MkdirAll(modPath, 0o755); err != nil {
		t.Fatalf("create managed mod folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modPath, "mod.esp"), []byte("plugin"), 0o644); err != nil {
		t.Fatalf("write managed mod file: %v", err)
	}
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", modPath)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	if _, err := store.AddModToProfile(context.Background(), profileID, modID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}

	service := NewModService(store, testLogger())
	if err := service.DeleteMod(context.Background(), modID); err != nil {
		t.Fatalf("DeleteMod() error = %v", err)
	}

	if _, err := os.Stat(modPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("managed mod folder stat error = %v, want removed", err)
	}
	if _, found, err := store.GetMod(context.Background(), modID); err != nil || found {
		t.Fatalf("GetMod() = found %v, error %v; want deleted", found, err)
	}
	profileMods, err := store.ListProfileMods(context.Background(), profileID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 0 {
		t.Fatalf("ListProfileMods() length = %d, want cascade delete", len(profileMods))
	}
}

func TestModServiceDeleteRejectsSourcePathOutsideManagedStorage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	externalPath := makeSourceFolder(t, map[string]string{"mod.esp": "plugin"})
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", externalPath)

	service := NewModService(store, testLogger())
	err := service.DeleteMod(context.Background(), modID)
	if err == nil {
		t.Fatal("DeleteMod() error = nil, want unsafe path error")
	}
	if !strings.Contains(err.Error(), "outside managed storage") {
		t.Fatalf("DeleteMod() error = %q, want managed storage guard", err.Error())
	}

	if _, found, err := store.GetMod(context.Background(), modID); err != nil || !found {
		t.Fatalf("GetMod() after rejected delete = found %v, error %v; want row preserved", found, err)
	}
	assertFileContents(t, filepath.Join(externalPath, "mod.esp"), "plugin")
}

func TestModServiceDeleteKeepsRowWhenFileRemovalFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod delete restrictions are not reliable on Windows")
	}
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	modPath := filepath.Join(gameStoragePath, "SkyUI")
	if err := os.MkdirAll(modPath, 0o755); err != nil {
		t.Fatalf("create managed mod folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modPath, "mod.esp"), []byte("plugin"), 0o644); err != nil {
		t.Fatalf("write managed mod file: %v", err)
	}
	if err := os.Chmod(gameStoragePath, 0o555); err != nil {
		t.Fatalf("chmod managed storage: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(gameStoragePath, 0o755)
	})
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", modPath)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	if _, err := store.AddModToProfile(context.Background(), profileID, modID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}

	service := NewModService(store, testLogger())
	err := service.DeleteMod(context.Background(), modID)
	if err == nil {
		t.Fatal("DeleteMod() error = nil, want file removal error")
	}
	if !strings.Contains(err.Error(), "remove managed mod files") {
		t.Fatalf("DeleteMod() error = %q, want file removal context", err.Error())
	}

	if _, found, err := store.GetMod(context.Background(), modID); err != nil || !found {
		t.Fatalf("GetMod() after failed delete = found %v, error %v; want row preserved", found, err)
	}
	profileMods, err := store.ListProfileMods(context.Background(), profileID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 1 {
		t.Fatalf("ListProfileMods() length = %d, want association preserved", len(profileMods))
	}
}

func TestModServiceGetsManagedModStorageUsage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstModPath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
		"Data/.DS_Store": "metadata",
		".DS_Store":      "metadata",
		"readme.txt":     "hello",
	})
	secondModPath := makeSourceFolder(t, map[string]string{
		"nested/config.json": "{}",
	})
	insertServiceProfileTestMod(t, store, gameID, "SkyUI", firstModPath)
	insertServiceProfileTestMod(t, store, gameID, "Config", secondModPath)

	service := NewModService(store, testLogger())
	got, err := service.GetGameManagedModStorageUsage(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetGameManagedModStorageUsage() error = %v", err)
	}

	want := int64(len("plugin") + len("hello") + len("{}"))
	if got != want {
		t.Fatalf("GetGameManagedModStorageUsage() = %d, want %d", got, want)
	}
}

func TestModServiceManagedStorageUsageIgnoresMissingAndUnreadablePaths(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	readablePath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	missingPath := filepath.Join(t.TempDir(), "missing")
	insertServiceProfileTestMod(t, store, gameID, "SkyUI", readablePath)
	insertServiceProfileTestMod(t, store, gameID, "Missing", missingPath)

	if runtime.GOOS != "windows" {
		unreadablePath := makeSourceFolder(t, map[string]string{
			"secret.txt": "secret",
		})
		if err := os.Chmod(unreadablePath, 0o000); err != nil {
			t.Fatalf("make unreadable folder: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(unreadablePath, 0o755)
		})
		insertServiceProfileTestMod(t, store, gameID, "Unreadable", unreadablePath)
	}

	service := NewModService(store, testLogger())
	got, err := service.GetGameManagedModStorageUsage(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetGameManagedModStorageUsage() error = %v", err)
	}

	want := int64(len("plugin"))
	if got != want {
		t.Fatalf("GetGameManagedModStorageUsage() = %d, want %d", got, want)
	}
}

func TestModServiceManagedStorageUsageDoesNotFollowSymlinks(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated permissions on Windows")
	}

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modPath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "plugin",
	})
	externalPath := makeSourceFolder(t, map[string]string{
		"external.txt": "external content",
	})
	if err := os.Symlink(filepath.Join(externalPath, "external.txt"), filepath.Join(modPath, "external-file-link")); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}
	if err := os.Symlink(externalPath, filepath.Join(modPath, "external-dir-link")); err != nil {
		t.Fatalf("create directory symlink: %v", err)
	}
	insertServiceProfileTestMod(t, store, gameID, "SkyUI", modPath)

	service := NewModService(store, testLogger())
	got, err := service.GetGameManagedModStorageUsage(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetGameManagedModStorageUsage() error = %v", err)
	}

	want := int64(len("plugin"))
	if got != want {
		t.Fatalf("GetGameManagedModStorageUsage() = %d, want %d", got, want)
	}
}

func TestModServiceListsImportStrategies(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewModService(store, testLogger())
	strategies, err := service.ListImportStrategies(context.Background())
	if err != nil {
		t.Fatalf("ListImportStrategies() error = %v", err)
	}

	if len(strategies) != 2 {
		t.Fatalf("ListImportStrategies() length = %d, want 2: %+v", len(strategies), strategies)
	}
	if strategies[0].Type != dto.StrategyTypeGenericCopy || strategies[0].Visibility != dto.StrategyVisibilitySelectable {
		t.Fatalf("ListImportStrategies()[0] = %+v, want selectable generic copy", strategies[0])
	}
	if strategies[1].Type != dto.StrategyTypeUnrealPak || strategies[1].Visibility != dto.StrategyVisibilitySelectable || !strategies[1].SupportsTargetDetection {
		t.Fatalf("ListImportStrategies()[1] = %+v, want detectable selectable Unreal PAK", strategies[1])
	}
}

func TestModServiceSuggestsUnrealPakForPackageSource(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	sourcePath := makeSourceFolder(t, map[string]string{"Nested/Example_P.pak": "pak"})
	service := NewModService(store, testLogger())

	result, err := service.PreValidateImport(context.Background(), dto.PreValidateImportInput{
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("PreValidateImport() error = %v", err)
	}
	if result.SuggestedStrategy == nil || *result.SuggestedStrategy != dto.StrategyTypeUnrealPak {
		t.Fatalf("PreValidateImport() = %+v, want Unreal suggestion", result)
	}
}

func TestModServiceDetectsUnrealPakTargets(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gameRoot, "Example", "Content", "Paks"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}
	gameID := insertServiceProfileTestGame(t, store, "Example", gameRoot)
	service := NewModService(store, testLogger())

	result, err := service.DetectImportTargets(context.Background(), gameID, dto.StrategyTypeUnrealPak)
	if err != nil {
		t.Fatalf("DetectImportTargets() error = %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0] != "Example/Content/Paks/~mods" {
		t.Fatalf("DetectImportTargets() = %+v, want Unreal target", result)
	}
}

func TestModServicePreviewsAndImportsUnrealPak(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gameRoot, "Example", "Content", "Paks"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}
	gameID := insertServiceProfileTestGame(t, store, "Example", gameRoot)
	sourcePath := makeSourceFolder(t, map[string]string{
		"Nested/Example_P.ucas": "ucas",
		"Nested/Example_P.utoc": "utoc",
		"readme.txt":            "ignored",
	})
	service := NewModService(store, testLogger())
	targetPath := "Example/Content/Paks/~mods"

	preview, err := service.PreviewImportConfiguration(context.Background(), dto.PreviewImportConfigurationInput{
		GameID:             gameID,
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeUnrealPak,
		TargetRelativePath: targetPath,
	})
	if err != nil {
		t.Fatalf("PreviewImportConfiguration() error = %v", err)
	}
	wantTargets := targetPath + "/Example_P.ucas|" + targetPath + "/Example_P.utoc"
	if preview.TotalFileCount != 2 || strings.Join(preview.TargetFilePaths, "|") != wantTargets {
		t.Fatalf("PreviewImportConfiguration() = %+v, want flattened Unreal files", preview)
	}

	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Example",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeUnrealPak,
		TargetRelativePath: targetPath,
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	if result.Config.StrategyType != string(dto.StrategyTypeUnrealPak) || len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Ignored 1") {
		t.Fatalf("ImportMod() = %+v, want Unreal config and ignored-file warning", result)
	}
}

func TestModServiceWarnsForManualUnrealTarget(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gameRoot, "Example", "Content", "Paks"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}
	gameID := insertServiceProfileTestGame(t, store, "Example", gameRoot)
	sourcePath := makeSourceFolder(t, map[string]string{"Example_P.pak": "pak"})
	service := NewModService(store, testLogger())

	preview, err := service.PreviewImportConfiguration(context.Background(), dto.PreviewImportConfigurationInput{
		GameID:             gameID,
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeUnrealPak,
		TargetRelativePath: "Custom/Mods",
	})
	if err != nil {
		t.Fatalf("PreviewImportConfiguration() error = %v", err)
	}
	if len(preview.Warnings) != 1 || !strings.Contains(preview.Warnings[0], "was not detected") {
		t.Fatalf("PreviewImportConfiguration() warnings = %v, want manual target warning", preview.Warnings)
	}
}

func TestModServiceRejectsInvalidUnrealPakImport(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gameRoot, "Example", "Content", "Paks"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}
	gameID := insertServiceProfileTestGame(t, store, "Example", gameRoot)
	sourcePath := makeSourceFolder(t, map[string]string{"Example_P.ucas": "ucas"})
	service := NewModService(store, testLogger())

	_, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Example",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeUnrealPak,
		TargetRelativePath: "Example/Content/Paks/~mods",
	})
	if err == nil || !strings.Contains(err.Error(), ".utoc") {
		t.Fatalf("ImportMod() error = %v, want incomplete IoStore rejection", err)
	}
}

func TestModServiceRejectsInvalidUnrealPakUpdate(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gameRoot, "Example", "Content", "Paks"), 0o755); err != nil {
		t.Fatalf("create Paks folder: %v", err)
	}
	gameID := insertServiceProfileTestGame(t, store, "Example", gameRoot)
	initialSource := makeSourceFolder(t, map[string]string{"Example_P.pak": "pak"})
	replacementSource := makeSourceFolder(t, map[string]string{"Example_P.utoc": "utoc"})
	service := NewModService(store, testLogger())

	imported, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Example",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         initialSource,
		StrategyType:       dto.StrategyTypeUnrealPak,
		TargetRelativePath: "Example/Content/Paks/~mods",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	_, err = service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.Mod.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementSource,
	})
	if err == nil || !strings.Contains(err.Error(), ".ucas") {
		t.Fatalf("PreviewUpdateMod() error = %v, want incomplete IoStore rejection", err)
	}
	assertFileContents(t, filepath.Join(imported.Mod.SourcePath, "Example_P.pak"), "pak")
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

	service := NewModService(store, testLogger())
	mod, err := importFolderMod(context.Background(), service, gameID, " SkyUI ", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	originalPath, err := storage.CanonicalModOriginalSourcePath(sourcePath)
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	wantSourcePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}), "SkyUI")
	if mod.Name != "SkyUI" || mod.SourceType != dto.ModSourceTypeFolder || mod.SourcePath != wantSourcePath || mod.OriginalSourcePath != originalPath || mod.OriginalSourceName != nil {
		t.Fatalf("ImportMod() = %+v, want trimmed name and managed/original paths", mod)
	}
	if mod.FileCount == nil || *mod.FileCount != 2 || mod.DirectoryCount == nil || *mod.DirectoryCount != 1 || mod.TotalSizeBytes == nil || *mod.TotalSizeBytes != 11 {
		t.Fatalf("ImportMod() metadata = %+v, want inventory metadata", mod)
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

	service := NewModService(store, testLogger())
	mod, err := importArchiveMod(context.Background(), service, gameID, " SkyUI ", archivePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	originalPath, err := storage.CanonicalModOriginalSourcePath(archivePath)
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	wantSourcePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}), "SkyUI")
	if mod.Name != "SkyUI" || mod.SourceType != dto.ModSourceTypeArchive || mod.SourcePath != wantSourcePath || mod.OriginalSourcePath != originalPath {
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
	service := NewModService(store, testLogger())

	preview, err := service.PreviewImportConfiguration(context.Background(), dto.PreviewImportConfigurationInput{
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Mods/SkyUI",
	})
	if err != nil {
		t.Fatalf("PreviewImportConfiguration() error = %v", err)
	}

	wantPaths := []string{"Mods/SkyUI/Data/SkyUI.esp", "Mods/SkyUI/readme.txt"}
	if preview.TotalFileCount != 2 || preview.TotalSizeBytes != 11 || preview.TargetRelativePath != "Mods/SkyUI" || strings.Join(preview.TargetFilePaths, "|") != strings.Join(wantPaths, "|") {
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
	service := NewModService(store, testLogger())

	preview, err := service.PreviewImportConfiguration(context.Background(), dto.PreviewImportConfigurationInput{
		SourceType:         dto.ModSourceTypeArchive,
		SourcePath:         archivePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
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
	service := NewModService(store, testLogger())

	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               " SkyUI ",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	if result.Mod.Name != "SkyUI" || result.Config.ModID != result.Mod.ID || result.Config.StrategyType != string(dto.StrategyTypeGenericCopy) || result.Config.TargetBase != installconfig.TargetBaseGameRoot || result.Config.TargetRelativePath != "Data" {
		t.Fatalf("ImportMod() = %+v, want imported mod and config", result)
	}
	assertFileContents(t, filepath.Join(result.Mod.SourcePath, "Data", "SkyUI.esp"), "plugin")
}

func TestModServiceImportContinuesWhenMetadataParsingFails(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "plugin"})
	service := NewModService(store, testLogger())
	service.metadataRegistry = modmetadata.NewRegistry(failingMetadataParser{})

	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	if result.Mod.ID == 0 || result.Mod.FileCount != nil || result.Mod.DirectoryCount != nil || result.Mod.TotalSizeBytes != nil {
		t.Fatalf("ImportMod() = %+v, want imported mod with unavailable metadata", result.Mod)
	}
	assertFileContents(t, filepath.Join(result.Mod.SourcePath, "Data", "SkyUI.esp"), "plugin")
}

func TestModServiceImportPersistsDetectedEditableMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "plugin"})
	service := NewModService(store, testLogger())
	service.metadataRegistry = modmetadata.NewRegistry(staticMetadataParser{
		metadata: modmetadata.Metadata{
			Version:     serviceStringPtr("1.0.0"),
			Author:      serviceStringPtr("Mod Author"),
			Description: serviceStringPtr("User interface mod"),
			SourceURL:   serviceStringPtr("https://example.com/skyui"),
		},
	})

	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	metadata, err := service.GetModMetadata(context.Background(), result.Mod.ID)
	if err != nil {
		t.Fatalf("GetModMetadata() error = %v", err)
	}
	if metadata.Version.Detected == nil || *metadata.Version.Detected != "1.0.0" ||
		metadata.Author.Detected == nil || *metadata.Author.Detected != "Mod Author" ||
		metadata.Description.Detected == nil || *metadata.Description.Detected != "User interface mod" ||
		metadata.SourceURL.Detected == nil || *metadata.SourceURL.Detected != "https://example.com/skyui" {
		t.Fatalf("GetModMetadata() = %+v, want parser-provided detected metadata", metadata)
	}
}

func TestModServicePreviewUpdateModShowsFolderReplacementWithoutChangingPackage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	replacementPath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "new",
		"Docs/readme.md": "notes",
	})
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	originalUpdatedAt := imported.UpdatedAt

	result, err := service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementPath,
	})
	if err != nil {
		t.Fatalf("PreviewUpdateMod() error = %v", err)
	}

	if result.Mod.ID != imported.ID || result.Mod.SourcePath != imported.SourcePath || result.Mod.OriginalSourcePath != replacementPath {
		t.Fatalf("PreviewUpdateMod() mod = %+v, want same identity/path with replacement source metadata", result.Mod)
	}
	if result.Before.FileCount == nil || *result.Before.FileCount != 1 || result.After.FileCount == nil || *result.After.FileCount != 2 {
		t.Fatalf("PreviewUpdateMod() snapshots = %+v -> %+v, want before/after counts", result.Before, result.After)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "Data", "SkyUI.esp"), "old")
	if _, err := os.Stat(filepath.Join(imported.SourcePath, "Docs", "readme.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created replacement file, Stat() error = %v", err)
	}
	stored, found, err := store.GetMod(context.Background(), imported.ID)
	if err != nil {
		t.Fatalf("GetMod() error = %v", err)
	}
	if !found || stored.OriginalSourcePath != imported.OriginalSourcePath || stored.UpdatedAt != originalUpdatedAt {
		t.Fatalf("GetMod() after preview = %+v, %v; want unchanged row", stored, found)
	}
}

func TestModServicePreviewUpdateModAcceptsArchiveReplacement(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	archivePath := makeZipArchive(t, map[string]string{"SkyUI/Data/SkyUI.esp": "new"})
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	result, err := service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeArchive,
		SourcePath: archivePath,
	})
	if err != nil {
		t.Fatalf("PreviewUpdateMod() error = %v", err)
	}

	if result.Mod.SourceType != dto.ModSourceTypeArchive || result.Mod.OriginalSourceName == nil || *result.Mod.OriginalSourceName != filepath.Base(archivePath) {
		t.Fatalf("PreviewUpdateMod() = %+v, want archive source metadata", result.Mod)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "Data", "SkyUI.esp"), "old")
}

func TestModServicePreviewUpdateModValidationFailureLeavesExistingPackage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	emptyReplacement := t.TempDir()
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	_, err = service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: emptyReplacement,
	})
	if err == nil {
		t.Fatal("PreviewUpdateMod() error = nil, want validation error")
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "Data", "SkyUI.esp"), "old")
}

func TestModServicePreviewUpdateModBlocksSourcePathUsedByAnotherMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstSource := makeSourceFolder(t, map[string]string{"first.esp": "one"})
	secondSource := makeSourceFolder(t, map[string]string{"second.esp": "two"})
	service := NewModService(store, testLogger())
	first, err := importFolderMod(context.Background(), service, gameID, "First", firstSource)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	if _, err := importFolderMod(context.Background(), service, gameID, "Second", secondSource); err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	_, err = service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      first.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: secondSource,
	})
	if err == nil || !strings.Contains(err.Error(), "already used by mod") {
		t.Fatalf("PreviewUpdateMod() error = %v, want source conflict", err)
	}
	assertFileContents(t, filepath.Join(first.SourcePath, "first.esp"), "one")
}

func TestModServicePreviewUpdateModMetadataFailurePreservesPreviousMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "old"})
	replacementPath := makeSourceFolder(t, map[string]string{"mod.esp": "new", "readme.txt": "notes"})
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	if _, err := store.UpdateModDetectedMetadata(context.Background(), imported.ID, dbtypes.ModMetadataDetectedInput{
		Version: serviceStringPtr("1.0.0"),
		Author:  serviceStringPtr("Original Author"),
	}); err != nil {
		t.Fatalf("UpdateModDetectedMetadata() error = %v", err)
	}
	service.metadataRegistry = modmetadata.NewRegistry(failingMetadataParser{})

	result, err := service.PreviewUpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementPath,
	})
	if err != nil {
		t.Fatalf("PreviewUpdateMod() error = %v", err)
	}

	if result.MetadataWarning == nil || !strings.Contains(*result.MetadataWarning, "forced metadata failure") {
		t.Fatalf("MetadataWarning = %v, want parser warning", result.MetadataWarning)
	}
	if result.After.FileCount == nil || *result.After.FileCount != 1 {
		t.Fatalf("After.FileCount = %v, want previous count preserved", result.After.FileCount)
	}
	if result.After.DetectedMetadata.Version == nil || *result.After.DetectedMetadata.Version != "1.0.0" || result.After.DetectedMetadata.Author == nil || *result.After.DetectedMetadata.Author != "Original Author" {
		t.Fatalf("After.DetectedMetadata = %+v, want previous detected metadata", result.After.DetectedMetadata)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "mod.esp"), "old")
}

func TestModServiceUpdateModReplacesFolderAndPreservesProfileState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	replacementPath := makeSourceFolder(t, map[string]string{
		"Data/SkyUI.esp": "new",
		"Docs/readme.md": "notes",
	})
	service := NewModService(store, testLogger())
	imported, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	if _, err := store.AddModToProfile(context.Background(), profileID, imported.Mod.ID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}
	if _, err := store.SetProfileModEnabled(context.Background(), profileID, imported.Mod.ID, false); err != nil {
		t.Fatalf("SetProfileModEnabled() error = %v", err)
	}

	result, err := service.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.Mod.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementPath,
	})
	if err != nil {
		t.Fatalf("UpdateMod() error = %v", err)
	}

	if result.Mod.ID != imported.Mod.ID || result.Mod.Name != "SkyUI" || result.Mod.SourcePath != imported.Mod.SourcePath || result.Mod.SourceType != dto.ModSourceTypeFolder {
		t.Fatalf("UpdateMod() mod = %+v, want same identity/path/name with folder source", result.Mod)
	}
	if result.Before.FileCount == nil || *result.Before.FileCount != 1 || result.After.FileCount == nil || *result.After.FileCount != 2 {
		t.Fatalf("UpdateMod() snapshots = %+v -> %+v, want before/after counts", result.Before, result.After)
	}
	assertFileContents(t, filepath.Join(imported.Mod.SourcePath, "Data", "SkyUI.esp"), "new")
	assertFileContents(t, filepath.Join(imported.Mod.SourcePath, "Docs", "readme.md"), "notes")
	config, found, err := store.GetModInstallConfig(context.Background(), imported.Mod.ID)
	if err != nil {
		t.Fatalf("GetModInstallConfig() error = %v", err)
	}
	if !found || config.TargetRelativePath != "Data" {
		t.Fatalf("GetModInstallConfig() = %+v, %v; want preserved config", config, found)
	}
	profileMods, err := store.ListProfileMods(context.Background(), profileID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 1 || profileMods[0].ModID != imported.Mod.ID || profileMods[0].Enabled || profileMods[0].LoadOrder != 0 {
		t.Fatalf("ListProfileMods() = %+v, want preserved disabled profile membership", profileMods)
	}
}

func TestModServiceUpdateModAcceptsArchiveReplacement(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	archivePath := makeZipArchive(t, map[string]string{"SkyUI/Data/SkyUI.esp": "new"})
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	result, err := service.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeArchive,
		SourcePath: archivePath,
	})
	if err != nil {
		t.Fatalf("UpdateMod() error = %v", err)
	}

	if result.Mod.ID != imported.ID || result.Mod.SourceType != dto.ModSourceTypeArchive || result.Mod.SourcePath != imported.SourcePath {
		t.Fatalf("UpdateMod() = %+v, want same mod path with archive source", result.Mod)
	}
	if result.Mod.OriginalSourceName == nil || *result.Mod.OriginalSourceName != filepath.Base(archivePath) {
		t.Fatalf("OriginalSourceName = %v, want archive file name", result.Mod.OriginalSourceName)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "Data", "SkyUI.esp"), "new")
}

func TestModServiceUpdateModValidationFailureLeavesExistingPackage(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"Data/SkyUI.esp": "old"})
	emptyReplacement := t.TempDir()
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}

	_, err = service.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: emptyReplacement,
	})
	if err == nil {
		t.Fatal("UpdateMod() error = nil, want validation error")
	}

	stored, found, readErr := store.GetMod(context.Background(), imported.ID)
	if readErr != nil {
		t.Fatalf("GetMod() error = %v", readErr)
	}
	if !found || stored.OriginalSourcePath != imported.OriginalSourcePath {
		t.Fatalf("GetMod() = %+v, %v; want original source metadata preserved", stored, found)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "Data", "SkyUI.esp"), "old")
}

func TestModServiceUpdateModBlocksSourcePathUsedByAnotherMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstSource := makeSourceFolder(t, map[string]string{"first.esp": "one"})
	secondSource := makeSourceFolder(t, map[string]string{"second.esp": "two"})
	service := NewModService(store, testLogger())
	first, err := importFolderMod(context.Background(), service, gameID, "First", firstSource)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	second, err := importFolderMod(context.Background(), service, gameID, "Second", secondSource)
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	_, err = service.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      first.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: secondSource,
	})
	if err == nil || !strings.Contains(err.Error(), "already used by mod") {
		t.Fatalf("UpdateMod() error = %v, want source conflict", err)
	}
	assertFileContents(t, filepath.Join(first.SourcePath, "first.esp"), "one")
	assertFileContents(t, filepath.Join(second.SourcePath, "second.esp"), "two")
}

func TestModServiceUpdateModMetadataFailurePreservesPreviousMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "old"})
	replacementPath := makeSourceFolder(t, map[string]string{"mod.esp": "new", "readme.txt": "notes"})
	service := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	if _, err := store.UpdateModDetectedMetadata(context.Background(), imported.ID, dbtypes.ModMetadataDetectedInput{
		Version: serviceStringPtr("1.0.0"),
		Author:  serviceStringPtr("Original Author"),
	}); err != nil {
		t.Fatalf("UpdateModDetectedMetadata() error = %v", err)
	}
	service.metadataRegistry = modmetadata.NewRegistry(failingMetadataParser{})

	result, err := service.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementPath,
	})
	if err != nil {
		t.Fatalf("UpdateMod() error = %v", err)
	}

	if result.MetadataWarning == nil || !strings.Contains(*result.MetadataWarning, "forced metadata failure") {
		t.Fatalf("MetadataWarning = %v, want parser warning", result.MetadataWarning)
	}
	if result.After.FileCount == nil || *result.After.FileCount != 1 {
		t.Fatalf("After.FileCount = %v, want previous count preserved", result.After.FileCount)
	}
	if result.After.DetectedMetadata.Version == nil || *result.After.DetectedMetadata.Version != "1.0.0" || result.After.DetectedMetadata.Author == nil || *result.After.DetectedMetadata.Author != "Original Author" {
		t.Fatalf("After.DetectedMetadata = %+v, want previous detected metadata", result.After.DetectedMetadata)
	}
	assertFileContents(t, filepath.Join(imported.SourcePath, "mod.esp"), "new")
	assertFileContents(t, filepath.Join(imported.SourcePath, "readme.txt"), "notes")
}

func TestModServiceUpdateModMarksAppliedProfileChanged(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "old"})
	replacementPath := makeSourceFolder(t, map[string]string{"mod.esp": "new"})
	modService := NewModService(store, testLogger())
	imported, err := importFolderMod(context.Background(), modService, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() error = %v", err)
	}
	if _, err := store.AddModToProfile(context.Background(), profileID, imported.ID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}
	saveServiceAppliedStateWithCurrentComposition(t, store, gameID, profileID)

	result, err := modService.UpdateMod(context.Background(), dto.UpdateModInput{
		ModID:      imported.ID,
		SourceType: dto.ModSourceTypeFolder,
		SourcePath: replacementPath,
	})
	if err != nil {
		t.Fatalf("UpdateMod() error = %v", err)
	}
	if !result.IsInAppliedProfile || !result.RequiresReapply {
		t.Fatalf("UpdateMod() applied flags = %v/%v, want true/true", result.IsInAppliedProfile, result.RequiresReapply)
	}

	profileService := NewProfileService(store, testLogger())
	summary, err := profileService.GetAppliedProfileSummary(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileSummary() error = %v", err)
	}
	if summary == nil || summary.HasAppliedProfileChanged == nil || !*summary.HasAppliedProfileChanged {
		t.Fatalf("GetAppliedProfileSummary() = %+v, want changed after package update", summary)
	}
}

func TestModServiceImportReturnsExistingModAndConfig(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	service := NewModService(store, testLogger())

	first, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}

	second, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Renamed",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: "Other",
	})
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if second.Mod.ID != first.Mod.ID || second.Mod.Name != first.Mod.Name || second.Config.TargetRelativePath != "Data" {
		t.Fatalf("ImportMod() repeat = %+v, want existing mod/config %+v", second, first)
	}
}

type failingMetadataParser struct{}

func (failingMetadataParser) Parse(context.Context, modmetadata.ParseInput) (modmetadata.Metadata, error) {
	return modmetadata.Metadata{}, fmt.Errorf("forced metadata failure")
}

type staticMetadataParser struct {
	metadata modmetadata.Metadata
}

func (p staticMetadataParser) Parse(context.Context, modmetadata.ParseInput) (modmetadata.Metadata, error) {
	return p.metadata, nil
}

func serviceStringPtr(value string) *string {
	return &value
}

func TestModServiceImportAddsConfigToExistingUnconfiguredMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	sourcePath := makeSourceFolder(t, map[string]string{"mod.esp": "one"})
	managedPath := filepath.Join(t.TempDir(), "SkyUI")
	existing, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourceType:         dbtypes.ModSourceTypeFolder,
		SourcePath:         managedPath,
		OriginalSourcePath: sourcePath,
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	service := NewModService(store, testLogger())
	result, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Renamed",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
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
	service := NewModService(store, testLogger())

	first, err := importArchiveMod(context.Background(), service, gameID, "SkyUI", archivePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}

	second, err := importArchiveMod(context.Background(), service, gameID, "Renamed", archivePath)
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
	service := NewModService(store, testLogger())

	first, err := importFolderMod(context.Background(), service, gameID, "SkyUI", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourcePath, "new.esp"), []byte("two"), 0o644); err != nil {
		t.Fatalf("write changed source file: %v", err)
	}

	second, err := importFolderMod(context.Background(), service, gameID, "Renamed", sourcePath)
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}

	if !reflect.DeepEqual(second, first) {
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
	service := NewModService(store, testLogger())

	first, err := importFolderMod(context.Background(), service, gameID, "SkyUI", firstSourcePath)
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	second, err := importFolderMod(context.Background(), service, gameID, "SkyUI", secondSourcePath)
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

	service := NewModService(store, testLogger())
	mod, err := importFolderMod(context.Background(), service, gameID, "Linked Mod", sourcePath)
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

	service := NewModService(store, testLogger())
	_, err := importFolderMod(context.Background(), service, gameID, "Broken Link", sourcePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want broken symlink error")
	}
	if !strings.Contains(err.Error(), "read source path") {
		t.Fatalf("ImportMod() error = %q, want source path context", err.Error())
	}

	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	assertNoManagedImportArtifacts(t, gameStoragePath)
}

func TestModServiceImportValidationErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewModService(store, testLogger())
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
			_, err := importFolderMod(context.Background(), service, gameID, tt.modName, tt.sourceFolderPath)
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
	service := NewModService(store, testLogger())
	archivePath := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(archivePath, []byte("not zip"), 0o644); err != nil {
		t.Fatalf("write corrupt archive: %v", err)
	}

	_, err := importArchiveMod(context.Background(), service, gameID, "Bad Archive", archivePath)
	if err == nil {
		t.Fatal("ImportMod() error = nil, want invalid archive error")
	}
	if !strings.Contains(err.Error(), "open zip archive") {
		t.Fatalf("ImportMod() error = %q, want archive context", err.Error())
	}

	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
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

	service := NewModService(store, testLogger())
	_, err := importFolderMod(context.Background(), service, gameID, "Unreadable", sourcePath)
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
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_mod_insert
		BEFORE INSERT ON mods
		BEGIN
			SELECT RAISE(FAIL, 'forced insert failure');
		END
	`); err != nil {
		t.Fatalf("create failing insert trigger: %v", err)
	}

	service := NewModService(store, testLogger())
	_, err := importFolderMod(context.Background(), service, gameID, "DB Fail", sourcePath)
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
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_mod_insert
		BEFORE INSERT ON mods
		BEGIN
			SELECT RAISE(FAIL, 'forced insert failure');
		END
	`); err != nil {
		t.Fatalf("create failing insert trigger: %v", err)
	}

	service := NewModService(store, testLogger())
	_, err := importArchiveMod(context.Background(), service, gameID, "DB Fail", archivePath)
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
	gameStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	if _, err := store.DB().Exec(`
		CREATE TRIGGER fail_config_insert
		BEFORE INSERT ON mod_install_configs
		BEGIN
			SELECT RAISE(FAIL, 'forced config failure');
		END
	`); err != nil {
		t.Fatalf("create failing config insert trigger: %v", err)
	}

	service := NewModService(store, testLogger())
	_, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Config Fail",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
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

func TestModServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.DB().Exec(`DROP TABLE mods`); err != nil {
		t.Fatalf("drop mods table: %v", err)
	}

	service := NewModService(store, testLogger())
	_, err := service.ListMods(context.Background(), 1)
	if err == nil {
		t.Fatal("ListMods() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "list mods") || !strings.Contains(err.Error(), "select game mods") {
		t.Fatalf("ListMods() error = %q, want distinct service and storage context", err.Error())
	}

	_, err = service.GetGameManagedModStorageUsage(context.Background(), 1)
	if err == nil {
		t.Fatal("GetGameManagedModStorageUsage() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "get game managed mod storage usage") || !strings.Contains(err.Error(), "select game mods") {
		t.Fatalf("GetGameManagedModStorageUsage() error = %q, want distinct service and storage context", err.Error())
	}
}
