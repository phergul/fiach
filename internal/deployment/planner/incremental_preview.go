package planner

import (
	"fmt"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
)

func PlanIncrementalPreview(
	state deployment.DesiredState,
	appliedStates []appliedstate.PersistedFileState,
	driftResults []drift.Result,
	gameInstallPath string,
) (plan DeploymentPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("plan incremental preview: %w", err)
		}
	}()

	plan, err = PlanIncremental(state, appliedStates, driftResults, gameInstallPath)
	if err != nil {
		return DeploymentPlan{}, err
	}

	plan.PreviewOnly = true
	return plan, nil
}
