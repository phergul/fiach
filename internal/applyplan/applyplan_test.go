package applyplan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/operationplan"
)

func TestExecuteAppliesDirectoriesBeforeDependentFiles(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "plugin", 0o644)
	targetDirectory := filepath.Join(gameRoot, "Data")
	targetPath := filepath.Join(targetDirectory, "plugin.txt")
	plan := applicablePlan(
		directoryOperation(targetDirectory),
		copyOperation(sourcePath, targetPath),
	)

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 2 {
		t.Fatalf("Execute() result = %+v, want two completed operations", result)
	}
	assertFileContents(t, targetPath, "plugin")
}

func TestExecuteCopyWritesContentsAndPreservesMode(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/tool.sh", "run", 0o754)
	targetPath := filepath.Join(gameRoot, "tool.sh")
	plan := applicablePlan(copyOperation(sourcePath, targetPath))

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("Execute() result = %+v, want one completed operation", result)
	}
	assertFileContents(t, targetPath, "run")
	assertFileMode(t, targetPath, 0o754)
}

func TestExecuteCopyRecordsAddedFileManifestEntry(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "plugin", 0o644)
	targetPath := filepath.Join(gameRoot, "plugin.txt")
	operation := copyOperation(sourcePath, targetPath)
	operation.Mod = operationplan.ModContext{ModID: 12, ModName: "SkyUI"}

	result, err := Execute(applicablePlan(operation), Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || len(result.Manifest.AddedFiles) != 1 {
		t.Fatalf("Execute() manifest = %+v, want one added file", result.Manifest)
	}
	entry := result.Manifest.AddedFiles[0]
	if entry.OperationIndex != 0 || entry.Mod != operation.Mod || entry.SourcePath != filepath.Clean(sourcePath) || entry.TargetPath != filepath.Clean(targetPath) {
		t.Fatalf("added file manifest entry = %+v, want operation/mod/source/target metadata", entry)
	}
	if entry.SHA256 != sha256String("plugin") || entry.SizeBytes != int64(len("plugin")) {
		t.Fatalf("added file integrity = %+v, want SHA-256 and size for copied file", entry)
	}
}

func TestExecuteCopyFailsSafelyWhenTargetNowExists(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "plugin", 0o644)
	targetPath := writeApplyTestFile(t, gameRoot, "plugin.txt", "existing", 0o644)
	plan := applicablePlan(copyOperation(sourcePath, targetPath))

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.FailedCount != 1 {
		t.Fatalf("Execute() result = %+v, want failed copy result", result)
	}
	assertFileContents(t, targetPath, "existing")
}

func TestExecuteReplaceCreatesBackupBeforeOverwrite(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "modded", 0o640)
	targetPath := writeApplyTestFile(t, gameRoot, "Data/plugin.txt", "vanilla", 0o644)
	backupPath := filepath.Join(storageRoot, "operation-backups", "Data", "plugin.txt")
	plan := applicablePlan(replaceOperation(sourcePath, targetPath, backupPath))

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 1 {
		t.Fatalf("Execute() result = %+v, want one completed replace", result)
	}
	assertFileContents(t, targetPath, "modded")
	assertFileContents(t, backupPath, "vanilla")
	assertFileMode(t, targetPath, 0o640)
}

