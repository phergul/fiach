package storage

import (
	"context"
	"strings"
	"testing"
)

func TestMigrateUpAddsAppliedProfileStateTable(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if !tableExists(t, store, "applied_profile_states") {
		t.Fatal("expected applied_profile_states table to exist")
	}
	for _, column := range []string{"game_id", "profile_id", "manifest_json", "profile_snapshot_json", "profile_snapshot_hash", "applied_at"} {
		if !columnExists(t, store, "applied_profile_states", column) {
			t.Fatalf("expected applied_profile_states.%s column to exist", column)
		}
	}
	if !indexExists(t, store, "idx_applied_profile_states_profile_id") {
		t.Fatal("expected idx_applied_profile_states_profile_id to exist")
	}
}

func TestSaveAppliedProfileStateInsertsReadsAndReplacesCurrentGameState(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	firstProfile := mustCreateProfile(t, store, gameID, "First")
	secondProfile := mustCreateProfile(t, store, gameID, "Second")

	first, err := store.SaveAppliedProfileState(context.Background(), SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           firstProfile.ID,
		ManifestJSON:        `{"version":1,"addedFiles":[]}`,
		ProfileSnapshotJSON: `{"version":1,"operations":[{"operationIndex":0}]}`,
		ProfileSnapshotHash: "first-hash",
	})
	if err != nil {
		t.Fatalf("SaveAppliedProfileState() insert error = %v", err)
	}
	if first.GameID != gameID || first.ProfileID != firstProfile.ID || first.ProfileSnapshotHash != "first-hash" || first.AppliedAt == "" {
		t.Fatalf("inserted applied profile state = %+v, want first profile state", first)
	}

	replaced, err := store.SaveAppliedProfileState(context.Background(), SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           secondProfile.ID,
		ManifestJSON:        `{"version":1,"addedFiles":[{"targetPath":"Data/SkyUI.esp"}]}`,
		ProfileSnapshotJSON: `{"version":1,"operations":[{"operationIndex":1}]}`,
		ProfileSnapshotHash: "second-hash",
	})
	if err != nil {
		t.Fatalf("SaveAppliedProfileState() replace error = %v", err)
	}
	if replaced.GameID != gameID || replaced.ProfileID != secondProfile.ID || replaced.ProfileSnapshotHash != "second-hash" {
		t.Fatalf("replaced applied profile state = %+v, want second profile state", replaced)
	}

	read, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if !found {
		t.Fatal("GetAppliedProfileState() found = false, want true")
	}
	if read.ProfileID != secondProfile.ID || read.ManifestJSON != replaced.ManifestJSON || read.ProfileSnapshotJSON != replaced.ProfileSnapshotJSON || read.ProfileSnapshotHash != "second-hash" {
		t.Fatalf("GetAppliedProfileState() = %+v, want replaced state", read)
	}

	var count int
	if err := store.DB().Get(&count, "SELECT COUNT(*) FROM applied_profile_states WHERE game_id = ?", gameID); err != nil {
		t.Fatalf("count applied profile states: %v", err)
	}
	if count != 1 {
		t.Fatalf("applied_profile_states count = %d, want 1", count)
	}
}

func TestGetAppliedProfileStateReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	_, found, err := store.GetAppliedProfileState(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if found {
		t.Fatal("GetAppliedProfileState() found = true, want false")
	}
}

func TestSaveAppliedProfileStateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	tests := []struct {
		name  string
		input SaveAppliedProfileStateInput
		want  string
	}{
		{
			name: "missing game",
			input: SaveAppliedProfileStateInput{
				ProfileID:           1,
				ManifestJSON:        `{"version":1}`,
				ProfileSnapshotJSON: `{"version":1}`,
				ProfileSnapshotHash: "hash",
			},
			want: "game ID must be positive",
		},
		{
			name: "missing profile",
			input: SaveAppliedProfileStateInput{
				GameID:              1,
				ManifestJSON:        `{"version":1}`,
				ProfileSnapshotJSON: `{"version":1}`,
				ProfileSnapshotHash: "hash",
			},
			want: "profile ID must be positive",
		},
		{
			name: "invalid manifest JSON",
			input: SaveAppliedProfileStateInput{
				GameID:              1,
				ProfileID:           1,
				ManifestJSON:        `{`,
				ProfileSnapshotJSON: `{"version":1}`,
				ProfileSnapshotHash: "hash",
			},
			want: "manifest JSON is invalid",
		},
		{
			name: "invalid snapshot JSON",
			input: SaveAppliedProfileStateInput{
				GameID:              1,
				ProfileID:           1,
				ManifestJSON:        `{"version":1}`,
				ProfileSnapshotJSON: `{`,
				ProfileSnapshotHash: "hash",
			},
			want: "profile snapshot JSON is invalid",
		},
		{
			name: "missing hash",
			input: SaveAppliedProfileStateInput{
				GameID:              1,
				ProfileID:           1,
				ManifestJSON:        `{"version":1}`,
				ProfileSnapshotJSON: `{"version":1}`,
			},
			want: "profile snapshot hash is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.SaveAppliedProfileState(context.Background(), tt.input)
			if err == nil {
				t.Fatal("SaveAppliedProfileState() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), "upsert applied profile state row") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("SaveAppliedProfileState() error = %q, want context and %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSaveAppliedProfileStateRejectsProfileFromAnotherGame(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	firstGameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	secondGameID := insertProfileTestGame(t, store, "Fallout", "/games/fallout")
	profile := mustCreateProfile(t, store, secondGameID, "Default")

	_, err := store.SaveAppliedProfileState(context.Background(), SaveAppliedProfileStateInput{
		GameID:              firstGameID,
		ProfileID:           profile.ID,
		ManifestJSON:        `{"version":1}`,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "hash",
	})
	if err == nil {
		t.Fatal("SaveAppliedProfileState() error = nil, want profile/game mismatch")
	}
	if !strings.Contains(err.Error(), "profile") || !strings.Contains(err.Error(), "does not belong to game") {
		t.Fatalf("SaveAppliedProfileState() error = %q, want profile/game mismatch detail", err.Error())
	}
}
