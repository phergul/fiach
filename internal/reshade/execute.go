package reshade

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
	"github.com/phergul/fiach/internal/optiscaler"
	"github.com/phergul/fiach/internal/storage/dbtypes"
	"github.com/phergul/fiach/internal/winversion"
)

func (m *Manager) execute(ctx context.Context, gameRoot string, preview Preview) (ApplyResult, error) {
	targetPath, err := ResolveWithinRoot(gameRoot, preview.Request.TargetRelativePath)
	if err != nil {
		return ApplyResult{}, err
	}
	journalID := fmt.Sprintf("%d-%x", m.now().UnixNano(), []byte(preview.PreviewHash[:6]))
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
	journal.Snapshots, err = filetxn.SnapshotOperations(
		filepath.Join(m.dataDir, "journals", journalID), preview.Operations)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := writeJournal(journalPath, journal); err != nil {
		return ApplyResult{}, err
	}

	rollback := func(operationErr error) (ApplyResult, error) {
		journal.Error = operationErr.Error()
		_ = writeJournal(journalPath, journal)
		if rollbackErr := m.rollbackSnapshots(journal.Snapshots); rollbackErr != nil {
			journal.Error = fmt.Sprintf("%v; rollback failed: %v", operationErr, rollbackErr)
			_ = writeJournal(journalPath, journal)
			_ = m.markRecoveryRequired(ctx, preview.Request)
			return ApplyResult{Message: journal.Error}, operationErr
		}
		_ = os.Remove(journalPath)
		_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
		return ApplyResult{RolledBack: true, Message: operationErr.Error()}, operationErr
	}

	if preview.Request.BackupAndContinue && len(preview.Drift) > 0 {
		if err := m.archiveDrift(targetPath, preview); err != nil {
			return rollback(err)
		}
	}
	if preview.Request.Action == ActionUninstall {
		if err := m.archiveUninstall(ctx, targetPath, preview); err != nil {
			return rollback(err)
		}
	}
	if err := persistOperationBackups(preview.Operations, journal.Snapshots); err != nil {
		return rollback(err)
	}
	for index, operation := range preview.Operations {
		if err := m.executeOperation(operation); err != nil {
			return rollback(err)
		}
		journal.CompletedSteps = index + 1
		if err := writeJournal(journalPath, journal); err != nil {
			return rollback(err)
		}
	}
	if err := filetxn.VerifyOperations(preview.Operations); err != nil {
		return rollback(err)
	}
	if err := m.verifyApplied(targetPath, preview); err != nil {
		return rollback(err)
	}
	if err := m.commitState(ctx, targetPath, preview); err != nil {
		return rollback(err)
	}
	journal.DatabaseCommitted = true
	if err := writeJournal(journalPath, journal); err != nil {
		_ = os.Remove(journalPath)
		_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
		return ApplyResult{Success: true, Message: "ReShade action completed; journal cleanup was forced."}, nil
	}
	if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return ApplyResult{}, err
	}
	_ = os.RemoveAll(filepath.Join(m.dataDir, "journals", journal.ID))
	return ApplyResult{Success: true, Message: "ReShade action completed."}, nil
}

func (m *Manager) commitState(ctx context.Context, targetPath string, preview Preview) error {
	if preview.Request.OptiScalerPrimaryProxyFilename != "" {
		if err := m.commitOptiScalerChainState(ctx, targetPath, preview); err != nil {
			return err
		}
	}
	if preview.Request.Action == ActionUninstall {
		return m.store.DeleteReShadeTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath)
	}
	if preview.DesiredTarget == nil {
		return errors.New("desired ReShade target state is required")
	}
	if err := applyBackupMetadata(targetPath, &preview, preview.DesiredTarget.Manifest.Files); err != nil {
		return err
	}
	manifestJSON, err := json.Marshal(preview.DesiredTarget.Manifest)
	if err != nil {
		return err
	}
	verifiedAt := m.now().UTC().Format(timeFormat)
	_, err = m.store.SaveReShadeTarget(ctx, dbtypes.SaveReShadeTargetInput{
		GameID:                 preview.Request.GameID,
		TargetRelativePath:     preview.Request.TargetRelativePath,
		ExecutableRelativePath: preview.Request.ExecutableRelativePath,
		RenderingAPI:           string(preview.Request.RenderingAPI),
		ProxyFilename:          preview.Request.ProxyFilename,
		ActiveRuntimeFilename:  preview.Request.ActiveRuntimeFilename,
		Architecture:           string(preview.Request.Architecture),
		BuildVariant:           string(preview.Request.BuildVariant),
		RuntimeVersion:         preview.DesiredTarget.RuntimeVersion,
		InstallerTag:           preview.DesiredTarget.Provenance.Tag,
		InstallerAssetName:     preview.DesiredTarget.Provenance.AssetName,
		InstallerURL:           preview.DesiredTarget.Provenance.URL,
		InstallerDigest:        preview.DesiredTarget.Provenance.Digest,
		InstallerSize:          preview.DesiredTarget.Provenance.Size,
		ManagementOrigin:       preview.DesiredTarget.ManagementOrigin,
		Status:                 "managed",
		ManifestJSON:           string(manifestJSON),
		LastVerifiedAt:         &verifiedAt,
	})
	return err
}

