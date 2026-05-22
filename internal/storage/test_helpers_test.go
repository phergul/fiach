package storage

import (
	"path/filepath"
	"testing"
)

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
