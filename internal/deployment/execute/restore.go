package execute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/filetxn"
)

const journalActionRestoreVanilla journalAction = "restore_vanilla"

func ExecuteRestore(ctx context.Context, restoreContext RestoreContext) (result VanillaRestoreResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute restore: %w", err)
		}
	}()

	if restoreContext.Plan.Mode != planner.PlanModeRestorePreview {
		return VanillaRestoreResult{}, errors.New("deployment plan mode must be restore_preview")
	}
	if !restoreContext.Plan.CanApply() {
		return failedVanillaPreflightResult(restoreContext.Plan, planner.PreflightRestorePlan(
			restoreContext.Plan,
			restoreContext.GameInstallPath,
			restoreContext.GameModStoragePath,
		)), nil
	}

	preflightFailures := planner.PreflightRestorePlan(
		restoreContext.Plan,
		restoreContext.GameInstallPath,
		restoreContext.GameModStoragePath,
	)
	if len(preflightFailures) > 0 {
		return failedVanillaPreflightResult(restoreContext.Plan, preflightFailures), nil
	}

	gameInstallPath, err := fileops.CleanRequiredAbsPath("game install path", restoreContext.GameInstallPath)
	if err != nil {
		return VanillaRestoreResult{}, err
	}
	gameModStoragePath, err := fileops.CleanRequiredAbsPath("game mod storage path", restoreContext.GameModStoragePath)
	if err != nil {
		return VanillaRestoreResult{}, err
	}

	emptyDesired := deployment.DesiredState{Files: map[string]deployment.DesiredFile{}}
	operations, skippedCount, err := BuildOperations(restoreContext.Plan, emptyDesired, gameInstallPath)
	if err != nil {
		return VanillaRestoreResult{}, err
	}

	fileResults := make([]VanillaRestoreOperationResult, 0, len(operations)+len(restoreContext.CreatedDirectories))
	operationIndex := 0

	if len(operations) == 0 {
		return finalizeVanillaRestore(
			restoreContext,
			gameInstallPath,
			gameModStoragePath,
			fileResults,
			operationIndex,
			skippedCount,
		)
	}

	if err := filetxn.ValidateOperations(operations, gameInstallPath, gameModStoragePath); err != nil {
		return VanillaRestoreResult{}, err
	}

	archiveTimestamp := time.Now()
	journalID := fmt.Sprintf("%d-restore", archiveTimestamp.UnixNano())
	journalRoot := filepath.Join(gameModStoragePath, "deployment-journals", journalID)
	journalPath := filepath.Join(gameModStoragePath, "deployment-journals", journalID+".json")

	journal := filetxn.Journal[journalAction]{
		Version:   journalVersion,
		ID:        journalID,
		GameID:    restoreContext.GameID,
		StartedAt: archiveTimestamp,
		Action:    journalActionRestoreVanilla,
	}

	journal.Snapshots, err = filetxn.SnapshotOperations(journalRoot, operations)
	if err != nil {
		return VanillaRestoreResult{}, err
	}
	if err := filetxn.WriteJournal(journalPath, journal); err != nil {
		return VanillaRestoreResult{}, err
	}

	rollback := func(operationErr error) (VanillaRestoreResult, error) {
		journal.Error = operationErr.Error()
		_ = filetxn.WriteJournal(journalPath, journal)
		rollbackErr := filetxn.RollbackSnapshots(journal.Snapshots)
		if rollbackErr != nil {
			return buildFailedVanillaFileResult(restoreContext.Plan, fileResults, operationIndex, skippedCount, operationErr), nil
		}
		return buildFailedVanillaFileResult(restoreContext.Plan, fileResults, operationIndex, skippedCount, operationErr), nil
	}

	for index, operation := range operations {
		pathPlan := pathPlanForOperation(restoreContext.Plan, operation, gameInstallPath)
		restoreOperation := vanillaOperationFromPathPlan(pathPlan, operation, operationIndex)
		message, operationErr := executeVanillaFileOperation(operation)
		if operationErr != nil {
			fileResults = append(fileResults, failedVanillaOperationResult(operationIndex, restoreOperation, operationErr))
			result, returnErr := rollback(operationErr)
			result.Results = append(result.Results, fileResults...)
			updateVanillaRestoreCounts(&result)
			return result, returnErr
		}

		fileResults = append(fileResults, VanillaRestoreOperationResult{
			OperationIndex: operationIndex,
			Operation:      restoreOperation,
			Status:         VanillaRestoreOperationStatusCompleted,
			Message:        message,
		})
		operationIndex++
		_ = index

		journal.CompletedSteps = index + 1
		if err := filetxn.WriteJournal(journalPath, journal); err != nil {
			result, returnErr := rollback(err)
			result.Results = append(result.Results, fileResults...)
			updateVanillaRestoreCounts(&result)
			return result, returnErr
		}
	}

	if err := filetxn.VerifyOperations(operations); err != nil {
		result, returnErr := rollback(err)
		result.Results = append(result.Results, fileResults...)
		updateVanillaRestoreCounts(&result)
		return result, returnErr
	}

	_ = removeDeploymentJournal(journalPath, journalRoot)

	return finalizeVanillaRestore(
		restoreContext,
		gameInstallPath,
		gameModStoragePath,
		fileResults,
		operationIndex,
		skippedCount,
	)
}

