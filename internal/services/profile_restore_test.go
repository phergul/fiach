package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestProfileServiceRestoreVanillaStateRestoresFilesClearsStateAndDeletesBackups(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	gameModStoragePath := serviceRestoreGameModStoragePath(t, store, gameID)
	addedPath := writeServiceRestoreFile(t, gameRoot, "Data/added.txt", "added")
	targetPath := writeServiceRestoreFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := writeServiceRestoreFile(t, gameModStoragePath, "operation-backups/Data/replaced.txt", "vanilla")
	createdDirectory := filepath.Join(gameRoot, "Mods", "Created")
	if err := os.MkdirAll(createdDirectory, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	saveServiceRestoreAppliedState(t, store, gameID, profileID, appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		AddedFiles: []appliedstate.AddedFile{
			serviceRestoreAddedFile(0, addedPath, "added"),
		},
		ReplacedFiles: []appliedstate.ReplacedFile{
			serviceRestoreReplacedFile(1, targetPath, "modded", backupPath, "vanilla"),
		},
		CreatedDirectories: []appliedstate.CreatedDirectory{
			{
				OperationIndex: 2,
				Mod:            appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:     createdDirectory,
			},
		},
	})

	result, err := NewProfileService(store, testLogger()).RestoreVanillaState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("RestoreVanillaState() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 4 {
		t.Fatalf("RestoreVanillaState() = %+v, want successful restore", result)
	}
	assertServicePathMissing(t, addedPath)
	assertServiceFileContents(t, targetPath, "vanilla")
	assertServicePathMissing(t, backupPath)
	assertServicePathMissing(t, createdDirectory)

	if _, found, err := store.GetAppliedProfileState(context.Background(), gameID); err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	} else if found {
		t.Fatal("GetAppliedProfileState() found = true, want cleared state")
	}
	hasFileStates, err := store.HasAppliedFileStates(context.Background(), gameID)
	if err != nil {
		t.Fatalf("HasAppliedFileStates() error = %v", err)
	}
	if hasFileStates {
		t.Fatal("HasAppliedFileStates() found = true, want cleared file states")
	}
}

func TestProfileServiceRestoreVanillaStateReturnsClearErrorWithoutAppliedState(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", t.TempDir())
	_, err := NewProfileService(store, testLogger()).RestoreVanillaState(context.Background(), gameID)
	if err == nil {
		t.Fatal("RestoreVanillaState() error = nil, want no applied state error")
	}
	if err.Error() != "No profile is currently applied for this game." {
		t.Fatalf("RestoreVanillaState() error = %q, want clear no-state detail", err.Error())
	}
}

func TestProfileServiceRestoreVanillaStatePreservesStateAndFilesOnPreflightFailure(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	gameModStoragePath := serviceRestoreGameModStoragePath(t, store, gameID)
	targetPath := writeServiceRestoreFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := filepath.Join(gameModStoragePath, "operation-backups", "Data", "replaced.txt")
	saveServiceRestoreAppliedState(t, store, gameID, profileID, appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		ReplacedFiles: []appliedstate.ReplacedFile{
			serviceRestoreReplacedFile(0, targetPath, "modded", backupPath, "vanilla"),
		},
	})

	result, err := NewProfileService(store, testLogger()).RestoreVanillaState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("RestoreVanillaState() error = %v, want result failure", err)
	}
	if result.Success || !serviceRestoreResultContainsError(result, "backup file") || !serviceRestoreResultContainsError(result, "missing") {
		t.Fatalf("RestoreVanillaState() = %+v, want missing backup failure", result)
	}
	assertServiceFileContents(t, targetPath, "modded")
	_, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if !found {
		t.Fatal("GetAppliedProfileState() found = false, want preserved state")
	}
}

func TestProfileServiceRestoreVanillaStatePreservesStateWhenBackupCleanupFails(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameRoot := t.TempDir()
	gameID := insertServiceProfileTestGame(t, store, "Skyrim", gameRoot)
	profileID := insertServiceProfileTestProfile(t, store, gameID, "Default")
	gameModStoragePath := serviceRestoreGameModStoragePath(t, store, gameID)
	targetPath := writeServiceRestoreFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := writeServiceRestoreFile(t, gameModStoragePath, "operation-backups/Data/replaced.txt", "vanilla")
	backupDirectory := filepath.Dir(backupPath)
	if err := os.Chmod(backupDirectory, 0o500); err != nil {
		t.Fatalf("chmod backup directory: %v", err)
	}
	defer func() {
		_ = os.Chmod(backupDirectory, 0o755)
	}()
	saveServiceRestoreAppliedState(t, store, gameID, profileID, appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		ReplacedFiles: []appliedstate.ReplacedFile{
			serviceRestoreReplacedFile(0, targetPath, "modded", backupPath, "vanilla"),
		},
	})

	result, err := NewProfileService(store, testLogger()).RestoreVanillaState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("RestoreVanillaState() error = %v, want result failure", err)
	}
	hasCleanupFailure := serviceRestoreResultContainsError(result, "delete restored backup") ||
		serviceRestoreResultContainsError(result, "remove empty backup directory")
	if result.Success || !hasCleanupFailure {
		t.Fatalf("RestoreVanillaState() = %+v, want cleanup failure", result)
	}
	assertServiceFileContents(t, targetPath, "vanilla")
	_, found, err := store.GetAppliedProfileState(context.Background(), gameID)
	if err != nil {
		t.Fatalf("GetAppliedProfileState() error = %v", err)
	}
	if !found {
		t.Fatal("GetAppliedProfileState() found = false, want preserved state")
	}
}

func saveServiceRestoreAppliedState(t *testing.T, store *storage.Store, gameID int64, profileID int64, manifest appliedstate.ManifestDocument) {
	t.Helper()

	manifestJSON, err := appliedstate.EncodeManifest(manifest)
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
}

func serviceRestoreGameModStoragePath(t *testing.T, store *storage.Store, gameID int64) string {
	t.Helper()

	path, err := store.ResolveGameModStoragePath(context.Background(), gameID, "")
	if err != nil {
		t.Fatalf("ResolveGameModStoragePath() error = %v", err)
	}

	return path
}

func serviceRestoreAddedFile(operationIndex int, targetPath string, content string) appliedstate.AddedFile {
	return appliedstate.AddedFile{
		OperationIndex: operationIndex,
		Mod:            appliedstate.Mod{ID: 1, Name: "Mod"},
		TargetPath:     targetPath,
		SHA256:         sha256Hex(content),
		SizeBytes:      int64(len(content)),
	}
}

func serviceRestoreReplacedFile(operationIndex int, targetPath string, targetContent string, backupPath string, backupContent string) appliedstate.ReplacedFile {
	return appliedstate.ReplacedFile{
		OperationIndex:  operationIndex,
		Mod:             appliedstate.Mod{ID: 1, Name: "Mod"},
		TargetPath:      targetPath,
		SHA256:          sha256Hex(targetContent),
		SizeBytes:       int64(len(targetContent)),
		BackupPath:      backupPath,
		BackupSHA256:    sha256Hex(backupContent),
		BackupSizeBytes: int64(len(backupContent)),
	}
}

func writeServiceRestoreFile(t *testing.T, root string, relativePath string, content string) string {
	t.Helper()

	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	return path
}

func serviceRestoreResultContainsError(result dto.RestoreResult, substring string) bool {
	for _, operationResult := range result.Results {
		if operationResult.Error != nil && strings.Contains(*operationResult.Error, substring) {
			return true
		}
	}

	return false
}

func assertServicePathMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%q exists, want missing", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", path, err)
	}
}
