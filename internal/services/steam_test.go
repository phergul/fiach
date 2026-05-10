package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestSteamServiceGetsSteamLibrariesFromManualPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	extraLibrary := filepath.Join(t.TempDir(), "SteamLibrary")
	writeLibraryFoldersVDF(t, steamRoot, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+steamRoot+`"
	}
	"1"
	{
		"path"		"`+extraLibrary+`"
	}
}
`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	got, err := service.GetSteamLibraries()
	if err != nil {
		t.Fatalf("GetSteamLibraries() error = %v", err)
	}

	want := []string{filepath.Clean(steamRoot), filepath.Clean(extraLibrary)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSteamLibraries() = %#v, want %#v", got, want)
	}
}

func TestSteamServiceReturnsLibraryFolderParseError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	_, err := service.GetSteamLibraries()
	if err == nil {
		t.Fatal("GetSteamLibraries() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "get Steam libraries") {
		t.Fatalf("GetSteamLibraries() error = %q, want library context", err.Error())
	}
	if !strings.Contains(err.Error(), "parse libraryfolders.vdf") {
		t.Fatalf("GetSteamLibraries() error = %q, want parse context", err.Error())
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

func writeFile(t *testing.T, path string, content ...string) {
	t.Helper()

	fileContent := "x"
	if len(content) > 0 {
		fileContent = content[0]
	}

	if err := os.WriteFile(path, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeLibraryFoldersVDF(t *testing.T, root string, content string) {
	t.Helper()

	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), content)
}
