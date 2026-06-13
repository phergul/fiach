package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestCreateProfileForGame(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")

	profile, err := store.CreateProfile(context.Background(), gameID, "  Default  ")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	if profile.ID == 0 || profile.GameID != gameID || profile.Name != "Default" {
		t.Fatalf("CreateProfile() = %+v, want profile for game", profile)
	}
	if profile.CreatedAt == "" || profile.UpdatedAt == "" {
		t.Fatalf("CreateProfile() timestamps are empty: %+v", profile)
	}
}

func TestCreateProfileRejectsMissingGame(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	_, err := store.CreateProfile(context.Background(), 999, "Default")
	if err == nil {
		t.Fatal("CreateProfile() error = nil, want foreign key error")
	}
}

func TestCreateProfileRejectsEmptyAndDuplicateNames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")

	if _, err := store.CreateProfile(context.Background(), gameID, "   "); err == nil {
		t.Fatal("CreateProfile() empty name error = nil, want error")
	}

	if _, err := store.CreateProfile(context.Background(), gameID, "Default"); err != nil {
		t.Fatalf("CreateProfile() initial error = %v", err)
	}
	if _, err := store.CreateProfile(context.Background(), gameID, "Default"); err == nil {
		t.Fatal("CreateProfile() duplicate error = nil, want error")
	}
}

func TestListProfilesOrdersByName(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	zeta := mustCreateProfile(t, store, gameID, "Zeta")
	alpha := mustCreateProfile(t, store, gameID, "alpha")
	middle := mustCreateProfile(t, store, gameID, "Middle")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	mustCreateProfile(t, store, otherGameID, "Other")

	profiles, err := store.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	if len(profiles) != 3 {
		t.Fatalf("ListProfiles() length = %d, want 3: %+v", len(profiles), profiles)
	}
	if profiles[0].ID != alpha.ID || profiles[1].ID != middle.ID || profiles[2].ID != zeta.ID {
		t.Fatalf("ListProfiles() order = %+v, want alpha, Middle, Zeta", profiles)
	}
}

func TestRenameProfileUpdatesName(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")

	renamed, err := store.RenameProfile(context.Background(), profile.ID, "  Survival  ")
	if err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}

	if renamed.ID != profile.ID || renamed.Name != "Survival" {
		t.Fatalf("RenameProfile() = %+v, want renamed same profile", renamed)
	}
}

func TestRenameProfileRejectsEmptyDuplicateAndMissingProfiles(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	first := mustCreateProfile(t, store, gameID, "Default")
	mustCreateProfile(t, store, gameID, "Survival")

	if _, err := store.RenameProfile(context.Background(), first.ID, "   "); err == nil {
		t.Fatal("RenameProfile() empty name error = nil, want error")
	}
	if _, err := store.RenameProfile(context.Background(), first.ID, "Survival"); err == nil {
		t.Fatal("RenameProfile() duplicate error = nil, want error")
	}
	if _, err := store.RenameProfile(context.Background(), 999, "Missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("RenameProfile() missing error = %v, want no rows", err)
	}
}

func TestDeleteProfileDeletesRow(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")

	if err := store.DeleteProfile(context.Background(), profile.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}

	profiles, err := store.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("ListProfiles() after delete = %+v, want empty", profiles)
	}
}

func TestDuplicateProfileCopiesModsAndState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	original := mustCreateProfile(t, store, gameID, "Default")
	firstModID := insertProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	secondModID := insertProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")

	if _, err := store.AddModToProfile(context.Background(), original.ID, firstModID); err != nil {
		t.Fatalf("AddModToProfile() first error = %v", err)
	}
	if _, err := store.AddModToProfile(context.Background(), original.ID, secondModID); err != nil {
		t.Fatalf("AddModToProfile() second error = %v", err)
	}
	if _, err := store.SetProfileModEnabled(context.Background(), original.ID, secondModID, false); err != nil {
		t.Fatalf("SetProfileModEnabled() error = %v", err)
	}

	duplicated, err := store.DuplicateProfile(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("DuplicateProfile() error = %v", err)
	}

	if duplicated.ID == original.ID || duplicated.GameID != gameID || duplicated.Name != "Default (copy)" {
		t.Fatalf("DuplicateProfile() = %+v, want copied profile", duplicated)
	}

	profileMods, err := store.ListProfileMods(context.Background(), duplicated.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 2 {
		t.Fatalf("ListProfileMods() length = %d, want 2: %+v", len(profileMods), profileMods)
	}
	if profileMods[0].ModID != firstModID || !profileMods[0].Enabled || profileMods[0].LoadOrder != 0 {
		t.Fatalf("first duplicated mod = %+v, want enabled mod at load order 0", profileMods[0])
	}
	if profileMods[1].ModID != secondModID || profileMods[1].Enabled || profileMods[1].LoadOrder != 1 {
		t.Fatalf("second duplicated mod = %+v, want disabled mod at load order 1", profileMods[1])
	}

	originalMods, err := store.ListProfileMods(context.Background(), original.ID)
	if err != nil {
		t.Fatalf("ListProfileMods(original) error = %v", err)
	}
	if !originalMods[0].Enabled || originalMods[1].Enabled {
		t.Fatalf("original profile mods changed unexpectedly: %+v", originalMods)
	}
}

func insertProfileTestGame(t *testing.T, store *Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, installPath)
	if err != nil {
		t.Fatalf("insert profile test game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("profile test game LastInsertId(): %v", err)
	}

	return id
}

func mustCreateProfile(t *testing.T, store *Store, gameID int64, name string) dbtypes.ModProfile {
	t.Helper()

	profile, err := store.CreateProfile(context.Background(), gameID, name)
	if err != nil {
		t.Fatalf("CreateProfile(%q) error = %v", name, err)
	}

	return profile
}
