package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/phergul/mod-manager/internal/storage/dbtypes"
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
	mod, err := store.CreateMod(context.Background(), dbtypes.CreateModInput{
		GameID:             gameID,
		Name:               " SkyUI ",
		SourceType:         dbtypes.ModSourceTypeArchive,
		SourcePath:         "/managed/skyui",
		OriginalSourcePath: originalPath,
		OriginalSourceName: &originalName,
	})
	if err != nil {
		t.Fatalf("CreateMod() error = %v", err)
	}

	if mod.ID == 0 || mod.GameID != gameID || mod.Name != "SkyUI" || mod.SourceType != dbtypes.ModSourceTypeArchive || mod.SourcePath != "/managed/skyui" || mod.OriginalSourcePath != originalPath || mod.OriginalSourceName == nil || *mod.OriginalSourceName != originalName {
		t.Fatalf("CreateMod() = %+v, want persisted mod fields", mod)
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
