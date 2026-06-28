package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/storage"
)

func insertServiceProfileTestProfile(t *testing.T, store *storage.Store, gameID int64, name string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		t.Fatalf("insert service profile test profile: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service profile test profile LastInsertId(): %v", err)
	}

	return id
}

func addServiceProfileMod(t *testing.T, store *storage.Store, profileID int64, modID int64, enabled bool, loadOrder int64) {
	t.Helper()

	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	if _, err := store.DB().Exec(`
		INSERT INTO profile_mods (profile_id, mod_id, enabled, load_order)
		VALUES (?, ?, ?, ?)
	`, profileID, modID, enabledValue, loadOrder); err != nil {
		t.Fatalf("insert service profile test profile mod: %v", err)
	}
}

func addServiceInstallConfig(t *testing.T, store *storage.Store, modID int64, strategyType string, targetBase string, targetRelativePath string, sourceSubpath *string) {
	t.Helper()

	if _, err := store.DB().Exec(`
		INSERT INTO mod_install_configs (mod_id, strategy_type, target_base, target_relative_path, source_subpath)
		VALUES (?, ?, ?, ?, ?)
	`, modID, strategyType, targetBase, targetRelativePath, sourceSubpath); err != nil {
		t.Fatalf("insert service profile test config: %v", err)
	}
}

func TestProfileServiceLoadAppliedFileStatesReturnsPersistedRows(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")

	saveTestAppliedProfileState(t, store, gameID, profileID, []testAppliedFileSeed{
		{
			GameRelativePath: "Data/modded.txt",
			AppliedSHA256:    testContentHash("modded"),
			AppliedSizeBytes: int64(len("modded")),
		},
	}, nil)

	service := NewProfileService(store, testLogger())
	states, err := service.LoadAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("LoadAppliedFileStates() error = %v", err)
	}
	if len(states) != 1 || states[0].GameRelativePath != "Data/modded.txt" {
		t.Fatalf("LoadAppliedFileStates() = %+v, want persisted file state", states)
	}
}

func assertServiceFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("os.ReadFile(%q) = %q, want %q", path, contents, want)
	}
}

func makeProfileDeploymentSourceTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for relativePath, contents := range files {
		path := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", path, err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}

	return root
}