func TestExecuteReplaceRecordsReplacedFileManifestEntry(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "modded", 0o640)
	targetPath := writeApplyTestFile(t, gameRoot, "Data/plugin.txt", "vanilla", 0o644)
	backupPath := filepath.Join(storageRoot, "operation-backups", "Data", "plugin.txt")
	operation := replaceOperation(sourcePath, targetPath, backupPath)
	operation.Mod = operationplan.ModContext{ModID: 12, ModName: "SkyUI"}

	result, err := Execute(applicablePlan(operation), Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || len(result.Manifest.ReplacedFiles) != 1 {
		t.Fatalf("Execute() manifest = %+v, want one replaced file", result.Manifest)
	}
	entry := result.Manifest.ReplacedFiles[0]
	if entry.OperationIndex != 0 || entry.Mod != operation.Mod || entry.SourcePath != filepath.Clean(sourcePath) || entry.TargetPath != filepath.Clean(targetPath) || entry.BackupPath != filepath.Clean(backupPath) {
		t.Fatalf("replaced file manifest entry = %+v, want operation/mod/source/target/backup metadata", entry)
	}
	if entry.SHA256 != sha256String("modded") || entry.SizeBytes != int64(len("modded")) {
		t.Fatalf("replaced file target integrity = %+v, want SHA-256 and size for modded target", entry)
	}
	if entry.BackupSHA256 != sha256String("vanilla") || entry.BackupSizeBytes != int64(len("vanilla")) {
		t.Fatalf("replaced file backup integrity = %+v, want SHA-256 and size for vanilla backup", entry)
	}
}

func TestExecuteReplaceFailsWhenBackupExists(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "modded", 0o644)
	targetPath := writeApplyTestFile(t, gameRoot, "Data/plugin.txt", "vanilla", 0o644)
	backupPath := writeApplyTestFile(t, storageRoot, "operation-backups/Data/plugin.txt", "old backup", 0o644)
	plan := applicablePlan(replaceOperation(sourcePath, targetPath, backupPath))

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.FailedCount != 1 {
		t.Fatalf("Execute() result = %+v, want failed replace result", result)
	}
	assertFileContents(t, targetPath, "vanilla")
	assertFileContents(t, backupPath, "old backup")
}

func TestExecuteRecordsOnlyNewlyCreatedDirectoriesInManifest(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	existingDirectory := filepath.Join(gameRoot, "Existing")
	if err := os.MkdirAll(existingDirectory, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", existingDirectory, err)
	}
	newDirectory := filepath.Join(gameRoot, "New")
	existingOperation := directoryOperation(existingDirectory)
	existingOperation.Mod = operationplan.ModContext{ModID: 1, ModName: "Already There"}
	newOperation := directoryOperation(newDirectory)
	newOperation.Mod = operationplan.ModContext{ModID: 2, ModName: "Directory Maker"}

	result, err := Execute(applicablePlan(existingOperation, newOperation), Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || len(result.Manifest.CreatedDirectories) != 1 {
		t.Fatalf("Execute() manifest = %+v, want one created directory", result.Manifest)
	}
	entry := result.Manifest.CreatedDirectories[0]
	if entry.OperationIndex != 1 || entry.Mod != newOperation.Mod || entry.TargetPath != filepath.Clean(newDirectory) {
		t.Fatalf("created directory manifest entry = %+v, want only newly-created directory metadata", entry)
	}
}

func TestExecuteReportsFilesystemFailuresAsResults(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	missingSourcePath := filepath.Join(t.TempDir(), "missing.txt")
	copyTargetPath := filepath.Join(gameRoot, "missing-source.txt")
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "modded", 0o644)
	missingReplaceTargetPath := filepath.Join(gameRoot, "missing-target.txt")
	directoryTargetPath := filepath.Join(gameRoot, "target-is-directory")
	if err := os.MkdirAll(directoryTargetPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", directoryTargetPath, err)
	}

	tests := []struct {
		name          string
		operation     operationplan.Operation
		errorContains string
	}{
		{
			name:          "source missing",
			operation:     copyOperation(missingSourcePath, copyTargetPath),
			errorContains: "does not exist",
		},
		{
			name:          "replace target missing",
			operation:     replaceOperation(sourcePath, missingReplaceTargetPath, filepath.Join(storageRoot, "backup", "missing-target.txt")),
			errorContains: "target file",
		},
		{
			name:          "target is directory",
			operation:     replaceOperation(sourcePath, directoryTargetPath, filepath.Join(storageRoot, "backup", "directory-target")),
			errorContains: "not a regular file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(applicablePlan(tt.operation), Context{
				GameInstallPath:    gameRoot,
				GameModStoragePath: storageRoot,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.Success || result.FailedCount != 1 || len(result.Results) != 1 {
				t.Fatalf("Execute() result = %+v, want one failed result", result)
			}
			if result.Results[0].Error == nil || !strings.Contains(*result.Results[0].Error, tt.errorContains) {
				t.Fatalf("Execute() error result = %+v, want message containing %q", result.Results[0], tt.errorContains)
			}
		})
	}
}

