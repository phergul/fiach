package review

import (
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
)

type StateComparison struct {
	AppliedMatchesCurrent bool
	AppliedMatchesDesired bool
	CurrentMatchesDesired bool
}

func snapshotsMatch(left planner.FileStateSnapshot, right planner.FileStateSnapshot) bool {
	if left.Exists != right.Exists {
		return false
	}

	if !left.Exists {
		return true
	}

	return left.SHA256 == right.SHA256
}

func buildStateComparison(
	applied planner.FileStateSnapshot,
	current planner.FileStateSnapshot,
	desired planner.FileStateSnapshot,
) StateComparison {
	return StateComparison{
		AppliedMatchesCurrent: snapshotsMatch(applied, current),
		AppliedMatchesDesired: snapshotsMatch(applied, desired),
		CurrentMatchesDesired: snapshotsMatch(current, desired),
	}
}

func buildDriftExplanation(
	driftKind deployment.DriftKind,
	comparison StateComparison,
	fileStatus deployment.FileStatus,
) string {
	switch driftKind {
	case deployment.DriftModified:
		return "This file was modified on disk since the last apply."
	case deployment.DriftMissing:
		return "This file is missing from disk but was present after the last apply."
	case deployment.DriftExternal:
		return "This file was kept as an external edit and will not be overwritten automatically."
	}

	if fileStatus == deployment.FileStatusDrifted {
		return "This file changed on disk since the last apply."
	}

	if !comparison.AppliedMatchesDesired && fileStatus != deployment.FileStatusAdded {
		return "This file has different content than what was previously applied."
	}

	return ""
}