func finalizeVanillaRestore(
	restoreContext RestoreContext,
	gameInstallPath string,
	gameModStoragePath string,
	fileResults []VanillaRestoreOperationResult,
	operationIndex int,
	skippedCount int,
) (VanillaRestoreResult, error) {
	directoryResults, nextIndex, err := removeRestoreCreatedDirectories(
		restoreContext.CreatedDirectories,
		gameInstallPath,
		operationIndex,
	)
	if err != nil {
		result := VanillaRestoreResult{
			Success: false,
			Results: append(fileResults, directoryResults...),
		}
		updateVanillaRestoreCounts(&result)
		return result, nil
	}
	fileResults = append(fileResults, directoryResults...)
	operationIndex = nextIndex

	backupResults, err := deleteRestoreBaselineBackups(restoreContext.Plan, gameModStoragePath, operationIndex)
	if err != nil {
		result := VanillaRestoreResult{
			Success: false,
			Results: append(fileResults, backupResults...),
		}
		updateVanillaRestoreCounts(&result)
		return result, nil
	}
	fileResults = append(fileResults, backupResults...)

	result := VanillaRestoreResult{
		Success: true,
		Results: fileResults,
	}
	updateVanillaRestoreCounts(&result)
	result.SkippedCount += skippedCount

	return result, nil
}

func executeVanillaFileOperation(operation filetxn.Operation) (string, error) {
	if err := filetxn.ExecuteOperation(operation, "restore file"); err != nil {
		return "", err
	}

	switch operation.Type {
	case "delete":
		return "Removed added file.", nil
	case "restore":
		return "Restored replaced file from backup.", nil
	default:
		return "Completed restore operation.", nil
	}
}

func removeRestoreCreatedDirectories(
	directories []RestoreCreatedDirectory,
	gameInstallPath string,
	startIndex int,
) ([]VanillaRestoreOperationResult, int, error) {
	sorted := sortedRestoreCreatedDirectories(directories, gameInstallPath)
	results := make([]VanillaRestoreOperationResult, 0, len(sorted))
	operationIndex := startIndex

	for _, directory := range sorted {
		restoreOperation := VanillaRestoreOperation{
			Type:                   VanillaRestoreOperationRemoveCreatedDir,
			ManifestOperationIndex: operationIndex,
			TargetPath:             directory.TargetPath,
		}
		if directory.ModID != nil {
			restoreOperation.Mod.ID = *directory.ModID
		}
		if directory.ModName != nil {
			restoreOperation.Mod.Name = *directory.ModName
		}

		message, err := removeRestoreCreatedDirectory(directory.TargetPath)
		if err != nil {
			results = append(results, failedVanillaOperationResult(operationIndex, restoreOperation, err))
			return results, operationIndex + 1, err
		}

		results = append(results, VanillaRestoreOperationResult{
			OperationIndex: operationIndex,
			Operation:      restoreOperation,
			Status:         VanillaRestoreOperationStatusCompleted,
			Message:        message,
		})
		operationIndex++
	}

	return results, operationIndex, nil
}

