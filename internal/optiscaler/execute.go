package optiscaler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (m *Manager) execute(ctx context.Context, gameRoot string, preview Preview) (result ApplyResult, err error) {
	targetPath, err := ResolveWithinRoot(gameRoot, preview.Request.TargetRelativePath)
	if err != nil {
		return ApplyResult{}, err
	}
	journalID := fmt.Sprintf("%d-%s", m.now().UnixNano(), hashBytes([]byte(targetPath))[:12])
	journalPath := filepath.Join(m.dataDir, "journals", journalID+".json")
	journal := journalDocument{
		Version:            JournalVersion,
		ID:                 journalID,
		GameID:             preview.Request.GameID,
		TargetPath:         targetPath,
		TargetRelativePath: preview.Request.TargetRelativePath,
		Action:             preview.Request.Action,
		StartedAt:          m.now(),
	}
	journal.Snapshots, err = m.snapshotOperationTargets(journalID, preview.Operations)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := writeJournal(journalPath, journal); err != nil {
		return ApplyResult{}, err
	}

	rollbackOnFailure := func(operationErr error) (ApplyResult, error) {
		journal.Error = operationErr.Error()
		_ = writeJournal(journalPath, journal)
		if rollbackErr := m.rollbackSnapshots(journal.Snapshots); rollbackErr != nil {
			journal.Error = fmt.Sprintf("%v; rollback failed: %v", operationErr, rollbackErr)
			_ = writeJournal(journalPath, journal)
			_ = m.markRecoveryRequired(ctx, preview.Request)
			return ApplyResult{
				Success: false,
				Message: journal.Error,
			}, operationErr
		}
		_ = os.Remove(journalPath)
		_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
		return ApplyResult{
			Success:    false,
			RolledBack: true,
			Message:    operationErr.Error(),
		}, operationErr
	}

	if preview.Request.BackupAndContinue && len(preview.Drift) > 0 {
		if err := m.archiveDrift(targetPath, preview); err != nil {
			return rollbackOnFailure(err)
		}
	}
	if preview.Request.Action == ActionUninstall {
		if err := m.archiveUninstall(targetPath, preview); err != nil {
			return rollbackOnFailure(err)
		}
	}
	for index, operation := range preview.Operations {
		if err := m.executeOperation(operation); err != nil {
			return rollbackOnFailure(err)
		}
		journal.CompletedSteps = index + 1
		if err := writeJournal(journalPath, journal); err != nil {
			return rollbackOnFailure(err)
		}
	}
	if err := verifyOperations(preview.Operations); err != nil {
		return rollbackOnFailure(err)
	}
	if err := m.commitState(ctx, targetPath, preview, journal.Snapshots); err != nil {
		return rollbackOnFailure(err)
	}
	journal.DatabaseCommitted = true
	if err := writeJournal(journalPath, journal); err != nil {
		if removeErr := os.Remove(journalPath); removeErr != nil {
			return ApplyResult{
				Success: true,
				Message: "OptiScaler action completed, but journal cleanup requires attention.",
			}, nil
		}
		_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journalID))
		return ApplyResult{
			Success: true,
			Message: "OptiScaler action completed.",
		}, nil
	}
	if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return ApplyResult{}, err
	}
	_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journalID))
	return ApplyResult{
		Success: true,
		Message: "OptiScaler action completed.",
	}, nil
}

func (m *Manager) snapshotOperationTargets(journalID string, operations []Operation) ([]journalSnapshot, error) {
	root := filepath.Join(m.dataDir, "journals", journalID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var snapshots []journalSnapshot
	for _, operation := range operations {
		for _, target := range operationTouchedPaths(operation) {
			key := strings.ToLower(filepath.Clean(target))
			if seen[key] {
				continue
			}
			seen[key] = true
			snapshot := journalSnapshot{TargetPath: target}
			info, err := os.Stat(target)
			if errors.Is(err, os.ErrNotExist) {
				snapshots = append(snapshots, snapshot)
				continue
			}
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("mutation target %q is not a regular file", target)
			}
			snapshot.Existed = true
			snapshot.BackupPath = filepath.Join(root, fmt.Sprintf("%03d.bak", len(snapshots)))
			if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
				SourcePath: target,
				TargetPath: snapshot.BackupPath,
				Mode:       0o644,
				Replace:    false,
				OpenLabel:  "mutation target",
			}); err != nil {
				return nil, err
			}
			snapshot.SHA256, snapshot.SizeBytes, err = fileops.FileIntegrity(snapshot.BackupPath)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

func operationTouchedPaths(operation Operation) []string {
	if operation.Type == "move" {
		return []string{operation.SourcePath, operation.TargetPath}
	}
	return []string{operation.TargetPath}
}

func executeOperation(operation Operation) error {
	switch operation.Type {
	case "copy", "restore":
		if err := os.MkdirAll(filepath.Dir(operation.TargetPath), 0o755); err != nil {
			return err
		}
		return fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: operation.SourcePath,
			TargetPath: operation.TargetPath,
			Mode:       0o644,
			Replace:    true,
			OpenLabel:  "staged OptiScaler file",
		})
	case "delete":
		if err := os.Remove(operation.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	case "move":
		if err := os.Remove(operation.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return os.Rename(operation.SourcePath, operation.TargetPath)
	case "adopt":
		return nil
	default:
		return fmt.Errorf("unsupported operation type %q", operation.Type)
	}
}

func verifyOperations(operations []Operation) error {
	for _, operation := range operations {
		switch operation.Type {
		case "copy", "restore", "adopt":
			matches, err := fileops.FileMatchesIntegrity(operation.TargetPath, operation.SHA256, operation.SizeBytes)
			if err != nil || !matches {
				return fmt.Errorf("verify managed file %q", operation.TargetPath)
			}
		case "delete":
			if _, err := os.Stat(operation.TargetPath); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("verify deleted file %q", operation.TargetPath)
			}
		case "move":
			info, err := os.Stat(operation.TargetPath)
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("verify moved target %q", operation.TargetPath)
			}
		}
	}
	return nil
}

