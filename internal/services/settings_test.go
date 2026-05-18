package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/storage"
)

func TestSettingsServiceGetsAndSetsGlobalModStorageRoot(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewSettingsService(store)
	if err := service.SetGlobalModStorageRoot("/mods/root"); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}

	root, err := service.GetGlobalModStorageRoot()
	if err != nil {
		t.Fatalf("GetGlobalModStorageRoot() error = %v", err)
	}
	if root != "/mods/root" {
		t.Fatalf("GetGlobalModStorageRoot() = %q, want /mods/root", root)
	}
}

func TestSettingsServiceGetsAndSetsThemeID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	service := NewSettingsService(store)
	if err := service.SetThemeID("midnight"); err != nil {
		t.Fatalf("SetThemeID() error = %v", err)
	}

	themeID, err := service.GetThemeID()
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

	service := NewSettingsService(store)
	if err := service.SetThemeID("   "); err == nil {
		t.Fatal("SetThemeID() error = nil, want validation error")
	}
}

func TestSettingsServiceResolvesGameModStoragePathWithOverride(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertSettingsServiceTestGame(t, store, "Skyrim", "/games/skyrim")
	service := NewSettingsService(store)
	if err := service.SetGlobalModStorageRoot("/managed/root"); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}
	if _, err := service.SetGameModStoragePathOverride(gameID, "/override/root"); err != nil {
		t.Fatalf("SetGameModStoragePathOverride() error = %v", err)
	}

	path, err := service.ResolveGameModStoragePath(gameID)
	if err != nil {
		t.Fatalf("ResolveGameModStoragePath() error = %v", err)
	}
	if path != "/override/root" {
		t.Fatalf("ResolveGameModStoragePath() = %q, want override", path)
	}
}

func TestSettingsServiceEnsuresGameModStoragePath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertSettingsServiceTestGame(t, store, "Skyrim", "/games/skyrim")
	root := filepath.Join(t.TempDir(), "managed")
	service := NewSettingsService(store)
	if err := service.SetGlobalModStorageRoot(root); err != nil {
		t.Fatalf("SetGlobalModStorageRoot() error = %v", err)
	}

	path, err := service.EnsureGameModStoragePath(gameID)
	if err != nil {
		t.Fatalf("EnsureGameModStoragePath() error = %v", err)
	}

	want := filepath.Join(root, storage.DefaultGameModStorageFolderName(storage.StoredGame{ID: gameID}))
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

func TestSettingsServiceErrorsHaveDistinctServiceAndStorageContext(t *testing.T) {
	t.Parallel()

	service := NewSettingsService(nil)
	_, err := service.ResolveGameModStoragePath(1)
	if err == nil {
		t.Fatal("ResolveGameModStoragePath() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "resolve game mod storage path") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("ResolveGameModStoragePath() error = %q, want service and storage context", err.Error())
	}
}

func TestSettingsServiceThemeErrorsHaveDistinctServiceAndStorageContext(t *testing.T) {
	t.Parallel()

	service := NewSettingsService(nil)
	_, err := service.GetThemeID()
	if err == nil {
		t.Fatal("GetThemeID() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "get theme ID") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("GetThemeID() error = %q, want service and storage context", err.Error())
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
