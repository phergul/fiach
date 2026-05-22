package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/appliedstate"
	"github.com/phergul/mod-manager/internal/applyplan"
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/restoreplan"
	"github.com/phergul/mod-manager/internal/storage"
)

func (s *ProfileService) BuildProfileOperationPlan(ctx context.Context, profileID int64) (plan operationplan.OperationPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build profile operation plan: %w", err)
		}
	}()

	resolved, err := operationplan.ResolveProfilePlan(ctx, s.store, profileID)
	if err != nil {
		return operationplan.OperationPlan{}, err
	}

	return operationplan.BuildOperationPlan(resolved)
}

func (s *ProfileService) ApplyProfileOperationPlan(ctx context.Context, profileID int64, plan operationplan.OperationPlan) (result operationplan.ApplyOperationPlanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply profile operation plan: %w", err)
		}
	}()

	if profileID <= 0 {
		return operationplan.ApplyOperationPlanResult{}, fmt.Errorf("profile ID must be positive")
	}
	if !plan.CanApply {
		return operationplan.ApplyOperationPlanResult{}, errors.New("operation plan has blocking issues")
	}

	profile, found, err := s.store.GetProfile(ctx, profileID)
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}
	if !found {
		return operationplan.ApplyOperationPlanResult{}, fmt.Errorf("profile %d was not found", profileID)
	}

	game, err := s.store.GetStoredGame(ctx, profile.GameID)
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, profile.GameID, "")
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}

	result, err = applyplan.Execute(plan, applyplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}
	if !result.Success {
		return result, nil
	}

	if err := s.saveAppliedProfileState(ctx, game.ID, profileID, plan, result.Manifest); err != nil {
		return result, err
	}

	return result, nil
}

func (s *ProfileService) RestoreVanillaState(ctx context.Context, gameID int64) (result restoreplan.RestoreResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("restore vanilla state: %w", err)
		}
	}()

	if gameID <= 0 {
		return restoreplan.RestoreResult{}, errors.New("game ID must be positive")
	}

	game, err := s.store.GetStoredGame(ctx, gameID)
	if err != nil {
		return restoreplan.RestoreResult{}, err
	}
	state, found, err := s.store.GetAppliedProfileState(ctx, gameID)
	if err != nil {
		return restoreplan.RestoreResult{}, err
	}
	if !found {
		return restoreplan.RestoreResult{}, fmt.Errorf("no applied profile state found for game %d", gameID)
	}

	manifest, err := appliedstate.DecodeManifest(state.ManifestJSON)
	if err != nil {
		return restoreplan.RestoreResult{}, err
	}
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(ctx, gameID, "")
	if err != nil {
		return restoreplan.RestoreResult{}, err
	}

	result, err = restoreplan.Execute(manifest, restoreplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
	if err != nil {
		return restoreplan.RestoreResult{}, err
	}
	if !result.Success {
		return result, nil
	}

	if err := s.store.DeleteAppliedProfileState(ctx, gameID); err != nil {
		return result, err
	}

	return result, nil
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

	_, err = s.store.SaveAppliedProfileState(ctx, storage.SaveAppliedProfileStateInput{
		GameID:              gameID,
		ProfileID:           profileID,
		ManifestJSON:        manifestJSON,
		ProfileSnapshotJSON: snapshot.JSON,
		ProfileSnapshotHash: snapshot.Hash,
	})
	if err != nil {
		return fmt.Errorf("save applied profile state: %w", err)
	}

	return nil
}
