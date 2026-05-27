package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/phergul/mod-manager/internal/appliedstate"
	"github.com/phergul/mod-manager/internal/applyplan"
	"github.com/phergul/mod-manager/internal/diagnostics"
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/restoreplan"
	"github.com/phergul/mod-manager/internal/services/dto"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

func (s *ProfileService) BuildProfileOperationPlan(ctx context.Context, profileID int64) (plan dto.OperationPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build profile operation plan: %w", err)
		}
	}()

	resolved, err := operationplan.ResolveProfilePlan(ctx, s.store, profileID)
	if err != nil {
		return dto.OperationPlan{}, err
	}

	operationPlan, err := operationplan.BuildOperationPlan(resolved)
	if err != nil {
		return dto.OperationPlan{}, err
	}

	return toDTOOperationPlan(operationPlan), nil
}

func (s *ProfileService) ApplyProfileOperationPlan(ctx context.Context, profileID int64, plan dto.OperationPlan) (result dto.ApplyOperationPlanResult, err error) {
	startedAt := time.Now()
	var gameID int64
	defer func() {
		if err != nil {
			s.logger.ErrorContext(ctx, "Profile apply failed",
				slog.String("operation", diagnostics.OperationApplyProfile),
				slog.String("event", diagnostics.EventFailed),
				slog.Int64("profile_id", profileID),
				slog.Int64("game_id", gameID),
				slog.Int("operation_count", len(plan.Operations)),
				diagnostics.DurationAttr(startedAt),
				diagnostics.ErrorAttr(err),
			)
			err = fmt.Errorf("apply profile operation plan: %w", err)
		}
	}()

	s.logger.InfoContext(ctx, "Profile apply started",
		slog.String("operation", diagnostics.OperationApplyProfile),
		slog.String("event", diagnostics.EventStarted),
		slog.Int64("profile_id", profileID),
		slog.Int("operation_count", len(plan.Operations)),
	)

	if profileID <= 0 {
		return dto.ApplyOperationPlanResult{}, fmt.Errorf("profile ID must be positive")
	}
	if !plan.CanApply {
		return dto.ApplyOperationPlanResult{}, errors.New("operation plan has blocking issues")
	}

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	if !found {
		return dto.ApplyOperationPlanResult{}, fmt.Errorf("profile %d was not found", profileID)
	}
	gameID = profile.GameID
	if appliedState, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID); err != nil {
		return dto.ApplyOperationPlanResult{}, err
	} else if appliedFound {
		return dto.ApplyOperationPlanResult{}, fmt.Errorf("profile %d is already applied for game %d; restore vanilla before applying another profile", appliedState.ProfileID, profile.GameID)
	}

	game, err := s.store.GetStoredGame(ctx, profile.GameID)
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, profile.GameID, "")
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}

	internalPlan := toInternalOperationPlan(plan)
	applyResult, err := applyplan.Execute(internalPlan, applyplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	result = toDTOApplyOperationPlanResult(applyResult)
	if !applyResult.Success {
		s.logger.WarnContext(ctx, "Profile apply completed with failures",
			slog.String("operation", diagnostics.OperationApplyProfile),
			slog.String("event", diagnostics.EventCompleted),
			slog.Bool("success", false),
			slog.Int64("profile_id", profileID),
			slog.Int64("game_id", profile.GameID),
			slog.Int("completed_count", applyResult.CompletedCount),
			slog.Int("failed_count", applyResult.FailedCount),
			slog.Int("skipped_count", applyResult.SkippedCount),
			slog.String("failure_summary", applyFailureSummary(applyResult)),
			diagnostics.DurationAttr(startedAt),
		)
		return result, nil
	}

	if err := s.saveAppliedProfileState(ctx, game.ID, profileID, internalPlan, applyResult.Manifest); err != nil {
		return result, err
	}

	s.logger.InfoContext(ctx, "Profile apply completed",
		slog.String("operation", diagnostics.OperationApplyProfile),
		slog.String("event", diagnostics.EventCompleted),
		slog.Bool("success", true),
		slog.Int64("profile_id", profileID),
		slog.Int64("game_id", profile.GameID),
		slog.Int("completed_count", applyResult.CompletedCount),
		slog.Int("failed_count", applyResult.FailedCount),
		slog.Int("skipped_count", applyResult.SkippedCount),
		diagnostics.DurationAttr(startedAt),
	)

	return result, nil
}

