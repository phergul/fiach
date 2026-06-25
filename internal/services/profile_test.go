package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestProfileServiceCreatesAndListsProfiles(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store, testLogger())

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
	service := NewProfileService(store, testLogger())
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
	service := NewProfileService(store, testLogger())
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

func TestProfileServiceRenamesAndDeletesProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store, testLogger())
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

func TestProfileServiceRejectsDeletingAppliedProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store, testLogger())
	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profile.ID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "snapshot",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() setup error = %v", err)
	}

	err = service.DeleteProfile(context.Background(), profile.ID)
	if err == nil {
		t.Fatal("DeleteProfile() error = nil, want applied profile guard")
	}
	if err.Error() != "Restore vanilla before deleting an applied profile." {
		t.Fatalf("DeleteProfile() error = %q, want applied guard detail", err.Error())
	}

	profiles, err := service.ListProfiles(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != profile.ID {
		t.Fatalf("ListProfiles() after rejected delete = %+v, want applied profile preserved", profiles)
	}
}

func TestProfileServiceGetsAppliedProfileSummary(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store, testLogger())
	profile, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	summary, err := service.GetAppliedProfileSummary(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileSummary() empty error = %v", err)
	}
	if summary != nil {
		t.Fatalf("GetAppliedProfileSummary() empty = %+v, want nil", summary)
	}

	state, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profile.ID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "snapshot",
	})
	if err != nil {
		t.Fatalf("SaveAppliedProfileState() setup error = %v", err)
	}

	summary, err = service.GetAppliedProfileSummary(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileSummary() error = %v", err)
	}
	if summary == nil || summary.GameID != gameID || summary.ProfileID != profile.ID || summary.ProfileName != profile.Name || summary.AppliedAt != state.AppliedAt {
		t.Fatalf("GetAppliedProfileSummary() = %+v, want applied profile summary", summary)
	}
	if summary.HasAppliedProfileChanged != nil {
		t.Fatalf("GetAppliedProfileSummary() HasAppliedProfileChanged = %v, want nil without composition snapshot", *summary.HasAppliedProfileChanged)
	}
}

func TestProfileServiceReportsAppliedProfileCompositionChangedState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	firstModID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/managed/skyui")
	secondModID := insertServiceProfileTestMod(t, store, gameID, "Patch", "/managed/patch")
	addServiceProfileMod(t, store, profileID, firstModID, true, 0)
	addServiceProfileMod(t, store, profileID, secondModID, false, 1)
	saveServiceAppliedStateWithCurrentComposition(t, store, gameID, profileID)

	service := NewProfileService(store, testLogger())
	summary, err := service.GetAppliedProfileSummary(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileSummary() unchanged error = %v", err)
	}
	if summary == nil || summary.HasAppliedProfileChanged == nil || *summary.HasAppliedProfileChanged {
		t.Fatalf("GetAppliedProfileSummary() unchanged = %+v, want changed=false", summary)
	}

	if _, err := store.SetProfileModEnabled(context.Background(), profileID, secondModID, true); err != nil {
		t.Fatalf("SetProfileModEnabled() error = %v", err)
	}

	summary, err = service.GetAppliedProfileSummary(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileSummary() changed error = %v", err)
	}
	if summary == nil || summary.HasAppliedProfileChanged == nil || !*summary.HasAppliedProfileChanged {
		t.Fatalf("GetAppliedProfileSummary() changed = %+v, want changed=true", summary)
	}
}

func TestProfileServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewProfileService(store, testLogger())
	_, err := service.CreateProfile(context.Background(), 999, "Default")
	if err == nil {
		t.Fatal("CreateProfile() error = nil, want storage error")
	}
	if !strings.Contains(apperror.Detail(err), "insert profile row") {
		t.Fatalf("CreateProfile() detail = %q, want storage context", apperror.Detail(err))
	}

	_, err = service.AddModToProfile(context.Background(), 1, 999)
	if err == nil {
		t.Fatal("AddModToProfile() error = nil, want storage error")
	}
	if !strings.Contains(apperror.Detail(err), "insert profile mod row") {
		t.Fatalf("AddModToProfile() detail = %q, want storage context", apperror.Detail(err))
	}
}

func TestProfileServiceReturnsFriendlyDuplicateNameError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewProfileService(store, testLogger())

	if _, err := service.CreateProfile(context.Background(), gameID, "Default"); err != nil {
		t.Fatalf("CreateProfile() initial error = %v", err)
	}

	_, err := service.CreateProfile(context.Background(), gameID, "Default")
	if err == nil {
		t.Fatal("CreateProfile() duplicate error = nil, want error")
	}
	if err.Error() != "A profile with this name already exists for this game." {
		t.Fatalf("CreateProfile() error = %q", err.Error())
	}
	if !errors.Is(err, storage.ErrDuplicateProfileName) {
		t.Fatal("errors.Is(err, storage.ErrDuplicateProfileName) = false, want true")
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

func saveServiceAppliedStateWithCurrentComposition(t *testing.T, store *storage.Store, gameID int64, profileID int64) {
	t.Helper()

	profileMods, err := store.ListProfileMods(context.Background(), profileID)
	if err != nil {
		t.Fatalf("ListProfileMods() setup error = %v", err)
	}
	compositionSnapshot, err := encodeProfileCompositionSnapshot(profileID, profileMods)
	if err != nil {
		t.Fatalf("encodeProfileCompositionSnapshot() setup error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:                         gameID,
		ProfileID:                      profileID,
		ManifestJSON:                   `{"version":1}`,
		ProfileSnapshotJSON:            `{"version":1}`,
		ProfileSnapshotHash:            "snapshot",
		ProfileCompositionSnapshotJSON: &compositionSnapshot.JSON,
		ProfileCompositionSnapshotHash: &compositionSnapshot.Hash,
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() setup error = %v", err)
	}
}
