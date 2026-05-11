package services

import (
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

	profile, err := service.CreateProfile(gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	profiles, err := service.ListProfiles(gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}

	if len(profiles) != 1 || profiles[0].ID != profile.ID {
		t.Fatalf("ListProfiles() = %+v, want created profile", profiles)
	}
}

func TestProfileServiceRenamesActivatesClearsAndDeletesProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store)
	profile, err := service.CreateProfile(gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	renamed, err := service.RenameProfile(profile.ID, "Survival")
	if err != nil {
		t.Fatalf("RenameProfile() error = %v", err)
	}
	if renamed.Name != "Survival" {
		t.Fatalf("RenameProfile() name = %q, want Survival", renamed.Name)
	}

	active, err := service.ActivateProfile(gameID, profile.ID)
	if err != nil {
		t.Fatalf("ActivateProfile() error = %v", err)
	}
	if !active.IsActive {
		t.Fatalf("ActivateProfile() = %+v, want active", active)
	}

	activePtr, err := service.GetActiveProfile(gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() error = %v", err)
	}
	if activePtr == nil || activePtr.ID != profile.ID {
		t.Fatalf("GetActiveProfile() = %+v, want active profile", activePtr)
	}

	if err := service.ClearActiveProfile(gameID); err != nil {
		t.Fatalf("ClearActiveProfile() error = %v", err)
	}
	activePtr, err = service.GetActiveProfile(gameID)
	if err != nil {
		t.Fatalf("GetActiveProfile() after clear error = %v", err)
	}
	if activePtr != nil {
		t.Fatalf("GetActiveProfile() after clear = %+v, want nil", activePtr)
	}

	if err := service.DeleteProfile(profile.ID); err != nil {
		t.Fatalf("DeleteProfile() error = %v", err)
	}
	profiles, err := service.ListProfiles(gameID)
	if err != nil {
		t.Fatalf("ListProfiles() after delete error = %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("ListProfiles() after delete = %+v, want empty", profiles)
	}
}

func TestProfileServiceReturnsStorageConfigurationError(t *testing.T) {
	t.Parallel()

	service := NewProfileService(nil)

	_, err := service.CreateProfile(1, "Default")
	if err == nil {
		t.Fatal("CreateProfile() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "create profile") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("CreateProfile() error = %q, want service context", err.Error())
	}
}

func TestProfileServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store)
	_, err := service.CreateProfile(999, "Default")
	if err == nil {
		t.Fatal("CreateProfile() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "create profile") || !strings.Contains(err.Error(), "insert profile row") {
		t.Fatalf("CreateProfile() error = %q, want distinct service and storage context", err.Error())
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