func (s *ProfileService) RestoreVanillaState(ctx context.Context, gameID int64) (result dto.RestoreResult, err error) {
	startedAt := time.Now()
	var profileID int64
	defer func() {
		if err != nil {
			s.logger.ErrorContext(ctx, "Vanilla restore failed",
				slog.String("operation", diagnostics.OperationRestoreVanilla),
				slog.String("event", diagnostics.EventFailed),
				slog.Int64("game_id", gameID),
				slog.Int64("profile_id", profileID),
				diagnostics.DurationAttr(startedAt),
				diagnostics.ErrorAttr(err),
			)
			err = fmt.Errorf("restore vanilla state: %w", err)
		}
	}()

	s.logger.InfoContext(ctx, "Vanilla restore started",
		slog.String("operation", diagnostics.OperationRestoreVanilla),
		slog.String("event", diagnostics.EventStarted),
		slog.Int64("game_id", gameID),
	)

	if gameID <= 0 {
		return dto.RestoreResult{}, errors.New("game ID must be positive")
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
		return dto.RestoreResult{}, fmt.Errorf("no applied profile state found for game %d", gameID)
	}
	profileID = state.ProfileID

	manifest, err := appliedstate.DecodeManifest(state.ManifestJSON)
	if err != nil {
		return dto.RestoreResult{}, err
	}
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, gameID, "")
	if err != nil {
		return dto.RestoreResult{}, err
	}

	restoreResult, err := restoreplan.Execute(manifest, restoreplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
	if err != nil {
		return dto.RestoreResult{}, err
	}
	result = toDTORestoreResult(restoreResult)
	if !restoreResult.Success {
		s.logger.WarnContext(ctx, "Vanilla restore completed with failures",
			slog.String("operation", diagnostics.OperationRestoreVanilla),
			slog.String("event", diagnostics.EventCompleted),
			slog.Bool("success", false),
			slog.Int64("game_id", gameID),
			slog.Int64("profile_id", profileID),
			slog.Int("completed_count", restoreResult.CompletedCount),
			slog.Int("failed_count", restoreResult.FailedCount),
			slog.Int("skipped_count", restoreResult.SkippedCount),
			slog.String("failure_summary", restoreFailureSummary(restoreResult)),
			diagnostics.DurationAttr(startedAt),
		)
		return result, nil
	}

	if err := s.store.DeleteAppliedProfileState(ctx, gameID); err != nil {
		return result, err
	}

	s.logger.InfoContext(ctx, "Vanilla restore completed",
		slog.String("operation", diagnostics.OperationRestoreVanilla),
		slog.String("event", diagnostics.EventCompleted),
		slog.Bool("success", true),
		slog.Int64("game_id", gameID),
		slog.Int64("profile_id", profileID),
		slog.Int("completed_count", restoreResult.CompletedCount),
		slog.Int("failed_count", restoreResult.FailedCount),
		slog.Int("skipped_count", restoreResult.SkippedCount),
		diagnostics.DurationAttr(startedAt),
	)

	return result, nil
}

func applyFailureSummary(result operationplan.ApplyOperationPlanResult) string {
	for _, operationResult := range result.Results {
		if operationResult.Error != nil && *operationResult.Error != "" {
			return *operationResult.Error
		}
	}

	return ""
}

func restoreFailureSummary(result restoreplan.RestoreResult) string {
	for _, operationResult := range result.Results {
		if operationResult.Error != nil && *operationResult.Error != "" {
			return *operationResult.Error
		}
	}

	return ""
}

func (s *ProfileService) saveAppliedProfileState(ctx context.Context, gameID int64, profileID int64, plan operationplan.OperationPlan, manifest operationplan.AppliedOperationManifest) error {
	manifestJSON, err := appliedstate.EncodeManifest(appliedstate.BuildManifestDocument(manifest))
	if err != nil {
		return fmt.Errorf("encode applied manifest: %w", err)
	}

	snapshot, err := appliedstate.EncodeProfileSnapshot(appliedstate.BuildProfileSnapshotDocument(plan))
	if err != nil {
		return fmt.Errorf("encode profile snapshot: %w", err)
	}

	profileMods, err := s.store.ListProfileMods(ctx, profileID)
	if err != nil {
		return err
	}
	compositionSnapshot, err := encodeProfileCompositionSnapshot(profileID, profileMods)
	if err != nil {
		return fmt.Errorf("encode profile composition snapshot: %w", err)
	}

	_, err = s.store.SaveAppliedProfileState(ctx, dbtypes.SaveAppliedProfileStateInput{
		GameID:                         gameID,
		ProfileID:                      profileID,
		ManifestJSON:                   manifestJSON,
		ProfileSnapshotJSON:            snapshot.JSON,
		ProfileSnapshotHash:            snapshot.Hash,
		ProfileCompositionSnapshotJSON: &compositionSnapshot.JSON,
		ProfileCompositionSnapshotHash: &compositionSnapshot.Hash,
	})
	if err != nil {
		return fmt.Errorf("save applied profile state: %w", err)
	}

	return nil
}
