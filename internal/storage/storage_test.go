package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestOpenCreatesDatabaseFile(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store, err := Open(context.Background(), Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeStore(t, store)

	expectedPath := filepath.Join(dataDir, defaultAppName, databaseName)
	if store.Path() != expectedPath {
		t.Fatalf("Path() = %q, want %q", store.Path(), expectedPath)
	}

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected database file to exist: %v", err)
	}

	if store.DB() == nil {
		t.Fatal("DB() returned nil")
	}
}

func TestOpenWithInvalidDataDirReturnsClearError(t *testing.T) {
	t.Parallel()

	dataDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(dataDir, []byte("file"), 0644); err != nil {
		t.Fatalf("write invalid data dir file: %v", err)
	}

	store, err := Open(context.Background(), Options{DataDir: dataDir})
	if err == nil {
		_ = store.Close()
		t.Fatal("Open() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "create database directory") {
		t.Fatalf("Open() error = %q, want create database directory context", err.Error())
	}
}

func TestMigrateUpCreatesCoreTables(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	version, err := gooseVersion(store.DB().DB)
	if err != nil {
		t.Fatalf("gooseVersion() error = %v", err)
	}

	if version != 3 {
		t.Fatalf("goose version = %d, want 3", version)
	}

	for _, table := range []string{
		"games",
		"mods",
		"profiles",
		"profile_mods",
		"applied_manifests",
		"settings",
	} {
		if !tableExists(t, store, table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}
}

func TestMigrateUpAddsSteamGameDetectionColumns(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	for _, column := range []string{"source", "source_id", "available", "last_seen_at"} {
		if !columnExists(t, store, "games", column) {
			t.Fatalf("expected games.%s column to exist", column)
		}
	}

	if !indexExists(t, store, "idx_games_source_source_id") {
		t.Fatal("expected idx_games_source_source_id to exist")
	}
}

func TestMigrateUpAddsProfileActiveState(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if !columnExists(t, store, "profiles", "is_active") {
		t.Fatal("expected profiles.is_active column to exist")
	}

	if !indexExists(t, store, "idx_profiles_active_game_id") {
		t.Fatal("expected idx_profiles_active_game_id to exist")
	}
}

func TestProfileActiveStateDownMigrationPreservesProfiles(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, "Skyrim", "/games/skyrim")
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("game LastInsertId(): %v", err)
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name, is_active)
		VALUES (?, ?, 1)
	`, gameID, "Default"); err != nil {
		t.Fatalf("insert profile: %v", err)
	}

	if err := runGoose(store.DB().DB, func(db *sql.DB, dir string, opts ...goose.OptionsFunc) error {
		return goose.DownTo(db, dir, 2, opts...)
	}); err != nil {
		t.Fatalf("goose DownTo(2) error = %v", err)
	}

	if columnExists(t, store, "profiles", "is_active") {
		t.Fatal("profiles.is_active exists after down migration, want removed")
	}

	var count int
	if err := store.DB().Get(&count, `
		SELECT COUNT(*)
		FROM profiles
		WHERE game_id = ?
			AND name = ?
	`, gameID, "Default"); err != nil {
		t.Fatalf("count preserved profiles: %v", err)
	}
	if count != 1 {
		t.Fatalf("preserved profile count = %d, want 1", count)
	}
}

func TestMigrateDownDropsCoreTables(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}
	if err := store.MigrateDown(); err != nil {
		t.Fatalf("MigrateDown() error = %v", err)
	}

	version, err := gooseVersion(store.DB().DB)
	if err != nil {
		t.Fatalf("gooseVersion() error = %v", err)
	}

	if version != 0 {
		t.Fatalf("goose version = %d, want 0", version)
	}

	for _, table := range []string{
		"games",
		"mods",
		"profiles",
		"profile_mods",
		"applied_manifests",
		"settings",
	} {
		if tableExists(t, store, table) {
			t.Fatalf("expected table %q to be dropped", table)
		}
	}
}

func TestOpenConfiguresSQLiteConnection(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	var enabled int
	if err := store.DB().Get(&enabled, "PRAGMA foreign_keys"); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}

	if enabled != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", enabled)
	}

	var busyTimeout int
	if err := store.DB().Get(&busyTimeout, "PRAGMA busy_timeout"); err != nil {
		t.Fatalf("query busy_timeout pragma: %v", err)
	}

	if busyTimeout != 5000 {
		t.Fatalf("PRAGMA busy_timeout = %d, want 5000", busyTimeout)
	}

	var journalMode string
	if err := store.DB().Get(&journalMode, "PRAGMA journal_mode"); err != nil {
		t.Fatalf("query journal_mode pragma: %v", err)
	}

	if journalMode != "wal" {
		t.Fatalf("PRAGMA journal_mode = %q, want %q", journalMode, "wal")
	}
}

func TestSettingsCanBeSetUpdatedAndRead(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	value, found, err := store.GetSetting(context.Background(), "steam.install_path")
	if err != nil {
		t.Fatalf("GetSetting() missing error = %v", err)
	}
	if found {
		t.Fatalf("GetSetting() found = true, want false with value %q", value)
	}

	if err := store.SetSetting(context.Background(), "steam.install_path", "/steam/one"); err != nil {
		t.Fatalf("SetSetting() insert error = %v", err)
	}
	if err := store.SetSetting(context.Background(), "steam.install_path", "/steam/two"); err != nil {
		t.Fatalf("SetSetting() update error = %v", err)
	}

	value, found, err = store.GetSetting(context.Background(), "steam.install_path")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}
	if !found {
		t.Fatal("GetSetting() found = false, want true")
	}
	if value != "/steam/two" {
		t.Fatalf("GetSetting() value = %q, want updated value", value)
	}
}

func TestListStoredGamesReturnsAvailableGamesOrderedByName(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	insertStoredGame(t, store, "Zoo Game", "/games/zoo", GameSourceSteam, "30", true)
	insertStoredGame(t, store, "alpha Game", "/games/alpha", GameSourceSteam, "10", true)
	insertStoredGame(t, store, "Missing Game", "/games/missing", GameSourceSteam, "20", false)

	games, err := store.ListStoredGames(context.Background())
	if err != nil {
		t.Fatalf("ListStoredGames() error = %v", err)
	}

	if len(games) != 2 {
		t.Fatalf("ListStoredGames() length = %d, want 2: %#v", len(games), games)
	}
	if games[0].Name != "alpha Game" || games[1].Name != "Zoo Game" {
		t.Fatalf("ListStoredGames() names = %q, %q; want alpha Game, Zoo Game", games[0].Name, games[1].Name)
	}
	for _, game := range games {
		if !game.Available {
			t.Fatalf("ListStoredGames() returned unavailable game: %+v", game)
		}
	}
}

func TestListStoredGamesReturnsEmptyForEmptyDatabase(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	games, err := store.ListStoredGames(context.Background())
	if err != nil {
		t.Fatalf("ListStoredGames() error = %v", err)
	}
	if len(games) != 0 {
		t.Fatalf("ListStoredGames() length = %d, want 0", len(games))
	}
}

func TestForeignKeyConstraintRejectsMissingParent(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	_, err := store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path)
		VALUES (?, ?, ?)
	`, 999, "Missing Game Mod", "/mods/missing-game")
	if err == nil {
		t.Fatal("insert mod with missing game succeeded, want foreign key error")
	}
}

