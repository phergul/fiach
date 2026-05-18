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
