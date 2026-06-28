package inspect

import (
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
)

func SelectDefaultPair(
	planMode string,
	comparison review.StateComparison,
) ComparePair {
	isIncremental := planMode == string(planner.PlanModeIncremental)
	hasDrift := isIncremental && !comparison.AppliedMatchesCurrent

	if hasDrift {
		return ComparePair{
			Left:  StateApplied,
			Right: StateCurrent,
		}
	}

	return ComparePair{
		Left:  StateCurrent,
		Right: StateDesired,
	}
}

func stateLabel(kind StateKind) string {
	switch kind {
	case StateBaseline:
		return "Baseline"
	case StateApplied:
		return "Last applied"
	case StateCurrent:
		return "Current"
	case StateDesired:
		return "Desired"
	default:
		return string(kind)
	}
}

func buildComparisonFromPathPlan(pathPlan planner.PathPlan) review.StateComparison {
	return review.StateComparison{
		AppliedMatchesCurrent: planner.SnapshotsMatch(pathPlan.Applied, pathPlan.Current),
		AppliedMatchesDesired: planner.SnapshotsMatch(pathPlan.Applied, pathPlan.Desired),
		CurrentMatchesDesired: planner.SnapshotsMatch(pathPlan.Current, pathPlan.Desired),
	}
}
