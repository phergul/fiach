package planner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
)

func PlanIncremental(
	state deployment.DesiredState,
	appliedStates []appliedstate.PersistedFileState,
	driftResults []drift.Result,
	gameInstallPath string,
) (plan DeploymentPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("plan incremental: %w", err)
		}
	}()

	firstApplyPlan, err := PlanFirstApply(state, gameInstallPath)
	if err != nil {
		return DeploymentPlan{}, err
	}

	appliedByPath := map[string]appliedstate.PersistedFileState{}
	for _, appliedState := range appliedStates {
		key := deployment.CanonicalGameRelativePath(appliedState.GameRelativePath)
		appliedByPath[key] = appliedState
	}

	driftByPath := map[string]drift.Result{}
	for _, driftResult := range driftResults {
		key := deployment.CanonicalGameRelativePath(driftResult.GameRelativePath)
		driftByPath[key] = driftResult
	}

	plan = DeploymentPlan{
		Mode:   PlanModeIncremental,
		Paths:  map[string]PathPlan{},
		Issues: append([]deployment.PlanIssue(nil), firstApplyPlan.Issues...),
	}

	gameInstallPath = filepath.Clean(gameInstallPath)
	canonicalPaths := unionCanonicalPaths(state.Files, appliedStates)

	for _, canonicalPath := range canonicalPaths {
		desiredFile, hasDesired := state.Files[canonicalPath]
		appliedState, hasApplied := appliedByPath[canonicalPath]

		pathPlan, planErr := planIncrementalPath(
			canonicalPath,
			desiredFile,
			hasDesired,
			appliedState,
			hasApplied,
			firstApplyPlan,
			driftByPath,
			gameInstallPath,
		)
		if planErr != nil {
			return DeploymentPlan{}, planErr
		}

		plan.Paths[canonicalPath] = pathPlan
	}

	return plan, nil
}

func planIncrementalPath(
	canonicalPath string,
	desiredFile deployment.DesiredFile,
	hasDesired bool,
	appliedState appliedstate.PersistedFileState,
	hasApplied bool,
	firstApplyPlan DeploymentPlan,
	driftByPath map[string]drift.Result,
	gameInstallPath string,
) (PathPlan, error) {
	pathPlan := PathPlan{
		DriftKind: deployment.DriftNone,
	}

	if hasDesired {
		pathPlan = firstApplyPlan.Paths[canonicalPath]
		pathPlan.GameRelativePath = desiredFile.GameRelativePath
	} else if hasApplied {
		pathPlan.GameRelativePath = appliedState.GameRelativePath
	}

	if hasApplied {
		pathPlan.Baseline = baselineSnapshot(appliedState, true)
		pathPlan.Applied = appliedSnapshot(appliedState, true)
		pathPlan.BaselineBackupPath = baselineBackupPath(appliedState, true)
		pathPlan.LastAppliedAt = appliedState.LastAppliedAt
	}

	gameRelativePath := pathPlan.GameRelativePath
	if gameRelativePath == "" && hasApplied {
		gameRelativePath = appliedState.GameRelativePath
		pathPlan.GameRelativePath = gameRelativePath
	}

	if driftResult, found := driftByPath[canonicalPath]; found {
		pathPlan.DriftKind = driftResult.Kind
		pathPlan.Current = currentSnapshotFromDrift(driftResult)
	} else if hasApplied || hasDesired {
		current, currentErr := readCurrentSnapshot(gameInstallPath, gameRelativePath)
		if currentErr != nil {
			return PathPlan{}, currentErr
		}
		pathPlan.Current = current
	}

	if !hasDesired && hasApplied {
		classifyProfileRemoval(&pathPlan, appliedState)
		return pathPlan, nil
	}

	if hasDesired && !hasApplied {
		return pathPlan, nil
	}

	if hasDesired && hasApplied {
		classifyIncrementalDelta(&pathPlan, desiredFile, appliedState)
	}

	return pathPlan, nil
}

func classifyProfileRemoval(pathPlan *PathPlan, appliedState appliedstate.PersistedFileState) {
	pathPlan.Desired = FileStateSnapshot{
		Exists: false,
		Label:  "Desired content",
	}

	if !appliedState.BaselineExists {
		pathPlan.PlannedAction = ReapplyDelete
		pathPlan.FileStatus = deployment.FileStatusDeleted
		pathPlan.RiskLevel = deployment.RiskInfo
		return
	}

	if !hasBaselineBackup(appliedState) {
		pathPlan.PlannedAction = ReapplyBlock
		pathPlan.FileStatus = deployment.FileStatusBlocked
		pathPlan.RiskLevel = deployment.RiskError
		return
	}

	pathPlan.PlannedAction = ReapplyRestoreBaseline
	pathPlan.FileStatus = deployment.FileStatusRestored
	pathPlan.RiskLevel = deployment.RiskInfo
}

