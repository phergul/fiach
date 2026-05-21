package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/phergul/mod-manager/internal/applyplan"
	"github.com/phergul/mod-manager/internal/operationplan"
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

	return applyplan.Execute(plan, applyplan.Context{
		GameInstallPath:    game.InstallPath,
		GameModStoragePath: gameModStoragePath,
	})
}
