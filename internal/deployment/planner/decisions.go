package planner

import (
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
)

func applyPersistedUserDecision(pathPlan *PathPlan, appliedState appliedstate.PersistedFileState) bool {
	pathPlan.UserDecision = appliedState.UserDecision

	if drift.IsSkippedDecision(appliedState.UserDecision) {
		pathPlan.PlannedAction = ReapplyNoOp
		pathPlan.FileStatus = deployment.FileStatusSkipped
		pathPlan.RiskLevel = deployment.RiskInfo
		return true
	}

	if drift.IsKeepExternalDecision(appliedState.UserDecision) {
		pathPlan.PlannedAction = ReapplyNoOp
		pathPlan.FileStatus = deployment.FileStatusExternal
		pathPlan.RiskLevel = deployment.RiskInfo
		if pathPlan.DriftKind == deployment.DriftNone {
			pathPlan.DriftKind = deployment.DriftExternal
		}
		return true
	}

	return false
}

func applyBackupAndApplyDecision(
	pathPlan *PathPlan,
	desiredFile deployment.DesiredFile,
	appliedState appliedstate.PersistedFileState,
	hasDesired bool,
) bool {
	if !drift.IsBackupAndApplyDecision(appliedState.UserDecision) {
		return false
	}

	pathPlan.UserDecision = appliedState.UserDecision

	if !hasDesired {
		if !appliedState.BaselineExists {
			pathPlan.PlannedAction = ReapplyBackupThenDelete
			pathPlan.FileStatus = deployment.FileStatusDeleted
			pathPlan.RiskLevel = deployment.RiskInfo
			pathPlan.RequiresDriftArchive = pathHasRemovalDrift(*pathPlan)
			return true
		}

		if !hasBaselineBackup(appliedState) {
			pathPlan.PlannedAction = ReapplyBlock
			pathPlan.FileStatus = deployment.FileStatusBlocked
			pathPlan.RiskLevel = deployment.RiskError
			return true
		}

		pathPlan.PlannedAction = ReapplyBackupThenRestore
		pathPlan.FileStatus = deployment.FileStatusRestored
		pathPlan.RiskLevel = deployment.RiskInfo
		pathPlan.RequiresDriftArchive = pathHasRemovalDrift(*pathPlan)
		return true
	}

	wantHash := desiredFile.SHA256
	diskHash := SnapshotHash(pathPlan.Current)

	if wantHash != "" && diskHash != "" && strings.EqualFold(wantHash, diskHash) {
		pathPlan.PlannedAction = ReapplyRepair
		pathPlan.FileStatus = deployment.FileStatusDrifted
		pathPlan.RiskLevel = deployment.RiskInfo
		return true
	}

	pathPlan.PlannedAction = ReapplyBackupThenReplace
	if pathPlan.FileStatus == "" || pathPlan.FileStatus == deployment.FileStatusDrifted {
		pathPlan.FileStatus = deployment.FileStatusReplaced
	}
	pathPlan.RiskLevel = desiredFile.RiskLevel
	pathPlan.RequiresDriftArchive = pathPlan.Current.Exists
	return true
}

func pathHasRemovalDrift(pathPlan PathPlan) bool {
	if pathPlan.DriftKind == deployment.DriftModified || pathPlan.DriftKind == deployment.DriftMissing {
		return true
	}

	return pathPlan.Applied.Exists && !SnapshotsMatch(pathPlan.Applied, pathPlan.Current)
}

func pathRequiresDriftDecision(pathPlan PathPlan) bool {
	if pathPlan.DriftKind == deployment.DriftModified || pathPlan.DriftKind == deployment.DriftMissing {
		return true
	}

	if !pathPlan.Applied.Exists {
		return false
	}

	return !SnapshotsMatch(pathPlan.Applied, pathPlan.Current)
}

func ShouldRemoveFromAppliedState(
	pathPlan PathPlan,
	desired map[string]deployment.DesiredFile,
	canonicalPath string,
) bool {
	if _, hasDesired := desired[canonicalPath]; hasDesired {
		return false
	}

	return pathPlan.PlannedAction == ReapplyNoOp &&
		(pathPlan.FileStatus == deployment.FileStatusExternal || pathPlan.FileStatus == deployment.FileStatusSkipped)
}
