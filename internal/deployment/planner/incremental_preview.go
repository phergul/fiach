package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/drift"
	"github.com/phergul/fiach/internal/fileops"
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
		Mode:        PlanModeIncremental,
		Paths:       map[string]PathPlan{},
		Issues:      append([]deployment.PlanIssue(nil), firstApplyPlan.Issues...),
		PreviewOnly: true,
	}

	gameInstallPath = filepath.Clean(gameInstallPath)
	canonicalPaths := sortedCanonicalPaths(state.Files)

	for _, canonicalPath := range canonicalPaths {
		file := state.Files[canonicalPath]
		pathPlan := firstApplyPlan.Paths[canonicalPath]
		appliedState, hasApplied := appliedByPath[canonicalPath]

		pathPlan.DriftKind = deployment.DriftNone
		pathPlan.Baseline = baselineSnapshot(appliedState, hasApplied)
		pathPlan.Applied = appliedSnapshot(appliedState, hasApplied)
		pathPlan.BaselineBackupPath = baselineBackupPath(appliedState, hasApplied)

		if driftResult, found := driftByPath[canonicalPath]; found {
			pathPlan.DriftKind = driftResult.Kind
			pathPlan.Current = currentSnapshotFromDrift(driftResult)
			applyDriftPlanning(&pathPlan)
		} else if hasApplied {
			current, currentErr := readCurrentSnapshot(gameInstallPath, file.GameRelativePath)
			if currentErr != nil {
				return DeploymentPlan{}, currentErr
			}
			pathPlan.Current = current
		}

		plan.Paths[canonicalPath] = pathPlan
	}

	return plan, nil
}

func applyDriftPlanning(pathPlan *PathPlan) {
	switch pathPlan.DriftKind {
	case deployment.DriftExternal:
		pathPlan.FileStatus = deployment.FileStatusExternal
		pathPlan.PlannedAction = ReapplyNoOp
		pathPlan.RiskLevel = deployment.RiskInfo
	case deployment.DriftMissing, deployment.DriftModified:
		pathPlan.FileStatus = deployment.FileStatusDrifted
		pathPlan.PlannedAction = ReapplyRequireDecision
		pathPlan.RiskLevel = deployment.RiskError
	case deployment.DriftNone:
		if pathPlan.Applied.Exists {
			pathPlan.FileStatus = deployment.FileStatusUnchanged
			pathPlan.PlannedAction = ReapplyNoOp
			pathPlan.RiskLevel = deployment.RiskNone
		}
	}
}

func baselineSnapshot(state appliedstate.PersistedFileState, hasApplied bool) FileStateSnapshot {
	if !hasApplied || !state.BaselineExists {
		return FileStateSnapshot{Exists: false}
	}

	snapshot := FileStateSnapshot{
		Exists: true,
		Label:  "Original game install",
	}
	if state.BaselineSHA256 != nil {
		snapshot.SHA256 = *state.BaselineSHA256
	}
	if state.BaselineSizeBytes != nil {
		snapshot.SizeBytes = *state.BaselineSizeBytes
	}

	return snapshot
}

func appliedSnapshot(state appliedstate.PersistedFileState, hasApplied bool) FileStateSnapshot {
	if !hasApplied || !state.AppliedExists {
		return FileStateSnapshot{Exists: false}
	}

	snapshot := FileStateSnapshot{
		Exists: true,
		Label:  "Last Fiach apply",
	}
	if state.AppliedSHA256 != nil {
		snapshot.SHA256 = *state.AppliedSHA256
	}
	if state.AppliedSizeBytes != nil {
		snapshot.SizeBytes = *state.AppliedSizeBytes
	}

	return snapshot
}

func baselineBackupPath(state appliedstate.PersistedFileState, hasApplied bool) string {
	if !hasApplied || state.BaselineBackupPath == nil {
		return ""
	}

	return *state.BaselineBackupPath
}

func currentSnapshotFromDrift(result drift.Result) FileStateSnapshot {
	if !result.CurrentExists {
		return FileStateSnapshot{
			Exists: false,
			Label:  "Current game install",
		}
	}

	return FileStateSnapshot{
		Exists:    true,
		SHA256:    result.CurrentSHA256,
		SizeBytes: result.CurrentSizeBytes,
		Label:     "Current game install",
	}
}

func readCurrentSnapshot(gameInstallPath string, gameRelativePath string) (FileStateSnapshot, error) {
	targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(gameRelativePath))
	hash, size, err := fileops.FileIntegrity(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return FileStateSnapshot{
			Exists: false,
			Label:  "Current game install",
		}, nil
	}
	if err != nil {
		return FileStateSnapshot{}, fmt.Errorf("hash current file %q: %w", gameRelativePath, err)
	}

	return FileStateSnapshot{
		Exists:    true,
		SHA256:    hash,
		SizeBytes: size,
		Label:     "Current game install",
	}, nil
}
