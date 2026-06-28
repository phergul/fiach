package storage

import "testing"

func TestMigrateUpAddsAppliedCreatedDirectoriesTable(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	defer closeStore(t, store)

	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if !tableExists(t, store, "applied_created_directories") {
		t.Fatal("expected applied_created_directories table to exist")
	}
	for _, column := range []string{"game_id", "game_relative_path", "mod_id", "mod_name"} {
		if !columnExists(t, store, "applied_created_directories", column) {
			t.Fatalf("expected applied_created_directories.%s column to exist", column)
		}
	}
}
