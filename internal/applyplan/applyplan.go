package applyplan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/mod-manager/internal/operationplan"
)

type Context struct {
	GameInstallPath    string
	GameModStoragePath string
}

type resolvedContext struct {
	gameInstallPath    string
	gameModStoragePath string
}

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
		message, operationErr := executeOperation(operation)
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
			Message:        message,
		})
	}

	updateCounts(&result)
	return result, nil
}

func validatePlan(plan operationplan.OperationPlan, context Context) (resolvedContext, error) {
	if !plan.CanApply {
		return resolvedContext{}, errors.New("operation plan has blocking issues")
	}

	gameInstallPath, err := cleanRootPath("game install path", context.GameInstallPath)
	if err != nil {
		return resolvedContext{}, err
	}
	gameModStoragePath, err := cleanRootPath("game mod storage path", context.GameModStoragePath)
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
		if err := requirePathWithinRoot("operation target path", operation.TargetPath, gameInstallPath); err != nil {
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
			if err := requirePathWithinRoot("operation backup path", *operation.BackupPath, gameModStoragePath); err != nil {
				return resolvedContext{}, fmt.Errorf("operation %d: %w", index, err)
			}
		default:
			return resolvedContext{}, fmt.Errorf("operation %d has unsupported type %q", index, operation.Type)
		}
	}

	return resolved, nil
}

func cleanRootPath(name string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", name, path, err)
	}

	return filepath.Clean(absolutePath), nil
}

func requirePathWithinRoot(name string, path string, root string) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s %q: %w", name, path, err)
	}
	cleanPath := filepath.Clean(absolutePath)

	relativePath, err := filepath.Rel(root, cleanPath)
	if err != nil {
		return fmt.Errorf("compare %s %q with root %q: %w", name, cleanPath, root, err)
	}
	if relativePath == "." {
		return nil
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q is outside %q", name, cleanPath, root)
	}

	return nil
}

func executeOperation(operation operationplan.Operation) (string, error) {
	switch operation.Type {
	case operationplan.OperationTypeCreateDirectory:
		return executeCreateDirectory(operation)
	case operationplan.OperationTypeCopy:
		return executeCopy(operation)
	case operationplan.OperationTypeReplace:
		return executeReplace(operation)
	default:
		return "", fmt.Errorf("unsupported operation type %q", operation.Type)
	}
}

func executeCreateDirectory(operation operationplan.Operation) (string, error) {
	info, err := os.Stat(operation.TargetPath)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("target path %q already exists and is not a directory", operation.TargetPath)
		}
		return "Directory already exists.", nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat target directory %q: %w", operation.TargetPath, err)
	}

	if err := os.MkdirAll(operation.TargetPath, 0o755); err != nil {
		return "", fmt.Errorf("create target directory %q: %w", operation.TargetPath, err)
	}
	return "Created directory.", nil
}

func executeCopy(operation operationplan.Operation) (string, error) {
	sourcePath := *operation.SourcePath
	sourceInfo, err := statRegularFile("source file", sourcePath)
	if err != nil {
		return "", err
	}

	if _, err := os.Lstat(operation.TargetPath); err == nil {
		return "", fmt.Errorf("target file %q already exists", operation.TargetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat target file %q: %w", operation.TargetPath, err)
	}

	if err := copyFileAtomic(sourcePath, operation.TargetPath, sourceInfo.Mode().Perm(), false); err != nil {
		return "", err
	}
	return "Copied file.", nil
}

func executeReplace(operation operationplan.Operation) (string, error) {
	sourcePath := *operation.SourcePath
	sourceInfo, err := statRegularFile("source file", sourcePath)
	if err != nil {
		return "", err
	}
	targetInfo, err := statRegularFile("target file", operation.TargetPath)
	if err != nil {
		return "", err
	}

	backupPath := *operation.BackupPath
	if _, err := os.Lstat(backupPath); err == nil {
		return "", fmt.Errorf("backup file %q already exists", backupPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat backup file %q: %w", backupPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return "", fmt.Errorf("create backup directory %q: %w", filepath.Dir(backupPath), err)
	}
	if err := copyFileAtomic(operation.TargetPath, backupPath, targetInfo.Mode().Perm(), false); err != nil {
		return "", fmt.Errorf("create backup file %q: %w", backupPath, err)
	}
	if err := copyFileAtomic(sourcePath, operation.TargetPath, sourceInfo.Mode().Perm(), true); err != nil {
		return "", err
	}
	return "Replaced file and created backup.", nil
}

func statRegularFile(label string, path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s %q does not exist", label, path)
		}
		return nil, fmt.Errorf("stat %s %q: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %q is not a regular file", label, path)
	}

	return info, nil
}

func copyFileAtomic(sourcePath string, targetPath string, mode fs.FileMode, replace bool) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer source.Close()

	targetDirectory := filepath.Dir(targetPath)
	tempFile, err := os.CreateTemp(targetDirectory, ".mod-manager-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", targetDirectory, err)
	}
	tempPath := tempFile.Name()
	shouldRemoveTemp := true
	defer func() {
		if shouldRemoveTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tempFile, source); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("copy %q to temporary file %q: %w", sourcePath, tempPath, err)
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set temporary file mode %q: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary file %q: %w", tempPath, err)
	}

	if replace {
		if err := os.Rename(tempPath, targetPath); err == nil {
			shouldRemoveTemp = false
			return nil
		}
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("remove existing target file %q: %w", targetPath, err)
		}
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("move temporary file %q to %q: %w", tempPath, targetPath, err)
	}
	shouldRemoveTemp = false
	return nil
}

func newFailedResult(index int, operation operationplan.Operation, err error) operationplan.ApplyOperationResult {
	errorMessage := err.Error()
	return operationplan.ApplyOperationResult{
		OperationIndex: index,
		Operation:      operation,
		Status:         operationplan.ApplyOperationStatusFailed,
		Message:        "Operation failed.",
		Error:          &errorMessage,
	}
}

func appendSkippedResults(operations []operationplan.Operation, startIndex int, result *operationplan.ApplyOperationPlanResult) {
	for index := startIndex; index < len(operations); index++ {
		result.Results = append(result.Results, operationplan.ApplyOperationResult{
			OperationIndex: index,
			Operation:      operations[index],
			Status:         operationplan.ApplyOperationStatusSkipped,
			Message:        "Skipped after a previous operation failed.",
		})
	}
}

func updateCounts(result *operationplan.ApplyOperationPlanResult) {
	result.CompletedCount = 0
	result.FailedCount = 0
	result.SkippedCount = 0

	for _, operationResult := range result.Results {
		switch operationResult.Status {
		case operationplan.ApplyOperationStatusCompleted:
			result.CompletedCount++
		case operationplan.ApplyOperationStatusFailed:
			result.FailedCount++
		case operationplan.ApplyOperationStatusSkipped:
			result.SkippedCount++
		}
	}
}