func (m *Manager) commitState(ctx context.Context, targetPath string, preview Preview, snapshots []journalSnapshot) error {
	if preview.Request.Action == ActionUninstall {
		return m.store.DeleteOptiScalerTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath)
	}
	manifest := Manifest{
		Version: ManifestVersion,
		Config: ManagedConfig{
			LoadReShade:       preview.Request.EnableReShadeCoexistence,
			DXGISpoofing:      preview.Request.DXGISpoofing,
			TargetProcessName: preview.Request.ProcessFilter,
			CheckForUpdate:    false,
		},
		HasPreAdoptionRollbackData: preview.Request.Action != ActionAdopt,
	}
	var previousManifest Manifest
	previousFiles := map[string]ManagedFile{}
	if existing, found, err := m.store.GetOptiScalerTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath); err != nil {
		return err
	} else if found {
		previousManifest, err = decodeManifest(existing.ManifestJSON)
		if err != nil {
			return err
		}
		manifest.OriginalReShadeProxy = previousManifest.OriginalReShadeProxy
		manifest.HasPreAdoptionRollbackData = previousManifest.HasPreAdoptionRollbackData
		for _, file := range previousManifest.Files {
			previousFiles[strings.ToLower(filepath.Clean(file.RelativePath))] = file
		}
	}
	snapshotByPath := map[string]journalSnapshot{}
	for _, snapshot := range snapshots {
		snapshotByPath[strings.ToLower(filepath.Clean(snapshot.TargetPath))] = snapshot
	}
	for _, operation := range preview.Operations {
		if operation.Type != "copy" && operation.Type != "adopt" && operation.Type != "move" {
			continue
		}
		managedPath := operation.TargetPath
		hash, size, err := fileops.FileIntegrity(managedPath)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(targetPath, managedPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		entry := ManagedFile{
			RelativePath: relative,
			SHA256:       hash,
			SizeBytes:    size,
		}
		previous, previouslyManaged := previousFiles[strings.ToLower(filepath.Clean(relative))]
		if previouslyManaged {
			entry.BackupPath = previous.BackupPath
			entry.BackupSHA256 = previous.BackupSHA256
			entry.BackupSize = previous.BackupSize
			entry.Ownership = previous.Ownership
			delete(previousFiles, strings.ToLower(filepath.Clean(relative)))
		}
		snapshot := snapshotByPath[strings.ToLower(filepath.Clean(managedPath))]
		if !previouslyManaged && snapshot.Existed && preview.Request.Action != ActionAdopt {
			if operation.BackupPath == "" {
				return fmt.Errorf("planned backup path is missing for %q", managedPath)
			}
			if err := os.MkdirAll(filepath.Dir(operation.BackupPath), 0o755); err != nil {
				return err
			}
			entry.BackupPath = operation.BackupPath
			if matches, matchErr := fileops.FileMatchesIntegrity(entry.BackupPath, snapshot.SHA256, snapshot.SizeBytes); matchErr != nil || !matches {
				if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
					SourcePath: snapshot.BackupPath,
					TargetPath: entry.BackupPath,
					Mode:       0o644,
					Replace:    true,
					OpenLabel:  "journal backup",
				}); err != nil {
					return err
				}
			}
			entry.BackupSHA256, entry.BackupSize, err = fileops.FileIntegrity(entry.BackupPath)
			if err != nil {
				return err
			}
		}
		manifest.Files = append(manifest.Files, entry)
		if operation.Type == "move" {
			name := filepath.Base(operation.SourcePath)
			manifest.OriginalReShadeProxy = &name
		}
	}
	for _, file := range previousFiles {
		manifest.Files = append(manifest.Files, file)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	origin := "installed"
	if preview.Request.Action == ActionAdopt {
		origin = "adopted"
	}
	var acknowledgedAt *string
	if preview.Request.AcknowledgeWarning {
		value := m.now().UTC().Format(timeFormat)
		acknowledgedAt = &value
	} else if existing, found, err := m.store.GetOptiScalerTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath); err == nil && found {
		origin = existing.ManagementOrigin
		acknowledgedAt = existing.WarningAcknowledgedAt
	}
	verifiedAt := m.now().UTC().Format(timeFormat)
	_, err = m.store.SaveOptiScalerTarget(ctx, dbtypes.SaveOptiScalerTargetInput{
		GameID:                 preview.Request.GameID,
		TargetRelativePath:     preview.Request.TargetRelativePath,
		ExecutableRelativePath: preview.Request.ExecutableRelativePath,
		GraphicsAPI:            string(preview.Request.GraphicsAPI),
		ProxyFilename:          preview.Request.ProxyFilename,
		DXGISpoofing:           preview.Request.DXGISpoofing,
		ProcessFilter:          preview.Request.ProcessFilter,
		ReleaseTag:             preview.Release.Tag,
		ReleaseVersion:         preview.Release.Version,
		ReleaseAssetName:       preview.Release.AssetName,
		ReleaseDigest:          preview.Release.Digest,
		ManagementOrigin:       origin,
		Status:                 "managed",
		ManifestJSON:           string(encoded),
		WarningVersion:         WarningVersion,
		WarningAcknowledgedAt:  acknowledgedAt,
		LastVerifiedAt:         &verifiedAt,
	})
	return err
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func (m *Manager) markRecoveryRequired(ctx context.Context, request Request) error {
	target, found, err := m.store.GetOptiScalerTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	target.Status = "recovery_required"
	_, err = m.store.SaveOptiScalerTarget(ctx, dbtypes.SaveOptiScalerTargetInput{
		GameID:                 target.GameID,
		TargetRelativePath:     target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath,
		GraphicsAPI:            target.GraphicsAPI,
		ProxyFilename:          target.ProxyFilename,
		DXGISpoofing:           target.DXGISpoofing,
		ProcessFilter:          target.ProcessFilter,
		ReleaseTag:             target.ReleaseTag,
		ReleaseVersion:         target.ReleaseVersion,
		ReleaseAssetName:       target.ReleaseAssetName,
		ReleaseDigest:          target.ReleaseDigest,
		ManagementOrigin:       target.ManagementOrigin,
		Status:                 target.Status,
		ManifestJSON:           target.ManifestJSON,
		WarningVersion:         target.WarningVersion,
		WarningAcknowledgedAt:  target.WarningAcknowledgedAt,
		LastVerifiedAt:         target.LastVerifiedAt,
	})
	return err
}

