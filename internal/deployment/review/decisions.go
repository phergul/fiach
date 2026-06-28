package review

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func BuildAvailableDriftActions(pathPlan planner.PathPlan, hasDesired bool) []string {
	return buildAvailableDriftActions(pathPlan, hasDesired)
}

func buildAvailableDriftActions(pathPlan planner.PathPlan, hasDesired bool) []string {
	if persistedDecision := persistedDecisionValue(pathPlan.UserDecision); persistedDecision != "" {
		if pathPlan.PlannedAction == planner.ReapplyRequireDecision {
			return availableActionsForUnresolvedDrift(pathPlan, hasDesired)
		}

		return []string{drift.UserDecisionClear}
	}

	if pathPlan.PlannedAction != planner.ReapplyRequireDecision {
		return nil
	}

	return availableActionsForUnresolvedDrift(pathPlan, hasDesired)
}

func availableActionsForUnresolvedDrift(pathPlan planner.PathPlan, hasDesired bool) []string {
	switch pathPlan.DriftKind {
	case deployment.DriftMissing:
		return []string{
			drift.UserDecisionSkipped,
			drift.UserDecisionBackupAndApply,
		}
	default:
		if !hasDesired {
			return []string{
				drift.UserDecisionBackupAndApply,
				drift.UserDecisionKeepExternal,
				drift.UserDecisionSkipped,
			}
		}

		return []string{
			drift.UserDecisionBackupAndApply,
			drift.UserDecisionKeepExternal,
			drift.UserDecisionSkipped,
		}
	}
}

func persistedDecisionValue(decision *string) string {
	if decision == nil {
		return ""
	}

	value := *decision
	if drift.IsPersistedDecision(value) {
		return value
	}

	return ""
}

func buildUserDecisionLabel(pathPlan planner.PathPlan) string {
	return drift.DecisionLabel(persistedDecisionValue(pathPlan.UserDecision))
}
