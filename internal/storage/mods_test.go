package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestListModsReturnsGameModsOrderedByName(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	insertProfileTestMod(t, store, gameID, "Zoo Mod", "/mods/zoo")
	alpha := insertProfileTestMod(t, store, gameID, "alpha Mod", "/mods/alpha")
	insertProfileTestMod(t, store, otherGameID, "Other Mod", "/mods/other")

	mods, err := store.ListMods(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}

	if len(mods) != 2 {
		t.Fatalf("ListMods() length = %d, want 2: %+v", len(mods), mods)
	}
	if mods[0].ID != alpha || mods[0].Name != "alpha Mod" || mods[1].Name != "Zoo Mod" {
		t.Fatalf("ListMods() = %+v, want alpha then Zoo", mods)
	}
	if mods[0].OriginalSourcePath == "" {
		t.Fatalf("OriginalSourcePath = empty, want imported source path")
	}
}

func TestFindModByOriginalSourcePathUsesCanonicalPathAndGame(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	originalPath, err := CanonicalModOriginalSourcePath(filepath.Join(t.TempDir(), "mods", "..", "SkyUI"))
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	modID := insertProfileTestModWithOriginalSource(t, store, gameID, "SkyUI", "/managed/skyrim/SkyUI", originalPath)
	insertProfileTestModWithOriginalSource(t, store, otherGameID, "SkyUI", "/managed/fallout/SkyUI", originalPath)

	foundMod, found, err := store.FindModByOriginalSourcePath(context.Background(), gameID, originalPath)
	if err != nil {
		t.Fatalf("FindModByOriginalSourcePath() error = %v", err)
	}
	if !found || foundMod.ID != modID || foundMod.OriginalSourcePath != originalPath {
		t.Fatalf("FindModByOriginalSourcePath() = %+v, %v; want mod %d", foundMod, found, modID)
	}

	_, found, err = store.FindModByOriginalSourcePath(context.Background(), gameID, filepath.Join(filepath.Dir(originalPath), "Missing"))
	if err != nil {
		t.Fatalf("FindModByOriginalSourcePath() missing error = %v", err)
	}
	if found {
		t.Fatal("FindModByOriginalSourcePath() missing found = true, want false")
	}
}

func TestCreateModPersistsOriginalSourcePath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	originalPath, err := CanonicalModOriginalSourcePath(filepath.Join(t.TempDir(), "SkyUI"))
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}

	originalName := "SkyUI.zip"
	fileCount := int64(2)
	directoryCount := int64(1)
	totalSizeBytes := int64(42)
	metadataJSON := `{"parser":"inventory"}`
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               " SkyUI ",
		SourceType:         dbtypes.ModSourceTypeArchive,
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: originalPath,
		OriginalSourceName: &originalName,
		FileCount:          &fileCount,
		DirectoryCount:     &directoryCount,
		TotalSizeBytes:     &totalSizeBytes,
		MetadataJSON:       &metadataJSON,
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	if mod.ID == 0 || mod.GameID != gameID || mod.Name != "SkyUI" || mod.SourceType != dbtypes.ModSourceTypeArchive || mod.SourcePath != "/managed/skyui" || mod.OriginalSourcePath != originalPath || mod.OriginalSourceName == nil || *mod.OriginalSourceName != originalName {
		t.Fatalf("CreateMod() = %+v, want persisted mod fields", mod)
	}
	if mod.FileCount == nil || *mod.FileCount != fileCount || mod.DirectoryCount == nil || *mod.DirectoryCount != directoryCount || mod.TotalSizeBytes == nil || *mod.TotalSizeBytes != totalSizeBytes || mod.MetadataJSON == nil || *mod.MetadataJSON != metadataJSON {
		t.Fatalf("CreateMod() metadata = %+v, want persisted metadata", mod)
	}
}