func (m *Manager) archiveDrift(targetPath string, preview Preview) error {
	root := filepath.Join(m.dataDir, "archives", "drift", fmt.Sprintf("%d", preview.Request.GameID), fmt.Sprintf("%d", m.now().UnixNano()))
	for _, drift := range preview.Drift {
		if drift.Missing {
			continue
		}
		source := filepath.Join(targetPath, drift.RelativePath)
		target := filepath.Join(root, drift.RelativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: source,
			TargetPath: target,
			Mode:       0o644,
			OpenLabel:  "drifted file",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) archiveUninstall(targetPath string, preview Preview) error {
	root := filepath.Join(m.dataDir, "archives", "uninstall", fmt.Sprintf("%d", preview.Request.GameID), fmt.Sprintf("%d", m.now().UnixNano()))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	target, found, err := m.store.GetOptiScalerTarget(context.Background(), preview.Request.GameID, preview.Request.TargetRelativePath)
	if err != nil {
		return err
	}
	if found {
		if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(target.ManifestJSON), 0o644); err != nil {
			return err
		}
		manifest, err := decodeManifest(target.ManifestJSON)
		if err != nil {
			return err
		}
		for index, file := range manifest.Files {
			if file.BackupPath == "" {
				continue
			}
			backupTarget := filepath.Join(root, "backups", fmt.Sprintf("%03d-%s", index, filepath.Base(file.RelativePath)))
			if err := os.MkdirAll(filepath.Dir(backupTarget), 0o755); err != nil {
				return err
			}
			if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
				SourcePath: file.BackupPath,
				TargetPath: backupTarget,
				Mode:       0o644,
				OpenLabel:  "OptiScaler rollback backup",
			}); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"OptiScaler.ini"} {
		source := filepath.Join(targetPath, name)
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: source,
			TargetPath: filepath.Join(root, name),
			Mode:       0o644,
			OpenLabel:  "OptiScaler settings",
		}); err != nil {
			return err
		}
	}
	return nil
}