type sortedRestoreDirectory struct {
	TargetPath       string
	GameRelativePath string
	ModID            *int64
	ModName          *string
}

func sortedRestoreCreatedDirectories(directories []RestoreCreatedDirectory, gameInstallPath string) []sortedRestoreDirectory {
	sorted := make([]sortedRestoreDirectory, 0, len(directories))
	for _, directory := range directories {
		targetPath, err := targetAbsolutePath(gameInstallPath, directory.GameRelativePath, directory.GameRelativePath)
		if err != nil {
			continue
		}
		sorted = append(sorted, sortedRestoreDirectory{
			TargetPath:       targetPath,
			GameRelativePath: directory.GameRelativePath,
			ModID:            directory.ModID,
			ModName:          directory.ModName,
		})
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		iPath := cleanRestorePathForSort(sorted[i].TargetPath)
		jPath := cleanRestorePathForSort(sorted[j].TargetPath)
		iDepth := restorePathDepth(iPath)
		jDepth := restorePathDepth(jPath)
		if iDepth != jDepth {
			return iDepth > jDepth
		}

		return iPath > jPath
	})

	return sorted
}

func removeRestoreCreatedDirectory(targetPath string) (string, error) {
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

func deleteRestoreBaselineBackups(
	plan planner.DeploymentPlan,
	gameModStoragePath string,
	startIndex int,
) ([]VanillaRestoreOperationResult, error) {
	results := make([]VanillaRestoreOperationResult, 0)
	operationIndex := startIndex

	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]
		if pathPlan.PlannedAction != planner.ReapplyRestoreBaseline {
			continue
		}
		if strings.TrimSpace(pathPlan.BaselineBackupPath) == "" {
			continue
		}

		backupPath := pathPlan.BaselineBackupPath
		restoreOperation := VanillaRestoreOperation{
			Type:                   VanillaRestoreOperationDeleteRestoredBackup,
			ManifestOperationIndex: operationIndex,
			TargetPath:             pathPlan.GameRelativePath,
			BackupPath:             &backupPath,
		}

		if err := os.Remove(backupPath); err != nil {
			results = append(results, failedVanillaOperationResult(operationIndex, restoreOperation, fmt.Errorf("delete restored backup %q: %w", backupPath, err)))
			return results, fmt.Errorf("delete restored backup %q: %w", backupPath, err)
		}
		if err := fileops.RemoveEmptyParentDirectories(filepath.Dir(backupPath), gameModStoragePath); err != nil {
			results = append(results, failedVanillaOperationResult(operationIndex, restoreOperation, err))
			return results, err
		}

		results = append(results, VanillaRestoreOperationResult{
			OperationIndex: operationIndex,
			Operation:      restoreOperation,
			Status:         VanillaRestoreOperationStatusCompleted,
			Message:        "Deleted restored backup.",
		})
		operationIndex++
	}

	return results, nil
}

func vanillaOperationFromPathPlan(pathPlan planner.PathPlan, operation filetxn.Operation, operationIndex int) VanillaRestoreOperation {
	restoreOperation := VanillaRestoreOperation{
		ManifestOperationIndex: operationIndex,
		TargetPath:             operation.TargetPath,
	}
	switch pathPlan.PlannedAction {
	case planner.ReapplyDelete:
		restoreOperation.Type = VanillaRestoreOperationRemoveAddedFile
	case planner.ReapplyRestoreBaseline:
		restoreOperation.Type = VanillaRestoreOperationRestoreReplacedFile
		if strings.TrimSpace(pathPlan.BaselineBackupPath) != "" {
			backupPath := pathPlan.BaselineBackupPath
			restoreOperation.BackupPath = &backupPath
		}
	}

	return restoreOperation
}

