package operationplan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

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

func (b *modPlanBuilder) ensureTargetDirectories(targetDirectoryPath string) error {
	rootPath := filepath.Clean(b.input.GameInstallPath)
	currentPath := filepath.Clean(targetDirectoryPath)
	if currentPath == "." || currentPath == string(filepath.Separator) {
		return nil
	}

	missingPaths := make([]string, 0)
	for {
		if currentPath == rootPath {
			break
		}

		info, err := os.Stat(currentPath)
		if err == nil {
			if !info.IsDir() {
				b.setBlockingIssue(b.newTargetDirectoryPathFileIssue(currentPath))
				return nil
			}
			break
		}
		if errors.Is(err, syscall.ENOTDIR) {
			blockingPath, found, blockingErr := findBlockingFilePath(rootPath, currentPath)
			if blockingErr != nil {
				return blockingErr
			}
			if found {
				b.setBlockingIssue(b.newTargetDirectoryPathFileIssue(blockingPath))
				return nil
			}
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat target directory %q: %w", currentPath, err)
		}

		missingPaths = append(missingPaths, currentPath)
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	for index := len(missingPaths) - 1; index >= 0; index-- {
		b.addDirectoryOperation(missingPaths[index])
	}

	return nil
}

func (b *modPlanBuilder) addDirectoryOperation(targetPath string) {
	if _, seen := b.seenDirectoryTargets[targetPath]; seen {
		return
	}
	b.seenDirectoryTargets[targetPath] = struct{}{}

	b.directoryOperations = append(b.directoryOperations, Operation{
		Type:       OperationTypeCreateDirectory,
		TargetPath: targetPath,
		Conflict:   false,
		Mod: ModContext{
			ModID:   b.input.Mod.ModID,
			ModName: b.input.Mod.ModName,
		},
	})
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
