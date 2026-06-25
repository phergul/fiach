package storage

import (
	"context"
	"errors"
	"testing"
)

func TestMapSQLiteErrorMapsDuplicateProfileName(t *testing.T) {
	t.Parallel()

	raw := errors.New("constraint failed: UNIQUE constraint failed: profiles.game_id, profiles.name (2067)")
	mapped := mapSQLiteError(raw)

	if !errors.Is(mapped, ErrDuplicateProfileName) {
		t.Fatalf("errors.Is(mapped, ErrDuplicateProfileName) = false, mapped = %v", mapped)
	}
}

func TestMapSQLiteErrorMapsDuplicateTagName(t *testing.T) {
	t.Parallel()

	raw := errors.New("constraint failed: UNIQUE constraint failed: tags.game_id, tags.normalized_name (2067)")
	mapped := mapSQLiteError(raw)

	if !errors.Is(mapped, ErrDuplicateTagName) {
		t.Fatalf("errors.Is(mapped, ErrDuplicateTagName) = false, mapped = %v", mapped)
	}
}

func TestMapSQLiteErrorLeavesUnknownErrorsUnchanged(t *testing.T) {
	t.Parallel()

	raw := errors.New("disk I/O error")
	mapped := mapSQLiteError(raw)

	if !errors.Is(mapped, raw) {
		t.Fatalf("mapped = %v, want original error", mapped)
	}
}

func TestMapSQLiteErrorFromStorageDuplicateProfile(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	if _, err := store.CreateProfile(context.Background(), gameID, "Default"); err != nil {
		t.Fatalf("CreateProfile() initial error = %v", err)
	}

	_, err := store.CreateProfile(context.Background(), gameID, "Default")
	if err == nil {
		t.Fatal("CreateProfile() duplicate error = nil, want error")
	}
	if !errors.Is(err, ErrDuplicateProfileName) {
		t.Fatalf("errors.Is(err, ErrDuplicateProfileName) = false, err = %v", err)
	}
}
