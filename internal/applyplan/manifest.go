package applyplan

import (
	"fmt"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/operationplan"
)

func appendManifestEntry(index int, operation operationplan.Operation, outcome operationOutcome, manifest *operationplan.AppliedOperationManifest) error {
	switch operation.Type {
	case operationplan.OperationTypeCreateDirectory:
		if !outcome.createdDirectory {
			return nil
		}
		targetPath, err := fileops.CleanAbsPath("created directory target path", operation.TargetPath)
		if err != nil {
			return err
		}
		manifest.CreatedDirectories = append(manifest.CreatedDirectories, operationplan.AppliedDirectoryManifestEntry{
			OperationIndex: index,
			Mod:            operation.Mod,
			TargetPath:     targetPath,
		})
	case operationplan.OperationTypeCopy:
		entry, err := buildAppliedFileManifestEntry(index, operation)
		if err != nil {
			return err
		}
		manifest.AddedFiles = append(manifest.AddedFiles, entry)
	case operationplan.OperationTypeReplace:
		entry, err := buildReplacedFileManifestEntry(index, operation)
		if err != nil {
			return err
		}
		manifest.ReplacedFiles = append(manifest.ReplacedFiles, entry)
	}

	return nil
}

func buildAppliedFileManifestEntry(index int, operation operationplan.Operation) (operationplan.AppliedFileManifestEntry, error) {
	sourcePath, err := fileops.CleanAbsPath("added file source path", *operation.SourcePath)
	if err != nil {
		return operationplan.AppliedFileManifestEntry{}, err
	}
	targetPath, err := fileops.CleanAbsPath("added file target path", operation.TargetPath)
	if err != nil {
		return operationplan.AppliedFileManifestEntry{}, err
	}
	targetSHA256, targetSize, err := computeFileIntegrity(targetPath)
	if err != nil {
		return operationplan.AppliedFileManifestEntry{}, fmt.Errorf("read added file integrity %q: %w", targetPath, err)
	}

	return operationplan.AppliedFileManifestEntry{
		OperationIndex: index,
		Mod:            operation.Mod,
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		SHA256:         targetSHA256,
		SizeBytes:      targetSize,
	}, nil
}

func buildReplacedFileManifestEntry(index int, operation operationplan.Operation) (operationplan.ReplacedFileManifestEntry, error) {
	sourcePath, err := fileops.CleanAbsPath("replaced file source path", *operation.SourcePath)
	if err != nil {
		return operationplan.ReplacedFileManifestEntry{}, err
	}
	targetPath, err := fileops.CleanAbsPath("replaced file target path", operation.TargetPath)
	if err != nil {
		return operationplan.ReplacedFileManifestEntry{}, err
	}
	backupPath, err := fileops.CleanAbsPath("replaced file backup path", *operation.BackupPath)
	if err != nil {
		return operationplan.ReplacedFileManifestEntry{}, err
	}
	targetSHA256, targetSize, err := computeFileIntegrity(targetPath)
	if err != nil {
		return operationplan.ReplacedFileManifestEntry{}, fmt.Errorf("read replaced file integrity %q: %w", targetPath, err)
	}
	backupSHA256, backupSize, err := computeFileIntegrity(backupPath)
	if err != nil {
		return operationplan.ReplacedFileManifestEntry{}, fmt.Errorf("read backup file integrity %q: %w", backupPath, err)
	}

	return operationplan.ReplacedFileManifestEntry{
		OperationIndex:  index,
		Mod:             operation.Mod,
		SourcePath:      sourcePath,
		TargetPath:      targetPath,
		SHA256:          targetSHA256,
		SizeBytes:       targetSize,
		BackupPath:      backupPath,
		BackupSHA256:    backupSHA256,
		BackupSizeBytes: backupSize,
	}, nil
}