func pathPlanForOperation(plan planner.DeploymentPlan, operation filetxn.Operation, gameInstallPath string) planner.PathPlan {
	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]
		targetPath, err := targetAbsolutePath(gameInstallPath, pathPlan.GameRelativePath, canonicalPath)
		if err != nil {
			continue
		}
		if filepath.Clean(targetPath) == filepath.Clean(operation.TargetPath) {
			return pathPlan
		}
	}

	return planner.PathPlan{GameRelativePath: operation.TargetPath}
}

func failedVanillaPreflightResult(plan planner.DeploymentPlan, failures map[string]error) VanillaRestoreResult {
	result := VanillaRestoreResult{
		Success: false,
		Results: make([]VanillaRestoreOperationResult, 0, len(plan.Paths)),
	}
	operationIndex := 0
	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]
		restoreOperation := vanillaOperationFromPathPlan(pathPlan, filetxn.Operation{TargetPath: pathPlan.GameRelativePath}, operationIndex)
		if err, failed := failures[canonicalPath]; failed {
			result.Results = append(result.Results, failedVanillaOperationResult(operationIndex, restoreOperation, err))
		} else {
			result.Results = append(result.Results, VanillaRestoreOperationResult{
				OperationIndex: operationIndex,
				Operation:      restoreOperation,
				Status:         VanillaRestoreOperationStatusSkipped,
				Message:        "Skipped because restore preflight failed.",
			})
		}
		operationIndex++
	}
	updateVanillaRestoreCounts(&result)

	return result
}

func buildFailedVanillaFileResult(
	plan planner.DeploymentPlan,
	completed []VanillaRestoreOperationResult,
	nextIndex int,
	skippedCount int,
	operationErr error,
) VanillaRestoreResult {
	result := VanillaRestoreResult{
		Success: false,
		Results: append([]VanillaRestoreOperationResult(nil), completed...),
	}
	for index := nextIndex; index < len(sortedPlanPaths(plan)); index++ {
		canonicalPath := sortedPlanPaths(plan)[index]
		pathPlan := plan.Paths[canonicalPath]
		restoreOperation := vanillaOperationFromPathPlan(pathPlan, filetxn.Operation{TargetPath: pathPlan.GameRelativePath}, index)
		result.Results = append(result.Results, VanillaRestoreOperationResult{
			OperationIndex: index,
			Operation:      restoreOperation,
			Status:         VanillaRestoreOperationStatusSkipped,
			Message:        "Skipped after a previous restore operation failed.",
		})
	}
	_ = operationErr
	result.SkippedCount = skippedCount
	updateVanillaRestoreCounts(&result)

	return result
}

func failedVanillaOperationResult(index int, operation VanillaRestoreOperation, err error) VanillaRestoreOperationResult {
	errorMessage := err.Error()
	return VanillaRestoreOperationResult{
		OperationIndex: index,
		Operation:      operation,
		Status:         VanillaRestoreOperationStatusFailed,
		Message:        "Restore operation failed.",
		Error:          &errorMessage,
	}
}

func updateVanillaRestoreCounts(result *VanillaRestoreResult) {
	result.CompletedCount = 0
	result.FailedCount = 0
	result.SkippedCount = 0

	for _, operationResult := range result.Results {
		switch operationResult.Status {
		case VanillaRestoreOperationStatusCompleted:
			result.CompletedCount++
		case VanillaRestoreOperationStatusFailed:
			result.FailedCount++
		case VanillaRestoreOperationStatusSkipped:
			result.SkippedCount++
		}
	}
}

func cleanRestorePathForSort(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}

func restorePathDepth(path string) int {
	volume := filepath.VolumeName(path)
	path = strings.TrimPrefix(path, volume)
	path = strings.Trim(path, string(filepath.Separator))
	if path == "" {
		return 0
	}

	return len(strings.Split(path, string(filepath.Separator)))
}
