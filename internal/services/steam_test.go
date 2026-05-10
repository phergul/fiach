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

func TestSteamServiceGetsInstalledSteamGames(t *testing.T) {
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
	writeAppManifest(t, steamRoot, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	writeAppManifest(t, extraLibrary, "appmanifest_2.acf", validManifest("2", "Game Two", "GameTwo"))
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	got, err := service.GetInstalledSteamGames()
	if err != nil {
		t.Fatalf("GetInstalledSteamGames() error = %v", err)
	}

	want := []steam.Game{
		{
			AppID:        "1",
			Name:         "Game One",
			InstallDir:   "GameOne",
			LibraryPath:  filepath.Clean(steamRoot),
			InstallPath:  filepath.Join(steamRoot, "steamapps", "common", "GameOne"),
			ManifestPath: filepath.Join(steamRoot, "steamapps", "appmanifest_1.acf"),
		},
		{
			AppID:        "2",
			Name:         "Game Two",
			InstallDir:   "GameTwo",
			LibraryPath:  filepath.Clean(extraLibrary),
			InstallPath:  filepath.Join(extraLibrary, "steamapps", "common", "GameTwo"),
			ManifestPath: filepath.Join(extraLibrary, "steamapps", "appmanifest_2.acf"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetInstalledSteamGames() = %#v, want %#v", got, want)
	}
}

func TestSteamServiceReturnsInstalledGamesLibraryFolderError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	_, err := service.GetInstalledSteamGames()
	if err == nil {
		t.Fatal("GetInstalledSteamGames() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "get installed Steam games") {
		t.Fatalf("GetInstalledSteamGames() error = %q, want installed games context", err.Error())
	}
	if !strings.Contains(err.Error(), "get Steam libraries") {
		t.Fatalf("GetInstalledSteamGames() error = %q, want library context", err.Error())
	}
}

func TestSteamServiceScansAndSavesSteamGames(t *testing.T) {
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
	writeAppManifest(t, steamRoot, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	writeAppManifest(t, extraLibrary, "appmanifest_2.acf", validManifest("2", "Game Two", "GameTwo"))
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	result, err := service.ScanAndSaveSteamGames()
	if err != nil {
		t.Fatalf("ScanAndSaveSteamGames() error = %v", err)
	}

	if result.Inserted != 2 || result.Updated != 0 || result.MarkedUnavailable != 0 {
		t.Fatalf("result = %+v, want 2 inserted only", result)
	}
	if len(result.Games) != 2 {
		t.Fatalf("Games length = %d, want 2", len(result.Games))
	}
}

func TestSteamServiceGetsStoredGames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.DB().Exec(`
		INSERT INTO games (name, install_path, source, source_id, available, last_seen_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, "Portal", "/games/Portal", storage.GameSourceSteam, "400", "2026-05-10T00:00:00Z"); err != nil {
		t.Fatalf("insert stored game: %v", err)
	}

	service := NewSteamService(store)
	games, err := service.GetStoredGames()
	if err != nil {
		t.Fatalf("GetStoredGames() error = %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("GetStoredGames() length = %d, want 1", len(games))
	}
	if games[0].Name != "Portal" || games[0].InstallPath != "/games/Portal" {
		t.Fatalf("GetStoredGames() = %+v, want Portal with install path", games[0])
	}
}

func TestSteamServiceGetStoredGamesReturnsStorageError(t *testing.T) {
	t.Parallel()

	service := NewSteamService(nil)
	_, err := service.GetStoredGames()
	if err == nil {
		t.Fatal("GetStoredGames() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "get stored games") {
		t.Fatalf("GetStoredGames() error = %q, want service context", err.Error())
	}
}

func TestSteamServiceScanAndSaveReturnsLibraryErrorWithoutWrites(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	service := NewSteamService(store)
	_, err := service.ScanAndSaveSteamGames()
	if err == nil {
		t.Fatal("ScanAndSaveSteamGames() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "scan and save Steam games") {
		t.Fatalf("ScanAndSaveSteamGames() error = %q, want scan/save context", err.Error())
	}

	var count int
	if err := store.DB().Get(&count, "SELECT COUNT(*) FROM games"); err != nil {
		t.Fatalf("count games: %v", err)
	}
	if count != 0 {
		t.Fatalf("game count = %d, want 0", count)
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

func writeAppManifest(t *testing.T, libraryPath string, name string, content string) string {
	t.Helper()

	steamAppsPath := filepath.Join(libraryPath, "steamapps")
	mkdirAll(t, steamAppsPath)

	path := filepath.Join(steamAppsPath, name)
	writeFile(t, path, content)
	return path
}

func validManifest(appID string, name string, installDir string) string {
	return `
"AppState"
{
	"appid"		"` + appID + `"
	"name"		"` + name + `"
	"installdir"		"` + installDir + `"
}
`
}