func TestCreateModPersistsDetectedMetadata(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: "/imports/skyui",
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version:     stringPtr("1.0.0"),
			Author:      stringPtr("Mod Author"),
			Description: stringPtr("User interface mod"),
			SourceURL:   stringPtr("https://example.com/skyui"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	metadata, found, err := store.GetModMetadata(context.Background(), mod.ID)
	if err != nil {
		t.Fatalf("GetModMetadata() error = %v", err)
	}
	if !found {
		t.Fatal("GetModMetadata() found = false, want true")
	}
	if metadata.DetectedVersion == nil || *metadata.DetectedVersion != "1.0.0" ||
		metadata.DetectedAuthor == nil || *metadata.DetectedAuthor != "Mod Author" ||
		metadata.DetectedDescription == nil || *metadata.DetectedDescription != "User interface mod" ||
		metadata.DetectedSourceURL == nil || *metadata.DetectedSourceURL != "https://example.com/skyui" {
		t.Fatalf("GetModMetadata() = %+v, want detected metadata", metadata)
	}
}

func TestUpdateModMetadataPersistsUserValuesClearsAndResets(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: "/imports/skyui",
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version:   stringPtr("1.0.0"),
			Author:    stringPtr("Detected Author"),
			SourceURL: stringPtr("https://example.com/detected"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	updated, err := store.UpdateModMetadata(context.Background(), dbtypes.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dbtypes.ModMetadataFieldUpdate{
			UserSet: true,
			Value:   stringPtr("2.0.0"),
		},
		Author: dbtypes.ModMetadataFieldUpdate{
			UserSet: true,
		},
		SourceURL: dbtypes.ModMetadataFieldUpdate{},
		Notes:     stringPtr("Local notes"),
	})
	if err != nil {
		t.Fatalf("UpdateModMetadata() error = %v", err)
	}

	if updated.UserVersion == nil || *updated.UserVersion != "2.0.0" || !updated.VersionUserSet {
		t.Fatalf("Version metadata = %+v, want user override", updated)
	}
	if updated.UserAuthor != nil || !updated.AuthorUserSet {
		t.Fatalf("Author metadata = %+v, want explicit clear", updated)
	}
	if updated.UserSourceURL != nil || updated.SourceURLUserSet {
		t.Fatalf("SourceURL metadata = %+v, want reset to detected", updated)
	}
	if updated.Notes == nil || *updated.Notes != "Local notes" {
		t.Fatalf("Notes = %v, want persisted notes", updated.Notes)
	}
}

func TestUpdateModDetectedMetadataPreservesUserValues(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: "/imports/skyui",
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version: stringPtr("1.0.0"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}
	if _, err := store.UpdateModMetadata(context.Background(), dbtypes.UpdateModMetadataInput{
		ModID: mod.ID,
		Version: dbtypes.ModMetadataFieldUpdate{
			UserSet: true,
			Value:   stringPtr("User Version"),
		},
	}); err != nil {
		t.Fatalf("UpdateModMetadata() error = %v", err)
	}

	updated, err := store.UpdateModDetectedMetadata(context.Background(), mod.ID, dbtypes.ModMetadataDetectedInput{
		Version: stringPtr("2.0.0"),
		Author:  stringPtr("Detected Author"),
	})
	if err != nil {
		t.Fatalf("UpdateModDetectedMetadata() error = %v", err)
	}

	if updated.DetectedVersion == nil || *updated.DetectedVersion != "2.0.0" || updated.UserVersion == nil || *updated.UserVersion != "User Version" || !updated.VersionUserSet {
		t.Fatalf("Version metadata = %+v, want detected update with user override preserved", updated)
	}
	if updated.DetectedAuthor == nil || *updated.DetectedAuthor != "Detected Author" {
		t.Fatalf("DetectedAuthor = %v, want updated author", updated.DetectedAuthor)
	}
}

func TestUpdateModPackageChangesPackageFieldsAndPreservesRelatedRows(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: "/imports/skyui-v1",
		FileCount:          int64TestPtr(1),
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version: stringPtr("1.0.0"),
		},
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}
	if _, err := store.CreateOrReplaceModInstallConfig(context.Background(), dbtypes.CreateModInstallConfigInput{
		ModID:              mod.ID,
		StrategyType:       "generic_copy",
		TargetBase:         "game_root",
		TargetRelativePath: "Data",
	}); err != nil {
		t.Fatalf("CreateOrReplaceModInstallConfig() error = %v", err)
	}
	if _, err := store.AddModToProfile(context.Background(), profile.ID, mod.ID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}

	replacementOriginalPath, err := CanonicalModOriginalSourcePath("/imports/skyui-v2.zip")
	if err != nil {
		t.Fatalf("CanonicalModOriginalSourcePath() error = %v", err)
	}
	updated, err := store.UpdateModPackage(context.Background(), dbtypes.UpdateModPackageInput{
		ModID:              mod.ID,
		SourceType:         dbtypes.ModSourceTypeArchive,
		OriginalSourcePath: replacementOriginalPath,
		OriginalSourceName: stringPtr("skyui-v2.zip"),
		FileCount:          int64TestPtr(2),
		DirectoryCount:     int64TestPtr(1),
		TotalSizeBytes:     int64TestPtr(10),
		DetectedMetadata: dbtypes.ModMetadataDetectedInput{
			Version: stringPtr("2.0.0"),
		},
	})
	if err != nil {
		t.Fatalf("UpdateModPackage() error = %v", err)
	}

	if updated.ID != mod.ID || updated.Name != "SkyUI" || updated.SourcePath != filepath.Clean("/managed/skyui") || updated.SourceType != dbtypes.ModSourceTypeArchive || updated.OriginalSourcePath != replacementOriginalPath || updated.OriginalSourceName == nil || *updated.OriginalSourceName != "skyui-v2.zip" {
		t.Fatalf("UpdateModPackage() = %+v, want same identity with updated package fields", updated)
	}
	if updated.FileCount == nil || *updated.FileCount != 2 || updated.DirectoryCount == nil || *updated.DirectoryCount != 1 || updated.TotalSizeBytes == nil || *updated.TotalSizeBytes != 10 {
		t.Fatalf("UpdateModPackage() counts = %+v, want updated counts", updated)
	}
	config, found, err := store.GetModInstallConfig(context.Background(), mod.ID)
	if err != nil {
		t.Fatalf("GetModInstallConfig() error = %v", err)
	}
	if !found || config.TargetRelativePath != "Data" {
		t.Fatalf("GetModInstallConfig() = %+v, %v; want preserved config", config, found)
	}
	profileMods, err := store.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 1 || profileMods[0].ModID != mod.ID {
		t.Fatalf("ListProfileMods() = %+v, want preserved profile membership", profileMods)
	}
	metadata, found, err := store.GetModMetadata(context.Background(), mod.ID)
	if err != nil {
		t.Fatalf("GetModMetadata() error = %v", err)
	}
	if !found || metadata.DetectedVersion == nil || *metadata.DetectedVersion != "2.0.0" {
		t.Fatalf("GetModMetadata() = %+v, %v; want updated detected metadata", metadata, found)
	}
}

