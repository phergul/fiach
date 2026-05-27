package operationplan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/phergul/mod-manager/internal/fileignore"
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

	directoryOperations, fileOperations, issues, err := buildResolvedModOperations(resolved)
	if err != nil {
		return OperationPlan{}, err
	}

	plan.Issues = append(plan.Issues, issues...)
	appendTargetPathConflictIssues(resolved.ProfileID, len(directoryOperations), fileOperations, &plan.Issues)

	sortDirectoryOperations(directoryOperations)

	plan.Operations = make([]Operation, 0, len(directoryOperations)+len(fileOperations))
	plan.Operations = append(plan.Operations, directoryOperations...)
	plan.Operations = append(plan.Operations, fileOperations...)
	plan.CanApply = canApplyPlan(plan.Issues)
	return plan, nil
}

func buildResolvedModOperations(resolved ResolveProfilePlanResult) ([]Operation, []Operation, []PlanIssue, error) {
	directoryOperations := make([]Operation, 0)
	fileOperations := make([]Operation, 0)
	issues := make([]PlanIssue, 0)
	seenDirectoryTargets := make(map[string]struct{})

	for _, mod := range resolved.Mods {
		buildResult, err := buildOperationsForMod(StrategyBuildInput{
			ProfileID:          resolved.ProfileID,
			GameInstallPath:    resolved.GameInstallPath,
			GameModStoragePath: resolved.GameModStoragePath,
			Mod:                mod,
		})
		if err != nil {
			return nil, nil, nil, err
		}

		issues = append(issues, buildResult.Issues...)
		appendStrategyOperations(buildResult.Operations, seenDirectoryTargets, &directoryOperations, &fileOperations)
	}

	return directoryOperations, fileOperations, issues, nil
}

func buildOperationsForMod(input StrategyBuildInput) (StrategyBuildResult, error) {
	adapter, found := strategyAdapters[input.Mod.StrategyType]
	if !found {
		return StrategyBuildResult{}, fmt.Errorf("unsupported install strategy %q", input.Mod.StrategyType)
	}

	buildResult, err := adapter.BuildOperations(input)
	if err != nil {
		return StrategyBuildResult{}, fmt.Errorf("build operations for mod %q: %w", input.Mod.ModName, err)
	}

	return buildResult, nil
}

func appendStrategyOperations(operations []Operation, seenDirectoryTargets map[string]struct{}, directoryOperations *[]Operation, fileOperations *[]Operation) {
	for _, operation := range operations {
		if operation.Type == OperationTypeCreateDirectory {
			if _, seen := seenDirectoryTargets[operation.TargetPath]; seen {
				continue
			}
			seenDirectoryTargets[operation.TargetPath] = struct{}{}
			*directoryOperations = append(*directoryOperations, operation)
			continue
		}

		*fileOperations = append(*fileOperations, operation)
	}
}

func sortDirectoryOperations(directoryOperations []Operation) {
	sort.SliceStable(directoryOperations, func(i int, j int) bool {
		leftDepth := pathDepth(directoryOperations[i].TargetPath)
		rightDepth := pathDepth(directoryOperations[j].TargetPath)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}

		return directoryOperations[i].TargetPath < directoryOperations[j].TargetPath
	})
}

type fileTreeStrategyAdapter struct{}

func (a fileTreeStrategyAdapter) BuildOperations(input StrategyBuildInput) (StrategyBuildResult, error) {
	if input.Mod.TargetBase != installconfig.TargetBaseGameRoot {
		return StrategyBuildResult{}, fmt.Errorf("unsupported target base %q", input.Mod.TargetBase)
	}

	sourceRoot := installpath.ResolveSourceRoot(input.Mod.ManagedSourcePath, input.Mod.SourceSubpath)
	builder, err := newModPlanBuilder(input, sourceRoot)
	if err != nil {
		return StrategyBuildResult{}, err
	}
	if builder.hasBlockingIssue() {
		return builder.result(), nil
	}

	if err := builder.walkSourceRoot(sourceRoot); err != nil {
		return StrategyBuildResult{}, err
	}

	builder.sortFileOperations()
	return builder.result(), nil
}

type modPlanBuilder struct {
	input StrategyBuildInput

	directoryOperations []Operation
	fileOperations      []Operation
	issues              []PlanIssue

	seenDirectoryTargets map[string]struct{}
	blockingIssue        *PlanIssue
}

func newModPlanBuilder(input StrategyBuildInput, sourceRoot string) (*modPlanBuilder, error) {
	builder := &modPlanBuilder{
		input:                input,
		directoryOperations:  []Operation{},
		fileOperations:       []Operation{},
		issues:               []PlanIssue{},
		seenDirectoryTargets: map[string]struct{}{},
	}

	sourceIssues, err := validateSourceRoot(input, sourceRoot)
	if err != nil {
		return nil, err
	}
	if len(sourceIssues) > 0 {
		builder.issues = append(builder.issues, sourceIssues...)
		builder.blockingIssue = &builder.issues[len(builder.issues)-1]
	}

	return builder, nil
}

