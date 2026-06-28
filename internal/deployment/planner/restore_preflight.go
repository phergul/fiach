package planner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/fileops"
)

func preflightRestoreDelete(pathPlan PathPlan, gameInstallPath string) error {
	targetPath, err := targetAbsolutePathForRestore(gameInstallPath, pathPlan.GameRelativePath)
	if err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat added file target %q: %w", targetPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("added file target %q is not a regular file", targetPath)
	}
	if !pathPlan.Applied.Exists {
		return nil
	}

	return requireRestoreFileIntegrity(targetPath, pathPlan.Applied.SHA256, pathPlan.Applied.SizeBytes, "added file target")
}

func preflightRestoreBaseline(
	pathPlan PathPlan,
	appliedState appliedstate.PersistedFileState,
	gameInstallPath string,
	gameModStoragePath string,
) error {
	targetPath, err := targetAbsolutePathForRestore(gameInstallPath, pathPlan.GameRelativePath)
	if err != nil {
		return err
	}
	backupPath, err := fileops.CleanRequiredAbsPath("backup file path", pathPlan.BaselineBackupPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("backup file path", backupPath, gameModStoragePath); err != nil {
		return err
	}

	info, err := fileops.StatRegularFile("replaced file target", targetPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("replaced file target %q is not a regular file", targetPath)
	}

	appliedHash := appliedHashFromState(appliedState)
	baselineHash := baselineHashFromState(appliedState)
	moddedMatch, moddedErr := fileMatchesRestoreIntegrity(targetPath, appliedHash.hash, appliedHash.size)
	backupMatch, backupErr := fileMatchesRestoreIntegrity(targetPath, baselineHash.hash, baselineHash.size)
	if moddedErr != nil {
		return fmt.Errorf("read replaced file target integrity %q: %w", targetPath, moddedErr)
	}
	if backupErr != nil {
		return fmt.Errorf("read restored file target integrity %q: %w", targetPath, backupErr)
	}
	if !moddedMatch && !backupMatch {
		return fmt.Errorf("replaced file target %q does not match the applied file or recorded backup integrity", targetPath)
	}

	return requireRestoreFileIntegrity(backupPath, baselineHash.hash, baselineHash.size, "backup file")
}

func targetAbsolutePathForRestore(gameInstallPath string, gameRelativePath string) (string, error) {
	gameInstallPath, err := fileops.CleanRequiredAbsPath("game install path", gameInstallPath)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(gameInstallPath, filepath.FromSlash(strings.TrimSpace(gameRelativePath)))
	if err := fileops.RequirePathWithinRoot("operation target path", targetPath, gameInstallPath); err != nil {
		return "", err
	}

	return targetPath, nil
}

func requireRestoreFileIntegrity(path string, sha256Hex string, sizeBytes int64, label string) error {
	matches, err := fileMatchesRestoreIntegrity(path, sha256Hex, sizeBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s %q is missing", label, path)
		}
		return fmt.Errorf("read %s integrity %q: %w", label, path, err)
	}
	if !matches {
		return fmt.Errorf("%s %q does not match recorded integrity", label, path)
	}

	return nil
}

func fileMatchesRestoreIntegrity(path string, sha256Hex string, sizeBytes int64) (bool, error) {
	if strings.TrimSpace(sha256Hex) == "" {
		return false, nil
	}

	hash, size, err := fileops.FileIntegrity(path)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(hash, sha256Hex) && size == sizeBytes, nil
}

type restoreIntegrity struct {
	hash string
	size int64
}

func appliedHashFromState(state appliedstate.PersistedFileState) restoreIntegrity {
	if state.AppliedSHA256 != nil {
		size := int64(0)
		if state.AppliedSizeBytes != nil {
			size = *state.AppliedSizeBytes
		}
		return restoreIntegrity{hash: *state.AppliedSHA256, size: size}
	}

	return restoreIntegrity{}
}

func baselineHashFromState(state appliedstate.PersistedFileState) restoreIntegrity {
	if state.BaselineSHA256 != nil {
		size := int64(0)
		if state.BaselineSizeBytes != nil {
			size = *state.BaselineSizeBytes
		}
		return restoreIntegrity{hash: *state.BaselineSHA256, size: size}
	}

	return restoreIntegrity{}
}
