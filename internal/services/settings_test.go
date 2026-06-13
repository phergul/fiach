package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestSettingsServiceGetsAndSetsGlobalModStorageRoot(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewSettingsService(store, testLogger())
	if err := service.SetGlobalModStorageRoot(context.Background(), "/mods/root"); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}

	root, err := service.GetGlobalModStorageRoot(context.Background())
	if err != nil {
		t.Fatalf("GetGlobalModStorageRoot() error = %v", err)
	}
	wantRoot := filepath.Clean("/mods/root")
	if root != wantRoot {
		t.Fatalf("GetGlobalModStorageRoot() = %q, want %q", root, wantRoot)
	}
}

func TestSettingsServiceGetsAndSetsThemeID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewSettingsService(store, testLogger())
	if err := service.SetThemeID(context.Background(), "midnight"); err != nil {
		t.Fatalf("SetThemeID() error = %v", err)
	}

	themeID, err := service.GetThemeID(context.Background())
	if err != nil {
		t.Fatalf("GetThemeID() error = %v", err)
	}
	if themeID != "midnight" {
		t.Fatalf("GetThemeID() = %q, want midnight", themeID)
	}
}

func TestSettingsServiceSetThemeIDRejectsBlankValues(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewSettingsService(store, testLogger())
	if err := service.SetThemeID(context.Background(), "   "); err == nil {
		t.Fatal("SetThemeID() error = nil, want validation error")
	}
}

func TestSettingsServiceResolvesGameModStoragePathWithOverride(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertSettingsServiceTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewSettingsService(store, testLogger())
	if err := service.SetGlobalModStorageRoot(context.Background(), "/managed/root"); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}
	if _, err := service.SetGameModStoragePathOverride(context.Background(), gameID, "/override/root"); err != nil {
		t.Fatalf("SetGameModStoragePathOverride() error = %v", err)
	}

	path, err := service.ResolveGameModStoragePath(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ResolveGameModStoragePath() error = %v", err)
	}
	wantPath := filepath.Clean("/override/root")
	if path != wantPath {
		t.Fatalf("ResolveGameModStoragePath() = %q, want %q", path, wantPath)
	}
}

func TestSettingsServiceEnsuresGameModStoragePath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertSettingsServiceTestGame(t, store, "Skyrim", "/games/skyrim")
	root := filepath.Join(t.TempDir(), "managed")
	service := NewSettingsService(store, testLogger())
	if err := service.SetGlobalModStorageRoot(context.Background(), root); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}

	path, err := service.EnsureGameModStoragePath(context.Background(), gameID)
	if err != nil {
		t.Fatalf("EnsureGameModStoragePath() error = %v", err)
	}

	want := filepath.Join(root, storage.DefaultGameModStorageFolderName(dbtypes.StoredGame{ID: gameID}))
	if path != want {
		t.Fatalf("EnsureGameModStoragePath() = %q, want %q", path, want)
	}
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", want, err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(%q).IsDir() = false, want true", want)
	}
}

func insertSettingsServiceTestGame(t *testing.T, store *storage.Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, installPath)
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("game LastInsertId(): %v", err)
	}

	return id
}
