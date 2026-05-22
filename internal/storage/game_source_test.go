package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSaveSourceScanInsertsNewGames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	result, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal", "/games/Portal"),
		sourceGame("20", "Half-Life", "/games/Half-Life"),
	})
	if err != nil {
		t.Fatalf("SaveSourceScan() error = %v", err)
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

func TestSaveSourceScanUpdatesExistingGameBySourceID(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Old Portal", "/games/OldPortal"),
	}); err != nil {
		t.Fatalf("first SaveSourceScan() error = %v", err)
	}

	result, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("second SaveSourceScan() error = %v", err)
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

func TestSaveSourceScanAttachesExistingInstallPathToSource(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	insertManualGame(t, store, "Portal", "/games/Portal")

	result, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal Updated", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("SaveSourceScan() error = %v", err)
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

func TestSaveSourceScanMarksMissingSourceGamesUnavailable(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	insertManualGame(t, store, "Manual", "/games/Manual")
	if _, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal", "/games/Portal"),
		sourceGame("20", "Half-Life", "/games/Half-Life"),
	}); err != nil {
		t.Fatalf("first SaveSourceScan() error = %v", err)
	}

	result, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal", "/games/Portal"),
	})
	if err != nil {
		t.Fatalf("second SaveSourceScan() error = %v", err)
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

func TestSaveSourceScanEmptyScanMarksAllSourceGamesUnavailable(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.SaveSourceScan(context.Background(), GameSourceSteam, []SourceGame{
		sourceGame("10", "Portal", "/games/Portal"),
	}); err != nil {
		t.Fatalf("first SaveSourceScan() error = %v", err)
	}

	result, err := store.SaveSourceScan(context.Background(), GameSourceSteam, nil)
	if err != nil {
		t.Fatalf("empty SaveSourceScan() error = %v", err)
	}

	if result.MarkedUnavailable != 1 {
		t.Fatalf("MarkedUnavailable = %d, want 1", result.MarkedUnavailable)
	}
	if storedGameAvailable(t, store, GameSourceSteam, "10") {
		t.Fatal("Steam app 10 available = true, want false")
	}
}

func sourceGame(sourceID string, name string, installPath string) SourceGame {
	return SourceGame{
		SourceID:    sourceID,
		Name:        name,
		InstallPath: installPath,
	}
}
