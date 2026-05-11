package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
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

	if profile.ID == 0 || profile.GameID != gameID || profile.Name != "Default" || profile.IsActive {
		t.Fatalf("CreateProfile() = %+v, want inactive profile for game", profile)
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

func TestListProfilesOrdersActiveFirstThenName(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	zeta := mustCreateProfile(t, store, gameID, "Zeta")
	mustCreateProfile(t, store, gameID, "alpha")
	active := mustCreateProfile(t, store, gameID, "Middle")
	if _, err := store.ActivateProfile(context.Background(), gameID, active.ID); err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	mustCreateProfile(t, store, otherGameID, "Other")

	profiles, err := store.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	if len(profiles) != 3 {
		t.Fatalf("ListProfiles() length = %d, want 3: %+v", len(profiles), profiles)
	}
	if profiles[0].ID != active.ID || !profiles[0].IsActive {
		t.Fatalf("ListProfiles() first = %+v, want active profile", profiles[0])
	}
	if profiles[1].Name != "alpha" || profiles[2].ID != zeta.ID {
		t.Fatalf("ListProfiles() order = %+v, want active, alpha, Zeta", profiles)
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

func TestDeleteProfileCanLeaveNoActiveProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	if _, err := store.ActivateProfile(context.Background(), gameID, profile.ID); err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}

	if err := store.DeleteProfile(context.Background(), profile.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}

	active, found, err := store.GetActiveProfile(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() error = %v", err)
	}
	if found {
		t.Fatalf("GetActiveProfile() found %+v, want none", active)
	}
}

func TestActivateProfileEnforcesSingleActiveProfilePerGame(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	first := mustCreateProfile(t, store, gameID, "Default")
	second := mustCreateProfile(t, store, gameID, "Survival")
	otherGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	otherProfile := mustCreateProfile(t, store, otherGameID, "Other")

	if _, err := store.ActivateProfile(context.Background(), gameID, first.ID); err != nil {
		t.Fatalf("ActivateProfile() first error = %v", err)
	}
	active, err := store.ActivateProfile(context.Background(), gameID, second.ID)
	if err != nil {
		t.Fatalf("ActivateProfile() second error = %v", err)
	}
	if !active.IsActive || active.ID != second.ID {
		t.Fatalf("ActivateProfile() = %+v, want active second profile", active)
	}
	if _, err := store.ActivateProfile(context.Background(), gameID, otherProfile.ID); err == nil {
		t.Fatal("ActivateProfile() cross-game error = nil, want error")
	}

	profiles, err := store.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	activeCount := 0
	for _, profile := range profiles {
		if profile.IsActive {
			activeCount++
			if profile.ID != second.ID {
				t.Fatalf("active profile = %+v, want second", profile)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("active count = %d, want 1 in %+v", activeCount, profiles)
	}
}

func TestClearActiveProfileLeavesNoActiveProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	if _, err := store.ActivateProfile(context.Background(), gameID, profile.ID); err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}

	if err := store.ClearActiveProfile(context.Background(), gameID); err != nil {
		t.Fatalf("ClearActiveProfile() error = %v", err)
	}

	_, found, err := store.GetActiveProfile(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() error = %v", err)
	}
	if found {
		t.Fatal("GetActiveProfile() found = true, want false")
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

func mustCreateProfile(t *testing.T, store *Store, gameID int64, name string) ModProfile {
	t.Helper()

	profile, err := store.CreateProfile(context.Background(), gameID, name)
	if err != nil {
		t.Fatalf("CreateProfile(%q) error = %v", name, err)
	}

	return profile
}