func TestModMetadataCascadesWhenModDeleted(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               "SkyUI",
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: "/imports/skyui",
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}
	if err := store.DeleteMod(context.Background(), mod.ID); err != nil {
		t.Fatalf("DeleteMod() error = %v", err)
	}

	var count int
	if err := store.DB().Get(&count, `SELECT COUNT(*) FROM mod_metadata WHERE mod_id = ?`, mod.ID); err != nil {
		t.Fatalf("count mod metadata: %v", err)
	}
	if count != 0 {
		t.Fatalf("metadata count = %d, want cascade delete", count)
	}
}

func TestGetModReportsMissingMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	mod, found, err := store.GetMod(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetMod() error = %v", err)
	}
	if found || mod != (dbtypes.Mod{}) {
		t.Fatalf("GetMod() = %+v, %v; want missing", mod, found)
	}
}

func TestRenameModUpdatesName(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")

	renamed, err := store.RenameMod(context.Background(), modID, "  USSEP  ")
	if err != nil {
		t.Fatalf("RenameMod() error = %v", err)
	}
	if renamed.ID != modID || renamed.Name != "USSEP" {
		t.Fatalf("RenameMod() = %+v, want renamed same mod", renamed)
	}
}

func TestRenameModRejectsEmptyAndMissingMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")

	if _, err := store.RenameMod(context.Background(), modID, "   "); err == nil {
		t.Fatal("RenameMod() empty name error = nil, want error")
	}
	if _, err := store.RenameMod(context.Background(), 999, "Missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("RenameMod() missing error = %v, want no rows", err)
	}
}

