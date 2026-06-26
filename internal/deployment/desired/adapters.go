package desired

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/fileignore"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/installconfig"
	"github.com/phergul/fiach/internal/installpath"
	"github.com/phergul/fiach/internal/operationplan"
	"github.com/phergul/fiach/internal/unrealpak"
)

var desiredFileAdapters = map[installconfig.StrategyType]DesiredFileAdapter{
	installconfig.StrategyTypeGenericCopy: fileTreeDesiredAdapter{},
	installconfig.StrategyTypeBepInEx:     fileTreeDesiredAdapter{},
	installconfig.StrategyTypeUnrealPak:   unrealPakDesiredAdapter{},
}

func inventoryFilesForMod(input operationplan.StrategyBuildInput) (DesiredInventoryResult, error) {
	adapter, found := desiredFileAdapters[input.Mod.StrategyType]
	if !found {
		return DesiredInventoryResult{}, fmt.Errorf("unsupported install strategy %q", input.Mod.StrategyType)
	}

	result, err := adapter.InventoryFiles(input)
	if err != nil {
		return DesiredInventoryResult{}, fmt.Errorf("inventory files for mod %q: %w", input.Mod.ModName, err)
	}

	return result, nil
}

type fileTreeDesiredAdapter struct{}

func (a fileTreeDesiredAdapter) InventoryFiles(input operationplan.StrategyBuildInput) (DesiredInventoryResult, error) {
	if input.Mod.TargetBase != installconfig.TargetBaseGameRoot {
		return DesiredInventoryResult{}, fmt.Errorf("unsupported target base %q", input.Mod.TargetBase)
	}

	sourceRoot := installpath.ResolveSourceRoot(input.Mod.ManagedSourcePath, input.Mod.SourceSubpath)
	inventory, err := newSourceInventory(input, sourceRoot)
	if err != nil {
		return DesiredInventoryResult{}, err
	}
	if inventory.hasBlockingIssue() {
		return inventory.result(), nil
	}

	if err := inventory.walkSourceRoot(sourceRoot); err != nil {
		return DesiredInventoryResult{}, err
	}

	return inventory.result(), nil
}

type unrealPakDesiredAdapter struct{}

func (a unrealPakDesiredAdapter) InventoryFiles(input operationplan.StrategyBuildInput) (DesiredInventoryResult, error) {
	if input.Mod.TargetBase != installconfig.TargetBaseGameRoot {
		return DesiredInventoryResult{}, fmt.Errorf("unsupported target base %q", input.Mod.TargetBase)
	}

	sourceRoot := installpath.ResolveSourceRoot(input.Mod.ManagedSourcePath, input.Mod.SourceSubpath)
	inventory, err := newSourceInventory(input, sourceRoot)
	if err != nil {
		return DesiredInventoryResult{}, err
	}
	if inventory.hasBlockingIssue() {
		return inventory.result(), nil
	}

	inspection, err := unrealpak.Inspect(sourceRoot)
	if err != nil {
		issue := newPlanIssue(
			deployment.PlanIssueSeverityError,
			deployment.PlanIssueInvalidUnrealPakSource,
			input.ProfileID,
			fmt.Sprintf("mod %q has an invalid Unreal package source: %v", input.Mod.ModName, err),
			modContextPtr(input.Mod.ModID, input.Mod.ModName),
			new(sourceRoot),
			nil,
		)
		return DesiredInventoryResult{Issues: []deployment.PlanIssue{issue}}, nil
	}

	for _, file := range inspection.Files {
		targetRelativePath := installpath.JoinTargetRelativePath(input.Mod.TargetRelativePath, file.Name)
		if err := inventory.addMappedSourceFile(file.SourcePath, targetRelativePath); err != nil {
			return DesiredInventoryResult{}, err
		}
	}

	return inventory.result(), nil
}

type sourceInventory struct {
	input operationplan.StrategyBuildInput

	mappings      []DesiredFileMapping
	issues        []deployment.PlanIssue
	blockingIssue *deployment.PlanIssue
}