func TestExecuteRejectsPathsOutsideRootsBeforeWriting(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	outsideRoot := t.TempDir()
	sourcePath := writeApplyTestFile(t, t.TempDir(), "mod/plugin.txt", "plugin", 0o644)

	tests := []struct {
		name      string
		operation operationplan.Operation
	}{
		{
			name:      "target outside game root",
			operation: copyOperation(sourcePath, filepath.Join(outsideRoot, "plugin.txt")),
		},
		{
			name: "backup outside storage root",
			operation: replaceOperation(
				sourcePath,
				writeApplyTestFile(t, gameRoot, "Data/plugin.txt", "vanilla", 0o644),
				filepath.Join(outsideRoot, "backup.txt"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Execute(applicablePlan(tt.operation), Context{
				GameInstallPath:    gameRoot,
				GameModStoragePath: storageRoot,
			})
			if err == nil {
				t.Fatal("Execute() error = nil, want path validation error")
			}
			if !strings.Contains(err.Error(), "outside") {
				t.Fatalf("Execute() error = %q, want outside root detail", err.Error())
			}
		})
	}
}

func TestExecuteStopsAtFirstFailureAndMarksRemainingSkipped(t *testing.T) {
	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	firstSourcePath := writeApplyTestFile(t, t.TempDir(), "mod/first.txt", "first", 0o644)
	firstTargetPath := filepath.Join(gameRoot, "first.txt")
	failingSourcePath := filepath.Join(t.TempDir(), "missing.txt")
	failingTargetPath := filepath.Join(gameRoot, "missing.txt")
	skippedSourcePath := writeApplyTestFile(t, t.TempDir(), "mod/skipped.txt", "skipped", 0o644)
	skippedTargetPath := filepath.Join(gameRoot, "skipped.txt")
	plan := applicablePlan(
		copyOperation(firstSourcePath, firstTargetPath),
		copyOperation(failingSourcePath, failingTargetPath),
		copyOperation(skippedSourcePath, skippedTargetPath),
	)

	result, err := Execute(plan, Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.CompletedCount != 1 || result.FailedCount != 1 || result.SkippedCount != 1 {
		t.Fatalf("Execute() result = %+v, want completed/failed/skipped counts", result)
	}
	if result.Results[0].Status != operationplan.ApplyOperationStatusCompleted ||
		result.Results[1].Status != operationplan.ApplyOperationStatusFailed ||
		result.Results[2].Status != operationplan.ApplyOperationStatusSkipped {
		t.Fatalf("Execute() statuses = %+v, want completed, failed, skipped", result.Results)
	}
	assertFileContents(t, firstTargetPath, "first")
	if _, err := os.Stat(skippedTargetPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want missing skipped target", skippedTargetPath, err)
	}
	if len(result.Manifest.AddedFiles) != 1 || result.Manifest.AddedFiles[0].TargetPath != filepath.Clean(firstTargetPath) {
		t.Fatalf("Execute() manifest = %+v, want only the completed first copy", result.Manifest)
	}
}

func TestExecuteEmptyApplicablePlanSucceedsAsNoop(t *testing.T) {
	result, err := Execute(operationplan.OperationPlan{CanApply: true}, Context{
		GameInstallPath:    t.TempDir(),
		GameModStoragePath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success || result.CompletedCount != 0 || result.FailedCount != 0 || result.SkippedCount != 0 || len(result.Results) != 0 {
		t.Fatalf("Execute() result = %+v, want no-op success", result)
	}
}

func TestExecuteTreatsManifestIntegrityFailureAsOperationFailure(t *testing.T) {
	originalComputeFileIntegrity := computeFileIntegrity
	defer func() {
		computeFileIntegrity = originalComputeFileIntegrity
	}()

	gameRoot := t.TempDir()
	storageRoot := t.TempDir()
	firstSourcePath := writeApplyTestFile(t, t.TempDir(), "mod/first.txt", "first", 0o644)
	firstTargetPath := filepath.Join(gameRoot, "first.txt")
	failingSourcePath := writeApplyTestFile(t, t.TempDir(), "mod/failing.txt", "failing", 0o644)
	failingTargetPath := filepath.Join(gameRoot, "failing.txt")
	skippedSourcePath := writeApplyTestFile(t, t.TempDir(), "mod/skipped.txt", "skipped", 0o644)
	skippedTargetPath := filepath.Join(gameRoot, "skipped.txt")
	forcedErr := errors.New("forced integrity failure")
	computeFileIntegrity = func(path string) (string, int64, error) {
		if filepath.Clean(path) == filepath.Clean(failingTargetPath) {
			return "", 0, forcedErr
		}
		return originalComputeFileIntegrity(path)
	}

	result, err := Execute(applicablePlan(
		copyOperation(firstSourcePath, firstTargetPath),
		copyOperation(failingSourcePath, failingTargetPath),
		copyOperation(skippedSourcePath, skippedTargetPath),
	), Context{
		GameInstallPath:    gameRoot,
		GameModStoragePath: storageRoot,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success || result.CompletedCount != 1 || result.FailedCount != 1 || result.SkippedCount != 1 {
		t.Fatalf("Execute() result = %+v, want manifest integrity failure to stop apply", result)
	}
	if len(result.Manifest.AddedFiles) != 1 || result.Manifest.AddedFiles[0].TargetPath != filepath.Clean(firstTargetPath) {
		t.Fatalf("Execute() manifest = %+v, want prior completed entries only", result.Manifest)
	}
	if result.Results[1].Error == nil || !strings.Contains(*result.Results[1].Error, forcedErr.Error()) {
		t.Fatalf("failed operation = %+v, want integrity failure detail", result.Results[1])
	}
	assertFileContents(t, failingTargetPath, "failing")
	if _, err := os.Stat(skippedTargetPath); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) error = %v, want missing skipped target", skippedTargetPath, err)
	}
}

func applicablePlan(operations ...operationplan.Operation) operationplan.OperationPlan {
	return operationplan.OperationPlan{
		Operations: operations,
		CanApply:   true,
	}
}

func directoryOperation(targetPath string) operationplan.Operation {
	return operationplan.Operation{
		Type:       operationplan.OperationTypeCreateDirectory,
		TargetPath: targetPath,
	}
}

func copyOperation(sourcePath string, targetPath string) operationplan.Operation {
	return operationplan.Operation{
		Type:       operationplan.OperationTypeCopy,
		SourcePath: &sourcePath,
		TargetPath: targetPath,
	}
}

func replaceOperation(sourcePath string, targetPath string, backupPath string) operationplan.Operation {
	return operationplan.Operation{
		Type:       operationplan.OperationTypeReplace,
		SourcePath: &sourcePath,
		TargetPath: targetPath,
		BackupPath: &backupPath,
	}
}

func writeApplyTestFile(t *testing.T, root string, relativePath string, contents string, mode os.FileMode) string {
	t.Helper()

	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("os.Chmod(%q) error = %v", path, err)
	}

	return path
}

func assertFileContents(t *testing.T, path string, want string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(contents) != want {
		t.Fatalf("os.ReadFile(%q) = %q, want %q", path, contents, want)
	}
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	if runtime.GOOS == "windows" {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode(%q) = %v, want %v", path, got, want)
	}
}

func sha256String(contents string) string {
	sum := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(sum[:])
}