func (m *Manager) commitOptiScalerChainState(ctx context.Context, targetPath string, preview Preview) error {
	store, ok := m.store.(optiScalerTargetStore)
	if !ok {
		return nil
	}
	target, found, err := store.GetOptiScalerTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	var manifest optiscaler.Manifest
	if err := json.Unmarshal([]byte(target.ManifestJSON), &manifest); err != nil {
		return fmt.Errorf("decode OptiScaler manifest for ReShade chain update: %w", err)
	}
	manifest.Config.LoadReShade = preview.Request.Action != ActionUninstall
	if err := refreshOptiScalerConfigManifestFile(targetPath, &manifest); err != nil {
		return err
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	verifiedAt := m.now().UTC().Format(timeFormat)
	_, err = store.SaveOptiScalerTarget(ctx, dbtypes.SaveOptiScalerTargetInput{
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
		ManifestJSON:           string(manifestJSON),
		WarningVersion:         target.WarningVersion,
		WarningAcknowledgedAt:  target.WarningAcknowledgedAt,
		LastVerifiedAt:         &verifiedAt,
	})
	return err
}

func refreshOptiScalerConfigManifestFile(targetPath string, manifest *optiscaler.Manifest) error {
	configPath := filepath.Join(targetPath, "OptiScaler.ini")
	hash, size, err := fileops.FileIntegrity(configPath)
	if err != nil {
		return fmt.Errorf("inspect managed OptiScaler configuration after ReShade chain update: %w", err)
	}
	for index := range manifest.Files {
		if !strings.EqualFold(filepath.Clean(manifest.Files[index].RelativePath), "OptiScaler.ini") {
			continue
		}
		manifest.Files[index].SHA256 = hash
		manifest.Files[index].SizeBytes = size
		return nil
	}
	manifest.Files = append(manifest.Files, optiscaler.ManagedFile{
		RelativePath: "OptiScaler.ini",
		SHA256:       hash,
		SizeBytes:    size,
	})
	return nil
}

func persistOperationBackups(operations []Operation, snapshots []filetxn.Snapshot) error {
	snapshotByPath := make(map[string]filetxn.Snapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotByPath[strings.ToLower(filepath.Clean(snapshot.TargetPath))] = snapshot
	}
	for _, operation := range operations {
		if strings.TrimSpace(operation.BackupPath) == "" {
			continue
		}
		snapshot := snapshotByPath[strings.ToLower(filepath.Clean(operation.TargetPath))]
		if !snapshot.Existed {
			return fmt.Errorf("planned backup target %q did not exist at apply time", operation.TargetPath)
		}
		if err := os.MkdirAll(filepath.Dir(operation.BackupPath), 0o755); err != nil {
			return err
		}
		matches, matchErr := fileops.FileMatchesIntegrity(
			operation.BackupPath, snapshot.SHA256, snapshot.SizeBytes)
		if matchErr == nil && matches {
			continue
		}
		if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: snapshot.BackupPath, TargetPath: operation.BackupPath,
			Mode: 0o644, Replace: true, OpenLabel: "ReShade journal backup",
		}); err != nil {
			return err
		}
	}
	return nil
}

func applyBackupMetadata(targetPath string, preview *Preview, files []ManagedFile) error {
	operationsByRelativePath := map[string]Operation{}
	for _, operation := range preview.Operations {
		if operation.BackupPath == "" {
			continue
		}
		relative, err := filepath.Rel(targetPath, operation.TargetPath)
		if err != nil {
			return err
		}
		operationsByRelativePath[strings.ToLower(filepath.Clean(relative))] = operation
	}
	for index := range files {
		operation, ok := operationsByRelativePath[strings.ToLower(filepath.Clean(files[index].RelativePath))]
		if !ok {
			continue
		}
		hash, size, err := fileops.FileIntegrity(operation.BackupPath)
		if err != nil {
			return err
		}
		backupPath := operation.BackupPath
		files[index].BackupPath = &backupPath
		files[index].BackupSHA256 = &hash
		files[index].BackupSize = &size
	}
	preview.DesiredTarget.Manifest.Files = files
	return nil
}