func TestDeleteModDeletesRowAndCascadesProfileMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}

	if err := store.DeleteMod(context.Background(), modID); err != nil {
		t.Fatalf("DeleteMod() error = %v", err)
	}

	_, found, err := store.GetMod(context.Background(), modID)
	if err != nil {
		t.Fatalf("GetMod() after delete error = %v", err)
	}
	if found {
		t.Fatal("GetMod() after delete found = true, want false")
	}

	profileMods, err := store.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 0 {
		t.Fatalf("ListProfileMods() length = %d, want 0: %+v", len(profileMods), profileMods)
	}
}

func TestDeleteModReportsMissingRow(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	err := store.DeleteMod(context.Background(), 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("DeleteMod() error = %v, want sql.ErrNoRows", err)
	}
}

func TestCountProfilesUsingModAndProfileUsesMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstProfile := mustCreateProfile(t, store, gameID, "Default")
	secondProfile := mustCreateProfile(t, store, gameID, "Survival")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	otherModID := insertProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")
	for _, profileID := range []int64{firstProfile.ID, secondProfile.ID} {
		if _, err := store.AddModToProfile(context.Background(), profileID, modID); err != nil {
			t.Fatalf("AddModToProfile(%d) error = %v", profileID, err)
		}
	}

	count, err := store.CountProfilesUsingMod(context.Background(), modID)
	if err != nil {
		t.Fatalf("CountProfilesUsingMod() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountProfilesUsingMod() = %d, want 2", count)
	}

	uses, err := store.ProfileUsesMod(context.Background(), firstProfile.ID, modID)
	if err != nil {
		t.Fatalf("ProfileUsesMod() error = %v", err)
	}
	if !uses {
		t.Fatal("ProfileUsesMod() = false, want true")
	}

	uses, err = store.ProfileUsesMod(context.Background(), firstProfile.ID, otherModID)
	if err != nil {
		t.Fatalf("ProfileUsesMod() other mod error = %v", err)
	}
	if uses {
		t.Fatal("ProfileUsesMod() other mod = true, want false")
	}
}

func TestCreateModRequiresNameAndPaths(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	tests := []struct {
		name               string
		modName            string
		sourcePath         string
		originalSourcePath string
	}{
		{
			name:               "missing name",
			modName:            " ",
			sourcePath:         "/managed/skyui",
			originalSourcePath: "/imports/skyui",
		},
		{
			name:               "missing managed source",
			modName:            "SkyUI",
			sourcePath:         " ",
			originalSourcePath: "/imports/skyui",
		},
		{
			name:               "missing original source",
			modName:            "SkyUI",
			sourcePath:         "/managed/skyui",
			originalSourcePath: " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
				GameID:             gameID,
				Name:               tt.modName,
				SourcePath:         tt.sourcePath,
				OriginalSourcePath: tt.originalSourcePath,
			})
			if err == nil {
				t.Fatal("CreateMod() error = nil, want validation error")
			}
		})
	}
}

