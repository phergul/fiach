package services

import (
	"context"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/storage"
)

func TestProfileServiceCreatesAndListsProfiles(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store)

	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	profiles, err := service.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	if len(profiles) != 1 || profiles[0].ID != profile.ID {
		t.Fatalf("ListProfiles() = %+v, want created profile", profiles)
	}
}

func TestProfileServiceManagesProfileMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store)
	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")

	profileMod, err := service.AddModToProfile(context.Background(), profile.ID, modID)
	if err != nil {
		t.Fatalf("AddModToProfile() error = %v", err)
	}
	if profileMod.ProfileID != profile.ID || profileMod.ModID != modID || !profileMod.Enabled {
		t.Fatalf("AddModToProfile() = %+v, want enabled profile mod", profileMod)
	}

	profileMods, err := service.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() error = %v", err)
	}
	if len(profileMods) != 1 || profileMods[0].ModID != modID {
		t.Fatalf("ListProfileMods() = %+v, want inserted profile mod", profileMods)
	}

	disabled, err := service.SetProfileModEnabled(context.Background(), profile.ID, modID, false)
	if err != nil {
		t.Fatalf("SetProfileModEnabled() error = %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("SetProfileModEnabled() = %+v, want disabled", disabled)
	}

	if err := service.RemoveModFromProfile(context.Background(), profile.ID, modID); err != nil {
		t.Fatalf("RemoveModFromProfile() error = %v", err)
	}
	profileMods, err = service.ListProfileMods(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("ListProfileMods() after remove error = %v", err)
	}
	if len(profileMods) != 0 {
		t.Fatalf("ListProfileMods() after remove = %+v, want empty", profileMods)
	}
}

func TestProfileServiceReordersProfileMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store)
	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	firstModID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	secondModID := insertServiceProfileTestMod(t, store, gameID, "USSEP", "/mods/ussep")

	if _, err := service.AddModToProfile(context.Background(), profile.ID, firstModID); err != nil {
		t.Fatalf("AddModToProfile() first error = %v", err)
	}
	if _, err := service.AddModToProfile(context.Background(), profile.ID, secondModID); err != nil {
		t.Fatalf("AddModToProfile() second error = %v", err)
	}

	reordered, err := service.ReorderProfileMods(context.Background(), profile.ID, []int64{secondModID, firstModID})
	if err != nil {
		t.Fatalf("ReorderProfileMods() error = %v", err)
	}
	if len(reordered) != 2 || reordered[0].ModID != secondModID || reordered[0].LoadOrder != 0 || reordered[1].ModID != firstModID || reordered[1].LoadOrder != 1 {
		t.Fatalf("ReorderProfileMods() = %+v, want second then first", reordered)
	}
}

func TestProfileServiceRenamesActivatesClearsAndDeletesProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store)
	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	renamed, err := service.RenameProfile(context.Background(), profile.ID, "Survival")
	if err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}
	if renamed.Name != "Survival" {
		t.Fatalf("RenameProfile() name = %q, want Survival", renamed.Name)
	}

	active, err := service.ActivateProfile(context.Background(), gameID, profile.ID)
	if err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}
	if !active.IsActive {
		t.Fatalf("ActivateProfile() = %+v, want active", active)
	}

	activePtr, err := service.GetActiveProfile(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() error = %v", err)
	}
	if activePtr == nil || activePtr.ID != profile.ID {
		t.Fatalf("GetActiveProfile() = %+v, want active profile", activePtr)
	}

	if err := service.DeactivateProfile(context.Background(), gameID); err != nil {
		t.Fatalf("DeactivateProfile() error = %v", err)
	}
	activePtr, err = service.GetActiveProfile(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() after clear error = %v", err)
	}
	if activePtr != nil {
		t.Fatalf("GetActiveProfile() after clear = %+v, want nil", activePtr)
	}

	if err := service.DeleteProfile(context.Background(), profile.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	profiles, err := service.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() after delete error = %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("ListProfiles() after delete = %+v, want empty", profiles)
	}
}

func TestProfileServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.CreateProfile(context.Background(), 999, "Default")
	if err == nil {
		t.Fatal("CreateProfile() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "create profile") || !strings.Contains(err.Error(), "insert profile row") {
		t.Fatalf("CreateProfile() error = %q, want distinct service and storage context", err.Error())
	}

	_, err = service.AddModToProfile(context.Background(), 1, 999)
	if err == nil {
		t.Fatal("AddModToProfile() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "add mod to profile") || !strings.Contains(err.Error(), "insert profile mod row") {
		t.Fatalf("AddModToProfile() error = %q, want distinct service and storage context", err.Error())
	}
}

func insertServiceProfileTestGame(t *testing.T, store *storage.Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, installPath)
	if err != nil {
		t.Fatalf("insert service profile test game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service profile test game LastInsertId(): %v", err)
	}

	return id
}

func insertServiceProfileTestMod(t *testing.T, store *storage.Store, gameID int64, name string, sourcePath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path, original_source_path)
		VALUES (?, ?, ?, ?)
	`, gameID, name, sourcePath, sourcePath)
	if err != nil {
		t.Fatalf("insert service profile test mod: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service profile test mod LastInsertId(): %v", err)
	}

	return id
}
