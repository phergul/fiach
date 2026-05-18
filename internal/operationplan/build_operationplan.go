package operationplan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/installpath"
)

const backupRootDirName = "operation-backups"

type StrategyBuildInput struct {
	ProfileID          int64
	GameInstallPath    string
	GameModStoragePath string
	Mod                ProfilePlanMod
}

type StrategyBuildResult struct {
	Operations []Operation
	Issues     []PlanIssue
}

type StrategyAdapter interface {
	BuildOperations(input StrategyBuildInput) (StrategyBuildResult, error)
}

var strategyAdapters = map[installconfig.StrategyType]StrategyAdapter{
	installconfig.StrategyTypeGenericCopy:  fileTreeStrategyAdapter{},
	installconfig.StrategyTypeReplaceFiles: fileTreeStrategyAdapter{},
	installconfig.StrategyTypeBepInEx:      fileTreeStrategyAdapter{},
	installconfig.StrategyTypeUnrealPak:    fileTreeStrategyAdapter{},
}

func BuildOperationPlan(resolved ResolveProfilePlanResult) (plan OperationPlan, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("build operation plan: %w", err)
		}
	}()

	plan.Issues = append(plan.Issues, resolved.Issues...)

	globalIssues, err := validateGameInstallPath(resolved.ProfileID, resolved.GameInstallPath)
	if err != nil {
		return OperationPlan{}, err
	}
	if len(globalIssues) > 0 {
		plan.Issues = append(plan.Issues, globalIssues...)
		plan.CanApply = canApplyPlan(plan.Issues)
		return plan, nil
	}

	directoryOperations := make([]Operation, 0)
	fileOperations := make([]Operation, 0)
	seenDirectoryTargets := make(map[string]struct{})

	for _, mod := range resolved.Mods {
		adapter, found := strategyAdapters[mod.StrategyType]
		if !found {
			return OperationPlan{}, fmt.Errorf("unsupported install strategy %q", mod.StrategyType)
		}

		buildResult, buildErr := adapter.BuildOperations(StrategyBuildInput{
			ProfileID:          resolved.ProfileID,
			GameInstallPath:    resolved.GameInstallPath,
			GameModStoragePath: resolved.GameModStoragePath,
			Mod:                mod,
		})
		if buildErr != nil {
			return OperationPlan{}, fmt.Errorf("build operations for mod %q: %w", mod.ModName, buildErr)
		}

		plan.Issues = append(plan.Issues, buildResult.Issues...)

		for _, operation := range buildResult.Operations {
			if operation.Type == OperationTypeCreateDirectory {
				if _, seen := seenDirectoryTargets[operation.TargetPath]; seen {
					continue
				}
				seenDirectoryTargets[operation.TargetPath] = struct{}{}
				directoryOperations = append(directoryOperations, operation)
				continue
			}

			fileOperations = append(fileOperations, operation)
		}
	}

	appendTargetPathConflictIssues(resolved.ProfileID, fileOperations, &plan.Issues)

	sort.SliceStable(directoryOperations, func(i int, j int) bool {
		leftDepth := pathDepth(directoryOperations[i].TargetPath)
		rightDepth := pathDepth(directoryOperations[j].TargetPath)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}

		return directoryOperations[i].TargetPath < directoryOperations[j].TargetPath
	})

	plan.Operations = make([]Operation, 0, len(directoryOperations)+len(fileOperations))
	plan.Operations = append(plan.Operations, directoryOperations...)
	plan.Operations = append(plan.Operations, fileOperations...)
	plan.CanApply = canApplyPlan(plan.Issues)
	return plan, nil
}

type fileTreeStrategyAdapter struct{}

