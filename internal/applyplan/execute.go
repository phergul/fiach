package applyplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/mod-manager/internal/fileops"
	"github.com/phergul/mod-manager/internal/operationplan"
)

func Execute(plan operationplan.OperationPlan, context Context) (result operationplan.ApplyOperationPlanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute operation plan: %w", err)
		}
	}()

	if _, err := validatePlan(plan, context); err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}

	result.Success = true
	result.Results = make([]operationplan.ApplyOperationResult, 0, len(plan.Operations))
	for index, operation := range plan.Operations {
		outcome, operationErr := executeOperation(operation)
		if operationErr == nil {
			operationErr = appendManifestEntry(index, operation, outcome, &result.Manifest)
		}
		if operationErr != nil {
			result.Success = false
			result.Results = append(result.Results, newFailedResult(index, operation, operationErr))
			appendSkippedResults(plan.Operations, index+1, &result)
			updateCounts(&result)
			return result, nil
		}

		result.Results = append(result.Results, operationplan.ApplyOperationResult{
			OperationIndex: index,
			Operation:      operation,
			Status:         operationplan.ApplyOperationStatusCompleted,
			Message:        outcome.message,
		})
	}

	updateCounts(&result)
	return result, nil
}

func validatePlan(plan operationplan.OperationPlan, context Context) (resolvedContext, error) {
	if !plan.CanApply {
		return resolvedContext{}, errors.New("operation plan has blocking issues")
	}

	gameInstallPath, err := fileops.CleanRequiredAbsPath("game install path", context.GameInstallPath)
	if err != nil {
		return resolvedContext{}, err
	}
	gameModStoragePath, err := fileops.CleanRequiredAbsPath("game mod storage path", context.GameModStoragePath)
	if err != nil {
		return resolvedContext{}, err
	}

	resolved := resolvedContext{
		gameInstallPath:    gameInstallPath,
		gameModStoragePath: gameModStoragePath,
	}
	for index, operation := range plan.Operations {
		if strings.TrimSpace(operation.TargetPath) == "" {
			return resolvedContext{}, fmt.Errorf("operation %d target path is required", index)
		}
		if err := fileops.RequirePathWithinRoot("operation target path", operation.TargetPath, gameInstallPath); err != nil {
			return resolvedContext{}, fmt.Errorf("operation %d: %w", index, err)
		}

		switch operation.Type {
		case operationplan.OperationTypeCreateDirectory:
		case operationplan.OperationTypeCopy:
			if operation.SourcePath == nil || strings.TrimSpace(*operation.SourcePath) == "" {
				return resolvedContext{}, fmt.Errorf("operation %d source path is required", index)
			}
		case operationplan.OperationTypeReplace:
			if operation.SourcePath == nil || strings.TrimSpace(*operation.SourcePath) == "" {
				return resolvedContext{}, fmt.Errorf("operation %d source path is required", index)
			}
			if operation.BackupPath == nil || strings.TrimSpace(*operation.BackupPath) == "" {
				return resolvedContext{}, fmt.Errorf("operation %d backup path is required", index)
			}
			if err := fileops.RequirePathWithinRoot("operation backup path", *operation.BackupPath, gameModStoragePath); err != nil {
				return resolvedContext{}, fmt.Errorf("operation %d: %w", index, err)
			}
		default:
			return resolvedContext{}, fmt.Errorf("operation %d has unsupported type %q", index, operation.Type)
		}
	}

	return resolved, nil
}

func executeOperation(operation operationplan.Operation) (operationOutcome, error) {
	switch operation.Type {
	case operationplan.OperationTypeCreateDirectory:
		return executeCreateDirectory(operation)
	case operationplan.OperationTypeCopy:
		return executeCopy(operation)
	case operationplan.OperationTypeReplace:
		return executeReplace(operation)
	default:
		return operationOutcome{}, fmt.Errorf("unsupported operation type %q", operation.Type)
	}
}

func executeCreateDirectory(operation operationplan.Operation) (operationOutcome, error) {
	info, err := os.Stat(operation.TargetPath)
	if err == nil {
		if !info.IsDir() {
			return operationOutcome{}, fmt.Errorf("target path %q already exists and is not a directory", operation.TargetPath)
		}
		return operationOutcome{message: "Directory already exists."}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return operationOutcome{}, fmt.Errorf("stat target directory %q: %w", operation.TargetPath, err)
	}

	if err := os.MkdirAll(operation.TargetPath, 0o755); err != nil {
		return operationOutcome{}, fmt.Errorf("create target directory %q: %w", operation.TargetPath, err)
	}
	return operationOutcome{message: "Created directory.", createdDirectory: true}, nil
}

func executeCopy(operation operationplan.Operation) (operationOutcome, error) {
	sourcePath := *operation.SourcePath
	sourceInfo, err := fileops.StatRegularFile("source file", sourcePath)
	if err != nil {
		return operationOutcome{}, err
	}

	if _, err := os.Lstat(operation.TargetPath); err == nil {
		return operationOutcome{}, fmt.Errorf("target file %q already exists", operation.TargetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return operationOutcome{}, fmt.Errorf("stat target file %q: %w", operation.TargetPath, err)
	}

	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: operation.TargetPath,
		Mode:       sourceInfo.Mode().Perm(),
		OpenLabel:  "source file",
	}); err != nil {
		return operationOutcome{}, err
	}
	return operationOutcome{message: "Copied file."}, nil
}

func executeReplace(operation operationplan.Operation) (operationOutcome, error) {
	sourcePath := *operation.SourcePath
	sourceInfo, err := fileops.StatRegularFile("source file", sourcePath)
	if err != nil {
		return operationOutcome{}, err
	}
	targetInfo, err := fileops.StatRegularFile("target file", operation.TargetPath)
	if err != nil {
		return operationOutcome{}, err
	}

	backupPath := *operation.BackupPath
	if _, err := os.Lstat(backupPath); err == nil {
		return operationOutcome{}, fmt.Errorf("backup file %q already exists", backupPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return operationOutcome{}, fmt.Errorf("stat backup file %q: %w", backupPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return operationOutcome{}, fmt.Errorf("create backup directory %q: %w", filepath.Dir(backupPath), err)
	}
	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: operation.TargetPath,
		TargetPath: backupPath,
		Mode:       targetInfo.Mode().Perm(),
		OpenLabel:  "source file",
	}); err != nil {
		return operationOutcome{}, fmt.Errorf("create backup file %q: %w", backupPath, err)
	}
	if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: operation.TargetPath,
		Mode:       sourceInfo.Mode().Perm(),
		Replace:    true,
		OpenLabel:  "source file",
	}); err != nil {
		return operationOutcome{}, err
	}
	return operationOutcome{message: "Replaced file and created backup."}, nil
}
