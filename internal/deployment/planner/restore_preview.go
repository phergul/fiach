package planner

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/deployment"
)

func PlanRestorePreview(
	appliedStates []appliedstate.PersistedFileState,
	gameInstallPath string,
) (plan DeploymentPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("plan restore preview: %w", err)
		}
	}()

	gameInstallPath = filepath.Clean(gameInstallPath)
	plan = DeploymentPlan{
		Mode:   PlanModeRestorePreview,
		Paths:  map[string]PathPlan{},
		Issues: []deployment.PlanIssue{},
	}

	canonicalPaths := make([]string, 0, len(appliedStates))
	for _, appliedState := range appliedStates {
		canonicalPaths = append(canonicalPaths, deployment.CanonicalGameRelativePath(appliedState.GameRelativePath))
	}
	sort.Strings(canonicalPaths)

	for _, canonicalPath := range canonicalPaths {
		appliedState := appliedStateByCanonicalPath(appliedStates, canonicalPath)
		pathPlan, planErr := planRestorePath(appliedState, gameInstallPath)
		if planErr != nil {
			return DeploymentPlan{}, planErr
		}
		plan.Paths[canonicalPath] = pathPlan

		if pathPlan.PlannedAction == ReapplyBlock {
			targetPath := pathPlan.GameRelativePath
			plan.Issues = append(plan.Issues, deployment.PlanIssue{
				TargetPath: &targetPath,
				Kind:       deployment.PlanIssueMissingBaselineBackup,
				Message:    fmt.Sprintf("Baseline backup is missing for %q.", pathPlan.GameRelativePath),
				Severity:   deployment.PlanIssueSeverityError,
			})
		}
	}

	return plan, nil
}

func appliedStateByCanonicalPath(states []appliedstate.PersistedFileState, canonicalPath string) appliedstate.PersistedFileState {
	for _, state := range states {
		if deployment.CanonicalGameRelativePath(state.GameRelativePath) == canonicalPath {
			return state
		}
	}

	return appliedstate.PersistedFileState{}
}

func planRestorePath(appliedState appliedstate.PersistedFileState, gameInstallPath string) (PathPlan, error) {
	pathPlan := PathPlan{
		GameRelativePath:   appliedState.GameRelativePath,
		Baseline:           baselineSnapshot(appliedState, true),
		Applied:            appliedSnapshot(appliedState, true),
		BaselineBackupPath: baselineBackupPath(appliedState, true),
		LastAppliedAt:      appliedState.LastAppliedAt,
		UserDecision:       appliedState.UserDecision,
		Desired:            FileStateSnapshot{Exists: false, Label: "Vanilla content"},
		DriftKind:          deployment.DriftNone,
		RiskLevel:          deployment.RiskInfo,
	}

	current, err := readCurrentSnapshot(gameInstallPath, appliedState.GameRelativePath)
	if err != nil {
		return PathPlan{}, err
	}
	pathPlan.Current = current

	if !appliedState.BaselineExists {
		pathPlan.PlannedAction = ReapplyDelete
		pathPlan.FileStatus = deployment.FileStatusDeleted
		return pathPlan, nil
	}

	if !hasBaselineBackup(appliedState) {
		pathPlan.PlannedAction = ReapplyBlock
		pathPlan.FileStatus = deployment.FileStatusBlocked
		pathPlan.RiskLevel = deployment.RiskError
		return pathPlan, nil
	}

	pathPlan.PlannedAction = ReapplyRestoreBaseline
	pathPlan.FileStatus = deployment.FileStatusRestored

	return pathPlan, nil
}

func PreflightRestorePlan(
	plan DeploymentPlan,
	gameInstallPath string,
	gameModStoragePath string,
) map[string]error {
	failures := map[string]error{}

	for canonicalPath, pathPlan := range plan.Paths {
		switch pathPlan.PlannedAction {
		case ReapplyDelete:
			if err := preflightRestoreDelete(pathPlan, gameInstallPath); err != nil {
				failures[canonicalPath] = err
			}
		case ReapplyRestoreBaseline:
			if err := preflightRestoreBaseline(pathPlan, appliedStateFromPathPlan(pathPlan), gameInstallPath, gameModStoragePath); err != nil {
				failures[canonicalPath] = err
			}
		case ReapplyBlock, ReapplyRequireDecision:
			failures[canonicalPath] = fmt.Errorf("restore blocked for %q", pathPlan.GameRelativePath)
		}
	}

	return failures
}

func appliedStateFromPathPlan(pathPlan PathPlan) appliedstate.PersistedFileState {
	state := appliedstate.PersistedFileState{
		GameRelativePath: pathPlan.GameRelativePath,
		BaselineExists:   pathPlan.Baseline.Exists,
		AppliedExists:    pathPlan.Applied.Exists,
	}
	if pathPlan.Baseline.Exists {
		baselineSHA256 := pathPlan.Baseline.SHA256
		baselineSizeBytes := pathPlan.Baseline.SizeBytes
		state.BaselineSHA256 = &baselineSHA256
		state.BaselineSizeBytes = &baselineSizeBytes
	}
	if pathPlan.Applied.Exists {
		appliedSHA256 := pathPlan.Applied.SHA256
		appliedSizeBytes := pathPlan.Applied.SizeBytes
		state.AppliedSHA256 = &appliedSHA256
		state.AppliedSizeBytes = &appliedSizeBytes
	}
	if strings.TrimSpace(pathPlan.BaselineBackupPath) != "" {
		backupPath := pathPlan.BaselineBackupPath
		state.BaselineBackupPath = &backupPath
	}

	return state
}