func (a fileTreeStrategyAdapter) BuildOperations(input StrategyBuildInput) (result StrategyBuildResult, err error) {
	if input.Mod.TargetBase != installconfig.TargetBaseGameRoot {
		return StrategyBuildResult{}, fmt.Errorf("unsupported target base %q", input.Mod.TargetBase)
	}

	sourceRoot := installpath.ResolveSourceRoot(input.Mod.ManagedSourcePath, input.Mod.SourceSubpath)
	sourceIssues, err := validateSourceRoot(input, sourceRoot)
	if err != nil {
		return StrategyBuildResult{}, err
	}
	if len(sourceIssues) > 0 {
		result.Issues = append(result.Issues, sourceIssues...)
		return result, nil
	}

	directoryOperations := make([]Operation, 0)
	fileOperations := make([]Operation, 0)
	warningIssues := make([]PlanIssue, 0)
	seenDirectoryTargets := make(map[string]struct{})

	var blockingIssue *PlanIssue
	stopWalk := errors.New("stop walk due to planner issue")

	walkErr := filepath.WalkDir(sourceRoot, func(sourceFilePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if sourceFilePath == sourceRoot {
			return nil
		}

		sourceRelativePath, err := filepath.Rel(sourceRoot, sourceFilePath)
		if err != nil {
			return fmt.Errorf("resolve source relative path %q: %w", sourceFilePath, err)
		}

		targetRelativePath := installpath.JoinTargetRelativePath(input.Mod.TargetRelativePath, filepath.ToSlash(sourceRelativePath))
		targetPath := filepath.Join(input.GameInstallPath, filepath.FromSlash(targetRelativePath))

		if entry.IsDir() {
			operations, issue, err := buildMissingDirectoryOperations(input, targetPath, seenDirectoryTargets)
			if err != nil {
				return err
			}
			if issue != nil {
				blockingIssue = issue
				return stopWalk
			}
			directoryOperations = append(directoryOperations, operations...)
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source path %q is not a regular file or folder", sourceFilePath)
		}

		operations, issue, err := buildMissingDirectoryOperations(input, filepath.Dir(targetPath), seenDirectoryTargets)
		if err != nil {
			return err
		}
		if issue != nil {
			blockingIssue = issue
			return stopWalk
		}
		directoryOperations = append(directoryOperations, operations...)

		sourcePath := sourceFilePath
		operation := Operation{
			Type:       OperationTypeCopy,
			SourcePath: &sourcePath,
			TargetPath: targetPath,
			Conflict:   false,
			Mod: ModContext{
				ModID:   input.Mod.ModID,
				ModName: input.Mod.ModName,
			},
		}

		targetInfo, err := os.Stat(targetPath)
		if err == nil {
			if targetInfo.IsDir() {
				issue := newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueTargetFilePathDirectory,
					input.ProfileID,
					fmt.Sprintf("mod %q targets file path %q, but that path is an existing directory", input.Mod.ModName, targetPath),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					&sourcePath,
					stringPtr(targetPath),
				)
				blockingIssue = &issue
				return stopWalk
			}
			if strings.TrimSpace(input.GameModStoragePath) == "" {
				issue := newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueMissingGameModStorage,
					input.ProfileID,
					fmt.Sprintf("mod %q would replace %q, but a game mod storage path is required to plan the backup", input.Mod.ModName, targetPath),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					&sourcePath,
					stringPtr(targetPath),
				)
				blockingIssue = &issue
				return stopWalk
			}

			backupPath := backupPathForTarget(input.GameModStoragePath, targetRelativePath)
			operation.Type = OperationTypeReplace
			operation.BackupPath = &backupPath
			warningIssues = append(warningIssues, newPlanIssue(
				PlanIssueSeverityWarning,
				PlanIssueReplaceExistingTarget,
				input.ProfileID,
				fmt.Sprintf("mod %q would replace existing target file %q", input.Mod.ModName, targetPath),
				modContextPtr(input.Mod.ModID, input.Mod.ModName),
				&sourcePath,
				stringPtr(targetPath),
			))
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat target file %q: %w", targetPath, err)
		}

		fileOperations = append(fileOperations, operation)
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, stopWalk) {
			if blockingIssue != nil {
				result.Issues = append(result.Issues, *blockingIssue)
			}
			return result, nil
		}
		return StrategyBuildResult{}, walkErr
	}

	sort.SliceStable(fileOperations, func(i int, j int) bool {
		return fileOperations[i].TargetPath < fileOperations[j].TargetPath
	})

	result.Operations = make([]Operation, 0, len(directoryOperations)+len(fileOperations))
	result.Operations = append(result.Operations, directoryOperations...)
	result.Operations = append(result.Operations, fileOperations...)
	result.Issues = append(result.Issues, warningIssues...)
	return result, nil
}

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

