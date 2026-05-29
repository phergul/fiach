package services

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/gamesource"
	"github.com/phergul/mod-manager/internal/storage"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func TestGamesServiceScansAndSavesGames(t *testing.T) {
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
	if err := store.SetSetting(context.Background(), gamesource.SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	steamSource := gamesource.NewSteamSource(store)
	service := NewGamesService(store, testLogger(), steamSource)
	result, err := service.ScanAndSaveGames(context.Background())
	if err != nil {
		t.Fatalf("ScanAndSaveGames() error = %v", err)
	}

	if result.Inserted != 2 || result.Updated != 0 || result.MarkedUnavailable != 0 {
		t.Fatalf("result = %+v, want 2 inserted only", result)
	}
	if len(result.Games) != 2 {
		t.Fatalf("Games length = %d, want 2", len(result.Games))
	}
}

func TestGamesServiceGetsStoredGames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.DB().Exec(`
		INSERT INTO games (name, install_path, source, source_id, available, last_seen_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, "Portal", "/games/Portal", dbtypes.GameSourceSteam, "400", "2026-05-10T00:00:00Z"); err != nil {
		t.Fatalf("insert stored game: %v", err)
	}

	service := NewGamesService(store, testLogger(), gamesource.NewSteamSource(store))
	games, err := service.GetStoredGames(context.Background())
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

func TestGamesServiceScanAndSaveReturnsLibraryErrorWithoutWrites(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), gamesource.SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	steamSource := gamesource.NewSteamSource(store)
	service := NewGamesService(store, testLogger(), steamSource)
	_, err := service.ScanAndSaveGames(context.Background())
	if err == nil {
		t.Fatal("ScanAndSaveGames() error = nil, want error")
	}
	if !contains(err.Error(), "scan and save games") {
		t.Fatalf("ScanAndSaveGames() error = %q, want scan/save context", err.Error())
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

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}

func insertServiceTestGame(t *testing.T, store *storage.Store, name string, installPath string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path, source, available)
		VALUES (?, ?, ?, 1)
	`, name, installPath, dbtypes.GameSourceManual)
	if err != nil {
		t.Fatalf("insert service test game: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service test game LastInsertId(): %v", err)
	}

	return id
}
