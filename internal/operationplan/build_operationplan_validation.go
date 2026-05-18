package operationplan

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func validateGameInstallPath(profileID int64, gameInstallPath string) ([]PlanIssue, error) {
	trimmedPath := strings.TrimSpace(gameInstallPath)
	if trimmedPath == "" {
		return []PlanIssue{
			newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueMissingGameInstallPath,
				profileID,
				"game install path is required to build an operation plan",
				nil,
				nil,
				nil,
			),
		}, nil
	}

	info, err := os.Stat(trimmedPath)
	if err == nil {
		if !info.IsDir() {
			return []PlanIssue{
				newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueGameInstallPathNotDir,
					profileID,
					fmt.Sprintf("game install path %q is not a directory", trimmedPath),
					nil,
					nil,
					stringPtr(trimmedPath),
				),
			}, nil
		}
		return nil, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return []PlanIssue{
			newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueMissingGameInstallDir,
				profileID,
				fmt.Sprintf("game install path %q does not exist", trimmedPath),
				nil,
				nil,
				stringPtr(trimmedPath),
			),
		}, nil
	}

	return nil, fmt.Errorf("stat game install path %q: %w", trimmedPath, err)
}

func validateSourceRoot(input StrategyBuildInput, sourceRoot string) ([]PlanIssue, error) {
	info, err := os.Stat(sourceRoot)
	if err == nil {
		if !info.IsDir() {
			return []PlanIssue{
				newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueSourceRootNotDirectory,
					input.ProfileID,
					fmt.Sprintf("mod %q source root %q is not a directory", input.Mod.ModName, sourceRoot),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					stringPtr(sourceRoot),
					nil,
				),
			}, nil
		}
		return nil, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return []PlanIssue{
			newPlanIssue(
				PlanIssueSeverityError,
				PlanIssueMissingSourceRoot,
				input.ProfileID,
				fmt.Sprintf("mod %q source root %q does not exist", input.Mod.ModName, sourceRoot),
				modContextPtr(input.Mod.ModID, input.Mod.ModName),
				stringPtr(sourceRoot),
				nil,
			),
		}, nil
	}

	return nil, fmt.Errorf("stat source root %q: %w", sourceRoot, err)
}

func (b *modPlanBuilder) newTargetFilePathDirectoryIssue(sourcePath string, targetPath string) PlanIssue {
	return newPlanIssue(
		PlanIssueSeverityError,
		PlanIssueTargetFilePathDirectory,
		b.input.ProfileID,
		fmt.Sprintf("mod %q targets file path %q, but that path is an existing directory", b.input.Mod.ModName, targetPath),
		modContextPtr(b.input.Mod.ModID, b.input.Mod.ModName),
		&sourcePath,
		stringPtr(targetPath),
	)
}

func (b *modPlanBuilder) newMissingBackupStorageIssue(sourcePath string, targetPath string) PlanIssue {
	return newPlanIssue(
		PlanIssueSeverityError,
		PlanIssueMissingGameModStorage,
		b.input.ProfileID,
		fmt.Sprintf("mod %q would replace %q, but a game mod storage path is required to plan the backup", b.input.Mod.ModName, targetPath),
		modContextPtr(b.input.Mod.ModID, b.input.Mod.ModName),
		&sourcePath,
		stringPtr(targetPath),
	)
}

func (b *modPlanBuilder) newReplaceExistingTargetWarning(sourcePath string, targetPath string) PlanIssue {
	return newPlanIssue(
		PlanIssueSeverityWarning,
		PlanIssueReplaceExistingTarget,
		b.input.ProfileID,
		fmt.Sprintf("mod %q would replace existing target file %q", b.input.Mod.ModName, targetPath),
		modContextPtr(b.input.Mod.ModID, b.input.Mod.ModName),
		&sourcePath,
		stringPtr(targetPath),
	)
}

func (b *modPlanBuilder) newTargetDirectoryPathFileIssue(targetPath string) PlanIssue {
	return newPlanIssue(
		PlanIssueSeverityError,
		PlanIssueTargetDirectoryPathFile,
		b.input.ProfileID,
		fmt.Sprintf("mod %q targets directory path %q, but that path is an existing file", b.input.Mod.ModName, targetPath),
		modContextPtr(b.input.Mod.ModID, b.input.Mod.ModName),
		nil,
		stringPtr(targetPath),
	)
}
