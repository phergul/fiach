package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestLegacyAppliedStateBackfillMigratesManifestRows(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := migrateUpTo(store.DB().DB, 3); err != nil {
		t.Fatalf("migrateUpTo(3) error = %v", err)
	}

	gameRoot := t.TempDir()
	insertManualGame(t, store, "Skyrim", gameRoot)

	var gameID int64
	if err := store.DB().Get(&gameID, `
		SELECT id
		FROM games
		WHERE install_path = ?
	`, filepath.Clean(gameRoot)); err != nil {
		t.Fatalf("select game id: %v", err)
	}

	var profileID int64
	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, "Default")
	if err != nil {
		t.Fatalf("insert profile: %v", err)
	}
	profileID, err = result.LastInsertId()
	if err != nil {
		t.Fatalf("profile LastInsertId(): %v", err)
	}

	for _, column := range []string{
		"manifest_json TEXT NOT NULL DEFAULT ''",
		"profile_snapshot_json TEXT NOT NULL DEFAULT ''",
		"profile_snapshot_hash TEXT NOT NULL DEFAULT ''",
	} {
		if _, err := store.DB().Exec(`ALTER TABLE applied_profile_states ADD COLUMN ` + column); err != nil {
			t.Fatalf("add legacy column %q: %v", column, err)
		}
	}

	addedTargetPath := filepath.Join(gameRoot, "Data", "added.esp")
	createdDirectoryPath := filepath.Join(gameRoot, "Mods", "Created")
	manifestJSON := `{
		"version":1,
		"addedFiles":[{
			"operationIndex":0,
			"mod":{"id":10,"name":"SkyUI"},
			"targetPath":"` + filepathToSlash(addedTargetPath) + `",
			"sha256":"added-sha",
			"sizeBytes":5
		}],
		"replacedFiles":[],
		"createdDirectories":[{
			"operationIndex":1,
			"mod":{"id":10,"name":"SkyUI"},
			"targetPath":"` + filepathToSlash(createdDirectoryPath) + `"
		}]
	}`

	if _, err := store.DB().Exec(`
		INSERT INTO applied_profile_states (
			game_id,
			profile_id,
			manifest_json,
			profile_snapshot_json,
			profile_snapshot_hash,
			applied_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`, gameID, profileID, manifestJSON, `{"version":1}`, "snapshot-hash", "2026-06-27T00:00:00Z"); err != nil {
		t.Fatalf("insert legacy applied profile state: %v", err)
	}

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	version, err := gooseVersion(store.DB().DB)
	if err != nil {
		t.Fatalf("gooseVersion() error = %v", err)
	}
	if version != 5 {
		t.Fatalf("goose version = %d, want 5", version)
	}

	if columnExists(t, store, "applied_profile_states", "manifest_json") {
		t.Fatal("expected manifest_json column to be dropped after backfill")
	}

	fileStates, err := store.ListAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedFileStates() error = %v", err)
	}
	if len(fileStates) != 1 || fileStates[0].GameRelativePath != "Data/added.esp" {
		t.Fatalf("ListAppliedFileStates() = %+v, want migrated added file", fileStates)
	}

	directories, err := store.ListAppliedCreatedDirectories(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListAppliedCreatedDirectories() error = %v", err)
	}
	if len(directories) != 1 || directories[0].GameRelativePath != "Mods/Created" {
		t.Fatalf("ListAppliedCreatedDirectories() = %+v, want migrated created directory", directories)
	}
}

func migrateUpTo(db *sql.DB, version int64) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect(gooseDialect); err != nil {
		return err
	}

	return goose.UpTo(db, migrationsDir, version)
}

func filepathToSlash(value string) string {
	return filepath.ToSlash(value)
}
