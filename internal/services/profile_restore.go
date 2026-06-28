package services

import (
	"context"
	"log/slog"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/deployment/execute"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
)

func (s *ProfileService) RestoreVanillaState(ctx context.Context, gameID int64) (result dto.RestoreResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRestoreVanilla, "Vanilla restore started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Vanilla restore failed", err, profileRestoreUserError)
		}
	}()

	if gameID <= 0 {
		return dto.RestoreResult{}, apperror.New("A valid game must be selected.")
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return dto.RestoreResult{}, err
	}
	state, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return dto.RestoreResult{}, err
	}
	if !found {
		return dto.RestoreResult{}, apperror.New("No profile is currently applied for this game.")
	}
	diag.attrs = append(diag.attrs,
		slog.String("game_name", game.Name),
		slog.Int64("profile_id", state.ProfileID),
	)

	appliedStates, err := s.LoadAppliedFileStates(ctx, gameID)
	if err != nil {
		return dto.RestoreResult{}, err
	}
	if len(appliedStates) == 0 {
		return dto.RestoreResult{}, apperror.New("No applied file state is recorded for this game.")
	}

	createdDirectoryRows, err := s.store.ListAppliedCreatedDirectories(ctx, gameID)
	if err != nil {
		return dto.RestoreResult{}, err
	}

	restorePlan, err := planner.PlanRestorePreview(appliedStates, game.InstallPath)
	if err != nil {
		return dto.RestoreResult{}, err
	}
	if !restorePlan.CanApply() {
		return dto.RestoreResult{}, apperror.New("Restore vanilla is blocked because required baseline backups are missing.")
	}

	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, gameID, "")
	if err != nil {
		return dto.RestoreResult{}, err
	}

	restoreResult, err := execute.ExecuteRestore(ctx, execute.RestoreContext{
		GameID:             gameID,
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
		Plan:               restorePlan,
		CreatedDirectories: fromDBAppliedCreatedDirectoryRows(createdDirectoryRows),
	})
	if err != nil {
		return dto.RestoreResult{}, err
	}
	result = mappers.ToDTORestoreResult(restoreResult)
	if !restoreResult.Success {
		diag.warn("Vanilla restore completed with failures",
			slog.Bool("success", false),
			slog.Int("completed_count", restoreResult.CompletedCount),
			slog.Int("failed_count", restoreResult.FailedCount),
			slog.Int("skipped_count", restoreResult.SkippedCount),
			slog.String("failure_summary", restoreFailureSummary(restoreResult)),
		)
		return result, nil
	}

	if err := s.store.DeleteAppliedProfileState(ctx, gameID); err != nil {
		return result, err
	}

	diag.complete("Vanilla restore completed",
		slog.Bool("success", true),
		slog.Int("completed_count", restoreResult.CompletedCount),
		slog.Int("failed_count", restoreResult.FailedCount),
		slog.Int("skipped_count", restoreResult.SkippedCount),
	)

	return result, nil
}

func restoreFailureSummary(result execute.VanillaRestoreResult) string {
	for _, operationResult := range result.Results {
		if operationResult.Error != nil && *operationResult.Error != "" {
			return *operationResult.Error
		}
	}

	return ""
}