func (b *modPlanBuilder) walkSourceRoot(sourceRoot string) error {
	stopWalk := errors.New("stop walk due to planner issue")

	walkErr := filepath.WalkDir(sourceRoot, func(sourceFilePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if sourceFilePath == sourceRoot {
			return nil
		}
		if fileignore.Has(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			if err := b.handleSourceDirectory(sourceRoot, sourceFilePath); err != nil {
				return err
			}
			if b.hasBlockingIssue() {
				return stopWalk
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source path %q is not a regular file or folder", sourceFilePath)
		}

		if err := b.handleSourceFile(sourceRoot, sourceFilePath); err != nil {
			return err
		}
		if b.hasBlockingIssue() {
			return stopWalk
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, stopWalk) {
		return walkErr
	}

	return nil
}

func (b *modPlanBuilder) handleSourceDirectory(sourceRoot string, sourceDirectoryPath string) error {
	targetPath, _, err := b.resolveTargetPaths(sourceRoot, sourceDirectoryPath)
	if err != nil {
		return err
	}

	return b.ensureTargetDirectories(targetPath)
}

func (b *modPlanBuilder) handleSourceFile(sourceRoot string, sourceFilePath string) error {
	targetPath, targetRelativePath, err := b.resolveTargetPaths(sourceRoot, sourceFilePath)
	if err != nil {
		return err
	}

	if err := b.ensureTargetDirectories(filepath.Dir(targetPath)); err != nil {
		return err
	}
	if b.hasBlockingIssue() {
		return nil
	}

	sourcePath := sourceFilePath
	operation := b.newFileOperation(sourcePath, targetPath)

	targetInfo, err := b.statTargetPath(targetPath)
	if err != nil {
		return err
	}
	if targetInfo == nil {
		b.fileOperations = append(b.fileOperations, operation)
		return nil
	}

	if targetInfo.IsDir() {
		b.setBlockingIssue(b.newTargetFilePathDirectoryIssue(sourcePath, targetPath))
		return nil
	}
	if strings.TrimSpace(b.input.GameModStoragePath) == "" {
		b.setBlockingIssue(b.newMissingBackupStorageIssue(sourcePath, targetPath))
		return nil
	}

	backupPath := backupPathForTarget(b.input.GameModStoragePath, targetRelativePath)
	operation.Type = OperationTypeReplace
	operation.BackupPath = &backupPath
	b.issues = append(b.issues, b.newReplaceExistingTargetWarning(sourcePath, targetPath))
	b.fileOperations = append(b.fileOperations, operation)
	return nil
}

func (b *modPlanBuilder) resolveTargetPaths(sourceRoot string, sourcePath string) (string, string, error) {
	sourceRelativePath, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve source relative path %q: %w", sourcePath, err)
	}

	targetRelativePath := installpath.JoinTargetRelativePath(
		b.input.Mod.TargetRelativePath,
		filepath.ToSlash(sourceRelativePath),
	)
	targetPath := filepath.Join(b.input.GameInstallPath, filepath.FromSlash(targetRelativePath))
	return targetPath, targetRelativePath, nil
}

func (b *modPlanBuilder) statTargetPath(targetPath string) (fs.FileInfo, error) {
	targetInfo, err := os.Stat(targetPath)
	if err == nil {
		return targetInfo, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	return nil, fmt.Errorf("stat target file %q: %w", targetPath, err)
}

func (b *modPlanBuilder) newFileOperation(sourcePath string, targetPath string) Operation {
	return Operation{
		Type:       OperationTypeCopy,
		SourcePath: &sourcePath,
		TargetPath: targetPath,
		Conflict:   false,
		Mod: ModContext{
			ModID:   b.input.Mod.ModID,
			ModName: b.input.Mod.ModName,
		},
	}
}

func (b *modPlanBuilder) setBlockingIssue(issue PlanIssue) {
	b.blockingIssue = &issue
}

func (b *modPlanBuilder) hasBlockingIssue() bool {
	return b.blockingIssue != nil
}

func (b *modPlanBuilder) sortFileOperations() {
	sort.SliceStable(b.fileOperations, func(i int, j int) bool {
		return b.fileOperations[i].TargetPath < b.fileOperations[j].TargetPath
	})
}

func (b *modPlanBuilder) result() StrategyBuildResult {
	result := StrategyBuildResult{
		Issues: append([]PlanIssue{}, b.issues...),
	}

	if b.blockingIssue != nil {
		if !containsPlanIssue(result.Issues, *b.blockingIssue) {
			result.Issues = append(result.Issues, *b.blockingIssue)
		}
		return result
	}

	result.Operations = make([]Operation, 0, len(b.directoryOperations)+len(b.fileOperations))
	result.Operations = append(result.Operations, b.directoryOperations...)
	result.Operations = append(result.Operations, b.fileOperations...)
	return result
}

func containsPlanIssue(issues []PlanIssue, target PlanIssue) bool {
	for _, issue := range issues {
		if planIssuesEqual(issue, target) {
			return true
		}
	}

	return false
}

func planIssuesEqual(left PlanIssue, right PlanIssue) bool {
	return left.Severity == right.Severity &&
		left.Kind == right.Kind &&
		left.Message == right.Message &&
		left.ProfileID == right.ProfileID &&
		stringPtrsEqual(left.SourcePath, right.SourcePath) &&
		stringPtrsEqual(left.TargetPath, right.TargetPath) &&
		modContextPtrsEqual(left.Mod, right.Mod) &&
		slices.Equal(left.ConflictingOperationIndexes, right.ConflictingOperationIndexes)
}

func stringPtrsEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func modContextPtrsEqual(left *ModContext, right *ModContext) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}
