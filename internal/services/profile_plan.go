package services

import (
	"context"
	"errors"
	"fmt"

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

func (s *ProfileService) ConfirmProfileOperationPlan(profileID int64, plan operationplan.OperationPlan) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("confirm profile operation plan: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return errors.New("storage is not configured")
	}
	if profileID <= 0 {
		return fmt.Errorf("profile ID must be positive")
	}
	if !plan.CanApply {
		return errors.New("operation plan has blocking issues")
	}

	return errors.New("apply execution is reserved for Epic 8")
}
