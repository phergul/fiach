package execute

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
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
			continue
		case planner.ReapplyDelete, planner.ReapplyRestoreBaseline:
			delete(updatedByPath, canonicalPath)
		case planner.ReapplyCreate, planner.ReapplyReplace, planner.ReapplyRepair:
			desiredFile, found := desired.Files[canonicalPath]
			if !found {
				return nil, fmt.Errorf("desired state missing for path %q", pathPlan.GameRelativePath)
			}

			state, err := fileStateAfterApply(desiredFile, profileID, existingByPath[canonicalPath], pathPlan.PlannedAction)
			if err != nil {
				return nil, err
			}
			updatedByPath[canonicalPath] = state
		}
	}

	merged := make([]appliedstate.PersistedFileState, 0, len(updatedByPath))
	for _, state := range updatedByPath {
		merged = append(merged, state)
	}

	sort.Slice(merged, func(i int, j int) bool {
		return deployment.CanonicalGameRelativePath(merged[i].GameRelativePath) <
			deployment.CanonicalGameRelativePath(merged[j].GameRelativePath)
	})

	return merged, nil
}

func fileStateAfterApply(
	desiredFile deployment.DesiredFile,
	profileID int64,
	existing appliedstate.PersistedFileState,
	action planner.ReapplyAction,
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
		UserDecision:     existing.UserDecision,
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
