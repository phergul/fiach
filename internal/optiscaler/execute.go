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
	"github.com/phergul/fiach/internal/filetxn"
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
	return filetxn.SnapshotOperations(filepath.Join(m.dataDir, "journals", journalID), operations)
}

func operationTouchedPaths(operation Operation) []string {
	return filetxn.TouchedPaths(operation)
}

func executeOperation(operation Operation) error {
	return filetxn.ExecuteOperation(operation, "staged OptiScaler file")
}

func verifyOperations(operations []Operation) error {
	return filetxn.VerifyOperations(operations)
}

func (m *Manager) commitState(ctx context.Context, targetPath string, preview Preview, snapshots []journalSnapshot) error {
	if preview.Request.Action == ActionUninstall {
		if err := m.restoreManagedReShadeAfterUninstall(ctx, targetPath, preview.Request); err != nil {
			return err
		}
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
			manifest.Files[len(manifest.Files)-1].Ownership = string(OwnershipReShade)
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
	if err != nil {
		return err
	}
	if manifest.Config.LoadReShade {
		if err := m.chainManagedReShade(ctx, targetPath, preview.Request, verifiedAt); err != nil {
			return err
		}
	}
	return nil
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

func (m *Manager) chainManagedReShade(
	ctx context.Context,
	targetPath string,
	request Request,
	verifiedAt string,
) error {
	store, ok := m.store.(reShadeTargetStore)
	if !ok {
		return nil
	}
	target, found, err := store.GetReShadeTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	activeRuntime := chainedReShadeRuntimeFilename(target.Architecture)
	manifestJSON, err := rewriteReShadeRuntimeManifest(
		targetPath,
		target.ManifestJSON,
		firstNonEmpty(target.ActiveRuntimeFilename, target.ProxyFilename),
		activeRuntime,
	)
	if err != nil {
		return err
	}
	_, err = store.SaveReShadeTarget(ctx, dbtypes.SaveReShadeTargetInput{
		GameID:                 target.GameID,
		TargetRelativePath:     target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath,
		RenderingAPI:           target.RenderingAPI,
		ProxyFilename:          target.ProxyFilename,
		ActiveRuntimeFilename:  activeRuntime,
		Architecture:           target.Architecture,
		BuildVariant:           target.BuildVariant,
		RuntimeVersion:         target.RuntimeVersion,
		InstallerTag:           target.InstallerTag,
		InstallerAssetName:     target.InstallerAssetName,
		InstallerURL:           target.InstallerURL,
		InstallerDigest:        target.InstallerDigest,
		InstallerSize:          target.InstallerSize,
		ManagementOrigin:       target.ManagementOrigin,
		Status:                 "managed",
		ManifestJSON:           manifestJSON,
		LastVerifiedAt:         &verifiedAt,
	})
	return err
}

func (m *Manager) restoreManagedReShadeAfterUninstall(
	ctx context.Context,
	targetPath string,
	request Request,
) error {
	store, ok := m.store.(reShadeTargetStore)
	if !ok {
		return nil
	}
	target, found, err := store.GetReShadeTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	activeRuntime := firstNonEmpty(target.ActiveRuntimeFilename, target.ProxyFilename)
	if strings.EqualFold(activeRuntime, target.ProxyFilename) {
		return nil
	}
	manifestJSON, err := rewriteReShadeRuntimeManifest(targetPath, target.ManifestJSON, activeRuntime, target.ProxyFilename)
	if err != nil {
		return err
	}
	verifiedAt := m.now().UTC().Format(timeFormat)
	_, err = store.SaveReShadeTarget(ctx, dbtypes.SaveReShadeTargetInput{
		GameID:                 target.GameID,
		TargetRelativePath:     target.TargetRelativePath,
		ExecutableRelativePath: target.ExecutableRelativePath,
		RenderingAPI:           target.RenderingAPI,
		ProxyFilename:          target.ProxyFilename,
		ActiveRuntimeFilename:  target.ProxyFilename,
		Architecture:           target.Architecture,
		BuildVariant:           target.BuildVariant,
		RuntimeVersion:         target.RuntimeVersion,
		InstallerTag:           target.InstallerTag,
		InstallerAssetName:     target.InstallerAssetName,
		InstallerURL:           target.InstallerURL,
		InstallerDigest:        target.InstallerDigest,
		InstallerSize:          target.InstallerSize,
		ManagementOrigin:       target.ManagementOrigin,
		Status:                 "managed",
		ManifestJSON:           manifestJSON,
		LastVerifiedAt:         &verifiedAt,
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

func rewriteReShadeRuntimeManifest(
	targetPath string,
	manifestJSON string,
	oldRuntime string,
	newRuntime string,
) (string, error) {
	var manifest map[string]any
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return "", fmt.Errorf("decode ReShade manifest for chain update: %w", err)
	}
	files, _ := manifest["files"].([]any)
	hash, size, integrityErr := fileops.FileIntegrity(filepath.Join(targetPath, newRuntime))
	if integrityErr != nil {
		return "", fmt.Errorf("inspect managed ReShade chained runtime %q: %w", newRuntime, integrityErr)
	}
	updated := false
	for _, item := range files {
		file, ok := item.(map[string]any)
		if !ok {
			continue
		}
		relativePath, _ := file["relativePath"].(string)
		if !strings.EqualFold(filepath.Clean(relativePath), filepath.Clean(oldRuntime)) &&
			!strings.EqualFold(filepath.Clean(relativePath), filepath.Clean(newRuntime)) {
			continue
		}
		file["relativePath"] = newRuntime
		file["sha256"] = hash
		file["sizeBytes"] = float64(size)
		updated = true
		break
	}
	if !updated {
		return "", fmt.Errorf("ReShade manifest runtime %q was not found", oldRuntime)
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func chainedReShadeRuntimeFilename(architecture string) string {
	if strings.EqualFold(architecture, "x86") {
		return "ReShade32.dll"
	}
	return "ReShade64.dll"
}