func (m *Manager) markRecoveryRequired(ctx context.Context, request Request) error {
	row, found, err := m.store.GetReShadeTarget(ctx, request.GameID, request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	row.Status = "recovery_required"
	_, err = m.store.SaveReShadeTarget(ctx, dbtypes.SaveReShadeTargetInput{
		GameID:                 row.GameID,
		TargetRelativePath:     row.TargetRelativePath,
		ExecutableRelativePath: row.ExecutableRelativePath,
		RenderingAPI:           row.RenderingAPI,
		ProxyFilename:          row.ProxyFilename,
		ActiveRuntimeFilename:  row.ActiveRuntimeFilename,
		Architecture:           row.Architecture,
		BuildVariant:           row.BuildVariant,
		RuntimeVersion:         row.RuntimeVersion,
		InstallerTag:           row.InstallerTag,
		InstallerAssetName:     row.InstallerAssetName,
		InstallerURL:           row.InstallerURL,
		InstallerDigest:        row.InstallerDigest,
		InstallerSize:          row.InstallerSize,
		ManagementOrigin:       row.ManagementOrigin,
		Status:                 row.Status,
		ManifestJSON:           row.ManifestJSON,
		LastVerifiedAt:         row.LastVerifiedAt,
	})
	return err
}

func (m *Manager) archiveDrift(targetPath string, preview Preview) error {
	root := filepath.Join(m.dataDir, "archives", "drift",
		fmt.Sprintf("%d", preview.Request.GameID), fmt.Sprintf("%d", m.now().UnixNano()))
	for _, drift := range preview.Drift {
		if drift.Missing {
			continue
		}
		source := filepath.Join(targetPath, drift.RelativePath)
		if err := copyArchiveFile(source, filepath.Join(root, drift.RelativePath)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) archiveUninstall(ctx context.Context, targetPath string, preview Preview) error {
	row, found, err := m.store.GetReShadeTarget(ctx, preview.Request.GameID, preview.Request.TargetRelativePath)
	if err != nil || !found {
		return err
	}
	root := filepath.Join(m.dataDir, "archives", "uninstall",
		fmt.Sprintf("%d", preview.Request.GameID), fmt.Sprintf("%d", m.now().UnixNano()))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(row.ManifestJSON), 0o644); err != nil {
		return err
	}
	manifest, err := DecodeManifest(row.ManifestJSON)
	if err != nil {
		return err
	}
	for index, file := range manifest.Files {
		if file.BackupPath == nil || strings.TrimSpace(*file.BackupPath) == "" {
			continue
		}
		name := fmt.Sprintf("%03d-%s", index, filepath.Base(file.RelativePath))
		if err := copyArchiveFile(*file.BackupPath, filepath.Join(root, "backups", name)); err != nil {
			return err
		}
	}
	_ = targetPath
	return nil
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func verifyAppliedReShadeState(targetPath string, preview Preview) error {
	if preview.Request.Action == ActionUninstall {
		return nil
	}
	if preview.DesiredTarget == nil {
		return errors.New("desired ReShade target state is required for verification")
	}
	switch preview.DesiredTarget.Manifest.VariantProvenance {
	case VariantProvenanceVerified:
		if preview.DesiredTarget.Provenance.Digest == nil ||
			strings.TrimSpace(*preview.DesiredTarget.Provenance.Digest) == "" {
			return errors.New("verified ReShade variant is missing installer provenance")
		}
	case VariantProvenanceUserDeclared:
		if preview.Request.Action != ActionAdopt &&
			preview.DesiredTarget.ManagementOrigin != "adopted" {
			return errors.New("user-declared ReShade variant is only valid for adopted targets")
		}
	default:
		return errors.New("ReShade variant provenance is missing")
	}
	activeRuntime := activeRuntimeFilename(preview.Request)
	runtimePath := filepath.Join(targetPath, activeRuntime)
	metadata, err := winversion.Read(runtimePath)
	if err != nil {
		return fmt.Errorf("read installed ReShade runtime metadata: %w", err)
	}
	if !isReShadeMetadata(metadata) {
		return errors.New("installed runtime is not positively identified as ReShade")
	}
	architecture, err := inspectPEArchitecture(runtimePath)
	if err != nil {
		return err
	}
	if architecture != preview.Request.Architecture {
		return fmt.Errorf(
			"installed runtime architecture %q does not match %q",
			architecture,
			preview.Request.Architecture,
		)
	}
	version := runtimeVersionFromMetadata(metadata)
	if version == "" || version != preview.DesiredTarget.RuntimeVersion {
		return fmt.Errorf(
			"installed runtime version %q does not match %q",
			version,
			preview.DesiredTarget.RuntimeVersion,
		)
	}
	reShadeProxies := 0
	for _, filename := range supportedLocalProxies {
		path := filepath.Join(targetPath, filename)
		info, statErr := os.Stat(path)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("inspect final rendering proxy %q", filename)
		}
		proxyMetadata, metadataErr := winversion.Read(path)
		if metadataErr == nil && isReShadeMetadata(proxyMetadata) {
			reShadeProxies++
			if !strings.EqualFold(filename, activeRuntime) {
				return fmt.Errorf("unexpected ReShade proxy %q remains installed", filename)
			}
		}
	}
	if strings.EqualFold(activeRuntime, preview.Request.ProxyFilename) && reShadeProxies != 1 {
		return fmt.Errorf("final ReShade proxy count is %d, want 1", reShadeProxies)
	}
	return nil
}