func TestCreateOrReplaceModInstallConfigPersistsNullableSourceSubpath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	config, err := store.CreateOrReplaceModInstallConfig(context.Background(), dbtypes.CreateModInstallConfigInput{
		ModID:              modID,
		StrategyType:       "generic_copy",
		TargetBase:         "game_root",
		TargetRelativePath: "Data",
	})
	if err != nil {
		t.Fatalf("CreateOrReplaceModInstallConfig() error = %v", err)
	}
	if config.ModID != modID || config.StrategyType != "generic_copy" || config.TargetBase != "game_root" || config.TargetRelativePath != "Data" || config.SourceSubpath != nil {
		t.Fatalf("CreateOrReplaceModInstallConfig() = %+v, want persisted config without source subpath", config)
	}

	sourceSubpath := "plugin"
	replaced, err := store.CreateOrReplaceModInstallConfig(context.Background(), dbtypes.CreateModInstallConfigInput{
		ModID:              modID,
		StrategyType:       "generic_copy",
		TargetBase:         "game_root",
		TargetRelativePath: "BepInEx/plugins",
		SourceSubpath:      &sourceSubpath,
	})
	if err != nil {
		t.Fatalf("CreateOrReplaceModInstallConfig() replace error = %v", err)
	}
	if replaced.TargetRelativePath != "BepInEx/plugins" || replaced.SourceSubpath == nil || *replaced.SourceSubpath != sourceSubpath {
		t.Fatalf("CreateOrReplaceModInstallConfig() replaced = %+v, want updated config", replaced)
	}
}

func TestGetModInstallConfigReportsMissingConfig(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	config, found, err := store.GetModInstallConfig(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetModInstallConfig() error = %v", err)
	}
	if found || config != (dbtypes.ModInstallConfig{}) {
		t.Fatalf("GetModInstallConfig() = %+v, %v; want missing", config, found)
	}
}

func TestCreateModWithInstallConfigIsTransactional(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	result, err := store.CreateModWithInstallConfig(context.Background(), dbtypes.CreateModWithInstallConfigInput{
		Mod: dbtypes.CreateModInput{
			GameID:             gameID,
			Name:               "SkyUI",
			SourcePath:         "/managed/skyui",
			OriginalSourcePath: "/imports/skyui",
		},
		Config: dbtypes.CreateModInstallConfigInput{
			StrategyType:       "generic_copy",
			TargetBase:         "game_root",
			TargetRelativePath: "Data",
		},
	})
	if err != nil {
		t.Fatalf("CreateModWithInstallConfig() error = %v", err)
	}
	if result.Mod.ID == 0 || result.Config.ModID != result.Mod.ID || result.Config.TargetRelativePath != "Data" {
		t.Fatalf("CreateModWithInstallConfig() = %+v, want mod and config", result)
	}

	_, err = store.CreateModWithInstallConfig(context.Background(), dbtypes.CreateModWithInstallConfigInput{
		Mod: dbtypes.CreateModInput{
			GameID:             gameID,
			Name:               "Broken",
			SourcePath:         "/managed/broken",
			OriginalSourcePath: "/imports/broken",
		},
		Config: dbtypes.CreateModInstallConfigInput{
			StrategyType:       "unsupported",
			TargetBase:         "game_root",
			TargetRelativePath: "Data",
		},
	})
	if err == nil {
		t.Fatal("CreateModWithInstallConfig() invalid config error = nil, want error")
	}

	var count int
	if err := store.DB().Get(&count, `SELECT COUNT(*) FROM mods WHERE name = 'Broken'`); err != nil {
		t.Fatalf("count broken mods: %v", err)
	}
	if count != 0 {
		t.Fatalf("broken mod count = %d, want transaction rollback", count)
	}
}

