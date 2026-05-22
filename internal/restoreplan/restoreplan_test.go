package restoreplan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/appliedstate"
)

func TestExecuteRestoresFilesRemovesSafeDirectoriesAndDeletesBackups(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	addedPath := writeRestoreTestFile(t, gameRoot, "Data/added.txt", "added")
	targetPath := writeRestoreTestFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := writeRestoreTestFile(t, modRoot, "operation-backups/Data/replaced.txt", "vanilla")
	createdDirectory := filepath.Join(gameRoot, "Mods", "Created")
	if err := os.MkdirAll(createdDirectory, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		AddedFiles: []appliedstate.AddedFile{
			restoreTestAddedFile(t, 0, addedPath, "Mod", "added"),
		},
		ReplacedFiles: []appliedstate.ReplacedFile{
			restoreTestReplacedFile(t, 1, targetPath, "modded", backupPath, "vanilla"),
		},
		CreatedDirectories: []appliedstate.CreatedDirectory{
			{
				OperationIndex: 2,
				Mod:            appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:     createdDirectory,
			},
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 4 || result.FailedCount != 0 || result.SkippedCount != 0 {
		t.Fatalf("Execute() = %+v, want four completed restore operations", result)
	}
	assertRestorePathMissing(t, addedPath)
	assertRestoreFileContents(t, targetPath, "vanilla")
	assertRestorePathMissing(t, backupPath)
	assertRestorePathMissing(t, createdDirectory)
}

func TestExecutePreflightReportsMissingBackupWithoutChangingFiles(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	addedPath := writeRestoreTestFile(t, gameRoot, "Data/added.txt", "added")
	targetPath := writeRestoreTestFile(t, gameRoot, "Data/replaced.txt", "modded")
	backupPath := filepath.Join(modRoot, "operation-backups", "Data", "replaced.txt")
	backupSHA, backupSize := restoreTestIntegrityForContent(t, "vanilla")

	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		AddedFiles: []appliedstate.AddedFile{
			restoreTestAddedFile(t, 0, addedPath, "Mod", "added"),
		},
		ReplacedFiles: []appliedstate.ReplacedFile{
			{
				OperationIndex:  1,
				Mod:             appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:      targetPath,
				SHA256:          restoreTestSHA(t, targetPath),
				SizeBytes:       int64(len("modded")),
				BackupPath:      backupPath,
				BackupSHA256:    backupSHA,
				BackupSizeBytes: backupSize,
			},
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.FailedCount == 0 {
		t.Fatalf("Execute() = %+v, want preflight failure", result)
	}
	if !restoreResultContainsError(result, "backup file") || !restoreResultContainsError(result, "missing") {
		t.Fatalf("Execute() = %+v, want missing backup detail", result)
	}
	assertRestoreFileContents(t, addedPath, "added")
	assertRestoreFileContents(t, targetPath, "modded")
}

func TestExecutePreflightRejectsTargetHashMismatchWithoutChangingFiles(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	targetPath := writeRestoreTestFile(t, gameRoot, "Data/replaced.txt", "user-changed")
	backupPath := writeRestoreTestFile(t, modRoot, "operation-backups/Data/replaced.txt", "vanilla")
	moddedSHA, moddedSize := restoreTestIntegrityForContent(t, "modded")
	backupSHA, backupSize := restoreTestIntegrityForContent(t, "vanilla")

	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		ReplacedFiles: []appliedstate.ReplacedFile{
			{
				OperationIndex:  0,
				Mod:             appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:      targetPath,
				SHA256:          moddedSHA,
				SizeBytes:       moddedSize,
				BackupPath:      backupPath,
				BackupSHA256:    backupSHA,
				BackupSizeBytes: backupSize,
			},
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || !restoreResultContainsError(result, "does not match the applied file or recorded backup integrity") {
		t.Fatalf("Execute() = %+v, want target integrity failure", result)
	}
	assertRestoreFileContents(t, targetPath, "user-changed")
	assertRestoreFileContents(t, backupPath, "vanilla")
}

func TestExecuteLeavesNonEmptyCreatedDirectoriesInPlace(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	createdDirectory := filepath.Join(gameRoot, "Data", "Created")
	userFile := writeRestoreTestFile(t, createdDirectory, "user.txt", "user")
	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		CreatedDirectories: []appliedstate.CreatedDirectory{
			{
				OperationIndex: 0,
				Mod:            appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:     createdDirectory,
			},
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("Execute() = %+v, want successful safe directory skip", result)
	}
	if !strings.Contains(result.Results[0].Message, "not empty") {
		t.Fatalf("Execute() message = %q, want non-empty directory detail", result.Results[0].Message)
	}
	assertRestoreFileContents(t, userFile, "user")
}

func TestExecuteRejectsTargetsOutsideGameRootWithoutChangingFiles(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	outsidePath := writeRestoreTestFile(t, t.TempDir(), "outside.txt", "added")
	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		AddedFiles: []appliedstate.AddedFile{
			restoreTestAddedFile(t, 0, outsidePath, "Mod", "added"),
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || !restoreResultContainsError(result, "outside") {
		t.Fatalf("Execute() = %+v, want outside-root failure", result)
	}
	assertRestoreFileContents(t, outsidePath, "added")
}

func TestExecuteSupportsRetryWhenTargetAlreadyRestoredAndBackupNeedsCleanup(t *testing.T) {
	t.Parallel()

	gameRoot := t.TempDir()
	modRoot := t.TempDir()
	targetPath := writeRestoreTestFile(t, gameRoot, "Data/replaced.txt", "vanilla")
	backupPath := writeRestoreTestFile(t, modRoot, "operation-backups/Data/replaced.txt", "vanilla")
	moddedSHA, moddedSize := restoreTestIntegrityForContent(t, "modded")
	backupSHA, backupSize := restoreTestIntegrityForContent(t, "vanilla")
	manifest := appliedstate.ManifestDocument{
		Version: appliedstate.DocumentVersion,
		ReplacedFiles: []appliedstate.ReplacedFile{
			{
				OperationIndex:  0,
				Mod:             appliedstate.Mod{ID: 1, Name: "Mod"},
				TargetPath:      targetPath,
				SHA256:          moddedSHA,
				SizeBytes:       moddedSize,
				BackupPath:      backupPath,
				BackupSHA256:    backupSHA,
				BackupSizeBytes: backupSize,
			},
		},
	}

	result, err := Execute(manifest, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: modRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 2 {
		t.Fatalf("Execute() = %+v, want restore retry and backup cleanup success", result)
	}
	assertRestoreFileContents(t, targetPath, "vanilla")
	assertRestorePathMissing(t, backupPath)
}

func restoreTestAddedFile(t *testing.T, operationIndex int, targetPath string, modName string, content string) appliedstate.AddedFile {
	t.Helper()

	sha, size := restoreTestIntegrityForContent(t, content)
	return appliedstate.AddedFile{
		OperationIndex: operationIndex,
		Mod:            appliedstate.Mod{ID: 1, Name: modName},
		TargetPath:     targetPath,
		SHA256:         sha,
		SizeBytes:      size,
	}
}

func restoreTestReplacedFile(t *testing.T, operationIndex int, targetPath string, targetContent string, backupPath string, backupContent string) appliedstate.ReplacedFile {
	t.Helper()

	targetSHA, targetSize := restoreTestIntegrityForContent(t, targetContent)
	backupSHA, backupSize := restoreTestIntegrityForContent(t, backupContent)
	return appliedstate.ReplacedFile{
		OperationIndex:  operationIndex,
		Mod:             appliedstate.Mod{ID: 1, Name: "Mod"},
		TargetPath:      targetPath,
		SHA256:          targetSHA,
		SizeBytes:       targetSize,
		BackupPath:      backupPath,
		BackupSHA256:    backupSHA,
		BackupSizeBytes: backupSize,
	}
}

func writeRestoreTestFile(t *testing.T, root string, relativePath string, content string) string {
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

func restoreTestIntegrityForContent(t *testing.T, content string) (string, int64) {
	t.Helper()

	path := writeRestoreTestFile(t, t.TempDir(), "file.txt", content)
	hash, size, err := fileIntegrity(path)
	if err != nil {
		t.Fatalf("fileIntegrity() error = %v", err)
	}

	return hash, size
}

func restoreTestSHA(t *testing.T, path string) string {
	t.Helper()

	hash, _, err := fileIntegrity(path)
	if err != nil {
		t.Fatalf("fileIntegrity() error = %v", err)
	}

	return hash
}

func restoreResultContainsError(result RestoreResult, substring string) bool {
	for _, operationResult := range result.Results {
		if operationResult.Error != nil && strings.Contains(*operationResult.Error, substring) {
			return true
		}
	}

	return false
}

func assertRestoreFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", path, contents, want)
	}
}

func assertRestorePathMissing(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("%q exists, want missing", path)
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want not exist", path, err)
	}
}
