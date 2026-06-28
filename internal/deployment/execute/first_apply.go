package execute

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/backup"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/fileops"
)

type BaselineBackup struct {
	GameRelativePath string
	BackupPath       string
	SHA256           string
	SizeBytes        int64
}

type CreatedDirectory struct {
	TargetPath string
	ModID      int64
	ModName    string
}

type FirstApplyOutcome struct {
	BaselineBackups    map[string]BaselineBackup
	CreatedDirectories []CreatedDirectory
}

func PrepareFirstApply(
	plan planner.DeploymentPlan,
	desired deployment.DesiredState,
	gameInstallPath string,
	gameModStoragePath string,
) (outcome FirstApplyOutcome, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare first apply: %w", err)
		}
	}()

	gameInstallPath, err = fileops.CleanRequiredAbsPath("game install path", gameInstallPath)
	if err != nil {
		return FirstApplyOutcome{}, err
	}
	gameModStoragePath, err = fileops.CleanRequiredAbsPath("game mod storage path", gameModStoragePath)
	if err != nil {
		return FirstApplyOutcome{}, err
	}

	outcome = FirstApplyOutcome{
		BaselineBackups:    map[string]BaselineBackup{},
		CreatedDirectories: []CreatedDirectory{},
	}
	seenDirectories := map[string]struct{}{}

	for _, canonicalPath := range sortedPlanPaths(plan) {
		pathPlan := plan.Paths[canonicalPath]
		desiredFile, found := desired.Files[canonicalPath]
		if !found {
			continue
		}

		switch pathPlan.PlannedAction {
		case planner.ReapplyCreate, planner.ReapplyReplace:
		default:
			continue
		}

		targetPath, targetErr := targetAbsolutePath(gameInstallPath, pathPlan.GameRelativePath, canonicalPath)
		if targetErr != nil {
			return FirstApplyOutcome{}, targetErr
		}

		modID, modName := winningModContext(desiredFile)
		createdDirs, dirErr := ensureTargetDirectories(
			gameInstallPath,
			filepath.Dir(targetPath),
			modID,
			modName,
			seenDirectories,
		)
		if dirErr != nil {
			return FirstApplyOutcome{}, dirErr
		}
		outcome.CreatedDirectories = append(outcome.CreatedDirectories, createdDirs...)

		if pathPlan.PlannedAction != planner.ReapplyReplace {
			continue
		}

		backupPath := backup.PathForTarget(gameModStoragePath, pathPlan.GameRelativePath)
		if err := fileops.RequirePathWithinRoot("operation backup path", backupPath, gameModStoragePath); err != nil {
			return FirstApplyOutcome{}, err
		}

		targetInfo, statErr := fileops.StatRegularFile("target file", targetPath)
		if statErr != nil {
			return FirstApplyOutcome{}, statErr
		}

		if _, err := os.Lstat(backupPath); err == nil {
			return FirstApplyOutcome{}, fmt.Errorf("backup file %q already exists", backupPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return FirstApplyOutcome{}, fmt.Errorf("stat backup file %q: %w", backupPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return FirstApplyOutcome{}, fmt.Errorf("create backup directory %q: %w", filepath.Dir(backupPath), err)
		}
		if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: targetPath,
			TargetPath: backupPath,
			Mode:       targetInfo.Mode().Perm(),
			OpenLabel:  "baseline backup source",
		}); err != nil {
			return FirstApplyOutcome{}, fmt.Errorf("create baseline backup %q: %w", backupPath, err)
		}

		backupSHA256, backupSize, integrityErr := fileops.FileIntegrity(backupPath)
		if integrityErr != nil {
			return FirstApplyOutcome{}, integrityErr
		}

		outcome.BaselineBackups[canonicalPath] = BaselineBackup{
			GameRelativePath: pathPlan.GameRelativePath,
			BackupPath:       backupPath,
			SHA256:           backupSHA256,
			SizeBytes:        backupSize,
		}
	}

	return outcome, nil
}

func ensureTargetDirectories(
	gameInstallPath string,
	targetDirectoryPath string,
	modID int64,
	modName string,
	seenDirectories map[string]struct{},
) ([]CreatedDirectory, error) {
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
				return nil, fmt.Errorf("target path %q already exists and is not a directory", currentPath)
			}
			break
		}
		if errors.Is(err, syscall.ENOTDIR) {
			blockingPath, found, blockingErr := findBlockingFilePath(rootPath, currentPath)
			if blockingErr != nil {
				return nil, blockingErr
			}
			if found {
				return nil, fmt.Errorf("target directory path %q is blocked by existing file %q", currentPath, blockingPath)
			}
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

	created := make([]CreatedDirectory, 0, len(missingPaths))
	for index := len(missingPaths) - 1; index >= 0; index-- {
		dirPath := missingPaths[index]
		key := strings.ToLower(filepath.Clean(dirPath))
		if _, seen := seenDirectories[key]; seen {
			continue
		}
		seenDirectories[key] = struct{}{}

		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return nil, fmt.Errorf("create target directory %q: %w", dirPath, err)
		}

		created = append(created, CreatedDirectory{
			TargetPath: dirPath,
			ModID:      modID,
			ModName:    modName,
		})
	}

	return created, nil
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

func winningModContext(file deployment.DesiredFile) (int64, string) {
	if file.Winner.ModID != nil {
		return *file.Winner.ModID, file.Winner.ModName
	}
	for _, writer := range file.Writers {
		if writer.SourceKind == deployment.SourceKindMod && writer.ModID != nil {
			return *writer.ModID, writer.ModName
		}
	}
	return 0, ""
}

func sortedCreatedDirectories(directories []CreatedDirectory) []CreatedDirectory {
	if len(directories) == 0 {
		return nil
	}

	copied := append([]CreatedDirectory(nil), directories...)
	sort.Slice(copied, func(i int, j int) bool {
		return strings.ToLower(copied[i].TargetPath) < strings.ToLower(copied[j].TargetPath)
	})
	return copied
}
