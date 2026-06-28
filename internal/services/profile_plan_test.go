package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func insertServiceProfileTestProfile(t *testing.T, store *storage.Store, gameID int64, name string) int64 {
	t.Helper()

	result, err := store.DB().Exec(`
		INSERT INTO profiles (game_id, name)
		VALUES (?, ?)
	`, gameID, name)
	if err != nil {
		t.Fatalf("insert service profile plan test profile: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("service profile plan test profile LastInsertId(): %v", err)
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
		t.Fatalf("insert service profile plan test profile mod: %v", err)
	}
}

func addServiceInstallConfig(t *testing.T, store *storage.Store, modID int64, strategyType string, targetBase string, targetRelativePath string, sourceSubpath *string) {
	t.Helper()

	if _, err := store.DB().Exec(`
		INSERT INTO mod_install_configs (mod_id, strategy_type, target_base, target_relative_path, source_subpath)
		VALUES (?, ?, ?, ?, ?)
	`, modID, strategyType, targetBase, targetRelativePath, sourceSubpath); err != nil {
		t.Fatalf("insert service profile plan test config: %v", err)
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestProfileServiceLoadAppliedFileStatesLazilyMigratesV1Manifest(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	targetPath := filepath.Join(gameRoot, "Data", "modded.txt")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("modded"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	manifestJSON, err := appliedstate.EncodeManifest(appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersionV1,
		AddedFiles: []appliedstate.AddedFile{
			{
				OperationIndex: 0,
				Mod:            appliedstate.Mod{ID: 10, Name: "SkyUI"},
				TargetPath:     targetPath,
				SHA256:         "added-sha",
				SizeBytes:      6,
			},
		},
	})
	if err != nil {
		t.Fatalf("EncodeManifest() error = %v", err)
	}
	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        manifestJSON,
		ProfileSnapshotJSON: `{"version":1}`,
		ProfileSnapshotHash: "hash",
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}

	service := NewProfileService(store, testLogger())
	states, err := service.LoadAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("LoadAppliedFileStates() error = %v", err)
	}
	if len(states) != 1 || states[0].GameRelativePath != "Data/modded.txt" || states[0].AppliedSHA256 == nil || *states[0].AppliedSHA256 != "added-sha" {
		t.Fatalf("LoadAppliedFileStates() = %+v, want migrated added file state", states)
	}

	found, err := store.HasAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if !found {
		t.Fatal("HasAppliedFileStates() found = false, want lazy migration persisted rows")
	}

	statesAgain, err := service.LoadAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("LoadAppliedFileStates() second error = %v", err)
	}
	if len(statesAgain) != 1 {
		t.Fatalf("LoadAppliedFileStates() second = %+v, want idempotent load", statesAgain)
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

func makeProfilePlanSourceTree(t *testing.T, files map[string]string) string {
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