func classifyIncrementalDelta(
	pathPlan *PathPlan,
	desiredFile deployment.DesiredFile,
	appliedState appliedstate.PersistedFileState,
) {
	if pathPlan.PlannedAction == ReapplyBlock {
		return
	}

	wantHash := desiredFile.SHA256
	wasHash := appliedHash(appliedState)
	diskHash := snapshotHash(pathPlan.Current)

	if wantHash != "" && wasHash != "" && diskHash != "" &&
		strings.EqualFold(wantHash, wasHash) &&
		strings.EqualFold(wasHash, diskHash) {
		pathPlan.PlannedAction = ReapplyNoOp
		pathPlan.FileStatus = deployment.FileStatusUnchanged
		pathPlan.RiskLevel = deployment.RiskNone
		pathPlan.DriftKind = deployment.DriftNone
		return
	}

	if wantHash != "" && wasHash != "" &&
		!strings.EqualFold(wantHash, wasHash) &&
		strings.EqualFold(diskHash, wasHash) {
		pathPlan.PlannedAction = ReapplyReplace
		pathPlan.FileStatus = deployment.FileStatusReplaced
		pathPlan.RiskLevel = desiredFile.RiskLevel
		pathPlan.DriftKind = deployment.DriftNone
		return
	}

	if wasHash != "" && !diskMatchesApplied(wasHash, pathPlan.Current) {
		if isKeepExternalDecision(appliedState.UserDecision) {
			pathPlan.PlannedAction = ReapplyNoOp
			pathPlan.FileStatus = deployment.FileStatusExternal
			pathPlan.RiskLevel = deployment.RiskInfo
			pathPlan.DriftKind = deployment.DriftExternal
			return
		}

		if wantHash != "" && diskHash != "" &&
			strings.EqualFold(wantHash, diskHash) &&
			singleModWriter(desiredFile.Writers) {
			pathPlan.PlannedAction = ReapplyRepair
			pathPlan.FileStatus = deployment.FileStatusDrifted
			pathPlan.RiskLevel = deployment.RiskInfo
			return
		}

		pathPlan.PlannedAction = ReapplyRequireDecision
		pathPlan.FileStatus = deployment.FileStatusDrifted
		pathPlan.RiskLevel = deployment.RiskError
		if pathPlan.DriftKind == deployment.DriftNone {
			pathPlan.DriftKind = deployment.DriftModified
		}
	}
}

func unionCanonicalPaths(desired map[string]deployment.DesiredFile, applied []appliedstate.PersistedFileState) []string {
	pathSet := map[string]struct{}{}
	for canonicalPath := range desired {
		pathSet[canonicalPath] = struct{}{}
	}
	for _, appliedState := range applied {
		pathSet[deployment.CanonicalGameRelativePath(appliedState.GameRelativePath)] = struct{}{}
	}

	paths := make([]string, 0, len(pathSet))
	for canonicalPath := range pathSet {
		paths = append(paths, canonicalPath)
	}
	sort.Strings(paths)
	return paths
}

func hasBaselineBackup(state appliedstate.PersistedFileState) bool {
	return state.BaselineBackupPath != nil && strings.TrimSpace(*state.BaselineBackupPath) != ""
}

func appliedHash(state appliedstate.PersistedFileState) string {
	if !state.AppliedExists || state.AppliedSHA256 == nil {
		return ""
	}
	return *state.AppliedSHA256
}

func snapshotHash(snapshot FileStateSnapshot) string {
	if !snapshot.Exists {
		return ""
	}
	return snapshot.SHA256
}

func diskMatchesApplied(appliedHash string, current FileStateSnapshot) bool {
	if appliedHash == "" {
		return false
	}
	if !current.Exists {
		return false
	}
	return strings.EqualFold(appliedHash, current.SHA256)
}

func singleModWriter(writers []deployment.WriterEntry) bool {
	count := 0
	for _, writer := range writers {
		if writer.SourceKind == deployment.SourceKindMod {
			count++
		}
	}
	return count == 1
}

func isKeepExternalDecision(decision *string) bool {
	if decision == nil {
		return false
	}
	return *decision == drift.UserDecisionKeepExternal
}
