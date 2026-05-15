package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/phergul/mod-manager/internal/steam"
)

func TestSaveSteamScanInsertsNewGames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	result, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal", "/games/Portal"),
		steamGame("20", "Half-Life", "/games/Half-Life"),
	})
	if err != nil {
		t.Fatalf("SaveSteamScan() error = %v", err)
	}

	if result.Inserted != 2 || result.Updated != 0 || result.MarkedUnavailable != 0 {
		t.Fatalf("result = %+v, want 2 inserted only", result)
	}
	if len(result.Games) != 2 {
		t.Fatalf("Games length = %d, want 2", len(result.Games))
	}

	for _, game := range result.Games {
		if game.Source != GameSourceSteam {
			t.Fatalf("Source = %q, want steam", game.Source)
		}
		if !game.Available {
			t.Fatal("Available = false, want true")
		}
		if game.SourceID == nil || *game.SourceID == "" {
			t.Fatalf("SourceID = %v, want non-empty pointer", game.SourceID)
		}
		if game.LastSeenAt == nil || *game.LastSeenAt == "" {
			t.Fatalf("LastSeenAt = %v, want non-empty pointer", game.LastSeenAt)
		}
		wantModStoragePath := filepath.Join(filepath.Dir(store.Path()), "mods", DefaultGameModStorageFolderName(game))
		if game.ModStoragePath == nil || *game.ModStoragePath != wantModStoragePath {
			t.Fatalf("ModStoragePath = %v, want %q", game.ModStoragePath, wantModStoragePath)
		}

		stored, err := store.GetStoredGame(context.Background(), game.ID)
		if err != nil {
			t.Fatalf("GetStoredGame() error = %v", err)
		}
		if stored.ModStoragePath == nil || *stored.ModStoragePath != wantModStoragePath {
			t.Fatalf("GetStoredGame().ModStoragePath = %v, want %q", stored.ModStoragePath, wantModStoragePath)
		}
	}

	storedGames, err := store.ListStoredGames(context.Background())
	if err != nil {
		t.Fatalf("ListStoredGames() error = %v", err)
	}
	for _, game := range storedGames {
		if game.ModStoragePath == nil || *game.ModStoragePath == "" {
			t.Fatalf("ListStoredGames() returned empty ModStoragePath: %+v", game)
		}
	}
}

func TestSaveSteamScanUpdatesExistingSteamGameByAppID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Old Portal", "/games/OldPortal"),
	}); err != nil {
		t.Fatalf("first SaveSteamScan() error = %v", err)
	}

	result, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("second SaveSteamScan() error = %v", err)
	}

	if result.Inserted != 0 || result.Updated != 1 {
		t.Fatalf("result = %+v, want one update", result)
	}
	if countGames(t, store) != 1 {
		t.Fatalf("game count = %d, want 1", countGames(t, store))
	}
	if result.Games[0].Name != "Portal" || result.Games[0].InstallPath != filepath.Clean("/games/Portal") {
		t.Fatalf("updated game = %+v, want new name and path", result.Games[0])
	}
}

func TestSaveSteamScanAttachesExistingInstallPathToSteam(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	insertManualGame(t, store, "Portal", "/games/Portal")

	result, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal Updated", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("SaveSteamScan() error = %v", err)
	}

	if result.Inserted != 0 || result.Updated != 1 {
		t.Fatalf("result = %+v, want one update", result)
	}
	if countGames(t, store) != 1 {
		t.Fatalf("game count = %d, want 1", countGames(t, store))
	}

	game := result.Games[0]
	if game.Source != GameSourceSteam || game.SourceID == nil || *game.SourceID != "10" || !game.Available {
		t.Fatalf("game metadata = %+v, want attached Steam metadata", game)
	}
}

func TestSaveSteamScanMarksMissingSteamGamesUnavailable(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	insertManualGame(t, store, "Manual", "/games/Manual")
	if _, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal", "/games/Portal"),
		steamGame("20", "Half-Life", "/games/Half-Life"),
	}); err != nil {
		t.Fatalf("first SaveSteamScan() error = %v", err)
	}

	result, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("second SaveSteamScan() error = %v", err)
	}

	if result.MarkedUnavailable != 1 {
		t.Fatalf("MarkedUnavailable = %d, want 1", result.MarkedUnavailable)
	}
	if !storedGameAvailable(t, store, GameSourceSteam, "10") {
		t.Fatal("Steam app 10 available = false, want true")
	}
	if storedGameAvailable(t, store, GameSourceSteam, "20") {
		t.Fatal("Steam app 20 available = true, want false")
	}
	if !storedGameByInstallPathAvailable(t, store, "/games/Manual") {
		t.Fatal("manual game available = false, want true")
	}
}

func TestSaveSteamScanEmptyScanMarksAllSteamGamesUnavailable(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.SaveSteamScan(context.Background(), []steam.Game{
		steamGame("10", "Portal", "/games/Portal"),
	}); err != nil {
		t.Fatalf("first SaveSteamScan() error = %v", err)
	}

	result, err := store.SaveSteamScan(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty SaveSteamScan() error = %v", err)
	}

	if result.MarkedUnavailable != 1 {
		t.Fatalf("MarkedUnavailable = %d, want 1", result.MarkedUnavailable)
	}
	if storedGameAvailable(t, store, GameSourceSteam, "10") {
		t.Fatal("Steam app 10 available = true, want false")
	}
}

func steamGame(appID string, name string, installPath string) steam.Game {
	return steam.Game{
		AppID:       appID,
		Name:        name,
		InstallPath: installPath,
	}
}

func openMigratedStore(t *testing.T) *Store {
	t.Helper()

	store := openStore(t)
	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	return store
}

func insertManualGame(t *testing.T, store *Store, name string, installPath string) {
	t.Helper()

	_, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, name, filepath.Clean(installPath))
	if err != nil {
		t.Fatalf("insert manual game: %v", err)
	}
}

func countGames(t *testing.T, store *Store) int {
	t.Helper()

	var count int
	if err := store.DB().Get(&count, "SELECT COUNT(*) FROM games"); err != nil {
		t.Fatalf("count games: %v", err)
	}

	return count
}

func storedGameAvailable(t *testing.T, store *Store, source string, sourceID string) bool {
	t.Helper()

	var available bool
	err := store.DB().Get(&available, `
		SELECT available
		FROM games
		WHERE source = ?
			AND source_id = ?
	`, source, sourceID)
	if err != nil {
		t.Fatalf("get available: %v", err)
	}

	return available
}

func storedGameByInstallPathAvailable(t *testing.T, store *Store, installPath string) bool {
	t.Helper()

	var available bool
	err := store.DB().Get(&available, `
		SELECT available
		FROM games
		WHERE install_path = ?
	`, filepath.Clean(installPath))
	if err != nil {
		t.Fatalf("get available by path: %v", err)
	}

	return available
}
