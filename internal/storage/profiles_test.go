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
