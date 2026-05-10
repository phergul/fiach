package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestMigrateUpAppliesInitialMigration(t *testing.T) {
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

	if version != 1 {
		t.Fatalf("goose version = %d, want 1", version)
	}
}

func TestMigrateDownRollsBackInitialMigration(t *testing.T) {
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

	if version != 1 {
		t.Fatalf("goose version = %d, want 1", version)
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
