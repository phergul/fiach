package execute

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/deployment/planner"
)

func MergeAppliedFileStates(
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	existingStates []appliedstate.PersistedFileState,
	profileID int64,
) ([]appliedstate.PersistedFileState, error) {
	existingByPath := map[string]appliedstate.PersistedFileState{}
	for _, state := range existingStates {
		key := deployment.CanonicalGameRelativePath(state.GameRelativePath)
		existingByPath[key] = state
	}

	updatedByPath := map[string]appliedstate.PersistedFileState{}
	for key, state := range existingByPath {
		updatedByPath[key] = state
	}

	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]

		switch pathPlan.PlannedAction {
		case planner.ReapplyNoOp:
			if planner.ShouldRemoveFromAppliedState(pathPlan, desired.Files, canonicalPath) {
				delete(updatedByPath, canonicalPath)
			}
			continue
		case planner.ReapplyDelete, planner.ReapplyRestoreBaseline, planner.ReapplyBackupThenDelete, planner.ReapplyBackupThenRestore:
			delete(updatedByPath, canonicalPath)
		case planner.ReapplyCreate, planner.ReapplyReplace, planner.ReapplyRepair, planner.ReapplyBackupThenReplace:
			desiredFile, found := desired.Files[canonicalPath]
			if !found {
				return nil, fmt.Errorf("desired state missing for path %q", pathPlan.GameRelativePath)
			}

			state, err := fileStateAfterApply(desiredFile, profileID, existingByPath[canonicalPath], pathPlan.PlannedAction, nil)
			if err != nil {
				return nil, err
			}
			updatedByPath[canonicalPath] = state
		}
	}

	return sortedFileStates(updatedByPath), nil
}

func BuildInitialAppliedFileStates(
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	outcome FirstApplyOutcome,
	profileID int64,
) ([]appliedstate.PersistedFileState, error) {
	states := make([]appliedstate.PersistedFileState, 0, len(plan.Paths))

	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]

		switch pathPlan.PlannedAction {
		case planner.ReapplyCreate, planner.ReapplyReplace:
		default:
			continue
		}

		desiredFile, found := desired.Files[canonicalPath]
		if !found {
			return nil, fmt.Errorf("desired state missing for path %q", pathPlan.GameRelativePath)
		}

		var baseline *BaselineBackup
		if backup, found := outcome.BaselineBackups[canonicalPath]; found {
			baseline = &backup
		}

		state, err := fileStateAfterApply(desiredFile, profileID, appliedstate.PersistedFileState{}, pathPlan.PlannedAction, baseline)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

func fileStateAfterApply(
	desiredFile deployment.DesiredFile,
	profileID int64,
	existing appliedstate.PersistedFileState,
	action planner.ReapplyAction,
	baseline *BaselineBackup,
) (appliedstate.PersistedFileState, error) {
	appliedSHA256 := desiredFile.SHA256
	appliedSizeBytes := desiredFile.SizeBytes
	winningSourceKind := appliedstate.WinningSourceKindMod

	state := appliedstate.PersistedFileState{
		GameRelativePath: desiredFile.GameRelativePath,
		ProfileID:        profileID,
		AppliedExists:    true,
		AppliedSHA256:    &appliedSHA256,
		AppliedSizeBytes: &appliedSizeBytes,
		OutputKind:       appliedstate.OutputKindCopied,
		UserDecision:     preservedUserDecision(existing.UserDecision, action),
	}

	if desiredFile.Winner.ModID != nil {
		winningModID := *desiredFile.Winner.ModID
		winningSourceID := strconv.FormatInt(winningModID, 10)
		winningLoadOrder := desiredFile.Winner.LoadOrder
		state.WinningModID = &winningModID
		state.WinningSourceKind = &winningSourceKind
		state.WinningSourceID = &winningSourceID
		state.WinningLoadOrder = &winningLoadOrder
	}

	if baseline != nil {
		state.BaselineExists = true
		state.BaselineSHA256 = &baseline.SHA256
		state.BaselineSizeBytes = &baseline.SizeBytes
		state.BaselineBackupPath = &baseline.BackupPath
		return state, nil
	}

	if action == planner.ReapplyCreate || !existing.BaselineExists {
		state.BaselineExists = false
		return state, nil
	}

	state.BaselineExists = existing.BaselineExists
	state.BaselineSHA256 = existing.BaselineSHA256
	state.BaselineSizeBytes = existing.BaselineSizeBytes
	state.BaselineBackupPath = existing.BaselineBackupPath

	return state, nil
}

func preservedUserDecision(existing *string, action planner.ReapplyAction) *string {
	if existing == nil {
		return nil
	}

	if action == planner.ReapplyBackupThenReplace ||
		action == planner.ReapplyBackupThenDelete ||
		action == planner.ReapplyBackupThenRestore {
		if *existing == drift.UserDecisionBackupAndApply {
			return nil
		}
	}

	return existing
}

func sortedFileStates(states map[string]appliedstate.PersistedFileState) []appliedstate.PersistedFileState {
	merged := make([]appliedstate.PersistedFileState, 0, len(states))
	for _, state := range states {
		merged = append(merged, state)
	}

	sort.Slice(merged, func(i int, j int) bool {
		return deployment.CanonicalGameRelativePath(merged[i].GameRelativePath) <
			deployment.CanonicalGameRelativePath(merged[j].GameRelativePath)
	})

	return merged
}
