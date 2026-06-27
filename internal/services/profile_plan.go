package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/applyplan"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/restoreplan"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (s *ProfileService) BuildProfileOperationPlan(ctx context.Context, profileID int64) (plan dto.OperationPlan, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationBuildProfilePlan, "Profile operation plan build started",
		slog.Int64("profile_id", profileID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile operation plan build failed", err, profilePlanUserError)
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

	plan = mappers.ToDTOOperationPlan(operationPlan)
	diag.complete("Profile operation plan build completed",
		slog.Bool("can_apply", plan.CanApply),
		slog.Int("issue_count", len(plan.Issues)),
		slog.Int("operation_count", len(plan.Operations)),
	)

	return plan, nil
}

func (s *ProfileService) ApplyProfileOperationPlan(ctx context.Context, profileID int64, plan dto.OperationPlan) (result dto.ApplyOperationPlanResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationApplyProfile, "Profile apply started",
		slog.Int64("profile_id", profileID),
		slog.Int("operation_count", len(plan.Operations)),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Profile apply failed", err, profilePlanUserError)
		}
	}()

	if profileID <= 0 {
		return dto.ApplyOperationPlanResult{}, apperror.New("A valid profile must be selected.")
	}
	if !plan.CanApply {
		return dto.ApplyOperationPlanResult{}, apperror.New("Fix the issues in the plan before applying.")
	}

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	if !found {
		return dto.ApplyOperationPlanResult{}, apperror.New("Profile was not found.")
	}
	if _, appliedFound, err := s.store.GetAppliedProfileState(ctx, profile.GameID); err != nil {
		return dto.ApplyOperationPlanResult{}, err
	} else if appliedFound {
		return dto.ApplyOperationPlanResult{}, apperror.New("Restore vanilla before applying another profile.")
	}

	game, err := s.store.GetStoredGame(ctx, profile.GameID)
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	diag.attrs = append(diag.attrs,
		slog.String("profile_name", profile.Name),
		slog.Int64("game_id", profile.GameID),
		slog.String("game_name", game.Name),
	)
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, profile.GameID, "")
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}

	internalPlan := mappers.ToInternalOperationPlan(plan)
	applyResult, err := applyplan.Execute(internalPlan, applyplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
	if err != nil {
		return dto.ApplyOperationPlanResult{}, err
	}
	result = mappers.ToDTOApplyOperationPlanResult(applyResult)
	if !applyResult.Success {
		diag.warn("Profile apply completed with failures",
			slog.Bool("success", false),
			slog.Int("completed_count", applyResult.CompletedCount),
			slog.Int("failed_count", applyResult.FailedCount),
			slog.Int("skipped_count", applyResult.SkippedCount),
			slog.String("failure_summary", applyFailureSummary(applyResult)),
		)
		return result, nil
	}

	if err := s.saveAppliedProfileState(ctx, game.ID, profileID, game.InstallPath, internalPlan, applyResult.Manifest); err != nil {
		return result, err
	}

	diag.complete("Profile apply completed",
		slog.Bool("success", true),
		slog.Int("completed_count", applyResult.CompletedCount),
		slog.Int("failed_count", applyResult.FailedCount),
		slog.Int("skipped_count", applyResult.SkippedCount),
	)

	return result, nil
}