func newSourceInventory(input operationplan.StrategyBuildInput, sourceRoot string) (*sourceInventory, error) {
	inventory := &sourceInventory{
		input:    input,
		mappings: []DesiredFileMapping{},
		issues:   []deployment.PlanIssue{},
	}

	sourceIssues, err := validateSourceRoot(input, sourceRoot)
	if err != nil {
		return nil, err
	}
	if len(sourceIssues) > 0 {
		inventory.issues = append(inventory.issues, sourceIssues...)
		inventory.blockingIssue = &inventory.issues[len(inventory.issues)-1]
	}

	return inventory, nil
}

func (i *sourceInventory) walkSourceRoot(sourceRoot string) error {
	stopWalk := errors.New("stop walk due to inventory issue")

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
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source path %q is not a regular file", sourceFilePath)
		}

		targetRelativePath, err := resolveTargetRelativePath(sourceRoot, sourceFilePath, i.input.Mod.TargetRelativePath)
		if err != nil {
			return err
		}
		if err := i.addMappedSourceFile(sourceFilePath, targetRelativePath); err != nil {
			return err
		}
		if i.hasBlockingIssue() {
			return stopWalk
		}

		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, stopWalk) {
		return walkErr
	}

	return nil
}

func (i *sourceInventory) addMappedSourceFile(sourceFilePath string, targetRelativePath string) error {
	sha256Hex, sizeBytes, err := fileops.FileIntegrity(sourceFilePath)
	if err != nil {
		return err
	}

	i.mappings = append(i.mappings, DesiredFileMapping{
		SourcePath:       sourceFilePath,
		GameRelativePath: targetRelativePath,
		SHA256:           sha256Hex,
		SizeBytes:        sizeBytes,
	})
	return nil
}

func (i *sourceInventory) hasBlockingIssue() bool {
	return i.blockingIssue != nil
}

func (i *sourceInventory) result() DesiredInventoryResult {
	result := DesiredInventoryResult{
		Issues: append([]deployment.PlanIssue{}, i.issues...),
	}
	if i.blockingIssue != nil {
		return result
	}

	result.Mappings = append([]DesiredFileMapping{}, i.mappings...)
	return result
}

func resolveTargetRelativePath(sourceRoot string, sourcePath string, targetRoot string) (string, error) {
	sourceRelativePath, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve source relative path %q: %w", sourcePath, err)
	}

	return installpath.JoinTargetRelativePath(targetRoot, filepath.ToSlash(sourceRelativePath)), nil
}

func validateSourceRoot(input operationplan.StrategyBuildInput, sourceRoot string) ([]deployment.PlanIssue, error) {
	info, err := os.Stat(sourceRoot)
	if err == nil {
		if !info.IsDir() {
			return []deployment.PlanIssue{
				newPlanIssue(
					deployment.PlanIssueSeverityError,
					deployment.PlanIssueSourceRootNotDirectory,
					input.ProfileID,
					fmt.Sprintf("mod %q source root %q is not a directory", input.Mod.ModName, sourceRoot),
					modContextPtr(input.Mod.ModID, input.Mod.ModName),
					new(sourceRoot),
					nil,
				),
			}, nil
		}
		return nil, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return []deployment.PlanIssue{
			newPlanIssue(
				deployment.PlanIssueSeverityError,
				deployment.PlanIssueMissingSourceRoot,
				input.ProfileID,
				fmt.Sprintf("mod %q source root %q does not exist", input.Mod.ModName, sourceRoot),
				modContextPtr(input.Mod.ModID, input.Mod.ModName),
				new(sourceRoot),
				nil,
			),
		}, nil
	}

	return nil, fmt.Errorf("stat source root %q: %w", sourceRoot, err)
}

func newPlanIssue(
	severity deployment.PlanIssueSeverity,
	kind deployment.PlanIssueKind,
	profileID int64,
	message string,
	mod *deployment.ModContext,
	sourcePath *string,
	targetPath *string,
) deployment.PlanIssue {
	return deployment.PlanIssue{
		Severity:   severity,
		Kind:       kind,
		Message:    message,
		ProfileID:  profileID,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mod:        mod,
	}
}

func modContextPtr(modID int64, modName string) *deployment.ModContext {
	return &deployment.ModContext{
		ModID:   modID,
		ModName: modName,
	}
}
