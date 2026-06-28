package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestMigrateUpAddsAppliedFileStatesTable(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if !tableExists(t, store, "applied_file_states") {
		t.Fatal("expected applied_file_states table to exist")
	}
	for _, column := range []string{
		"game_id",
		"game_relative_path",
		"profile_id",
		"baseline_exists",
		"baseline_sha256",
		"baseline_size_bytes",
		"baseline_backup_path",
		"applied_exists",
		"applied_sha256",
		"applied_size_bytes",
		"winning_source_kind",
		"winning_source_id",
		"winning_mod_id",
		"winning_load_order",
		"output_kind",
		"user_decision",
		"last_applied_at",
	} {
		if !columnExists(t, store, "applied_file_states", column) {
			t.Fatalf("expected applied_file_states.%s column to exist", column)
		}
	}
	if !indexExists(t, store, "idx_applied_file_states_profile_id") {
		t.Fatal("expected idx_applied_file_states_profile_id to exist")
	}
}

func TestReplaceAppliedFileStatesInsertsReadsAndReplacesCurrentGameState(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	saveAppliedProfileStateFixture(t, store, gameID, profile.ID)

	appliedSHA256 := "applied-sha"
	appliedSizeBytes := int64(42)
	winningModID := int64(10)
	winningSourceKind := "mod"
	winningSourceID := "10"

	first := dbtypes.ReplaceAppliedFileStatesInput{
		GameID:    gameID,
		ProfileID: profile.ID,
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:            gameID,
				GameRelativePath:  "Data/SkyUI.esp",
				ProfileID:         profile.ID,
				AppliedExists:     true,
				AppliedSHA256:     &appliedSHA256,
				AppliedSizeBytes:  &appliedSizeBytes,
				WinningSourceKind: &winningSourceKind,
				WinningSourceID:   &winningSourceID,
				WinningModID:      &winningModID,
				OutputKind:        "copied",
				LastAppliedAt:     "2026-06-27T00:00:00Z",
			},
		},
	}
	if err := store.ReplaceAppliedFileStates(context.Background(), first); err != nil {
		t.Fatalf("ReplaceAppliedFileStates() first error = %v", err)
	}

	replacedSHA256 := "replaced-sha"
	second := dbtypes.ReplaceAppliedFileStatesInput{
		GameID:    gameID,
		ProfileID: profile.ID,
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:            gameID,
				GameRelativePath:  "Data/Other.esp",
				ProfileID:         profile.ID,
				AppliedExists:     true,
				AppliedSHA256:     &replacedSHA256,
				WinningSourceKind: &winningSourceKind,
				WinningSourceID:   &winningSourceID,
				WinningModID:      &winningModID,
				OutputKind:        "copied",
				LastAppliedAt:     "2026-06-27T01:00:00Z",
			},
		},
	}
	if err := store.ReplaceAppliedFileStates(context.Background(), second); err != nil {
		t.Fatalf("ReplaceAppliedFileStates() replace error = %v", err)
	}

	rows, err := store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() error = %v", err)
	}
	if len(rows) != 1 || rows[0].GameRelativePath != "Data/Other.esp" || rows[0].AppliedSHA256 == nil || *rows[0].AppliedSHA256 != replacedSHA256 {
		t.Fatalf("ListAppliedFileStates() = %+v, want replaced row", rows)
	}
}

func TestHasAppliedFileStatesReturnsFalseForMissingGameState(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	found, err := store.HasAppliedFileStates(context.Background(), 999)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if found {
		t.Fatal("HasAppliedFileStates() found = true, want false")
	}
}