func (s *ProfileService) RestoreVanillaState(ctx context.Context, gameID int64) (result dto.RestoreResult, err error) {
	diag := startDiagnosticOperation(ctx, s.logger, diagnostics.OperationRestoreVanilla, "Vanilla restore started",
		slog.Int64("game_id", gameID),
	)
	defer func() {
		if err != nil {
			err = diag.failWithMappedError("Vanilla restore failed", err, profilePlanUserError)
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

func (s *ProfileService) LoadAppliedFileStates(ctx context.Context, gameID int64) (states []appliedstate.PersistedFileState, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("load applied file states: %w", err)
		}
	}()

	if gameID <= 0 {
		return nil, apperror.New("A valid game must be selected.")
	}

	appliedState, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if !found {
		return []appliedstate.PersistedFileState{}, nil
	}

	hasRows, err := s.store.HasAppliedFileStates(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if hasRows {
		rows, err := s.store.ListAppliedFileStates(ctx, gameID)
		if err != nil {
			return nil, err
		}

		return fromDBAppliedFileStateRows(rows), nil
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return nil, err
	}

	fileStates, err := appliedstate.FileStatesFromStoredManifest(
		appliedState.ManifestJSON,
		game.InstallPath,
		appliedState.ProfileID,
		appliedState.AppliedAt,
	)
	if err != nil {
		return nil, err
	}

	dbRows := toDBAppliedFileStateRows(gameID, fileStates, appliedState.AppliedAt)
	if err := s.store.ReplaceAppliedFileStates(ctx, dbtypes.ReplaceAppliedFileStatesInput{
		GameID:     gameID,
		ProfileID:  appliedState.ProfileID,
		FileStates: dbRows,
	}); err != nil {
		return nil, err
	}

	return withAppliedAt(fileStates, appliedState.AppliedAt), nil
}

func (s *ProfileService) saveAppliedProfileState(ctx context.Context, gameID int64, profileID int64, installPath string, plan operationplan.OperationPlan, manifest operationplan.AppliedOperationManifest) error {
	manifestDocument := appliedstate.BuildManifestDocument(manifest)
	fileStates, err := appliedstate.BuildFileStatesFromManifest(manifestDocument, installPath, profileID, "")
	if err != nil {
		return fmt.Errorf("build applied file states: %w", err)
	}
	appliedstate.AttachManifestFiles(&manifestDocument, fileStates)

	manifestJSON, err := appliedstate.EncodeManifest(manifestDocument)
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
		FileStates:                     toDBAppliedFileStateRows(gameID, fileStates, ""),
	})
	if err != nil {
		return fmt.Errorf("save applied profile state: %w", err)
	}

	return nil
}

func toDBAppliedFileStateRows(gameID int64, states []appliedstate.PersistedFileState, appliedAt string) []dbtypes.AppliedFileStateRow {
	rows := make([]dbtypes.AppliedFileStateRow, len(states))
	for index, state := range states {
		lastAppliedAt := state.LastAppliedAt
		if lastAppliedAt == "" {
			lastAppliedAt = appliedAt
		}

		rows[index] = dbtypes.AppliedFileStateRow{
			GameID:             gameID,
			GameRelativePath:   state.GameRelativePath,
			ProfileID:          state.ProfileID,
			BaselineExists:     state.BaselineExists,
			BaselineSHA256:     state.BaselineSHA256,
			BaselineSizeBytes:  state.BaselineSizeBytes,
			BaselineBackupPath: state.BaselineBackupPath,
			AppliedExists:      state.AppliedExists,
			AppliedSHA256:      state.AppliedSHA256,
			AppliedSizeBytes:   state.AppliedSizeBytes,
			WinningSourceKind:  state.WinningSourceKind,
			WinningSourceID:    state.WinningSourceID,
			WinningModID:       state.WinningModID,
			WinningLoadOrder:   state.WinningLoadOrder,
			OutputKind:         state.OutputKind,
			UserDecision:       state.UserDecision,
			LastAppliedAt:      lastAppliedAt,
		}
	}

	return rows
}

func fromDBAppliedFileStateRows(rows []dbtypes.AppliedFileStateRow) []appliedstate.PersistedFileState {
	states := make([]appliedstate.PersistedFileState, len(rows))
	for index, row := range rows {
		states[index] = appliedstate.PersistedFileState{
			GameID:             row.GameID,
			GameRelativePath:   row.GameRelativePath,
			ProfileID:          row.ProfileID,
			BaselineExists:     row.BaselineExists,
			BaselineSHA256:     row.BaselineSHA256,
			BaselineSizeBytes:  row.BaselineSizeBytes,
			BaselineBackupPath: row.BaselineBackupPath,
			AppliedExists:      row.AppliedExists,
			AppliedSHA256:      row.AppliedSHA256,
			AppliedSizeBytes:   row.AppliedSizeBytes,
			WinningSourceKind:  row.WinningSourceKind,
			WinningSourceID:    row.WinningSourceID,
			WinningModID:       row.WinningModID,
			WinningLoadOrder:   row.WinningLoadOrder,
			OutputKind:         row.OutputKind,
			UserDecision:       row.UserDecision,
			LastAppliedAt:      row.LastAppliedAt,
		}
	}

	return states
}

func withAppliedAt(states []appliedstate.PersistedFileState, appliedAt string) []appliedstate.PersistedFileState {
	if appliedAt == "" {
		return states
	}

	copied := make([]appliedstate.PersistedFileState, len(states))
	copy(copied, states)
	for index := range copied {
		if copied[index].LastAppliedAt == "" {
			copied[index].LastAppliedAt = appliedAt
		}
	}

	return copied
}
