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

const journalActionIncrementalApply journalAction = "incremental_apply"

func Execute(ctx context.Context, execContext Context, saver AppliedStateSaver) (result Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute incremental deployment: %w", err)
		}
	}()

	if saver == nil {
		return Result{}, errors.New("applied state saver is required")
	}
	if execContext.Plan.Mode != planner.PlanModeIncremental {
		return Result{}, errors.New("deployment plan mode must be incremental")
	}
	if !execContext.Plan.CanApply() {
		return Result{}, errors.New("deployment plan has blocking issues")
	}

	operations, skippedCount, err := BuildOperations(execContext.Plan, execContext.Desired, execContext.GameInstallPath)
	if err != nil {
		return Result{}, err
	}

	if len(operations) == 0 {
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

	journalID := fmt.Sprintf("%d-%x", now().UnixNano(), []byte(execContext.PreviewHash[:min(6, len(execContext.PreviewHash))]))
	journalRoot := filepath.Join(execContext.GameModStoragePath, "deployment-journals", journalID)
	journalPath := filepath.Join(execContext.GameModStoragePath, "deployment-journals", journalID+".json")

	journal := filetxn.Journal[journalAction]{
		Version:   journalVersion,
		ID:        journalID,
		GameID:    execContext.GameID,
		StartedAt: now(),
		Action:    journalActionIncrementalApply,
	}

	journal.Snapshots, err = filetxn.SnapshotOperations(journalRoot, operations)
	if err != nil {
		return Result{}, err
	}
	if err := filetxn.WriteJournal(journalPath, journal); err != nil {
		return Result{}, err
	}

	rollback := func(operationErr error) (Result, error) {
		journal.Error = operationErr.Error()
		_ = filetxn.WriteJournal(journalPath, journal)
		rollbackErr := filetxn.RollbackSnapshots(journal.Snapshots)
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

	if err := saver.SaveIncrementalAppliedProfileState(
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
			Message:        "Incremental deployment completed; journal cleanup was forced.",
		}, nil
	}
	if err := removeDeploymentJournal(journalPath, journalRoot); err != nil {
		return Result{}, err
	}

	message := "Incremental deployment completed."
	if len(operations) == 1 {
		message = "Incremental deployment completed 1 file change."
	} else if len(operations) > 1 {
		message = fmt.Sprintf("Incremental deployment completed %d file changes.", len(operations))
	}

	return Result{
		Success:        true,
		CompletedCount: len(operations),
		SkippedCount:   skippedCount,
		Message:        message,
	}, nil
}

func removeDeploymentJournal(journalPath string, journalRoot string) error {
	if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = os.RemoveAll(journalRoot)
	return nil
}
