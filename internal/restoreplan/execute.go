package restoreplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/fileops"
)

func Execute(manifest appliedstate.ManifestDocument, context Context) (result RestoreResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute restore plan: %w", err)
		}
	}()

	resolved, err := validateContext(context)
	if err != nil {
		return RestoreResult{}, err
	}

	operations := buildOperations(manifest)
	if failures := preflightOperations(operations, manifest, resolved); len(failures) > 0 {
		return failedPreflightResult(operations, failures), nil
	}

	result.Success = true
	result.Results = make([]RestoreOperationResult, 0, len(operations))
	for index, operation := range operations {
		message, operationErr := executeOperation(operation, resolved)
		if operationErr != nil {
			result.Success = false
			result.Results = append(result.Results, newFailedResult(index, operation, operationErr))
			appendSkippedResults(operations, index+1, &result)
			updateCounts(&result)
			return result, nil
		}

		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operation,
			Status:         RestoreOperationStatusCompleted,
			Message:        message,
		})
	}

	updateCounts(&result)
	return result, nil
}

func validateContext(context Context) (resolvedContext, error) {
	gameInstallPath, err := fileops.CleanRequiredAbsPath("game install path", context.GameInstallPath)
	if err != nil {
		return resolvedContext{}, err
	}
	gameModStoragePath, err := fileops.CleanRequiredAbsPath("game mod storage path", context.GameModStoragePath)
	if err != nil {
		return resolvedContext{}, err
	}

	return resolvedContext{
		gameInstallPath:    gameInstallPath,
		gameModStoragePath: gameModStoragePath,
	}, nil
}

func executeOperation(operation RestoreOperation, context resolvedContext) (string, error) {
	switch operation.Type {
	case RestoreOperationTypeRemoveAddedFile:
		return removeAddedFile(operation.TargetPath)
	case RestoreOperationTypeRestoreReplacedFile:
		return restoreReplacedFile(operation.TargetPath, *operation.BackupPath)
	case RestoreOperationTypeRemoveCreatedDir:
		return removeCreatedDirectory(operation.TargetPath)
	case RestoreOperationTypeDeleteRestoredBackup:
		return deleteRestoredBackup(*operation.BackupPath, context.gameModStoragePath)
	default:
		return "", fmt.Errorf("unsupported restore operation type %q", operation.Type)
	}
}

func removeAddedFile(targetPath string) (string, error) {
	if err := os.Remove(targetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "Added file already absent.", nil
		}
		return "", fmt.Errorf("remove added file %q: %w", targetPath, err)
	}

	return "Removed added file.", nil
}

func restoreReplacedFile(targetPath string, backupPath string) (string, error) {
	backupInfo, err := fileops.StatRegularFile("backup file", backupPath)
	if err != nil {
		return "", err
	}
	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: backupPath,
		TargetPath: targetPath,
		Mode:       backupInfo.Mode().Perm(),
		Replace:    true,
		TempPrefix: ".fiach-restore-*",
		OpenLabel:  "backup file",
	}); err != nil {
		return "", err
	}

	return "Restored replaced file from backup.", nil
}

func removeCreatedDirectory(targetPath string) (string, error) {
	err := os.Remove(targetPath)
	if err == nil {
		return "Removed empty created directory.", nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "Created directory already absent.", nil
	}
	if fileops.IsDirectoryNotEmptyError(err) {
		return "Created directory was not empty and was left in place.", nil
	}

	return "", fmt.Errorf("remove created directory %q: %w", targetPath, err)
}

func deleteRestoredBackup(backupPath string, gameModStoragePath string) (string, error) {
	if err := os.Remove(backupPath); err != nil {
		return "", fmt.Errorf("delete restored backup %q: %w", backupPath, err)
	}
	if err := fileops.RemoveEmptyParentDirectories(filepath.Dir(backupPath), gameModStoragePath); err != nil {
		return "", err
	}

	return "Deleted restored backup.", nil
}
