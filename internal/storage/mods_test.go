package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
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

func insertProfileTestMod(t *testing.T, store *Store, gameID int64, name string, sourcePath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path)
		VALUES (?, ?, ?)
	`, gameID, name, sourcePath)
	if err != nil {
		t.Fatalf("insert profile test mod: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("profile test mod LastInsertId(): %v", err)
	}

	return id
}
