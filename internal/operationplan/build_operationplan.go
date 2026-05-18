package operationplan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

type StrategyAdapter interface {
	BuildOperations(input StrategyBuildInput) ([]Operation, error)
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

	if strings.TrimSpace(resolved.GameInstallPath) == "" {
		return OperationPlan{}, errors.New("game install path is required")
	}
	if strings.TrimSpace(resolved.GameModStoragePath) == "" {
		return OperationPlan{}, errors.New("game mod storage path is required")
	}

	directoryOperations := make([]Operation, 0)
	fileOperations := make([]Operation, 0)
	seenDirectoryTargets := make(map[string]struct{})

	for _, mod := range resolved.Mods {
		adapter, found := strategyAdapters[mod.StrategyType]
		if !found {
			return OperationPlan{}, fmt.Errorf("unsupported install strategy %q", mod.StrategyType)
		}

		operations, buildErr := adapter.BuildOperations(StrategyBuildInput{
			ProfileID:          resolved.ProfileID,
			GameInstallPath:    resolved.GameInstallPath,
			GameModStoragePath: resolved.GameModStoragePath,
			Mod:                mod,
		})
		if buildErr != nil {
			return OperationPlan{}, fmt.Errorf("build operations for mod %q: %w", mod.ModName, buildErr)
		}

		for _, operation := range operations {
			// directory operations must be de-duplicated and ordered before file operations to ensure correct execution.
			// if multiple mods require the same directory to be created, only one create_directory operation should be
			// included in the plan, and it should be ordered before any file operations that target paths within that directory.
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
	return plan, nil
}

type fileTreeStrategyAdapter struct{}

func (a fileTreeStrategyAdapter) BuildOperations(input StrategyBuildInput) ([]Operation, error) {
	if input.Mod.TargetBase != installconfig.TargetBaseGameRoot {
		return nil, fmt.Errorf("unsupported target base %q", input.Mod.TargetBase)
	}

	sourceRoot := installpath.ResolveSourceRoot(input.Mod.ManagedSourcePath, input.Mod.SourceSubpath)
	sourceRootInfo, err := os.Stat(sourceRoot)
	if err != nil {
		return nil, fmt.Errorf("stat source root %q: %w", sourceRoot, err)
	}
	if !sourceRootInfo.IsDir() {
		return nil, fmt.Errorf("source root %q is not a directory", sourceRoot)
	}

	directoryOperations := make([]Operation, 0)
	fileOperations := make([]Operation, 0)
	seenDirectoryTargets := make(map[string]struct{})

	err = filepath.WalkDir(sourceRoot, func(sourceFilePath string, entry fs.DirEntry, walkErr error) error {
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
			operations, err := buildMissingDirectoryOperations(input.GameInstallPath, targetPath, seenDirectoryTargets, input.Mod.ModID, input.Mod.ModName)
			if err != nil {
				return err
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

		operations, err := buildMissingDirectoryOperations(input.GameInstallPath, filepath.Dir(targetPath), seenDirectoryTargets, input.Mod.ModID, input.Mod.ModName)
		if err != nil {
			return err
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
				return fmt.Errorf("target file path %q is an existing directory", targetPath)
			}

			backupPath := backupPathForTarget(input.GameModStoragePath, targetRelativePath)
			operation.Type = OperationTypeReplace
			operation.BackupPath = &backupPath
			operation.Conflict = true
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat target file %q: %w", targetPath, err)
		}

		fileOperations = append(fileOperations, operation)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(fileOperations, func(i int, j int) bool {
		return fileOperations[i].TargetPath < fileOperations[j].TargetPath
	})

	operations := make([]Operation, 0, len(directoryOperations)+len(fileOperations))
	operations = append(operations, directoryOperations...)
	operations = append(operations, fileOperations...)
	return operations, nil
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

func buildMissingDirectoryOperations(gameInstallPath string, targetDirectoryPath string, seenDirectoryTargets map[string]struct{}, modID int64, modName string) ([]Operation, error) {
	rootPath := filepath.Clean(gameInstallPath)
	currentPath := filepath.Clean(targetDirectoryPath)
	if currentPath == "." || currentPath == string(filepath.Separator) {
		return nil, nil
	}

	missingPaths := make([]string, 0)
	for {
		if currentPath == rootPath {
			break
		}

		info, err := os.Stat(currentPath)
		if err == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("target directory path %q is an existing file", currentPath)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat target directory %q: %w", currentPath, err)
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
				ModID:   modID,
				ModName: modName,
			},
		})
	}

	return operations, nil
}