func TestDeletingGameCascadesDependents(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	result, err := store.DB().Exec(`
		INSERT INTO games (name, install_path)
		VALUES (?, ?)
	`, "Skyrim", "/games/skyrim")
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("game LastInsertId(): %v", err)
	}

	result, err = store.DB().Exec(`
		INSERT INTO mods (game_id, name, source_path)
		VALUES (?, ?, ?)
	`, gameID, "SkyUI", "/mods/skyui")
	if err != nil {
		t.Fatalf("insert mod: %v", err)
	}
	modID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("mod LastInsertId(): %v", err)
	}

	result, err = store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, "Default")
	if err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	profileID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("profile LastInsertId(): %v", err)
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
		VALUES (?, ?, ?, ?)
	`, profileID, modID, 1, 0); err != nil {
		t.Fatalf("insert profile_mod: %v", err)
	}

	if _, err := store.DB().Exec(`
		INSERT INTO applied_manifests (profile_id, mod_id, source_path, destination_path, checksum, file_size)
		VALUES (?, ?, ?, ?, ?, ?)
	`, profileID, modID, "/mods/skyui/file.esp", "/games/skyrim/Data/file.esp", "abc123", 42); err != nil {
		t.Fatalf("insert applied_manifest: %v", err)
	}

	if _, err := store.DB().Exec("DELETE FROM games WHERE id = ?", gameID); err != nil {
		t.Fatalf("delete game: %v", err)
	}

	for _, table := range []string{"games", "mods", "profiles", "profile_mods", "applied_manifests"} {
		var count int
		if err := store.DB().Get(&count, "SELECT COUNT(*) FROM "+table); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}

func TestMigrateUpCanReopenWithoutReapplying(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	store, err := Open(context.Background(), Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("first MigrateUp() error = %v", err)
	}
	closeStore(t, store)

	reopened, err := Open(context.Background(), Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("reopen Open() error = %v", err)
	}
	defer closeStore(t, reopened)

	if err := reopened.MigrateUp(); err != nil {
		t.Fatalf("second MigrateUp() error = %v", err)
	}

	version, err := gooseVersion(reopened.DB().DB)
	if err != nil {
		t.Fatalf("gooseVersion() error = %v", err)
	}

	if version != 3 {
		t.Fatalf("goose version = %d, want 3", version)
	}
}

func openStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(context.Background(), Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	return store
}

func closeStore(t *testing.T, store *Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func insertStoredGame(t *testing.T, store *Store, name string, installPath string, source string, sourceID string, available bool) {
	t.Helper()

	availableValue := 0
	if available {
		availableValue = 1
	}

	if _, err := store.DB().Exec(`
		INSERT INTO games (name, install_path, source, source_id, available, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, installPath, source, sourceID, availableValue, "2026-05-10T00:00:00Z"); err != nil {
		t.Fatalf("insert stored game: %v", err)
	}
}

func tableExists(t *testing.T, store *Store, table string) bool {
	t.Helper()

	var count int
	if err := store.DB().Get(&count, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'table'
			AND name = ?
	`, table); err != nil {
		t.Fatalf("query table %q existence: %v", table, err)
	}

	return count == 1
}

func columnExists(t *testing.T, store *Store, table string, column string) bool {
	t.Helper()

	rows, err := store.DB().Queryx("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("query table %q columns: %v", table, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Fatalf("close table info rows: %v", err)
		}
	}()

	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}

	return false
}

func indexExists(t *testing.T, store *Store, index string) bool {
	t.Helper()

	var count int
	if err := store.DB().Get(&count, `
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'index'
			AND name = ?
	`, index); err != nil {
		t.Fatalf("query index %q existence: %v", index, err)
	}

	return count == 1
}
