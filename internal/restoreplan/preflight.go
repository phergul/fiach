package restoreplan

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/phergul/fiach/internal/appliedstate"
	"github.com/phergul/fiach/internal/fileops"
)

func preflightOperations(operations []RestoreOperation, manifest appliedstate.ManifestDocument, context resolvedContext) map[int]error {
	failures := map[int]error{}
	addedFiles := map[int]appliedstate.AddedFile{}
	replacedFiles := map[int]appliedstate.ReplacedFile{}
	createdDirectories := map[int]appliedstate.CreatedDirectory{}

	for _, entry := range manifest.AddedFiles {
		addedFiles[entry.OperationIndex] = entry
	}
	for _, entry := range manifest.ReplacedFiles {
		replacedFiles[entry.OperationIndex] = entry
	}
	for _, entry := range manifest.CreatedDirectories {
		createdDirectories[entry.OperationIndex] = entry
	}

	for index, operation := range operations {
		var err error
		switch operation.Type {
		case RestoreOperationTypeRemoveAddedFile:
			err = preflightAddedFile(addedFiles[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeRestoreReplacedFile:
			err = preflightReplacedFile(replacedFiles[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeRemoveCreatedDir:
			err = preflightCreatedDirectory(createdDirectories[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeDeleteRestoredBackup:
			err = preflightBackupCleanup(replacedFiles[operation.ManifestOperationIndex], context)
		default:
			err = fmt.Errorf("unsupported restore operation type %q", operation.Type)
		}
		if err != nil {
			failures[index] = err
		}
	}

	return failures
}

func preflightAddedFile(entry appliedstate.AddedFile, context resolvedContext) error {
	targetPath, err := fileops.CleanRequiredAbsPath("added file target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("added file target path", targetPath, context.gameInstallPath); err != nil {
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
	if err := requireFileIntegrity(targetPath, entry.SHA256, entry.SizeBytes, "added file target"); err != nil {
		return err
	}

	return nil
}

func preflightReplacedFile(entry appliedstate.ReplacedFile, context resolvedContext) error {
	targetPath, err := fileops.CleanRequiredAbsPath("replaced file target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("replaced file target path", targetPath, context.gameInstallPath); err != nil {
		return err
	}
	backupPath, err := fileops.CleanRequiredAbsPath("backup file path", entry.BackupPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("backup file path", backupPath, context.gameModStoragePath); err != nil {
		return err
	}

	info, err := fileops.StatRegularFile("replaced file target", targetPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("replaced file target %q is not a regular file", targetPath)
	}

	moddedMatch, moddedErr := fileMatchesIntegrity(targetPath, entry.SHA256, entry.SizeBytes)
	backupMatch, backupErr := fileMatchesIntegrity(targetPath, entry.BackupSHA256, entry.BackupSizeBytes)
	if moddedErr != nil {
		return fmt.Errorf("read replaced file target integrity %q: %w", targetPath, moddedErr)
	}
	if backupErr != nil {
		return fmt.Errorf("read restored file target integrity %q: %w", targetPath, backupErr)
	}
	if !moddedMatch && !backupMatch {
		return fmt.Errorf("replaced file target %q does not match the applied file or recorded backup integrity", targetPath)
	}
	if err := requireFileIntegrity(backupPath, entry.BackupSHA256, entry.BackupSizeBytes, "backup file"); err != nil {
		return err
	}

	return nil
}

func preflightCreatedDirectory(entry appliedstate.CreatedDirectory, context resolvedContext) error {
	targetPath, err := fileops.CleanRequiredAbsPath("created directory target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("created directory target path", targetPath, context.gameInstallPath); err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat created directory %q: %w", targetPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("created directory target %q is not a directory", targetPath)
	}

	return nil
}

func preflightBackupCleanup(entry appliedstate.ReplacedFile, context resolvedContext) error {
	backupPath, err := fileops.CleanRequiredAbsPath("backup file path", entry.BackupPath)
	if err != nil {
		return err
	}
	if err := fileops.RequirePathWithinRoot("backup file path", backupPath, context.gameModStoragePath); err != nil {
		return err
	}

	return nil
}

func requireFileIntegrity(path string, sha256Hex string, sizeBytes int64, label string) error {
	matches, err := fileMatchesIntegrity(path, sha256Hex, sizeBytes)
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

func fileMatchesIntegrity(path string, sha256Hex string, sizeBytes int64) (bool, error) {
	hash, size, err := computeFileIntegrity(path)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(hash, sha256Hex) && size == sizeBytes, nil
}