func appendTargetPathConflictIssues(profileID int64, fileOperations []Operation, issues *[]PlanIssue) {
	targets := make(map[string][]int)
	for index := range fileOperations {
		targets[fileOperations[index].TargetPath] = append(targets[fileOperations[index].TargetPath], index)
	}

	conflictingTargets := make([]string, 0)
	for targetPath, indexes := range targets {
		if len(indexes) > 1 {
			conflictingTargets = append(conflictingTargets, targetPath)
		}
	}
	sort.Strings(conflictingTargets)

	for _, targetPath := range conflictingTargets {
		indexes := targets[targetPath]
		modNames := make([]string, 0, len(indexes))
		for _, index := range indexes {
			fileOperations[index].Conflict = true
			modNames = append(modNames, fmt.Sprintf("%q", fileOperations[index].Mod.ModName))
		}
		sort.Strings(modNames)

		*issues = append(*issues, newPlanIssue(
			PlanIssueSeverityError,
			PlanIssueTargetPathConflict,
			profileID,
			fmt.Sprintf("multiple planned operations target %q (mods: %s)", targetPath, strings.Join(modNames, ", ")),
			nil,
			nil,
			stringPtr(targetPath),
		))
	}
}

func canApplyPlan(issues []PlanIssue) bool {
	for _, issue := range issues {
		if issue.Severity == PlanIssueSeverityError {
			return false
		}
	}

	return true
}

func backupPathForTarget(gameModStoragePath string, gameRelativeTargetPath string) string {
	return filepath.Join(gameModStoragePath, backupRootDirName, filepath.FromSlash(gameRelativeTargetPath))
}

func pathDepth(targetPath string) int {
	cleanPath := filepath.Clean(targetPath)
	if cleanPath == string(filepath.Separator) || cleanPath == "." {
		return 0
	}

	trimmed := strings.TrimPrefix(filepath.ToSlash(cleanPath), "/")
	if trimmed == "" {
		return 0
	}

	return strings.Count(trimmed, "/") + 1
}

func buildMissingDirectoryOperations(input StrategyBuildInput, targetDirectoryPath string, seenDirectoryTargets map[string]struct{}) ([]Operation, *PlanIssue, error) {
	rootPath := filepath.Clean(input.GameInstallPath)
	currentPath := filepath.Clean(targetDirectoryPath)
	if currentPath == "." || currentPath == string(filepath.Separator) {
		return nil, nil, nil
	}

	missingPaths := make([]string, 0)
	for {
		if currentPath == rootPath {
			break
		}

		info, err := os.Stat(currentPath)
		if err == nil {
			if !info.IsDir() {
				issue := newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueTargetDirectoryPathFile,
					input.ProfileID,
					fmt.Sprintf("mod %q targets directory path %q, but that path is an existing file", input.Mod.ModName, currentPath),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					nil,
					stringPtr(currentPath),
				)
				return nil, &issue, nil
			}
			break
		}
		if errors.Is(err, syscall.ENOTDIR) {
			blockingPath, found, blockingErr := findBlockingFilePath(rootPath, currentPath)
			if blockingErr != nil {
				return nil, nil, blockingErr
			}
			if found {
				issue := newPlanIssue(
					PlanIssueSeverityError,
					PlanIssueTargetDirectoryPathFile,
					input.ProfileID,
					fmt.Sprintf("mod %q targets directory path %q, but that path is an existing file", input.Mod.ModName, blockingPath),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					nil,
					stringPtr(blockingPath),
				)
				return nil, &issue, nil
			}
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("stat target directory %q: %w", currentPath, err)
		}

		missingPaths = append(missingPaths, currentPath)
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	operations := make([]Operation, 0, len(missingPaths))
	for index := len(missingPaths) - 1; index >= 0; index-- {
		missingPath := missingPaths[index]
		if _, seen := seenDirectoryTargets[missingPath]; seen {
			continue
		}
		seenDirectoryTargets[missingPath] = struct{}{}
		operations = append(operations, Operation{
			Type:       OperationTypeCreateDirectory,
			TargetPath: missingPath,
			Conflict:   false,
			Mod: ModContext{
				ModID:   input.Mod.ModID,
				ModName: input.Mod.ModName,
			},
		})
	}

	return operations, nil, nil
}

func findBlockingFilePath(rootPath string, currentPath string) (string, bool, error) {
	searchPath := filepath.Clean(currentPath)
	cleanRootPath := filepath.Clean(rootPath)

	for {
		if searchPath == cleanRootPath || searchPath == "." || searchPath == string(filepath.Separator) {
			return "", false, nil
		}

		info, err := os.Stat(searchPath)
		if err == nil {
			if !info.IsDir() {
				return searchPath, true, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return "", false, fmt.Errorf("stat target directory %q: %w", searchPath, err)
		}

		parentPath := filepath.Dir(searchPath)
		if parentPath == searchPath {
			return "", false, nil
		}
		searchPath = parentPath
	}
}
