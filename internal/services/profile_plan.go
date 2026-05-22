package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/appliedstate"
	"github.com/phergul/mod-manager/internal/applyplan"
	"github.com/phergul/mod-manager/internal/operationplan"
	"github.com/phergul/mod-manager/internal/storage"
)

func (s *ProfileService) BuildProfileOperationPlan(profileID int64) (plan operationplan.OperationPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build profile operation plan: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return operationplan.OperationPlan{}, errors.New("storage is not configured")
	}

	resolved, err := operationplan.ResolveProfilePlan(context.Background(), s.store, profileID)
	if err != nil {
		return operationplan.OperationPlan{}, err
	}

	return operationplan.BuildOperationPlan(resolved)
}

func (s *ProfileService) ApplyProfileOperationPlan(profileID int64, plan operationplan.OperationPlan) (result operationplan.ApplyOperationPlanResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("apply profile operation plan: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return operationplan.ApplyOperationPlanResult{}, errors.New("storage is not configured")
	}
	if profileID <= 0 {
		return operationplan.ApplyOperationPlanResult{}, fmt.Errorf("profile ID must be positive")
	}
	if !plan.CanApply {
		return operationplan.ApplyOperationPlanResult{}, errors.New("operation plan has blocking issues")
	}

	profile, found, err := s.store.GetProfile(context.Background(), profileID)
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}
	if !found {
		return operationplan.ApplyOperationPlanResult{}, fmt.Errorf("profile %d was not found", profileID)
	}

	game, err := s.store.GetStoredGame(context.Background(), profile.GameID)
	if err != nil {
		return operationplan.ApplyOperationPlanResult{}, err
	}
	gameModStoragePath, err := s.store.ResolveGameModStoragePath(context.Background(), profile.GameID, "")
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

	if err := s.saveAppliedProfileState(context.Background(), game.ID, profileID, plan, result.Manifest); err != nil {
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
