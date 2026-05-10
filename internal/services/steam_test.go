package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/steam"
	"github.com/phergul/mod-manager/internal/storage"
)

func TestSteamServiceLocatesManualSteamPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	got, err := service.LocateSteamInstallation()
	if err != nil {
		t.Fatalf("LocateSteamInstallation() error = %v", err)
	}

	if got.Root != filepath.Clean(steamRoot) {
		t.Fatalf("Root = %q, want %q", got.Root, steamRoot)
	}
}

func TestSteamServiceReturnsClearNotFoundError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	_, err := service.LocateSteamInstallation()
	if !errors.Is(err, steam.ErrSteamNotFound) {
		t.Fatalf("LocateSteamInstallation() error = %v, want ErrSteamNotFound", err)
	}
	if !strings.Contains(err.Error(), "Steam installation could not be found") {
		t.Fatalf("LocateSteamInstallation() error = %q, want clear message", err.Error())
	}
}

func openMigratedStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.Open(context.Background(), storage.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	return store
}

func closeStore(t *testing.T, store *storage.Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func createSteamRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "steamapps"))
	mkdirAll(t, filepath.Join(root, "userdata"))
	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"))

	return root
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