func TestDeleteAppliedProfileStateCascadesAppliedFileStates(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	saveAppliedProfileStateFixture(t, store, gameID, profile.ID)

	appliedSHA256 := "applied-sha"
	winningSourceKind := "mod"
	winningSourceID := "10"
	winningModID := int64(10)
	if err := store.ReplaceAppliedFileStates(context.Background(), dbtypes.ReplaceAppliedFileStatesInput{
		GameID:    gameID,
		ProfileID: profile.ID,
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:            gameID,
				GameRelativePath:  "Data/SkyUI.esp",
				ProfileID:         profile.ID,
				AppliedExists:     true,
				AppliedSHA256:     &appliedSHA256,
				WinningSourceKind: &winningSourceKind,
				WinningSourceID:   &winningSourceID,
				WinningModID:      &winningModID,
				OutputKind:        "copied",
				LastAppliedAt:     "2026-06-27T00:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("ReplaceAppliedFileStates() error = %v", err)
	}

	if err := store.DeleteAppliedProfileState(context.Background(), gameID); err != nil {
		t.Fatalf("DeleteAppliedProfileState() error = %v", err)
	}

	found, err := store.HasAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if found {
		t.Fatal("HasAppliedFileStates() found = true, want cascade delete")
	}
}

func TestSaveAppliedProfileStatePersistsFileStatesInSameTransaction(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	appliedSHA256 := "applied-sha"
	winningSourceKind := "mod"
	winningSourceID := "10"
	winningModID := int64(10)

	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profile.ID,
		ManifestJSON:        `{"version":2,"addedFiles":[],"replacedFiles":[],"createdDirectories":[]}`,
		ProfileSnapshotJSON: `{"version":2}`,
		ProfileSnapshotHash: "hash",
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:            gameID,
				GameRelativePath:  "Data/SkyUI.esp",
				ProfileID:         profile.ID,
				AppliedExists:     true,
				AppliedSHA256:     &appliedSHA256,
				WinningSourceKind: &winningSourceKind,
				WinningSourceID:   &winningSourceID,
				WinningModID:      &winningModID,
				OutputKind:        "copied",
			},
		},
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	rows, err := store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() error = %v", err)
	}
	if len(rows) != 1 || rows[0].LastAppliedAt == "" {
		t.Fatalf("ListAppliedFileStates() = %+v, want persisted file state with applied timestamp", rows)
	}
}

func TestReplaceAppliedFileStatesRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	err := store.ReplaceAppliedFileStates(context.Background(), dbtypes.ReplaceAppliedFileStatesInput{
		GameID:    0,
		ProfileID: 1,
	})
	if err == nil {
		t.Fatal("ReplaceAppliedFileStates() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "game ID must be positive") {
		t.Fatalf("ReplaceAppliedFileStates() error = %q, want game ID validation", err.Error())
	}
}

func TestUpdateAppliedFileStateUserDecisionSetsAndClearsWithoutCollapsingFields(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	profile := mustCreateProfile(t, store, gameID, "Default")
	saveAppliedProfileStateFixture(t, store, gameID, profile.ID)

	appliedSHA256 := "applied-sha"
	appliedSizeBytes := int64(42)
	if err := store.ReplaceAppliedFileStates(context.Background(), dbtypes.ReplaceAppliedFileStatesInput{
		GameID:    gameID,
		ProfileID: profile.ID,
		FileStates: []dbtypes.AppliedFileStateRow{
			{
				GameID:           gameID,
				GameRelativePath: "Data/SkyUI.esp",
				ProfileID:        profile.ID,
				AppliedExists:    true,
				AppliedSHA256:    &appliedSHA256,
				AppliedSizeBytes: &appliedSizeBytes,
				OutputKind:       "copied",
				LastAppliedAt:    "2026-06-27T00:00:00Z",
			},
		},
	}); err != nil {
		t.Fatalf("ReplaceAppliedFileStates() error = %v", err)
	}

	decision := "keep_external"
	if err := store.UpdateAppliedFileStateUserDecision(context.Background(), gameID, profile.ID, "Data/SkyUI.esp", &decision); err != nil {
		t.Fatalf("UpdateAppliedFileStateUserDecision() error = %v", err)
	}

	rows, err := store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("ListAppliedFileStates() count = %d, want 1", len(rows))
	}
	if rows[0].UserDecision == nil || *rows[0].UserDecision != decision {
		t.Fatalf("UserDecision = %+v, want %q", rows[0].UserDecision, decision)
	}
	if rows[0].AppliedSHA256 == nil || *rows[0].AppliedSHA256 != appliedSHA256 {
		t.Fatalf("AppliedSHA256 = %+v, want preserved", rows[0].AppliedSHA256)
	}

	if err := store.UpdateAppliedFileStateUserDecision(context.Background(), gameID, profile.ID, "Data/SkyUI.esp", nil); err != nil {
		t.Fatalf("UpdateAppliedFileStateUserDecision(clear) error = %v", err)
	}

	rows, err = store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() after clear error = %v", err)
	}
	if rows[0].UserDecision != nil {
		t.Fatalf("UserDecision after clear = %+v, want nil", rows[0].UserDecision)
	}
	if rows[0].AppliedSizeBytes == nil || *rows[0].AppliedSizeBytes != appliedSizeBytes {
		t.Fatalf("AppliedSizeBytes = %+v, want preserved", rows[0].AppliedSizeBytes)
	}
}

func saveAppliedProfileStateFixture(t *testing.T, store *Store, gameID int64, profileID int64) {
	t.Helper()

	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        `{"version":1,"addedFiles":[],"replacedFiles":[],"createdDirectories":[]}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "hash",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() fixture error = %v", err)
	}
}
