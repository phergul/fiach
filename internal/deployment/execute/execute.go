package execute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/filetxn"
)

type journalAction string

const (
	journalActionFirstApply       journalAction = "first_apply"
	journalActionIncrementalApply journalAction = "incremental_apply"
)

func Execute(ctx context.Context, execContext Context, saver AppliedStateSaver) (result Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute deployment: %w", err)
		}
	}()

	if saver == nil {
		return Result{}, errors.New("applied state saver is required")
	}
	if execContext.Plan.Mode != planner.PlanModeFirstApply && execContext.Plan.Mode != planner.PlanModeIncremental {
		return Result{}, errors.New("deployment plan mode must be first_apply or incremental")
	}
	if !execContext.Plan.CanApply() {
		return Result{}, errors.New("deployment plan has blocking issues")
	}

	firstApplyOutcome := execContext.FirstApplyOutcome
	if execContext.Plan.Mode == planner.PlanModeFirstApply {
		firstApplyOutcome, err = PrepareFirstApply(
			execContext.Plan,
			execContext.Desired,
			execContext.GameInstallPath,
			execContext.GameModStoragePath,
		)
		if err != nil {
			return Result{}, err
		}
		execContext.FirstApplyOutcome = firstApplyOutcome
	}

	operations, skippedCount, err := BuildOperations(execContext.Plan, execContext.Desired, execContext.GameInstallPath)
	if err != nil {
		return Result{}, err
	}

	if len(operations) == 0 {
		if execContext.Plan.Mode == planner.PlanModeFirstApply {
			if err := saver.SaveFirstApplyAppliedProfileState(
				ctx,
				execContext.GameID,
				execContext.ProfileID,
				execContext.GameInstallPath,
				execContext.Plan,
				execContext.Desired,
				firstApplyOutcome,
			); err != nil {
				return Result{}, err
			}
		}

		return Result{
			Success:        true,
			CompletedCount: 0,
			SkippedCount:   skippedCount,
			Message:        "No file changes were needed.",
		}, nil
	}

	if err := filetxn.ValidateOperations(operations, execContext.GameInstallPath, execContext.GameModStoragePath); err != nil {
		return Result{}, err
	}

	now := execContext.Now
	if now == nil {
		now = time.Now
	}

	archiveTimestamp := now()
	if execContext.Plan.Mode == planner.PlanModeIncremental {
		if _, err := archiveDriftedFiles(
			execContext.GameInstallPath,
			execContext.GameModStoragePath,
			execContext.GameID,
			execContext.Plan,
			archiveTimestamp,
		); err != nil {
			return Result{}, err
		}
	}

	journalActionValue := journalActionIncrementalApply
	if execContext.Plan.Mode == planner.PlanModeFirstApply {
		journalActionValue = journalActionFirstApply
	}

	journalID := fmt.Sprintf("%d-%x", archiveTimestamp.UnixNano(), []byte(execContext.PreviewHash[:min(6, len(execContext.PreviewHash))]))
	journalRoot := filepath.Join(execContext.GameModStoragePath, "deployment-journals", journalID)
	journalPath := filepath.Join(execContext.GameModStoragePath, "deployment-journals", journalID+".json")

	journal := filetxn.Journal[journalAction]{
		Version:   journalVersion,
		ID:        journalID,
		GameID:    execContext.GameID,
		StartedAt: archiveTimestamp,
		Action:    journalActionValue,
	}

	journal.Snapshots, err = filetxn.SnapshotOperations(journalRoot, operations)
	if err != nil {
		if execContext.Plan.Mode == planner.PlanModeFirstApply {
			_ = rollbackFirstApplyBaselines(firstApplyOutcome)
		}
		return Result{}, err
	}
	if err := filetxn.WriteJournal(journalPath, journal); err != nil {
		if execContext.Plan.Mode == planner.PlanModeFirstApply {
			_ = rollbackFirstApplyBaselines(firstApplyOutcome)
		}
		return Result{}, err
	}

	rollback := func(operationErr error) (Result, error) {
		journal.Error = operationErr.Error()
		_ = filetxn.WriteJournal(journalPath, journal)
		rollbackErr := filetxn.RollbackSnapshots(journal.Snapshots)
		if execContext.Plan.Mode == planner.PlanModeFirstApply {
			if baselineErr := rollbackFirstApplyBaselines(firstApplyOutcome); baselineErr != nil && rollbackErr == nil {
				rollbackErr = baselineErr
			}
		}
		if rollbackErr != nil {
			return Result{
				RolledBack: false,
				Message:    fmt.Sprintf("%v; rollback failed: %v", operationErr, rollbackErr),
			}, operationErr
		}
		return Result{
			RolledBack: true,
			Message:    operationErr.Error(),
		}, operationErr
	}

	for index, operation := range operations {
		if err := filetxn.ExecuteOperation(operation, "deployment file"); err != nil {
			failedResult, returnErr := rollback(err)
			failedResult.SkippedCount = skippedCount
			failedResult.CompletedCount = index
			return failedResult, returnErr
		}
		journal.CompletedSteps = index + 1
		if err := filetxn.WriteJournal(journalPath, journal); err != nil {
			failedResult, returnErr := rollback(err)
			failedResult.SkippedCount = skippedCount
			failedResult.CompletedCount = index + 1
			return failedResult, returnErr
		}
	}

	if err := filetxn.VerifyOperations(operations); err != nil {
		failedResult, returnErr := rollback(err)
		failedResult.SkippedCount = skippedCount
		failedResult.CompletedCount = len(operations)
		return failedResult, returnErr
	}

	if execContext.Plan.Mode == planner.PlanModeFirstApply {
		if err := saver.SaveFirstApplyAppliedProfileState(
			ctx,
			execContext.GameID,
			execContext.ProfileID,
			execContext.GameInstallPath,
			execContext.Plan,
			execContext.Desired,
			firstApplyOutcome,
		); err != nil {
			failedResult, returnErr := rollback(err)
			failedResult.SkippedCount = skippedCount
			failedResult.CompletedCount = len(operations)
			return failedResult, returnErr
		}
	} else if err := saver.SaveIncrementalAppliedProfileState(
		ctx,
		execContext.GameID,
		execContext.ProfileID,
		execContext.GameInstallPath,
		execContext.Plan,
		execContext.Desired,
		execContext.AppliedFileStates,
	); err != nil {
		failedResult, returnErr := rollback(err)
		failedResult.SkippedCount = skippedCount
		failedResult.CompletedCount = len(operations)
		return failedResult, returnErr
	}

	journal.DatabaseCommitted = true
	if err := filetxn.WriteJournal(journalPath, journal); err != nil {
		_ = removeDeploymentJournal(journalPath, journalRoot)
		return Result{
			Success:        true,
			CompletedCount: len(operations),
			SkippedCount:   skippedCount,
			Message:        deploymentSuccessMessage(execContext.Plan.Mode, len(operations), true),
		}, nil
	}
	if err := removeDeploymentJournal(journalPath, journalRoot); err != nil {
		return Result{}, err
	}

	return Result{
		Success:        true,
		CompletedCount: len(operations),
		SkippedCount:   skippedCount,
		Message:        deploymentSuccessMessage(execContext.Plan.Mode, len(operations), false),
	}, nil
}

func deploymentSuccessMessage(mode planner.PlanMode, completedCount int, forcedCleanup bool) string {
	prefix := "Deployment"
	if mode == planner.PlanModeIncremental {
		prefix = "Incremental deployment"
	} else if mode == planner.PlanModeFirstApply {
		prefix = "Profile deployment"
	}

	if forcedCleanup {
		return fmt.Sprintf("%s completed; journal cleanup was forced.", prefix)
	}
	if completedCount == 0 {
		return fmt.Sprintf("%s completed.", prefix)
	}
	if completedCount == 1 {
		return fmt.Sprintf("%s completed 1 file change.", prefix)
	}
	return fmt.Sprintf("%s completed %d file changes.", prefix, completedCount)
}

func rollbackFirstApplyBaselines(outcome FirstApplyOutcome) error {
	var lastErr error
	for _, backupEntry := range outcome.BaselineBackups {
		if err := os.Remove(backupEntry.BackupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			lastErr = err
		}
	}
	return lastErr
}

func removeDeploymentJournal(journalPath string, journalRoot string) error {
	if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = os.RemoveAll(journalRoot)
	return nil
}
