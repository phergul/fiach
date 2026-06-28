package execute

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/filetxn"
)

func BuildOperations(
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	gameInstallPath string,
) (operations []filetxn.Operation, skippedCount int, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build deployment operations: %w", err)
		}
	}()

	gameInstallPath, err = fileops.CleanRequiredAbsPath("game install path", gameInstallPath)
	if err != nil {
		return nil, 0, err
	}

	canonicalPaths := sortedPlanPaths(plan)
	operations = make([]filetxn.Operation, 0, len(canonicalPaths))

	for _, canonicalPath := range canonicalPaths {
		pathPlan := plan.Paths[canonicalPath]

		operation, include, buildErr := buildPathOperation(pathPlan, desired, gameInstallPath, canonicalPath)
		if buildErr != nil {
			return nil, 0, buildErr
		}
		if !include {
			skippedCount++
			continue
		}

		operations = append(operations, operation)
	}

	return operations, skippedCount, nil
}

func sortedPlanPaths(plan planner.DeploymentPlan) []string {
	paths := make([]string, 0, len(plan.Paths))
	for canonicalPath := range plan.Paths {
		paths = append(paths, canonicalPath)
	}
	sort.Strings(paths)
	return paths
}

func buildPathOperation(
	pathPlan planner.PathPlan,
	desired deployment.DesiredState,
	gameInstallPath string,
	canonicalPath string,
) (operation filetxn.Operation, include bool, err error) {
	switch pathPlan.PlannedAction {
	case planner.ReapplyNoOp:
		return filetxn.Operation{}, false, nil
	case planner.ReapplyBlock, planner.ReapplyRequireDecision:
		return filetxn.Operation{}, false, fmt.Errorf("path %q has blocking action %q", pathPlan.GameRelativePath, pathPlan.PlannedAction)
	}

	targetPath, err := targetAbsolutePath(gameInstallPath, pathPlan.GameRelativePath, canonicalPath)
	if err != nil {
		return filetxn.Operation{}, false, err
	}

	switch pathPlan.PlannedAction {
	case planner.ReapplyCreate, planner.ReapplyReplace, planner.ReapplyBackupThenReplace:
		desiredFile, found := desired.Files[canonicalPath]
		if !found {
			return filetxn.Operation{}, false, fmt.Errorf("desired state missing for path %q", pathPlan.GameRelativePath)
		}
		if strings.TrimSpace(desiredFile.SourcePath) == "" {
			return filetxn.Operation{}, false, fmt.Errorf("desired source path missing for %q", pathPlan.GameRelativePath)
		}

		return filetxn.Operation{
			Type:       "copy",
			SourcePath: desiredFile.SourcePath,
			TargetPath: targetPath,
			SHA256:     desiredFile.SHA256,
			SizeBytes:  desiredFile.SizeBytes,
		}, true, nil

	case planner.ReapplyDelete, planner.ReapplyBackupThenDelete:
		return filetxn.Operation{
			Type:       "delete",
			TargetPath: targetPath,
		}, true, nil

	case planner.ReapplyRestoreBaseline, planner.ReapplyBackupThenRestore:
		if strings.TrimSpace(pathPlan.BaselineBackupPath) == "" {
			return filetxn.Operation{}, false, fmt.Errorf("baseline backup path missing for %q", pathPlan.GameRelativePath)
		}

		operation := filetxn.Operation{
			Type:       "restore",
			SourcePath: pathPlan.BaselineBackupPath,
			TargetPath: targetPath,
		}
		if pathPlan.Baseline.Exists {
			operation.SHA256 = pathPlan.Baseline.SHA256
			operation.SizeBytes = pathPlan.Baseline.SizeBytes
		}

		return operation, true, nil

	case planner.ReapplyRepair:
		desiredFile, found := desired.Files[canonicalPath]
		if !found {
			return filetxn.Operation{}, false, fmt.Errorf("desired state missing for repair path %q", pathPlan.GameRelativePath)
		}

		return filetxn.Operation{
			Type:       "adopt",
			TargetPath: targetPath,
			SHA256:     desiredFile.SHA256,
			SizeBytes:  desiredFile.SizeBytes,
		}, true, nil

	default:
		return filetxn.Operation{}, false, fmt.Errorf("unsupported planned action %q for path %q", pathPlan.PlannedAction, pathPlan.GameRelativePath)
	}
}

func targetAbsolutePath(gameInstallPath string, gameRelativePath string, canonicalPath string) (string, error) {
	relativePath := strings.TrimSpace(gameRelativePath)
	if relativePath == "" {
		relativePath = strings.ReplaceAll(canonicalPath, "\\", "/")
	}

	targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(relativePath))
	if err := fileops.RequirePathWithinRoot("operation target path", targetPath, gameInstallPath); err != nil {
		return "", err
	}

	return targetPath, nil
}