func TestOriginalSourcePathIsUniquePerGame(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	originalPath := "/imports/skyui"
	insertProfileTestModWithOriginalSource(t, store, gameID, "SkyUI", "/managed/skyrim/SkyUI", originalPath)
	insertProfileTestModWithOriginalSource(t, store, otherGameID, "SkyUI", "/managed/fallout/SkyUI", originalPath)

	if _, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path, original_source_path)
		VALUES (?, ?, ?, ?)
	`, gameID, "SkyUI Duplicate", "/managed/skyrim/SkyUI-copy", originalPath); err == nil {
		t.Fatal("insert duplicate original source path succeeded, want unique constraint error")
	}
}

func TestAddModToProfilePersistsEnabledLoadOrderAndIgnoresDuplicates(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	firstModID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	secondModID := insertProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")

	first, err := store.AddModToProfile(context.Background(), profile.ID, firstModID)
	if err != nil {
		t.Fatalf("AddModToProfile() first error = %v", err)
	}
	second, err := store.AddModToProfile(context.Background(), profile.ID, secondModID)
	if err != nil {
		t.Fatalf("AddModToProfile() second error = %v", err)
	}
	duplicate, err := store.AddModToProfile(context.Background(), profile.ID, firstModID)
	if err != nil {
		t.Fatalf("AddModToProfile() duplicate error = %v", err)
	}

	if !first.Enabled || first.LoadOrder != 0 || first.Name != "SkyUI" {
		t.Fatalf("first profile mod = %+v, want enabled SkyUI load order 0", first)
	}
	if !second.Enabled || second.LoadOrder != 1 || second.Name != "USSEP" {
		t.Fatalf("second profile mod = %+v, want enabled USSEP load order 1", second)
	}
	if duplicate != first {
		t.Fatalf("duplicate = %+v, want existing %+v", duplicate, first)
	}

	profileMods, err := store.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 2 {
		t.Fatalf("ListProfileMods() length = %d, want 2: %+v", len(profileMods), profileMods)
	}
}

func TestAddModToProfileRejectsCrossGameMod(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	profile := mustCreateProfile(t, store, gameID, "Default")
	modID := insertProfileTestMod(t, store, otherGameID, "Other Mod", "/mods/other")

	if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err == nil {
		t.Fatal("AddModToProfile() cross-game error = nil, want error")
	}
}

func TestRemoveModFromProfileDeletesAssignmentAndIsIdempotent(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")

	if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}
	if err := store.RemoveModFromProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("RemoveModFromProfile() error = %v", err)
	}
	if err := store.RemoveModFromProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("RemoveModFromProfile() missing error = %v", err)
	}

	profileMods, err := store.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 0 {
		t.Fatalf("ListProfileMods() length = %d, want 0: %+v", len(profileMods), profileMods)
	}
}

func TestSetProfileModEnabledPersistsState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	modID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}

	disabled, err := store.SetProfileModEnabled(context.Background(), profile.ID, modID, false)
	if err != nil {
		t.Fatalf("SetProfileModEnabled(false) error = %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("SetProfileModEnabled(false) = %+v, want disabled", disabled)
	}

	enabled, err := store.SetProfileModEnabled(context.Background(), profile.ID, modID, true)
	if err != nil {
		t.Fatalf("SetProfileModEnabled(true) error = %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("SetProfileModEnabled(true) = %+v, want enabled", enabled)
	}

	if _, err := store.SetProfileModEnabled(context.Background(), profile.ID, 999, true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("SetProfileModEnabled() missing error = %v, want no rows", err)
	}
}

func TestReorderProfileModsPersistsContiguousOrderAndPreservesState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	firstModID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	secondModID := insertProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")
	thirdModID := insertProfileTestMod(t, store, gameID, "ENB", "/mods/enb")

	for _, modID := range []int64{firstModID, secondModID, thirdModID} {
		if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err != nil {
			t.Fatalf("AddModToProfile(%d) error = %v", modID, err)
		}
	}
	if _, err := store.SetProfileModEnabled(context.Background(), profile.ID, secondModID, false); err != nil {
		t.Fatalf("SetProfileModEnabled() error = %v", err)
	}

	reordered, err := store.ReorderProfileMods(context.Background(), profile.ID, []int64{thirdModID, secondModID, firstModID})
	if err != nil {
		t.Fatalf("ReorderProfileMods() error = %v", err)
	}

	assertProfileModOrder(t, reordered, []int64{thirdModID, secondModID, firstModID})
	if reordered[1].Enabled {
		t.Fatalf("disabled mod after reorder = %+v, want still disabled", reordered[1])
	}

	listed, err := store.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	assertProfileModOrder(t, listed, []int64{thirdModID, secondModID, firstModID})
}

func TestReorderProfileModsRejectsInvalidOrdersAndRollsBack(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	profile := mustCreateProfile(t, store, gameID, "Default")
	otherProfile := mustCreateProfile(t, store, otherGameID, "Default")
	firstModID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	secondModID := insertProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")
	otherProfileModID := insertProfileTestMod(t, store, otherGameID, "FallUI", "/mods/fallui")

	for _, modID := range []int64{firstModID, secondModID} {
		if _, err := store.AddModToProfile(context.Background(), profile.ID, modID); err != nil {
			t.Fatalf("AddModToProfile(%d) error = %v", modID, err)
		}
	}
	if _, err := store.AddModToProfile(context.Background(), otherProfile.ID, otherProfileModID); err != nil {
		t.Fatalf("AddModToProfile() other profile error = %v", err)
	}

	invalidOrders := map[string][]int64{
		"duplicate":     {firstModID, firstModID},
		"empty":         {},
		"extra":         {firstModID, secondModID, otherProfileModID},
		"missing":       {firstModID},
		"wrong-profile": {firstModID, otherProfileModID},
	}
	for name, modIDs := range invalidOrders {
		t.Run(name, func(t *testing.T) {
			if _, err := store.ReorderProfileMods(context.Background(), profile.ID, modIDs); err == nil {
				t.Fatal("ReorderProfileMods() error = nil, want error")
			}

			listed, err := store.ListProfileMods(context.Background(), profile.ID)
			if err != nil {
				t.Fatalf("ListProfileMods() error = %v", err)
			}
			assertProfileModOrder(t, listed, []int64{firstModID, secondModID})
		})
	}
}

func TestReorderProfileModsAcceptsEmptyOrderForEmptyProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")

	reordered, err := store.ReorderProfileMods(context.Background(), profile.ID, nil)
	if err != nil {
		t.Fatalf("ReorderProfileMods() error = %v", err)
	}
	if len(reordered) != 0 {
		t.Fatalf("ReorderProfileMods() length = %d, want 0: %+v", len(reordered), reordered)
	}
}

func assertProfileModOrder(t *testing.T, profileMods []dbtypes.ProfileMod, modIDs []int64) {
	t.Helper()

	if len(profileMods) != len(modIDs) {
		t.Fatalf("profile mod length = %d, want %d: %+v", len(profileMods), len(modIDs), profileMods)
	}

	for index, modID := range modIDs {
		if profileMods[index].ModID != modID {
			t.Fatalf("profileMods[%d].ModID = %d, want %d: %+v", index, profileMods[index].ModID, modID, profileMods)
		}
		if profileMods[index].LoadOrder != int64(index) {
			t.Fatalf("profileMods[%d].LoadOrder = %d, want %d: %+v", index, profileMods[index].LoadOrder, index, profileMods)
		}
	}
}

func insertProfileTestMod(t *testing.T, store *Store, gameID int64, name string, sourcePath string) int64 {
	t.Helper()

	return insertProfileTestModWithOriginalSource(t, store, gameID, name, sourcePath, sourcePath)
}

func insertProfileTestModWithOriginalSource(t *testing.T, store *Store, gameID int64, name string, sourcePath string, originalSourcePath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path, original_source_path)
		VALUES (?, ?, ?, ?)
	`, gameID, name, sourcePath, originalSourcePath)
	if err != nil {
		t.Fatalf("insert profile test mod: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("profile test mod LastInsertId(): %v", err)
	}

	return id
}
